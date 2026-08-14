package service

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) handleNonStreamingResponsePassthroughForAccount(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	originalModel string,
	mappedModel string,
) (*openaiNonStreamingResultPassthrough, error) {
	restore := bindOpenAISSEJSONContextCustom(ctx, c, account)
	defer restore()
	return s.handleNonStreamingResponsePassthrough(ctx, resp, c, originalModel, mappedModel)
}

func (s *OpenAIGatewayService) handlePassthroughSSEToJSONWithContext(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	body []byte,
	originalModel string,
	mappedModel string,
) (*openaiNonStreamingResultPassthrough, error) {
	restore := bindOpenAISSEJSONContextCustom(ctx, c, account)
	defer restore()
	return s.handlePassthroughSSEToJSON(resp, c, body, originalModel, mappedModel)
}

func (s *OpenAIGatewayService) handleOpenAIPassthroughSSEFailedFromContextCustom(c *gin.Context, resp *http.Response, payload []byte, message string) (error, bool) {
	state, ok := openAISSEJSONContextFromGin(c)
	if !ok {
		return nil, false
	}
	if err := s.openAICompactFailedContextWindowError(c, state.account, resp, payload, true, message); err != nil {
		return err, true
	}
	if s.autoDisableCodexImageBridgeForUnsupportedUpstream(state.ctx, state.account, message, payload) {
		return s.newOpenAIStreamFailoverError(c, state.account, true, resp.Header.Get("x-request-id"), payload, message), true
	}
	return nil, false
}

func (s *OpenAIGatewayService) validateOpenAIPassthroughResponseFromContextCustom(c *gin.Context, resp *http.Response, body []byte) (error, bool) {
	state, ok := openAISSEJSONContextFromGin(c)
	if !ok {
		return nil, false
	}
	if value, exists := c.Get(openAIPassthroughAttemptImageIntentKey); exists {
		if imageIntent, typed := value.(bool); typed && !imageIntent {
			return nil, true
		}
	}
	if err := s.openAICompactFailedContextWindowError(c, state.account, resp, body, true, extractOpenAISSEErrorMessage(body)); err != nil {
		return err, true
	}
	if err := s.validateOpenAICompactResponseForFailover(c, state.account, resp, body, true); err != nil {
		return err, true
	}
	if err := s.validateOpenAIEmptyOutputResponseForFailover(c, state.account, resp, body, true); err != nil {
		return err, true
	}
	return nil, true
}
