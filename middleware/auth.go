package middleware

import (
	"new/errors"
	"new/response"
	"new/services"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware xử lý authentication
func AuthMiddleware(roles ...int) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		userID, userRole, err := services.GetUserIDFromToken(tokenString)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// Kiểm tra role nếu có yêu cầu
		if len(roles) > 0 {
			hasRole := false
			for _, role := range roles {
				if role == userRole {
					hasRole = true
					break
				}
			}
			if !hasRole {
				response.Forbidden(c)
				c.Abort()
				return
			}
		}

		// Lưu thông tin user vào context
		c.Set("userID", userID)
		c.Set("userRole", userRole)
		c.Next()
	}
}

// RoleMiddleware kiểm tra role của user
func RoleMiddleware(roles ...int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		role := userRole.(int)
		hasRole := false
		for _, r := range roles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			response.Forbidden(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

// ErrorHandler xử lý lỗi
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Kiểm tra lỗi
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			if appErr, ok := err.(*errors.AppError); ok {
				response.Error(c, 0, appErr.Message)
				return
			}

			response.ServerError(c)
		}
	}
}
