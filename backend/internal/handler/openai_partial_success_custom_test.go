package handler

import (
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpenAIForwardNeedsCyberErrorUsage(t *testing.T) {
	forwardErr := errors.New("stream ended after final image")

	require.False(t, openAIForwardNeedsCyberErrorUsage(nil, nil))
	require.True(t, openAIForwardNeedsCyberErrorUsage(nil, forwardErr))
	require.True(t, openAIForwardNeedsCyberErrorUsage(&service.OpenAIForwardResult{}, forwardErr))
	require.False(t, openAIForwardNeedsCyberErrorUsage(&service.OpenAIForwardResult{ImageCount: 1}, forwardErr),
		"billable partial image continues through RecordUsage and must not create a second cyber usage row")
}
