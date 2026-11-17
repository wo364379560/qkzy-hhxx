package tencentim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/config"
)

const baseURL = "https://console.tim.qq.com/v4"

// Client 是腾讯云 IM REST API 客户端。
type Client struct {
	sdkAppID  int
	appKey    string
	adminUser string
	http      *http.Client
}

// NewClient 创建腾讯云 IM 客户端。
func NewClient() *Client {
	cfg := config.Cfg.Get().TencentIM
	return &Client{
		sdkAppID:  cfg.SDKAppID,
		appKey:    cfg.AppKey,
		adminUser: cfg.AdminUser,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

// IsConfigured 检查腾讯 IM 是否已配置（SDKAppID 和 AppKey 均不为空）。
func (c *Client) IsConfigured() bool {
	return c.sdkAppID != 0 && c.appKey != ""
}

const adminSigExpire = int64(24 * 60 * 60) // 管理员 sig 有效期 1 天

// sigCache 包级别的管理员 UserSig 缓存，避免每次请求都重新计算 HMAC-SHA256。
var sigCache struct {
	sync.Mutex
	value  string
	expiry time.Time
}

// userSig 返回管理员 UserSig，有效期内复用缓存。
func (c *Client) userSig() string {
	sigCache.Lock()
	defer sigCache.Unlock()

	// 提前 30 分钟刷新，留出网络请求裕量
	if sigCache.value != "" && time.Now().Add(30*time.Minute).Before(sigCache.expiry) {
		return sigCache.value
	}

	sig, err := GenUserSig(c.sdkAppID, c.appKey, c.adminUser, adminSigExpire)
	if err != nil {
		return ""
	}
	sigCache.value = sig
	sigCache.expiry = time.Now().Add(time.Duration(adminSigExpire) * time.Second)
	return sig
}

// buildURL 构建 REST API 请求 URL。
func (c *Client) buildURL(serviceName, command string) string {
	nonce := rand.Intn(999999)
	sig := c.userSig()
	return fmt.Sprintf("%s/%s/%s?sdkappid=%d&identifier=%s&usersig=%s&random=%d&contenttype=json",
		baseURL, serviceName, command,
		c.sdkAppID, c.adminUser, sig, nonce,
	)
}

func (c *Client) post(url string, reqBody, respBody any) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if respBody != nil {
		return json.NewDecoder(resp.Body).Decode(respBody)
	}
	return nil
}

// postContext bounds the whole ownership proof request and never returns a URL
// containing the administrator UserSig in transport errors.
func (c *Client) postContext(ctx context.Context, service, command string, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(service, command), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid tencent request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("tencent request unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("tencent HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
}
