package service

import (
	"encoding/json"
	"net/http"
	"strings"
)

func countGeminiImagePartsFromBytes(body []byte) int {
	if len(body) == 0 {
		return 0
	}
	var geminiResp map[string]any
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return 0
	}
	return countGeminiImagePartsFromMap(geminiResp)
}

func countGeminiImagePartsFromMap(geminiResp map[string]any) int {
	count := 0
	for _, part := range extractGeminiParts(geminiResp) {
		if isGeminiImagePart(part) {
			count++
		}
	}
	return count
}

func isGeminiImagePart(part map[string]any) bool {
	inlineData, ok := part["inlineData"].(map[string]any)
	if !ok {
		inlineData, ok = part["inline_data"].(map[string]any)
	}
	if !ok {
		return false
	}
	mimeType, _ := inlineData["mimeType"].(string)
	if strings.TrimSpace(mimeType) == "" {
		mimeType, _ = inlineData["mime_type"].(string)
	}
	data, _ := inlineData["data"].(string)
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "image/") && strings.TrimSpace(data) != ""
}

func newGeminiEmptyImageFailoverError() *UpstreamFailoverError {
	return &UpstreamFailoverError{
		StatusCode:        http.StatusBadGateway,
		ResponseBody:      []byte(`{"error":{"code":502,"message":"Upstream returned no image data","status":"BAD_GATEWAY"}}`),
		SchedulerCategory: "transient",
	}
}
