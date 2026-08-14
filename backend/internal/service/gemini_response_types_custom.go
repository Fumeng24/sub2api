package service

type geminiNonStreamingResult struct {
	usage      *ClaudeUsage
	imageCount int
}

type geminiNativeNonStreamingResult struct {
	usage      *ClaudeUsage
	imageCount int
}
