package middleware

import (
	"net/http"
	"new-api-lite/config"
	"new-api-lite/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint   `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(
				time.Duration(config.C.JWT.ExpireHours) * time.Hour,
			)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.JWT.Secret))
}

func parseJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.C.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return token.Claims.(*Claims), nil
}

// AuthRequired validates the JWT and injects user into context.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		claims, err := parseJWT(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		user, err := model.GetUserByID(claims.UserID)
		if err != nil || user.Status != model.StatusEnabled {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found or disabled"})
			return
		}

		c.Set("user", user)
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminRequired ensures the authenticated user has admin role.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != model.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		c.Next()
	}
}

// GetCurrentUser is a helper to extract user from context.
func GetCurrentUser(c *gin.Context) *model.User {
	user, _ := c.Get("user")
	return user.(*model.User)
}
