package tencentim

import (
	"bytes"
	"compress/zlib"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GenUserSig 生成腾讯云 IM UserSig（HMAC-SHA256 / TLS 2.0）。
// expire: 有效期（秒），建议 604800（7 天）。
func GenUserSig(sdkAppID int, secretKey, userID string, expire int64) (string, error) {
	now := time.Now().Unix()

	// 1. 拼接待签名字符串
	content := strings.Join([]string{
		"TLS.identifier:" + userID,
		"TLS.sdkappid:" + strconv.Itoa(sdkAppID),
		"TLS.time:" + strconv.FormatInt(now, 10),
		"TLS.expire:" + strconv.FormatInt(expire, 10),
	}, "\n") + "\n"

	// 2. HMAC-SHA256 签名 → Base64
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(content))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 3. 构造 JSON Payload
	payload := map[string]any{
		"TLS.ver":        "2.0",
		"TLS.identifier": userID,
		"TLS.sdkappid":   sdkAppID,
		"TLS.expire":     expire,
		"TLS.time":       now,
		"TLS.sig":        sig,
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("usersig: marshal payload: %w", err)
	}

	// 4. zlib 压缩
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err = w.Write(jsonBytes); err != nil {
		return "", fmt.Errorf("usersig: zlib write: %w", err)
	}
	if err = w.Close(); err != nil {
		return "", fmt.Errorf("usersig: zlib close: %w", err)
	}

	// 5. Base64 编码，腾讯改版：+ → *，/ → -，= → _
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	var sb strings.Builder
	sb.Grow(len(b64))
	for _, ch := range b64 {
		switch ch {
		case '+':
			sb.WriteByte('*')
		case '/':
			sb.WriteByte('-')
		case '=':
			sb.WriteByte('_')
		default:
			sb.WriteRune(ch)
		}
	}
	return sb.String(), nil
}
