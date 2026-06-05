package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const (
	openAIResponsesImageGenerationSchedulerEndpoint          = "/v1/responses:image_generation"
	openAIResponsesWebSocketImageGenerationSchedulerEndpoint = "/v1/responses/ws:image_generation"
)

// OpenAIResponsesSchedulerEndpointForIntent returns the scheduler-health
// endpoint for a Responses request. Image generation tool requests are isolated
// from ordinary text /v1/responses health because some compatible upstreams
// support text responses but fail that tool chain.
func OpenAIResponsesSchedulerEndpointForIntent(endpoint string, imageIntent bool) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = "/v1/responses"
	}
	if !imageIntent {
		return endpoint
	}
	switch strings.ToLower(strings.TrimRight(endpoint, "/")) {
	case "/v1/responses/ws":
		return openAIResponsesWebSocketImageGenerationSchedulerEndpoint
	default:
		return openAIResponsesImageGenerationSchedulerEndpoint
	}
}

func openAIRequestRequiresImageGenerationBridge(req OpenAIAccountScheduleRequest) bool {
	return req.RequireCodexImageGenerationBridge ||
		isOpenAIImageGenerationSchedulerEndpoint(schedulerEndpointFromOpenAIRequest(req))
}

func isOpenAIImageGenerationSchedulerEndpoint(endpoint string) bool {
	switch strings.ToLower(strings.TrimRight(strings.TrimSpace(endpoint), "/")) {
	case openAIResponsesImageGenerationSchedulerEndpoint, openAIResponsesWebSocketImageGenerationSchedulerEndpoint:
		return true
	default:
		return false
	}
}

// WithSchedulerEndpoint stores the logical endpoint used by runtime scheduling
// health. It is intentionally separate from usage logging because account
// selection happens before the final upstream platform is known.
func WithSchedulerEndpoint(ctx context.Context, endpoint string) context.Context {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxkey.SchedulerEndpoint, endpoint)
}

func schedulerEndpointFromContext(ctx context.Context, fallback string) string {
	if ctx != nil {
		if endpoint, ok := ctx.Value(ctxkey.SchedulerEndpoint).(string); ok && strings.TrimSpace(endpoint) != "" {
			return strings.TrimSpace(endpoint)
		}
	}
	return strings.TrimSpace(fallback)
}
