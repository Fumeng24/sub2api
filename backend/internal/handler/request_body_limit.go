package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func extractMaxBytesError(err error) (*http.MaxBytesError, bool) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return maxErr, true
	}
	return nil, false
}

func formatBodyLimit(limit int64) string {
	const mb = 1024 * 1024
	if limit >= mb {
		return fmt.Sprintf("%dMB", limit/mb)
	}
	return fmt.Sprintf("%dB", limit)
}

func buildBodyTooLargeMessage(limit int64) string {
	return fmt.Sprintf("Request body too large, limit is %s", formatBodyLimit(limit))
}

func classifyRequestBodyReadError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, context.Canceled):
		return "client_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "unexpected_eof"
	case errors.Is(err, net.ErrClosed):
		return "connection_closed"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "client disconnected"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "use of closed network connection"):
		return "client_disconnected"
	case strings.Contains(msg, "unexpected eof"):
		return "unexpected_eof"
	default:
		return "read_error"
	}
}

func logRequestBodyReadError(c *gin.Context, log *zap.Logger, event string, err error) {
	if log == nil {
		return
	}
	contentLength := int64(-1)
	contextCanceled := false
	if c != nil && c.Request != nil {
		contentLength = c.Request.ContentLength
		if c.Request.Context() != nil && c.Request.Context().Err() != nil {
			contextCanceled = true
		}
	}
	log.Warn(event,
		zap.String("read_error_category", classifyRequestBodyReadError(err)),
		zap.Int64("content_length", contentLength),
		zap.Bool("request_context_canceled", contextCanceled),
		zap.Error(err),
	)
}
