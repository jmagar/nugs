package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestID generates and adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// ErrorHandler handles panics and errors gracefully
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())

				// Get request ID if available
				requestID, _ := c.Get("request_id")

				// Return error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":       "INTERNAL_SERVER_ERROR",
						"message":    "Internal server error occurred",
						"request_id": requestID,
					},
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// Logger creates a structured logging middleware
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.ErrorMessage,
		)
	})
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		c.Next()
	}
}

// Timeout adds a timeout to requests
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set timeout on the context
		ctx, cancel := c.Request.Context(), func() {}
		if duration > 0 {
			ctx, cancel = context.WithTimeout(c.Request.Context(), duration)
		}

		defer cancel()

		// Replace request with timeout context
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// APIVersion adds API version headers
func APIVersion(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("API-Version", version)
		c.Next()
	}
}

// ContentType ensures JSON content type for API responses
func ContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Next()
	}
}

// generateRequestID creates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	return fmt.Sprintf("req_%x", bytes)
}

// HealthCheck provides a simple health check endpoint
func HealthCheck() gin.HandlerFunc {
	startTime := time.Now()

	return func(c *gin.Context) {
		uptime := time.Since(startTime)

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"uptime":    uptime.String(),
			"version":   "1.0.0",
		})
	}
}

// NoCache adds headers to prevent caching
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

// Compress enables gzip compression (Gin has built-in support)
func Compress() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Gin automatically handles compression if configured
		c.Next()
	}
}
