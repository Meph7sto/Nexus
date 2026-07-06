package service

import "testing"

func TestUserRoleHelpers(t *testing.T) {
	cases := []struct {
		role     string
		admin    bool
		super    bool
		adminish bool
	}{
		{role: RoleUser, admin: false, super: false, adminish: false},
		{role: RoleAdmin, admin: true, super: false, adminish: true},
		{role: RoleSuperAdmin, admin: false, super: true, adminish: true},
	}

	for _, tc := range cases {
		u := &User{Role: tc.role}
		if got := u.IsAdmin(); got != tc.admin {
			t.Fatalf("IsAdmin(%q) = %v, want %v", tc.role, got, tc.admin)
		}
		if got := u.IsSuperAdmin(); got != tc.super {
			t.Fatalf("IsSuperAdmin(%q) = %v, want %v", tc.role, got, tc.super)
		}
		if got := u.IsAdminLike(); got != tc.adminish {
			t.Fatalf("IsAdminLike(%q) = %v, want %v", tc.role, got, tc.adminish)
		}
	}
}

func TestValidateAdminPermissions(t *testing.T) {
	valid := []AdminPermission{
		{Resource: AdminResourceUsers, Actions: []AdminPermissionAction{AdminActionView, AdminActionUpdate}},
		{Resource: AdminResourceUsage, Actions: []AdminPermissionAction{AdminActionView, AdminActionExport}},
	}
	if err := ValidateAdminPermissions(valid); err != nil {
		t.Fatalf("valid permissions rejected: %v", err)
	}

	invalidResource := []AdminPermission{{Resource: AdminPermissionResource("unknown"), Actions: []AdminPermissionAction{AdminActionView}}}
	if err := ValidateAdminPermissions(invalidResource); err == nil {
		t.Fatalf("unknown resource accepted")
	}

	superOnly := []AdminPermission{{Resource: AdminResourceSettings, Actions: []AdminPermissionAction{AdminActionView}}}
	if err := ValidateAdminPermissions(superOnly); err == nil {
		t.Fatalf("super-admin-only resource accepted for limited admin")
	}

	missingView := []AdminPermission{{Resource: AdminResourceUsers, Actions: []AdminPermissionAction{AdminActionUpdate}}}
	if err := ValidateAdminPermissions(missingView); err == nil {
		t.Fatalf("non-view action without view accepted")
	}
}
