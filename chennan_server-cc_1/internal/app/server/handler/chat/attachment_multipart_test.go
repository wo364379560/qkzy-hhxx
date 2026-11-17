package handler_chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/tencentyun/cos-go-sdk-v5"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type zeroStream struct{}

func (zeroStream) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestMultipartConfirmsLostMergeResponseAndStreamsFullHash(t *testing.T) {
	// Storage transport is local and streams the payload, including 500 MiB.
	// No Tencent credentials, real buckets or application .env are involved.
	for _, megabytes := range []int64{1, 120, 500} {
		t.Run(fmt.Sprint(megabytes), func(t *testing.T) {
			size := megabytes << 20
			hash := sha256.New()
			_, _ = io.Copy(hash, io.LimitReader(zeroStream{}, size))
			fingerprint := hex.EncodeToString(hash.Sum(nil))
			reads := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" || r.Method != http.MethodHead && r.Method != http.MethodGet {
					t.Errorf("lost merge was repeated: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Content-Length", fmt.Sprint(size))
				if r.Method == http.MethodGet {
					reads++
					_, _ = io.Copy(w, io.LimitReader(zeroStream{}, size))
				}
			}))
			defer server.Close()
			endpoint, _ := url.Parse(server.URL)
			client := cos.NewClient(&cos.BaseURL{BucketURL: endpoint}, server.Client())
			connection, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: connection, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE `chat_attachments` SET .*uploaded_at.* WHERE file_id = \\? AND status = \\?").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			err = finishMultipart(context.Background(), db, client, model.ChatAttachment{FileID: "local-test", ObjectKey: "local-object", FileSize: size}, model.ChatMultipartUpload{Fingerprint: fingerprint})
			if err != nil {
				t.Fatal(err)
			}
			if reads != 1 {
				t.Fatalf("expected a single streamed verification, got %d", reads)
			}
			if err = mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMultipartRejectsCorruptContentBeforePublishing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "3")
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, "bad")
		}
	}))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	err := finishMultipart(context.Background(), nil, cos.NewClient(&cos.BaseURL{BucketURL: endpoint}, server.Client()), model.ChatAttachment{ObjectKey: "local-object", FileSize: 3}, model.ChatMultipartUpload{Fingerprint: fmt.Sprintf("%064d", 0)})
	if err != errMultipartHash {
		t.Fatalf("corrupt file became publishable: %v", err)
	}
}

func TestMultipartRejectsMissingReorderedOrWrongSizeParts(t *testing.T) {
	valid := []cos.Object{{PartNumber: 1, Size: multipartPartSize, ETag: "first"}, {PartNumber: 2, Size: 1, ETag: "last"}}
	if err := validateMultipartParts(valid, multipartPartSize+1, multipartPartSize); err != nil {
		t.Fatal(err)
	}
	for _, parts := range [][]cos.Object{valid[:1], {valid[1], valid[0]}, {{PartNumber: 1, Size: multipartPartSize, ETag: ""}, valid[1]}, {{PartNumber: 1, Size: 4, ETag: "first"}, valid[1]}} {
		if validateMultipartParts(parts, multipartPartSize+1, multipartPartSize) == nil {
			t.Fatal("invalid parts were accepted")
		}
	}
}
