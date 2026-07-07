package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/nexus/internal/pkg/errors"
)

var ErrUsageInteractionNotFound = infraerrors.NotFound("USAGE_INTERACTION_NOT_FOUND", "usage interaction not found")

type UsageInteractionService struct {
	repo        UsageInteractionRepository
	settingRepo SettingRepository
}

func NewUsageInteractionService(repo UsageInteractionRepository, settingRepo SettingRepository) *UsageInteractionService {
	return &UsageInteractionService{repo: repo, settingRepo: settingRepo}
}

func (s *UsageInteractionService) Settings(ctx context.Context) UsageInteractionSettings {
	settings := UsageInteractionSettings{RetentionDays: 7}
	if s == nil || s.settingRepo == nil {
		return settings
	}
	values, _ := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyUsageInteractionRecordingEnabled,
		SettingKeyUsageInteractionStoreRawEnabled,
		SettingKeyUsageInteractionRetentionDays,
	})
	settings.RecordingEnabled = values[SettingKeyUsageInteractionRecordingEnabled] == "true"
	settings.StoreRawEnabled = values[SettingKeyUsageInteractionStoreRawEnabled] == "true"
	if parsed, err := strconv.Atoi(strings.TrimSpace(values[SettingKeyUsageInteractionRetentionDays])); err == nil && parsed >= 0 {
		settings.RetentionDays = parsed
	}
	return settings
}

func (s *UsageInteractionService) RecordComplete(ctx context.Context, input UsageInteractionInput) error {
	input.CaptureStatus = UsageInteractionCaptureComplete
	return s.record(ctx, input)
}

func (s *UsageInteractionService) RecordFailed(ctx context.Context, input UsageInteractionInput, message string) error {
	input.CaptureStatus = UsageInteractionCaptureFailed
	input.CaptureError = &message
	return s.record(ctx, input)
}

func (s *UsageInteractionService) record(ctx context.Context, input UsageInteractionInput) error {
	if s == nil || s.repo == nil {
		return nil
	}
	settings := s.Settings(ctx)
	if !settings.RecordingEnabled {
		return nil
	}
	if input.RequestContent == nil {
		input.RequestContent = map[string]any{}
	}
	if input.ResponseContent == nil {
		input.ResponseContent = map[string]any{}
	}
	if input.RequestParameters == nil {
		input.RequestParameters = map[string]any{}
	}
	if input.RoutingContext == nil {
		input.RoutingContext = map[string]any{}
	}
	if !settings.StoreRawEnabled {
		input.RawRequestJSON = nil
		input.RawResponseJSON = nil
	}
	redactedRequest, requestKeys, requestChanged := RedactUsageInteractionPayload(input.RequestContent)
	redactedResponse, responseKeys, responseChanged := RedactUsageInteractionPayload(input.ResponseContent)
	redactedParams, paramKeys, paramChanged := RedactUsageInteractionPayload(input.RequestParameters)
	redactedRouting, routingKeys, routingChanged := RedactUsageInteractionPayload(input.RoutingContext)
	input.RequestContent = redactedRequest
	input.ResponseContent = redactedResponse
	input.RequestParameters = redactedParams
	input.RoutingContext = redactedRouting
	keys := append(append(append(requestKeys, responseKeys...), paramKeys...), routingKeys...)
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now()
	}
	return s.repo.Create(ctx, input, requestChanged || responseChanged || paramChanged || routingChanged, keys)
}

func (s *UsageInteractionService) GetByUsageLogID(ctx context.Context, usageLogID int64, includeRaw bool) (*UsageInteraction, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUsageInteractionNotFound
	}
	detail, err := s.repo.GetByUsageLogID(ctx, usageLogID, includeRaw)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrUsageInteractionNotFound
	}
	return detail, nil
}

func (s *UsageInteractionService) CleanupExpired(ctx context.Context, now time.Time) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	settings := s.Settings(ctx)
	if settings.RetentionDays == 0 {
		return 0, nil
	}
	return s.repo.DeleteOlderThan(ctx, now.AddDate(0, 0, -settings.RetentionDays))
}
