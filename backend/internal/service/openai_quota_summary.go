package service

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const StatusInactive = "inactive"
const openAIQuotaSummaryUnknownPlanType = "unknown"

var openAIQuotaSummaryPlanTypeExtraKeys = []string{
	"chatgpt_plan",
	"plan",
	"account_plan",
	"subscription_plan",
	"subscription_tier",
	"tier",
	"quota_tier",
	"workspace_plan",
	"codex_plan",
}

var openAIQuotaSummaryPlanTypeCredentialKeys = []string{
	"plan_type",
	"chatgpt_plan",
	"subscription_plan",
	"subscription_tier",
	"tier",
}

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
	GroupID   *int64                  `json:"group_id"`
	GroupName string                  `json:"group_name"`
	Ungrouped bool                    `json:"ungrouped"`
	Rows      []OpenAIQuotaSummaryRow `json:"rows"`
}

type OpenAIQuotaSummaryRow struct {
	AccountType            string               `json:"account_type"`
	IncludedCount          int                  `json:"included_count"`
	ErrorCount             int                  `json:"error_count"`
	InactiveCount          int                  `json:"inactive_count"`
	OtherExcludedCount     int                  `json:"other_excluded_count"`
	Missing5HSnapshotCount int                  `json:"missing_5h_snapshot_count"`
	Missing7DSnapshotCount int                  `json:"missing_7d_snapshot_count"`
	Avg5HRemainingPercent  float64              `json:"avg_5h_remaining_percent"`
	Avg7DRemainingPercent  float64              `json:"avg_7d_remaining_percent"`
	Earliest5HRecovery     *OpenAIQuotaRecovery `json:"earliest_5h_recovery"`
	Earliest7DRecovery     *OpenAIQuotaRecovery `json:"earliest_7d_recovery"`
}

type OpenAIQuotaRecovery struct {
	AccountID              int64     `json:"account_id"`
	AccountName            string    `json:"account_name"`
	AccountType            string    `json:"account_type"`
	ResetAt                time.Time `json:"reset_at"`
	RemainingBeforePercent float64   `json:"remaining_before_percent"`
	RemainingAfterPercent  float64   `json:"remaining_after_percent"`
}

type openAIQuotaSummaryGroupBucket struct {
	id        *int64
	name      string
	ungrouped bool
	rows      map[string]*openAIQuotaSummaryRowBucket
}

type openAIQuotaSummaryRowBucket struct {
	row       OpenAIQuotaSummaryRow
	sum5h     float64
	sum7d     float64
	windows5h []openAIQuotaSummaryWindowSnapshot
	windows7d []openAIQuotaSummaryWindowSnapshot
}

type openAIQuotaSummaryWindowSnapshot struct {
	resetAt   time.Time
	remaining float64
	missing   bool
}

type openAIQuotaSummaryGroupMembership struct {
	id        *int64
	name      string
	ungrouped bool
}

func BuildOpenAIQuotaSummary(accounts []Account, input OpenAIQuotaSummaryInput) OpenAIQuotaSummaryResponse {
	groups := make(map[string]*openAIQuotaSummaryGroupBucket)
	for i := range accounts {
		account := &accounts[i]
		planType := openAIQuotaSummaryPlanType(account)
		if account.Platform != PlatformOpenAI || !matchesOpenAIQuotaPlanType(planType, input.AccountType) {
			continue
		}

		for _, membership := range openAIQuotaSummaryMemberships(account, input.GroupFilter) {
			groupBucket := getOpenAIQuotaSummaryGroupBucket(groups, membership)
			rowBucket := getOpenAIQuotaSummaryRowBucket(groupBucket, planType)
			addOpenAIQuotaSummaryAccount(rowBucket, account, input.ProjectionAt)
		}
	}

	out := OpenAIQuotaSummaryResponse{
		ProjectionAt: input.ProjectionAt,
		GeneratedAt:  input.GeneratedAt,
		Groups:       make([]OpenAIQuotaSummaryGroup, 0, len(groups)),
	}
	for _, groupBucket := range groups {
		group := OpenAIQuotaSummaryGroup{
			GroupID:   groupBucket.id,
			GroupName: groupBucket.name,
			Ungrouped: groupBucket.ungrouped,
			Rows:      make([]OpenAIQuotaSummaryRow, 0, len(groupBucket.rows)),
		}
		for _, rowBucket := range groupBucket.rows {
			row := rowBucket.row
			if row.IncludedCount > 0 {
				row.Avg5HRemainingPercent = rowBucket.sum5h / float64(row.IncludedCount)
				row.Avg7DRemainingPercent = rowBucket.sum7d / float64(row.IncludedCount)
				row.Earliest5HRecovery = openAIQuotaSummaryAggregateRecovery(rowBucket.windows5h, row.Avg5HRemainingPercent, row.IncludedCount, input.ProjectionAt)
				row.Earliest7DRecovery = openAIQuotaSummaryAggregateRecovery(rowBucket.windows7d, row.Avg7DRemainingPercent, row.IncludedCount, input.ProjectionAt)
			}
			group.Rows = append(group.Rows, row)
		}
		sort.Slice(group.Rows, func(i, j int) bool {
			return group.Rows[i].AccountType < group.Rows[j].AccountType
		})
		out.Groups = append(out.Groups, group)
	}
	sort.Slice(out.Groups, func(i, j int) bool {
		if out.Groups[i].Ungrouped != out.Groups[j].Ungrouped {
			return !out.Groups[i].Ungrouped
		}
		if out.Groups[i].GroupID == nil || out.Groups[j].GroupID == nil {
			return out.Groups[j].GroupID == nil
		}
		return *out.Groups[i].GroupID < *out.Groups[j].GroupID
	})

	return out
}

func matchesOpenAIQuotaPlanType(planType, filter string) bool {
	filter = strings.TrimSpace(filter)
	return filter == "" || strings.EqualFold(planType, filter)
}

func openAIQuotaSummaryPlanType(account *Account) string {
	if account == nil {
		return openAIQuotaSummaryUnknownPlanType
	}
	for _, key := range openAIQuotaSummaryPlanTypeCredentialKeys {
		if planType := normalizeOpenAIQuotaSummaryPlanType(openAIQuotaSummaryMapString(account.Credentials, key)); planType != "" {
			return planType
		}
	}
	for _, key := range openAIQuotaSummaryPlanTypeExtraKeys {
		if planType := normalizeOpenAIQuotaSummaryPlanType(account.getExtraString(key)); planType != "" {
			return planType
		}
	}
	return openAIQuotaSummaryUnknownPlanType
}

func normalizeOpenAIQuotaSummaryPlanType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "chatgpt ")
	value = strings.TrimPrefix(value, "chatgpt-")
	value = strings.TrimPrefix(value, "chatgpt_")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, " ", "-")
}

func openAIQuotaSummaryMapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func openAIQuotaSummaryMemberships(account *Account, filter *OpenAIQuotaSummaryGroupFilter) []openAIQuotaSummaryGroupMembership {
	if filter != nil && filter.Ungrouped {
		if len(account.Groups) == 0 && len(account.GroupIDs) == 0 {
			return []openAIQuotaSummaryGroupMembership{{ungrouped: true, name: "Ungrouped"}}
		}
		return nil
	}

	memberships := make([]openAIQuotaSummaryGroupMembership, 0, max(len(account.Groups), len(account.GroupIDs)))
	seen := make(map[int64]struct{}, len(account.Groups)+len(account.GroupIDs))
	for _, group := range account.Groups {
		if group == nil {
			continue
		}
		if filter != nil && filter.ID != nil && group.ID != *filter.ID {
			continue
		}
		id := group.ID
		memberships = append(memberships, openAIQuotaSummaryGroupMembership{id: &id, name: group.Name})
		seen[group.ID] = struct{}{}
	}
	for _, groupID := range account.GroupIDs {
		if _, ok := seen[groupID]; ok {
			continue
		}
		if filter != nil && filter.ID != nil && groupID != *filter.ID {
			continue
		}
		id := groupID
		memberships = append(memberships, openAIQuotaSummaryGroupMembership{id: &id})
		seen[groupID] = struct{}{}
	}
	if len(memberships) == 0 && len(account.Groups) == 0 && len(account.GroupIDs) == 0 && (filter == nil || filter.ID == nil) {
		return []openAIQuotaSummaryGroupMembership{{ungrouped: true, name: "Ungrouped"}}
	}
	return memberships
}

func getOpenAIQuotaSummaryGroupBucket(groups map[string]*openAIQuotaSummaryGroupBucket, membership openAIQuotaSummaryGroupMembership) *openAIQuotaSummaryGroupBucket {
	key := openAIQuotaSummaryGroupKey(membership)
	if bucket, ok := groups[key]; ok {
		if bucket.name == "" {
			bucket.name = membership.name
		}
		return bucket
	}
	bucket := &openAIQuotaSummaryGroupBucket{
		id:        membership.id,
		name:      membership.name,
		ungrouped: membership.ungrouped,
		rows:      make(map[string]*openAIQuotaSummaryRowBucket),
	}
	groups[key] = bucket
	return bucket
}

func getOpenAIQuotaSummaryRowBucket(group *openAIQuotaSummaryGroupBucket, accountType string) *openAIQuotaSummaryRowBucket {
	if bucket, ok := group.rows[accountType]; ok {
		return bucket
	}
	bucket := &openAIQuotaSummaryRowBucket{
		row: OpenAIQuotaSummaryRow{AccountType: accountType},
	}
	group.rows[accountType] = bucket
	return bucket
}

func addOpenAIQuotaSummaryAccount(bucket *openAIQuotaSummaryRowBucket, account *Account, projectionAt time.Time) {
	switch strings.TrimSpace(account.Status) {
	case StatusActive:
		bucket.row.IncludedCount++
	case StatusError:
		bucket.row.ErrorCount++
		return
	case StatusInactive, StatusDisabled:
		bucket.row.InactiveCount++
		return
	default:
		bucket.row.OtherExcludedCount++
		return
	}

	remaining5h, missing5h := openAIQuotaSummaryRemaining(account, "codex_5h_used_percent", "codex_5h_reset_at", projectionAt)
	remaining7d, missing7d := openAIQuotaSummaryRemaining(account, "codex_7d_used_percent", "codex_7d_reset_at", projectionAt)
	if missing5h {
		bucket.row.Missing5HSnapshotCount++
	}
	if missing7d {
		bucket.row.Missing7DSnapshotCount++
	}
	bucket.sum5h += remaining5h
	bucket.sum7d += remaining7d
	bucket.windows5h = append(bucket.windows5h, openAIQuotaSummaryWindowSnapshot{
		resetAt:   account.getExtraTime("codex_5h_reset_at"),
		remaining: remaining5h,
		missing:   missing5h,
	})
	bucket.windows7d = append(bucket.windows7d, openAIQuotaSummaryWindowSnapshot{
		resetAt:   account.getExtraTime("codex_7d_reset_at"),
		remaining: remaining7d,
		missing:   missing7d,
	})
}

func openAIQuotaSummaryRemaining(account *Account, usedKey, resetKey string, projectionAt time.Time) (float64, bool) {
	if account.Extra == nil {
		return 100, true
	}
	rawUsed, ok := account.Extra[usedKey]
	if !ok {
		return 100, true
	}
	usedPercent, ok := parseOpenAIQuotaSummaryUsedPercent(rawUsed)
	if !ok {
		return 100, true
	}
	resetAt := account.getExtraTime(resetKey)
	if resetAt.IsZero() {
		return 100, true
	}
	if !projectionAt.Before(resetAt) {
		return 100, false
	}
	return 100 - clampPercent(usedPercent), false
}

func openAIQuotaSummaryAggregateRecovery(windows []openAIQuotaSummaryWindowSnapshot, currentAverage float64, includedCount int, projectionAt time.Time) *OpenAIQuotaRecovery {
	if includedCount <= 0 {
		return nil
	}

	var earliest time.Time
	for _, window := range windows {
		if window.missing || window.remaining >= 100 || window.resetAt.IsZero() || !window.resetAt.After(projectionAt) {
			continue
		}
		if earliest.IsZero() || window.resetAt.Before(earliest) {
			earliest = window.resetAt
		}
	}
	if earliest.IsZero() {
		return nil
	}

	afterSum := 0.0
	for _, window := range windows {
		remaining := window.remaining
		if !window.missing && !window.resetAt.IsZero() && !window.resetAt.After(earliest) {
			remaining = 100
		}
		afterSum += remaining
	}

	return &OpenAIQuotaRecovery{
		ResetAt:                earliest,
		RemainingBeforePercent: currentAverage,
		RemainingAfterPercent:  afterSum / float64(includedCount),
	}
}

func parseOpenAIQuotaSummaryUsedPercent(value any) (float64, bool) {
	var (
		percent float64
		ok      bool
	)
	switch v := value.(type) {
	case float64:
		percent, ok = v, true
	case float32:
		percent, ok = float64(v), true
	case int:
		percent, ok = float64(v), true
	case int64:
		percent, ok = float64(v), true
	case json.Number:
		parsed, err := v.Float64()
		if err == nil {
			percent, ok = parsed, true
		}
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			percent, ok = parsed, true
		}
	}
	if !ok || math.IsNaN(percent) || math.IsInf(percent, 0) {
		return 0, false
	}
	return percent, true
}

func openAIQuotaSummaryGroupKey(membership openAIQuotaSummaryGroupMembership) string {
	if membership.ungrouped {
		return "ungrouped"
	}
	if membership.id == nil {
		return "group:"
	}
	return "group:" + strconvFormatInt64(*membership.id)
}

func strconvFormatInt64(v int64) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
