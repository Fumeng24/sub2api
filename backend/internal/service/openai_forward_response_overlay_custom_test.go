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

func TestResolveOpenAIForwardStreamResultCustomPreservesSuccessfulResult(t *testing.T) {
	svc := &OpenAIGatewayService{}
	result := &openaiStreamingResult{imageCount: 1}

	resolved, partialErr, terminalErr := svc.resolveOpenAIForwardStreamResultCustom(context.Background(), nil, nil, result, nil)

	require.Same(t, result, resolved)
	require.NoError(t, partialErr)
	require.NoError(t, terminalErr)
}

func TestResolveOpenAIForwardStreamResultCustomBillsCompletedImageBeforeReturningError(t *testing.T) {
	svc := &OpenAIGatewayService{}
	result := &openaiStreamingResult{imageCount: 1}
	streamErr := errors.New("stream ended after image output")

	resolved, partialErr, terminalErr := svc.resolveOpenAIForwardStreamResultCustom(context.Background(), nil, nil, result, streamErr)

	require.Same(t, result, resolved)
	require.ErrorIs(t, partialErr, streamErr)
	require.NoError(t, terminalErr)
}

func TestResolveOpenAIForwardStreamResultCustomFailsOverBeforeOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 42, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	resolved, partialErr, terminalErr := svc.resolveOpenAIForwardStreamResultCustom(
		context.Background(), c, account, &openaiStreamingResult{}, errors.New("stream read error: unexpected EOF"),
	)

	require.Nil(t, resolved)
	require.NoError(t, partialErr)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, terminalErr, &failoverErr)
	require.False(t, c.Writer.Written())
}
