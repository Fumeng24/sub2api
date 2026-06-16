package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *GeminiMessagesCompatService) forwardImageAsRawChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
) (*ForwardResult, error) {
	startTime := time.Now()
	originalModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if originalModel == "" {
		return nil, s.writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
	}
	clientStream := gjson.GetBytes(body, "stream").Bool()
	if clientStream {
		return nil, s.writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", "Gemini image generation only supports non-streaming requests")
	}

	mappedModel := originalModel
	if account.Type == AccountTypeAPIKey || account.Type == AccountTypeServiceAccount {
		mappedModel = account.GetMappedModel(originalModel)
	}
	upstreamBody := body
	if mappedModel != originalModel {
		upstreamBody = ReplaceModelInBody(body, mappedModel)
	}
	upstreamBody = stripGeminiRawImageClientFields(upstreamBody)

	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api_key not configured")
	}
	baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	targetURL := buildOpenAIChatCompletionsURL(normalizedBaseURL)

	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	upstreamReq, err := http.NewRequestWithContext(upstreamCtx, http.MethodPost, targetURL, bytes.NewReader(upstreamBody))
	releaseUpstreamCtx()
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Accept", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+apiKey)
	if ua := strings.TrimSpace(c.GetHeader("User-Agent")); ua != "" {
		upstreamReq.Header.Set("User-Agent", ua)
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "failover",
			Message:            safeErr,
		})
		setOpsUpstreamError(c, 0, safeErr, "")
		return nil, newNetworkUpstreamFailoverError(safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	requestID := resp.Header.Get("x-request-id")
	if requestID == "" {
		requestID = resp.Header.Get("x-goog-request-id")
	}
	if requestID != "" {
		c.Header("x-request-id", requestID)
	}

	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		if s.shouldFailoverGeminiUpstreamError(resp.StatusCode) {
			upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  requestID,
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody}
		}
		return nil, s.writeGeminiChatCompletionsMappedError(c, account, resp.StatusCode, requestID, respBody)
	}

	respBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}

	usage := geminiRawChatCompletionsUsage(respBody)
	imageCount := countGeminiRawChatCompletionsImages(respBody)
	if imageCount <= 0 {
		return nil, newGeminiEmptyImageFailoverError()
	}
	imageOutputSizes := collectOpenAIResponseImageOutputSizesFromJSONBytes(respBody)
	imageInputSize := extractGeminiRawChatCompletionsImageInputSize(body)
	imageSize := normalizeOpenAIImageSizeTier(imageInputSize)
	if imageSize == "" && len(imageOutputSizes) > 0 {
		imageSize = normalizeOpenAIImageSizeTier(imageOutputSizes[0])
	}

	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Writer.Header().Set("Content-Type", ct)
	} else {
		c.Writer.Header().Set("Content-Type", "application/json")
	}
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(respBody)

	return &ForwardResult{
		RequestID:        requestID,
		Usage:            usage,
		Model:            originalModel,
		UpstreamModel:    mappedModel,
		Stream:           false,
		Duration:         time.Since(startTime),
		ImageCount:       imageCount,
		ImageSize:        imageSize,
		ImageInputSize:   imageInputSize,
		ImageOutputSizes: imageOutputSizes,
		ClientDisconnect: false,
	}, nil
}

func geminiRawChatCompletionsUsage(body []byte) ClaudeUsage {
	var usage ClaudeUsage
	if !gjson.ValidBytes(body) {
		return usage
	}
	usage.InputTokens = int(gjson.GetBytes(body, "usage.prompt_tokens").Int())
	usage.OutputTokens = int(gjson.GetBytes(body, "usage.completion_tokens").Int())
	usage.CacheReadInputTokens = int(gjson.GetBytes(body, "usage.prompt_tokens_details.cached_tokens").Int())
	return usage
}

func countGeminiRawChatCompletionsImages(body []byte) int {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return 0
	}
	if count := extractOpenAIImageCountFromJSONBytes(body); count > 0 {
		return count
	}
	count := 0
	seen := make(map[string]struct{})
	gjson.GetBytes(body, "choices").ForEach(func(_, choice gjson.Result) bool {
		choice.Get("message.images").ForEach(func(_, image gjson.Result) bool {
			if key := geminiRawChatImageKey(image); key != "" {
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					count++
				}
			}
			return true
		})
		return true
	})
	return count
}

func geminiRawChatImageKey(image gjson.Result) string {
	for _, path := range []string{
		"image_url.url",
		"url",
		"b64_json",
		"image_b64",
		"image_base64",
		"base64_json",
		"base64",
	} {
		if value := strings.TrimSpace(image.Get(path).String()); value != "" {
			return hashOpenAIImageOutputResult(value)
		}
	}
	return ""
}

func extractGeminiRawChatCompletionsImageInputSize(body []byte) string {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return ""
	}
	for _, path := range []string{"image_size", "imageSize", "size"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			if tier := normalizeOpenAIImageSizeTier(value); tier != "" {
				return tier
			}
			return value
		}
	}
	return ""
}

func stripGeminiRawImageClientFields(body []byte) []byte {
	stripped := string(body)
	for _, field := range []string{
		"n",
	} {
		if next, changed := deleteJSONField(stripped, field); changed {
			stripped = next
		}
	}
	return []byte(stripped)
}
