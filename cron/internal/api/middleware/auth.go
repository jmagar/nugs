package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth creates a JWT authentication middleware
func JWTAuth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_TOKEN",
					"message": "Authorization token is required",
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Check Bearer prefix
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TOKEN_FORMAT",
					"message": "Authorization token must be in format: Bearer <token>",
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secretKey), nil
		})

		if err != nil {
			var errorCode, errorMessage string

			switch {
			case jwt.ErrSignatureInvalid.Error() == err.Error():
				errorCode = "INVALID_SIGNATURE"
				errorMessage = "Invalid token signature"
			case jwt.ErrTokenExpired.Error() == err.Error():
				errorCode = "TOKEN_EXPIRED"
				errorMessage = "Token has expired"
			case jwt.ErrTokenNotValidYet.Error() == err.Error():
				errorCode = "TOKEN_NOT_VALID_YET"
				errorMessage = "Token is not valid yet"
			default:
				errorCode = "INVALID_TOKEN"
				errorMessage = "Invalid token"
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    errorCode,
					"message": errorMessage,
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_CLAIMS",
					"message": "Invalid token claims",
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)

		c.Next()
	}
}

// RequireRole creates a middleware that requires specific user role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NO_ROLE",
					"message": "User role not found",
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok || userRole != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_PRIVILEGES",
					"message": "Insufficient privileges for this operation",
					"details": gin.H{
						"required_role": requiredRole,
						"user_role":     userRole,
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

// RequireAnyRole creates a middleware that requires any of the specified roles
func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NO_ROLE",
					"message": "User role not found",
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ROLE",
					"message": "Invalid user role",
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			if userRole == requiredRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_PRIVILEGES",
					"message": "Insufficient privileges for this operation",
					"details": gin.H{
						"required_roles": roles,
						"user_role":      userRole,
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

// GenerateToken generates a JWT token for a user
func GenerateToken(userID int, username, role, secretKey string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "nugs-api",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// RefreshToken creates a new token with extended expiration
func RefreshToken(tokenString, secretKey string) (string, error) {
	// Parse existing token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", jwt.ErrInvalidKey
	}

	// Create new token with extended expiration
	return GenerateToken(claims.UserID, claims.Username, claims.Role, secretKey)
}
