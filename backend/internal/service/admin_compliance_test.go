package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Wei-Shaw/nexus/internal/config"
	"github.com/stretchr/testify/require"
)

type adminComplianceRepoStub struct {
	values map[string]string
}

func (r *adminComplianceRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	if value, ok := r.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (r *adminComplianceRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *adminComplianceRepoStub) Set(ctx context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *adminComplianceRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (r *adminComplianceRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	return nil
}

func (r *adminComplianceRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (r *adminComplianceRepoStub) Delete(ctx context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestAdminComplianceStatusAlwaysNotRequired(t *testing.T) {
	// Admin compliance check is disabled — status is always not required.
	svc := NewSettingService(&adminComplianceRepoStub{}, &config.Config{})

	status, err := svc.GetAdminComplianceStatus(context.Background(), 1)
	require.NoError(t, err)
	require.False(t, status.Required)
	require.Equal(t, AdminComplianceVersion, status.Version)
	require.Equal(t, AdminComplianceAckPhraseZH, status.AckPhraseZH)
	require.Equal(t, AdminComplianceDocumentPathZH, status.DocumentPathZH)
}

func TestAdminComplianceStatusNotRequiredEvenOnOldVersion(t *testing.T) {
	// Admin compliance check is disabled — even old version acknowledgements don't trigger required.
	old, err := json.Marshal(AdminComplianceAcknowledgement{Version: "v2026.01.01"})
	require.NoError(t, err)
	svc := NewSettingService(&adminComplianceRepoStub{
		values: map[string]string{adminComplianceAcknowledgementKey(1): string(old)},
	}, &config.Config{})

	status, err := svc.GetAdminComplianceStatus(context.Background(), 1)
	require.NoError(t, err)
	require.False(t, status.Required)
}

func TestAcceptAdminComplianceRejectsWrongPhrase(t *testing.T) {
	svc := NewSettingService(&adminComplianceRepoStub{}, &config.Config{})

	_, err := svc.AcceptAdminCompliance(context.Background(), AdminComplianceAcceptInput{
		AdminUserID: 1,
		Language:    "zh",
		Phrase:      "我同意",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrAdminComplianceInvalidPhrase))
}

func TestAcceptAdminCompliancePersistsCurrentVersion(t *testing.T) {
	repo := &adminComplianceRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	status, err := svc.AcceptAdminCompliance(context.Background(), AdminComplianceAcceptInput{
		AdminUserID: 42,
		Language:    "zh-CN",
		Phrase:      AdminComplianceAckPhraseZH,
		IPAddress:   "203.0.113.10",
		UserAgent:   "test-agent",
	})
	require.NoError(t, err)
	require.False(t, status.Required)

	// Acceptance is still persisted even though the guard is disabled.
	var stored AdminComplianceAcknowledgement
	require.NoError(t, json.Unmarshal([]byte(repo.values[adminComplianceAcknowledgementKey(42)]), &stored))
	require.Equal(t, int64(42), stored.AdminUserID)
	require.Equal(t, "203.0.113.10", stored.IPAddress)
	require.Equal(t, AdminComplianceVersion, stored.Version)
	require.Equal(t, AdminComplianceDocumentPathZH, stored.DocumentZH)
}

func TestAdminComplianceStatusNotRequiredForAnyUser(t *testing.T) {
	// Admin compliance check is disabled — all users get Required=false.
	current, err := json.Marshal(AdminComplianceAcknowledgement{
		Version:     AdminComplianceVersion,
		AdminUserID: 1,
	})
	require.NoError(t, err)
	svc := NewSettingService(&adminComplianceRepoStub{
		values: map[string]string{adminComplianceAcknowledgementKey(1): string(current)},
	}, &config.Config{})

	statusForUserOne, err := svc.GetAdminComplianceStatus(context.Background(), 1)
	require.NoError(t, err)
	require.False(t, statusForUserOne.Required)

	statusForUserTwo, err := svc.GetAdminComplianceStatus(context.Background(), 2)
	require.NoError(t, err)
	require.False(t, statusForUserTwo.Required)
}
