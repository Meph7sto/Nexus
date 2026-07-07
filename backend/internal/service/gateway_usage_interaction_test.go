package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/nexus/internal/config"
)

type usageInteractionUsageLogRepoStub struct {
	UsageLogRepository

	nextID   int64
	calls    int
	lastLog  *UsageLog
	deleted  bool
	deleteID int64
}

func (s *usageInteractionUsageLogRepoStub) Create(_ context.Context, log *UsageLog) (bool, error) {
	s.calls++
	s.lastLog = log
	if log.ID == 0 {
		if s.nextID == 0 {
			s.nextID = 1
		}
		log.ID = s.nextID
		s.nextID++
	}
	return true, nil
}

func (s *usageInteractionUsageLogRepoStub) Delete(_ context.Context, id int64) error {
	s.deleted = true
	s.deleteID = id
	return nil
}

func (s *usageInteractionUsageLogRepoStub) CreateWithUsageInteraction(ctx context.Context, log *UsageLog, interactionService *UsageInteractionService, capture *UsageInteractionCapture, captureErr error) (bool, error) {
	inserted, err := s.Create(ctx, log)
	if err != nil {
		return false, err
	}
	if log.ID > 0 && interactionService != nil {
		if err := interactionService.RecordForUsageLogWithFallback(ctx, log, capture, captureErr, "service.gateway.test"); err != nil {
			return false, err
		}
	}
	return inserted, nil
}

type usageInteractionBestEffortUsageLogRepoStub struct {
	usageInteractionUsageLogRepoStub
	bestEffortCalls int
}

func (s *usageInteractionBestEffortUsageLogRepoStub) CreateBestEffort(_ context.Context, _ *UsageLog) error {
	s.bestEffortCalls++
	return nil
}

func newGatewayUsageInteractionServiceForTest(usageRepo UsageLogRepository, interactionService *UsageInteractionService) *GatewayService {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1.1
	svc := NewGatewayService(
		nil,
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
		nil,
		&DeferredService{},
		nil,
		nil,
		nil,
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

func TestGatewayRecordUsage_RecordingEnabledBypassesBestEffortForPersistedID(t *testing.T) {
	usageRepo := &usageInteractionBestEffortUsageLogRepoStub{
		usageInteractionUsageLogRepoStub: usageInteractionUsageLogRepoStub{nextID: 301},
	}
	spy := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := newGatewayUsageInteractionServiceForTest(usageRepo, NewUsageInteractionService(spy, settings))

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway-best-effort-interaction-req",
			Usage: ClaudeUsage{
				InputTokens:  10,
				OutputTokens: 6,
			},
			Model:    "claude-sonnet-4",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 502, Quota: 100},
		User:    &User{ID: 602},
		Account: &Account{ID: 702},
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
	if !spy.created || spy.last.UsageLogID != 301 {
		t.Fatalf("interaction must be linked to persisted usage log id 301, got created=%v input=%#v", spy.created, spy.last)
	}
}

func TestGatewayRecordUsage_CreatesInteractionForUsageLog(t *testing.T) {
	usageRepo := &usageInteractionUsageLogRepoStub{nextID: 101}
	spy := &usageInteractionRepoSpy{}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := newGatewayUsageInteractionServiceForTest(usageRepo, NewUsageInteractionService(spy, settings))

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway-interaction-req",
			Usage: ClaudeUsage{
				InputTokens:  10,
				OutputTokens: 6,
			},
			Model:    "claude-sonnet-4",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 501, Quota: 100},
		User:    &User{ID: 601},
		Account: &Account{ID: 701},
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

func TestGatewayRecordUsage_ReturnsErrorWhenInteractionCreateFails(t *testing.T) {
	usageRepo := &usageInteractionUsageLogRepoStub{nextID: 501}
	spy := &usageInteractionRepoSpy{createErr: errors.New("interaction insert failed")}
	settings := &usageInteractionServiceSettingRepoStub{values: map[string]string{
		SettingKeyUsageInteractionRecordingEnabled: "true",
		SettingKeyUsageInteractionStoreRawEnabled:  "false",
		SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	svc := newGatewayUsageInteractionServiceForTest(usageRepo, NewUsageInteractionService(spy, settings))

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway-interaction-fail-req",
			Usage: ClaudeUsage{
				InputTokens:  10,
				OutputTokens: 6,
			},
			Model:    "claude-sonnet-4",
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 503, Quota: 100},
		User:    &User{ID: 603},
		Account: &Account{ID: 703},
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
