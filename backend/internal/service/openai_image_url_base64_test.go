package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewriteOpenAIImageURLsToBase64(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	client := &http.Client{Transport: imageURLRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(bytes.NewReader(png)),
			Request:    req,
		}, nil
	})}

	body, err := rewriteOpenAIImageURLsToBase64(
		context.Background(),
		[]byte(`{"created":1,"data":[{"url":"https://images.example.test/image.png","revised_prompt":"square"}]}`),
		client,
		1024,
	)
	require.NoError(t, err)
	require.NotContains(t, string(body), "images.example.test")

	var response struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &response))
	require.Len(t, response.Data, 1)
	require.Empty(t, response.Data[0].URL)
	require.Equal(t, base64.StdEncoding.EncodeToString(png), response.Data[0].B64JSON)
}

type imageURLRoundTripFunc func(*http.Request) (*http.Response, error)

func (f imageURLRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRewriteOpenAIImageURLsToBase64RejectsResidualURL(t *testing.T) {
	_, err := rewriteOpenAIImageURLsToBase64(
		context.Background(),
		[]byte(`{"data":[{"b64_json":"aGVsbG8=","metadata":"https://provider.example/internal"}]}`),
		http.DefaultClient,
		1024,
	)
	require.ErrorContains(t, err, "still contains a provider URL")
}

func TestAccountShouldConvertOpenAIImageURLToBase64(t *testing.T) {
	require.True(t, (&Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_image_url_to_b64": true}}).ShouldConvertOpenAIImageURLToBase64())
	require.False(t, (&Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_image_url_to_b64": false}}).ShouldConvertOpenAIImageURLToBase64())
	require.False(t, (&Account{Platform: PlatformGemini, Extra: map[string]any{"openai_image_url_to_b64": true}}).ShouldConvertOpenAIImageURLToBase64())
}
