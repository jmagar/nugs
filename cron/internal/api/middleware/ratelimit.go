package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter holds rate limiting data
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.Mutex
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// IsAllowed checks if a request is allowed for the given key
func (rl *RateLimiter) IsAllowed(key string) (bool, int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get existing requests for this key
	requests, exists := rl.requests[key]
	if !exists {
		requests = []time.Time{}
	}

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we're within the limit
	if len(validRequests) >= rl.limit {
		rl.requests[key] = validRequests
		return false, rl.limit - len(validRequests)
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	return true, rl.limit - len(validRequests)
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)

		for key, requests := range rl.requests {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if reqTime.After(cutoff) {
					validRequests = append(validRequests, reqTime)
				}
			}

			if len(validRequests) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validRequests
			}
		}
		rl.mutex.Unlock()
	}
}

// RateLimit creates a rate limiting middleware
func RateLimit(requestsPerMinute int) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerMinute, time.Minute)

	return func(c *gin.Context) {
		// Use IP address as the key (could be enhanced to use user ID)
		key := c.ClientIP()

		// Check if authenticated user exists and use user ID instead
		if userID, exists := c.Get("user_id"); exists {
			key = fmt.Sprintf("user_%v", userID)
		}

		allowed, remaining := limiter.IsAllowed(key)

		// Set rate limit headers
		c.Header("X-Rate-Limit-Limit", strconv.Itoa(requestsPerMinute))
		c.Header("X-Rate-Limit-Remaining", strconv.Itoa(remaining))
		c.Header("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
					"details": gin.H{
						"limit":     requestsPerMinute,
						"remaining": remaining,
						"reset_at":  time.Now().Add(time.Minute).Unix(),
					},
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitWithConfig creates a rate limiting middleware with custom configuration
type RateLimitConfig struct {
	RequestsPerWindow int
	Window            time.Duration
	KeyGenerator      func(*gin.Context) string
	SkipFunc          func(*gin.Context) bool
}

// RateLimitWithCustomConfig creates a rate limiter with custom configuration
func RateLimitWithCustomConfig(config RateLimitConfig) gin.HandlerFunc {
	limiter := NewRateLimiter(config.RequestsPerWindow, config.Window)

	return func(c *gin.Context) {
		// Skip rate limiting if skip function returns true
		if config.SkipFunc != nil && config.SkipFunc(c) {
			c.Next()
			return
		}

		// Generate key for rate limiting
		key := c.ClientIP() // default
		if config.KeyGenerator != nil {
			key = config.KeyGenerator(c)
		}

		allowed, remaining := limiter.IsAllowed(key)

		// Set rate limit headers
		c.Header("X-Rate-Limit-Limit", strconv.Itoa(config.RequestsPerWindow))
		c.Header("X-Rate-Limit-Remaining", strconv.Itoa(remaining))
		c.Header("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
					"details": gin.H{
						"limit":     config.RequestsPerWindow,
						"remaining": remaining,
						"window":    config.Window.String(),
						"reset_at":  time.Now().Add(config.Window).Unix(),
					},
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
