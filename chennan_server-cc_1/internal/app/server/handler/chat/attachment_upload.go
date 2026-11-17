package handler_chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"gorm.io/gorm"

	"github.com/ixayda/ix-Go/internal/app/server/handler/authlimit"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/ratelimit"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

const (
	chatAttachmentStatusPending  = "pending"
	chatAttachmentStatusUploaded = "uploaded"
)

var errChatAttachmentCOSConfigIncomplete = errors.New("chat attachment cos config is incomplete")

type attachmentSettingsData struct {
	Enabled          bool   `json:"enabled"`
	MaxSizeMB        int64  `json:"max_size_mb"`
	HardLimitMB      int64  `json:"hard_limit_mb"`
	NativeLimitMB    int64  `json:"native_limit_mb"`
	PutURLTTLSeconds int64  `json:"put_url_ttl_seconds"`
	GetURLTTLSeconds int64  `json:"get_url_ttl_seconds"`
	RetentionDays    int64  `json:"retention_days"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type attachmentPresignRequest struct {
	ConversationID string `json:"conversation_id"`
	FileName       string `json:"file_name"`
	FileSize       int64  `json:"file_size"`
	ContentType    string `json:"content_type"`
	IsVideo        bool   `json:"is_video"`
}

type attachmentPresignData struct {
	FileID        string `json:"file_id"`
	FileName      string `json:"file_name"`
	PutURL        string `json:"put_url"`
	DownloadURL   string `json:"download_url"`
	PreviewURL    string `json:"preview_url"`
	ObjectKey     string `json:"object_key"`
	ContentType   string `json:"content_type"`
	MaxSizeMB     int64  `json:"max_size_mb"`
	NativeLimitMB int64  `json:"native_limit_mb"`
	IsVideo       bool   `json:"is_video"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

type attachmentCompleteRequest struct {
	FileID string `json:"file_id"`
}

type attachmentAccessData struct {
	URL         string `json:"url"`
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
	IsVideo     bool   `json:"is_video"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

func attachmentSettingsResponse(settings config.ChatAttachmentUploadSettings) attachmentSettingsData {
	updatedAt := ""
	if settings.UpdatedAt != nil && !settings.UpdatedAt.IsZero() {
		updatedAt = settings.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return attachmentSettingsData{
		Enabled:          settings.Enabled,
		MaxSizeMB:        settings.MaxSizeMB,
		HardLimitMB:      settings.HardLimitMB,
		NativeLimitMB:    settings.NativeLimitMB,
		PutURLTTLSeconds: settings.PutURLTTLSeconds,
		GetURLTTLSeconds: settings.GetURLTTLSeconds,
		RetentionDays:    settings.RetentionDays,
		UpdatedAt:        updatedAt,
	}
}

// GetChatAttachmentUploadSettings 获取聊天大文件上传配置。
func GetChatAttachmentUploadSettings(c *gin.Context) {
	settings, err := config.GetChatAttachmentUploadSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, attachmentSettingsResponse(settings))
}

// PresignChatAttachment 生成聊天大文件 COS 直传签名。
func PresignChatAttachment(c *gin.Context) {
	var req attachmentPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	db := datasource.Datasource.Mysql.GetClient()
	settings, err := config.GetChatAttachmentUploadSettings(context.Background(), db)
	if err != nil {
		response.InternalError(c)
		return
	}
	if !settings.Enabled {
		response.BadRequest(c, "聊天大文件上传已关闭")
		return
	}

	currentUser, err := loadChatAttachmentCurrentUser(c.GetString("im_user_id"))
	if err != nil {
		response.Unauthorized(c, "账号不存在或已失效")
		return
	}

	normalized, err := normalizeAttachmentPresignRequest(req, settings)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	conversation, err := resolveAttachmentConversation(normalized.ConversationID, currentUser)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	fileID, err := randomHexID("att_", 16)
	if err != nil {
		response.InternalError(c)
		return
	}
	downloadToken, err := randomHexID("", 24)
	if err != nil {
		response.InternalError(c)
		return
	}

	cosCfg, err := chatAttachmentCOSConfig()
	if err != nil {
		response.BadRequest(c, "聊天大文件 COS 配置未完成，请联系管理员")
		return
	}
	client, err := newCOSClient(cosCfg)
	if err != nil {
		response.InternalError(c)
		return
	}

	objectKey := buildAttachmentObjectKey(currentUser.IMUserID, fileID, normalized.FileName)
	putURL, err := client.Object.GetPresignedURL(
		context.Background(),
		http.MethodPut,
		objectKey,
		cosCfg.SecretID,
		cosCfg.SecretKey,
		time.Duration(settings.PutURLTTLSeconds)*time.Second,
		nil,
		false,
	)
	if err != nil {
		response.InternalError(c)
		return
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(settings.RetentionDays) * 24 * time.Hour)
	isVideo := normalized.IsVideo || isVideoAttachment(normalized.ContentType, normalized.FileName)
	attachment := model.ChatAttachment{
		FileID:           fileID,
		DownloadToken:    downloadToken,
		UploaderUserID:   currentUser.ID,
		UploaderIMUserID: currentUser.IMUserID,
		ConversationID:   normalized.ConversationID,
		ConversationType: conversation.conversationType,
		PeerUserID:       conversation.peerUserID,
		GroupID:          conversation.groupID,
		ObjectKey:        objectKey,
		FileName:         normalized.FileName,
		FileSize:         normalized.FileSize,
		ContentType:      normalized.ContentType,
		IsVideo:          isVideo,
		Status:           chatAttachmentStatusPending,
		ExpiresAt:        &expiresAt,
	}
	if err := db.WithContext(context.Background()).Create(&attachment).Error; err != nil {
		response.InternalError(c)
		return
	}

	downloadURL := buildAttachmentDownloadURL(c, fileID, downloadToken, false)
	response.OK(c, attachmentPresignData{
		FileID:        fileID,
		FileName:      normalized.FileName,
		PutURL:        putURL.String(),
		DownloadURL:   downloadURL,
		PreviewURL:    downloadURL,
		ObjectKey:     objectKey,
		ContentType:   normalized.ContentType,
		MaxSizeMB:     settings.MaxSizeMB,
		NativeLimitMB: settings.NativeLimitMB,
		IsVideo:       isVideo,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
	})
}

// CompleteChatAttachment 确认聊天大文件已上传到 COS。
func CompleteChatAttachment(c *gin.Context) {
	var req attachmentCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	fileID := strings.TrimSpace(req.FileID)
	if fileID == "" {
		response.BadRequest(c, "file_id is required")
		return
	}

	db := datasource.Datasource.Mysql.GetClient()
	var attachment model.ChatAttachment
	err := db.WithContext(context.Background()).
		Where("file_id = ?", fileID).
		Take(&attachment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文件不存在")
			return
		}
		response.InternalError(c)
		return
	}
	if attachment.UploaderIMUserID != c.GetString("im_user_id") {
		response.Forbidden(c, "无权确认该文件")
		return
	}

	var multipart model.ChatMultipartUpload
	multipartErr := db.WithContext(c.Request.Context()).First(&multipart, "file_id = ?", fileID).Error
	if multipartErr == nil && multipart.State != "uploaded" {
		response.BadRequest(c, "分片上传需完成合并与内容校验")
		return
	}
	if multipartErr != nil && !errors.Is(multipartErr, gorm.ErrRecordNotFound) {
		response.InternalError(c)
		return
	}
	if attachment.ExpiresAt != nil && !attachment.ExpiresAt.After(time.Now()) {
		response.BadRequest(c, "文件已过期")
		return
	}
	cosCfg, err := chatAttachmentCOSConfig()
	if err != nil {
		response.BadRequest(c, "聊天大文件 COS 配置未完成，请联系管理员")
		return
	}
	client, err := newCOSClient(cosCfg)
	if err != nil {
		response.InternalError(c)
		return
	}
	uploadVerified := false
	sizeMismatch := false
	for _, delay := range []time.Duration{
		0,
		200 * time.Millisecond,
		500 * time.Millisecond,
		time.Second,
		1500 * time.Millisecond,
	} {
		if delay > 0 {
			time.Sleep(delay)
		}

		headResp, err := client.Object.Head(context.Background(), attachment.ObjectKey, nil)
		if err != nil {
			continue
		}
		if headResp == nil || headResp.Header == nil {
			continue
		}
		contentLength := strings.TrimSpace(headResp.Header.Get("Content-Length"))
		if contentLength == "" {
			continue
		}
		uploadedSize, parseErr := strconv.ParseInt(contentLength, 10, 64)
		if parseErr != nil {
			sizeMismatch = true
			continue
		}
		if uploadedSize == attachment.FileSize {
			uploadVerified = true
			break
		}
		sizeMismatch = true
	}
	if !uploadVerified {
		if sizeMismatch {
			response.BadRequest(c, "文件大小校验失败")
			return
		}
		response.BadRequest(c, "文件尚未上传完成")
		return
	}

	now := time.Now()
	updates := map[string]any{
		"status":      chatAttachmentStatusUploaded,
		"uploaded_at": &now,
	}
	if err := db.WithContext(context.Background()).
		Model(&model.ChatAttachment{}).
		Where("id = ?", attachment.ID).
		Updates(updates).Error; err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, nil)
}

// DownloadChatAttachment 下载或预览聊天大文件。
func DownloadChatAttachment(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("file_id"))
	token := strings.TrimSpace(c.Query("token"))
	if fileID == "" || len(fileID) > 128 || token == "" || len(token) > 256 {
		response.NotFound(c, "文件不存在")
		return
	}
	if !authlimit.Enforce(c, "attachment_download", []ratelimit.SubjectRules{
		{Subject: ratelimit.HashSubject("file", fileID), Rules: ratelimit.PublicDownloadFileRules},
	}) {
		return
	}

	var attachment model.ChatAttachment
	err := datasource.Datasource.Mysql.GetClient().
		WithContext(c.Request.Context()).
		Where("file_id = ? AND download_token = ? AND status = ?", fileID, token, chatAttachmentStatusUploaded).
		Take(&attachment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文件不存在")
			return
		}
		response.InternalError(c)
		return
	}

	settings, err := config.GetChatAttachmentUploadSettings(c.Request.Context(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		response.InternalError(c)
		return
	}
	cosURL, err := signedAttachmentGetURL(attachment, settings, c.Query("download") == "1")
	if err != nil {
		if errors.Is(err, errChatAttachmentCOSConfigIncomplete) {
			response.BadRequest(c, "聊天大文件 COS 配置未完成，请联系管理员")
			return
		}
		response.InternalError(c)
		return
	}

	if !attachment.IsVideo || c.Query("download") == "1" {
		c.Redirect(http.StatusFound, cosURL)
		return
	}

	downloadURL := buildAttachmentDownloadURL(c, attachment.FileID, attachment.DownloadToken, true)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderVideoPreviewHTML(attachment, cosURL, downloadURL)))
}

// GetChatAttachmentAccess 获取已上传聊天附件的短时访问地址。
func GetChatAttachmentAccess(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("file_id"))
	if fileID == "" {
		response.NotFound(c, "文件不存在")
		return
	}

	currentUser, err := loadChatAttachmentCurrentUser(c.GetString("im_user_id"))
	if err != nil {
		response.Unauthorized(c, "账号不存在或已失效")
		return
	}

	var attachment model.ChatAttachment
	err = datasource.Datasource.Mysql.GetClient().
		WithContext(context.Background()).
		Where("file_id = ? AND status = ?", fileID, chatAttachmentStatusUploaded).
		Take(&attachment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "文件不存在")
			return
		}
		response.InternalError(c)
		return
	}

	allowed, err := canAccessChatAttachment(attachment, currentUser)
	if err != nil {
		response.InternalError(c)
		return
	}
	if !allowed {
		response.Forbidden(c, "无权访问该文件")
		return
	}

	settings, err := config.GetChatAttachmentUploadSettings(context.Background(), datasource.Datasource.Mysql.GetClient())
	if err != nil {
		response.InternalError(c)
		return
	}
	cosURL, err := signedAttachmentGetURL(attachment, settings, c.Query("download") == "1")
	if err != nil {
		if errors.Is(err, errChatAttachmentCOSConfigIncomplete) {
			response.BadRequest(c, "聊天大文件 COS 配置未完成，请联系管理员")
			return
		}
		response.InternalError(c)
		return
	}

	expiresAt := ""
	if attachment.ExpiresAt != nil && !attachment.ExpiresAt.IsZero() {
		expiresAt = attachment.ExpiresAt.Format(time.RFC3339)
	}
	response.OK(c, attachmentAccessData{
		URL:         cosURL,
		FileID:      attachment.FileID,
		FileName:    attachment.FileName,
		FileSize:    attachment.FileSize,
		ContentType: attachment.ContentType,
		IsVideo:     attachment.IsVideo,
		ExpiresAt:   expiresAt,
	})
}

type attachmentConversation struct {
	conversationType string
	peerUserID       string
	groupID          string
}

func normalizeAttachmentPresignRequest(
	req attachmentPresignRequest,
	settings config.ChatAttachmentUploadSettings,
) (attachmentPresignRequest, error) {
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	if req.ConversationID == "" {
		return req, errors.New("conversation_id is required")
	}
	if req.FileSize <= 0 {
		return req, errors.New("file_size is required")
	}
	if req.FileSize > settings.MaxSizeMB*1024*1024 {
		return req, fmt.Errorf("文件大小不能超过 %dMB", settings.MaxSizeMB)
	}
	fileName := normalizeAttachmentFileName(req.FileName)
	if fileName == "" {
		return req, errors.New("file_name is required")
	}
	if req.IsVideo {
		fileName = normalizeVideoAttachmentFileName(fileName)
	}
	req.FileName = fileName
	req.ContentType = normalizeAttachmentContentType(req.ContentType, fileName, req.IsVideo)
	return req, nil
}

func loadChatAttachmentCurrentUser(imUserID string) (*model.User, error) {
	var user model.User
	err := datasource.Datasource.Mysql.GetClient().
		Where("im_user_id = ? AND status = ?", strings.TrimSpace(imUserID), "active").
		Take(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func resolveAttachmentConversation(conversationID string, currentUser *model.User) (attachmentConversation, error) {
	if strings.HasPrefix(conversationID, "c2c_") {
		peerUserID := strings.TrimSpace(strings.TrimPrefix(conversationID, "c2c_"))
		if peerUserID == "" {
			return attachmentConversation{}, errors.New("单聊会话无效")
		}
		var peer model.User
		err := datasource.Datasource.Mysql.GetClient().
			Where("im_user_id = ? AND status = ?", peerUserID, "active").
			Take(&peer).Error
		if err != nil {
			return attachmentConversation{}, errors.New("单聊用户不存在或已失效")
		}
		if currentUser.OwnerID != peer.OwnerID {
			return attachmentConversation{}, errors.New("不能向非同归属用户分享文件")
		}
		return attachmentConversation{
			conversationType: "c2c",
			peerUserID:       peerUserID,
		}, nil
	}

	if strings.HasPrefix(conversationID, "group_") {
		groupID := strings.TrimSpace(strings.TrimPrefix(conversationID, "group_"))
		if groupID == "" {
			return attachmentConversation{}, errors.New("群聊会话无效")
		}
		var group model.Group
		err := datasource.Datasource.Mysql.GetClient().
			Where("im_group_id = ? AND status = ?", groupID, "active").
			Take(&group).Error
		if err != nil {
			return attachmentConversation{}, errors.New("群组不存在或已解散")
		}
		if currentUser.OwnerID != group.OwnerID {
			return attachmentConversation{}, errors.New("不能向非同归属群组分享文件")
		}
		return attachmentConversation{
			conversationType: "group",
			groupID:          groupID,
		}, nil
	}

	return attachmentConversation{}, errors.New("conversation_id must start with c2c_ or group_")
}

func canAccessChatAttachment(attachment model.ChatAttachment, currentUser *model.User) (bool, error) {
	if currentUser == nil {
		return false, nil
	}
	if attachment.UploaderIMUserID == currentUser.IMUserID {
		return true, nil
	}
	switch strings.ToLower(strings.TrimSpace(attachment.ConversationType)) {
	case "c2c":
		return attachment.PeerUserID == currentUser.IMUserID, nil
	case "group":
		if strings.TrimSpace(attachment.GroupID) == "" {
			return false, nil
		}
		var group model.Group
		err := datasource.Datasource.Mysql.GetClient().
			WithContext(context.Background()).
			Where("im_group_id = ? AND status = ?", attachment.GroupID, "active").
			Take(&group).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		return group.OwnerID == currentUser.OwnerID, nil
	default:
		return false, nil
	}
}

func normalizeAttachmentFileName(fileName string) string {
	fileName = filepath.Base(strings.TrimSpace(strings.ReplaceAll(fileName, "\\", "/")))
	fileName = strings.Trim(fileName, ". ")
	if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
		return ""
	}
	if len([]rune(fileName)) > 120 {
		runes := []rune(fileName)
		ext := filepath.Ext(fileName)
		prefix := string(runes[:120])
		if ext != "" && !strings.HasSuffix(prefix, ext) {
			prefix = strings.TrimSuffix(prefix, filepath.Ext(prefix)) + ext
		}
		fileName = prefix
	}
	return fileName
}

func normalizeAttachmentContentType(contentType string, fileName string, forceVideo bool) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if forceVideo {
		if strings.HasPrefix(contentType, "video/") {
			return contentType
		}
		if detected := videoAttachmentContentTypeByExtension(fileName); detected != "" {
			return detected
		}
		return "video/mp4"
	}
	if contentType != "" && strings.Contains(contentType, "/") {
		return contentType
	}
	if detected := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName))); detected != "" {
		return strings.Split(detected, ";")[0]
	}
	return "application/octet-stream"
}

func normalizeVideoAttachmentFileName(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if _, ok := allowedChatVideoExtensions[ext]; ok {
		return fileName
	}
	base := strings.TrimSpace(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	base = strings.Trim(base, ". ")
	if base == "" {
		base = "video"
	}
	return base + ".mp4"
}

func videoAttachmentContentTypeByExtension(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".m4v":
		return "video/x-m4v"
	case ".webm":
		return "video/webm"
	case ".3gp", ".3gpp":
		return "video/3gpp"
	case ".avi":
		return "video/x-msvideo"
	case ".mkv":
		return "video/x-matroska"
	case ".flv":
		return "video/x-flv"
	case ".wmv":
		return "video/x-ms-wmv"
	default:
		return ""
	}
}

func isVideoAttachment(contentType string, fileName string) bool {
	if strings.HasPrefix(strings.ToLower(contentType), "video/") {
		return true
	}
	_, ok := allowedChatVideoExtensions[strings.ToLower(filepath.Ext(fileName))]
	return ok
}

func buildAttachmentObjectKey(imUserID string, fileID string, fileName string) string {
	dateSegment := time.Now().Format("20060102")
	userSegment := sanitizeObjectSegment(imUserID)
	ext := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	base = sanitizeObjectSegment(base)
	if base == "anonymous" {
		base = "file"
	}
	return fmt.Sprintf("chat-files/%s/%s/%s/%s%s", dateSegment, userSegment, fileID, base, ext)
}

func chatAttachmentCOSConfig() (config.COS, error) {
	cfg := config.Cfg.Get().COS
	chatCfg := config.COS{
		SecretID:  strings.TrimSpace(cfg.ChatSecretID),
		SecretKey: strings.TrimSpace(cfg.ChatSecretKey),
		Bucket:    strings.TrimSpace(cfg.ChatBucket),
		Region:    strings.TrimSpace(cfg.ChatRegion),
	}
	if chatCfg.SecretID == "" || chatCfg.SecretKey == "" || chatCfg.Bucket == "" || chatCfg.Region == "" {
		return config.COS{}, errChatAttachmentCOSConfigIncomplete
	}
	return chatCfg, nil
}

func signedAttachmentGetURL(
	attachment model.ChatAttachment,
	settings config.ChatAttachmentUploadSettings,
	forceDownload bool,
) (string, error) {
	if attachment.ExpiresAt != nil && !attachment.ExpiresAt.After(time.Now()) {
		return "", errors.New("文件已过期")
	}
	cosCfg, err := chatAttachmentCOSConfig()
	if err != nil {
		return "", err
	}
	client, err := newCOSClient(cosCfg)
	if err != nil {
		return "", err
	}

	disposition := "inline"
	if forceDownload {
		disposition = "attachment"
	}
	opt := &cos.ObjectGetOptions{
		ResponseContentType:        attachment.ContentType,
		ResponseContentDisposition: buildContentDisposition(disposition, attachment.FileName),
	}
	signedURL, err := client.Object.GetPresignedURL(
		context.Background(),
		http.MethodGet,
		attachment.ObjectKey,
		cosCfg.SecretID,
		cosCfg.SecretKey,
		time.Duration(settings.GetURLTTLSeconds)*time.Second,
		opt,
		false,
	)
	if err != nil {
		return "", err
	}
	return signedURL.String(), nil
}

func buildContentDisposition(disposition string, fileName string) string {
	fallback := sanitizeDispositionFilename(fileName)
	escaped := strings.ReplaceAll(url.PathEscape(fileName), "+", "%20")
	return fmt.Sprintf("%s; filename=%q; filename*=UTF-8''%s", disposition, fallback, escaped)
}

func sanitizeDispositionFilename(fileName string) string {
	var builder strings.Builder
	for _, char := range fileName {
		if char >= 32 && char <= 126 && char != '"' && char != '\\' {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "file"
	}
	return builder.String()
}

func buildAttachmentDownloadURL(c *gin.Context, fileID string, token string, forceDownload bool) string {
	base := requestBaseURL(c)
	values := url.Values{}
	values.Set("token", token)
	if forceDownload {
		values.Set("download", "1")
	}
	return fmt.Sprintf("%s/api/v1/chat/files/%s/download?%s", base, url.PathEscape(fileID), values.Encode())
}

func requestBaseURL(c *gin.Context) string {
	proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if proto == "" {
		if c.Request != nil && c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" && c.Request != nil {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s", proto, host)
}

func randomHexID(prefix string, byteCount int) (string, error) {
	buf := make([]byte, byteCount)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}

func renderVideoPreviewHTML(attachment model.ChatAttachment, videoURL string, downloadURL string) string {
	title := html.EscapeString(attachment.FileName)
	videoSrc := html.EscapeString(videoURL)
	downloadHref := html.EscapeString(downloadURL)
	sizeText := html.EscapeString(formatAttachmentSize(attachment.FileSize))
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + title + `</title>
  <style>
    body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #111827; color: #f8fafc; }
    .wrap { min-height: 100vh; display: flex; flex-direction: column; box-sizing: border-box; padding: 20px; gap: 16px; }
    .title { font-size: 17px; font-weight: 700; line-height: 1.4; word-break: break-word; }
    .meta { color: #cbd5e1; font-size: 13px; }
    video { width: 100%; max-height: 74vh; background: #000; border-radius: 8px; }
    .actions { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
    a { color: #dbeafe; border: 1px solid rgba(219,234,254,.35); border-radius: 6px; padding: 9px 12px; text-decoration: none; }
  </style>
</head>
<body>
  <main class="wrap">
    <div>
      <div class="title">` + title + `</div>
      <div class="meta">` + sizeText + `</div>
    </div>
    <video controls playsinline preload="metadata" src="` + videoSrc + `"></video>
    <div class="actions"><a href="` + downloadHref + `">下载原文件</a></div>
  </main>
</body>
</html>`
}

func formatAttachmentSize(size int64) string {
	if size <= 0 {
		return ""
	}
	const mb = 1024 * 1024
	if size >= mb {
		return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
	}
	return fmt.Sprintf("%.1f KB", float64(size)/1024)
}
