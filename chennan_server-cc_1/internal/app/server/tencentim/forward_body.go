package tencentim

import (
	"encoding/json"
	"errors"
	"strconv"
)

// Older Web jobs persisted SDK image sizes (0 original, 1 thumbnail, 2 large)
// in a REST payload. Only the presence of an actual numeric zero unambiguously
// identifies that format. Never reinterpret an already-valid REST 1/2/3 array.
// The returned copy is per-dispatch; durable body and message identity stay put.
func normalizeForwardImages(body []MessageElement) ([]MessageElement, error) {
	result := body
	copied := false
	for i, message := range body {
		if message.MsgType != "TIMImageElem" {
			continue
		}
		var content map[string]json.RawMessage
		if json.Unmarshal(message.MsgContent, &content) != nil {
			continue // Preserve ordinary request validation for unrelated shapes.
		}
		var images []map[string]json.RawMessage
		if json.Unmarshal(content["ImageInfoArray"], &images) != nil || len(images) == 0 {
			continue
		}
		types := make([]int, len(images))
		hasSDKZero, validSDKTypes := false, true
		for j, image := range images {
			raw, exists := image["Type"]
			if !exists || string(raw) == "null" || json.Unmarshal(raw, &types[j]) != nil {
				validSDKTypes = false
				continue
			}
			if types[j] == 0 {
				hasSDKZero = true
			}
			if types[j] < 0 || types[j] > 2 {
				validSDKTypes = false
			}
		}
		if !hasSDKZero {
			continue
		}
		if !validSDKTypes {
			return nil, errors.New("旧图片尺寸类型混合，无法安全转换")
		}
		for j, image := range images {
			image["Type"] = json.RawMessage(strconv.Itoa([3]int{1, 3, 2}[types[j]]))
		}
		encoded, err := json.Marshal(images)
		if err != nil {
			return nil, errors.New("旧图片资料无法转换")
		}
		content["ImageInfoArray"] = encoded
		encoded, err = json.Marshal(content)
		if err != nil {
			return nil, errors.New("旧图片资料无法转换")
		}
		if !copied {
			result = append([]MessageElement(nil), body...)
			copied = true
		}
		result[i].MsgContent = encoded
	}
	return result, nil
}
