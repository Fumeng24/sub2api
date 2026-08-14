package service

func geminiMessagesRequestFailoverCustom(safeErr string) (error, bool) {
	return newNetworkUpstreamFailoverError(safeErr), true
}
