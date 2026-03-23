package proxy

import (
	"fmt"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"

	"github.com/auron/api-gateway/config"
	"github.com/auron/api-gateway/middleware"
	"github.com/gin-gonic/gin"
)

// ProxyHandler handles reverse proxying to downstream services
type ProxyHandler struct {
	proxies map[string]*httputil.ReverseProxy
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(cfg *config.Config) (*ProxyHandler, error) {
	proxies := make(map[string]*httputil.ReverseProxy, len(cfg.ServiceURLs))
	invalidServices := make([]string, 0)
	for serviceName, targetURL := range cfg.ServiceURLs {
		proxy, err := newSingleHostProxy(targetURL)
		if err != nil {
			invalidServices = append(invalidServices, fmt.Sprintf("%s=%s", serviceName, targetURL))
			continue
		}

		proxies[serviceName] = proxy
	}

	if len(invalidServices) > 0 {
		sort.Strings(invalidServices)
		return nil, fmt.Errorf("invalid service URL configuration: %s", strings.Join(invalidServices, ", "))
	}

	return &ProxyHandler{proxies: proxies}, nil
}

// newSingleHostProxy creates a reverse proxy for a single host
func newSingleHostProxy(target string) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(target)
	if err != nil || targetURL.Scheme == "" || targetURL.Host == "" {
		return nil, fmt.Errorf("invalid target url: %s", target)
	}

	return httputil.NewSingleHostReverseProxy(targetURL), nil
}

// ProxyTo returns a Gin handler that proxies to the configured downstream service.
func (p *ProxyHandler) ProxyTo(serviceName string) gin.HandlerFunc {
	return p.ProxyToWithStrip(serviceName, "/api")
}

// ProxyToWithStrip returns a Gin handler that proxies to the configured downstream service
// while stripping a custom prefix from the incoming request path.
func (p *ProxyHandler) ProxyToWithStrip(serviceName, stripPrefix string) gin.HandlerFunc {
	serviceName = strings.ToLower(strings.TrimSpace(serviceName))

	return func(c *gin.Context) {
		proxy, exists := p.proxyForService(serviceName)
		if !exists {
			c.JSON(502, gin.H{
				"error":   "service unavailable",
				"service": serviceName,
			})
			return
		}

		p.handleProxy(c, proxy, stripPrefix)
	}
}

// ProxyByPathParam routes requests to a service resolved from a path parameter.
// Example: /api/notification/send -> service=notification, forwarded path=/send.
func (p *ProxyHandler) ProxyByPathParam(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := strings.ToLower(strings.TrimSpace(c.Param(param)))
		proxy, exists := p.proxyForService(serviceName)
		if !exists {
			c.JSON(502, gin.H{
				"error":   "service unavailable",
				"service": serviceName,
			})
			return
		}

		stripPrefix := "/api/" + serviceName
		p.handleProxy(c, proxy, stripPrefix)
	}
}

// handleProxy handles the actual proxying
func (p *ProxyHandler) handleProxy(c *gin.Context, proxy *httputil.ReverseProxy, stripPrefix string) {
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
	if stripPrefix != "" && strings.HasPrefix(path, stripPrefix) {
		path = strings.TrimPrefix(path, stripPrefix)
	}
	if path == "" {
		path = "/"
	}
	c.Request.URL.Path = path

	// Handle the proxy
	proxy.ServeHTTP(c.Writer, c.Request)
}

func (p *ProxyHandler) proxyForService(serviceName string) (*httputil.ReverseProxy, bool) {
	proxy, exists := p.proxies[serviceName]
	if !exists {
		return nil, false
	}

	return proxy, true
}
