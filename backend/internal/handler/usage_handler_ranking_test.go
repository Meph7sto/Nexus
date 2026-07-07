package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/pkg/pagination"
	"github.com/Wei-Shaw/nexus/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/nexus/internal/server/middleware"
	"github.com/Wei-Shaw/nexus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type rankingUsageRepoCapture struct {
	service.UsageLogRepository
	params    pagination.PaginationParams
	rankBy    usagestats.UserUsageRankingSort
	startTime time.Time
	endTime   time.Time
	rows      []usagestats.UserUsageRankingItem
	total     int64
}

func (s *rankingUsageRepoCapture) GetUserUsageRanking(ctx context.Context, params pagination.PaginationParams, rankBy usagestats.UserUsageRankingSort, startTime, endTime time.Time) ([]usagestats.UserUsageRankingItem, *pagination.PaginationResult, error) {
	s.params = params
	s.rankBy = rankBy
	s.startTime = startTime
	s.endTime = endTime
	total := s.total
	if total == 0 {
		total = int64(len(s.rows))
	}
	return s.rows, &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    1,
	}, nil
}

func newUsageRankingTestRouter(repo *rankingUsageRepoCapture, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	handler := NewUsageHandler(usageSvc, nil, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Set(string(middleware2.ContextKeyUserRole), role)
		c.Next()
	})
	router.GET("/usage/ranking", handler.Ranking)
	return router
}

func decodeRankingResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	data, ok := envelope["data"].(map[string]any)
	require.True(t, ok)
	return data
}

func TestUsageRankingMasksIdentityForRegularUsers(t *testing.T) {
	repo := &rankingUsageRepoCapture{
		rows: []usagestats.UserUsageRankingItem{{
			Rank:            1,
			UserID:          7,
			Nickname:        "alice",
			Email:           "alice@example.com",
			Requests:        3,
			TotalTokens:     123,
			TotalActualCost: 0.45,
		}},
	}
	router := newUsageRankingTestRouter(repo, service.RoleUser)

	req := httptest.NewRequest(http.MethodGet, "/usage/ranking?rank_by=tokens&start_date=2026-03-01&end_date=2026-03-01&page=2&page_size=10&timezone=UTC", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, usagestats.UserUsageRankingByTokens, repo.rankBy)
	require.Equal(t, 2, repo.params.Page)
	require.Equal(t, 10, repo.params.PageSize)
	require.Equal(t, "2026-03-01T00:00:00Z", repo.startTime.UTC().Format(time.RFC3339))
	require.Equal(t, "2026-03-02T00:00:00Z", repo.endTime.UTC().Format(time.RFC3339))

	data := decodeRankingResponse(t, rec)
	items := data["items"].([]any)
	item := items[0].(map[string]any)
	require.NotContains(t, item, "user_id")
	require.Equal(t, "a***e", item["nickname"])
	require.Equal(t, "a****@example.com", item["email"])
}

func TestUsageRankingShowsIdentityForAdmins(t *testing.T) {
	repo := &rankingUsageRepoCapture{
		rows: []usagestats.UserUsageRankingItem{{
			Rank:            1,
			UserID:          7,
			Nickname:        "alice",
			Email:           "alice@example.com",
			Requests:        3,
			TotalTokens:     123,
			TotalActualCost: 0.45,
		}},
	}
	router := newUsageRankingTestRouter(repo, service.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/usage/ranking?rank_by=cost", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, usagestats.UserUsageRankingByCost, repo.rankBy)
	data := decodeRankingResponse(t, rec)
	item := data["items"].([]any)[0].(map[string]any)
	require.Equal(t, float64(7), item["user_id"])
	require.Equal(t, "alice", item["nickname"])
	require.Equal(t, "alice@example.com", item["email"])
}

func TestUsageRankingShowsIdentityForSuperAdmins(t *testing.T) {
	repo := &rankingUsageRepoCapture{
		rows: []usagestats.UserUsageRankingItem{{
			Rank:            1,
			UserID:          7,
			Nickname:        "alice",
			Email:           "alice@example.com",
			Requests:        3,
			TotalTokens:     123,
			TotalActualCost: 0.45,
		}},
	}
	router := newUsageRankingTestRouter(repo, service.RoleSuperAdmin)

	req := httptest.NewRequest(http.MethodGet, "/usage/ranking?rank_by=cost", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeRankingResponse(t, rec)
	item := data["items"].([]any)[0].(map[string]any)
	require.Equal(t, float64(7), item["user_id"])
	require.Equal(t, "alice", item["nickname"])
	require.Equal(t, "alice@example.com", item["email"])
}

func TestUsageRankingRejectsInvalidInputs(t *testing.T) {
	tests := []string{
		"/usage/ranking?rank_by=requests",
		"/usage/ranking?start_date=bad-date",
	}

	for _, path := range tests {
		repo := &rankingUsageRepoCapture{}
		router := newUsageRankingTestRouter(repo, service.RoleUser)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, path)
	}
}
