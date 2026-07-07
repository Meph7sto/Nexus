package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/nexus/internal/pkg/logger"
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

type UsageInteractionCapture struct {
	RequestContent    map[string]any
	ResponseContent   map[string]any
	RequestParameters map[string]any
	RoutingContext    map[string]any
	RawRequestJSON    map[string]any
	RawResponseJSON   map[string]any
}

type UsageInteractionRepository interface {
	Create(ctx context.Context, input UsageInteractionInput, redactionApplied bool, redactionKeys []string) error
	GetByUsageLogID(ctx context.Context, usageLogID int64, includeRaw bool) (*UsageInteraction, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type usageLogInteractionWriter interface {
	CreateWithUsageInteraction(ctx context.Context, usageLog *UsageLog, interactionService *UsageInteractionService, capture *UsageInteractionCapture, captureErr error) (bool, error)
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
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"raw_text": string(raw)}
	}
	if object, ok := out.(map[string]any); ok {
		return object
	}
	return map[string]any{"raw_json": out}
}

func JSONMapFromRawForUsageInteraction(raw []byte) map[string]any {
	return jsonMapFromRaw(raw)
}

func BuildUsageInteractionContentFromRequestBody(body []byte) map[string]any {
	return jsonMapFromRaw(rawUsageInteractionPayload(body))
}

func BuildUsageInteractionContentFromResponseBody(body []byte) map[string]any {
	return jsonMapFromRaw(rawUsageInteractionPayload(body))
}

func BuildUsageInteractionCapture(requestBody, responseBody []byte, requestParameters map[string]any) *UsageInteractionCapture {
	if requestParameters == nil {
		requestParameters = map[string]any{}
	}
	return &UsageInteractionCapture{
		RequestContent:    BuildUsageInteractionContentFromRequestBody(requestBody),
		ResponseContent:   BuildUsageInteractionContentFromResponseBody(responseBody),
		RequestParameters: requestParameters,
		RawRequestJSON:    JSONMapFromRawForUsageInteraction(requestBody),
		RawResponseJSON:   JSONMapFromRawForUsageInteraction(responseBody),
	}
}

func rawUsageInteractionPayload(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func usageInteractionInputFromUsageLog(usageLog *UsageLog, capture *UsageInteractionCapture) UsageInteractionInput {
	input := UsageInteractionInput{}
	if usageLog != nil {
		input.UsageLogID = usageLog.ID
		input.RequestID = usageLog.RequestID
		input.UserID = usageLog.UserID
		input.APIKeyID = usageLog.APIKeyID
		input.AccountID = usageLog.AccountID
		input.GroupID = usageLog.GroupID
		input.CreatedAt = usageLog.CreatedAt
		input.RoutingContext = usageInteractionRoutingContextFromUsageLog(usageLog)
	}
	if capture != nil {
		input.RequestContent = capture.RequestContent
		input.ResponseContent = capture.ResponseContent
		input.RequestParameters = capture.RequestParameters
		if input.RoutingContext == nil {
			input.RoutingContext = map[string]any{}
		}
		for key, value := range capture.RoutingContext {
			input.RoutingContext[key] = value
		}
		input.RawRequestJSON = capture.RawRequestJSON
		input.RawResponseJSON = capture.RawResponseJSON
	}
	return input
}

func usageInteractionRoutingContextFromUsageLog(usageLog *UsageLog) map[string]any {
	if usageLog == nil {
		return map[string]any{}
	}
	routing := map[string]any{}
	addString := func(key string, value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			routing[key] = trimmed
		}
	}
	addStringPtr := func(key string, value *string) {
		if value != nil {
			addString(key, *value)
		}
	}
	if usageLog.UserID > 0 {
		routing["user_id"] = usageLog.UserID
	}
	if usageLog.APIKeyID > 0 {
		routing["api_key_id"] = usageLog.APIKeyID
	}
	if usageLog.AccountID > 0 {
		routing["account_id"] = usageLog.AccountID
	}
	if usageLog.GroupID != nil {
		routing["group_id"] = *usageLog.GroupID
	}
	if usageLog.ChannelID != nil {
		routing["channel_id"] = *usageLog.ChannelID
	}
	addStringPtr("inbound_endpoint", usageLog.InboundEndpoint)
	addStringPtr("upstream_endpoint", usageLog.UpstreamEndpoint)
	requestedModel := usageLog.RequestedModel
	if strings.TrimSpace(requestedModel) == "" {
		requestedModel = usageLog.Model
	}
	addString("requested_model", requestedModel)
	addString("mapped_model", usageLog.Model)
	if usageLog.UpstreamModel != nil {
		addString("upstream_model", *usageLog.UpstreamModel)
	} else {
		addString("upstream_model", usageLog.Model)
	}
	addStringPtr("model_mapping_chain", usageLog.ModelMappingChain)
	if requestType := usageLog.EffectiveRequestType().String(); requestType != RequestTypeUnknown.String() {
		routing["request_type"] = requestType
	}
	if usageLog.Stream {
		routing["stream"] = true
	}
	return routing
}

func (s *UsageInteractionService) RecordForUsageLog(ctx context.Context, usageLog *UsageLog, capture *UsageInteractionCapture, captureErr error) error {
	if s == nil || usageLog == nil || usageLog.ID <= 0 {
		return nil
	}
	input := usageInteractionInputFromUsageLog(usageLog, capture)
	if captureErr != nil {
		return s.RecordFailed(ctx, input, captureErr.Error())
	}
	return s.RecordComplete(ctx, input)
}

func (s *UsageInteractionService) RecordForUsageLogWithFallback(
	ctx context.Context,
	usageLog *UsageLog,
	capture *UsageInteractionCapture,
	captureErr error,
	logKey string,
) error {
	if s == nil || usageLog == nil || usageLog.ID <= 0 {
		return nil
	}

	err := s.RecordForUsageLog(ctx, usageLog, capture, captureErr)
	if err == nil {
		return nil
	}
	logger.LegacyPrintf(logKey, "Record usage interaction failed: %v", err)

	if captureErr == nil {
		if failedErr := s.RecordForUsageLog(ctx, usageLog, nil, err); failedErr == nil {
			return nil
		} else {
			logger.LegacyPrintf(logKey, "Record failed usage interaction placeholder failed: %v", failedErr)
			err = failedErr
		}
	}
	return err
}

func writeUsageLogWithOptionalInteraction(
	ctx context.Context,
	repo UsageLogRepository,
	usageLog *UsageLog,
	interactionService *UsageInteractionService,
	capture *UsageInteractionCapture,
	captureErr error,
	logKey string,
	recordInteractions bool,
) error {
	if recordInteractions && interactionService != nil {
		usageCtx, cancel := detachedBillingContext(ctx)
		defer cancel()
		writer, ok := repo.(usageLogInteractionWriter)
		if !ok {
			return fmt.Errorf("usage interaction recording requires atomic usage log writer")
		}
		if _, err := writer.CreateWithUsageInteraction(usageCtx, usageLog, interactionService, capture, captureErr); err != nil {
			logger.LegacyPrintf(logKey, "Create usage log with interaction failed: %v", err)
			return err
		}
		return nil
	}
	_, err := writeUsageLogBestEffort(ctx, repo, usageLog, logKey, false)
	return err
}
