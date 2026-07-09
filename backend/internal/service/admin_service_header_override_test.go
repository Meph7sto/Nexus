//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type headerOverrideAccountRepoStub struct {
	accountRepoStub
	nextID   int64
	accounts map[int64]*Account
}

func newHeaderOverrideAccountRepoStub() *headerOverrideAccountRepoStub {
	return &headerOverrideAccountRepoStub{
		accounts: make(map[int64]*Account),
	}
}

func (s *headerOverrideAccountRepoStub) Create(_ context.Context, account *Account) error {
	s.nextID++
	account.ID = s.nextID
	cp := *account
	s.accounts[account.ID] = &cp
	return nil
}

func (s *headerOverrideAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	account, ok := s.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (s *headerOverrideAccountRepoStub) Update(_ context.Context, account *Account) error {
	if _, ok := s.accounts[account.ID]; !ok {
		return ErrAccountNotFound
	}
	cp := *account
	s.accounts[account.ID] = &cp
	return nil
}

func (s *headerOverrideAccountRepoStub) ListShadowsByParent(_ context.Context, _ int64) ([]*Account, error) {
	return nil, nil
}

func TestAdminServiceCreateAccountStripsHeaderOverridesForIneligibleType(t *testing.T) {
	ctx := context.Background()
	repo := newHeaderOverrideAccountRepoStub()
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(ctx, &CreateAccountInput{
		Name:     "anthropic-setup-token",
		Platform: PlatformAnthropic,
		Type:     AccountTypeSetupToken,
		Credentials: map[string]any{
			"access_token":               "token",
			credKeyHeaderOverrideEnabled: true,
			credKeyHeaderOverrides:       map[string]any{"user-agent": "cli"},
		},
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.Equal(t, "token", account.Credentials["access_token"])
	require.NotContains(t, account.Credentials, credKeyHeaderOverrideEnabled)
	require.NotContains(t, account.Credentials, credKeyHeaderOverrides)
	require.NotContains(t, repo.accounts[account.ID].Credentials, credKeyHeaderOverrideEnabled)
	require.NotContains(t, repo.accounts[account.ID].Credentials, credKeyHeaderOverrides)
}

func TestAdminServiceUpdateAccountStripsHeaderOverridesWhenTypeBecomesIneligible(t *testing.T) {
	ctx := context.Background()
	repo := newHeaderOverrideAccountRepoStub()
	svc := &adminServiceImpl{accountRepo: repo}

	account := &Account{
		Name:        "openai-api-key",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Credentials: map[string]any{"api_key": "sk-existing"},
	}
	require.NoError(t, repo.Create(ctx, account))

	updated, err := svc.UpdateAccount(ctx, account.ID, &UpdateAccountInput{
		Type: AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":               "oauth-token",
			credKeyHeaderOverrideEnabled: true,
			credKeyHeaderOverrides:       map[string]any{"user-agent": "cli"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, AccountTypeOAuth, updated.Type)
	require.Equal(t, "oauth-token", updated.Credentials["access_token"])
	require.NotContains(t, updated.Credentials, credKeyHeaderOverrideEnabled)
	require.NotContains(t, updated.Credentials, credKeyHeaderOverrides)
	require.NotContains(t, repo.accounts[account.ID].Credentials, credKeyHeaderOverrideEnabled)
	require.NotContains(t, repo.accounts[account.ID].Credentials, credKeyHeaderOverrides)
}
