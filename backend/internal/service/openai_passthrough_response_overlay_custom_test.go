package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveOpenAIPassthroughStreamResultCustomPreservesSuccessfulResult(t *testing.T) {
	svc := &OpenAIGatewayService{}
	result := &openaiStreamingResultPassthrough{imageCount: 1}

	resolved, partialErr, terminalErr := svc.resolveOpenAIPassthroughStreamResultCustom(context.Background(), nil, nil, result, nil)

	require.Same(t, result, resolved)
	require.NoError(t, partialErr)
	require.NoError(t, terminalErr)
}

func TestResolveOpenAIPassthroughStreamResultCustomBillsCompletedImageBeforeReturningError(t *testing.T) {
	svc := &OpenAIGatewayService{}
	result := &openaiStreamingResultPassthrough{imageCount: 1}
	streamErr := errors.New("stream ended after image output")

	resolved, partialErr, terminalErr := svc.resolveOpenAIPassthroughStreamResultCustom(context.Background(), nil, nil, result, streamErr)

	require.Same(t, result, resolved)
	require.ErrorIs(t, partialErr, streamErr)
	require.NoError(t, terminalErr)
}

func TestResolveOpenAIPassthroughStreamResultCustomFailsOverBeforeOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	resolved, partialErr, terminalErr := svc.resolveOpenAIPassthroughStreamResultCustom(
		context.Background(), c, account, &openaiStreamingResultPassthrough{}, errors.New("stream read error: unexpected EOF"),
	)

	require.Nil(t, resolved)
	require.NoError(t, partialErr)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, terminalErr, &failoverErr)
	require.False(t, c.Writer.Written())
}

func TestOpenAIPassthroughFailedStreamStatePreservesTerminalPayload(t *testing.T) {
	state := &openAIStreamFailedState{}
	payload := []byte(`{"type":"response.failed","error":{"code":"invalid_request_error","message":"bad request"}}`)
	state.Observe(payload, "bad request")

	err := state.TerminalError("bad request")

	require.Error(t, err)
	require.Contains(t, err.Error(), "bad request")
}

func TestOpenAIPassthroughFailedStreamStateMarksFailoverAfterOutput(t *testing.T) {
	state := &openAIStreamFailedState{}
	payload := []byte(`{"type":"response.failed","error":{"code":"insufficient_quota"}}`)
	state.Observe(payload, "")

	err := state.TerminalError("quota exhausted")

	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream response failed: quota exhausted")
}

func TestOpenAIStreamOutputObservationTracksTextAndImages(t *testing.T) {
	observation := &openAIStreamOutputObservation{}
	imageCounter := newOpenAIImageOutputCounter()
	data := []byte(`{"type":"response.output_text.delta","delta":"hello"}`)

	imageCounter.AddSSEData(data)
	observation.Observe(data)

	require.True(t, observation.HasEffectiveOutput(imageCounter))
	require.Zero(t, imageCounter.Count())
}
