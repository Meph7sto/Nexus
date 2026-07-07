package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/nexus/internal/config"
)

func TestRedactUsageInteractionPayload_RedactsCredentialsAndKeepsPrompt(t *testing.T) {
	payload := map[string]any{
		"messages":      []any{map[string]any{"role": "user", "content": "keep this exact prompt"}},
		"Authorization": "Bearer secret",
		"nested": map[string]any{
			"access_token": "tok",
			"normal":       "visible",
		},
	}

	got, keys, changed := RedactUsageInteractionPayload(payload)
	if !changed {
		t.Fatal("expected redaction to be applied")
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 redaction keys, got %v", keys)
	}
	blob, _ := json.Marshal(got)
	text := string(blob)
	if !containsAll(text, "keep this exact prompt", "[REDACTED]", "visible") {
		t.Fatalf("unexpected redacted payload: %s", text)
	}
	if strings.Contains(text, "Bearer secret") || strings.Contains(text, `"access_token":"tok"`) {
		t.Fatalf("secret leaked in payload: %s", text)
	}
}

func TestRedactUsageInteractionPayload_PreservesCredentialKeysWithRedactedValues(t *testing.T) {
	credentialKeys := []string{
		"Authorization",
		"Proxy-Authorization",
		"Cookie",
		"Set-Cookie",
		"api_key",
		"apiKey",
		"key",
		"token",
		"access_token",
		"refresh_token",
		"id_token",
		"session_token",
		"secret",
		"client_secret",
	}
	payload := make(map[string]any, len(credentialKeys)+1)
	payload["normal"] = "visible"
	for _, key := range credentialKeys {
		payload[key] = "leak-" + key
	}

	got, keys, changed := RedactUsageInteractionPayload(payload)
	if !changed {
		t.Fatal("expected redaction to be applied")
	}
	if len(keys) != len(credentialKeys) {
		t.Fatalf("redaction keys = %v, want %d keys", keys, len(credentialKeys))
	}
	for _, key := range credentialKeys {
		value, ok := got[key]
		if !ok {
			t.Fatalf("expected credential key %q to remain present; payload: %#v", key, got)
		}
		if value != "[REDACTED]" {
			t.Fatalf("expected %q to be [REDACTED], got %#v", key, value)
		}
		if !containsString(keys, key) {
			t.Fatalf("expected redaction keys to include %q; got %v", key, keys)
		}
	}
	if got["normal"] != "visible" {
		t.Fatalf("normal key = %#v, want visible", got["normal"])
	}
}

func TestJSONMapFromRawForUsageInteraction_WrapsRootArraysForStructuredRedaction(t *testing.T) {
	raw := []byte(`[{"api_key":"raw-secret","message":"keep"}]`)

	payload := JSONMapFromRawForUsageInteraction(raw)
	redacted, keys, changed := RedactUsageInteractionPayload(payload)

	if !changed {
		t.Fatal("expected root array credential fields to be redacted")
	}
	if !containsString(keys, "api_key") {
		t.Fatalf("redaction keys = %v, want api_key", keys)
	}
	blob, _ := json.Marshal(redacted)
	text := string(blob)
	if !containsAll(text, `"raw_json"`, `"message":"keep"`, `"api_key":"[REDACTED]"`) {
		t.Fatalf("unexpected redacted root array payload: %s", text)
	}
	if strings.Contains(text, "raw-secret") || strings.Contains(text, "raw_text") {
		t.Fatalf("root array payload was not structurally redacted: %s", text)
	}
}

func TestUsageInteractionInputFromUsageLog_PopulatesRoutingContextFromUsageLog(t *testing.T) {
	groupID := int64(12)
	channelID := int64(34)
	upstreamModel := "claude-sonnet-upstream"
	mappingChain := "claude-sonnet->claude-sonnet-upstream"
	inboundEndpoint := "/v1/messages"
	upstreamEndpoint := "/v1/chat/completions"

	input := usageInteractionInputFromUsageLog(&UsageLog{
		ID:                99,
		RequestID:         "req-routing",
		UserID:            101,
		APIKeyID:          202,
		AccountID:         303,
		GroupID:           &groupID,
		ChannelID:         &channelID,
		Model:             "claude-sonnet-mapped",
		RequestedModel:    "claude-sonnet",
		UpstreamModel:     &upstreamModel,
		ModelMappingChain: &mappingChain,
		InboundEndpoint:   &inboundEndpoint,
		UpstreamEndpoint:  &upstreamEndpoint,
	}, &UsageInteractionCapture{
		RoutingContext: map[string]any{"capture": "normal"},
	})

	got := input.RoutingContext
	if got["capture"] != "normal" {
		t.Fatalf("capture routing context was not preserved: %#v", got)
	}
	if got["inbound_endpoint"] != inboundEndpoint || got["upstream_endpoint"] != upstreamEndpoint {
		t.Fatalf("routing endpoints = %#v", got)
	}
	if got["requested_model"] != "claude-sonnet" || got["mapped_model"] != "claude-sonnet-mapped" || got["upstream_model"] != upstreamModel {
		t.Fatalf("routing models = %#v", got)
	}
	if got["account_id"] != int64(303) || got["api_key_id"] != int64(202) || got["user_id"] != int64(101) {
		t.Fatalf("routing identifiers = %#v", got)
	}
	if got["group_id"] != groupID || got["channel_id"] != channelID || got["model_mapping_chain"] != mappingChain {
		t.Fatalf("routing channel context = %#v", got)
	}
}

func containsAll(s string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(s, value) {
			return false
		}
	}
	return true
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestUsageInteractionSettings_ParseDefaultsAndExplicitValues(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{})

	defaults := svc.parseSettings(map[string]string{
		SettingKeyDefaultConcurrency: "1",
		SettingKeyDefaultBalance:     "0",
	})
	if defaults.UsageInteractionRecordingEnabled {
		t.Fatal("recording should default off")
	}
	if defaults.UsageInteractionStoreRawEnabled {
		t.Fatal("raw JSON storage should default off")
	}
	if defaults.UsageInteractionRetentionDays != 7 {
		t.Fatalf("retention days default = %d, want 7", defaults.UsageInteractionRetentionDays)
	}

	explicit := svc.parseSettings(map[string]string{
		SettingKeyDefaultConcurrency:               "1",
		SettingKeyDefaultBalance:                   "0",
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "true",
		SettingKeyUsageInteractionRetentionDays:    "0",
	})
	if !explicit.UsageInteractionRecordingEnabled {
		t.Fatal("recording should parse as enabled")
	}
	if !explicit.UsageInteractionStoreRawEnabled {
		t.Fatal("raw JSON storage should parse as enabled")
	}
	if explicit.UsageInteractionRetentionDays != 0 {
		t.Fatalf("retention days = %d, want 0 for indefinite", explicit.UsageInteractionRetentionDays)
	}
}

func TestUsageInteractionSettings_PersistAndRejectNegativeRetention(t *testing.T) {
	repo := &usageInteractionSettingRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		UsageInteractionRecordingEnabled: true,
		UsageInteractionStoreRawEnabled:  true,
		UsageInteractionRetentionDays:    0,
	})
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	if got := repo.updates[SettingKeyUsageInteractionRecordingEnabled]; got != "true" {
		t.Fatalf("recording update = %q, want true", got)
	}
	if got := repo.updates[SettingKeyUsageInteractionStoreRawEnabled]; got != "true" {
		t.Fatalf("store raw update = %q, want true", got)
	}
	if got := repo.updates[SettingKeyUsageInteractionRetentionDays]; got != "0" {
		t.Fatalf("retention update = %q, want 0", got)
	}

	err = svc.UpdateSettings(context.Background(), &SystemSettings{
		UsageInteractionRetentionDays: -1,
	})
	if err == nil {
		t.Fatal("expected negative retention to be rejected")
	}
}

type usageInteractionSettingRepoStub struct {
	updates map[string]string
}

func (s *usageInteractionSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *usageInteractionSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *usageInteractionSettingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *usageInteractionSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *usageInteractionSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for key, value := range settings {
		s.updates[key] = value
	}
	return nil
}

func (s *usageInteractionSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *usageInteractionSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}
