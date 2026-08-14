package routes

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCustomPaymentRoutesRegisterGMPayWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterCustomPaymentRoutes(router.Group("/api/v1"), &handler.PaymentWebhookHandler{})

	registered := 0
	for _, route := range router.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/payment/webhook/gmpay" {
			registered++
		}
	}

	require.Equal(t, 1, registered)
}
