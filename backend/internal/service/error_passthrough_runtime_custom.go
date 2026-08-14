package service

func shouldBypassErrorPassthroughCustom(platform string, upstreamStatus int, responseBody []byte) bool {
	upstreamMessage := ExtractUpstreamErrorMessage(responseBody)
	if isUpstreamBillingExhaustionError(upstreamStatus, upstreamMessage, responseBody) {
		return true
	}
	return platform == PlatformOpenAI && isOpenAIGroupDisabledUpstreamError(upstreamStatus, upstreamMessage, responseBody)
}
