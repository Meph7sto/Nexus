package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/nexus/ent"
	"github.com/Wei-Shaw/nexus/internal/service"
)

type usageInteractionRepository struct {
	db *sql.DB
}

func NewUsageInteractionRepository(db *sql.DB) service.UsageInteractionRepository {
	return &usageInteractionRepository{db: db}
}

func (r *usageInteractionRepository) Create(ctx context.Context, input service.UsageInteractionInput, redactionApplied bool, redactionKeys []string) error {
	if r == nil || r.db == nil {
		return errors.New("usage interaction repository db is nil")
	}
	sqlq := sqlExecutor(r.db)
	if tx := dbent.TxFromContext(ctx); tx != nil {
		sqlq = tx.Client()
	}
	return createUsageInteraction(ctx, sqlq, input, redactionApplied, redactionKeys)
}

func createUsageInteraction(ctx context.Context, sqlq sqlExecutor, input service.UsageInteractionInput, redactionApplied bool, redactionKeys []string) error {
	if sqlq == nil {
		return errors.New("usage interaction sql executor is nil")
	}
	status := input.CaptureStatus
	if status == "" {
		status = service.UsageInteractionCaptureComplete
	}
	requestContent, err := marshalJSONMap(input.RequestContent)
	if err != nil {
		return fmt.Errorf("marshal request content: %w", err)
	}
	responseContent, err := marshalJSONMap(input.ResponseContent)
	if err != nil {
		return fmt.Errorf("marshal response content: %w", err)
	}
	requestParameters, err := marshalJSONMap(input.RequestParameters)
	if err != nil {
		return fmt.Errorf("marshal request parameters: %w", err)
	}
	routingContext, err := marshalJSONMap(input.RoutingContext)
	if err != nil {
		return fmt.Errorf("marshal routing context: %w", err)
	}
	rawRequestJSON, err := marshalNullableJSONMap(input.RawRequestJSON)
	if err != nil {
		return fmt.Errorf("marshal raw request json: %w", err)
	}
	rawResponseJSON, err := marshalNullableJSONMap(input.RawResponseJSON)
	if err != nil {
		return fmt.Errorf("marshal raw response json: %w", err)
	}
	if redactionKeys == nil {
		redactionKeys = []string{}
	}
	redactionKeysJSON, err := json.Marshal(redactionKeys)
	if err != nil {
		return fmt.Errorf("marshal redaction keys: %w", err)
	}
	var createdAt any
	if !input.CreatedAt.IsZero() {
		createdAt = input.CreatedAt
	}

	_, err = sqlq.ExecContext(ctx, `
		INSERT INTO usage_interactions (
			usage_log_id, request_id, user_id, api_key_id, account_id, group_id,
			capture_status, capture_error, request_content, response_content,
			request_parameters, routing_context, raw_request_json, raw_response_json,
			redaction_applied, redaction_keys, created_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9::jsonb, $10::jsonb,
			$11::jsonb, $12::jsonb, $13::jsonb, $14::jsonb,
			$15, $16::jsonb, COALESCE($17, NOW())
		)
		ON CONFLICT (usage_log_id) DO UPDATE SET
			request_id = EXCLUDED.request_id,
			user_id = EXCLUDED.user_id,
			api_key_id = EXCLUDED.api_key_id,
			account_id = EXCLUDED.account_id,
			group_id = EXCLUDED.group_id,
			capture_status = EXCLUDED.capture_status,
			capture_error = EXCLUDED.capture_error,
			request_content = EXCLUDED.request_content,
			response_content = EXCLUDED.response_content,
			request_parameters = EXCLUDED.request_parameters,
			routing_context = EXCLUDED.routing_context,
			raw_request_json = EXCLUDED.raw_request_json,
			raw_response_json = EXCLUDED.raw_response_json,
			redaction_applied = EXCLUDED.redaction_applied,
			redaction_keys = EXCLUDED.redaction_keys
	`, input.UsageLogID, input.RequestID, input.UserID, input.APIKeyID, input.AccountID, input.GroupID,
		status, input.CaptureError, requestContent, responseContent, requestParameters, routingContext,
		rawRequestJSON, rawResponseJSON, redactionApplied, string(redactionKeysJSON), createdAt)
	if err != nil {
		return fmt.Errorf("create usage interaction: %w", err)
	}
	return nil
}

func (r *usageInteractionRepository) GetByUsageLogID(ctx context.Context, usageLogID int64, includeRaw bool) (*service.UsageInteraction, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("usage interaction repository db is nil")
	}
	rawRequestSelect := "NULL::jsonb AS raw_request_json"
	rawResponseSelect := "NULL::jsonb AS raw_response_json"
	if includeRaw {
		rawRequestSelect = "raw_request_json"
		rawResponseSelect = "raw_response_json"
	}
	query := fmt.Sprintf(`
		SELECT
			id, usage_log_id, request_id, user_id, api_key_id, account_id, group_id,
			capture_status, capture_error, request_content, response_content,
			request_parameters, routing_context, %s, %s,
			(raw_request_json IS NOT NULL OR raw_response_json IS NOT NULL) AS raw_available,
			redaction_applied, redaction_keys, created_at
		FROM usage_interactions
		WHERE usage_log_id = $1
	`, rawRequestSelect, rawResponseSelect)

	var interaction service.UsageInteraction
	var groupID sql.NullInt64
	var captureError sql.NullString
	var requestContent, responseContent, requestParameters, routingContext []byte
	var rawRequestJSON, rawResponseJSON []byte
	var redactionKeys []byte
	err := r.db.QueryRowContext(ctx, query, usageLogID).Scan(
		&interaction.ID,
		&interaction.UsageLogID,
		&interaction.RequestID,
		&interaction.UserID,
		&interaction.APIKeyID,
		&interaction.AccountID,
		&groupID,
		&interaction.CaptureStatus,
		&captureError,
		&requestContent,
		&responseContent,
		&requestParameters,
		&routingContext,
		&rawRequestJSON,
		&rawResponseJSON,
		&interaction.RawAvailable,
		&interaction.RedactionApplied,
		&redactionKeys,
		&interaction.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get usage interaction: %w", err)
	}
	if groupID.Valid {
		interaction.GroupID = &groupID.Int64
	}
	if captureError.Valid {
		interaction.CaptureError = &captureError.String
	}
	if interaction.RequestContent, err = unmarshalJSONMap(requestContent); err != nil {
		return nil, fmt.Errorf("unmarshal request content: %w", err)
	}
	if interaction.ResponseContent, err = unmarshalJSONMap(responseContent); err != nil {
		return nil, fmt.Errorf("unmarshal response content: %w", err)
	}
	if interaction.RequestParameters, err = unmarshalJSONMap(requestParameters); err != nil {
		return nil, fmt.Errorf("unmarshal request parameters: %w", err)
	}
	if interaction.RoutingContext, err = unmarshalJSONMap(routingContext); err != nil {
		return nil, fmt.Errorf("unmarshal routing context: %w", err)
	}
	if includeRaw {
		if interaction.RawRequestJSON, err = unmarshalNullableJSONMap(rawRequestJSON); err != nil {
			return nil, fmt.Errorf("unmarshal raw request json: %w", err)
		}
		if interaction.RawResponseJSON, err = unmarshalNullableJSONMap(rawResponseJSON); err != nil {
			return nil, fmt.Errorf("unmarshal raw response json: %w", err)
		}
	}
	if len(redactionKeys) > 0 {
		if err := json.Unmarshal(redactionKeys, &interaction.RedactionKeys); err != nil {
			return nil, fmt.Errorf("unmarshal redaction keys: %w", err)
		}
	}
	if interaction.RedactionKeys == nil {
		interaction.RedactionKeys = []string{}
	}
	return &interaction, nil
}

func (r *usageInteractionRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("usage interaction repository db is nil")
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM usage_interactions WHERE created_at < $1", cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old usage interactions: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("usage interaction rows affected: %w", err)
	}
	return rows, nil
}

func marshalJSONMap(value map[string]any) (string, error) {
	if value == nil {
		value = map[string]any{}
	}
	blob, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

func marshalNullableJSONMap(value map[string]any) (any, error) {
	if value == nil {
		return nil, nil
	}
	blob, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return string(blob), nil
}

func unmarshalJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

func unmarshalNullableJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	return unmarshalJSONMap(raw)
}
