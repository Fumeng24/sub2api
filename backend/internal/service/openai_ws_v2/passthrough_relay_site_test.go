package openai_ws_v2

import (
	"context"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRelay_SanitizeClientPayloadHookRunsBeforeClientWrite(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	rawEvent := []byte(`{"type":"error","error":{"type":"server_error","message":"raw upstream internal detail"}}`)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: rawEvent,
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		SanitizeClientPayload: func(msgType coderws.MessageType, payload []byte) []byte {
			if msgType != coderws.MessageText {
				return payload
			}
			return []byte(`{"type":"error","error":{"type":"server_error","message":"safe"}}`)
		},
	})
	require.Nil(t, relayExit)

	clientWrites := clientConn.Writes()
	require.Len(t, clientWrites, 1)
	require.Equal(t, coderws.MessageText, clientWrites[0].msgType)
	require.JSONEq(t, `{"type":"error","error":{"type":"server_error","message":"safe"}}`, string(clientWrites[0].payload))
}
