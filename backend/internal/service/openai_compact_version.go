package service

import (
	"strings"

	"github.com/Wei-Shaw/nexus/internal/config"
	"github.com/Wei-Shaw/nexus/internal/pkg/openai"
)

// resolveOpenAICompactCodexVersion uses a complete version from a strictly
// recognized official Codex User-Agent. An inbound Version header is never an
// input to this decision.
func resolveOpenAICompactCodexVersion(cfg *config.Config, userAgent string) string {
	if version, ok := openai.ParseOfficialCodexClientVersion(userAgent); ok {
		return version
	}

	if cfg != nil {
		if fallback := strings.TrimSpace(cfg.Gateway.OpenAICompactFallbackVersion); openai.IsValidCodexVersion(fallback) {
			return fallback
		}
	}
	return config.DefaultOpenAICompactFallbackVersion
}
