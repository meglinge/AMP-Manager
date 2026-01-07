package middleware

import (
	"net/http"
	"strings"

	"ampmanager/internal/repository"
	"ampmanager/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyUsername = "username"
	ContextKeyIsAdmin  = "is_admin"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	jwtService := service.NewJWTService()

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 Authorization 头"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization 格式错误"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "Token 验证失败"
			if err == service.ErrExpiredToken {
				msg = "Token 已过期"
			}
			c.JSON(status, gin.H{"error": msg})
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	userRepo := repository.NewUserRepository()

	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		user, err := userRepo.GetByID(userID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		if !user.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		c.Set(ContextKeyIsAdmin, true)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	userID, _ := c.Get(ContextKeyUserID)
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

func GetUsername(c *gin.Context) string {
	username, _ := c.Get(ContextKeyUsername)
	if name, ok := username.(string); ok {
		return name
	}
	return ""
}

func IsAdmin(c *gin.Context) bool {
	v, exists := c.Get(ContextKeyIsAdmin)
	if !exists {
		return false
	}
	isAdmin, ok := v.(bool)
	return ok && isAdmin
}
