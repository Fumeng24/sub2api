package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

type openAIStreamFailedState struct {
	payload        []byte
	shouldFailover bool
}

func (s *openAIStreamFailedState) Observe(payload []byte, message string) {
	if s == nil {
		return
	}
	s.payload = append(s.payload[:0], payload...)
	s.shouldFailover = s.shouldFailover || openAIStreamFailedEventShouldFailover(payload, message)
}

func (s *openAIStreamFailedState) SanitizeForClient(payload []byte, message string, clientOutputStarted bool) ([]byte, bool) {
	if s == nil || !s.shouldFailover || !clientOutputStarted {
		return payload, false
	}
	sanitized := sanitizeOpenAITransientFailedEventForClient(payload, message)
	return sanitized, !bytes.Equal(sanitized, payload)
}

func (s *openAIStreamFailedState) Resolve(
	service *OpenAIGatewayService,
	c *gin.Context,
	account *Account,
	clientOutputStarted bool,
	upstreamRequestID string,
	message string,
	allowPreOutputFailover bool,
	passthrough bool,
) error {
	if s == nil || !s.shouldFailover {
		return newOpenAIStreamTerminalError(s.failedPayload(), message)
	}
	if allowPreOutputFailover && !openAIStreamClientOutputStarted(c, clientOutputStarted) {
		return service.newOpenAIStreamFailoverError(c, account, passthrough, upstreamRequestID, s.payload, message)
	}
	return fmt.Errorf("upstream response failed: %s", message)
}

func (s *openAIStreamFailedState) TerminalError(message string) error {
	if s == nil || !s.shouldFailover {
		return newOpenAIStreamTerminalError(s.failedPayload(), message)
	}
	return fmt.Errorf("upstream response failed: %s", message)
}

func (s *openAIStreamFailedState) failedPayload() []byte {
	if s == nil {
		return nil
	}
	return s.payload
}

func (s *OpenAIGatewayService) resolveOpenAIPassthroughStreamResultCustom(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	result *openaiStreamingResultPassthrough,
	streamErr error,
) (*openaiStreamingResultPassthrough, error, error) {
	if streamErr == nil {
		return result, nil, nil
	}
	if result != nil && result.imageCount > 0 {
		return result, streamErr, nil
	}
	return nil, nil, s.resolveOpenAIPassthroughResponseErrorCustom(ctx, c, account, streamErr)
}

func (s *OpenAIGatewayService) resolveOpenAIPassthroughResponseErrorCustom(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	err error,
) error {
	var failoverErr *UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		return failoverErr
	}
	if failoverErr := s.handleOpenAIUpstreamStreamError(ctx, c, account, err, "", true); failoverErr != nil {
		return failoverErr
	}
	return err
}
