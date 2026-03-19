package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/auron/api-gateway/config"
	"github.com/auron/api-gateway/middleware"
	"github.com/gin-gonic/gin"
)

// ProxyHandler handles reverse proxying to downstream services
type ProxyHandler struct {
	userServiceProxy      *httputil.ReverseProxy
	productServiceProxy   *httputil.ReverseProxy
	orderServiceProxy     *httputil.ReverseProxy
	paymentServiceProxy   *httputil.ReverseProxy
	inventoryServiceProxy *httputil.ReverseProxy
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(cfg *config.Config) *ProxyHandler {
	return &ProxyHandler{
		userServiceProxy:      newSingleHostProxy(cfg.UserServiceURL),
		productServiceProxy:   newSingleHostProxy(cfg.ProductServiceURL),
		orderServiceProxy:     newSingleHostProxy(cfg.OrderServiceURL),
		paymentServiceProxy:   newSingleHostProxy(cfg.PaymentServiceURL),
		inventoryServiceProxy: newSingleHostProxy(cfg.InventoryServiceURL),
	}
}

// newSingleHostProxy creates a reverse proxy for a single host
func newSingleHostProxy(target string) *httputil.ReverseProxy {
	targetURL, _ := url.Parse(target)
	return httputil.NewSingleHostReverseProxy(targetURL)
}

// ProxyToUserService proxies requests to the user service
func (p *ProxyHandler) ProxyToUserService(c *gin.Context) {
	p.handleProxy(c, p.userServiceProxy)
}

// ProxyToProductService proxies requests to the product service
func (p *ProxyHandler) ProxyToProductService(c *gin.Context) {
	p.handleProxy(c, p.productServiceProxy)
}

// ProxyToOrderService proxies requests to the order service
func (p *ProxyHandler) ProxyToOrderService(c *gin.Context) {
	p.handleProxy(c, p.orderServiceProxy)
}

// ProxyToPaymentService proxies requests to the payment service
func (p *ProxyHandler) ProxyToPaymentService(c *gin.Context) {
	p.handleProxy(c, p.paymentServiceProxy)
}

// ProxyToInventoryService proxies requests to the inventory service
func (p *ProxyHandler) ProxyToInventoryService(c *gin.Context) {
	p.handleProxy(c, p.inventoryServiceProxy)
}

// handleProxy handles the actual proxying
func (p *ProxyHandler) handleProxy(c *gin.Context, proxy *httputil.ReverseProxy) {
	// Get request ID from context
	requestID := middleware.GetRequestID(c)
	if requestID != "" {
		c.Request.Header.Set("X-Request-ID", requestID)
	}

	// Add user info to headers if available
	if userID := middleware.GetUserID(c); userID != "" {
		c.Request.Header.Set("X-User-ID", userID)
	}
	if userEmail := middleware.GetUserEmail(c); userEmail != "" {
		c.Request.Header.Set("X-User-Email", userEmail)
	}
	if userRole := middleware.GetUserRole(c); userRole != "" {
		c.Request.Header.Set("X-User-Role", userRole)
	}

	// Modify the request path (strip /api prefix)
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/api") {
		path = strings.TrimPrefix(path, "/api")
	}
	c.Request.URL.Path = path

	// Handle the proxy
	proxy.ServeHTTP(c.Writer, c.Request)
}
