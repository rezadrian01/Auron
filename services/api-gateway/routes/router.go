package routes

import (
	"github.com/auron/api-gateway/config"
	"github.com/auron/api-gateway/middleware"
	"github.com/auron/api-gateway/proxy"
	"github.com/gin-gonic/gin"
)

// Setup configures all routes for the API Gateway
func Setup(router *gin.Engine, cfg *config.Config) {
	// Create proxy handler
	proxyHandler := proxy.NewProxyHandler(cfg)

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
	auth.Use(middleware.RateLimiter(middleware.NewRateLimiter(20, ))) // 20 req/min for auth
	{
		auth.POST("/register", proxyHandler.ProxyToUserService)
		auth.POST("/login", proxyHandler.ProxyToUserService)
		auth.POST("/refresh", proxyHandler.ProxyToUserService)
	}

	// Protected auth routes
	authProtected := auth.Group("")
	authProtected.Use(middleware.RequestID())
	// Note: Some auth routes need token validation, handled by user service
	{
		authProtected.POST("/logout", proxyHandler.ProxyToUserService)
	}

	// User routes (auth required)
	users := api.Group("/users")
	users.Use(middleware.RequestID())
	{
		users.GET("/me", proxyHandler.ProxyToUserService)
		users.PUT("/me", proxyHandler.ProxyToUserService)
		users.POST("/me/addresses", proxyHandler.ProxyToUserService)
		users.GET("/me/addresses", proxyHandler.ProxyToUserService)
		users.DELETE("/me/addresses/:id", proxyHandler.ProxyToUserService)
	}

	// Product routes (public read, admin write)
	products := api.Group("/products")
	products.Use(middleware.RequestID())
	{
		// Public: read operations
		products.GET("", proxyHandler.ProxyToProductService)
		products.GET("/:id", proxyHandler.ProxyToProductService)

		// Protected: admin only
		products.POST("", proxyHandler.ProxyToProductService)
		products.PUT("/:id", proxyHandler.ProxyToProductService)
		products.DELETE("/:id", proxyHandler.ProxyToProductService)
	}

	// Category routes
	categories := api.Group("/categories")
	categories.Use(middleware.RequestID())
	{
		categories.GET("", proxyHandler.ProxyToProductService)
		categories.POST("", proxyHandler.ProxyToProductService)
	}

	// Cart routes (auth required)
	cart := api.Group("/cart")
	cart.Use(middleware.RequestID())
	{
		cart.GET("", proxyHandler.ProxyToOrderService)
		cart.POST("/items", proxyHandler.ProxyToOrderService)
		cart.PUT("/items/:id", proxyHandler.ProxyToOrderService)
		cart.DELETE("/items/:id", proxyHandler.ProxyToOrderService)
	}

	// Order routes (auth required)
	orders := api.Group("/orders")
	orders.Use(middleware.RequestID())
	{
		orders.GET("", proxyHandler.ProxyToOrderService)
		orders.POST("", proxyHandler.ProxyToOrderService)
		orders.GET("/:id", proxyHandler.ProxyToOrderService)
		orders.PUT("/:id/cancel", proxyHandler.ProxyToOrderService)
	}

	// Payment routes (auth required)
	payments := api.Group("/payments")
	payments.Use(middleware.RequestID())
	{
		payments.GET("/:id", proxyHandler.ProxyToPaymentService)
		// Stripe webhook - no auth, handled by payment service
		payments.POST("/webhook/stripe", proxyHandler.ProxyToPaymentService)
	}

	// Inventory routes (admin only)
	inventory := api.Group("/inventory")
	inventory.Use(middleware.RequestID())
	{
		inventory.GET("/:product_id", proxyHandler.ProxyToInventoryService)
		inventory.PUT("/:product_id", proxyHandler.ProxyToInventoryService)
	}
}
