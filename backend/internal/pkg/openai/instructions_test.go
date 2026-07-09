package openai

import (
	"strings"
	"testing"
)

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// CodexBaseInstructionsForModel 应按模型返回对应的真实 Codex base prompt。
func TestCodexBaseInstructionsForModel(t *testing.T) {
	cases := []struct {
		model    string
		wantHead string
	}{
		{"gpt-5-codex", "You are Codex, based on GPT-5"},
		{"gpt-5.3-codex", "You are Codex, based on GPT-5"},
		{"gpt-5.3-codex-spark", "You are Codex, based on GPT-5"},
		{"gpt-5.1-codex-max", "You are Codex, based on GPT-5"},
		{"gpt-5.2-codex", "You are Codex, based on GPT-5"},
		{"gpt-5.5", "You are Codex, a coding agent based on GPT-5"},
		{" GPT-5.5 ", "You are Codex, a coding agent based on GPT-5"},
		{"gpt-5.2", "You are GPT-5.2 running in the Codex CLI"},
		{"gpt-5.1", "You are GPT-5.1 running in the Codex CLI"},
		{"gpt-5", "You are Codex, a coding agent based on GPT-5"},   // 回退到最新（GPT-5.5）
		{"gpt-5.4", "You are Codex, a coding agent based on GPT-5"}, // 未单独维护 → 最新
		{"gpt-5.3", "You are Codex, a coding agent based on GPT-5"}, // 未单独维护 → 最新
		{"some-unknown-model", "You are Codex, a coding agent based on GPT-5"},
		{"", "You are Codex, a coding agent based on GPT-5"}, // 回退到最新
	}
	for _, c := range cases {
		got := strings.TrimSpace(CodexBaseInstructionsForModel(c.model))
		if got == "" {
			t.Errorf("model %q: got empty instructions", c.model)
			continue
		}
		if !strings.HasPrefix(got, c.wantHead) {
			t.Errorf("model %q: got prefix %q, want %q", c.model, firstLine(got), c.wantHead)
		}
	}
}

func TestDefaultModelsContainsGPT56Models(t *testing.T) {
	models := map[string]Model{}
	for _, model := range DefaultModels {
		models[model.ID] = model
	}

	for _, tc := range []struct {
		id          string
		displayName string
	}{
		{"gpt-5.6-sol", "GPT-5.6 Sol"},
		{"gpt-5.6-terra", "GPT-5.6 Terra"},
		{"gpt-5.6-luna", "GPT-5.6 Luna"},
	} {
		model, ok := models[tc.id]
		if !ok {
			t.Fatalf("DefaultModels missing %q", tc.id)
		}
		if model.DisplayName != tc.displayName {
			t.Fatalf("DefaultModels[%q].DisplayName = %q, want %q", tc.id, model.DisplayName, tc.displayName)
		}
	}
}
