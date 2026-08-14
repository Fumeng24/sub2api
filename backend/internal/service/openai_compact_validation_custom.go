package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func IsOpenAIContextWindowErrorForTest(text string) bool {
	return isOpenAIContextWindowError(text, nil)
}

func openAICompactBadOutputReason(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return "empty_response"
	}
	if !gjson.ValidBytes(body) {
		return "invalid_json"
	}
	root := gjson.ParseBytes(body)
	status := strings.ToLower(strings.TrimSpace(root.Get("status").String()))
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(root.Get("response.status").String()))
	}
	switch status {
	case "failed", "incomplete", "cancelled", "canceled":
		return "status_" + status
	}

	// Remote compaction v2 is not a normal Responses completion: the client
	// requires exactly one compaction output item.  A few upstreams have
	// returned HTTP 200 with an otherwise well-formed response containing only
	// message/tool items; forwarding that body makes Codex report
	// "expected exactly one compaction output item, got 0 from N output items"
	// and retry the same request blindly.  Treat this as an unusable upstream
	// result so the caller can fail over before any bytes are committed.
	compactionItems := openAICompactOutputCompactionItemCount(root)
	if compactionItems == 0 {
		return "missing_compaction_output_item"
	}
	if compactionItems > 1 {
		return "multiple_compaction_output_items"
	}

	outputRunes := utf8.RuneCountInString(strings.TrimSpace(extractOpenAICompactOutputText(root)))
	if outputRunes == 0 {
		return "empty_output"
	}
	if outputRunes < openAICompactMinOutputRunes {
		return "too_short_output"
	}

	usage, usageOK := extractOpenAIUsageFromJSONBytes(body)
	if !usageOK || usage.InputTokens < openAICompactLargeInputTokenThreshold {
		return ""
	}
	if usage.OutputTokens > 0 && usage.OutputTokens < openAICompactLargeInputMinOutputTokens {
		return "too_few_output_tokens"
	}
	if outputRunes < openAICompactLargeInputMinOutputRunes {
		return "too_short_output_for_large_input"
	}
	return ""
}

// openAICompactOutputCompactionItemCount counts compaction items in either a
// final Response object or a response.completed envelope.  The latter is
// accepted because this helper is also used on SSE-to-JSON fallback paths.
func openAICompactOutputCompactionItemCount(root gjson.Result) int {
	output := root.Get("output")
	if !output.Exists() || !output.IsArray() {
		output = root.Get("response.output")
	}
	count := 0
	for _, item := range output.Array() {
		if isResponsesCompactionItemType(item.Get("type").String()) {
			count++
		}
	}
	return count
}

func extractOpenAICompactOutputText(root gjson.Result) string {
	var parts []string
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	add(root.Get("output_text").String())
	add(root.Get("response.output_text").String())
	appendOutputText := func(output gjson.Result) {
		if !output.Exists() || !output.IsArray() {
			return
		}
		for _, item := range output.Array() {
			add(item.Get("text").String())
			add(item.Get("output_text").String())
			add(item.Get("encrypted_content").String())
			content := item.Get("content")
			if content.IsArray() {
				for _, part := range content.Array() {
					if part.Type == gjson.String {
						add(part.String())
					} else {
						add(part.Get("text").String())
						add(part.Get("output_text").String())
					}
				}
			} else if content.Type == gjson.String {
				add(content.String())
			}
			for _, part := range item.Get("summary").Array() {
				if part.Type == gjson.String {
					add(part.String())
				} else {
					add(part.Get("text").String())
				}
			}
		}
	}
	appendOutputText(root.Get("output"))
	appendOutputText(root.Get("response.output"))
	add(root.Get("message.content").String())
	for _, choice := range root.Get("choices").Array() {
		add(choice.Get("message.content").String())
	}
	return strings.Join(parts, "\n")
}

func (s *OpenAIGatewayService) openAICompactFailedContextWindowError(c *gin.Context, account *Account, resp *http.Response, payload []byte, passthrough bool, message string) error {
	if !isOpenAILogicalCompactRequest(c) || !openAIFailedPayloadIsContextWindowError(payload, message) {
		return nil
	}
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(payload), maxBytes)
	}
	return s.newOpenAICompactContextWindowFailoverError(c, account, resp, payload, passthrough, message, upstreamDetail)
}

func openAIFailedPayloadIsContextWindowError(payload []byte, message string) bool {
	if len(bytes.TrimSpace(payload)) == 0 || !gjson.ValidBytes(payload) {
		return false
	}
	root := gjson.ParseBytes(payload)
	eventType := strings.ToLower(strings.TrimSpace(root.Get("type").String()))
	status := strings.ToLower(strings.TrimSpace(root.Get("status").String()))
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(root.Get("response.status").String()))
	}
	if eventType != "response.failed" && status != "failed" {
		return false
	}
	return isOpenAIContextWindowError(message, payload)
}

func (s *OpenAIGatewayService) validateOpenAICompactResponseForFailover(c *gin.Context, account *Account, resp *http.Response, body []byte, passthrough bool) error {
	if !isOpenAILogicalCompactRequest(c) {
		return nil
	}
	reason := openAICompactBadOutputReason(body)
	if reason == "" {
		return nil
	}
	reason = sanitizeUpstreamErrorMessage(reason)
	message := "OpenAI compact returned unusable output: " + reason
	if c != nil {
		setOpsUpstreamError(c, http.StatusBadGateway, message, "")
		event := OpsUpstreamErrorEvent{
			Platform:           PlatformOpenAI,
			UpstreamStatusCode: http.StatusBadGateway,
			Passthrough:        passthrough,
			Kind:               "failover",
			Message:            message,
			Detail:             openAICompactBadOutputCode,
			CooldownApplied:    false,
		}
		if resp != nil {
			event.UpstreamRequestID = strings.TrimSpace(resp.Header.Get("x-request-id"))
		}
		if account != nil {
			event.Platform = account.Platform
			event.AccountID = account.ID
			event.AccountName = account.Name
		}
		appendOpsUpstreamError(c, event)
	}
	responseBody, _ := json.Marshal(gin.H{
		"error": gin.H{
			"type":    "upstream_error",
			"code":    openAICompactBadOutputCode,
			"message": ClientFacingErrorMessage(http.StatusBadGateway, "upstream_error", message),
		},
	})
	var headers http.Header
	if resp != nil && resp.Header != nil {
		headers = resp.Header.Clone()
	}
	return &UpstreamFailoverError{StatusCode: http.StatusBadGateway, ResponseBody: responseBody, ResponseHeaders: headers}
}

func (s *OpenAIGatewayService) newOpenAICompactContextWindowFailoverError(c *gin.Context, account *Account, resp *http.Response, body []byte, passthrough bool, upstreamMsg, upstreamDetail string) *UpstreamFailoverError {
	upstreamMsg = sanitizeUpstreamErrorMessage(strings.TrimSpace(upstreamMsg))
	if upstreamMsg == "" {
		upstreamMsg = "OpenAI compact input exceeds the context window"
	}
	statusCode := http.StatusBadRequest
	requestID := ""
	var headers http.Header
	if resp != nil {
		statusCode = resp.StatusCode
		requestID = strings.TrimSpace(resp.Header.Get("x-request-id"))
		headers = resp.Header.Clone()
	}
	if c != nil {
		event := OpsUpstreamErrorEvent{
			Platform:             PlatformOpenAI,
			UpstreamStatusCode:   statusCode,
			UpstreamRequestID:    requestID,
			Passthrough:          passthrough,
			Kind:                 "failover",
			Message:              upstreamMsg,
			Detail:               openAICompactContextWindowFallbackCode,
			UpstreamResponseBody: upstreamDetail,
			CooldownApplied:      false,
		}
		if account != nil {
			event.Platform = account.Platform
			event.AccountID = account.ID
			event.AccountName = account.Name
		}
		appendOpsUpstreamError(c, event)
	}
	return &UpstreamFailoverError{StatusCode: statusCode, ResponseBody: body, ResponseHeaders: headers}
}

func (s *OpenAIGatewayService) validateOpenAIEmptyOutputResponseForFailover(c *gin.Context, account *Account, resp *http.Response, body []byte, passthrough bool) error {
	if reason := openAIEmptyOutputReason(body); reason != "" {
		return s.newOpenAIEmptyOutputFailoverError(c, account, resp, passthrough, "", reason)
	}
	return nil
}

func (s *OpenAIGatewayService) newOpenAIEmptyOutputFailoverError(c *gin.Context, account *Account, resp *http.Response, passthrough bool, upstreamRequestID string, reasons ...string) *UpstreamFailoverError {
	reason := openAIEmptyOutputCode
	for _, candidate := range reasons {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			reason = trimmed
			break
		}
	}
	if upstreamRequestID == "" && resp != nil {
		upstreamRequestID = strings.TrimSpace(resp.Header.Get("x-request-id"))
	}
	message := "OpenAI response completed with empty effective output"
	if reason != "" && reason != openAIEmptyOutputCode {
		message += ": " + sanitizeUpstreamErrorMessage(reason)
	}
	if c != nil {
		setOpsUpstreamError(c, http.StatusBadGateway, message, openAIEmptyOutputCode)
		event := OpsUpstreamErrorEvent{
			Platform:           PlatformOpenAI,
			UpstreamStatusCode: http.StatusBadGateway,
			UpstreamRequestID:  upstreamRequestID,
			Passthrough:        passthrough,
			Kind:               "failover",
			Message:            message,
			Detail:             openAIEmptyOutputCode,
			CooldownApplied:    false,
		}
		if account != nil {
			event.Platform = account.Platform
			event.AccountID = account.ID
			event.AccountName = account.Name
		}
		appendOpsUpstreamError(c, event)
	}
	responseBody, _ := json.Marshal(gin.H{
		"error": gin.H{
			"type":    "upstream_error",
			"code":    openAIEmptyOutputCode,
			"message": ClientFacingErrorMessage(http.StatusBadGateway, "upstream_error", message),
		},
	})
	var headers http.Header
	if resp != nil && resp.Header != nil {
		headers = resp.Header.Clone()
	}
	return &UpstreamFailoverError{StatusCode: http.StatusBadGateway, ResponseBody: responseBody, ResponseHeaders: headers}
}

func openAIEmptyOutputReason(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 || !gjson.ValidBytes(body) {
		return ""
	}
	root := gjson.ParseBytes(body)
	status := strings.ToLower(strings.TrimSpace(root.Get("status").String()))
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(root.Get("response.status").String()))
	}
	if status != "" && status != "completed" && status != "done" {
		return ""
	}
	if status == "" && !root.Get("output").Exists() && !root.Get("response.output").Exists() && !root.Get("output_text").Exists() && !root.Get("response.output_text").Exists() {
		return ""
	}
	if openAIResponsePayloadHasEffectiveOutput(root) {
		return ""
	}
	usage, usageOK := extractOpenAIUsageFromJSONBytes(body)
	if !usageOK {
		return "missing_usage_empty_output"
	}
	if usage.OutputTokens > 0 || usage.ImageOutputTokens > 0 {
		return ""
	}
	return "zero_output_tokens_empty_output"
}

func openAIResponseBodyHasEffectiveOutput(body []byte) bool {
	return len(bytes.TrimSpace(body)) > 0 && gjson.ValidBytes(body) && openAIResponsePayloadHasEffectiveOutput(gjson.ParseBytes(body))
}

func openAIResponsePayloadHasEffectiveOutput(root gjson.Result) bool {
	if !root.Exists() {
		return false
	}
	eventType := strings.TrimSpace(root.Get("type").String())
	if (strings.HasPrefix(eventType, "response.output_text") || strings.HasPrefix(eventType, "response.refusal")) &&
		(strings.TrimSpace(root.Get("delta").String()) != "" || strings.TrimSpace(root.Get("text").String()) != "") {
		return true
	}
	if strings.Contains(eventType, "function_call_arguments") && strings.TrimSpace(root.Get("delta").String()) != "" {
		return true
	}
	if item := root.Get("item"); item.Exists() && openAIResponseOutputItemHasEffectiveOutput(item) {
		return true
	}
	return openAIResponseObjectHasEffectiveOutput(root) || openAIResponseObjectHasEffectiveOutput(root.Get("response"))
}

func openAIResponseObjectHasEffectiveOutput(obj gjson.Result) bool {
	if !obj.Exists() {
		return false
	}
	for _, path := range []string{"output_text", "text", "refusal"} {
		if strings.TrimSpace(obj.Get(path).String()) != "" {
			return true
		}
	}
	for _, outputPath := range []string{"output", "response.output"} {
		for _, item := range obj.Get(outputPath).Array() {
			if openAIResponseOutputItemHasEffectiveOutput(item) {
				return true
			}
		}
	}
	for _, choice := range obj.Get("choices").Array() {
		if strings.TrimSpace(choice.Get("message.content").String()) != "" {
			return true
		}
	}
	return false
}

func openAIResponseOutputItemHasEffectiveOutput(item gjson.Result) bool {
	if !item.Exists() || !item.IsObject() {
		return false
	}
	itemType := strings.ToLower(strings.TrimSpace(item.Get("type").String()))
	// A remote-compaction item is the result itself.  It commonly carries only
	// opaque/encrypted content, so it must count as effective output even when it
	// has no user-visible text/summary fields.
	if isResponsesCompactionItemType(itemType) {
		return true
	}
	for _, path := range []string{"text", "output_text", "refusal"} {
		if strings.TrimSpace(item.Get(path).String()) != "" {
			return true
		}
	}
	if itemType == "image_generation_call" && strings.TrimSpace(item.Get("result").String()) != "" {
		return true
	}
	if isOpenAIToolOutputItemType(itemType) {
		for _, path := range []string{"name", "arguments", "call_id", "id", "status"} {
			if strings.TrimSpace(item.Get(path).String()) != "" {
				return true
			}
		}
	}
	content := item.Get("content")
	if content.IsArray() {
		for _, part := range content.Array() {
			if part.Type == gjson.String && strings.TrimSpace(part.String()) != "" {
				return true
			}
			for _, path := range []string{"text", "output_text", "refusal"} {
				if strings.TrimSpace(part.Get(path).String()) != "" {
					return true
				}
			}
		}
	} else if content.Type == gjson.String && strings.TrimSpace(content.String()) != "" {
		return true
	}
	for _, part := range item.Get("summary").Array() {
		if (part.Type == gjson.String && strings.TrimSpace(part.String()) != "") || strings.TrimSpace(part.Get("text").String()) != "" {
			return true
		}
	}
	return false
}

func isOpenAIToolOutputItemType(itemType string) bool {
	switch itemType {
	case "function_call", "custom_tool_call", "tool_call", "web_search_call", "file_search_call", "computer_call", "mcp_call":
		return true
	default:
		return itemType != "" && (strings.Contains(itemType, "tool") || strings.Contains(itemType, "function_call"))
	}
}
