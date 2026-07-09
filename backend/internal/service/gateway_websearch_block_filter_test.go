//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const emulatedWebsearchBody = `{"model":"claude-sonnet-4-6","max_tokens":1024,"messages":[` +
	`{"role":"user","content":[{"type":"text","text":"search the weather"}]},` +
	`{"role":"assistant","content":[` +
	`{"type":"server_tool_use","id":"srvtoolu_ws_0123456789abcdef","name":"web_search","input":{"query":"weather"}},` +
	`{"type":"web_search_tool_result","tool_use_id":"srvtoolu_ws_0123456789abcdef","content":[{"type":"web_search_result","url":"https://example.com","title":"Weather"}]},` +
	`{"type":"text","text":"Here are the search results for \"weather\":"}]},` +
	`{"role":"user","content":[{"type":"text","text":"thanks, continue"}]}]}`

const genuineWebsearchBody = `{"model":"claude-sonnet-4-6","max_tokens":1024,"messages":[` +
	`{"role":"user","content":[{"type":"text","text":"search"}]},` +
	`{"role":"assistant","content":[` +
	`{"type":"server_tool_use","id":"srvtoolu_01ABCDEF","name":"web_search","input":{"query":"weather"}},` +
	`{"type":"web_search_tool_result","tool_use_id":"srvtoolu_01ABCDEF","content":[{"type":"web_search_result","url":"https://example.com","title":"Weather"}]},` +
	`{"type":"text","text":"summary with citations"}]}]}`

func collectWebsearchContentTypes(t *testing.T, body []byte) []string {
	t.Helper()
	var types []string
	for _, msg := range gjson.GetBytes(body, "messages").Array() {
		for _, block := range msg.Get("content").Array() {
			types = append(types, block.Get("type").String())
		}
	}
	return types
}

func TestWebsearchHistoryBlockFilter_StripsEmulatedBlocksForAnthropicStrict(t *testing.T) {
	out := FilterWebSearchHistoryBlocks([]byte(emulatedWebsearchBody), "claude-sonnet-4-6")

	require.Equal(t, []string{"text", "text", "text"}, collectWebsearchContentTypes(t, out))
	require.Contains(t, string(out), "Here are the search results")
	require.NotContains(t, string(out), "srvtoolu_ws_")
	require.True(t, gjson.ValidBytes(out))
}

func TestWebsearchHistoryBlockFilter_KeepsGenuineBlocksForAnthropicStrict(t *testing.T) {
	body := []byte(genuineWebsearchBody)
	out := FilterWebSearchHistoryBlocks(body, "claude-sonnet-4-6")

	require.Equal(t, string(body), string(out))
}

func TestWebsearchHistoryBlockFilter_StripsAllBlocksForPassbackRequired(t *testing.T) {
	out := FilterWebSearchHistoryBlocks([]byte(genuineWebsearchBody), "glm-4.7")

	require.Equal(t, []string{"text", "text"}, collectWebsearchContentTypes(t, out))
	require.NotContains(t, string(out), "server_tool_use")
	require.NotContains(t, string(out), "web_search_tool_result")
	require.Contains(t, string(out), "summary with citations")
}

func TestWebsearchHistoryBlockFilter_UnknownModelOnlyStripsEmulatedBlocks(t *testing.T) {
	emulated := FilterWebSearchHistoryBlocks([]byte(emulatedWebsearchBody), "totally-unknown-model")
	require.Equal(t, []string{"text", "text", "text"}, collectWebsearchContentTypes(t, emulated))
	require.NotContains(t, string(emulated), "srvtoolu_ws_")

	genuine := []byte(genuineWebsearchBody)
	require.Equal(t, string(genuine), string(FilterWebSearchHistoryBlocks(genuine, "totally-unknown-model")))
}

func TestWebsearchHistoryBlockFilter_NoWebsearchBlocksFastPath(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	out := FilterWebSearchHistoryBlocks(body, "claude-sonnet-4-6")

	require.Equal(t, string(body), string(out))
}

func TestWebsearchHistoryBlockFilter_EmptiedMessageGetsPlaceholder(t *testing.T) {
	body := []byte(`{"model":"glm-4.7","messages":[` +
		`{"role":"user","content":[{"type":"text","text":"search"}]},` +
		`{"role":"assistant","content":[` +
		`{"type":"server_tool_use","id":"srvtoolu_01X","name":"web_search","input":{"query":"q"}},` +
		`{"type":"web_search_tool_result","tool_use_id":"srvtoolu_01X","content":[]}]}]}`)

	out := FilterWebSearchHistoryBlocks(body, "glm-4.7")

	msgs := gjson.GetBytes(out, "messages").Array()
	require.Len(t, msgs, 2)
	content := msgs[1].Get("content").Array()
	require.Len(t, content, 1)
	require.Equal(t, "text", content[0].Get("type").String())
	require.Equal(t, "(assistant content removed)", content[0].Get("text").String())
}

func TestWebsearchHistoryBlockFilter_InvalidOrStringMessagesUnchanged(t *testing.T) {
	for _, body := range [][]byte{
		[]byte(`{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"please explain \"server_tool_use\" blocks"}]}`),
		[]byte(`{"model":"claude-sonnet-4-6","messages":"server_tool_use"}`),
	} {
		out := FilterWebSearchHistoryBlocks(body, "claude-sonnet-4-6")
		require.Equal(t, string(body), string(out))
	}
}

func TestWebsearchHistoryBlockFilter_PreservesOtherToolBlocks(t *testing.T) {
	body := []byte(`{"model":"glm-4.7","messages":[` +
		`{"role":"assistant","content":[` +
		`{"type":"tool_use","id":"toolu_01A","name":"get_weather","input":{}},` +
		`{"type":"server_tool_use","id":"srvtoolu_ws_abc","name":"web_search","input":{"query":"q"}},` +
		`{"type":"text","text":"result"}]},` +
		`{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_01A","content":"sunny"}]}]}`)

	out := FilterWebSearchHistoryBlocks(body, "glm-4.7")

	require.Equal(t, []string{"tool_use", "text", "tool_result"}, collectWebsearchContentTypes(t, out))
}
