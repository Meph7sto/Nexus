package handler

import (
	"regexp"
	"strconv"

	"github.com/Wei-Shaw/nexus/internal/service"
	"go.uber.org/zap"
)

const parseFailureSnippetLen = 256

var sensitiveSnippetPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)((?:"?(?:api[_-]?key|authorization|access[_-]?token|refresh[_-]?token|token|credential|credentials|secret|password)"?\s*:\s*"))([^"\\]*(?:\\.[^"\\]*)*)`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/\-]+=*`),
	regexp.MustCompile(`sk-[A-Za-z0-9][A-Za-z0-9._-]{6,}`),
}

func logRequestBodyParseFailure(reqLog *zap.Logger, body []byte, err error) {
	if reqLog == nil {
		return
	}
	if err == nil {
		err = service.DescribeInvalidJSON(body)
	}

	head := body
	var tail []byte
	if len(body) > parseFailureSnippetLen {
		head = body[:parseFailureSnippetLen]
		tail = body[len(body)-parseFailureSnippetLen:]
	}

	fields := []zap.Field{
		zap.Error(err),
		zap.Int("body_len", len(body)),
		zap.String("body_head", sanitizeBodySnippet(head)),
	}
	if len(tail) > 0 {
		fields = append(fields, zap.String("body_tail", sanitizeBodySnippet(tail)))
	}
	reqLog.Warn("parse request body failed", fields...)
}

func sanitizeBodySnippet(body []byte) string {
	snippet := string(body)
	if len(sensitiveSnippetPatterns) > 0 {
		snippet = sensitiveSnippetPatterns[0].ReplaceAllString(snippet, `${1}[REDACTED]`)
		for _, pattern := range sensitiveSnippetPatterns[1:] {
			snippet = pattern.ReplaceAllString(snippet, "[REDACTED]")
		}
	}
	return strconv.Quote(snippet)
}
