package handler_chat

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/base/response"
)

type avatarPresignData struct {
	PutURL    string `json:"put_url"`
	AvatarURL string `json:"avatar_url"`
}

type mediaPresignData struct {
	PutURL    string `json:"put_url"`
	FileURL   string `json:"file_url"`
	ObjectKey string `json:"object_key"`
	MaxSizeMB int64  `json:"max_size_mb"`
}

type avatarPresignRequest struct {
	Type    string `form:"type"`     // "user" 或 "group"
	UserID  string `form:"user_id"`  // 用户 IM ID（腾讯云原生 ID，如 u_xxx）
	GroupID string `form:"group_id"` // 群组 IM ID（腾讯云原生 ID，如 @TGS#xxx）
	BizID   int64  `form:"biz_id"`   // 业务 ID（igs.Snowflake ID，用于文件路径，用户和群组通用）
}

type mediaPresignRequest struct {
	MediaType   string `form:"media_type"`
	FileName    string `form:"file_name"`
	FileSize    int64  `form:"file_size"`
	ContentType string `form:"content_type"`
}

type avatarUploadData struct {
	AvatarURL string `json:"avatar_url"`
}

const maxChatVideoUploadSizeBytes int64 = 200 * 1024 * 1024
const maxAvatarUploadSizeBytes int64 = 50 * 1024 * 1024

type avatarContentType struct {
	ContentType string
	Extension   string
}

var allowedAvatarContentTypes = map[string]avatarContentType{
	"image/jpeg": {ContentType: "image/jpeg", Extension: ".jpg"},
	"image/png":  {ContentType: "image/png", Extension: ".png"},
	"image/webp": {ContentType: "image/webp", Extension: ".webp"},
	"image/gif":  {ContentType: "image/gif", Extension: ".gif"},
}

var allowedChatVideoExtensions = map[string]struct{}{
	".mp4":  {},
	".mov":  {},
	".m4v":  {},
	".avi":  {},
	".mkv":  {},
	".webm": {},
	".3gp":  {},
	".3gpp": {},
	".flv":  {},
	".wmv":  {},
}

func normalizeAvatarUploadRequest(req avatarPresignRequest) (avatarPresignRequest, error) {
	if req.Type == "" {
		req.Type = "user"
	}
	if req.Type == "group" {
		if req.GroupID == "" {
			return req, fmt.Errorf("group_id is required for group avatar")
		}
	}
	if req.Type == "user" {
		if req.UserID == "" {
			return req, fmt.Errorf("user_id is required for user avatar")
		}
	}
	// BizID 可以为 0，此时会从数据库查询
	if req.Type != "user" && req.Type != "group" {
		return req, fmt.Errorf("type must be user or group")
	}
	return req, nil
}

// resolveBizID 根据 IM ID 解析业务 ID，如果传入的 bizID 为 0 则从数据库查询
func resolveBizID(req avatarPresignRequest) (int64, error) {
	if req.BizID > 0 {
		return req.BizID, nil
	}

	if req.Type == "user" {
		var user model.User
		if err := datasource.Datasource.Mysql.GetClient().Where("im_user_id = ?", req.UserID).First(&user).Error; err != nil {
			// 回退为基于用户 IM ID 的稳定哈希，避免因本地索引缺失导致头像上传失败。
			return fallbackBizID(req.UserID), nil
		}
		if user.BizID == 0 {
			return fallbackBizID(req.UserID), nil
		}
		return user.BizID, nil
	}

	if req.Type == "group" {
		var group model.Group
		if err := datasource.Datasource.Mysql.GetClient().Where("im_group_id = ?", req.GroupID).First(&group).Error; err != nil {
			// 回退为基于群组 IM ID 的稳定哈希，避免因本地群记录缺失导致上传失败。
			return fallbackBizID(req.GroupID), nil
		}
		if group.BizID == 0 {
			return fallbackBizID(req.GroupID), nil
		}
		return group.BizID, nil
	}

	return 0, fmt.Errorf("unknown type")
}

func fallbackBizID(identifier string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(identifier))
	value := int64(hasher.Sum64() & 0x7fffffffffffffff)
	if value == 0 {
		return time.Now().UnixNano()
	}
	return value
}

func newCOSClient(cfg config.COS) (*cos.Client, error) {
	if cfg.Bucket == "" || cfg.SecretID == "" || cfg.SecretKey == "" || cfg.Region == "" {
		return nil, fmt.Errorf("cos config is incomplete")
	}

	bucketURL, err := url.Parse(fmt.Sprintf(
		"https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region,
	))
	if err != nil {
		return nil, err
	}

	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Timeout: 30 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	})
	return client, nil
}

func buildAvatarObjectKey(req avatarPresignRequest, ext string) string {
	timestamp := time.Now().UnixNano()
	if ext == "" {
		ext = ".jpg"
	}

	if req.Type == "group" {
		// 使用业务 ID（纯数字）作为路径前缀
		return fmt.Sprintf("group-avatars/%d/%d%s", req.BizID, timestamp, ext)
	}

	// 使用业务 ID（纯数字）作为路径前缀
	return fmt.Sprintf("avatars/%d/%d%s", req.BizID, timestamp, ext)
}

func buildAvatarURL(cfg config.COS, key string) string {
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", cfg.Bucket, cfg.Region, key)
}

func normalizeMediaPresignRequest(req mediaPresignRequest) (mediaPresignRequest, error) {
	if req.MediaType == "" {
		req.MediaType = "video"
	}
	if req.MediaType != "video" {
		return req, fmt.Errorf("media_type must be video")
	}
	if req.FileSize <= 0 {
		return req, fmt.Errorf("file_size is required")
	}
	if req.FileSize > maxChatVideoUploadSizeBytes {
		return req, fmt.Errorf("video file size must not exceed 200MB")
	}

	contentType := strings.ToLower(strings.TrimSpace(req.ContentType))
	ext := normalizeVideoExtension(req.FileName)
	if !strings.HasPrefix(contentType, "video/") {
		if _, ok := allowedChatVideoExtensions[ext]; !ok {
			return req, fmt.Errorf("unsupported video type")
		}
	}
	if ext == "" {
		ext = ".mp4"
	}
	req.FileName = ext
	return req, nil
}

func normalizeVideoExtension(fileName string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	if _, ok := allowedChatVideoExtensions[ext]; ok {
		return ext
	}
	return ""
}

func sanitizeObjectSegment(value string) string {
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '_' || char == '-':
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "anonymous"
	}
	return builder.String()
}

func buildMediaObjectKey(imUserID string, req mediaPresignRequest) string {
	userSegment := sanitizeObjectSegment(imUserID)
	dateSegment := time.Now().Format("20060102")
	return fmt.Sprintf("chat-media/videos/%s/%s/%d%s", dateSegment, userSegment, time.Now().UnixNano(), req.FileName)
}

func detectAvatarContentType(file io.ReadSeeker, declared string) (avatarContentType, error) {
	head := make([]byte, 512)
	n, readErr := file.Read(head)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return avatarContentType{}, readErr
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return avatarContentType{}, err
	}

	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(head[:n])))
	if allowed, ok := allowedAvatarContentTypes[detected]; ok {
		return allowed, nil
	}

	declared = strings.ToLower(strings.TrimSpace(strings.Split(declared, ";")[0]))
	if allowed, ok := allowedAvatarContentTypes[declared]; ok && n == 0 {
		return allowed, nil
	}

	return avatarContentType{}, fmt.Errorf("unsupported avatar content type: detected=%s declared=%s", detected, declared)
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// UploadAvatarPresign 生成头像上传的 COS 预签名 PUT URL（5 分钟有效）。
// 支持用户头像和群组头像：
// - 用户头像：GET /upload/avatar/presign?type=user&user_id=u_xxx&biz_id=123
// - 群组头像：GET /upload/avatar/presign?type=group&group_id=@TGS#xxx&biz_id=456
// 如果 biz_id 为 0，会自动从数据库查询
func UploadAvatarPresign(c *gin.Context) {
	var req avatarPresignRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid parameters")
		return
	}
	req, err := normalizeAvatarUploadRequest(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := controller_chat.AuthorizeAvatarUpload(c.GetString("im_user_id"), req.Type, req.UserID, req.GroupID); err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	req.BizID = 0 // Resolve the authenticated resource's ID; do not trust a form path.

	// 解析 biz_id，如果为 0 则从数据库查询
	bizID, err := resolveBizID(req)
	if err != nil {
		response.InternalError(c)
		return
	}
	req.BizID = bizID

	cfg := config.Cfg.Get().COS
	client, err := newCOSClient(cfg)
	if err != nil {
		response.InternalError(c)
		return
	}

	key := buildAvatarObjectKey(req, ".jpg")

	putURL, err := client.Object.GetPresignedURL(
		context.Background(),
		http.MethodPut,
		key,
		cfg.SecretID,
		cfg.SecretKey,
		5*time.Minute,
		nil,
		false,
	)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, avatarPresignData{
		PutURL:    putURL.String(),
		AvatarURL: buildAvatarURL(cfg, key),
	})
}

// UploadMediaPresign 生成 H5 聊天大视频上传的 COS 预签名 PUT URL（5 分钟有效）。
// 当前只允许 video，最大 200MB。100MB 以内仍优先走腾讯 IM 原生视频消息。
func UploadMediaPresign(c *gin.Context) {
	var req mediaPresignRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "invalid parameters")
		return
	}
	req, err := normalizeMediaPresignRequest(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	cfg := config.Cfg.Get().COS
	client, err := newCOSClient(cfg)
	if err != nil {
		response.InternalError(c)
		return
	}

	key := buildMediaObjectKey(c.GetString("im_user_id"), req)
	putURL, err := client.Object.GetPresignedURL(
		context.Background(),
		http.MethodPut,
		key,
		cfg.SecretID,
		cfg.SecretKey,
		5*time.Minute,
		nil,
		false,
	)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, mediaPresignData{
		PutURL:    putURL.String(),
		FileURL:   buildAvatarURL(cfg, key),
		ObjectKey: key,
		MaxSizeMB: maxChatVideoUploadSizeBytes / 1024 / 1024,
	})
}

// UploadAvatar 接收浏览器上传的头像文件，并由服务端写入 COS。
// 支持用户头像和群组头像：
// - 用户头像：POST /upload/avatar?type=user&user_id=u_xxx&biz_id=123
// - 群组头像：POST /upload/avatar?type=group&group_id=@TGS#xxx&biz_id=456
// 如果 biz_id 为 0，会自动从数据库查询
func UploadAvatar(c *gin.Context) {
	var req avatarPresignRequest
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, "invalid parameters")
		return
	}
	req, err := normalizeAvatarUploadRequest(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := controller_chat.AuthorizeAvatarUpload(c.GetString("im_user_id"), req.Type, req.UserID, req.GroupID); err != nil {
		response.Forbidden(c, err.Error())
		return
	}
	req.BizID = 0

	// 解析 biz_id，如果为 0 则从数据库查询
	bizID, err := resolveBizID(req)
	if err != nil {
		response.InternalError(c)
		return
	}
	req.BizID = bizID

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	if fileHeader.Size <= 0 {
		response.BadRequest(c, "头像文件不能为空")
		return
	}
	if fileHeader.Size > maxAvatarUploadSizeBytes {
		response.BadRequest(c, "头像文件不能超过50MB")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.InternalError(c)
		return
	}
	defer func() {
		_ = file.Close()
	}()
	content, err := detectAvatarContentType(file, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		response.BadRequest(c, "仅支持 JPG、PNG、WEBP、GIF 格式头像")
		return
	}

	cfg := config.Cfg.Get().COS
	client, err := newCOSClient(cfg)
	if err != nil {
		response.InternalError(c)
		return
	}

	key := buildAvatarObjectKey(req, content.Extension)
	reader, ok := file.(io.Reader)
	if !ok {
		response.InternalError(c)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: content.ContentType,
		},
	}
	if _, err = client.Object.Put(ctx, key, reader, opt); err != nil {
		writeAvatarUploadFailure(c, err, "cos_put")
		return
	}

	response.OK(c, avatarUploadData{
		AvatarURL: buildAvatarURL(cfg, key),
	})
}
