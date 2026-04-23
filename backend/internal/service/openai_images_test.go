package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_JSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat","size":"1024x1024","quality":"high","stream":true}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "/v1/images/generations", parsed.Endpoint)
	require.Equal(t, "gpt-image-2", parsed.Model)
	require.Equal(t, "draw a cat", parsed.Prompt)
	require.True(t, parsed.Stream)
	require.Equal(t, "1024x1024", parsed.Size)
	require.Equal(t, "1K", parsed.SizeTier)
	// stream=true and quality=high are both soft-ignorable for classification;
	// OAuth path (Basic) can serve this. Before v99.7.14 this expected Native.
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
	require.False(t, parsed.Multipart)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_MultipartEditAllowsBasic(t *testing.T) {
	// gpt-image-* with a multipart edit (model + size + image upload) is
	// supported by the ChatGPT OAuth backend via the conversation upload API,
	// so classification should return Basic, not Native.
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "replace background"))
	require.NoError(t, writer.WriteField("size", "1536x1024"))
	part, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-image-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body.Bytes())
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "/v1/images/edits", parsed.Endpoint)
	require.True(t, parsed.Multipart)
	require.Equal(t, "gpt-image-2", parsed.Model)
	require.Equal(t, "replace background", parsed.Prompt)
	require.Equal(t, "1536x1024", parsed.Size)
	require.Equal(t, "2K", parsed.SizeTier)
	require.Len(t, parsed.Uploads, 1)
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_PromptOnlyDefaultsRemainBasic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"prompt":"draw a cat"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "gpt-image-2", parsed.Model)
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_ExplicitSizeStaysBasic(t *testing.T) {
	// Regression: before this fix, any explicit size forced Native capability,
	// which caused schedulers to walk the entire OAuth pool looking for an
	// APIKey account that didn't exist. gpt-image-* at standard sizes is
	// supported by the OAuth backend and must remain Basic.
	gin.SetMode(gin.TestMode)
	body := []byte(`{"prompt":"draw a cat","size":"1024x1024"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.True(t, parsed.ExplicitSize)
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_CherryStudioShapeStaysBasic(t *testing.T) {
	// The actual payload shape CherryStudio sends for gpt-image generation.
	// Under the pre-fix classifier this was Native (forcing APIKey accounts)
	// — it should be Basic so OAuth accounts can serve it.
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"a small cat","size":"1024x1024","n":1,"response_format":"b64_json"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_ResponseFormatURLRequiresNative(t *testing.T) {
	// Real Native trigger: response_format=url. OAuth path only produces b64.
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"a cat","size":"1024x1024","response_format":"url"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.Equal(t, OpenAIImagesCapabilityNative, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_RejectsNonImageModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-5.4","prompt":"draw a cat"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.Nil(t, parsed)
	require.ErrorContains(t, err, `images endpoint requires an image model, got "gpt-5.4"`)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_MultiImageRequiresNative(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"a cat","n":2}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.Equal(t, OpenAIImagesCapabilityNative, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_CherryStudioPaintingStaysBasic(t *testing.T) {
	// Regression: CherryStudio's AihubmixPage always sends quality and
	// moderation fields for gpt-image-* models. Before this fix, presence of
	// any HasNativeOptions field forced Native capability, scheduler walked
	// the OAuth-only pool for 60s looking for APIKey accounts that didn't
	// exist, client timed out, UI stuck in loading. These fields are soft
	// presentation hints the OAuth path silently ignores — must stay Basic.
	gin.SetMode(gin.TestMode)
	body := []byte(`{"prompt":"cat","model":"gpt-image-2","size":"1024x1024","n":1,"quality":"auto","moderation":"auto"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.True(t, parsed.HasNativeOptions, "quality/moderation should still register as native options for tracking")
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability,
		"soft native options (quality, moderation, etc.) must not force Native capability")
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_StreamStaysBasic(t *testing.T) {
	// stream=true alone should not force Native. The OAuth path now wraps the
	// final payload in an SSE envelope, so clients (e.g. CherryStudio) that
	// always set stream=true can still be served by OAuth accounts.
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"a cat","size":"1024x1024","n":1,"stream":true}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.True(t, parsed.Stream)
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
}

func TestCollectOpenAIImagePointers_RecognizesDirectAssets(t *testing.T) {
	items := collectOpenAIImagePointers([]byte(`{
		"revised_prompt": "cat astronaut",
		"parts": [
			{"b64_json":"QUJD"},
			{"download_url":"https://files.example.com/image.png?sig=1"},
			{"asset_pointer":"file-service://file_123"}
		]
	}`))

	require.Len(t, items, 3)
	var sawBase64, sawURL, sawPointer bool
	for _, item := range items {
		if item.B64JSON == "QUJD" {
			sawBase64 = true
			require.Equal(t, "cat astronaut", item.Prompt)
		}
		if item.DownloadURL == "https://files.example.com/image.png?sig=1" {
			sawURL = true
		}
		if item.Pointer == "file-service://file_123" {
			sawPointer = true
		}
	}
	require.True(t, sawBase64)
	require.True(t, sawURL)
	require.True(t, sawPointer)
}

func TestResolveOpenAIImageBytes_PrefersInlineBase64(t *testing.T) {
	data, err := resolveOpenAIImageBytes(context.Background(), nil, nil, "", openAIImagePointerInfo{
		B64JSON: "data:image/png;base64,QUJD",
	})
	require.NoError(t, err)
	require.Equal(t, []byte("ABC"), data)
}

// --- SSE writer tests -----------------------------------------------------

func newSSETestWriter(t *testing.T) (*openaiImagesStreamWriter, *httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	return newOpenAIImagesStreamWriter(c), rec, c
}

func TestSSEWriter_CommitHeadersEmitsEventStream(t *testing.T) {
	w, rec, _ := newSSETestWriter(t)
	w.commitHeaders()

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/event-stream; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	require.Equal(t, "keep-alive", rec.Header().Get("Connection"))
	require.Equal(t, "no", rec.Header().Get("X-Accel-Buffering"))
	require.Equal(t, ": generating\n\n", rec.Body.String())
}

func TestSSEWriter_WriteFinalEmitsSingleCompletedEvent(t *testing.T) {
	w, rec, _ := newSSETestWriter(t)
	w.commitHeaders()

	jsonBody := []byte(`{"created":1700000000,"data":[{"b64_json":"SGVsbG8="}]}`)
	parsed := &OpenAIImagesRequest{
		Size: "1024x1024",
		Body: []byte(`{"model":"gpt-image-2","prompt":"a cat","size":"1024x1024","quality":"high"}`),
	}
	usage := OpenAIUsage{InputTokens: 7, OutputTokens: 14, ImageOutputTokens: 1024}

	w.writeFinal(jsonBody, parsed, "gpt-image-2", usage)

	// Exactly one `data:` line plus the initial `: generating`.
	body := rec.Body.String()
	require.Equal(t, 1, strings.Count(body, "\ndata:"), "expected exactly one data: event after the initial comment, got:\n%s", body)
	require.NotContains(t, body, "[DONE]", "images endpoint must not emit [DONE]")
	require.NotContains(t, body, "\ndata: {\"created\"", "must not emit legacy wrapper data event")

	// Parse the single data event and check required fields.
	idx := strings.Index(body, "\ndata:")
	require.GreaterOrEqual(t, idx, 0)
	line := strings.TrimPrefix(strings.SplitN(body[idx+1:], "\n\n", 2)[0], "data: ")
	var ev map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &ev))
	require.Equal(t, "image_generation.completed", ev["type"])
	require.Equal(t, "SGVsbG8=", ev["b64_json"])
	require.Equal(t, float64(1700000000), ev["created_at"])
	require.Equal(t, "gpt-image-2", ev["model"])
	require.Equal(t, "1024x1024", ev["size"])
	require.Equal(t, "png", ev["output_format"])
	require.Equal(t, "high", ev["quality"], "quality must be echoed from request")
	usageMap, ok := ev["usage"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(7), usageMap["input_tokens"])
	require.Equal(t, float64(14), usageMap["output_tokens"])
	require.Equal(t, float64(1024), usageMap["image_output_tokens"])
}

func TestSSEWriter_WriteFinalOmitsUnknownFields(t *testing.T) {
	// Without quality/background in request body and without real usage,
	// the event must omit those fields rather than inventing defaults.
	w, rec, _ := newSSETestWriter(t)
	w.commitHeaders()

	parsed := &OpenAIImagesRequest{
		Size: "1024x1024",
		Body: []byte(`{"model":"gpt-image-2","prompt":"cat"}`),
	}
	w.writeFinal([]byte(`{"created":1,"data":[{"b64_json":"QQ=="}]}`), parsed, "gpt-image-2", OpenAIUsage{})

	body := rec.Body.String()
	idx := strings.Index(body, "\ndata:")
	line := strings.TrimPrefix(strings.SplitN(body[idx+1:], "\n\n", 2)[0], "data: ")
	var ev map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &ev))

	_, hasQuality := ev["quality"]
	_, hasBackground := ev["background"]
	_, hasUsage := ev["usage"]
	require.False(t, hasQuality, "quality must be omitted when not sent in request")
	require.False(t, hasBackground, "background must be omitted when not sent in request")
	require.False(t, hasUsage, "usage must be omitted when we have no real counts")
}

func TestSSEWriter_WriteErrorEmitsFailedTypeNoDoneMarker(t *testing.T) {
	w, rec, _ := newSSETestWriter(t)
	w.commitHeaders()
	w.writeError(errors.New("upstream boom"))

	body := rec.Body.String()
	require.NotContains(t, body, "[DONE]", "error path must not emit [DONE]")
	require.Contains(t, body, `"type":"image_generation.failed"`)
	require.Contains(t, body, "upstream boom")
}

// TestSSEWriter_EmitsEventPrefixForDispatch guards the "event:" dispatch line
// on both success and error terminators. OpenAI's real images stream emits
// both "event: <type>\n" AND "data: <json>\n\n"; strict SSE parsers (Vercel
// AI SDK used by CherryStudio) dispatch by the event name. Without the
// event: line, the client treats the terminator as an anonymous "message"
// event and never matches its completed/failed handler — image renders but
// spinner spins forever.
func TestSSEWriter_EmitsEventPrefixForDispatch(t *testing.T) {
	t.Run("writeFinal emits event: image_generation.completed", func(t *testing.T) {
		w, rec, _ := newSSETestWriter(t)
		w.commitHeaders()
		parsed := &OpenAIImagesRequest{Size: "1024x1024", Body: []byte(`{"model":"gpt-image-2"}`)}
		w.writeFinal([]byte(`{"created":1,"data":[{"b64_json":"QQ=="}]}`), parsed, "gpt-image-2", OpenAIUsage{})
		body := rec.Body.String()
		require.Contains(t, body, "event: image_generation.completed\ndata:",
			"terminator must be prefixed with 'event: image_generation.completed' so strict SSE clients dispatch correctly; body was:\n%s", body)
	})

	t.Run("writeError emits event: image_generation.failed", func(t *testing.T) {
		w, rec, _ := newSSETestWriter(t)
		w.commitHeaders()
		w.writeError(errors.New("boom"))
		body := rec.Body.String()
		require.Contains(t, body, "event: image_generation.failed\ndata:",
			"error terminator must be prefixed with 'event: image_generation.failed'; body was:\n%s", body)
	})
}

func TestSSEWriter_WriteFinalStopsKeepaliveAtomically(t *testing.T) {
	// After writeFinal returns, the keepalive goroutine must be stopped so
	// no `: keepalive\n\n` can land after the completed event.
	w, rec, _ := newSSETestWriter(t)
	w.commitHeaders()
	w.startKeepalive()

	w.writeFinal(
		[]byte(`{"created":1,"data":[{"b64_json":"QQ=="}]}`),
		&OpenAIImagesRequest{Size: "1K"},
		"gpt-image-2",
		OpenAIUsage{},
	)

	// Give any lingering goroutine a chance to misbehave.
	time.Sleep(50 * time.Millisecond)

	// Keepalive goroutine must have terminated (stopCh closed).
	select {
	case <-w.stopCh:
		// closed, expected
	default:
		t.Fatal("stopCh should be closed after writeFinal")
	}

	// The last data event in the body must be the completed event,
	// not a keepalive comment.
	body := rec.Body.String()
	lastIdx := strings.LastIndex(body, "\ndata:")
	require.GreaterOrEqual(t, lastIdx, 0)
	tail := body[lastIdx:]
	require.Contains(t, tail, "image_generation.completed")
	// Nothing should follow the trailing \n\n of the completed event except
	// possibly whitespace.
	trailing := strings.TrimSpace(body[lastIdx+len(tail):])
	require.Empty(t, trailing, "no bytes should appear after the completed event; got %q", trailing)
}

func TestSSEWriter_StopIsIdempotent(t *testing.T) {
	w, _, _ := newSSETestWriter(t)
	w.commitHeaders()
	w.startKeepalive()
	w.stop()
	// Must not panic on second call.
	require.NotPanics(t, func() { w.stop() })
}
