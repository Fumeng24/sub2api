package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func mapGeminiUpstreamErrorCustom(statusCode int) (int, string, bool) {
	status, message := mapGeminiUpstreamErrorUnsafe(statusCode)
	return status, service.ClientFacingErrorMessage(status, "upstream_error", message), true
}

func mapGeminiUpstreamErrorUnsafe(statusCode int) (int, string) {
	switch statusCode {
	case 401:
		return http.StatusBadGateway, "Upstream authentication failed, please contact administrator"
	case 403:
		return http.StatusBadGateway, "Upstream access forbidden, please contact administrator"
	case 429:
		return http.StatusTooManyRequests, "Upstream rate limit exceeded, please retry later"
	case 529:
		return http.StatusServiceUnavailable, "Upstream service overloaded, please retry later"
	case 500, 502, 503, 504:
		return http.StatusBadGateway, "Upstream service temporarily unavailable"
	default:
		return http.StatusBadGateway, "Upstream request failed"
	}
}

func writeGoogleErrorCustom(c *gin.Context, status int, message string) bool {
	errType := "upstream_error"
	if status == http.StatusTooManyRequests {
		errType = "rate_limit_error"
	} else if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		errType = "invalid_request_error"
	}
	message = service.ClientFacingErrorMessage(status, errType, message)
	c.JSON(status, gin.H{"error": gin.H{
		"code": status, "message": message,
		"status": googleapi.HTTPStatusToGoogleStatusWithExtensions(status),
	}})
	return true
}

func writeGeminiUpstreamErrorCustom(c *gin.Context, res *service.UpstreamHTTPResult) bool {
	if res == nil || res.StatusCode < http.StatusBadRequest {
		return false
	}
	upstreamMsg := service.ExtractUpstreamErrorMessage(res.Body)
	service.SetOpsUpstreamError(c, res.StatusCode, upstreamMsg, "")
	status, message := mapGeminiUpstreamError(res.StatusCode)
	googleError(c, status, message)
	return true
}
