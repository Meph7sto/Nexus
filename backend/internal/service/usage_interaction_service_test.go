package service

import (
	"context"
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

type usageInteractionRepoSpy struct {
	created bool
	last    UsageInteractionInput
}

func (r *usageInteractionRepoSpy) Create(_ context.Context, input UsageInteractionInput, _ bool, _ []string) error {
	r.created = true
	r.last = input
	return nil
}
func (r *usageInteractionRepoSpy) GetByUsageLogID(context.Context, int64, bool) (*UsageInteraction, error) {
	return nil, ErrUsageInteractionNotFound
}
func (r *usageInteractionRepoSpy) DeleteOlderThan(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type usageInteractionServiceSettingRepoStub struct {
	values map[string]string
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
