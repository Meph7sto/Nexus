package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/nexus/internal/config"
	"github.com/Wei-Shaw/nexus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminComplianceGuardAlwaysAllows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewSettingService(nil, &config.Config{})
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 1})
		c.Next()
	})
	router.Use(AdminComplianceGuard(svc))
	router.GET("/api/v1/admin/users", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "ok", w.Body.String())
}

func TestAdminComplianceGuardAllowsComplianceEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewSettingService(nil, &config.Config{})
	router := gin.New()
	router.Use(AdminComplianceGuard(svc))
	router.GET("/api/v1/admin/compliance", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/compliance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "ok", w.Body.String())
}
