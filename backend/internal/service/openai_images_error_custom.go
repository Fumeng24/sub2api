package service

import (
	"net/http"
	"strings"
)

func openAIImagesClientStatusCodeCustom(err *OpenAIImagesUpstreamError) (int, bool) {
	if err == nil || err.StatusCode <= 0 {
		return http.StatusBadGateway, true
	}
	errType := strings.TrimSpace(err.ErrorType)
	if errType == "" {
		errType = "upstream_error"
	}
	return NormalizeUpstreamClientError(err.StatusCode, errType, err.Message).Status, true
}

func openAIImagesClientErrorTypeCustom(err *OpenAIImagesUpstreamError) (string, bool) {
	if err == nil {
		return "upstream_error", true
	}
	errType := strings.TrimSpace(err.ErrorType)
	if errType == "" {
		errType = "upstream_error"
	}
	return NormalizeUpstreamClientError(err.StatusCode, errType, err.Message).Type, true
}

func openAIImagesClientMessageCustom(err *OpenAIImagesUpstreamError) (string, bool) {
	statusCode, _ := openAIImagesClientStatusCodeCustom(err)
	errType, _ := openAIImagesClientErrorTypeCustom(err)
	if err != nil {
		if message := strings.TrimSpace(err.Message); message != "" {
			return ClientFacingErrorMessage(statusCode, errType, message), true
		}
		if code := strings.TrimSpace(err.Code); code != "" {
			return ClientFacingErrorMessage(statusCode, errType, code), true
		}
	}
	return ClientFacingErrorMessage(statusCode, errType, "Upstream request failed"), true
}

func isOpenAIImagesRetryableCustom(err *OpenAIImagesUpstreamError) bool {
	return err != nil && err.StatusCode == http.StatusTooManyRequests
}
