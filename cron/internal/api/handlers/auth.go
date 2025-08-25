package handlers

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	DB        *sql.DB
	JWTSecret []byte
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         User   `json:"user,omitempty"`
	Error        string `json:"error,omitempty"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthHandler(db *sql.DB, jwtSecret []byte) *AuthHandler {
	return &AuthHandler{
		DB:        db,
		JWTSecret: jwtSecret,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// Get user from database
	var user User
	var passwordHash string
	err := h.DB.QueryRow(`
		SELECT id, username, email, password_hash, role, active 
		FROM users 
		WHERE username = ? AND active = true
	`, req.Username).Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.Role, &user.Active)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, LoginResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, LoginResponse{
			Success: false,
			Error:   fmt.Sprintf("Database error: %v", err),
		})
		return
	}

	// Check password using SHA256 (note: bcrypt would be better in production)
	hasher := sha256.New()
	hasher.Write([]byte(req.Password))
	inputHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if inputHash != passwordHash {
		c.JSON(http.StatusUnauthorized, LoginResponse{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Generate JWT token
	token, err := h.generateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, LoginResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	// Update last login
	_, err = h.DB.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?", user.ID)
	if err != nil {
		// Log error but don't fail the login
	}

	c.JSON(http.StatusOK, LoginResponse{
		Success: true,
		Token:   token,
		User:    user,
	})
}

func (h *AuthHandler) generateJWT(user User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.JWTSecret)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// For JWT tokens, logout is typically handled client-side
	// by removing the token from storage
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

func (h *AuthHandler) Verify(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "No user context",
		})
		return
	}

	// Get user details from database
	var user User
	err := h.DB.QueryRow(`
		SELECT id, username, email, role, active 
		FROM users 
		WHERE id = ? AND active = true
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Active)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
	})
}
