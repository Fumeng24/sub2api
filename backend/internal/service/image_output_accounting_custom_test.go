package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIImageOutputCounterCountsChatCompletionsImages(t *testing.T) {
	counter := newOpenAIImageOutputCounter()
	counter.AddJSONResponse([]byte(`{
		"choices":[
			{"message":{"images":[
				{"image_url":{"url":"https://example.com/a.png"},"size":"1024x1024"},
				{"b64_json":"image-b","size":"2048x1152"}
			]}},
			{"message":{"images":[{"url":"https://example.com/a.png"}]}}
		]
	}`))

	require.Equal(t, 2, counter.Count())
	require.Equal(t, []string{"1024x1024", "2048x1152"}, counter.Sizes())
}
