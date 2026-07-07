package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUsageHandlerGetInteractionReturnsDetailAndPassesIncludeRaw(t *testing.T) {
	repo := &usageInteractionHandlerRepoStub{
		detail: &service.UsageInteraction{
			ID:              1,
			UsageLogID:      42,
			RequestID:       "req-42",
			CaptureStatus:   service.UsageInteractionCaptureComplete,
			RequestContent:  map[string]any{"prompt": "full prompt"},
			ResponseContent: map[string]any{"output": "full output"},
			RawRequestJSON:  map[string]any{"messages": []any{"full raw"}},
			RawAvailable:    true,
			CreatedAt:       time.Date(2026, 7, 6, 23, 0, 0, 0, time.UTC),
		},
	}
	router := setupUsageInteractionRouter(service.NewUsageInteractionService(repo, nil))

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/42/interaction?include_raw=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.gotUsageLogID)
	require.True(t, repo.gotIncludeRaw)

	var resp struct {
		Data struct {
			Exists      bool                      `json:"exists"`
			Interaction *service.UsageInteraction `json:"interaction"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Data.Exists)
	require.NotNil(t, resp.Data.Interaction)
	require.Equal(t, "full prompt", resp.Data.Interaction.RequestContent["prompt"])
	require.Equal(t, "full output", resp.Data.Interaction.ResponseContent["output"])
	require.NotNil(t, resp.Data.Interaction.RawRequestJSON)
}

func TestUsageHandlerGetInteractionReturnsNotRecordedWhenMissing(t *testing.T) {
	repo := &usageInteractionHandlerRepoStub{err: service.ErrUsageInteractionNotFound}
	router := setupUsageInteractionRouter(service.NewUsageInteractionService(repo, nil))

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/7/interaction", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Exists bool   `json:"exists"`
			Reason string `json:"reason"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.False(t, resp.Data.Exists)
	require.Equal(t, "not_recorded", resp.Data.Reason)
}

func setupUsageInteractionRouter(interactionService *service.UsageInteractionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewUsageHandler(nil, nil, nil, nil, interactionService)
	router := gin.New()
	router.GET("/admin/usage/:id/interaction", handler.GetInteraction)
	return router
}

type usageInteractionHandlerRepoStub struct {
	detail        *service.UsageInteraction
	err           error
	gotUsageLogID int64
	gotIncludeRaw bool
}

func (s *usageInteractionHandlerRepoStub) Create(context.Context, service.UsageInteractionInput, bool, []string) error {
	return nil
}

func (s *usageInteractionHandlerRepoStub) GetByUsageLogID(_ context.Context, usageLogID int64, includeRaw bool) (*service.UsageInteraction, error) {
	s.gotUsageLogID = usageLogID
	s.gotIncludeRaw = includeRaw
	return s.detail, s.err
}

func (s *usageInteractionHandlerRepoStub) DeleteOlderThan(context.Context, time.Time) (int64, error) {
	return 0, nil
}
