package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/nexus/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newOpenAICompactVersionPolicyContext(t *testing.T, userAgent, inboundVersion string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	c.Request.Header.Set("User-Agent", userAgent)
	c.Request.Header.Set("Version", inboundVersion)
	return c
}

func TestBuildUpstreamRequestCompactUsesTrustedClientVersion(t *testing.T) {
	c := newOpenAICompactVersionPolicyContext(t, "codex_vscode/0.145.0-alpha.18", "999.999.999")
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{
		OpenAICompactFallbackVersion: "0.146.0",
	}}}
	account := &Account{
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"chatgpt_account_id": "chatgpt-acc"},
	}

	req, err := svc.buildUpstreamRequest(c.Request.Context(), c, account, []byte("{\"model\":\"gpt-5.6-sol\"}"), "token", false, "", true)

	require.NoError(t, err)
	require.Equal(t, "0.145.0-alpha.18", req.Header.Get("Version"))
	require.NotEqual(t, "999.999.999", req.Header.Get("Version"))
}

func TestBuildUpstreamRequestOpenAIPassthroughCompactUsesConfiguredFallback(t *testing.T) {
	for _, userAgent := range []string{"curl/8.0", "codex_vscode/not-a-version"} {
		t.Run(userAgent, func(t *testing.T) {
			c := newOpenAICompactVersionPolicyContext(t, userAgent, "999.999.999")
			svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{
				OpenAICompactFallbackVersion: "0.146.0",
			}}}
			account := &Account{
				Type:        AccountTypeOAuth,
				Credentials: map[string]any{"chatgpt_account_id": "chatgpt-acc"},
			}

			req, err := svc.buildUpstreamRequestOpenAIPassthrough(c.Request.Context(), c, account, []byte("{\"model\":\"gpt-5.6-sol\"}"), "token")

			require.NoError(t, err)
			require.Equal(t, "0.146.0", req.Header.Get("Version"))
			require.NotEqual(t, "999.999.999", req.Header.Get("Version"))
		})
	}
}

func TestHandleErrorResponseCodexVersionTooLowReturnsDiagnosticBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(
			"{\"error\":{\"message\":\"The 'gpt-5.6-sol' model requires a newer version of Codex. Please upgrade to the latest app or CLI and try again.\",\"type\":\"invalid_request_error\"}}",
		)),
	}
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 1, Name: "oauth", Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_request_error", gjson.Get(rec.Body.String(), "error.type").String())
	require.Equal(t, "codex_version_too_old", gjson.Get(rec.Body.String(), "error.code").String())
	require.Contains(t, gjson.Get(rec.Body.String(), "error.message").String(), "requires a newer version of Codex")
	require.NotContains(t, rec.Body.String(), "Upstream request failed")
}
