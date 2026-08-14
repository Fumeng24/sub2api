package service

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// gatewayTransportFailoverError keeps the protocol path free of direct 502
// writes. The outer failover loop owns the final client response and can try
// another account first.
func (s *GatewayService) gatewayTransportFailoverError(c *gin.Context, account *Account, upstreamURL string, err error) error {
	if err == nil {
		err = errors.New("upstream request failed")
	}
	safeErr := sanitizeUpstreamErrorMessage(err.Error())
	setOpsUpstreamError(c, 0, safeErr, "")
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           accountPlatform(account),
		AccountID:          accountID(account),
		AccountName:        accountName(account),
		UpstreamStatusCode: 0,
		UpstreamURL:        safeUpstreamURL(upstreamURL),
		Kind:               "failover",
		Message:            safeErr,
	})
	return newNetworkUpstreamFailoverError(safeErr)
}
