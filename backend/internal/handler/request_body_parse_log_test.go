package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func observedWarnLogger(t *testing.T) (*zap.Logger, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zap.WarnLevel)
	return zap.New(core), logs
}

func parseFailureFields(t *testing.T, logs *observer.ObservedLogs) map[string]any {
	t.Helper()
	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, "parse request body failed", entries[0].Message)

	fields := map[string]any{}
	for _, field := range entries[0].Context {
		switch field.Key {
		case "body_len":
			fields[field.Key] = int(field.Integer)
		case "error":
			fields[field.Key] = field.Interface.(error).Error()
		default:
			fields[field.Key] = field.String
		}
	}
	return fields
}

func TestLogRequestBodyParseFailure_DerivesDiagnosticError(t *testing.T) {
	log, logs := observedWarnLogger(t)
	body := []byte(`{"model": bad}`)

	logRequestBodyParseFailure(log, body, nil)

	fields := parseFailureFields(t, logs)
	require.Equal(t, len(body), fields["body_len"])
	require.Contains(t, fields["error"], "invalid json")
	require.Contains(t, fields["error"], "offset=11")
}

func TestLogRequestBodyParseFailure_BoundsLargeBodySnippets(t *testing.T) {
	log, logs := observedWarnLogger(t)
	body := []byte(`{"model":"gpt-5.5","big":"` + strings.Repeat("A", 1<<20) + `"`)

	logRequestBodyParseFailure(log, body, nil)

	fields := parseFailureFields(t, logs)
	head := fields["body_head"].(string)
	tail := fields["body_tail"].(string)
	require.Contains(t, head, "gpt-5.5")
	require.NotContains(t, tail, "gpt-5.5")
	require.LessOrEqual(t, len(head), parseFailureSnippetLen*4)
	require.LessOrEqual(t, len(tail), parseFailureSnippetLen*4)
}

func TestLogRequestBodyParseFailure_RedactsSensitiveSnippetValues(t *testing.T) {
	log, logs := observedWarnLogger(t)
	body := []byte(`{"api_key":"sk-super-secret-value","authorization":"Bearer secret-token","broken":`)

	logRequestBodyParseFailure(log, body, nil)

	fields := parseFailureFields(t, logs)
	head := fields["body_head"].(string)
	require.NotContains(t, head, "sk-super-secret-value")
	require.NotContains(t, head, "secret-token")
	require.Contains(t, head, "[REDACTED]")
}

func TestLogRequestBodyParseFailure_EscapesControlCharacters(t *testing.T) {
	log, logs := observedWarnLogger(t)
	body := []byte("{\"model\":\x01\n\"x\"}")

	logRequestBodyParseFailure(log, body, nil)

	fields := parseFailureFields(t, logs)
	head := fields["body_head"].(string)
	require.NotContains(t, head, "\n")
	require.NotContains(t, head, "\x01")
	require.Contains(t, head, `\n`)
	require.Contains(t, head, `\x01`)
}
