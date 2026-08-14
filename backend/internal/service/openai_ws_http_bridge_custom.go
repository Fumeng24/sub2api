package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

type openAIWSForceHTTPBridgeContextKey struct{}

func WithOpenAIWSForceHTTPBridge(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAIWSForceHTTPBridgeContextKey{}, true)
}

func openAIWSForceHTTPBridgeFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	force, _ := ctx.Value(openAIWSForceHTTPBridgeContextKey{}).(bool)
	return force
}

func (s *OpenAIGatewayService) shouldBridgeOpenAIWSHTTPWithContext(ctx context.Context, payloadBytes int, previousResponseID string) bool {
	if !s.openAIWSHTTPBridgeEnabled() || strings.TrimSpace(previousResponseID) != "" {
		return false
	}
	if openAIWSForceHTTPBridgeFromContext(ctx) {
		return true
	}
	return s.shouldBridgeOpenAIWSHTTP(nil, payloadBytes, previousResponseID)
}

func buildOpenAIWSHTTPBridgeUpstreamFailoverError(statusCode int, headers http.Header, body []byte) *UpstreamFailoverError {
	if statusCode <= 0 {
		statusCode = http.StatusBadGateway
	}
	return &UpstreamFailoverError{StatusCode: statusCode, ResponseBody: append([]byte(nil), body...), ResponseHeaders: cloneHeader(headers)}
}

func (s *OpenAIGatewayService) newOpenAIWSHTTPBridgeFailoverError(ctx context.Context, account *Account, statusCode int, headers http.Header, body []byte, updateRuntime bool) *UpstreamFailoverError {
	failoverErr := buildOpenAIWSHTTPBridgeUpstreamFailoverError(statusCode, headers, body)
	if s != nil && account != nil {
		if updateRuntime && statusCode > 0 && len(body) > 0 {
			reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
			_ = s.handleOpenAIAccountUpstreamError(ctx, account, statusCode, headers, body, reqModel)
		}
		failoverErr.SchedulerCategory = s.schedulerCategoryOverrideForOpenAIUpstreamError(ctx, account, failoverErr.StatusCode, failoverErr.ResponseBody)
	}
	return failoverErr
}

func (s *OpenAIGatewayService) openAIWSHTTPBridgeRequestErrorCustom(ctx context.Context, account *Account, safeErr string) (error, bool) {
	return nil, false
}
