package service

import (
	"net/http"
	"strings"
)

const (
	ClientErrorTypeUpstream = "upstream_error"
	ClientErrorTypeRate     = "rate_limit_error"
)

type UpstreamClientError struct {
	Status  int
	Type    string
	Message string
}

func NormalizeUpstreamClientError(status int, errType, message string) UpstreamClientError {
	errType = strings.TrimSpace(errType)
	message = strings.TrimSpace(message)
	if errType == "" {
		errType = ClientErrorTypeUpstream
	}
	if message == "" {
		message = "Upstream request failed"
	}

	switch {
	case status < http.StatusBadRequest || status >= 600:
		return UpstreamClientError{
			Status:  http.StatusBadGateway,
			Type:    ClientErrorTypeUpstream,
			Message: clientFacingTemporaryUnavailableMessage,
		}
	case status == http.StatusUnauthorized:
		return UpstreamClientError{
			Status:  http.StatusBadGateway,
			Type:    ClientErrorTypeUpstream,
			Message: clientFacingTemporaryUnavailableMessage,
		}
	case status == http.StatusPaymentRequired:
		return UpstreamClientError{
			Status:  http.StatusBadGateway,
			Type:    ClientErrorTypeUpstream,
			Message: clientFacingTemporaryUnavailableMessage,
		}
	case status == http.StatusForbidden:
		return UpstreamClientError{
			Status:  http.StatusBadGateway,
			Type:    ClientErrorTypeUpstream,
			Message: clientFacingTemporaryUnavailableMessage,
		}
	case status == http.StatusTooManyRequests:
		return UpstreamClientError{
			Status:  http.StatusServiceUnavailable,
			Type:    ClientErrorTypeUpstream,
			Message: clientFacingTemporaryUnavailableMessage,
		}
	case status == 529:
		return UpstreamClientError{
			Status:  http.StatusServiceUnavailable,
			Type:    ClientErrorTypeUpstream,
			Message: clientFacingTemporaryUnavailableMessage,
		}
	case status >= http.StatusInternalServerError:
		return UpstreamClientError{
			Status:  http.StatusBadGateway,
			Type:    ClientErrorTypeUpstream,
			Message: clientFacingTemporaryUnavailableMessage,
		}
	default:
		return UpstreamClientError{
			Status:  status,
			Type:    errType,
			Message: ClientFacingErrorMessage(status, errType, message),
		}
	}
}

func IsLikelyClientRequestError(status int, message string, payload []byte) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		text = strings.ToLower(strings.TrimSpace(ExtractUpstreamErrorMessage(payload)))
	}
	raw := strings.ToLower(string(payload))
	switch {
	case strings.Contains(text, "invalid argument"),
		strings.Contains(raw, `"status":"invalid_argument"`),
		strings.Contains(raw, `"status": "invalid_argument"`):
		return true
	case IsClaudeCodeVersionError(message, payload):
		return true
	case strings.Contains(text, "request contains an invalid argument"),
		strings.Contains(text, "contents is required"),
		strings.Contains(text, "function call turn"),
		strings.Contains(text, "tool_config.include_server_side_tool_invocations"):
		return true
	case IsContextWindowExceededError(message, payload):
		return true
	default:
		return false
	}
}

func ClientRequestErrorFromUpstream(status int, message string, payload []byte) (int, string, string, bool) {
	if status, errType, msg, ok := ClaudeCodeVersionClientError(message, payload); ok {
		return status, errType, msg, true
	}
	if status, errType, msg, ok := ContextWindowExceededClientError("", message, payload); ok {
		return status, errType, msg, true
	}
	if !IsLikelyClientRequestError(status, message, payload) {
		return 0, "", "", false
	}
	return http.StatusBadRequest, "invalid_request_error", "Invalid request", true
}
