package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildOpenAICacheAffinityHash_StableAcrossGrowingChatTurns(t *testing.T) {
	groupID := int64(1001)
	turn1 := []byte(`{
		"model": "gpt-5.4",
		"messages": [
			{"role": "system", "content": "You are helpful."},
			{"role": "user", "content": "start this project"}
		]
	}`)
	turn2 := []byte(`{
		"model": "gpt-5.4",
		"messages": [
			{"role": "system", "content": "You are helpful."},
			{"role": "user", "content": "start this project"},
			{"role": "assistant", "content": "Sure."},
			{"role": "user", "content": "continue"}
		]
	}`)

	first := BuildOpenAICacheAffinityHash(1, 2, &groupID, "gpt-5.4", "/v1/chat/completions", turn1)
	second := BuildOpenAICacheAffinityHash(1, 2, &groupID, "gpt-5.4", "/v1/chat/completions", turn2)
	require.NotEmpty(t, first)
	require.Equal(t, first, second)
}

func TestBuildOpenAICacheAffinityHash_StableAcrossGrowingResponsesInput(t *testing.T) {
	groupID := int64(1002)
	turn1 := []byte(`{
		"model": "gpt-5.5",
		"input": [
			{"role": "system", "content": "You are a coding assistant."},
			{"role": "user", "content": "inspect this repository"}
		]
	}`)
	turn2 := []byte(`{
		"model": "gpt-5.5",
		"input": [
			{"role": "system", "content": "You are a coding assistant."},
			{"role": "user", "content": "inspect this repository"},
			{"role": "assistant", "content": "I found the entry points."},
			{"role": "user", "content": "patch it"}
		]
	}`)

	first := BuildOpenAICacheAffinityHash(1, 2, &groupID, "gpt-5.5", "/v1/responses", turn1)
	second := BuildOpenAICacheAffinityHash(1, 2, &groupID, "gpt-5.5", "/v1/responses", turn2)
	require.NotEmpty(t, first)
	require.Equal(t, first, second)
}

func TestBuildOpenAICacheAffinityHash_IsolatedByUserKeyGroupModelEndpoint(t *testing.T) {
	groupA := int64(1003)
	groupB := int64(1004)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"same prompt"}]}`)
	base := BuildOpenAICacheAffinityHash(1, 2, &groupA, "gpt-5.4", "/v1/responses", body)
	require.NotEmpty(t, base)

	require.NotEqual(t, base, BuildOpenAICacheAffinityHash(9, 2, &groupA, "gpt-5.4", "/v1/responses", body))
	require.NotEqual(t, base, BuildOpenAICacheAffinityHash(1, 9, &groupA, "gpt-5.4", "/v1/responses", body))
	require.NotEqual(t, base, BuildOpenAICacheAffinityHash(1, 2, &groupB, "gpt-5.4", "/v1/responses", body))
	require.NotEqual(t, base, BuildOpenAICacheAffinityHash(1, 2, &groupA, "gpt-5.5", "/v1/responses", body))
	require.NotEqual(t, base, BuildOpenAICacheAffinityHash(1, 2, &groupA, "gpt-5.4", "/v1/chat/completions", body))
}

func TestBuildOpenAICacheAffinityHash_DefaultEndpoint(t *testing.T) {
	groupID := int64(1005)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"same prompt"}]}`)

	withDefault := BuildOpenAICacheAffinityHash(1, 2, &groupID, "gpt-5.4", "", body)
	withResponses := BuildOpenAICacheAffinityHash(1, 2, &groupID, "gpt-5.4", "/v1/responses", body)
	require.NotEmpty(t, withDefault)
	require.Equal(t, withResponses, withDefault)
}
