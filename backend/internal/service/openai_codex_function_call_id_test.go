//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexFunctionCallID_StripsItemIDWhenPreservingReferences(t *testing.T) {
	input := []any{
		map[string]any{
			"type":    "function_call",
			"id":      "item_A9v0SNfS3VaLrfX0j3y4xhyK",
			"call_id": "fc_abc123",
			"name":    "bash",
		},
		map[string]any{
			"type":    "function_call_output",
			"call_id": "fc_abc123",
			"output":  "done",
		},
	}

	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{
		PreserveReferences: true,
	})

	require.Len(t, filtered, 2)
	fc, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "function_call", fc["type"])
	require.NotContains(t, fc, "id")
	require.Equal(t, "fc_abc123", fc["call_id"])
	require.Equal(t, "bash", fc["name"])
}

func TestCodexFunctionCallID_KeepsFcIDWhenPreservingReferences(t *testing.T) {
	input := []any{
		map[string]any{
			"type":    "function_call",
			"id":      "fc_validID123",
			"call_id": "fc_validID123",
			"name":    "bash",
		},
	}

	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{
		PreserveReferences: true,
	})

	require.Len(t, filtered, 1)
	fc, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "fc_validID123", fc["id"])
}

func TestCodexFunctionCallID_StripsItemIDFromToolCallInputTypes(t *testing.T) {
	types := []string{"function_call", "tool_call", "local_shell_call", "custom_tool_call", "mcp_tool_call"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			input := []any{
				map[string]any{
					"type":    typ,
					"id":      "item_xyz",
					"call_id": "fc_001",
					"name":    "tool",
				},
			}

			filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{
				PreserveReferences: true,
			})

			require.Len(t, filtered, 1)
			item, ok := filtered[0].(map[string]any)
			require.True(t, ok)
			require.NotContains(t, item, "id")
		})
	}
}

func TestCodexFunctionCallID_OutputAndMessageItemsKeepID(t *testing.T) {
	input := []any{
		map[string]any{
			"type":    "function_call_output",
			"id":      "o1",
			"call_id": "fc_abc",
			"output":  "done",
		},
		map[string]any{
			"type": "message",
			"id":   "item_msg_001",
			"role": "user",
		},
	}

	filtered := filterCodexInputWithOptions(input, codexInputFilterOptions{
		PreserveReferences: true,
	})

	require.Len(t, filtered, 2)
	out, ok := filtered[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "o1", out["id"])
	msg, ok := filtered[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "item_msg_001", msg["id"])
}
