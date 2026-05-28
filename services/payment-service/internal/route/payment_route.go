package route

import (
	"auron/payment-service/internal/handler"
	"auron/payment-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPaymentRoutes(router *gin.Engine, paymentHandler *handler.PaymentHandler) {
	api := router.Group("/")
	api.GET("/payments/:id", paymentHandler.GetPaymentByID)
	api.POST("/payments/webhook/stripe", middleware.CaptureRawBody(), paymentHandler.HandleStripeWebhook)
}
