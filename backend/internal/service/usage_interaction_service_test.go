package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUsageInteractionService_DefaultDisabledDoesNotRecord(t *testing.T) {
	repo := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "false",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := NewUsageInteractionService(repo, settings)

	err := svc.RecordComplete(context.Background(), UsageInteractionInput{UsageLogID: 1, RequestID: "req"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.created {
		t.Fatal("disabled recording must not create interaction rows")
	}
}

func TestUsageInteractionService_EnabledRecordsFailedPlaceholder(t *testing.T) {
	repo := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := NewUsageInteractionService(repo, settings)

	err := svc.RecordFailed(context.Background(), UsageInteractionInput{UsageLogID: 7, RequestID: "req-7"}, "capture exploded")
	if err != nil {
		t.Fatal(err)
	}
	if !repo.created || repo.last.CaptureStatus != UsageInteractionCaptureFailed || repo.last.CaptureError == nil {
		t.Fatalf("expected failed placeholder, got %#v", repo.last)
	}
}

func TestUsageInteractionService_StoreRawEnabledRedactsRawCredentialFields(t *testing.T) {
	repo := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "true",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := NewUsageInteractionService(repo, settings)

	err := svc.RecordComplete(context.Background(), UsageInteractionInput{
		UsageLogID:      11,
		RequestID:       "req-raw",
		RequestContent:  map[string]any{"prompt": "keep me"},
		ResponseContent: map[string]any{"output": "visible"},
		RawRequestJSON: map[string]any{
			"Authorization": "Bearer raw-secret",
			"messages":      []any{map[string]any{"content": "raw prompt"}},
		},
		RawResponseJSON: map[string]any{
			"result": map[string]any{
				"token": "raw-token",
				"text":  "raw response",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.created {
		t.Fatal("expected interaction row to be created")
	}
	if got := repo.last.RawRequestJSON["Authorization"]; got != "[REDACTED]" {
		t.Fatalf("raw request Authorization = %#v, want redacted", got)
	}
	result, ok := repo.last.RawResponseJSON["result"].(map[string]any)
	if !ok {
		t.Fatalf("raw response result = %#v, want map", repo.last.RawResponseJSON["result"])
	}
	if got := result["token"]; got != "[REDACTED]" {
		t.Fatalf("raw response token = %#v, want redacted", got)
	}
	if !repo.redactionApplied {
		t.Fatal("expected raw credential redaction to mark redaction_applied")
	}
	if !usageInteractionTestContains(repo.redactionKeys, "Authorization") || !usageInteractionTestContains(repo.redactionKeys, "token") {
		t.Fatalf("redaction keys = %v, want raw credential keys", repo.redactionKeys)
	}
}

func TestUsageInteractionService_SettingsReadErrorsPropagate(t *testing.T) {
	settingsErr := errors.New("settings read failed")
	tests := []struct {
		name string
		call func(*UsageInteractionService) error
	}{
		{
			name: "record complete",
			call: func(svc *UsageInteractionService) error {
				return svc.RecordComplete(context.Background(), UsageInteractionInput{UsageLogID: 1, RequestID: "req"})
			},
		},
		{
			name: "record failed",
			call: func(svc *UsageInteractionService) error {
				return svc.RecordFailed(context.Background(), UsageInteractionInput{UsageLogID: 2, RequestID: "req"}, "failed")
			},
		},
		{
			name: "cleanup expired",
			call: func(svc *UsageInteractionService) error {
				_, err := svc.CleanupExpired(context.Background(), time.Now())
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &usageInteractionRepoSpy{}
			settings := &usageInteractionServiceSettingRepoStub{err: settingsErr}
			svc := NewUsageInteractionService(repo, settings)

			err := tt.call(svc)
			if !errors.Is(err, settingsErr) {
				t.Fatalf("error = %v, want settings read failure", err)
			}
			if repo.created {
				t.Fatal("settings read failure must not create interaction rows")
			}
			if repo.deleted {
				t.Fatal("settings read failure must not delete interaction rows")
			}
		})
	}
}

type usageInteractionRepoSpy struct {
	created          bool
	last             UsageInteractionInput
	redactionApplied bool
	redactionKeys    []string
	deleted          bool
	deleteCutoff     time.Time
	deleteResult     int64
	deleteErr        error
}

func (r *usageInteractionRepoSpy) Create(_ context.Context, input UsageInteractionInput, redactionApplied bool, redactionKeys []string) error {
	r.created = true
	r.last = input
	r.redactionApplied = redactionApplied
	r.redactionKeys = append([]string(nil), redactionKeys...)
	return nil
}
func (r *usageInteractionRepoSpy) GetByUsageLogID(context.Context, int64, bool) (*UsageInteraction, error) {
	return nil, ErrUsageInteractionNotFound
}
func (r *usageInteractionRepoSpy) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	r.deleted = true
	r.deleteCutoff = cutoff
	return r.deleteResult, r.deleteErr
}

type usageInteractionServiceSettingRepoStub struct {
	values map[string]string
	err    error
}

func (s *usageInteractionServiceSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, nil
}
func (s *usageInteractionServiceSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}
func (s *usageInteractionServiceSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}
func (s *usageInteractionServiceSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}
func (s *usageInteractionServiceSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (s *usageInteractionServiceSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *usageInteractionServiceSettingRepoStub) Delete(context.Context, string) error { return nil }

func usageInteractionTestContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
