package handler_chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
	"github.com/ixayda/ix-Go/pkg/logger"
	"github.com/tencentyun/cos-go-sdk-v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const multipartPartSize int64 = 8 << 20

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var errMultipartHash = errors.New("multipart content hash mismatch")

type multipartInitRequest struct {
	attachmentPresignRequest
	RequestID   string `json:"request_id"`
	ContentHash string `json:"content_hash"`
}

func InitAttachmentMultipart(c *gin.Context) {
	var req multipartInitRequest
	if !forwardBody(c, &req) {
		return
	}
	req.ContentHash = strings.ToLower(strings.TrimSpace(req.ContentHash))
	if !sha256Pattern.MatchString(req.ContentHash) || len(req.RequestID) < 16 || len(req.RequestID) > 64 {
		response.BadRequest(c, "文件指纹或上传标识无效")
		return
	}
	db := datasource.Datasource.Mysql.GetClient()
	settings, err := config.GetChatAttachmentUploadSettings(c.Request.Context(), db)
	if err != nil {
		response.InternalError(c)
		return
	}
	if !settings.Enabled {
		response.BadRequest(c, "文件上传已关闭")
		return
	}
	normalized, err := normalizeAttachmentPresignRequest(req.attachmentPresignRequest, settings)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := loadChatAttachmentCurrentUser(c.GetString("im_user_id"))
	if err != nil {
		response.Unauthorized(c, "账号不可用")
		return
	}
	conversation, err := resolveAttachmentConversation(normalized.ConversationID, user)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cosCfg, err := chatAttachmentCOSConfig()
	if err != nil {
		response.BadRequest(c, "聊天大文件 COS 配置未完成")
		return
	}
	client, err := newCOSClient(cosCfg)
	if err != nil {
		response.InternalError(c)
		return
	}
	hash := sha256.Sum256([]byte(user.IMUserID + "\x00" + req.RequestID))
	key := hex.EncodeToString(hash[:])
	// Finish the idempotent reservation even if the HTTP response is lost.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var upload model.ChatMultipartUpload
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		upload = model.ChatMultipartUpload{ID: key, Owner: user.IMUserID, State: "initializing", Fingerprint: req.ContentHash, PartSize: multipartPartSize, ExpiresAt: time.Now().Add(24 * time.Hour)}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&upload).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&upload, "id = ?", key).Error; err != nil {
			return err
		}
		if upload.Fingerprint != req.ContentHash {
			return ErrMultipartConflict
		}
		if upload.FileID != "" {
			var previous model.ChatAttachment
			if err := tx.First(&previous, "file_id = ?", upload.FileID).Error; err != nil {
				return err
			}
			if previous.ConversationID != normalized.ConversationID || previous.FileSize != normalized.FileSize || previous.FileName != normalized.FileName || previous.ContentType != normalized.ContentType {
				return ErrMultipartConflict
			}
			if upload.State != "failed" && upload.State != "aborted" && upload.State != "cleaned" && upload.ExpiresAt.After(time.Now()) {
				return nil
			}
			if upload.State == "completing" || upload.State == "verifying" {
				return errors.New("上传仍在确认，请稍后查询")
			}
			// A new explicit init after definitive failure/cancel/expiry may restart
			// the upload; a lost response while confirming never takes this path.
			upload.Attempts = 0
			upload.LeaseUntil = 0
			upload.ExpiresAt = time.Now().Add(24 * time.Hour)
		}
		fileID, err := randomHexID("att_", 16)
		if err != nil {
			return err
		}
		token, err := randomHexID("", 24)
		if err != nil {
			return err
		}
		expires := time.Now().Add(time.Duration(settings.RetentionDays) * 24 * time.Hour)
		file := model.ChatAttachment{FileID: fileID, DownloadToken: token, UploaderUserID: user.ID, UploaderIMUserID: user.IMUserID,
			ConversationID: normalized.ConversationID, ConversationType: conversation.conversationType, PeerUserID: conversation.peerUserID, GroupID: conversation.groupID,
			ObjectKey: buildAttachmentObjectKey(user.IMUserID, fileID, normalized.FileName), FileName: normalized.FileName, FileSize: normalized.FileSize,
			ContentType: normalized.ContentType, IsVideo: normalized.IsVideo || isVideoAttachment(normalized.ContentType, normalized.FileName), Status: chatAttachmentStatusPending, ExpiresAt: &expires}
		if err := tx.Create(&file).Error; err != nil {
			return err
		}
		initiated, _, err := client.Object.InitiateMultipartUpload(ctx, file.ObjectKey, &cos.InitiateMultipartUploadOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: file.ContentType}})
		if err != nil {
			return err
		}
		upload.FileID, upload.UploadID, upload.State = fileID, initiated.UploadID, "uploading"
		if err := tx.Save(&upload).Error; err != nil {
			_, _ = client.Object.AbortMultipartUpload(ctx, file.ObjectKey, initiated.UploadID)
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrMultipartConflict) {
			response.BadRequest(c, err.Error())
		} else {
			response.InternalError(c)
		}
		return
	}
	multipartSnapshot(c, upload)
}

var ErrMultipartConflict = errors.New("同一上传标识的文件内容或会话已变化")

func loadMultipart(c *gin.Context) (model.ChatMultipartUpload, model.ChatAttachment, bool) {
	var upload model.ChatMultipartUpload
	var file model.ChatAttachment
	db := datasource.Datasource.Mysql.GetClient().WithContext(c.Request.Context())
	if err := db.First(&upload, "file_id = ? AND owner = ?", c.Param("file_id"), c.GetString("im_user_id")).Error; err != nil {
		response.NotFound(c, "上传任务不存在")
		return upload, file, false
	}
	if err := db.First(&file, "file_id = ?", upload.FileID).Error; err != nil {
		response.NotFound(c, "附件不存在")
		return upload, file, false
	}
	return upload, file, true
}

func multipartClient() (*cos.Client, error) {
	cfg, err := chatAttachmentCOSConfig()
	if err != nil {
		return nil, err
	}
	return newCOSClient(cfg)
}

func multipartParts(ctx context.Context, client *cos.Client, file model.ChatAttachment, upload model.ChatMultipartUpload) ([]cos.Object, error) {
	parts := []cos.Object{}
	marker := ""
	for {
		page, _, err := client.Object.ListParts(ctx, file.ObjectKey, upload.UploadID, &cos.ObjectListPartsOptions{MaxParts: "1000", PartNumberMarker: marker})
		if err != nil {
			return nil, err
		}
		parts = append(parts, page.Parts...)
		if !page.IsTruncated {
			break
		}
		if page.NextPartNumberMarker == marker {
			return nil, errors.New("invalid parts cursor")
		}
		marker = page.NextPartNumberMarker
		if len(parts) > 10000 {
			return nil, errors.New("too many parts")
		}
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].PartNumber < parts[j].PartNumber })
	return parts, nil
}

func multipartSnapshot(c *gin.Context, upload model.ChatMultipartUpload) {
	var file model.ChatAttachment
	if err := datasource.Datasource.Mysql.GetClient().First(&file, "file_id = ?", upload.FileID).Error; err != nil {
		response.InternalError(c)
		return
	}
	rows := []map[string]any{}
	if upload.State == "uploading" {
		client, err := multipartClient()
		if err != nil {
			response.InternalError(c)
			return
		}
		parts, err := multipartParts(c.Request.Context(), client, file, upload)
		if err != nil {
			response.BadRequest(c, "未能确认已上传分片，请稍后重试")
			return
		}
		for _, part := range parts {
			rows = append(rows, map[string]any{"part_number": part.PartNumber, "etag": part.ETag, "size": part.Size})
		}
	}
	// No private signed URL is written to the message payload.
	response.OK(c, gin.H{"file_id": file.FileID, "file_name": file.FileName, "file_size": file.FileSize, "content_type": file.ContentType, "is_video": file.IsVideo,
		"content_hash": upload.Fingerprint, "part_size": upload.PartSize, "state": upload.State, "parts": rows, "expires_at": upload.ExpiresAt,
		"download_url": buildAttachmentDownloadURL(c, file.FileID, file.DownloadToken, false), "preview_url": buildAttachmentDownloadURL(c, file.FileID, file.DownloadToken, false)})
}

func GetAttachmentMultipart(c *gin.Context) {
	upload, _, ok := loadMultipart(c)
	if ok {
		multipartSnapshot(c, upload)
	}
}

func SignAttachmentPart(c *gin.Context) {
	upload, file, ok := loadMultipart(c)
	if !ok {
		return
	}
	part, err := strconv.Atoi(c.Param("part_number"))
	count := (file.FileSize + upload.PartSize - 1) / upload.PartSize
	if err != nil || part < 1 || int64(part) > count || upload.State != "uploading" || !upload.ExpiresAt.After(time.Now()) {
		response.BadRequest(c, "上传已过期或分片编号无效")
		return
	}
	cfg, err := chatAttachmentCOSConfig()
	if err != nil {
		response.InternalError(c)
		return
	}
	client, err := newCOSClient(cfg)
	if err != nil {
		response.InternalError(c)
		return
	}
	options := struct {
		UploadID   string `url:"uploadId" header:"-"`
		PartNumber int    `url:"partNumber" header:"-"`
	}{upload.UploadID, part}
	signed, err := client.Object.GetPresignedURL(c.Request.Context(), http.MethodPut, file.ObjectKey, cfg.SecretID, cfg.SecretKey, 5*time.Minute, &options)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"put_url": signed.String(), "expires_in": 300})
}

func CompleteAttachmentMultipart(c *gin.Context) {
	upload, _, ok := loadMultipart(c)
	if !ok {
		return
	}
	if upload.State == "aborted" || upload.State == "failed" || !upload.ExpiresAt.After(time.Now()) {
		response.BadRequest(c, "上传已取消、过期或校验失败")
		return
	}
	if upload.State == "uploading" {
		if err := datasource.Datasource.Mysql.GetClient().Model(&model.ChatMultipartUpload{}).Where("id = ? AND state = ?", upload.ID, "uploading").Updates(map[string]any{"state": "completing", "lease_until": 0}).Error; err != nil {
			response.InternalError(c)
			return
		}
		upload.State = "completing"
	}
	multipartSnapshot(c, upload)
}

func AbortAttachmentMultipart(c *gin.Context) {
	upload, file, ok := loadMultipart(c)
	if !ok {
		return
	}
	// Do not race an already-started merge/verification. Those bytes must first settle.
	if upload.State != "uploading" && upload.State != "aborted" {
		response.BadRequest(c, "文件正在确认或已经完成，不能取消")
		return
	}
	db := datasource.Datasource.Mysql.GetClient()
	result := db.Model(&model.ChatMultipartUpload{}).Where("id = ? AND state = ?", upload.ID, "uploading").Update("state", "aborted")
	if result.Error != nil {
		response.InternalError(c)
		return
	}
	if result.RowsAffected == 1 {
		client, err := multipartClient()
		if err == nil {
			_, err = client.Object.AbortMultipartUpload(c.Request.Context(), file.ObjectKey, upload.UploadID)
		}
		if err != nil {
			response.BadRequest(c, "取消已记录，存储清理待后台重试")
			return
		}
	}
	response.OK(c, nil)
}

func validateMultipartParts(parts []cos.Object, size, partSize int64) error {
	count := (size + partSize - 1) / partSize
	if int64(len(parts)) != count {
		return errors.New("分片未全部上传")
	}
	for i, part := range parts {
		expected := partSize
		if int64(i) == count-1 {
			expected = size - int64(i)*partSize
		}
		if part.PartNumber != i+1 || part.Size != expected || strings.Trim(part.ETag, "\"") == "" {
			return errors.New("分片大小或ETag校验失败")
		}
	}
	return nil
}

// The same worker resumes after process death and cleans expired multipart objects.
func StartAttachmentMultipartWorker(ctx context.Context, db *gorm.DB) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			var uploads []model.ChatMultipartUpload
			if db.WithContext(ctx).Where("(state IN ? AND lease_until < ?) OR (state IN ? AND expires_at < ?)", []string{"completing", "verifying"}, time.Now().UnixMilli(), []string{"uploading", "aborted"}, time.Now()).Limit(20).Find(&uploads).Error != nil {
				continue
			}
			for _, upload := range uploads {
				if ctx.Err() != nil {
					return
				}
				processMultipart(ctx, db, upload)
			}
		}
	}()
}

func processMultipart(parent context.Context, db *gorm.DB, upload model.ChatMultipartUpload) {
	lease := time.Now().Add(10 * time.Minute).UnixMilli()
	claimed := db.Model(&model.ChatMultipartUpload{}).Where("id = ? AND state = ? AND lease_until < ?", upload.ID, upload.State, time.Now().UnixMilli()).Updates(map[string]any{"lease_until": lease})
	if claimed.Error != nil || claimed.RowsAffected != 1 {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 8*time.Minute)
	defer cancel()
	client, err := multipartClient()
	if err != nil {
		return
	}
	var file model.ChatAttachment
	if db.First(&file, "file_id = ?", upload.FileID).Error != nil {
		return
	}
	if upload.State == "aborted" || upload.ExpiresAt.Before(time.Now()) {
		_, err = client.Object.AbortMultipartUpload(ctx, file.ObjectKey, upload.UploadID)
		// COS reports NoSuchUpload for an already-cleaned upload.
		var serviceErr *cos.ErrorResponse
		if err == nil || errors.As(err, &serviceErr) && serviceErr.Code == "NoSuchUpload" {
			db.Model(&upload).Updates(map[string]any{"state": "cleaned", "lease_until": 0})
		}
		return
	}
	err = finishMultipart(ctx, db, client, file, upload)
	if parent.Err() != nil {
		return
	}
	fields := map[string]any{"lease_until": time.Now().Add(15 * time.Second).UnixMilli(), "attempts": upload.Attempts + 1}
	if errors.Is(err, errMultipartHash) {
		fields["state"] = "failed"
	} else if err == nil {
		fields["state"] = "uploaded"
	} else if upload.Attempts >= 4 {
		fields["state"] = "failed"
	}
	if err != nil {
		logger.Warnf("multipart confirmation pending: file_id=%s attempt=%d", upload.FileID, upload.Attempts+1)
	}
	db.Model(&model.ChatMultipartUpload{}).Where("id = ? AND lease_until = ?", upload.ID, lease).Updates(fields)
}

func finishMultipart(ctx context.Context, db *gorm.DB, client *cos.Client, file model.ChatAttachment, upload model.ChatMultipartUpload) error {
	// HEAD first handles a lost CompleteMultipartUpload response without reupload.
	head, headErr := client.Object.Head(ctx, file.ObjectKey, nil)
	if headErr != nil || head == nil || head.Header.Get("Content-Length") != fmt.Sprint(file.FileSize) {
		parts, err := multipartParts(ctx, client, file, upload)
		if err != nil {
			return err
		}
		if err = validateMultipartParts(parts, file.FileSize, upload.PartSize); err != nil {
			return err
		}
		if _, _, err = client.Object.CompleteMultipartUpload(ctx, file.ObjectKey, upload.UploadID, &cos.CompleteMultipartUploadOptions{Parts: parts}); err != nil {
			return err
		}
	}
	result, err := client.Object.Get(ctx, file.ObjectKey, nil)
	if err != nil {
		return err
	}
	defer result.Body.Close()
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(result.Body, file.FileSize+1))
	if err != nil {
		return err
	}
	if n != file.FileSize || hex.EncodeToString(hash.Sum(nil)) != upload.Fingerprint {
		return errMultipartHash
	}
	now := time.Now()
	return db.WithContext(ctx).Model(&model.ChatAttachment{}).Where("file_id = ? AND status = ?", file.FileID, chatAttachmentStatusPending).Updates(map[string]any{"status": chatAttachmentStatusUploaded, "uploaded_at": now}).Error
}
