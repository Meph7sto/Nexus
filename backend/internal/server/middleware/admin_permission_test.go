package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type permissionRepoStub struct {
	allowed bool
	err     error
}

func (s permissionRepoStub) ListByUserID(context.Context, int64) ([]service.AdminPermission, error) {
	return nil, nil
}

func (s permissionRepoStub) ReplaceForUser(context.Context, int64, []service.AdminPermission) error {
	return nil
}

func (s permissionRepoStub) DeleteForUser(context.Context, int64) error {
	return nil
}

func (s permissionRepoStub) HasPermission(context.Context, int64, service.AdminPermissionResource, service.AdminPermissionAction) (bool, error) {
	return s.allowed, s.err
}

func TestAdminPermissionMiddlewareAllowsSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	guard := NewAdminPermissionMiddleware(permissionRepoStub{allowed: false})
	r.GET("/admin/users", func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})
		c.Set(string(ContextKeyUserRole), service.RoleSuperAdmin)
	}, guard(service.AdminResourceUsers, service.AdminActionView), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestAdminPermissionMiddlewareRejectsLimitedAdminWithoutPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	guard := NewAdminPermissionMiddleware(permissionRepoStub{allowed: false})
	r.GET("/admin/users", func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 2})
		c.Set(string(ContextKeyUserRole), service.RoleAdmin)
	}, guard(service.AdminResourceUsers, service.AdminActionUpdate), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if body := w.Body.String(); !containsAll(body, "ADMIN_PERMISSION_DENIED") {
		t.Fatalf("body = %s, want ADMIN_PERMISSION_DENIED", body)
	}
}

func TestAdminPermissionMiddlewareAllowsLimitedAdminWithPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	guard := NewAdminPermissionMiddleware(permissionRepoStub{allowed: true})
	r.GET("/admin/users", func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 2})
		c.Set(string(ContextKeyUserRole), service.RoleAdmin)
	}, guard(service.AdminResourceUsers, service.AdminActionView), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
