package service

import (
	"context"

	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) resolveOpenAIForwardStreamResultCustom(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	result *openaiStreamingResult,
	streamErr error,
) (*openaiStreamingResult, error, error) {
	if streamErr == nil {
		return result, nil, nil
	}
	if result != nil && result.imageCount > 0 {
		return result, streamErr, nil
	}
	return nil, nil, s.resolveOpenAIForwardResponseErrorCustom(ctx, c, account, streamErr)
}

func (s *OpenAIGatewayService) resolveOpenAIForwardResponseErrorCustom(ctx context.Context, c *gin.Context, account *Account, err error) error {
	if failoverErr := s.handleOpenAIUpstreamStreamError(ctx, c, account, err, "", false); failoverErr != nil {
		return failoverErr
	}
	return err
}
