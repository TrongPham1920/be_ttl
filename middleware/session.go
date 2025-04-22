package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SessionMiddleware tạo sessionId nếu chưa có và gán vào context
func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionId := c.GetHeader("X-Session-ID")
		if sessionId == "" {
			// Tạo sessionId mới
			sessionId = uuid.NewString()
		}

		// Gán vào context để dùng trong controller hoặc service
		c.Set("sessionId", sessionId)

		// Gán lại header (optional)
		c.Writer.Header().Set("X-Session-ID", sessionId)

		c.Next()
	}
}
