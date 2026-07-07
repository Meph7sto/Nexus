package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuthHandlerQuotaSummary_ParsesProjectionGroupAndType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projection := time.Date(2026, 7, 6, 15, 0, 0, 0, time.UTC)
	adminSvc := &stubAdminService{
		openAIQuotaSummary: &service.OpenAIQuotaSummaryResponse{
			ProjectionAt: projection,
			GeneratedAt:  projection,
			Groups:       []service.OpenAIQuotaSummaryGroup{},
		},
	}
	h := NewOpenAIOAuthHandler(nil, adminSvc, nil)
	router := gin.New()
	router.GET("/api/v1/admin/openai/quota-summary", h.QuotaSummary)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/openai/quota-summary?projection_at=2026-07-06T15:00:00Z&group=12&type=plus", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "plus", adminSvc.lastOpenAIQuotaSummaryInput.AccountType)
	require.NotNil(t, adminSvc.lastOpenAIQuotaSummaryInput.GroupFilter)
	require.NotNil(t, adminSvc.lastOpenAIQuotaSummaryInput.GroupFilter.ID)
	require.Equal(t, int64(12), *adminSvc.lastOpenAIQuotaSummaryInput.GroupFilter.ID)
	require.Equal(t, projection, adminSvc.lastOpenAIQuotaSummaryInput.ProjectionAt)
}

func TestOpenAIOAuthHandlerQuotaSummary_RejectsInvalidProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOpenAIOAuthHandler(nil, &stubAdminService{}, nil)
	router := gin.New()
	router.GET("/api/v1/admin/openai/quota-summary", h.QuotaSummary)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/openai/quota-summary?projection_at=nope", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOpenAIOAuthHandlerQuotaSummary_ParsesUngrouped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := &stubAdminService{openAIQuotaSummary: &service.OpenAIQuotaSummaryResponse{}}
	h := NewOpenAIOAuthHandler(nil, adminSvc, nil)
	router := gin.New()
	router.GET("/api/v1/admin/openai/quota-summary", h.QuotaSummary)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/openai/quota-summary?group=ungrouped", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, adminSvc.lastOpenAIQuotaSummaryInput.GroupFilter.Ungrouped)
}

func (s *stubAdminService) GetOpenAIQuotaSummary(ctx context.Context, input service.OpenAIQuotaSummaryInput) (*service.OpenAIQuotaSummaryResponse, error) {
	s.lastOpenAIQuotaSummaryInput = input
	if s.openAIQuotaSummary != nil {
		return s.openAIQuotaSummary, nil
	}
	return &service.OpenAIQuotaSummaryResponse{ProjectionAt: input.ProjectionAt, GeneratedAt: input.GeneratedAt}, nil
}
