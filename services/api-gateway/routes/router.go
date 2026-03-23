package routes

import (
	"fmt"
	"time"

	"github.com/auron/api-gateway/config"
	"github.com/auron/api-gateway/middleware"
	"github.com/auron/api-gateway/proxy"
	"github.com/gin-gonic/gin"
)

// Setup configures all routes for the API Gateway
func Setup(router *gin.Engine, cfg *config.Config) error {
	// Create proxy handler
	proxyHandler, err := proxy.NewProxyHandler(cfg)
	if err != nil {
		return fmt.Errorf("create proxy handler: %w", err)
	}
	toUserAuthService := proxyHandler.ProxyToWithStrip(config.ServiceUser, "/api/auth")
	toUserAuthLegacyService := proxyHandler.ProxyToWithStrip(config.ServiceUser, "/api")
	toUserService := proxyHandler.ProxyToWithStrip(config.ServiceUser, "/api/users")
	toProductService := proxyHandler.ProxyTo(config.ServiceProduct)
	toOrderService := proxyHandler.ProxyTo(config.ServiceOrder)
	toPaymentService := proxyHandler.ProxyTo(config.ServicePayment)
	toInventoryService := proxyHandler.ProxyTo(config.ServiceInventory)

	// API v1 group
	api := router.Group("/api")

	// Health check for API (no auth)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "auron-api",
		})
	})

	// Auth routes (no auth required)
	auth := api.Group("/auth")
	auth.Use(middleware.RequestID())
	auth.Use(middleware.RateLimit(middleware.NewInMemoryRateLimiter(20, time.Minute))) // 20 req/min for auth
	{
		auth.POST("/register", toUserAuthService)
		auth.POST("/login", toUserAuthService)
		auth.POST("/refresh", toUserAuthService)
	}

	// Backward-compatible auth aliases:
	// /api/login, /api/register, /api/refresh
	api.POST("/register", toUserAuthLegacyService)
	api.POST("/register/", toUserAuthLegacyService)
	api.POST("/login", toUserAuthLegacyService)
	api.POST("/login/", toUserAuthLegacyService)
	api.POST("/refresh", toUserAuthLegacyService)
	api.POST("/refresh/", toUserAuthLegacyService)

	// Protected auth routes
	authProtected := auth.Group("")
	authProtected.Use(middleware.RequestID())
	// Note: Some auth routes need token validation, handled by user service
	{
		authProtected.POST("/logout", toUserAuthService)
	}

	// Backward-compatible logout alias
	api.POST("/logout", toUserAuthLegacyService)
	api.POST("/logout/", toUserAuthLegacyService)

	// User routes (auth required)
	users := api.Group("/users")
	users.Use(middleware.RequestID())
	{
		users.GET("/me", toUserService)
		users.PUT("/me", toUserService)
		users.POST("/me/addresses", toUserService)
		users.GET("/me/addresses", toUserService)
		users.PUT("/me/addresses/:id", toUserService)
		users.DELETE("/me/addresses/:id", toUserService)
	}

	// Product routes (public read, admin write)
	products := api.Group("/products")
	products.Use(middleware.RequestID())
	{
		// Public: read operations
		products.GET("", toProductService)
		products.GET("/:id", toProductService)

		// Protected: admin only
		products.POST("", toProductService)
		products.PUT("/:id", toProductService)
		products.DELETE("/:id", toProductService)
	}

	// Category routes
	categories := api.Group("/categories")
	categories.Use(middleware.RequestID())
	{
		categories.GET("", toProductService)
		categories.POST("", toProductService)
	}

	// Cart routes (auth required)
	cart := api.Group("/cart")
	cart.Use(middleware.RequestID())
	{
		cart.GET("", toOrderService)
		cart.POST("/items", toOrderService)
		cart.PUT("/items/:id", toOrderService)
		cart.DELETE("/items/:id", toOrderService)
	}

	// Order routes (auth required)
	orders := api.Group("/orders")
	orders.Use(middleware.RequestID())
	{
		orders.GET("", toOrderService)
		orders.POST("", toOrderService)
		orders.GET("/:id", toOrderService)
		orders.PUT("/:id/cancel", toOrderService)
	}

	// Payment routes (auth required)
	payments := api.Group("/payments")
	payments.Use(middleware.RequestID())
	{
		payments.GET("/:id", toPaymentService)
		// Stripe webhook - no auth, handled by payment service
		payments.POST("/webhook/stripe", toPaymentService)
	}

	// Inventory routes (admin only)
	inventory := api.Group("/inventory")
	inventory.Use(middleware.RequestID())
	{
		inventory.GET("/:product_id", toInventoryService)
		inventory.PUT("/:product_id", toInventoryService)
	}

	// Generic service route for future services.
	// Example: /api/services/notification/health -> SERVICE_URL_NOTIFICATION
	api.Any("/services/:service/*proxyPath", proxyHandler.ProxyByPathParam("service"))

	return nil
}
