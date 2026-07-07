package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/config"
)

func newOpenAIUsageInteractionServiceForTest(usageRepo UsageLogRepository, interactionService *UsageInteractionService) *OpenAIGatewayService {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1.1
	svc := NewOpenAIGatewayService(
		nil,
		usageRepo,
		nil,
		&openAIRecordUsageUserRepoStub{},
		&openAIRecordUsageSubRepoStub{},
		nil,
		nil,
		cfg,
		nil,
		nil,
		NewBillingService(cfg, nil),
		nil,
		&BillingCacheService{},
		nil,
		&DeferredService{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		interactionService,
	)
	return svc
}

func TestOpenAIRecordUsage_CreatesInteractionForUsageLog(t *testing.T) {
	usageRepo := &usageInteractionUsageLogRepoStub{nextID: 201}
	spy := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := newOpenAIUsageInteractionServiceForTest(usageRepo, NewUsageInteractionService(spy, settings))

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID: "openai-interaction-req",
			Usage: OpenAIUsage{
				InputTokens:  12,
				OutputTokens: 4,
			},
			Model:    "gpt-5.1",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 1001, Quota: 100, Group: &Group{RateMultiplier: 1}},
		User:    &User{ID: 2001},
		Account: &Account{ID: 3001, Type: AccountTypeAPIKey},
		Interaction: &UsageInteractionCapture{
			RequestContent:  map[string]any{"prompt": "full prompt"},
			ResponseContent: map[string]any{"output": "full response"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !spy.created {
		t.Fatal("expected usage interaction to be recorded")
	}
	if spy.last.UsageLogID <= 0 {
		t.Fatalf("interaction must be linked to persisted usage log id, got %d", spy.last.UsageLogID)
	}
	if spy.last.RequestID != usageRepo.lastLog.RequestID {
		t.Fatalf("interaction request_id = %q, want %q", spy.last.RequestID, usageRepo.lastLog.RequestID)
	}
}

func TestOpenAIRecordUsage_RecordingEnabledBypassesBestEffortForPersistedID(t *testing.T) {
	usageRepo := &usageInteractionBestEffortUsageLogRepoStub{
		usageInteractionUsageLogRepoStub: usageInteractionUsageLogRepoStub{nextID: 401},
	}
	spy := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := newOpenAIUsageInteractionServiceForTest(usageRepo, NewUsageInteractionService(spy, settings))

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID: "openai-best-effort-interaction-req",
			Usage: OpenAIUsage{
				InputTokens:  12,
				OutputTokens: 4,
			},
			Model:    "gpt-5.1",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 1002, Quota: 100, Group: &Group{RateMultiplier: 1}},
		User:    &User{ID: 2002},
		Account: &Account{ID: 3002, Type: AccountTypeAPIKey},
		Interaction: &UsageInteractionCapture{
			RequestContent:  map[string]any{"prompt": "full prompt"},
			ResponseContent: map[string]any{"output": "full response"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if usageRepo.bestEffortCalls != 0 {
		t.Fatalf("recording enabled must use Create for a persisted id, best-effort calls=%d", usageRepo.bestEffortCalls)
	}
	if usageRepo.calls != 1 {
		t.Fatalf("expected one synchronous usage log create, got %d", usageRepo.calls)
	}
	if !spy.created || spy.last.UsageLogID != 401 {
		t.Fatalf("interaction must be linked to persisted usage log id 401, got created=%v input=%#v", spy.created, spy.last)
	}
}

func TestOpenAIRecordUsage_ReturnsErrorWhenInteractionCreateFails(t *testing.T) {
	usageRepo := &usageInteractionUsageLogRepoStub{nextID: 601}
	spy := &usageInteractionRepoSpy{createErr: errors.New("interaction insert failed")}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := newOpenAIUsageInteractionServiceForTest(usageRepo, NewUsageInteractionService(spy, settings))

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID: "openai-interaction-fail-req",
			Usage: OpenAIUsage{
				InputTokens:  12,
				OutputTokens: 4,
			},
			Model:    "gpt-5.1",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 1003, Quota: 100, Group: &Group{RateMultiplier: 1}},
		User:    &User{ID: 2003},
		Account: &Account{ID: 3003, Type: AccountTypeAPIKey},
		Interaction: &UsageInteractionCapture{
			RequestContent:  map[string]any{"prompt": "full prompt"},
			ResponseContent: map[string]any{"output": "full response"},
		},
	})
	if err == nil {
		t.Fatal("expected interaction persistence error")
	}
	if usageRepo.deleted {
		t.Fatalf("must not delete billed usage log outside an atomic repository transaction, deleted id=%d", usageRepo.deleteID)
	}
}

func TestOpenAIRecordUsage_SettingsReadFailureWritesUsageWithoutInteraction(t *testing.T) {
	usageRepo := &usageInteractionUsageLogRepoStub{nextID: 801}
	spy := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{err: errors.New("settings unavailable")}
	svc := newOpenAIUsageInteractionServiceForTest(usageRepo, NewUsageInteractionService(spy, settings))

	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID: "openai-interaction-settings-fail-req",
			Usage: OpenAIUsage{
				InputTokens:  12,
				OutputTokens: 4,
			},
			Model:    "gpt-5.1",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 1004, Quota: 100, Group: &Group{RateMultiplier: 1}},
		User:    &User{ID: 2004},
		Account: &Account{ID: 3004, Type: AccountTypeAPIKey},
		Interaction: &UsageInteractionCapture{
			RequestContent:  map[string]any{"prompt": "full prompt"},
			ResponseContent: map[string]any{"output": "full response"},
		},
	})
	if err != nil {
		t.Fatalf("settings read failure must not veto usage log persistence: %v", err)
	}
	if usageRepo.calls != 1 || usageRepo.lastLog == nil || usageRepo.lastLog.ID != 801 {
		t.Fatalf("usage log was not persisted after settings failure: calls=%d log=%#v", usageRepo.calls, usageRepo.lastLog)
	}
	if spy.created {
		t.Fatalf("settings failure must disable optional interaction recording, got %#v", spy.last)
	}
}
