package service

import (
	"context"

	"github.com/gin-gonic/gin"
)

func openAIHTTPUpstreamProfileContextForRequest(ctx context.Context, c *gin.Context) context.Context {
	// Logical compact requests have their own per-attempt deadline.  Keep the
	// generic transport header timer disabled for both native v2 and legacy
	// /responses/compact; the latter is protected by the compact response-body
	// guard in the forwarding path, so a large context is not cut at the normal
	// 45-second OpenAI header budget.
	compactNeedsOwnHeaderGuard := isOpenAILogicalCompactRequest(c) || openAICompactClientWantsStream(c)
	if compactNeedsOwnHeaderGuard ||
		OpenAIImageGenerationIntentFromContext(ctx) ||
		isOpenAINativeImagesRequestPath(c) {
		ctx = WithOpenAINoHeaderTimeoutUpstream(ctx, true)
	}
	return WithOpenAIHTTPUpstreamProfile(ctx)
}

func isOpenAINativeImagesRequestPath(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	path := c.Request.URL.Path
	return path == openAIImagesGenerationsEndpoint || path == openAIImagesEditsEndpoint
}

func openAIHTTPUpstreamProfileContextForRequestCustom(ctx context.Context, c *gin.Context) (context.Context, bool) {
	return openAIHTTPUpstreamProfileContextForRequest(ctx, c), true
}
