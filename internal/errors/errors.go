package errors

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 定义错误类型常量
const (
	ErrTypeDatabase   = "database"
	ErrTypeWeChat     = "wechat"
	ErrTypeHTTP       = "http"
	ErrTypeConfig     = "config"
	ErrTypeInvalidArg = "invalid_argument"
)

// AppError 表示应用程序错误
type AppError struct {
	Type    string `json:"type"`    // 错误类型
	Message string `json:"message"` // 错误消息
	Cause   error  `json:"-"`       // 原始错误
	Code    int    `json:"-"`       // HTTP Code
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AppError) String() string {
	return e.Error()
}

// New 创建新的应用错误
func New(errType, message string, cause error, code int) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Cause:   cause,
		Code:    code,
	}
}

// ErrInvalidArg 无效参数错误
func ErrInvalidArg(param string) *AppError {
	return New(ErrTypeInvalidArg, fmt.Sprintf("invalid arg: %s", param), nil, http.StatusBadRequest)
}

// Database 创建数据库错误
func Database(message string, cause error) *AppError {
	return New(ErrTypeDatabase, message, cause, http.StatusInternalServerError)
}

// WeChat 创建微信相关错误
func WeChat(message string, cause error) *AppError {
	return New(ErrTypeWeChat, message, cause, http.StatusInternalServerError)
}

// HTTP 创建HTTP服务错误
func HTTP(message string, cause error) *AppError {
	return New(ErrTypeHTTP, message, cause, http.StatusInternalServerError)
}

// Config 创建配置错误
func Config(message string, cause error) *AppError {
	return New(ErrTypeConfig, message, cause, http.StatusInternalServerError)
}

// Err 在HTTP响应中返回错误
func Err(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.Code, appErr)
		return
	}

	// 未知错误
	c.JSON(http.StatusInternalServerError, gin.H{
		"type":    "unknown",
		"message": err.Error(),
	})
}
