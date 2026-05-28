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

	// Create JWT middleware (HS256, shared secret with user-service)
	jwtMiddleware, err := middleware.NewJWTMiddleware(cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("create jwt middleware: %w", err)
	}
	requireAuth := jwtMiddleware.RequireAuth()
	requireRole := jwtMiddleware.RequireRole("admin")

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

	// Auth routes — public (register, login, refresh)
	auth := api.Group("/auth")
	auth.Use(middleware.RequestID())
	auth.Use(middleware.RateLimit(middleware.NewInMemoryRateLimiter(20, time.Minute)))
	{
		auth.POST("/register", toUserAuthService)
		auth.POST("/login", toUserAuthService)
		auth.POST("/refresh", toUserAuthService)
	}

	// Backward-compatible auth aliases
	api.POST("/register", toUserAuthLegacyService)
	api.POST("/register/", toUserAuthLegacyService)
	api.POST("/login", toUserAuthLegacyService)
	api.POST("/login/", toUserAuthLegacyService)
	api.POST("/refresh", toUserAuthLegacyService)
	api.POST("/refresh/", toUserAuthLegacyService)

	// Protected auth routes (logout requires a valid token)
	authProtected := auth.Group("")
	authProtected.Use(middleware.RequestID())
	authProtected.Use(requireAuth)
	{
		authProtected.POST("/logout", toUserAuthService)
	}

	// Backward-compatible logout alias
	api.POST("/logout", requireAuth, toUserAuthLegacyService)
	api.POST("/logout/", requireAuth, toUserAuthLegacyService)

	// User routes — auth required
	users := api.Group("/users")
	users.Use(middleware.RequestID())
	users.Use(requireAuth)
	{
		users.GET("/me", toUserService)
		users.PUT("/me", toUserService)
		users.POST("/me/addresses", toUserService)
		users.GET("/me/addresses", toUserService)
		users.PUT("/me/addresses/:id", toUserService)
		users.DELETE("/me/addresses/:id", toUserService)
	}

	// Product routes — public reads, admin writes
	products := api.Group("/products")
	products.Use(middleware.RequestID())
	{
		products.GET("", toProductService)
		products.GET("/:id", toProductService)
	}
	adminProducts := products.Group("")
	adminProducts.Use(requireAuth, requireRole)
	{
		adminProducts.POST("", toProductService)
		adminProducts.PUT("/:id", toProductService)
		adminProducts.DELETE("/:id", toProductService)
	}

	// Category routes — public read, admin write
	categories := api.Group("/categories")
	categories.Use(middleware.RequestID())
	{
		categories.GET("", toProductService)
	}
	adminCategories := categories.Group("")
	adminCategories.Use(requireAuth, requireRole)
	{
		adminCategories.POST("", toProductService)
	}

	// Cart routes — auth required
	cart := api.Group("/cart")
	cart.Use(middleware.RequestID())
	cart.Use(requireAuth)
	{
		cart.GET("", toOrderService)
		cart.POST("/items", toOrderService)
		cart.PUT("/items/:id", toOrderService)
		cart.DELETE("/items/:id", toOrderService)
	}

	// Order routes — auth required
	orders := api.Group("/orders")
	orders.Use(middleware.RequestID())
	orders.Use(requireAuth)
	{
		orders.GET("", toOrderService)
		orders.POST("", toOrderService)
		orders.GET("/:id", toOrderService)
		orders.PUT("/:id/cancel", toOrderService)
	}

	// Payment routes — GET requires auth; Stripe webhook is public (Stripe signs its own payload)
	payments := api.Group("/payments")
	payments.Use(middleware.RequestID())
	{
		payments.GET("/:id", requireAuth, toPaymentService)
		payments.GET("/order/:order_id", requireAuth, toPaymentService)
		payments.POST("/webhook/stripe", toPaymentService)
	}

	// Inventory routes — GET is public (product pages show stock); PUT is admin only
	inventory := api.Group("/inventory")
	inventory.Use(middleware.RequestID())
	{
		inventory.GET("/:product_id", toInventoryService)
	}
	adminInventory := inventory.Group("")
	adminInventory.Use(requireAuth, requireRole)
	{
		adminInventory.PUT("/:product_id", toInventoryService)
	}

	// Generic escape hatch: /api/services/:service/*path -> SERVICE_URL_<SERVICE>
	api.Any("/services/:service/*proxyPath", proxyHandler.ProxyByPathParam("service"))

	return nil
}
