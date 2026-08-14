package service

func sanitizeOpenAIWSFallbackClientMessageCustom(status int, errType, message string) string {
	return ClientFacingErrorMessage(status, errType, message)
}
