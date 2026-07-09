package service

import "github.com/tidwall/gjson"

// HasCompactionTriggerInInput detects Codex remote compact v2 requests sent to
// the root Responses endpoint. These requests need the same routing,
// normalization, and account selection as /responses/compact.
func HasCompactionTriggerInInput(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		return false
	}
	found := false
	input.ForEach(func(_, item gjson.Result) bool {
		if item.Get("type").String() == "compaction_trigger" {
			found = true
			return false
		}
		return true
	})
	return found
}
