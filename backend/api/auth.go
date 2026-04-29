package api

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal(`[SECURITY] JWT_SECRET environment variable is required.
Set a strong random secret:
  export JWT_SECRET=$(openssl rand -base64 32)
Or add it to config.yaml or .env file.

WARNING: Using the default secret is INSECURE. Anyone who knows it can forge tokens.`)
	}
	jwtSecret = []byte(secret)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username, role string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),  // Reduced from 72h to 24h
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "deepseek-monitor",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// AuthRequired is a Gin middleware that checks JWT in Authorization header
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "authorization header required"})
			return
		}

		var tokenStr string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		} else {
			tokenStr = authHeader
		}

		claims, err := ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	id, _ := c.Get("user_id")
	if uid, ok := id.(uint); ok {
		return uid
	}
	return 0
}
