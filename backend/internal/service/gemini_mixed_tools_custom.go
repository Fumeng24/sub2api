package service

import "encoding/json"

func normalizeGeminiRequestForAIStudioCustom(body []byte) []byte {
	return ensureGeminiMixedToolConfigBytes(body)
}

func ensureGeminiMixedToolConfigBytes(body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	if !ensureGeminiMixedToolConfig(payload) {
		return body
	}
	updated, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return updated
}

func ensureGeminiMixedToolConfig(payload map[string]any) bool {
	if payload == nil || !geminiToolsMixFunctionAndServerSide(payload["tools"]) {
		return false
	}

	if snakeConfig, ok := payload["tool_config"].(map[string]any); ok {
		if snakeConfig["include_server_side_tool_invocations"] == true {
			return false
		}
		snakeConfig["include_server_side_tool_invocations"] = true
		return true
	}

	toolConfig, ok := payload["toolConfig"].(map[string]any)
	if !ok {
		toolConfig = make(map[string]any)
		payload["toolConfig"] = toolConfig
	}
	if toolConfig["includeServerSideToolInvocations"] == true {
		return false
	}
	toolConfig["includeServerSideToolInvocations"] = true
	return true
}

func geminiToolsMixFunctionAndServerSide(rawTools any) bool {
	tools, ok := rawTools.([]any)
	if !ok || len(tools) == 0 {
		return false
	}

	hasFunctionDeclarations := false
	hasServerSideTool := false
	for _, rawTool := range tools {
		tool, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}
		if geminiToolHasFunctionDeclarations(tool) {
			hasFunctionDeclarations = true
		}
		if geminiToolHasServerSideTool(tool) {
			hasServerSideTool = true
		}
		if hasFunctionDeclarations && hasServerSideTool {
			return true
		}
	}
	return false
}

func geminiToolHasFunctionDeclarations(tool map[string]any) bool {
	if hasNonEmptyJSONArray(tool["functionDeclarations"]) {
		return true
	}
	return hasNonEmptyJSONArray(tool["function_declarations"])
}

func geminiToolHasServerSideTool(tool map[string]any) bool {
	for _, key := range []string{
		"googleSearch",
		"google_search",
		"googleSearchRetrieval",
		"google_search_retrieval",
		"codeExecution",
		"code_execution",
		"urlContext",
		"url_context",
		"retrieval",
		"enterpriseWebSearch",
		"enterprise_web_search",
	} {
		if _, ok := tool[key]; ok {
			return true
		}
	}
	return false
}

func hasNonEmptyJSONArray(v any) bool {
	arr, ok := v.([]any)
	return ok && len(arr) > 0
}
