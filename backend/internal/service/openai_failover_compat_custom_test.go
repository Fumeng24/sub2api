package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldFailoverOpenAIUpstreamResponseUsesOfficialStatusPolicyFirst(t *testing.T) {
	svc := &OpenAIGatewayService{}

	require.True(t, svc.shouldFailoverOpenAIUpstreamResponse(http.StatusBadGateway, "", nil))
	require.True(t, svc.shouldFailoverOpenAIUpstreamResponse(http.StatusTooManyRequests, "", nil))
	require.False(t, svc.shouldFailoverOpenAIUpstreamResponse(http.StatusBadGateway, "input exceeds the context window", nil))
}

func TestShouldFailoverOpenAIUpstreamResponseCustomAddsModelUnsupported(t *testing.T) {
	svc := &OpenAIGatewayService{}
	body := []byte(`{"error":{"code":"model_not_found","message":"model not found"}}`)

	require.True(t, svc.shouldFailoverOpenAIUpstreamResponse(http.StatusNotFound, "model not found", body))
}

func TestShouldFailoverOpenAIUpstreamResponseEndpointMigrationIsImmediate(t *testing.T) {
	svc := &OpenAIGatewayService{}
	body := []byte(`{"error":{"message":"The API endpoint is not served from the panel domain. Please use the published API endpoint for this service."}}`)

	require.True(t, svc.shouldFailoverOpenAIUpstreamResponse(http.StatusGone, "", body))
	require.Equal(t, openAIUpstreamErrorAuth, classifyOpenAIUpstreamError(http.StatusGone, "", body))
	require.False(t, isOpenAIEndpointMigrationError(http.StatusGone, "", []byte(`{"error":{"message":"resource gone"}}`)))
}

func TestCustomOpenAITransientProcessingRequiresStructuredEvidenceFor400(t *testing.T) {
	message := "Our servers are currently overloaded. Please try again later."

	require.False(t, isCustomOpenAITransientProcessingError(
		http.StatusBadRequest,
		message,
		[]byte(`{"error":{"message":"Our servers are currently overloaded. Please try again later."}}`),
	))
	require.True(t, isCustomOpenAITransientProcessingError(
		http.StatusBadRequest,
		message,
		[]byte(`{"error":{"message":"Our servers are currently overloaded. Please try again later.","type":"server_error"}}`),
	))
	require.True(t, isCustomOpenAITransientProcessingError(
		http.StatusServiceUnavailable,
		"Service temporarily unavailable, please try again.",
		nil,
	))
}
