package service

import (
	"bytes"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/gjson"
)

type openAIRequestBodyDecoder func() (map[string]any, error)
type openAIRequestMutationMarker func()
type openAIRequestPathDeleter func(string)

func applyOpenAIDisabledImageToolsOverlay(
	body []byte,
	imageGenerationAllowed bool,
	ensureReqBody openAIRequestBodyDecoder,
	markDecodedModified openAIRequestMutationMarker,
) error {
	if imageGenerationAllowed || !openAIRequestBodyHasImageGenerationTool(body) || openAIRequestBodyHasExplicitImageGenerationTool(body) {
		return nil
	}
	decoded, err := ensureReqBody()
	if err != nil {
		return err
	}
	if removeOpenAIImplicitImageGenerationTools(decoded) {
		markDecodedModified()
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Removed implicit /responses image_generation tool for disabled group")
	}
	return nil
}

func applyOpenAIForwardCompatibilityOverlay(
	body []byte,
	ensureReqBody openAIRequestBodyDecoder,
	markPatchDelete openAIRequestPathDeleter,
	markDecodedModified openAIRequestMutationMarker,
) error {
	if shouldStripInvalidOpenAIPreviousResponseID(body) {
		markPatchDelete("previous_response_id")
	}
	if !bytes.Contains(body, []byte(`"status"`)) || !gjson.GetBytes(body, "input").IsArray() {
		return nil
	}
	decoded, err := ensureReqBody()
	if err != nil {
		return err
	}
	if sanitizeOpenAIResponsesInputStatusFields(decoded) {
		markDecodedModified()
	}
	return nil
}
