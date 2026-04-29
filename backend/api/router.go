package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(handler *Handler, webDist string) *gin.Engine {
	r := gin.Default()

	// Security headers middleware
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; "+
				"font-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'")
		c.Next()
	})
	log.Println("[security] security headers middleware registered")

	// CORS — restrict origins in production
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false, // Changed from true — incompatible with wildcard origins
	}))

	// Initialize sub-handlers
	authHandler := NewAuthHandler()
	settingsHandler := NewSettingsHandler()
	keyHandler := NewKeyHandler()
	proxyHandler := NewProxyHandler("logs/api_calls.log")

	// === PUBLIC ROUTES ===
	r.POST("/api/v1/auth/login", LoginRateLimit(), authHandler.Login)
	r.POST("/api/v1/auth/register", authHandler.Register)
	r.GET("/api/v1/health", handler.Health)

	// Proxy endpoint — with rate limiting
	r.Any("/v1/*path", ProxyRateLimit(), proxyHandler.HandleProxy)

	// === AUTHENTICATED ROUTES ===
	auth := r.Group("/api/v1")
	auth.Use(AuthRequired())
	{
		// User
		auth.GET("/auth/me", authHandler.Me)

		// Stats
		auth.GET("/stats/overview", handler.GetOverview)
		auth.GET("/stats/balance", handler.GetBalance)
		auth.GET("/stats/balance/history", handler.GetBalanceHistory)
		auth.GET("/stats/usage/trend", handler.GetUsageTrend)
		auth.GET("/stats/usage/summary", handler.GetUsageSummary)
		auth.GET("/stats/usage/models", handler.GetModelDistribution)
		auth.GET("/stats/ratelimit", handler.GetRateLimit)
		auth.GET("/stats/errors", handler.GetRecentErrors)

		// Settings
		auth.GET("/settings", settingsHandler.GetSettings)
		auth.PUT("/settings", settingsHandler.UpdateSettings)

		// API Keys
		auth.GET("/keys/names", keyHandler.ListKeyNames)
		auth.GET("/keys", keyHandler.ListKeys)
		auth.POST("/keys", keyHandler.CreateKey)
		auth.PUT("/keys/:id", keyHandler.UpdateKey)
		auth.DELETE("/keys/:id", keyHandler.DeleteKey)
		auth.POST("/keys/:id/test", keyHandler.TestKey)

		// Proxy logs & realtime metrics
		auth.GET("/proxy/logs", proxyHandler.QueryLogs)
		auth.GET("/proxy/realtime", proxyHandler.RealtimeMetrics)
	}

	// === FRONTEND STATIC FILES ===
	if webDist != "" {
		serveFrontend(r, webDist)
	} else {
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.Status(http.StatusNotFound)
			} else {
				c.String(http.StatusOK, "DeepSeek API Monitor — Frontend not built")
			}
		})
	}

	return r
}

func serveFrontend(r *gin.Engine, distDir string) {
	absPath, err := filepath.Abs(distDir)
	if err != nil {
		panic("invalid web dist path: " + err.Error())
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return
	}

	r.Static("/assets", filepath.Join(absPath, "assets"))
	r.StaticFile("/favicon.svg", filepath.Join(absPath, "favicon.svg"))
	r.StaticFile("/icons.svg", filepath.Join(absPath, "icons.svg"))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/v1/") {
			c.Status(http.StatusNotFound)
			return
		}
		filePath := filepath.Join(absPath, path)
		if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
			c.File(filePath)
			return
		}
		c.File(filepath.Join(absPath, "index.html"))
	})
}
