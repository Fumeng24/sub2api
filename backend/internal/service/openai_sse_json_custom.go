package service

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type openAISSEJSONContextKey struct{}

type openAISSEJSONContext struct {
	ctx     context.Context
	account *Account
}

func (s *OpenAIGatewayService) handleSSEToJSONWithContext(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, body []byte, originalModel, mappedModel string) (*openaiNonStreamingResult, error) {
	restore := bindOpenAISSEJSONContextCustom(ctx, c, account)
	defer restore()
	return s.handleSSEToJSON(resp, c, body, originalModel, mappedModel)
}

func bindOpenAISSEJSONContextCustom(ctx context.Context, c *gin.Context, account *Account) func() {
	if c == nil || c.Request == nil {
		return func() {}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	originalRequest := c.Request
	policyCtx := context.WithValue(ctx, openAISSEJSONContextKey{}, openAISSEJSONContext{ctx: ctx, account: account})
	c.Request = c.Request.WithContext(policyCtx)
	return func() { c.Request = originalRequest }
}

func openAISSEJSONContextFromGin(c *gin.Context) (openAISSEJSONContext, bool) {
	if c == nil || c.Request == nil {
		return openAISSEJSONContext{}, false
	}
	state, ok := c.Request.Context().Value(openAISSEJSONContextKey{}).(openAISSEJSONContext)
	return state, ok
}

func (s *OpenAIGatewayService) handleOpenAISSEJSONFailedTerminalFromContextCustom(c *gin.Context, resp *http.Response, payload []byte, usage *OpenAIUsage) (error, bool) {
	state, ok := openAISSEJSONContextFromGin(c)
	if !ok {
		return nil, false
	}
	return s.handleOpenAISSEJSONFailedTerminalCustom(state.ctx, resp, c, state.account, payload, usage), true
}

func (s *OpenAIGatewayService) validateOpenAISSEJSONResponseFromContextCustom(c *gin.Context, resp *http.Response, body []byte) (error, bool) {
	state, ok := openAISSEJSONContextFromGin(c)
	if !ok {
		return nil, false
	}
	if err := s.validateOpenAICompactResponseForFailover(c, state.account, resp, body, false); err != nil {
		return err, true
	}
	if err := s.validateOpenAIEmptyOutputResponseForFailover(c, state.account, resp, body, false); err != nil {
		return err, true
	}
	return nil, true
}
