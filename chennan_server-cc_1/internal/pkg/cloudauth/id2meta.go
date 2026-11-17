package cloudauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	endpoint        = "https://cloudauth.aliyuncs.com"
	apiVersion      = "2019-03-07"
	bizCodeMatch    = "1"
	bizCodeMismatch = "2"
)

var ErrNotConfigured = errors.New("实名服务未配置")

// Credentials 阿里云实人认证 AccessKey（仅从环境变量读取）。
type Credentials struct {
	AccessKeyID     string
	AccessKeySecret string
}

// VerifyResult 二要素核验结果。
type VerifyResult struct {
	RequestID string
	BizCode   string
	Matched   bool
}

// LoadCredentialsFromEnv 从环境变量加载 AccessKey。
func LoadCredentialsFromEnv() (Credentials, error) {
	creds := Credentials{
		AccessKeyID:     strings.TrimSpace(os.Getenv("ALIYUN_CLOUDAUTH_ACCESS_KEY_ID")),
		AccessKeySecret: strings.TrimSpace(os.Getenv("ALIYUN_CLOUDAUTH_ACCESS_KEY_SECRET")),
	}
	if creds.AccessKeyID == "" || creds.AccessKeySecret == "" {
		return Credentials{}, ErrNotConfigured
	}
	return creds, nil
}

// Id2MetaVerify 调用阿里云 Id2MetaVerify 接口核验姓名与身份证号。
func Id2MetaVerify(creds Credentials, userName, identifyNum string) (*VerifyResult, error) {
	if creds.AccessKeyID == "" || creds.AccessKeySecret == "" {
		return nil, ErrNotConfigured
	}

	nonce, err := genNonce()
	if err != nil {
		return nil, fmt.Errorf("cloudauth: gen nonce: %w", err)
	}

	params := map[string]string{
		"Action":           "Id2MetaVerify",
		"Version":          apiVersion,
		"Format":           "JSON",
		"RegionId":         "cn-hangzhou",
		"AccessKeyId":      creds.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   nonce,
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"ParamType":        "normal",
		"UserName":         userName,
		"IdentifyNum":      identifyNum,
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, aliyunEncode(k)+"="+aliyunEncode(params[k]))
	}
	queryStr := strings.Join(parts, "&")

	stringToSign := "POST&%2F&" + aliyunEncode(queryStr)
	mac := hmac.New(sha1.New, []byte(creds.AccessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	params["Signature"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("cloudauth: http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cloudauth: read body: %w", err)
	}

	var apiResp struct {
		RequestID    string `json:"RequestId"`
		Code         string `json:"Code"`
		Message      string `json:"Message"`
		ResultObject struct {
			BizCode string `json:"BizCode"`
		} `json:"ResultObject"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("cloudauth: parse response: %w", err)
	}
	if apiResp.Code != "200" {
		msg := strings.TrimSpace(apiResp.Message)
		if msg == "" {
			msg = "实名核验服务调用失败"
		}
		return nil, fmt.Errorf("cloudauth: %s (code=%s)", msg, apiResp.Code)
	}

	bizCode := strings.TrimSpace(apiResp.ResultObject.BizCode)
	return &VerifyResult{
		RequestID: apiResp.RequestID,
		BizCode:   bizCode,
		Matched:   bizCode == bizCodeMatch,
	}, nil
}

// HashIDNumber 计算身份证号 SHA-256 摘要（大写 X）。
func HashIDNumber(idNumber string) string {
	normalized := strings.ToUpper(strings.TrimSpace(idNumber))
	sum := sha256Sum(normalized)
	return sum
}

// MaskIDNumber 脱敏身份证号：前 3 位 + **** + 后 2 位。
func MaskIDNumber(idNumber string) string {
	normalized := strings.ToUpper(strings.TrimSpace(idNumber))
	runes := []rune(normalized)
	if len(runes) <= 5 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:3]) + strings.Repeat("*", len(runes)-5) + string(runes[len(runes)-2:])
}

// MaskRealName 脱敏姓名：保留首字，其余以 * 代替（如 李家林 → 李**）。
func MaskRealName(name string) string {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	switch len(runes) {
	case 0:
		return ""
	case 1:
		return string(runes[0]) + "*"
	default:
		return string(runes[0]) + strings.Repeat("*", len(runes)-1)
	}
}

func sha256Sum(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}

func genNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func aliyunEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
