package service

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestBuildOpenAIQuotaSummary_ExcludesErrorAndInactiveButCountsThem(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "OpenAI Main", Platform: PlatformOpenAI}
	accounts := []Account{
		openAIQuotaSummaryTestAccount(1, "active-a", AccountTypeOAuth, StatusActive, 20, 40, now.Add(time.Hour), now.Add(24*time.Hour), group),
		openAIQuotaSummaryTestAccount(2, "active-b", AccountTypeOAuth, StatusActive, 0, 0, now.Add(time.Hour), now.Add(24*time.Hour), group),
		openAIQuotaSummaryTestAccount(3, "error-a", AccountTypeOAuth, StatusError, 90, 90, now.Add(time.Hour), now.Add(24*time.Hour), group),
		openAIQuotaSummaryTestAccount(4, "inactive-a", AccountTypeOAuth, StatusInactive, 90, 90, now.Add(time.Hour), now.Add(24*time.Hour), group),
	}

	out := BuildOpenAIQuotaSummary(accounts, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	require.Len(t, out.Groups, 1)
	require.Len(t, out.Groups[0].Rows, 1)
	row := out.Groups[0].Rows[0]
	require.Equal(t, int64(10), *out.Groups[0].GroupID)
	require.Equal(t, "OpenAI Main", out.Groups[0].GroupName)
	require.Equal(t, "plus", row.AccountType)
	require.Equal(t, 2, row.IncludedCount)
	require.Equal(t, 1, row.ErrorCount)
	require.Equal(t, 1, row.InactiveCount)
	require.InDelta(t, 90, row.Avg5HRemainingPercent, 0.001)
	require.InDelta(t, 80, row.Avg7DRemainingPercent, 0.001)
}

func TestBuildOpenAIQuotaSummary_MissingSnapshotsCountAsFullQuota(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "OpenAI Main", Platform: PlatformOpenAI}
	withSnapshot := openAIQuotaSummaryTestAccount(1, "with", AccountTypeOAuth, StatusActive, 50, 25, now.Add(time.Hour), now.Add(24*time.Hour), group)
	withoutSnapshot := Account{
		ID:       2,
		Name:     "new",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Groups:   []*Group{group},
		GroupIDs: []int64{10},
		Extra: map[string]any{
			"chatgpt_plan": "plus",
		},
	}

	out := BuildOpenAIQuotaSummary([]Account{withSnapshot, withoutSnapshot}, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	row := out.Groups[0].Rows[0]
	require.Equal(t, 1, row.Missing5HSnapshotCount)
	require.Equal(t, 1, row.Missing7DSnapshotCount)
	require.InDelta(t, 75, row.Avg5HRemainingPercent, 0.001)
	require.InDelta(t, 87.5, row.Avg7DRemainingPercent, 0.001)
}

func TestBuildOpenAIQuotaSummary_PartialSnapshotsCountAsMissingAndFullQuota(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "OpenAI Main", Platform: PlatformOpenAI}
	withSnapshot := openAIQuotaSummaryTestAccount(1, "with", AccountTypeOAuth, StatusActive, 40, 30, now.Add(time.Hour), now.Add(24*time.Hour), group)
	missingAndBadReset := Account{
		ID:       2,
		Name:     "bad-reset",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Groups:   []*Group{group},
		GroupIDs: []int64{10},
		Extra: map[string]any{
			"chatgpt_plan":          "plus",
			"codex_5h_used_percent": 80,
			"codex_7d_used_percent": 70,
			"codex_7d_reset_at":     "not-rfc3339",
		},
	}
	missingAndBadUsed := Account{
		ID:       3,
		Name:     "bad-used",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Groups:   []*Group{group},
		GroupIDs: []int64{10},
		Extra: map[string]any{
			"chatgpt_plan":          "plus",
			"codex_5h_used_percent": "not-a-percent",
			"codex_5h_reset_at":     now.Add(time.Hour).Format(time.RFC3339),
			"codex_7d_reset_at":     now.Add(24 * time.Hour).Format(time.RFC3339),
		},
	}

	out := BuildOpenAIQuotaSummary([]Account{withSnapshot, missingAndBadReset, missingAndBadUsed}, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	row := out.Groups[0].Rows[0]
	require.Equal(t, 2, row.Missing5HSnapshotCount)
	require.Equal(t, 2, row.Missing7DSnapshotCount)
	require.InDelta(t, 86.666, row.Avg5HRemainingPercent, 0.001)
	require.InDelta(t, 90, row.Avg7DRemainingPercent, 0.001)
}

func TestBuildOpenAIQuotaSummary_NonFiniteSnapshotsCountAsMissingAndFullQuota(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "OpenAI Main", Platform: PlatformOpenAI}
	accounts := []Account{
		openAIQuotaSummaryTestAccountWithRawUsed(1, "nan-string", "NaN", now.Add(time.Hour), group),
		openAIQuotaSummaryTestAccountWithRawUsed(2, "infinity-string", "Infinity", now.Add(time.Hour), group),
		openAIQuotaSummaryTestAccountWithRawUsed(3, "nan-float", math.NaN(), now.Add(time.Hour), group),
		openAIQuotaSummaryTestAccountWithRawUsed(4, "inf-float", math.Inf(1), now.Add(time.Hour), group),
		openAIQuotaSummaryTestAccountWithRawUsed(5, "nan-json-number", json.Number("NaN"), now.Add(time.Hour), group),
		openAIQuotaSummaryTestAccountWithRawUsed(6, "inf-json-number", json.Number("Infinity"), now.Add(time.Hour), group),
	}

	out := BuildOpenAIQuotaSummary(accounts, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	row := out.Groups[0].Rows[0]
	require.Equal(t, len(accounts), row.Missing5HSnapshotCount)
	require.Equal(t, len(accounts), row.Missing7DSnapshotCount)
	require.InDelta(t, 100, row.Avg5HRemainingPercent, 0.001)
	require.InDelta(t, 100, row.Avg7DRemainingPercent, 0.001)
	require.Nil(t, row.Earliest5HRecovery)
	require.Nil(t, row.Earliest7DRecovery)
	_, err := json.Marshal(out)
	require.NoError(t, err)
}

func TestBuildOpenAIQuotaSummary_FutureProjectionResetsWindows(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "OpenAI Main", Platform: PlatformOpenAI}
	account := openAIQuotaSummaryTestAccount(1, "limited", AccountTypeOAuth, StatusActive, 100, 80, now.Add(time.Hour), now.Add(24*time.Hour), group)

	out := BuildOpenAIQuotaSummary([]Account{account}, OpenAIQuotaSummaryInput{ProjectionAt: now.Add(2 * time.Hour), GeneratedAt: now})

	row := out.Groups[0].Rows[0]
	require.InDelta(t, 100, row.Avg5HRemainingPercent, 0.001)
	require.InDelta(t, 20, row.Avg7DRemainingPercent, 0.001)
	require.Nil(t, row.Earliest5HRecovery)
	require.NotNil(t, row.Earliest7DRecovery)
	require.Zero(t, row.Earliest7DRecovery.AccountID)
	require.Empty(t, row.Earliest7DRecovery.AccountName)
	require.InDelta(t, 20, row.Earliest7DRecovery.RemainingBeforePercent, 0.001)
	require.InDelta(t, 100, row.Earliest7DRecovery.RemainingAfterPercent, 0.001)
}

func TestBuildOpenAIQuotaSummary_EarliestRecoveryReportsRowAverageChange(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "OpenAI Main", Platform: PlatformOpenAI}
	accounts := []Account{
		openAIQuotaSummaryTestAccount(1, "empty", AccountTypeOAuth, StatusActive, 100, 20, now.Add(time.Hour), now.Add(24*time.Hour), group),
		openAIQuotaSummaryTestAccount(2, "full", AccountTypeOAuth, StatusActive, 0, 40, now.Add(2*time.Hour), now.Add(48*time.Hour), group),
	}

	out := BuildOpenAIQuotaSummary(accounts, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	row := out.Groups[0].Rows[0]
	require.InDelta(t, 50, row.Avg5HRemainingPercent, 0.001)
	require.NotNil(t, row.Earliest5HRecovery)
	require.Equal(t, now.Add(time.Hour), row.Earliest5HRecovery.ResetAt)
	require.InDelta(t, 50, row.Earliest5HRecovery.RemainingBeforePercent, 0.001)
	require.InDelta(t, 100, row.Earliest5HRecovery.RemainingAfterPercent, 0.001)
	require.Zero(t, row.Earliest5HRecovery.AccountID)
	require.Empty(t, row.Earliest5HRecovery.AccountName)
}

func TestBuildOpenAIQuotaSummary_MultiGroupAndUngroupedBuckets(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	groupA := &Group{ID: 10, Name: "A", Platform: PlatformOpenAI}
	groupB := &Group{ID: 11, Name: "B", Platform: PlatformOpenAI}
	multi := openAIQuotaSummaryTestAccount(1, "multi", AccountTypeOAuth, StatusActive, 10, 10, now.Add(time.Hour), now.Add(24*time.Hour), groupA, groupB)
	ungrouped := openAIQuotaSummaryTestAccount(2, "ungrouped", AccountTypeAPIKey, StatusActive, 0, 0, now.Add(time.Hour), now.Add(24*time.Hour))

	out := BuildOpenAIQuotaSummary([]Account{multi, ungrouped}, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	require.Len(t, out.Groups, 3)
	require.Equal(t, int64(10), *out.Groups[0].GroupID)
	require.Equal(t, int64(11), *out.Groups[1].GroupID)
	require.Nil(t, out.Groups[2].GroupID)
	require.True(t, out.Groups[2].Ungrouped)
}

func TestBuildOpenAIQuotaSummary_GroupAndPlanTypeFilters(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	groupA := &Group{ID: 10, Name: "A", Platform: PlatformOpenAI}
	groupB := &Group{ID: 11, Name: "B", Platform: PlatformOpenAI}
	accounts := []Account{
		openAIQuotaSummaryTestAccountWithPlan(1, "a-plus", AccountTypeOAuth, "plus", StatusActive, 10, 10, now.Add(time.Hour), now.Add(24*time.Hour), groupA),
		openAIQuotaSummaryTestAccountWithPlan(2, "b-pro", AccountTypeOAuth, "pro", StatusActive, 10, 10, now.Add(time.Hour), now.Add(24*time.Hour), groupB),
	}
	filterID := int64(10)

	out := BuildOpenAIQuotaSummary(accounts, OpenAIQuotaSummaryInput{
		ProjectionAt: now,
		GeneratedAt:  now,
		AccountType:  "plus",
		GroupFilter:  &OpenAIQuotaSummaryGroupFilter{ID: &filterID},
	})

	require.Len(t, out.Groups, 1)
	require.Equal(t, int64(10), *out.Groups[0].GroupID)
	require.Len(t, out.Groups[0].Rows, 1)
	require.Equal(t, "plus", out.Groups[0].Rows[0].AccountType)
}

func TestBuildOpenAIQuotaSummary_GroupsOAuthAccountsByChatGPTPlanType(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "A", Platform: PlatformOpenAI}
	accounts := []Account{
		openAIQuotaSummaryTestAccountWithPlan(1, "plus-a", AccountTypeOAuth, "plus", StatusActive, 10, 10, now.Add(time.Hour), now.Add(24*time.Hour), group),
		openAIQuotaSummaryTestAccountWithPlan(2, "pro-a", AccountTypeOAuth, "pro", StatusActive, 20, 20, now.Add(time.Hour), now.Add(24*time.Hour), group),
	}

	out := BuildOpenAIQuotaSummary(accounts, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	require.Len(t, out.Groups, 1)
	require.Len(t, out.Groups[0].Rows, 2)
	require.Equal(t, "plus", out.Groups[0].Rows[0].AccountType)
	require.Equal(t, "pro", out.Groups[0].Rows[1].AccountType)
}

func TestBuildOpenAIQuotaSummary_UsesCredentialPlanType(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	group := &Group{ID: 10, Name: "A", Platform: PlatformOpenAI}
	account := openAIQuotaSummaryTestAccountWithPlan(1, "pro-a", AccountTypeOAuth, "", StatusActive, 10, 10, now.Add(time.Hour), now.Add(24*time.Hour), group)
	account.Credentials = map[string]any{"plan_type": "pro"}
	delete(account.Extra, "chatgpt_plan")

	out := BuildOpenAIQuotaSummary([]Account{account}, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	require.Len(t, out.Groups, 1)
	require.Len(t, out.Groups[0].Rows, 1)
	require.Equal(t, "pro", out.Groups[0].Rows[0].AccountType)
}

func TestAdminServiceGetOpenAIQuotaSummary_LoadsAllOpenAIAccountsAcrossPages(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	repo := &openAIQuotaSummaryPagedRepo{
		pages: [][]Account{
			{openAIQuotaSummaryTestAccountWithPlan(1, "a", AccountTypeOAuth, "plus", StatusActive, 0, 0, now.Add(time.Hour), now.Add(24*time.Hour))},
			{openAIQuotaSummaryTestAccountWithPlan(2, "b", AccountTypeOAuth, "pro", StatusActive, 0, 0, now.Add(time.Hour), now.Add(24*time.Hour))},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	out, err := svc.GetOpenAIQuotaSummary(context.Background(), OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now, AccountType: "plus"})

	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Groups, 1)
	require.Len(t, out.Groups[0].Rows, 1)
	require.Equal(t, "plus", out.Groups[0].Rows[0].AccountType)
	require.Equal(t, []string{"openai", "openai"}, repo.platforms)
	require.Equal(t, []string{"", ""}, repo.accountTypes)
	require.Equal(t, []int{1, 2}, repo.pagesRequested)
}

type openAIQuotaSummaryPagedRepo struct {
	AccountRepository

	pages          [][]Account
	platforms      []string
	accountTypes   []string
	pagesRequested []int
}

func (r *openAIQuotaSummaryPagedRepo) ListWithFilters(_ context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, *pagination.PaginationResult, error) {
	r.platforms = append(r.platforms, platform)
	r.accountTypes = append(r.accountTypes, accountType)
	r.pagesRequested = append(r.pagesRequested, params.Page)

	pageIndex := params.Page - 1
	if pageIndex < 0 || pageIndex >= len(r.pages) {
		return nil, &pagination.PaginationResult{Total: 2, Page: params.Page, PageSize: params.PageSize}, nil
	}
	return r.pages[pageIndex], &pagination.PaginationResult{Total: 2, Page: params.Page, PageSize: params.PageSize}, nil
}

func openAIQuotaSummaryTestAccount(id int64, name, accountType, status string, used5h, used7d float64, reset5h, reset7d time.Time, groups ...*Group) Account {
	return openAIQuotaSummaryTestAccountWithPlan(id, name, accountType, "plus", status, used5h, used7d, reset5h, reset7d, groups...)
}

func openAIQuotaSummaryTestAccountWithPlan(id int64, name, accountType, planType, status string, used5h, used7d float64, reset5h, reset7d time.Time, groups ...*Group) Account {
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	return Account{
		ID:       id,
		Name:     name,
		Platform: PlatformOpenAI,
		Type:     accountType,
		Status:   status,
		Groups:   groups,
		GroupIDs: groupIDs,
		Extra: map[string]any{
			"chatgpt_plan":          planType,
			"codex_5h_used_percent": used5h,
			"codex_5h_reset_at":     reset5h.Format(time.RFC3339),
			"codex_7d_used_percent": used7d,
			"codex_7d_reset_at":     reset7d.Format(time.RFC3339),
		},
	}
}

func openAIQuotaSummaryTestAccountWithRawUsed(id int64, name string, used any, reset time.Time, groups ...*Group) Account {
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	return Account{
		ID:       id,
		Name:     name,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Groups:   groups,
		GroupIDs: groupIDs,
		Extra: map[string]any{
			"codex_5h_used_percent": used,
			"codex_5h_reset_at":     reset.Format(time.RFC3339),
			"codex_7d_used_percent": used,
			"codex_7d_reset_at":     reset.Format(time.RFC3339),
		},
	}
}
