//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/config"
	middleware2 "github.com/Wei-Shaw/nexus/internal/server/middleware"
	"github.com/Wei-Shaw/nexus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminPermissionRepoStub struct {
	permissions []service.AdminPermission
}

func (s adminPermissionRepoStub) ListByUserID(context.Context, int64) ([]service.AdminPermission, error) {
	out := make([]service.AdminPermission, len(s.permissions))
	copy(out, s.permissions)
	return out, nil
}

func (s adminPermissionRepoStub) ReplaceForUser(context.Context, int64, []service.AdminPermission) error {
	return nil
}

func (s adminPermissionRepoStub) DeleteForUser(context.Context, int64) error {
	return nil
}

func (s adminPermissionRepoStub) HasPermission(_ context.Context, _ int64, resource service.AdminPermissionResource, action service.AdminPermissionAction) (bool, error) {
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

func TestAuthHandlerGetCurrentUserReturnsProfileCompatibilityFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	verifiedAt := time.Date(2026, 4, 20, 8, 30, 0, 0, time.UTC)
	repo := &userHandlerRepoStub{
		user: &service.User{
			ID:           31,
			Email:        "me@example.com",
			Username:     "linuxdo-handle",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
			AvatarURL:    "https://cdn.example.com/linuxdo.png",
			AvatarSource: "remote_url",
		},
		identities: []service.UserAuthIdentityRecord{
			{
				ProviderType:    "linuxdo",
				ProviderKey:     "linuxdo",
				ProviderSubject: "linuxdo-subject-31",
				VerifiedAt:      &verifiedAt,
				Metadata: map[string]any{
					"username":   "linuxdo-handle",
					"avatar_url": "https://cdn.example.com/linuxdo.png",
				},
			},
		},
	}

	handler := &AuthHandler{
		userService: service.NewUserService(repo, nil, nil, nil),
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 31})

	handler.GetCurrentUser(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["email_bound"])
	require.Equal(t, true, resp.Data["linuxdo_bound"])
	require.Equal(t, "https://cdn.example.com/linuxdo.png", resp.Data["avatar_url"])

	authBindings, ok := resp.Data["auth_bindings"].(map[string]any)
	require.True(t, ok)
	linuxdoBinding, ok := authBindings["linuxdo"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, linuxdoBinding["bound"])

	avatarSource, ok := resp.Data["avatar_source"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "linuxdo", avatarSource["provider"])
	require.Equal(t, "linuxdo", avatarSource["source"])

	profileSources, ok := resp.Data["profile_sources"].(map[string]any)
	require.True(t, ok)
	usernameSource, ok := profileSources["username"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "linuxdo", usernameSource["provider"])
	require.Equal(t, "linuxdo", usernameSource["source"])
}

func TestAuthHandlerGetCurrentUserReturnsAdminPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &userHandlerRepoStub{
		user: &service.User{
			ID:       41,
			Email:    "limited-admin@example.com",
			Username: "limited-admin",
			Role:     service.RoleAdmin,
			Status:   service.StatusActive,
		},
	}
	permissionRepo := adminPermissionRepoStub{
		permissions: []service.AdminPermission{
			{
				Resource: service.AdminResourceUsers,
				Actions: []service.AdminPermissionAction{
					service.AdminActionView,
					service.AdminActionUpdate,
				},
			},
		},
	}

	handler := &AuthHandler{
		userService: service.NewUserService(repo, nil, nil, nil, permissionRepo),
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 41})

	handler.GetCurrentUser(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AdminPermissions []service.AdminPermission `json:"admin_permissions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.AdminPermissions, 1)
	require.Equal(t, service.AdminResourceUsers, resp.Data.AdminPermissions[0].Resource)
	require.ElementsMatch(t,
		[]service.AdminPermissionAction{service.AdminActionView, service.AdminActionUpdate},
		resp.Data.AdminPermissions[0].Actions,
	)
}

func TestAuthHandlerLoginReturnsAdminPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.JWT.Secret = "unit-test-secret"
	cfg.JWT.ExpireHour = 1
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	passwordHash, err := authService.HashPassword("123456")
	require.NoError(t, err)

	repo := &userHandlerRepoStub{
		user: &service.User{
			ID:           42,
			Email:        "limited-admin" + service.LinuxDoConnectSyntheticEmailDomain,
			Username:     "limited-admin",
			PasswordHash: passwordHash,
			Role:         service.RoleAdmin,
			Status:       service.StatusActive,
		},
	}
	authService = service.NewAuthService(nil, repo, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	permissionRepo := adminPermissionRepoStub{
		permissions: []service.AdminPermission{
			{
				Resource: service.AdminResourceGroups,
				Actions: []service.AdminPermissionAction{
					service.AdminActionView,
					service.AdminActionCreate,
				},
			},
		},
	}
	handler := &AuthHandler{
		authService: authService,
		userService: service.NewUserService(repo, nil, nil, nil, permissionRepo),
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"limited-admin`+service.LinuxDoConnectSyntheticEmailDomain+`","password":"123456"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			User struct {
				AdminPermissions []service.AdminPermission `json:"admin_permissions"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.User.AdminPermissions, 1)
	require.Equal(t, service.AdminResourceGroups, resp.Data.User.AdminPermissions[0].Resource)
	require.ElementsMatch(t,
		[]service.AdminPermissionAction{service.AdminActionView, service.AdminActionCreate},
		resp.Data.User.AdminPermissions[0].Actions,
	)
}
