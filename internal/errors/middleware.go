package errors

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorHandlerMiddleware 是一个 Gin 中间件，用于统一处理请求过程中的错误
// 它会为每个请求生成一个唯一的请求 ID，并在错误发生时将其添加到错误响应中
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求 ID
		requestID := uuid.New().String()
		c.Set("RequestID", requestID)
		c.Header("X-Request-ID", requestID)

		// 处理请求
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			// 获取第一个错误
			err := c.Errors[0].Err

			// 使用 Err 函数处理错误响应
			Err(c, err)

			// 已经处理过错误，不需要继续
			c.Abort()
		}
	}
}

// RecoveryMiddleware 是一个 Gin 中间件，用于从 panic 恢复并返回 500 错误
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 获取请求 ID
				requestID, _ := c.Get("RequestID")
				requestIDStr, _ := requestID.(string)

				// 创建内部服务器错误
				var err *AppError
				switch v := r.(type) {
				case error:
					err = Internal("panic recovered", v).WithRequestID(requestIDStr)
				default:
					err = Internal(fmt.Sprintf("panic recovered: %v", r), nil).WithRequestID(requestIDStr)
				}

				// 记录错误日志
				fmt.Printf("PANIC RECOVERED: %v\n", err)

				// 返回 500 错误
				c.JSON(http.StatusInternalServerError, err)
				c.Abort()
			}
		}()

		c.Next()
	}
}
