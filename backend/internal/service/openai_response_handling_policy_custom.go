package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func openAINonStreamingProtocolClientMessageCustom(message string) string {
	return ClientFacingErrorMessage(http.StatusBadGateway, "upstream_error", message)
}

func handleOpenAIStreamTimeoutCustom(imageCount int) bool {
	return imageCount > 0
}

func (s *OpenAIGatewayService) validateOpenAINonStreamingFailureCustom(
	c *gin.Context,
	account *Account,
	resp *http.Response,
	body []byte,
) error {
	return s.openAICompactFailedContextWindowError(
		c, account, resp, body, false, extractOpenAISSEErrorMessage(body),
	)
}

func (s *OpenAIGatewayService) validateOpenAINonStreamingOutputCustom(
	c *gin.Context,
	account *Account,
	resp *http.Response,
	body []byte,
) error {
	if err := s.validateOpenAICompactResponseForFailover(c, account, resp, body, false); err != nil {
		return err
	}
	return s.validateOpenAIEmptyOutputResponseForFailover(c, account, resp, body, false)
}

func (s *OpenAIGatewayService) openAIEmptyOutputStreamValidationError(
	c *gin.Context,
	account *Account,
	resp *http.Response,
	payload []byte,
	hasEffectiveOutput bool,
	clientOutputStarted bool,
	passthrough bool,
	upstreamRequestID string,
) error {
	if !openAICompletedPayloadIsEmptyEffectiveOutput(payload, hasEffectiveOutput) {
		return nil
	}
	if !clientOutputStarted {
		return s.newOpenAIEmptyOutputFailoverError(c, account, resp, passthrough, upstreamRequestID)
	}
	return fmt.Errorf("stream usage incomplete: %s", openAIEmptyOutputCode)
}

func (s *OpenAIGatewayService) handleOpenAISSEJSONFailedTerminalCustom(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	payload []byte,
	usage *OpenAIUsage,
) error {
	message := extractOpenAISSEErrorMessage(payload)
	if message == "" {
		message = "Upstream compact response failed"
	}
	if hit, code, cyberMessage := detectOpenAICyberPolicy(payload); hit {
		s.parseSSEUsageBytes(payload, usage)
		MarkOpsCyberPolicy(c, CyberPolicyMark{
			Code:           code,
			Message:        cyberMessage,
			Body:           truncateString(string(payload), 4096),
			UpstreamStatus: http.StatusOK,
			UpstreamInTok:  usage.InputTokens,
			UpstreamOutTok: usage.OutputTokens,
		})
		return s.writeOpenAINonStreamingProtocolError(resp, c, message)
	}
	if err := s.openAICompactFailedContextWindowError(c, account, resp, payload, false, message); err != nil {
		return err
	}
	if s.autoDisableCodexImageBridgeForUnsupportedUpstream(ctx, account, message, payload) {
		return s.newOpenAIStreamFailoverError(c, account, false, resp.Header.Get("x-request-id"), payload, message)
	}
	return s.writeOpenAINonStreamingProtocolError(resp, c, message)
}
