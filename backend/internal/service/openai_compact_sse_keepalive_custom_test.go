package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIStreamClientOutputStarted_IgnoresCompactKeepalive(t *testing.T) {
	c, _ := newCompactBridgeTestContext(t, true)
	stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
	defer stop()

	waitForKeepaliveBeats()
	require.True(t, c.Writer.Written(), "heartbeat commits the HTTP writer")
	require.False(t, openAIStreamClientOutputStarted(c, false), "heartbeat is not semantic client output")

	_, err := c.Writer.Write([]byte("data: semantic\n\n"))
	require.NoError(t, err)
	require.True(t, openAIStreamClientOutputStarted(c, false))
}
