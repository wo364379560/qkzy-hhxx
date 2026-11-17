package controller_chat

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/model"
)

func TestProfileAssignment(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ownerID  uint
		username string
		role     string
		assigned bool
	}{
		{name: "no owner"},
		{name: "feng88 super admin", ownerID: 8, username: "feng88", role: "super_admin"},
		{name: "other super admin", ownerID: 21, username: "other", role: "super_admin"},
		{name: "admin", ownerID: 31, role: "admin", assigned: true},
		{name: "assistant", ownerID: 41, role: "assistant", assigned: true},
		{name: "role determines assignment, not username", ownerID: 8, username: "feng88", role: "admin", assigned: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			// 有邀请码不代表已归属运营方，仍须检查归属人的角色。
			user := &model.User{OwnerID: tt.ownerID, Status: "active", InviteCode: "example"}
			result, err := toProfileResultWithOwnerLookup(user, func(id uint) (*model.AdminUser, error) {
				calls++
				if id != tt.ownerID {
					t.Fatalf("looked up owner %d, want %d", id, tt.ownerID)
				}
				return &model.AdminUser{ID: id, Username: tt.username, Role: tt.role, Status: "active", DisplayName: " Owner "}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if user.OwnerID != tt.ownerID {
				t.Fatal("profile must not modify the stored owner ID")
			}
			wantOwnerID := tt.ownerID
			wantOwnerName := "Owner"
			if tt.ownerID == 0 || tt.role == "super_admin" {
				wantOwnerID = 0
				wantOwnerName = ""
			}
			if result.OwnerID != wantOwnerID || result.OwnerName != wantOwnerName {
				t.Fatalf("profile owner = (%d, %q), want (%d, %q)", result.OwnerID, result.OwnerName, wantOwnerID, wantOwnerName)
			}
			if tt.ownerID == 0 {
				if calls != 0 || result.OwnerName != "" {
					t.Fatal("unowned user must not query an owner")
				}
			} else if calls != 1 {
				t.Fatal("owner name and assignment must use the same lookup")
			}
			payload, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal(payload, &body); err != nil {
				t.Fatal(err)
			}
			if got, ok := body["is_assigned"].(bool); !ok || got != tt.assigned {
				t.Fatalf("is_assigned = %v, want explicit %v", body["is_assigned"], tt.assigned)
			}
			if got, ok := body["owner_id"].(float64); !ok || got != float64(wantOwnerID) {
				t.Fatalf("owner_id = %v, want explicit %d", body["owner_id"], wantOwnerID)
			}
			if !tt.assigned {
				if _, ok := body["owner_name"]; ok {
					t.Fatal("unassigned profile must not expose an owner name")
				}
			}
		})
	}
}

func TestProfileAssignmentLookupError(t *testing.T) {
	want := errors.New("owner lookup unavailable")
	result, err := toProfileResultWithOwnerLookup(&model.User{OwnerID: 8}, func(uint) (*model.AdminUser, error) {
		return nil, want
	})
	if !errors.Is(err, want) || result != nil {
		t.Fatal("lookup failure must not return is_assigned=false as a successful profile")
	}
}
