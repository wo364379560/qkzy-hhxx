package controller_chat

import (
	"testing"

	"github.com/ixayda/ix-Go/internal/app/server/model"
)

func isTestSuperAdmin(id uint) bool {
	return id == 99
}

func TestCanSearchAndAddFriend(t *testing.T) {
	t.Parallel()

	poolZero := &model.User{OwnerID: 0}
	poolSuper := &model.User{OwnerID: 99}
	assignedHidden := &model.User{OwnerID: 13, InviteCode: "A"}
	assignedSearchable := &model.User{OwnerID: 13, InviteCode: "A", GloballySearchable: true}
	assignedOther := &model.User{OwnerID: 2, InviteCode: "B", GloballySearchable: true}

	cases := []struct {
		name string
		from *model.User
		to   *model.User
		want bool
	}{
		{"same assigned", assignedHidden, &model.User{OwnerID: 13}, true},
		{"two assigned different even searchable", assignedSearchable, assignedOther, false},
		{"pool super finds hidden assigned", poolSuper, assignedHidden, false},
		{"pool super finds searchable assigned", poolSuper, assignedSearchable, true},
		{"pool zero finds hidden assigned", poolZero, assignedHidden, false},
		{"pool zero finds searchable assigned", poolZero, assignedSearchable, true},
		{"assigned finds pool zero", assignedHidden, poolZero, true},
		{"assigned finds pool super", assignedHidden, poolSuper, true},
		{"two pool", poolZero, poolSuper, true},
		{"same super pool", poolSuper, &model.User{OwnerID: 99}, true},
		{"nil from", nil, assignedSearchable, false},
		{"nil to", assignedSearchable, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := canSearchUserWith(tc.from, tc.to, isTestSuperAdmin); got != tc.want {
				t.Fatalf("canSearchUserWith()=%v want=%v", got, tc.want)
			}
			if got := canCrossOwnerFriendAddWith(tc.from, tc.to, isTestSuperAdmin); got != tc.want {
				t.Fatalf("canCrossOwnerFriendAddWith()=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestPickFriendAddMigrate(t *testing.T) {
	t.Parallel()

	unassigned := &model.User{ID: 1, OwnerID: 0, IMUserID: "u_pool"}
	superPool := &model.User{ID: 4, OwnerID: 99, IMUserID: "u_super"}
	assignedA := &model.User{ID: 2, OwnerID: 13, IMUserID: "u_a", GloballySearchable: true}
	assignedB := &model.User{ID: 3, OwnerID: 2, IMUserID: "u_b"}

	t.Run("unassigned joins assigned", func(t *testing.T) {
		t.Parallel()
		move, owner, ok := pickFriendAddMigrateWith(unassigned, assignedA, isTestSuperAdmin)
		if !ok || move != unassigned || owner != 13 {
			t.Fatalf("got move=%v owner=%d ok=%v", move, owner, ok)
		}
	})
	t.Run("assigned pulls unassigned", func(t *testing.T) {
		t.Parallel()
		move, owner, ok := pickFriendAddMigrateWith(assignedA, unassigned, isTestSuperAdmin)
		if !ok || move != unassigned || owner != 13 {
			t.Fatalf("got move=%v owner=%d ok=%v", move, owner, ok)
		}
	})
	t.Run("super pool joins assigned", func(t *testing.T) {
		t.Parallel()
		move, owner, ok := pickFriendAddMigrateWith(superPool, assignedA, isTestSuperAdmin)
		if !ok || move != superPool || owner != 13 {
			t.Fatalf("got move=%v owner=%d ok=%v", move, owner, ok)
		}
	})
	t.Run("assigned pulls super pool", func(t *testing.T) {
		t.Parallel()
		move, owner, ok := pickFriendAddMigrateWith(assignedA, superPool, isTestSuperAdmin)
		if !ok || move != superPool || owner != 13 {
			t.Fatalf("got move=%v owner=%d ok=%v", move, owner, ok)
		}
	})
	t.Run("same assigned no migrate", func(t *testing.T) {
		t.Parallel()
		if _, _, ok := pickFriendAddMigrateWith(assignedA, &model.User{OwnerID: 13}, isTestSuperAdmin); ok {
			t.Fatal("expected no migrate")
		}
	})
	t.Run("different assigned no migrate", func(t *testing.T) {
		t.Parallel()
		if _, _, ok := pickFriendAddMigrateWith(assignedA, assignedB, isTestSuperAdmin); ok {
			t.Fatal("expected no migrate")
		}
	})
	t.Run("both pool no migrate", func(t *testing.T) {
		t.Parallel()
		if _, _, ok := pickFriendAddMigrateWith(unassigned, superPool, isTestSuperAdmin); ok {
			t.Fatal("expected no migrate")
		}
	})
}

func TestShouldMigrateOwnerOnPrevFriendAdd(t *testing.T) {
	t.Parallel()

	if ShouldMigrateOwnerOnPrevFriendAdd(nil) {
		t.Fatal("nil should wait")
	}
	if ShouldMigrateOwnerOnPrevFriendAdd(&model.User{AllowType: 0}) {
		t.Fatal("AllowAny still must wait for confirmed bilateral friendship")
	}
	if ShouldMigrateOwnerOnPrevFriendAdd(&model.User{AllowType: 1}) {
		t.Fatal("NeedConfirm should wait for agree")
	}
	if ShouldMigrateOwnerOnPrevFriendAdd(&model.User{AllowType: 2}) {
		t.Fatal("DenyAny should not migrate on prev-add")
	}
}

func TestPickMigratedInviteCode(t *testing.T) {
	t.Parallel()

	stay := &model.User{OwnerID: 13, InviteCode: "aaabbb"}
	if got := pickMigratedInviteCode("YZRGXYZS", stay); got != "YZRGXYZS" {
		t.Fatalf("prefer owner active code, got %q", got)
	}
	if got := pickMigratedInviteCode("", stay); got != "aaabbb" {
		t.Fatalf("fallback stay invite code got %q want aaabbb", got)
	}
	if got := pickMigratedInviteCode("", nil); got != "" {
		t.Fatalf("empty got %q", got)
	}
}

func TestPickFriendAddMigrateUnassignedToAssigned(t *testing.T) {
	t.Parallel()

	unassignedUser := &model.User{ID: 100, OwnerID: 0, IMUserID: "a_18822224444", Phone: "18822224444"}
	lanxuzhuliUser := &model.User{ID: 101, OwnerID: 13, IMUserID: "a_14422224444", Phone: "14422224444", GloballySearchable: true}

	t.Run("unassigned user adds assistant user", func(t *testing.T) {
		t.Parallel()
		move, owner, ok := pickFriendAddMigrateWith(unassignedUser, lanxuzhuliUser, isTestSuperAdmin)
		if !ok {
			t.Fatal("expected migrate")
		}
		if move != unassignedUser {
			t.Fatalf("should move unassigned user, got %+v", move)
		}
		if owner != 13 {
			t.Fatalf("owner=%d want 13 (lanxuzhuli)", owner)
		}
	})

	t.Run("assistant user accepts then pulls unassigned user", func(t *testing.T) {
		t.Parallel()
		move, owner, ok := pickFriendAddMigrateWith(lanxuzhuliUser, unassignedUser, isTestSuperAdmin)
		if !ok || move != unassignedUser || owner != 13 {
			t.Fatalf("got move=%v owner=%d ok=%v", move, owner, ok)
		}
	})
}
