# OpenAI Quota Summary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an admin OpenAI quota summary page that aggregates OpenAI accounts by group and account type, excludes error/inactive accounts from averages while counting them, and projects current or future 5h/7d quota headroom.

**Architecture:** Put the quota math in a small backend service helper so it can be unit-tested without HTTP or database dependencies. Expose it through a read-only admin endpoint that reuses the existing account-management permission resource. Add a focused Vue admin page that calls the endpoint, controls the projection timestamp, and renders grouped summary tables.

**Tech Stack:** Go, Gin, Ent-backed repositories through existing service interfaces, Vue 3, TypeScript, Vitest, Tailwind utility classes.

## Global Constraints

- Only OpenAI accounts are considered.
- `active` accounts are included in averages.
- `error` accounts are excluded from averages and counted.
- `inactive` accounts are excluded from averages and counted.
- Active accounts without Codex usage snapshots are included as theoretical full quota.
- Missing 5-hour snapshot means 5-hour remaining is `100`.
- Missing weekly snapshot means weekly remaining is `100`.
- Future projection sets a window to `100` when `projection_at >= reset_at`.
- Accounts that belong to multiple groups are counted in each group.
- Accounts without a group are counted in an "Ungrouped" bucket.
- The endpoint path is `GET /api/v1/admin/openai/quota-summary`.
- The frontend route is `/admin/openai-quota-summary`.

---

## File Structure

- Create `backend/internal/service/openai_quota_summary.go`: pure aggregation types and `BuildOpenAIQuotaSummary`.
- Create `backend/internal/service/openai_quota_summary_test.go`: backend aggregation and projection tests.
- Modify `backend/internal/service/admin_service.go`: add `GetOpenAIQuotaSummary` to `AdminService` and implement paged account loading plus aggregation.
- Modify `backend/internal/handler/admin/openai_oauth_handler.go`: add request parsing and `QuotaSummary` handler method.
- Modify `backend/internal/server/routes/admin.go`: register `GET /admin/openai/quota-summary`.
- Modify `backend/internal/handler/admin/admin_service_stub_test.go`: add the new admin service interface method to the test stub.
- Create `backend/internal/handler/admin/openai_quota_summary_handler_test.go`: HTTP handler tests for query parsing and JSON response.
- Modify `frontend/src/api/admin/accounts.ts`: add quota summary request/response types and `getOpenAIQuotaSummary`.
- Modify `frontend/src/router/index.ts`: add `/admin/openai-quota-summary`.
- Modify `frontend/src/components/layout/AppSidebar.vue`: add nav item near accounts.
- Modify `frontend/src/i18n/locales/zh.ts` and `frontend/src/i18n/locales/en.ts`: add nav and page labels.
- Create `frontend/src/views/admin/OpenAIQuotaSummaryView.vue`: page UI.
- Create `frontend/src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`: frontend behavior tests.

---

### Task 1: Backend Quota Aggregation Helper

**Files:**
- Create: `backend/internal/service/openai_quota_summary.go`
- Create: `backend/internal/service/openai_quota_summary_test.go`

**Interfaces:**
- Consumes: `service.Account`, `service.Group`, account `Extra` canonical Codex fields.
- Produces:
  - `type OpenAIQuotaSummaryInput struct { ProjectionAt time.Time; GeneratedAt time.Time; GroupFilter *OpenAIQuotaSummaryGroupFilter; AccountType string }`
  - `type OpenAIQuotaSummaryGroupFilter struct { ID *int64; Ungrouped bool }`
  - `type OpenAIQuotaSummaryResponse struct { ProjectionAt time.Time; GeneratedAt time.Time; Groups []OpenAIQuotaSummaryGroup }`
  - `func BuildOpenAIQuotaSummary(accounts []Account, input OpenAIQuotaSummaryInput) OpenAIQuotaSummaryResponse`

- [ ] **Step 1: Write the failing aggregation tests**

Create `backend/internal/service/openai_quota_summary_test.go` with tests named:

```go
package service

import (
	"testing"
	"time"

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
	require.Equal(t, AccountTypeOAuth, row.AccountType)
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
	withoutSnapshot := Account{ID: 2, Name: "new", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Groups: []*Group{group}, GroupIDs: []int64{10}}

	out := BuildOpenAIQuotaSummary([]Account{withSnapshot, withoutSnapshot}, OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	row := out.Groups[0].Rows[0]
	require.Equal(t, 1, row.Missing5HSnapshotCount)
	require.Equal(t, 1, row.Missing7DSnapshotCount)
	require.InDelta(t, 75, row.Avg5HRemainingPercent, 0.001)
	require.InDelta(t, 87.5, row.Avg7DRemainingPercent, 0.001)
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
	require.Equal(t, int64(1), row.Earliest7DRecovery.AccountID)
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

func TestBuildOpenAIQuotaSummary_GroupAndTypeFilters(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	groupA := &Group{ID: 10, Name: "A", Platform: PlatformOpenAI}
	groupB := &Group{ID: 11, Name: "B", Platform: PlatformOpenAI}
	accounts := []Account{
		openAIQuotaSummaryTestAccount(1, "a-oauth", AccountTypeOAuth, StatusActive, 10, 10, now.Add(time.Hour), now.Add(24*time.Hour), groupA),
		openAIQuotaSummaryTestAccount(2, "b-apikey", AccountTypeAPIKey, StatusActive, 10, 10, now.Add(time.Hour), now.Add(24*time.Hour), groupB),
	}
	filterID := int64(10)

	out := BuildOpenAIQuotaSummary(accounts, OpenAIQuotaSummaryInput{
		ProjectionAt: now,
		GeneratedAt:  now,
		AccountType:  AccountTypeOAuth,
		GroupFilter:  &OpenAIQuotaSummaryGroupFilter{ID: &filterID},
	})

	require.Len(t, out.Groups, 1)
	require.Equal(t, int64(10), *out.Groups[0].GroupID)
	require.Len(t, out.Groups[0].Rows, 1)
	require.Equal(t, AccountTypeOAuth, out.Groups[0].Rows[0].AccountType)
}

func openAIQuotaSummaryTestAccount(id int64, name, accountType, status string, used5h, used7d float64, reset5h, reset7d time.Time, groups ...*Group) Account {
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
			"codex_5h_used_percent": used5h,
			"codex_5h_reset_at":      reset5h.Format(time.RFC3339),
			"codex_7d_used_percent": used7d,
			"codex_7d_reset_at":      reset7d.Format(time.RFC3339),
		},
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend; go test ./internal/service -run 'TestBuildOpenAIQuotaSummary'`

Expected: FAIL with undefined symbols such as `BuildOpenAIQuotaSummary` and `OpenAIQuotaSummaryInput`.

- [ ] **Step 3: Implement the aggregation helper**

Create `backend/internal/service/openai_quota_summary.go` with the exact exported types named in the Interfaces section. Implementation requirements:

```go
package service

import (
	"sort"
	"strings"
	"time"
)

type OpenAIQuotaSummaryInput struct {
	ProjectionAt time.Time
	GeneratedAt  time.Time
	GroupFilter  *OpenAIQuotaSummaryGroupFilter
	AccountType  string
}

type OpenAIQuotaSummaryGroupFilter struct {
	ID        *int64
	Ungrouped bool
}

type OpenAIQuotaSummaryResponse struct {
	ProjectionAt time.Time                 `json:"projection_at"`
	GeneratedAt  time.Time                 `json:"generated_at"`
	Groups       []OpenAIQuotaSummaryGroup `json:"groups"`
}

type OpenAIQuotaSummaryGroup struct {
	GroupID   *int64                    `json:"group_id"`
	GroupName string                    `json:"group_name"`
	Ungrouped bool                      `json:"ungrouped"`
	Rows      []OpenAIQuotaSummaryRow   `json:"rows"`
}

type OpenAIQuotaSummaryRow struct {
	AccountType                string                       `json:"account_type"`
	IncludedCount             int                          `json:"included_count"`
	ErrorCount                int                          `json:"error_count"`
	InactiveCount             int                          `json:"inactive_count"`
	OtherExcludedCount        int                          `json:"other_excluded_count"`
	Missing5HSnapshotCount    int                          `json:"missing_5h_snapshot_count"`
	Missing7DSnapshotCount    int                          `json:"missing_7d_snapshot_count"`
	Avg5HRemainingPercent     float64                      `json:"avg_5h_remaining_percent"`
	Avg7DRemainingPercent     float64                      `json:"avg_7d_remaining_percent"`
	Earliest5HRecovery        *OpenAIQuotaRecovery         `json:"earliest_5h_recovery"`
	Earliest7DRecovery        *OpenAIQuotaRecovery         `json:"earliest_7d_recovery"`
}

type OpenAIQuotaRecovery struct {
	AccountID                int64     `json:"account_id"`
	AccountName              string    `json:"account_name"`
	AccountType              string    `json:"account_type"`
	ResetAt                  time.Time `json:"reset_at"`
	RemainingBeforePercent   float64   `json:"remaining_before_percent"`
	RemainingAfterPercent    float64   `json:"remaining_after_percent"`
}
```

Use private bucket structs to sum remaining percentages, then sort groups by ungrouped last and ID ascending, and sort rows by `account_type` ascending. Use `parseExtraFloat64`, `Account.getExtraTime`, and a small `clampPercent` helper.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd backend; go test ./internal/service -run 'TestBuildOpenAIQuotaSummary'`

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add backend/internal/service/openai_quota_summary.go backend/internal/service/openai_quota_summary_test.go
git commit -m "feat: add OpenAI quota summary aggregation"
```

---

### Task 2: Backend Admin Service Method

**Files:**
- Modify: `backend/internal/service/admin_service.go`

**Interfaces:**
- Consumes: `BuildOpenAIQuotaSummary(accounts []Account, input OpenAIQuotaSummaryInput) OpenAIQuotaSummaryResponse`
- Produces: `GetOpenAIQuotaSummary(ctx context.Context, input OpenAIQuotaSummaryInput) (*OpenAIQuotaSummaryResponse, error)` on `AdminService`.

- [ ] **Step 1: Write the failing service test**

Add this test to `backend/internal/service/openai_quota_summary_test.go`:

```go
func TestAdminServiceGetOpenAIQuotaSummary_LoadsAllOpenAIAccountsAcrossPages(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	repo := &openAIQuotaSummaryPagedRepo{
		pages: [][]Account{
			{openAIQuotaSummaryTestAccount(1, "a", AccountTypeOAuth, StatusActive, 0, 0, now.Add(time.Hour), now.Add(24*time.Hour))},
			{openAIQuotaSummaryTestAccount(2, "b", AccountTypeAPIKey, StatusActive, 0, 0, now.Add(time.Hour), now.Add(24*time.Hour))},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	out, err := svc.GetOpenAIQuotaSummary(context.Background(), OpenAIQuotaSummaryInput{ProjectionAt: now, GeneratedAt: now})

	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Groups, 1)
	require.Len(t, out.Groups[0].Rows, 2)
	require.Equal(t, []string{"openai", "openai"}, repo.platforms)
	require.Equal(t, []int{1, 2}, repo.pagesRequested)
}
```

Add a local repo stub in the same test file. Embed `AccountRepository` and implement only `ListWithFilters`; return total `2` and one item per requested page.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend; go test ./internal/service -run 'TestAdminServiceGetOpenAIQuotaSummary'`

Expected: FAIL because `GetOpenAIQuotaSummary` is not defined on `adminServiceImpl`.

- [ ] **Step 3: Implement the admin service method**

Modify `AdminService` in `backend/internal/service/admin_service.go` to include:

```go
GetOpenAIQuotaSummary(ctx context.Context, input OpenAIQuotaSummaryInput) (*OpenAIQuotaSummaryResponse, error)
```

Add implementation on `adminServiceImpl`:

```go
func (s *adminServiceImpl) GetOpenAIQuotaSummary(ctx context.Context, input OpenAIQuotaSummaryInput) (*OpenAIQuotaSummaryResponse, error) {
	if input.GeneratedAt.IsZero() {
		input.GeneratedAt = time.Now().UTC()
	}
	if input.ProjectionAt.IsZero() {
		input.ProjectionAt = input.GeneratedAt
	}
	const pageSize = 500
	page := 1
	var groupID int64
	if input.GroupFilter != nil {
		if input.GroupFilter.Ungrouped {
			groupID = AccountListGroupUngrouped
		} else if input.GroupFilter.ID != nil {
			groupID = *input.GroupFilter.ID
		}
	}
	accounts := make([]Account, 0, pageSize)
	for {
		params := pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    "id",
			SortOrder: pagination.SortOrderAsc,
		}
		batch, pag, err := s.accountRepo.ListWithFilters(ctx, params, PlatformOpenAI, input.AccountType, "", "", groupID, "")
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, batch...)
		if pag == nil || int64(len(accounts)) >= pag.Total {
			break
		}
		page++
	}
	out := BuildOpenAIQuotaSummary(accounts, input)
	return &out, nil
}
```

- [ ] **Step 4: Run the service tests**

Run: `cd backend; go test ./internal/service -run 'Test(BuildOpenAIQuotaSummary|AdminServiceGetOpenAIQuotaSummary)'`

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add backend/internal/service/admin_service.go backend/internal/service/openai_quota_summary_test.go
git commit -m "feat: expose OpenAI quota summary service"
```

---

### Task 3: Backend HTTP Endpoint

**Files:**
- Modify: `backend/internal/handler/admin/openai_oauth_handler.go`
- Modify: `backend/internal/server/routes/admin.go`
- Modify: `backend/internal/handler/admin/admin_service_stub_test.go`
- Create: `backend/internal/handler/admin/openai_quota_summary_handler_test.go`

**Interfaces:**
- Consumes: `AdminService.GetOpenAIQuotaSummary(ctx, input)`.
- Produces: `GET /api/v1/admin/openai/quota-summary`.

- [ ] **Step 1: Write the failing handler test**

Create `backend/internal/handler/admin/openai_quota_summary_handler_test.go`:

```go
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/openai/quota-summary?projection_at=2026-07-06T15:00:00Z&group=12&type=oauth", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.AccountTypeOAuth, adminSvc.lastOpenAIQuotaSummaryInput.AccountType)
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
```

Modify `stubAdminService` in `backend/internal/handler/admin/admin_service_stub_test.go` to add fields:

```go
openAIQuotaSummary          *service.OpenAIQuotaSummaryResponse
lastOpenAIQuotaSummaryInput service.OpenAIQuotaSummaryInput
```

- [ ] **Step 2: Run the handler tests to verify they fail**

Run: `cd backend; go test ./internal/handler/admin -run 'TestOpenAIOAuthHandlerQuotaSummary'`

Expected: FAIL because `QuotaSummary` is not defined and the stub does not satisfy `AdminService`.

- [ ] **Step 3: Implement the handler and route**

In `backend/internal/handler/admin/openai_oauth_handler.go`, add:

```go
// QuotaSummary handles GET /api/v1/admin/openai/quota-summary.
func (h *OpenAIOAuthHandler) QuotaSummary(c *gin.Context) {
	if h.adminService == nil {
		response.Error(c, http.StatusServiceUnavailable, "admin service not available")
		return
	}
	now := time.Now().UTC()
	input := service.OpenAIQuotaSummaryInput{
		ProjectionAt: now,
		GeneratedAt:  now,
		AccountType:  strings.TrimSpace(c.Query("type")),
	}
	if raw := strings.TrimSpace(c.Query("projection_at")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.BadRequest(c, "invalid projection_at")
			return
		}
		input.ProjectionAt = t.UTC()
	}
	if rawGroup := strings.TrimSpace(c.Query("group")); rawGroup != "" {
		filter := &service.OpenAIQuotaSummaryGroupFilter{}
		if rawGroup == accountListGroupUngroupedQueryValue {
			filter.Ungrouped = true
		} else {
			id, err := strconv.ParseInt(rawGroup, 10, 64)
			if err != nil || id <= 0 {
				response.BadRequest(c, "invalid group")
				return
			}
			filter.ID = &id
		}
		input.GroupFilter = filter
	}
	out, err := h.adminService.GetOpenAIQuotaSummary(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}
```

Add the route in `registerOpenAIOAuthRoutes`:

```go
openai.GET("/quota-summary", h.Admin.OpenAIOAuth.QuotaSummary)
```

- [ ] **Step 4: Run the handler tests**

Run: `cd backend; go test ./internal/handler/admin -run 'TestOpenAIOAuthHandlerQuotaSummary'`

Expected: PASS.

- [ ] **Step 5: Run backend focused tests**

Run: `cd backend; go test ./internal/service ./internal/handler/admin -run 'OpenAIQuotaSummary|QuotaSummary'`

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add backend/internal/handler/admin/openai_oauth_handler.go backend/internal/server/routes/admin.go backend/internal/handler/admin/admin_service_stub_test.go backend/internal/handler/admin/openai_quota_summary_handler_test.go
git commit -m "feat: add OpenAI quota summary endpoint"
```

---

### Task 4: Frontend API Client

**Files:**
- Modify: `frontend/src/api/admin/accounts.ts`

**Interfaces:**
- Produces:
  - `OpenAIQuotaSummaryResponse`
  - `OpenAIQuotaSummaryGroup`
  - `OpenAIQuotaSummaryRow`
  - `OpenAIQuotaRecovery`
  - `getOpenAIQuotaSummary(params?: OpenAIQuotaSummaryParams): Promise<OpenAIQuotaSummaryResponse>`

- [ ] **Step 1: Write the failing API test**

Create or extend `frontend/src/api/__tests__/accounts.openaiQuotaSummary.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest'
import { getOpenAIQuotaSummary } from '@/api/admin/accounts'
import { apiClient } from '@/api/client'

vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn()
  }
}))

describe('admin accounts OpenAI quota summary API', () => {
  it('passes projection, group, and type query params', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { projection_at: '2026-07-06T15:00:00Z', generated_at: '2026-07-06T14:00:00Z', groups: [] }
    })

    await getOpenAIQuotaSummary({
      projection_at: '2026-07-06T15:00:00Z',
      group: 'ungrouped',
      type: 'oauth'
    })

    expect(apiClient.get).toHaveBeenCalledWith('/admin/openai/quota-summary', {
      params: {
        projection_at: '2026-07-06T15:00:00Z',
        group: 'ungrouped',
        type: 'oauth'
      }
    })
  })
})
```

- [ ] **Step 2: Run the API test to verify it fails**

Run: `cd frontend; pnpm test:run src/api/__tests__/accounts.openaiQuotaSummary.spec.ts`

Expected: FAIL because `getOpenAIQuotaSummary` is not exported.

- [ ] **Step 3: Implement the API client**

In `frontend/src/api/admin/accounts.ts`, add the types near the OpenAI quota types and export:

```ts
export interface OpenAIQuotaSummaryParams {
 projection_at?: string
 group?: string
 type?: string
}

export interface OpenAIQuotaRecovery {
 account_id: number
 account_name: string
 account_type: string
 reset_at: string
 remaining_before_percent: number
 remaining_after_percent: number
}

export interface OpenAIQuotaSummaryRow {
 account_type: string
 included_count: number
 error_count: number
 inactive_count: number
 other_excluded_count: number
 missing_5h_snapshot_count: number
 missing_7d_snapshot_count: number
 avg_5h_remaining_percent: number
 avg_7d_remaining_percent: number
 earliest_5h_recovery: OpenAIQuotaRecovery | null
 earliest_7d_recovery: OpenAIQuotaRecovery | null
}

export interface OpenAIQuotaSummaryGroup {
 group_id: number | null
 group_name: string
 ungrouped: boolean
 rows: OpenAIQuotaSummaryRow[]
}

export interface OpenAIQuotaSummaryResponse {
 projection_at: string
 generated_at: string
 groups: OpenAIQuotaSummaryGroup[]
}

export async function getOpenAIQuotaSummary(params?: OpenAIQuotaSummaryParams): Promise<OpenAIQuotaSummaryResponse> {
 const { data } = await apiClient.get<OpenAIQuotaSummaryResponse>('/admin/openai/quota-summary', { params })
 return data
}
```

Add `getOpenAIQuotaSummary` to the `accountsAPI` object.

- [ ] **Step 4: Run the API test**

Run: `cd frontend; pnpm test:run src/api/__tests__/accounts.openaiQuotaSummary.spec.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add frontend/src/api/admin/accounts.ts frontend/src/api/__tests__/accounts.openaiQuotaSummary.spec.ts
git commit -m "feat: add OpenAI quota summary API client"
```

---

### Task 5: Frontend Page, Route, Navigation, and Locales

**Files:**
- Create: `frontend/src/views/admin/OpenAIQuotaSummaryView.vue`
- Create: `frontend/src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

**Interfaces:**
- Consumes: `accountsAPI.getOpenAIQuotaSummary(params)`.
- Produces: admin route `/admin/openai-quota-summary` and page rendering grouped quota summary rows.

- [ ] **Step 1: Write the failing page test**

Create `frontend/src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`:

```ts
import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import OpenAIQuotaSummaryView from '../OpenAIQuotaSummaryView.vue'
import { accountsAPI } from '@/api/admin/accounts'

vi.mock('vue-i18n', () => ({
 useI18n: () => ({
  t: (key: string, params?: Record<string, unknown>) => params ? `${key} ${JSON.stringify(params)}` : key
 })
}))

vi.mock('@/api/admin/accounts', () => ({
 accountsAPI: {
  getOpenAIQuotaSummary: vi.fn()
 }
}))

vi.mock('@/stores/app', () => ({
 useAppStore: () => ({
  showError: vi.fn()
 })
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
 default: { template: '<div><slot /></div>' }
}))

const response = {
 projection_at: '2026-07-06T15:00:00Z',
 generated_at: '2026-07-06T14:00:00Z',
 groups: [
  {
   group_id: 12,
   group_name: 'OpenAI Main',
   ungrouped: false,
   rows: [
    {
     account_type: 'oauth',
     included_count: 10,
     error_count: 1,
     inactive_count: 2,
     other_excluded_count: 0,
     missing_5h_snapshot_count: 3,
     missing_7d_snapshot_count: 4,
     avg_5h_remaining_percent: 90,
     avg_7d_remaining_percent: 84.5,
     earliest_5h_recovery: {
      account_id: 42,
      account_name: 'openai-01',
      account_type: 'oauth',
      reset_at: '2026-07-06T16:30:00Z',
      remaining_before_percent: 0,
      remaining_after_percent: 100
     },
     earliest_7d_recovery: null
    }
   ]
  }
 ]
}

describe('OpenAIQuotaSummaryView', () => {
 beforeEach(() => {
  vi.mocked(accountsAPI.getOpenAIQuotaSummary).mockReset()
  vi.mocked(accountsAPI.getOpenAIQuotaSummary).mockResolvedValue(response)
 })

 it('loads and renders grouped summary rows', async () => {
  const wrapper = mount(OpenAIQuotaSummaryView)
  await flushPromises()

  expect(accountsAPI.getOpenAIQuotaSummary).toHaveBeenCalledWith({})
  expect(wrapper.text()).toContain('OpenAI Main')
  expect(wrapper.text()).toContain('oauth')
  expect(wrapper.text()).toContain('90.0%')
  expect(wrapper.text()).toContain('84.5%')
  expect(wrapper.text()).toContain('openai-01')
  expect(wrapper.text()).toContain('#42')
 })

 it('sends a future projection when hours mode is applied', async () => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-07-06T14:00:00Z'))
  const wrapper = mount(OpenAIQuotaSummaryView)
  await flushPromises()

  await wrapper.get('[data-test="projection-mode-hours"]').trigger('click')
  await wrapper.get('[data-test="projection-amount"]').setValue('2')
  await wrapper.get('[data-test="refresh"]').trigger('click')
  await flushPromises()

  expect(accountsAPI.getOpenAIQuotaSummary).toHaveBeenLastCalledWith({
   projection_at: '2026-07-06T16:00:00.000Z'
  })
  vi.useRealTimers()
 })
})
```

- [ ] **Step 2: Run the page test to verify it fails**

Run: `cd frontend; pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`

Expected: FAIL because `OpenAIQuotaSummaryView.vue` does not exist.

- [ ] **Step 3: Implement the Vue page**

Create `frontend/src/views/admin/OpenAIQuotaSummaryView.vue` with:

- `AppLayout` wrapper.
- Projection mode state: `'current' | 'hours' | 'days'`.
- Numeric amount default `1`.
- `projectionParams()` returning `{}` for current, or `{ projection_at: date.toISOString() }` for future modes.
- `loadSummary()` calling `accountsAPI.getOpenAIQuotaSummary`.
- Compact table sections per group.
- Use `data-test="projection-mode-hours"`, `data-test="projection-mode-days"`, `data-test="projection-amount"`, and `data-test="refresh"` for the tests.

Use these formatting helpers in the component:

```ts
function formatPercent(value: number | null | undefined): string {
 if (value == null || Number.isNaN(value)) return '-'
 return `${value.toFixed(1)}%`
}

function formatDateTime(value: string | null | undefined): string {
 if (!value) return '-'
 const date = new Date(value)
 if (Number.isNaN(date.getTime())) return value
 return date.toLocaleString()
}
```

Keep layout utilitarian: toolbar, filters, grouped tables, no hero section.

- [ ] **Step 4: Add route, sidebar item, and locales**

In `frontend/src/router/index.ts`, add an admin route near `/admin/accounts` or `/admin/usage`:

```ts
{
 path: '/admin/openai-quota-summary',
 name: 'AdminOpenAIQuotaSummary',
 component: () => import('@/views/admin/OpenAIQuotaSummaryView.vue'),
 meta: {
  requiresAuth: true,
  requiresAdmin: true,
  title: 'OpenAI Quota Summary',
  titleKey: 'admin.openAIQuotaSummary.title',
  descriptionKey: 'admin.openAIQuotaSummary.description'
 }
}
```

In `AppSidebar.vue`, add a normal admin nav item near accounts:

```ts
{
 path: '/admin/openai-quota-summary',
 label: t('nav.openAIQuotaSummary'),
 icon: ChartBarIcon
}
```

If `ChartBarIcon` is not already defined locally, use an existing nearby icon component in the same file rather than adding a new dependency.

In locales:

```ts
nav: {
 openAIQuotaSummary: 'OpenAI 额度汇总'
}
admin: {
 openAIQuotaSummary: {
  title: 'OpenAI 额度汇总',
  description: '按分区和账号类型查看 OpenAI/Codex 额度状态',
  current: '当前',
  hoursLater: '小时后',
  daysLater: '天后'
 }
}
```

English:

```ts
nav: {
 openAIQuotaSummary: 'OpenAI Quota Summary'
}
admin: {
 openAIQuotaSummary: {
  title: 'OpenAI Quota Summary',
  description: 'View OpenAI/Codex quota status by group and account type',
  current: 'Current',
  hoursLater: 'Hours later',
  daysLater: 'Days later'
 }
}
```

- [ ] **Step 5: Run the page test**

Run: `cd frontend; pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add frontend/src/views/admin/OpenAIQuotaSummaryView.vue frontend/src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts frontend/src/router/index.ts frontend/src/components/layout/AppSidebar.vue frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: add OpenAI quota summary page"
```

---

### Task 6: Final Verification

**Files:**
- No new files.

**Interfaces:**
- Consumes: completed backend and frontend tasks.
- Produces: verified working feature.

- [ ] **Step 1: Run backend focused tests**

Run: `cd backend; go test ./internal/service ./internal/handler/admin -run 'OpenAIQuotaSummary|QuotaSummary'`

Expected: PASS.

- [ ] **Step 2: Run frontend focused tests**

Run: `cd frontend; pnpm test:run src/api/__tests__/accounts.openaiQuotaSummary.spec.ts src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`

Expected: PASS.

- [ ] **Step 3: Run frontend typecheck**

Run: `cd frontend; pnpm typecheck`

Expected: PASS.

- [ ] **Step 4: Review final diff**

Run: `git diff --stat HEAD~5..HEAD`

Expected: Shows backend service/helper/handler/route changes, frontend API/view/route/nav/locale changes, and tests.

- [ ] **Step 5: Commit any verification-only fixes**

If Step 1, 2, or 3 required fixes, commit the fixes with:

```bash
git add <fixed-files>
git commit -m "fix: stabilize OpenAI quota summary"
```

---

## Self-Review Notes

- Spec coverage: backend aggregation, endpoint, page, projection controls, excluded counts, missing snapshot counts, earliest recovery, group/type grouping, ungrouped handling, tests, and navigation are covered.
- Red-flag scan: no task uses unresolved tokens or open-ended "handle later" language.
- Type consistency: backend response fields match frontend API types and the JSON response shape from the design spec.
