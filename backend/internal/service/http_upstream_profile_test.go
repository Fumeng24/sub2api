package service

import (
	"context"
	"testing"
)

func TestWithHTTPUpstreamProfile_DefaultKeepsContext(t *testing.T) {
	ctx := context.Background()
	got := WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileDefault)
	if got != ctx {
		t.Fatal("default profile should not wrap context")
	}
}

func TestWithHTTPUpstreamProfile_OpenAI(t *testing.T) {
	ctx := WithHTTPUpstreamProfile(context.TODO(), HTTPUpstreamProfileOpenAI)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAI {
		t.Fatalf("expected profile %q, got %q", HTTPUpstreamProfileOpenAI, profile)
	}
}

func TestWithOpenAIHTTPUpstreamProfile_NoHeaderTimeout(t *testing.T) {
	ctx := WithOpenAINoHeaderTimeoutUpstream(context.Background(), true)
	ctx = WithOpenAIHTTPUpstreamProfile(ctx)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAINoHeaderTimeout {
		t.Fatalf("expected profile %q, got %q", HTTPUpstreamProfileOpenAINoHeaderTimeout, profile)
	}
}

func TestWithOpenAIHTTPUpstreamProfile_NoHeaderTimeoutTakesPriority(t *testing.T) {
	ctx := WithOpenAINoHeaderTimeoutUpstream(context.Background(), true)
	ctx = WithOpenAIWeakFallbackUpstream(ctx, true)
	ctx = WithOpenAIHTTPUpstreamProfile(ctx)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAINoHeaderTimeout {
		t.Fatalf("expected profile %q, got %q", HTTPUpstreamProfileOpenAINoHeaderTimeout, profile)
	}
}
