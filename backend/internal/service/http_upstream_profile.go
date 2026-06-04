package service

import "context"

// HTTPUpstreamProfile marks HTTP upstream requests that need provider-specific
// transport policy.
type HTTPUpstreamProfile string

const (
	HTTPUpstreamProfileDefault            HTTPUpstreamProfile = ""
	HTTPUpstreamProfileOpenAI             HTTPUpstreamProfile = "openai"
	HTTPUpstreamProfileOpenAIWeakFallback HTTPUpstreamProfile = "openai_weak_fallback"
)

type httpUpstreamProfileContextKey struct{}
type openAIWeakFallbackUpstreamContextKey struct{}

// WithHTTPUpstreamProfile injects an upstream transport profile into ctx.
func WithHTTPUpstreamProfile(ctx context.Context, profile HTTPUpstreamProfile) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if profile == HTTPUpstreamProfileDefault {
		return ctx
	}
	return context.WithValue(ctx, httpUpstreamProfileContextKey{}, profile)
}

// HTTPUpstreamProfileFromContext resolves the upstream transport profile from ctx.
func HTTPUpstreamProfileFromContext(ctx context.Context) HTTPUpstreamProfile {
	if ctx == nil {
		return HTTPUpstreamProfileDefault
	}
	profile, ok := ctx.Value(httpUpstreamProfileContextKey{}).(HTTPUpstreamProfile)
	if !ok {
		return HTTPUpstreamProfileDefault
	}
	switch profile {
	case HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileOpenAIWeakFallback:
		return profile
	default:
		return HTTPUpstreamProfileDefault
	}
}

func HTTPUpstreamProfileForOpenAIWeakFallback(weakFallback bool) HTTPUpstreamProfile {
	if weakFallback {
		return HTTPUpstreamProfileOpenAIWeakFallback
	}
	return HTTPUpstreamProfileOpenAI
}

func WithOpenAIWeakFallbackUpstream(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if !enabled {
		return ctx
	}
	return context.WithValue(ctx, openAIWeakFallbackUpstreamContextKey{}, true)
}

func OpenAIWeakFallbackUpstreamFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	enabled, _ := ctx.Value(openAIWeakFallbackUpstreamContextKey{}).(bool)
	return enabled
}

func WithOpenAIHTTPUpstreamProfile(ctx context.Context) context.Context {
	return WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileForOpenAIWeakFallback(OpenAIWeakFallbackUpstreamFromContext(ctx)))
}
