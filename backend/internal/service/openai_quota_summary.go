package service

import (
	"sort"
	"strings"
	"time"
)

const StatusInactive = "inactive"

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
	row   OpenAIQuotaSummaryRow
	sum5h float64
	sum7d float64
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
		if account.Platform != PlatformOpenAI || !matchesOpenAIQuotaAccountType(account, input.AccountType) {
			continue
		}

		for _, membership := range openAIQuotaSummaryMemberships(account, input.GroupFilter) {
			groupBucket := getOpenAIQuotaSummaryGroupBucket(groups, membership)
			rowBucket := getOpenAIQuotaSummaryRowBucket(groupBucket, account.Type)
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

func matchesOpenAIQuotaAccountType(account *Account, filter string) bool {
	filter = strings.TrimSpace(filter)
	return filter == "" || account.Type == filter
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

	if recovery := openAIQuotaSummaryRecovery(account, "codex_5h_reset_at", remaining5h, missing5h, projectionAt); recovery != nil {
		bucket.row.Earliest5HRecovery = earlierOpenAIQuotaRecovery(bucket.row.Earliest5HRecovery, recovery)
	}
	if recovery := openAIQuotaSummaryRecovery(account, "codex_7d_reset_at", remaining7d, missing7d, projectionAt); recovery != nil {
		bucket.row.Earliest7DRecovery = earlierOpenAIQuotaRecovery(bucket.row.Earliest7DRecovery, recovery)
	}
}

func openAIQuotaSummaryRemaining(account *Account, usedKey, resetKey string, projectionAt time.Time) (float64, bool) {
	if account.Extra == nil {
		return 100, true
	}
	rawUsed, ok := account.Extra[usedKey]
	if !ok {
		return 100, true
	}
	resetAt := account.getExtraTime(resetKey)
	if !resetAt.IsZero() && !projectionAt.Before(resetAt) {
		return 100, false
	}
	return 100 - clampPercent(parseExtraFloat64(rawUsed)), false
}

func openAIQuotaSummaryRecovery(account *Account, resetKey string, remaining float64, missing bool, projectionAt time.Time) *OpenAIQuotaRecovery {
	if missing || remaining >= 100 {
		return nil
	}
	resetAt := account.getExtraTime(resetKey)
	if resetAt.IsZero() || !resetAt.After(projectionAt) {
		return nil
	}
	return &OpenAIQuotaRecovery{
		AccountID:              account.ID,
		AccountName:            account.Name,
		AccountType:            account.Type,
		ResetAt:                resetAt,
		RemainingBeforePercent: remaining,
		RemainingAfterPercent:  100,
	}
}

func earlierOpenAIQuotaRecovery(current, candidate *OpenAIQuotaRecovery) *OpenAIQuotaRecovery {
	if current == nil || candidate.ResetAt.Before(current.ResetAt) {
		return candidate
	}
	return current
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
