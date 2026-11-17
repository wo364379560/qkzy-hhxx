package tencentim

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func forwardMediaFixture(kind string) map[string]any {
	switch kind {
	case "TIMImageElem":
		return map[string]any{"UUID": "asset-image", "ImageFormat": 3, "ImageInfoArray": []any{map[string]any{"Type": 1, "Size": 100, "Width": 20, "Height": 10, "URL": "https://media.invalid/picture.png?token=must-not-leak"}}}
	case "TIMVideoFileElem":
		return map[string]any{"VideoUUID": "asset-video", "VideoUrl": "https://media.invalid/movie.mp4", "VideoSize": 1200, "VideoSecond": 0, "VideoFormat": "mp4", "VideoDownloadFlag": 2, "ThumbUUID": "asset-thumb", "ThumbUrl": "https://media.invalid/thumb.jpg", "ThumbSize": 100, "ThumbWidth": 20, "ThumbHeight": 10, "ThumbDownloadFlag": 2}
	case "TIMFileElem":
		return map[string]any{"UUID": "asset-file", "Url": "https://media.invalid/document.tn6", "FileSize": 500, "FileName": "文件.tn6", "Download_Flag": 2}
	case "TIMSoundElem":
		return map[string]any{"UUID": "asset-audio", "Url": "https://media.invalid/audio.mp3", "Size": 500, "Second": 1, "Download_Flag": 2}
	case "TIMTextElem":
		return map[string]any{"Text": "文字😊"}
	case "TIMCustomElem":
		return map[string]any{"Data": `{"businessID":"zhihuan_contact_card","userId":"synthetic"}`, "Desc": "名片", "Ext": ""}
	case "TIMFaceElem":
		return map[string]any{"Index": 0, "Data": "smile"}
	case "TIMLocationElem":
		return map[string]any{"Desc": "位置", "Latitude": 20.5, "Longitude": -90.5}
	}
	panic("unknown fixture")
}

func forwardFixtureElement(kind string, p map[string]any) MessageElement {
	raw, _ := json.Marshal(p)
	return MessageElement{MsgType: kind, MsgContent: raw}
}

func TestForwardValidationPreservesValidMixedMediaAndZeroValues(t *testing.T) {
	var body []MessageElement
	for _, kind := range []string{"TIMTextElem", "TIMImageElem", "TIMVideoFileElem", "TIMFileElem", "TIMSoundElem", "TIMFaceElem", "TIMLocationElem", "TIMCustomElem"} {
		body = append(body, forwardFixtureElement(kind, forwardMediaFixture(kind)))
	}
	before, _ := json.Marshal(body)
	prepared, err := PrepareForwardBody(body)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(prepared)
	if string(before) != string(after) {
		t.Fatal("valid payload or order was changed")
	}
	image := forwardMediaFixture("TIMImageElem")
	variant := image["ImageInfoArray"].([]any)[0].(map[string]any)
	variant["Size"], variant["Width"], variant["Height"] = 0, 0, 0
	if _, err := PrepareForwardBody([]MessageElement{forwardFixtureElement("TIMImageElem", image)}); err != nil {
		t.Fatal("documented zero dimensions rejected:", err)
	}
}

func TestForwardInvalidMediaNeverReachesTransport(t *testing.T) {
	for _, tc := range []struct {
		name, kind, field string
		mutate            func(map[string]any)
	}{
		{"image missing ID", "TIMImageElem", "UUID", func(p map[string]any) { delete(p, "UUID") }},
		{"image format string", "TIMImageElem", "ImageFormat", func(p map[string]any) { p["ImageFormat"] = "PNG" }},
		{"image unknown format", "TIMImageElem", "ImageFormat", func(p map[string]any) { p["ImageFormat"] = 0 }},
		{"image empty array", "TIMImageElem", "ImageInfoArray", func(p map[string]any) { p["ImageInfoArray"] = []any{} }},
		{"image duplicate type", "TIMImageElem", "Type", func(p map[string]any) { a := p["ImageInfoArray"].([]any); p["ImageInfoArray"] = append(a, a[0]) }},
		{"image missing width", "TIMImageElem", "Width", func(p map[string]any) { delete(p["ImageInfoArray"].([]any)[0].(map[string]any), "Width") }},
		{"image string height", "TIMImageElem", "Height", func(p map[string]any) { p["ImageInfoArray"].([]any)[0].(map[string]any)["Height"] = "10" }},
		{"image negative size", "TIMImageElem", "Size", func(p map[string]any) { p["ImageInfoArray"].([]any)[0].(map[string]any)["Size"] = -1 }},
		{"image malformed URL", "TIMImageElem", "URL", func(p map[string]any) {
			p["ImageInfoArray"].([]any)[0].(map[string]any)["URL"] = "https://?token=must-not-leak"
		}},
		{"image credentials URL", "TIMImageElem", "URL", func(p map[string]any) {
			p["ImageInfoArray"].([]any)[0].(map[string]any)["URL"] = "https://user:must-not-leak@media.invalid/a"
		}},
		{"video missing thumb ID", "TIMVideoFileElem", "ThumbUUID", func(p map[string]any) { delete(p, "ThumbUUID") }},
		{"video string length", "TIMVideoFileElem", "VideoSecond", func(p map[string]any) { p["VideoSecond"] = "0" }},
		{"video local thumbnail", "TIMVideoFileElem", "ThumbUrl", func(p map[string]any) { p["ThumbUrl"] = "blob:must-not-leak" }},
		{"video wrong flag", "TIMVideoFileElem", "ThumbDownloadFlag", func(p map[string]any) { p["ThumbDownloadFlag"] = 1 }},
		{"file missing ID", "TIMFileElem", "UUID", func(p map[string]any) { delete(p, "UUID") }},
		{"file missing name", "TIMFileElem", "FileName", func(p map[string]any) { p["FileName"] = " " }},
		{"file null size", "TIMFileElem", "FileSize", func(p map[string]any) { p["FileSize"] = nil }},
		{"file missing flag", "TIMFileElem", "Download_Flag", func(p map[string]any) { delete(p, "Download_Flag") }},
		{"sound missing duration", "TIMSoundElem", "Second", func(p map[string]any) { delete(p, "Second") }},
		{"face bad data", "TIMFaceElem", "Data", func(p map[string]any) { p["Data"] = map[string]any{} }},
		{"custom bad data", "TIMCustomElem", "Data", func(p map[string]any) { p["Data"] = map[string]any{} }},
		{"location bad latitude", "TIMLocationElem", "Latitude", func(p map[string]any) { p["Latitude"] = 91 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := forwardMediaFixture(tc.kind)
			tc.mutate(p)
			calls := 0
			client := forwardTestClient(func(*http.Request) (*http.Response, error) {
				calls++
				return forwardReply(`{"ActionStatus":"OK","ErrorCode":0}`), nil
			})
			_, err := client.SendForward(context.Background(), "alice", "GROUPreview", []MessageElement{forwardFixtureElement(tc.kind, p)}, 71, 2, nil)
			var detail *DeliveryError
			if calls != 0 || !errors.As(err, &detail) || detail.Unknown || detail.Code != 400 || detail.Command != "validate_forward_message" || !strings.Contains(detail.Message, tc.field) {
				t.Fatalf("invalid media was sent or diagnostic missing: %v calls=%d", err, calls)
			}
			if strings.Contains(detail.Message, "must-not-leak") || strings.Contains(detail.Message, "https:") {
				t.Fatal("validation leaked content or resource URL")
			}
		})
	}
}

func TestForwardInvalidBodyStructureAndNoUnsafeLegacyGuess(t *testing.T) {
	for _, body := range [][]MessageElement{nil, {{MsgType: "TIMImageElem", MsgContent: json.RawMessage(`null`)}}, {{MsgType: "TIMUnknownElem", MsgContent: json.RawMessage(`{}`)}}, {{MsgType: "TIMTextElem", MsgContent: json.RawMessage(`{"Text":""}`)}}} {
		if _, err := PrepareForwardBody(body); err == nil {
			t.Fatal("invalid body accepted")
		}
	}
	image := forwardMediaFixture("TIMImageElem")
	variant := image["ImageInfoArray"].([]any)[0].(map[string]any)
	variant["Type"] = 0
	delete(variant, "Width")
	if _, err := PrepareForwardBody([]MessageElement{forwardFixtureElement("TIMImageElem", image)}); err == nil || !strings.Contains(err.Error(), "Width") {
		t.Fatal("legacy conversion manufactured missing width")
	}
}
