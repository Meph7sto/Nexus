package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasCompactionTriggerInInput_DetectsCompactSignal(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.5",
		"stream":true,
		"input":[
			{"type":"message","role":"user","content":"hello"},
			{"type":"compaction_trigger"}
		]
	}`)

	require.True(t, HasCompactionTriggerInInput(body))
}

func TestHasCompactionTriggerInInput_IgnoresNonArrayInput(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","input":"compaction_trigger"}`)

	require.False(t, HasCompactionTriggerInInput(body))
}

func TestHasCompactionTriggerInInput_NoTrigger(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.5",
		"input":[{"type":"message","role":"user","content":"hello"}]
	}`)

	require.False(t, HasCompactionTriggerInInput(body))
	require.False(t, HasCompactionTriggerInInput(nil))
}
