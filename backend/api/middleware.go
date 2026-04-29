package api

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs API requests with duration
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()
		log.Printf("[API] %s %s -> %d (%s)", c.Request.Method, path, status, duration)
	}
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Set headers after handler runs (ensures they make it to the response)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-DNS-Prefetch-Control", "off")
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; "+
				"font-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'")
	}
}

// RateLimiter is a simple in-memory rate limiter per IP
type RateLimiter struct {
	visits    map[string][]time.Time
	limit     int
	window    time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visits: make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
	// Periodically clean up old entries
	go func() {
		for {
			time.Sleep(window)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) cleanup() {
	cutoff := time.Now().Add(-rl.window)
	for ip, times := range rl.visits {
		var active []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				active = append(active, t)
			}
		}
		if len(active) == 0 {
			delete(rl.visits, ip)
		} else {
			rl.visits[ip] = active
		}
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old entries for this IP
	var recent []time.Time
	for _, t := range rl.visits[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		rl.visits[ip] = recent
		return false
	}

	rl.visits[ip] = append(recent, now)
	return true
}

// Global rate limiters
var (
	LoginRateLimiter = NewRateLimiter(10, time.Minute)     // 10 login attempts per minute per IP
	ProxyRateLimiter = NewRateLimiter(60, time.Minute)     // 60 proxy calls per minute per IP
)

// LoginRateLimit middleware for login endpoint
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !LoginRateLimiter.Allow(ip) {
			log.Printf("[rate-limit] login exceeded for IP: %s", ip)
			c.AbortWithStatusJSON(429, gin.H{
				"error": "too many login attempts, please try again later",
			})
			return
		}
		c.Next()
	}
}

// ProxyRateLimit middleware for proxy endpoint
func ProxyRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !ProxyRateLimiter.Allow(ip) {
			c.AbortWithStatusJSON(429, gin.H{
				"error": "rate limit exceeded, please slow down",
			})
			return
		}
		c.Next()
	}
}
