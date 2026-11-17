// cos-check verifies the configured avatar bucket without reading user objects.
// Credentials come only from the server environment/.env and are never printed.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/tencentyun/cos-go-sdk-v5"
)

func report(stage string, response *cos.Response, err error) {
	row := map[string]any{"stage": stage, "ok": err == nil}
	if response != nil {
		row["http_status"] = response.StatusCode
		row["request_id"] = response.Header.Get("X-Cos-Request-Id")
	}
	var service *cos.ErrorResponse
	if errors.As(err, &service) {
		row["cos_code"] = service.Code
		row["request_id"] = service.RequestID
	}
	if err != nil && service == nil {
		row["error"] = "network_or_configuration_error"
	}
	_ = json.NewEncoder(os.Stdout).Encode(row)
}

func main() {
	write := flag.Bool("write-check", false, "write/read/delete one unique diagnostic image")
	flag.Parse()
	_ = config.Setup()
	cfg := config.Cfg.Get().COS
	endpoint, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region))
	if err != nil || cfg.Bucket == "" || cfg.SecretID == "" || cfg.SecretKey == "" {
		fmt.Println("COS configuration incomplete")
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"bucket": cfg.Bucket, "region": cfg.Region, "host": endpoint.Host, "write_check": *write})
	if !*write {
		return
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: endpoint}, &http.Client{Timeout: 30 * time.Second, Transport: &cos.AuthorizationTransport{SecretID: cfg.SecretID, SecretKey: cfg.SecretKey}})
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		os.Exit(1)
	}
	key := "rui-diagnostics/avatar-" + hex.EncodeToString(nonce[:]) + ".png"
	var payload bytes.Buffer
	_ = png.Encode(&payload, image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	failed := false
	response, err := client.Object.Put(ctx, key, bytes.NewReader(payload.Bytes()), &cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: "image/png"}})
	report("put", response, err)
	if err != nil {
		failed = true
	} else {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String()+"/"+key, nil)
		public, getErr := (&http.Client{Timeout: 20 * time.Second}).Do(request)
		if getErr == nil {
			content, readErr := io.ReadAll(io.LimitReader(public.Body, 4096))
			_ = public.Body.Close()
			ok := public.StatusCode == 200 && readErr == nil && bytes.Equal(content, payload.Bytes())
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"stage": "unsigned_avatar_read", "http_status": public.StatusCode, "ok": ok})
			failed = !ok
		} else {
			report("unsigned_avatar_read", nil, getErr)
			failed = true
		}
	}
	// Delete only our random key, even if the upload's acknowledgment was lost.
	cleanup, stop := context.WithTimeout(context.Background(), 20*time.Second)
	defer stop()
	response, err = client.Object.Delete(cleanup, key)
	report("cleanup_own_object", response, err)
	if failed || err != nil {
		os.Exit(1)
	}
}
