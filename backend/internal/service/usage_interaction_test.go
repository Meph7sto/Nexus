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
	if containsAll(text, "Bearer secret") || containsAll(text, "tok") {
		t.Fatalf("secret leaked in payload: %s", text)
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
