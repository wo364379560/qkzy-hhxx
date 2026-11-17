package tencentim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"unicode"
)

// PrepareForwardBody is shared by admission and dispatch. It validates a copy
// of identifiable legacy image types without rewriting the persisted input,
// its fingerprint, or the Random/Sequence used for retry deduplication.
func PrepareForwardBody(body []MessageElement) ([]MessageElement, error) {
	normalized, err := normalizeForwardImages(body)
	if err == nil {
		err = validateForwardBody(normalized)
	}
	if err != nil {
		// This is a local, definite rejection before HTTP, not Tencent 10004.
		return nil, (&Client{}).forwardError("validate_forward_message", 400, false, err.Error(), "", "")
	}
	return normalized, nil
}

func validateForwardBody(body []MessageElement) error {
	if len(body) == 0 {
		return fmt.Errorf("消息内容不能为空")
	}
	encoded, err := json.Marshal(body)
	if err != nil || len(encoded) > 10000 {
		return fmt.Errorf("消息内容不是有效JSON或超过10KB")
	}
	for index, element := range body {
		var p map[string]any
		decoder := json.NewDecoder(bytes.NewReader(element.MsgContent))
		decoder.UseNumber()
		if decoder.Decode(&p) != nil || p == nil {
			return fmt.Errorf("第%d个消息的MsgContent必须是对象", index+1)
		}
		if err := validateForwardElement(element.MsgType, p); err != nil {
			return fmt.Errorf("第%d个消息：%w", index+1, err)
		}
	}
	return nil
}

func validateForwardElement(kind string, p map[string]any) error {
	switch kind {
	case "TIMTextElem":
		return requiredForwardString(p, "Text", true)
	case "TIMImageElem":
		if err := requiredForwardID(p, "UUID"); err != nil {
			return err
		}
		format, ok := forwardInteger(p["ImageFormat"])
		if !ok || (format != 1 && format != 2 && format != 3 && format != 4 && format != 255) {
			return fmt.Errorf("ImageFormat必须为1、2、3、4或255")
		}
		images, ok := p["ImageInfoArray"].([]any)
		if !ok || len(images) == 0 || len(images) > 3 {
			return fmt.Errorf("ImageInfoArray必须包含1至3种图片规格")
		}
		seen := map[int64]bool{}
		for i, value := range images {
			image, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("ImageInfoArray[%d]必须是对象", i)
			}
			variant, ok := forwardInteger(image["Type"])
			if !ok || variant < 1 || variant > 3 || seen[variant] {
				return fmt.Errorf("ImageInfoArray[%d].Type必须为不重复的1、2、3", i)
			}
			seen[variant] = true
			if err := forwardNumbers(image, "Size", "Width", "Height"); err != nil {
				return fmt.Errorf("ImageInfoArray[%d].%w", i, err)
			}
			if err := requiredForwardURL(image, "URL"); err != nil {
				return fmt.Errorf("ImageInfoArray[%d].%w", i, err)
			}
		}
	case "TIMFileElem":
		if err := requiredForwardID(p, "UUID"); err != nil {
			return err
		}
		if err := requiredForwardURL(p, "Url"); err != nil {
			return err
		}
		if err := requiredForwardString(p, "FileName", true); err != nil {
			return err
		}
		if err := forwardNumbers(p, "FileSize"); err != nil {
			return err
		}
		return forwardDownloadFlags(p, "Download_Flag")
	case "TIMSoundElem":
		if err := requiredForwardID(p, "UUID"); err != nil {
			return err
		}
		if err := requiredForwardURL(p, "Url"); err != nil {
			return err
		}
		if err := forwardNumbers(p, "Size", "Second"); err != nil {
			return err
		}
		return forwardDownloadFlags(p, "Download_Flag")
	case "TIMVideoFileElem":
		for _, key := range []string{"VideoUUID", "ThumbUUID"} {
			if err := requiredForwardID(p, key); err != nil {
				return err
			}
		}
		for _, key := range []string{"VideoUrl", "ThumbUrl"} {
			if err := requiredForwardURL(p, key); err != nil {
				return err
			}
		}
		if err := forwardNumbers(p, "VideoSize", "VideoSecond", "ThumbSize", "ThumbWidth", "ThumbHeight"); err != nil {
			return err
		}
		if err := forwardOptionalStrings(p, "VideoFormat", "ThumbFormat"); err != nil {
			return err
		}
		return forwardDownloadFlags(p, "VideoDownloadFlag", "ThumbDownloadFlag")
	case "TIMCustomElem":
		if err := requiredForwardString(p, "Data", false); err != nil {
			return err
		}
		return forwardOptionalStrings(p, "Desc", "Ext")
	case "TIMFaceElem":
		if value, exists := p["Index"]; exists {
			if _, ok := forwardInteger(value); !ok {
				return fmt.Errorf("Index必须是非负整数")
			}
		}
		if err := forwardOptionalStrings(p, "Data"); err != nil {
			return err
		}
		if _, exists := p["Index"]; !exists && p["Data"] != nil && p["Data"] != "" {
			return nil
		}
		if _, exists := p["Index"]; !exists {
			return fmt.Errorf("表情必须包含Index或非空Data")
		}
	case "TIMLocationElem":
		if err := requiredForwardString(p, "Desc", false); err != nil {
			return err
		}
		for key, bound := range map[string]float64{"Latitude": 90, "Longitude": 180} {
			value, ok := p[key].(json.Number)
			number, err := value.Float64()
			if !ok || err != nil || math.IsNaN(number) || math.IsInf(number, 0) || math.Abs(number) > bound {
				return fmt.Errorf("%s必须是有效范围内的数字", key)
			}
		}
	default:
		return fmt.Errorf("不支持此消息类型")
	}
	return nil
}

func forwardInteger(value any) (int64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	n, err := number.Int64()
	return n, err == nil && n >= 0 && n <= 9007199254740991
}

func forwardNumbers(p map[string]any, keys ...string) error {
	for _, key := range keys {
		if _, ok := forwardInteger(p[key]); !ok {
			return fmt.Errorf("%s必须是非负整数，不能缺失或使用字符串", key)
		}
	}
	return nil
}

func requiredForwardString(p map[string]any, key string, nonempty bool) error {
	s, ok := p[key].(string)
	if !ok || nonempty && strings.TrimSpace(s) == "" {
		if nonempty {
			return fmt.Errorf("%s必须是非空字符串", key)
		}
		return fmt.Errorf("%s必须是字符串", key)
	}
	return nil
}

func requiredForwardID(p map[string]any, key string) error {
	if err := requiredForwardString(p, key, true); err != nil {
		return err
	}
	if strings.IndexFunc(p[key].(string), unicode.IsControl) >= 0 {
		return fmt.Errorf("%s包含无效控制字符", key)
	}
	return nil
}

func forwardOptionalStrings(p map[string]any, keys ...string) error {
	for _, key := range keys {
		if _, exists := p[key]; exists {
			if err := requiredForwardString(p, key, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// Validate URL syntax only; never fetch a user-supplied media URL on admission.
// Diagnostics contain field names, never signed URLs or user content.
func requiredForwardURL(p map[string]any, key string) error {
	s, ok := p[key].(string)
	u, err := url.Parse(s)
	if !ok || err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return fmt.Errorf("%s必须是有效的HTTP/HTTPS资源地址", key)
	}
	return nil
}

func forwardDownloadFlags(p map[string]any, keys ...string) error {
	for _, key := range keys {
		if value, ok := forwardInteger(p[key]); !ok || value != 2 {
			return fmt.Errorf("%s必须为数字2", key)
		}
	}
	return nil
}
