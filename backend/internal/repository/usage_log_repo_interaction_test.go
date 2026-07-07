package repository

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/nexus/ent"
	"github.com/Wei-Shaw/nexus/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryCreateWithUsageInteractionSkipsInteractionOnIdempotentConflict(t *testing.T) {
	db, mock := newSQLMock(t)
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(drv))
	t.Cleanup(func() { _ = client.Close() })
	repo := NewUsageLogRepository(client, db).(interface {
		CreateWithUsageInteraction(context.Context, *service.UsageLog, *service.UsageInteractionService, *service.UsageInteractionCapture, error) (bool, error)
	})

	createdAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	log := &service.UsageLog{
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		RequestID:      "req-idempotent",
		Model:          "gpt-5",
		RequestedModel: "gpt-5",
		CreatedAt:      createdAt,
	}
	interactionRepo := &serviceUsageInteractionRepoSpy{}
	settings := &serviceUsageInteractionSettingRepoStub{values: map[string]string{
		service.SettingKeyUsageInteractionRecordingEnabled: "true",
		service.SettingKeyUsageInteractionStoreRawEnabled:  "false",
		service.SettingKeyUsageInteractionRetentionDays:    "7",
	}}
	interactionSvc := service.NewUsageInteractionService(interactionRepo, settings)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO usage_logs").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}))
	mock.ExpectQuery("SELECT id, created_at FROM usage_logs WHERE request_id = \\$1 AND api_key_id = \\$2").
		WithArgs("req-idempotent", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(55), createdAt))
	mock.ExpectCommit()

	inserted, err := repo.CreateWithUsageInteraction(context.Background(), log, interactionSvc, &service.UsageInteractionCapture{
		RequestContent: map[string]any{"prompt": "duplicate prompt"},
	}, nil)
	require.NoError(t, err)
	require.False(t, inserted)
	require.Equal(t, int64(55), log.ID)
	require.False(t, interactionRepo.created, "duplicate usage log writes must not mutate the original interaction row")
	require.NoError(t, mock.ExpectationsWereMet())
}

type serviceUsageInteractionRepoSpy struct {
	created bool
}

func (r *serviceUsageInteractionRepoSpy) Create(context.Context, service.UsageInteractionInput, bool, []string) error {
	r.created = true
	return nil
}

func (r *serviceUsageInteractionRepoSpy) GetByUsageLogID(context.Context, int64, bool) (*service.UsageInteraction, error) {
	return nil, service.ErrUsageInteractionNotFound
}

func (r *serviceUsageInteractionRepoSpy) DeleteOlderThan(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type serviceUsageInteractionSettingRepoStub struct {
	values map[string]string
}

func (s *serviceUsageInteractionSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (s *serviceUsageInteractionSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *serviceUsageInteractionSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}

func (s *serviceUsageInteractionSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = s.values[key]
	}
	return out, nil
}

func (s *serviceUsageInteractionSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (s *serviceUsageInteractionSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *serviceUsageInteractionSettingRepoStub) Delete(context.Context, string) error {
	return nil
}
