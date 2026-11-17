package tencentim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestForwardOldSDKImageTypesCopiedForREST(t *testing.T) {
	raw := json.RawMessage(`{"UUID":"old-media-uuid","ImageFormat":1,"Extra":{"preserve":true},"ImageInfoArray":[{"Type":0,"Size":100,"Width":800,"Height":600,"URL":"https://image.invalid/original"},{"Type":1,"Size":10,"Width":80,"Height":60,"URL":"https://image.invalid/thumb"},{"Type":2,"Size":50,"Width":400,"Height":300,"URL":"https://image.invalid/large"}]}`)
	body := []MessageElement{{MsgType: "TIMTextElem", MsgContent: json.RawMessage(`{"Text":"before"}`)}, {MsgType: "TIMImageElem", MsgContent: raw}}
	before, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	converted, err := normalizeForwardImages(body)
	if err != nil {
		t.Fatal(err)
	}
	var content struct {
		UUID           string
		ImageFormat    int
		Extra          map[string]bool
		ImageInfoArray []struct {
			Type                int
			Size, Width, Height int
			URL                 string
		}
	}
	if err := json.Unmarshal(converted[1].MsgContent, &content); err != nil {
		t.Fatal(err)
	}
	if content.UUID != "old-media-uuid" || content.ImageFormat != 1 || !content.Extra["preserve"] || !reflect.DeepEqual([]int{content.ImageInfoArray[0].Type, content.ImageInfoArray[1].Type, content.ImageInfoArray[2].Type}, []int{1, 3, 2}) {
		t.Fatalf("wrong legacy image normalization: %+v", content)
	}
	if content.ImageInfoArray[0].URL != "https://image.invalid/original" || content.ImageInfoArray[1].Size != 10 || content.ImageInfoArray[2].Width != 400 || string(converted[0].MsgContent) != string(body[0].MsgContent) {
		t.Fatal("normalization changed media fields or another message")
	}
	after, _ := json.Marshal(body)
	if string(after) != string(before) || string(body[1].MsgContent) != string(raw) {
		t.Fatal("normalization mutated the persisted/original message")
	}
	convertedAgain, err := normalizeForwardImages(converted)
	if err != nil || string(convertedAgain[1].MsgContent) != string(converted[1].MsgContent) {
		t.Fatal("normalization must be idempotent for REST types")
	}
}

func TestForwardRESTImageTypesRemainByteIdentical(t *testing.T) {
	for _, types := range []string{`{"Type":1},{"Type":3},{"Type":2}`, `{"Type":1}`, `{"Type":2}`, `{"Type":3}`} {
		raw := json.RawMessage(fmt.Sprintf(`{"UUID":"rest","ImageInfoArray":[%s]}`, types))
		body := []MessageElement{{MsgType: "TIMImageElem", MsgContent: raw}}
		converted, err := normalizeForwardImages(body)
		if err != nil || string(converted[0].MsgContent) != string(raw) {
			t.Fatalf("existing REST image was reinterpreted: %s (%v)", types, err)
		}
	}
}

func TestForwardMixedImageTypesAreNotGuessedOrSent(t *testing.T) {
	for _, types := range []string{
		`{"Type":0},{"Type":3}`, `{"Type":0},{"Type":-1}`, `{"Type":0},{"Type":4}`,
		`{"Type":0},{"Type":"1"}`, `{"Type":0},{"Type":1.5}`, `{"Type":0},{"Type":null}`, `{"Type":0},{"URL":"https://image.invalid/no-type"}`, `{"Type":0},null`,
	} {
		t.Run(types, func(t *testing.T) {
			calls := 0
			client := forwardTestClient(func(*http.Request) (*http.Response, error) {
				calls++
				return forwardReply(`{"ActionStatus":"OK","ErrorCode":0}`), nil
			})
			raw := json.RawMessage(fmt.Sprintf(`{"UUID":"mixed","ImageInfoArray":[%s]}`, types))
			_, err := client.SendForward(context.Background(), "alice", "GROUPg", []MessageElement{{MsgType: "TIMImageElem", MsgContent: raw}}, 71, 2, nil)
			var detail *DeliveryError
			if calls != 0 || !errors.As(err, &detail) || detail.Unknown || detail.Code != 400 || detail.Command != "validate_forward_message" || !strings.Contains(detail.Message, "无法安全转换") {
				t.Fatalf("ambiguous legacy image dispatched or lost diagnostic: %v, calls=%d", err, calls)
			}
		})
	}
}

func TestForwardOldImageRetryUsesSameMessageIdentity(t *testing.T) {
	for _, target := range []string{"GROUPg", "C2Cbob"} {
		t.Run(target, func(t *testing.T) {
			var originalRequest string
			calls := 0
			client := forwardTestClient(func(r *http.Request) (*http.Response, error) {
				calls++
				bytes, _ := io.ReadAll(r.Body)
				if calls == 1 {
					originalRequest = string(bytes)
				} else if string(bytes) != originalRequest {
					t.Fatal("legacy image retry changed serialized message identity/body")
				}
				var request struct {
					Random, MsgRandom, MsgSeq uint32
					MsgBody                   []MessageElement
				}
				if err := json.Unmarshal(bytes, &request); err != nil {
					t.Fatal(err)
				}
				var content struct{ ImageInfoArray []struct{ Type int } }
				if err := json.Unmarshal(request.MsgBody[0].MsgContent, &content); err != nil || content.ImageInfoArray[0].Type != 1 || content.ImageInfoArray[1].Type != 3 {
					t.Fatal("SendForward did not normalize the stored legacy payload")
				}
				if strings.HasPrefix(target, "GROUP") && request.Random != 71 || strings.HasPrefix(target, "C2C") && (request.MsgRandom != 71 || request.MsgSeq != 2) {
					t.Fatal("legacy normalization replaced the stored Random/Sequence")
				}
				return forwardReply(`{"ActionStatus":"OK","ErrorCode":0,"MsgId":"test-ack"}`), nil
			})
			raw := json.RawMessage(`{"UUID":"legacy","ImageFormat":1,"ImageInfoArray":[{"Type":0,"Size":100,"Width":20,"Height":10,"URL":"https://image.invalid/original"},{"Type":1,"Size":10,"Width":2,"Height":1,"URL":"https://image.invalid/thumb"},{"Type":2,"Size":50,"Width":10,"Height":5,"URL":"https://image.invalid/large"}]}`)
			body := []MessageElement{{MsgType: "TIMImageElem", MsgContent: raw}}
			for i := 0; i < 2; i++ {
				if _, err := client.SendForward(context.Background(), "alice", target, body, 71, 2, nil); err != nil {
					t.Fatal(err)
				}
			}
			if !strings.Contains(string(body[0].MsgContent), `"Type":0`) {
				t.Fatal("dispatch rewrote durable source body")
			}
		})
	}
}
