package service

func cloneIntPtr(src *int) *int {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func cloneStringPtr(src *string) *string {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func cloneIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneOpenAIForwardResultForUsage(src *OpenAIForwardResult) *OpenAIForwardResult {
	if src == nil {
		return nil
	}
	dst := *src
	dst.ServiceTier = cloneStringPtr(src.ServiceTier)
	dst.ReasoningEffort = cloneStringPtr(src.ReasoningEffort)
	dst.FirstTokenMs = cloneIntPtr(src.FirstTokenMs)
	dst.ImageOutputSizes = cloneStringSlice(src.ImageOutputSizes)
	dst.ImageSizeBreakdown = cloneIntMap(src.ImageSizeBreakdown)
	dst.ResponseHeaders = src.ResponseHeaders.Clone()
	return &dst
}

func claudeUsageHasTokens(usage *ClaudeUsage) bool {
	if usage == nil {
		return false
	}
	return usage.InputTokens > 0 || usage.OutputTokens > 0 || usage.CacheCreationInputTokens > 0 || usage.CacheReadInputTokens > 0 || usage.CacheCreation5mTokens > 0 || usage.CacheCreation1hTokens > 0 || usage.ImageOutputTokens > 0
}
