package service

import (
	"context"

	"github.com/gin-gonic/gin"
)

func (s *OpenAIGatewayService) handleOpenAIUpstreamRequestError(ctx context.Context, c *gin.Context, account *Account, err error, upstreamURL string, passthrough bool) (string, *UpstreamFailoverError) {
	safeErr := ""
	if err != nil {
		safeErr = sanitizeUpstreamErrorMessage(err.Error())
	}
	if safeErr == "" {
		safeErr = "upstream request failed"
	}

	platform := ""
	accountID := int64(0)
	accountName := ""
	if account != nil {
		platform = account.Platform
		accountID = account.ID
		accountName = account.Name
	}

	setOpsUpstreamError(c, 0, safeErr, "")
	if ctx != nil && ctx.Err() != nil {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           platform,
			AccountID:          accountID,
			AccountName:        accountName,
			UpstreamStatusCode: 0,
			UpstreamURL:        upstreamURL,
			Passthrough:        passthrough,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return safeErr, nil
	}

	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()

	cooldownApplied := s.markOpenAIAccountTemporarilyUnschedulable(
		stateCtx,
		account,
		0,
		"openai_request_error",
		openAIRequestErrorCooldown,
		[]byte(safeErr),
	)
	cooldownReason := ""
	if cooldownApplied {
		cooldownReason = "openai_request_error"
	}

	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           platform,
		AccountID:          accountID,
		AccountName:        accountName,
		UpstreamStatusCode: 0,
		UpstreamURL:        upstreamURL,
		Passthrough:        passthrough,
		Kind:               "failover",
		Message:            safeErr,
		CooldownApplied:    cooldownApplied,
		CooldownReason:     cooldownReason,
	})
	return safeErr, &UpstreamFailoverError{
		StatusCode:   0,
		ResponseBody: []byte(safeErr),
	}
}
