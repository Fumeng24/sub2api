package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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

func TestOpenAIHTTPUpstreamProfile_ImageIntentDisablesHeaderTimeout(t *testing.T) {
	ctx := WithOpenAIImageGenerationIntent(context.Background())
	ctx = openAIHTTPUpstreamProfileContextForRequest(ctx, nil)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAINoHeaderTimeout {
		t.Fatalf("expected image profile %q, got %q", HTTPUpstreamProfileOpenAINoHeaderTimeout, profile)
	}
}

func TestOpenAIHTTPUpstreamProfile_RemoteCompactionDisablesHeaderTimeout(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	MarkOpenAIRemoteCompactionV2(c)

	ctx := openAIHTTPUpstreamProfileContextForRequest(context.Background(), c)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAINoHeaderTimeout {
		t.Fatalf("expected compact profile %q, got %q", HTTPUpstreamProfileOpenAINoHeaderTimeout, profile)
	}
}

func TestOpenAIHTTPUpstreamProfile_LegacyUnaryCompactDisablesGenericHeaderTimeout(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	ctx := openAIHTTPUpstreamProfileContextForRequest(context.Background(), c)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAINoHeaderTimeout {
		t.Fatalf("expected legacy unary compact profile %q, got %q", HTTPUpstreamProfileOpenAINoHeaderTimeout, profile)
	}
}

func TestOpenAIHTTPUpstreamProfile_BodySignalStreamingCompactDisablesHeaderTimeout(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	MarkOpenAICompactClientStream(c)

	ctx := openAIHTTPUpstreamProfileContextForRequest(context.Background(), c)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAINoHeaderTimeout {
		t.Fatalf("expected streaming compact profile %q, got %q", HTTPUpstreamProfileOpenAINoHeaderTimeout, profile)
	}
}
