package service

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

func openAIJSONToolsContainExplicitImageGeneration(tools gjson.Result) bool {
	if !tools.IsArray() {
		return false
	}
	found := false
	tools.ForEach(func(_, item gjson.Result) bool {
		if isImageGenNamespaceTool(item) ||
			(openAIJSONString(item.Get("type")) == "image_generation" && openAIJSONImageGenerationToolIsExplicit(item)) {
			found = true
			return false
		}
		return true
	})
	return found
}

func openAIJSONImageGenerationToolIsExplicit(tool gjson.Result) bool {
	if !tool.IsObject() {
		return false
	}
	for _, key := range openAIExplicitImageGenerationToolKeys() {
		if tool.Get(key).Exists() {
			return true
		}
	}
	return false
}

func hasOpenAIExplicitImageGenerationTool(reqBody map[string]any) bool {
	rawTools, ok := reqBody["tools"]
	if !ok || rawTools == nil {
		return false
	}
	tools, ok := rawTools.([]any)
	if !ok {
		return false
	}
	for _, rawTool := range tools {
		toolMap, ok := rawTool.(map[string]any)
		if !ok {
			continue
		}
		if isImageGenNamespaceToolMap(toolMap) ||
			(strings.TrimSpace(firstNonEmptyString(toolMap["type"])) == "image_generation" && openAIMapImageGenerationToolIsExplicit(toolMap)) {
			return true
		}
	}
	return false
}

func openAIMapImageGenerationToolIsExplicit(toolMap map[string]any) bool {
	for _, key := range openAIExplicitImageGenerationToolKeys() {
		if _, ok := toolMap[key]; ok {
			return true
		}
	}
	return false
}

func openAIExplicitImageGenerationToolKeys() []string {
	return []string{
		"model",
		"size",
		"quality",
		"background",
		"action",
		"output_format",
		"format",
		"output_compression",
		"compression",
		"partial_images",
	}
}

func openAIRequestBodyHasImageGenerationTool(body []byte) bool {
	return openAIRequestBodyHasImageGenerationDeclaration(body)
}

func openAIRequestBodyHasExplicitImageGenerationTool(body []byte) bool {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return false
	}
	return openAIJSONToolsContainExplicitImageGeneration(gjson.GetBytes(body, "tools"))
}

func removeOpenAIImplicitImageGenerationTools(reqBody map[string]any) bool {
	return stripOpenAIImageGenerationTools(reqBody)
}

func stripDisabledGroupImplicitOpenAIImageTools(body []byte) ([]byte, bool, error) {
	if !openAIRequestBodyHasImageGenerationTool(body) || openAIRequestBodyHasExplicitImageGenerationTool(body) {
		return body, false, nil
	}
	reqBody, err := getOpenAIRequestBodyMap(nil, body)
	if err != nil {
		return body, false, fmt.Errorf("parse request body: %w", err)
	}
	if !removeOpenAIImplicitImageGenerationTools(reqBody) {
		return body, false, nil
	}
	normalized, err := marshalOpenAIUpstreamJSON(reqBody)
	if err != nil {
		return body, false, fmt.Errorf("serialize request body: %w", err)
	}
	return normalized, true, nil
}
