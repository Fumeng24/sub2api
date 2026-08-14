package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterCustomPaymentRoutes registers site-specific payment endpoints.
func RegisterCustomPaymentRoutes(
	v1 *gin.RouterGroup,
	webhookHandler *handler.PaymentWebhookHandler,
) {
	v1.POST("/payment/webhook/gmpay", webhookHandler.GMPayWebhook)
}
