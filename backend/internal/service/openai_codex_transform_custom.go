package service

const codexImageGenerationBridgeTextCustom = codexImageGenerationBridgeMarker + "\nWhen the user asks for raster image generation or editing, use the native `image_generation` tool.\n</sub2api-codex-image-generation>"

func init() {
	for _, model := range []string{
		"gpt-5.4-chat-latest",
		"gpt-5.3", "gpt-5.3-none", "gpt-5.3-low", "gpt-5.3-medium", "gpt-5.3-high", "gpt-5.3-xhigh",
		"gpt-5", "gpt-5-mini", "gpt-5-nano", "gpt-5.1",
		"gpt-5.1-codex", "gpt-5.1-codex-max", "gpt-5.1-codex-mini",
		"gpt-5.2-codex", "codex-mini-latest", "gpt-5-codex",
	} {
		delete(codexModelMap, model)
	}
}

func normalizeEmptyCodexModelCustom() (string, bool) {
	return "", true
}
