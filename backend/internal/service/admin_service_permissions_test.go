package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/pkg/pagination"
)

type adminServiceUserRepoStub struct {
	user *User
}

func (s adminServiceUserRepoStub) Create(context.Context, *User) error { return nil }
func (s adminServiceUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	cloned := *s.user
	return &cloned, nil
}
func (s adminServiceUserRepoStub) GetByIDIncludeDeleted(context.Context, int64) (*User, error) {
	cloned := *s.user
	return &cloned, nil
}
func (s adminServiceUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	cloned := *s.user
	return &cloned, nil
}
func (s adminServiceUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	cloned := *s.user
	return &cloned, nil
}
func (s adminServiceUserRepoStub) Update(context.Context, *User) error { return nil }
func (s adminServiceUserRepoStub) Delete(context.Context, int64) error { return nil }
func (s adminServiceUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	return nil, nil
}
func (s adminServiceUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	return nil, nil
}
func (s adminServiceUserRepoStub) DeleteUserAvatar(context.Context, int64) error { return nil }
func (s adminServiceUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s adminServiceUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s adminServiceUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	return map[int64]*time.Time{}, nil
}
func (s adminServiceUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	return nil, nil
}
func (s adminServiceUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	return nil
}
func (s adminServiceUserRepoStub) UpdateBalance(context.Context, int64, float64) error { return nil }
func (s adminServiceUserRepoStub) DeductBalance(context.Context, int64, float64) error { return nil }
func (s adminServiceUserRepoStub) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (s adminServiceUserRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (s adminServiceUserRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (s adminServiceUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}
func (s adminServiceUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s adminServiceUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s adminServiceUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s adminServiceUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	return nil, nil
}
func (s adminServiceUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	return nil
}
func (s adminServiceUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error {
	return nil
}
func (s adminServiceUserRepoStub) EnableTotp(context.Context, int64) error  { return nil }
func (s adminServiceUserRepoStub) DisableTotp(context.Context, int64) error { return nil }

type adminServicePermissionRepoStub struct {
	permissions []AdminPermission
}

func (s adminServicePermissionRepoStub) ListByUserID(context.Context, int64) ([]AdminPermission, error) {
	out := make([]AdminPermission, len(s.permissions))
	copy(out, s.permissions)
	return out, nil
}

func (s adminServicePermissionRepoStub) ReplaceForUser(context.Context, int64, []AdminPermission) error {
	return nil
}

func (s adminServicePermissionRepoStub) DeleteForUser(context.Context, int64) error {
	return nil
}

func (s adminServicePermissionRepoStub) HasPermission(_ context.Context, _ int64, resource AdminPermissionResource, action AdminPermissionAction) (bool, error) {
	for _, perm := range s.permissions {
		if perm.Resource != resource {
			continue
		}
		for _, existing := range perm.Actions {
			if existing == action {
				return true, nil
			}
		}
	}
	return false, nil
}

func TestAdminServiceGetUserLoadsAdminPermissions(t *testing.T) {
	svc := &adminServiceImpl{
		userRepo: adminServiceUserRepoStub{
			user: &User{
				ID:     7,
				Email:  "limited-admin@example.com",
				Role:   RoleAdmin,
				Status: StatusActive,
			},
		},
		adminPermissionRepo: adminServicePermissionRepoStub{
			permissions: []AdminPermission{
				{
					Resource: AdminResourceUsers,
					Actions:  []AdminPermissionAction{AdminActionView, AdminActionUpdate},
				},
			},
		},
	}

	user, err := svc.GetUser(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if len(user.AdminPermissions) != 1 {
		t.Fatalf("AdminPermissions length = %d, want 1: %#v", len(user.AdminPermissions), user.AdminPermissions)
	}
	if user.AdminPermissions[0].Resource != AdminResourceUsers {
		t.Fatalf("resource = %s, want %s", user.AdminPermissions[0].Resource, AdminResourceUsers)
	}
	if !containsAdminAction(user.AdminPermissions[0].Actions, AdminActionUpdate) {
		t.Fatalf("actions = %#v, want update", user.AdminPermissions[0].Actions)
	}
}

func containsAdminAction(actions []AdminPermissionAction, want AdminPermissionAction) bool {
	for _, action := range actions {
		if action == want {
			return true
		}
	}
	return false
}
