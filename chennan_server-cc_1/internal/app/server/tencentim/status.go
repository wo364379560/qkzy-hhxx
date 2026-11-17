package tencentim

import "fmt"

type queryStateReq struct {
	ToAccount []string `json:"To_Account"`
}

type queryStateResp struct {
	ActionStatus string `json:"ActionStatus"`
	ErrorCode    int    `json:"ErrorCode"`
	ErrorInfo    string `json:"ErrorInfo"`
	QueryResult  []struct {
		ToAccount string `json:"To_Account"`
		Status    string `json:"Status"`
		Detail    []struct {
			Platform string `json:"Platform"`
			Status   string `json:"Status"`
		} `json:"Detail"`
	} `json:"QueryResult"`
	ErrorList []struct {
		ToAccount string `json:"To_Account"`
		ErrorCode int    `json:"ErrorCode"`
	} `json:"ErrorList"`
}

// QueryUserStatus 查询单个用户在线状态。
func QueryUserStatus(imUserID string) (status string, platform string, err error) {
	client := NewClient()
	if !client.IsConfigured() {
		return "", "", fmt.Errorf("腾讯云 IM 未配置")
	}

	result, err := queryUserStates(client, []string{imUserID})
	if err != nil {
		return "", "", err
	}
	if len(result.QueryResult) == 0 {
		return "offline", "", nil
	}

	userResult := result.QueryResult[0]
	if userResult.Status == "Offline" {
		return "offline", "", nil
	}

	if len(userResult.Detail) > 0 {
		platform = userResult.Detail[0].Platform
	}

	return "online", platform, nil
}

// BatchQueryUserStatus 批量查询用户在线状态。
func BatchQueryUserStatus(imUserIDs []string) ([]map[string]interface{}, error) {
	if len(imUserIDs) == 0 {
		return []map[string]interface{}{}, nil
	}

	client := NewClient()
	if !client.IsConfigured() {
		return nil, fmt.Errorf("腾讯云 IM 未配置")
	}

	result, err := queryUserStates(client, imUserIDs)
	if err != nil {
		return nil, err
	}

	statusMap := make(map[string]map[string]interface{}, len(imUserIDs))
	for _, imUserID := range imUserIDs {
		statusMap[imUserID] = map[string]interface{}{
			"im_user_id": imUserID,
			"status":     "offline",
			"platform":   "",
		}
	}

	for _, item := range result.QueryResult {
		entry, ok := statusMap[item.ToAccount]
		if !ok {
			entry = map[string]interface{}{
				"im_user_id": item.ToAccount,
				"status":     "offline",
				"platform":   "",
			}
		}
		if item.Status != "Offline" {
			entry["status"] = "online"
			if len(item.Detail) > 0 {
				entry["platform"] = item.Detail[0].Platform
			}
		}
		statusMap[item.ToAccount] = entry
	}

	results := make([]map[string]interface{}, 0, len(imUserIDs))
	for _, imUserID := range imUserIDs {
		results = append(results, statusMap[imUserID])
	}

	return results, nil
}

func queryUserStates(client *Client, imUserIDs []string) (*queryStateResp, error) {
	var result queryStateResp
	if err := client.post(
		client.buildURL("openim", "querystate"),
		queryStateReq{ToAccount: imUserIDs},
		&result,
	); err != nil {
		return nil, err
	}

	if result.ErrorCode != 0 {
		return nil, fmt.Errorf("腾讯云 IM API 错误: %s", result.ErrorInfo)
	}

	return &result, nil
}
