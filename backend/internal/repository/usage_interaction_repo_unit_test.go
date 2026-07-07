package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/nexus/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageInteractionRepositoryCreatePersistsFullPayloads(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewUsageInteractionRepository(db)

	groupID := int64(6)
	captureError := "stream ended early"
	createdAt := time.Date(2026, 7, 6, 23, 0, 0, 0, time.UTC)
	longPrompt := strings.Repeat("preserve ", 200)
	input := service.UsageInteractionInput{
		UsageLogID:        11,
		RequestID:         "req-123",
		UserID:            22,
		APIKeyID:          33,
		AccountID:         44,
		GroupID:           &groupID,
		CaptureStatus:     service.UsageInteractionCapturePartial,
		CaptureError:      &captureError,
		RequestContent:    map[string]any{"prompt": longPrompt},
		ResponseContent:   map[string]any{"output": strings.Repeat("answer ", 200)},
		RequestParameters: map[string]any{"model": "gpt-5"},
		RoutingContext:    map[string]any{"platform": "openai"},
		RawRequestJSON:    map[string]any{"Authorization": "[REDACTED]"},
		RawResponseJSON:   map[string]any{"choices": []any{map[string]any{"text": "done"}}},
		CreatedAt:         createdAt,
	}

	mock.ExpectExec("INSERT INTO usage_interactions").
		WithArgs(
			input.UsageLogID,
			input.RequestID,
			input.UserID,
			input.APIKeyID,
			input.AccountID,
			input.GroupID,
			input.CaptureStatus,
			input.CaptureError,
			jsonTextContaining(t, longPrompt),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			true,
			jsonTextContaining(t, "Authorization"),
			input.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), input, true, []string{"Authorization"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageInteractionRepositoryGetByUsageLogIDHidesRawUnlessRequested(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewUsageInteractionRepository(db)

	createdAt := time.Date(2026, 7, 6, 23, 1, 0, 0, time.UTC)
	mock.ExpectQuery("NULL::jsonb AS raw_request_json").
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "usage_log_id", "request_id", "user_id", "api_key_id", "account_id", "group_id",
			"capture_status", "capture_error", "request_content", "response_content", "request_parameters",
			"routing_context", "raw_request_json", "raw_response_json", "raw_available", "redaction_applied",
			"redaction_keys", "created_at",
		}).AddRow(
			int64(1), int64(11), "req-123", int64(22), int64(33), int64(44), nil,
			service.UsageInteractionCaptureComplete, nil,
			[]byte(`{"prompt":"keep"}`), []byte(`{"output":"done"}`), []byte(`{"model":"gpt-5"}`),
			[]byte(`{"route":"primary"}`), nil, nil, true, true, []byte(`["Authorization"]`), createdAt,
		))

	got, err := repo.GetByUsageLogID(context.Background(), 11, false)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, map[string]any{"prompt": "keep"}, got.RequestContent)
	require.True(t, got.RawAvailable)
	require.Nil(t, got.RawRequestJSON)
	require.Nil(t, got.RawResponseJSON)
	require.Equal(t, []string{"Authorization"}, got.RedactionKeys)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageInteractionRepositoryDeleteOlderThan(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := NewUsageInteractionRepository(db)
	cutoff := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectExec("DELETE FROM usage_interactions WHERE created_at < \\$1").
		WithArgs(cutoff).
		WillReturnResult(sqlmock.NewResult(0, 3))

	deleted, err := repo.DeleteOlderThan(context.Background(), cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(3), deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

type jsonTextContainingMatcher struct {
	t      *testing.T
	needle string
}

func jsonTextContaining(t *testing.T, needle string) sqlmock.Argument {
	t.Helper()
	return jsonTextContainingMatcher{t: t, needle: needle}
}

func (m jsonTextContainingMatcher) Match(v driver.Value) bool {
	text, ok := v.(string)
	if !ok {
		bytes, bytesOK := v.([]byte)
		if !bytesOK {
			return false
		}
		text = string(bytes)
	}
	if !strings.Contains(text, m.needle) {
		m.t.Errorf("JSON argument %q does not contain %q", text, m.needle)
		return false
	}
	return true
}
