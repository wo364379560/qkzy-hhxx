package tencentim

import (
	"context"
	"fmt"
	"time"
)

const friendGetPageLimit = 100

type friendGetReq struct {
	FromAccount string `json:"From_Account"`
	StartIndex  int    `json:"StartIndex"`
}

type friendGetItem struct {
	ToAccount string `json:"To_Account"`
}

type friendGetResp struct {
	accountBaseResp
	UserDataItem   []friendGetItem `json:"UserDataItem"`
	CompleteFlag   int             `json:"CompleteFlag"`
	NextStartIndex int             `json:"NextStartIndex"`
	FriendNum      int             `json:"FriendNum"`
}

// ListFriendIDs 拉取某账号的好友 UserID 列表。
func (c *Client) ListFriendIDs(userID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	return c.ListFriendIDsContext(ctx, userID)
}

func (c *Client) ListFriendIDsContext(ctx context.Context, userID string) ([]string, error) {
	if c == nil || !c.IsConfigured() || userID == "" {
		return nil, nil
	}
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	start := 0
	for i := 0; i < friendGetPageLimit; i++ {
		var resp friendGetResp
		if err := c.postContext(ctx, "sns", "friend_get", friendGetReq{
			FromAccount: userID,
			StartIndex:  start,
		}, &resp); err != nil {
			return ids, err
		}
		if err := resp.err(); err != nil {
			return ids, err
		}
		for _, item := range resp.UserDataItem {
			id := item.ToAccount
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if resp.CompleteFlag == 1 || resp.NextStartIndex <= start || len(resp.UserDataItem) == 0 {
			return ids, nil
		}
		start = resp.NextStartIndex
	}
	return ids, fmt.Errorf("friend_get exceeded page limit for %s", userID)
}

func (c *Client) CheckBilateral(ctx context.Context, from string, targets []string) (map[string]bool, error) {
	if c == nil || !c.IsConfigured() {
		return nil, fmt.Errorf("tencent friendship proof unavailable")
	}
	confirmed := map[string]bool{}
	for start := 0; start < len(targets); start += 100 {
		batch := targets[start:min(start+100, len(targets))]
		var resp struct {
			accountBaseResp
			InfoItem []struct {
				To       string `json:"To_Account"`
				Relation string `json:"Relation"`
				Code     int    `json:"ResultCode"`
			} `json:"InfoItem"`
		}
		if err := c.postContext(ctx, "sns", "friend_check", map[string]any{"From_Account": from, "To_Account": batch, "CheckType": "CheckResult_Type_Both"}, &resp); err != nil {
			return nil, err
		}
		if err := resp.err(); err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, item := range resp.InfoItem {
			if item.Code != 0 {
				return nil, fmt.Errorf("friend_check result %d", item.Code)
			}
			seen[item.To] = true
			confirmed[item.To] = item.Relation == "CheckResult_Type_BothWay"
		}
		for _, target := range batch {
			if !seen[target] {
				return nil, fmt.Errorf("incomplete friendship proof")
			}
		}
	}
	return confirmed, nil
}
