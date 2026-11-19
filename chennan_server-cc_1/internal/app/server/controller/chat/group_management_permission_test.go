package controller_chat

import "testing"

func TestNativeGroupRolesDoNotRequireLocalAdminMirror(t *testing.T) {
	// No datasource is configured. These roles must be resolved from the
	// verified Tencent response without requiring a local mirror row.
	for _, role := range []string{groupRoleOwner, groupRoleAdmin} {
		permission, err := groupManagementPermissionForRole("group", "user", role)
		if err != nil || permission == nil {
			t.Fatalf("verified native %s rejected: %v", role, err)
		}
		if permission.IsOwner != (role == groupRoleOwner) || permission.IsTencentAdmin != (role == groupRoleAdmin) {
			t.Fatalf("incorrect native privilege: %+v", permission)
		}
	}
	if permission, err := groupManagementPermissionForRole("group", "user", "NotMember"); permission != nil || err != nil {
		t.Fatal("departed user regained a native management role")
	}
	for _, role := range []string{"", "Unknown", "admin"} {
		if permission, err := groupManagementPermissionForRole("group", "user", role); permission != nil || err == nil {
			t.Fatalf("unconfirmed role %q was accepted", role)
		}
	}
}
