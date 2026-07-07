package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const (
	UsageInteractionCaptureComplete = "complete"
	UsageInteractionCapturePartial  = "partial"
	UsageInteractionCaptureFailed   = "failed"
)

type UsageInteractionSettings struct {
	RecordingEnabled bool `json:"usage_interaction_recording_enabled"`
	StoreRawEnabled  bool `json:"usage_interaction_store_raw_enabled"`
	RetentionDays    int  `json:"usage_interaction_retention_days"`
}

type UsageInteraction struct {
	ID                int64          `json:"id"`
	UsageLogID        int64          `json:"usage_log_id"`
	RequestID         string         `json:"request_id"`
	UserID            int64          `json:"user_id"`
	APIKeyID          int64          `json:"api_key_id"`
	AccountID         int64          `json:"account_id"`
	GroupID           *int64         `json:"group_id,omitempty"`
	CaptureStatus     string         `json:"capture_status"`
	CaptureError      *string        `json:"capture_error,omitempty"`
	RequestContent    map[string]any `json:"request_content"`
	ResponseContent   map[string]any `json:"response_content"`
	RequestParameters map[string]any `json:"request_parameters"`
	RoutingContext    map[string]any `json:"routing_context"`
	RawRequestJSON    map[string]any `json:"raw_request_json,omitempty"`
	RawResponseJSON   map[string]any `json:"raw_response_json,omitempty"`
	RawAvailable      bool           `json:"raw_available"`
	RedactionApplied  bool           `json:"redaction_applied"`
	RedactionKeys     []string       `json:"redaction_keys"`
	CreatedAt         time.Time      `json:"created_at"`
}

type UsageInteractionInput struct {
	UsageLogID        int64
	RequestID         string
	UserID            int64
	APIKeyID          int64
	AccountID         int64
	GroupID           *int64
	CaptureStatus     string
	CaptureError      *string
	RequestContent    map[string]any
	ResponseContent   map[string]any
	RequestParameters map[string]any
	RoutingContext    map[string]any
	RawRequestJSON    map[string]any
	RawResponseJSON   map[string]any
	CreatedAt         time.Time
}

type UsageInteractionRepository interface {
	Create(ctx context.Context, input UsageInteractionInput, redactionApplied bool, redactionKeys []string) error
	GetByUsageLogID(ctx context.Context, usageLogID int64, includeRaw bool) (*UsageInteraction, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

var usageInteractionSecretKeys = map[string]struct{}{
	"authorization": {}, "proxy-authorization": {}, "cookie": {}, "set-cookie": {},
	"api_key": {}, "apikey": {}, "api-key": {}, "key": {}, "token": {}, "access_token": {},
	"refresh_token": {}, "id_token": {}, "session_token": {}, "secret": {}, "client_secret": {},
}

func RedactUsageInteractionPayload(input map[string]any) (map[string]any, []string, bool) {
	out, keys, changed := redactUsageInteractionValue(input)
	result, ok := out.(map[string]any)
	if !ok || result == nil {
		result = map[string]any{}
	}
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	keys = keys[:0]
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return result, keys, changed
}

func redactUsageInteractionValue(value any) (any, []string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		var keys []string
		changed := false
		for key, child := range typed {
			normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "_", ""))
			lookupKey := strings.ToLower(strings.TrimSpace(key))
			if _, ok := usageInteractionSecretKeys[normalized]; ok {
				out[key] = "[REDACTED]"
				keys = append(keys, key)
				changed = true
				continue
			}
			if _, ok := usageInteractionSecretKeys[lookupKey]; ok {
				out[key] = "[REDACTED]"
				keys = append(keys, key)
				changed = true
				continue
			}
			next, childKeys, childChanged := redactUsageInteractionValue(child)
			out[key] = next
			keys = append(keys, childKeys...)
			changed = changed || childChanged
		}
		return out, keys, changed
	case []any:
		out := make([]any, len(typed))
		var keys []string
		changed := false
		for i, child := range typed {
			next, childKeys, childChanged := redactUsageInteractionValue(child)
			out[i] = next
			keys = append(keys, childKeys...)
			changed = changed || childChanged
		}
		return out, keys, changed
	default:
		return value, nil, false
	}
}

func jsonMapFromRaw(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"raw_text": string(raw)}
	}
	return out
}
