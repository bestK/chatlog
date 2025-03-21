package errors

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// 定义错误类型常量
const (
	ErrTypeDatabase   = "database"
	ErrTypeWeChat     = "wechat"
	ErrTypeHTTP       = "http"
	ErrTypeConfig     = "config"
	ErrTypeInvalidArg = "invalid_argument"
	ErrTypeAuth       = "authentication"
	ErrTypePermission = "permission"
	ErrTypeNotFound   = "not_found"
	ErrTypeValidation = "validation"
	ErrTypeRateLimit  = "rate_limit"
	ErrTypeInternal   = "internal"
)

// AppError 表示应用程序错误
type AppError struct {
	Type      string   `json:"type"`                 // 错误类型
	Message   string   `json:"message"`              // 错误消息
	Cause     error    `json:"-"`                    // 原始错误
	Code      int      `json:"-"`                    // HTTP Code
	Stack     []string `json:"-"`                    // 错误堆栈
	RequestID string   `json:"request_id,omitempty"` // 请求ID，用于跟踪
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// String 返回错误的字符串表示
func (e *AppError) String() string {
	return e.Error()
}

// Unwrap 实现 errors.Unwrap 接口，用于错误链
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithStack 添加堆栈信息到错误
func (e *AppError) WithStack() *AppError {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	stack := make([]string, 0, n)
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		}
		if !more {
			break
		}
	}

	e.Stack = stack
	return e
}

// WithRequestID 添加请求ID到错误
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
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

// Wrap 包装现有错误为 AppError
func Wrap(err error, errType, message string, code int) *AppError {
	if err == nil {
		return nil
	}

	// 如果已经是 AppError，保留原始类型但更新消息
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Type:    appErr.Type,
			Message: message,
			Cause:   appErr.Cause,
			Code:    appErr.Code,
			Stack:   appErr.Stack,
		}
	}

	return New(errType, message, err, code)
}

// Is 检查错误是否为特定类型
func Is(err error, errType string) bool {
	if err == nil {
		return false
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errType
	}

	return false
}

// GetType 获取错误类型
func GetType(err error) string {
	if err == nil {
		return ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type
	}

	return "unknown"
}

// GetCode 获取错误的 HTTP 状态码
func GetCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	return http.StatusInternalServerError
}

// RootCause 获取错误链中的根本原因
func RootCause(err error) error {
	for err != nil {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
	return err
}

// ErrInvalidArg 无效参数错误
func ErrInvalidArg(param string) *AppError {
	return New(ErrTypeInvalidArg, fmt.Sprintf("invalid arg: %s", param), nil, http.StatusBadRequest).WithStack()
}

// Database 创建数据库错误
func Database(message string, cause error) *AppError {
	return New(ErrTypeDatabase, message, cause, http.StatusInternalServerError).WithStack()
}

// WeChat 创建微信相关错误
func WeChat(message string, cause error) *AppError {
	return New(ErrTypeWeChat, message, cause, http.StatusInternalServerError).WithStack()
}

// HTTP 创建HTTP服务错误
func HTTP(message string, cause error) *AppError {
	return New(ErrTypeHTTP, message, cause, http.StatusInternalServerError).WithStack()
}

// Config 创建配置错误
func Config(message string, cause error) *AppError {
	return New(ErrTypeConfig, message, cause, http.StatusInternalServerError).WithStack()
}

// NotFound 创建资源不存在错误
func NotFound(resource string, cause error) *AppError {
	message := fmt.Sprintf("resource not found: %s", resource)
	return New(ErrTypeNotFound, message, cause, http.StatusNotFound).WithStack()
}

// Unauthorized 创建未授权错误
func Unauthorized(message string, cause error) *AppError {
	return New(ErrTypeAuth, message, cause, http.StatusUnauthorized).WithStack()
}

// Forbidden 创建权限不足错误
func Forbidden(message string, cause error) *AppError {
	return New(ErrTypePermission, message, cause, http.StatusForbidden).WithStack()
}

// Validation 创建数据验证错误
func Validation(message string, cause error) *AppError {
	return New(ErrTypeValidation, message, cause, http.StatusBadRequest).WithStack()
}

// RateLimit 创建请求频率限制错误
func RateLimit(message string, cause error) *AppError {
	return New(ErrTypeRateLimit, message, cause, http.StatusTooManyRequests).WithStack()
}

// Internal 创建内部服务器错误
func Internal(message string, cause error) *AppError {
	return New(ErrTypeInternal, message, cause, http.StatusInternalServerError).WithStack()
}

// Err 在HTTP响应中返回错误
func Err(c *gin.Context, err error) {
	// 获取请求ID（如果有）
	requestID := c.GetString("RequestID")

	if appErr, ok := err.(*AppError); ok {
		if requestID != "" {
			appErr.RequestID = requestID
		}
		c.JSON(appErr.Code, appErr)
		return
	}

	// 未知错误
	unknownErr := &AppError{
		Type:      "unknown",
		Message:   err.Error(),
		Code:      http.StatusInternalServerError,
		RequestID: requestID,
	}
	c.JSON(http.StatusInternalServerError, unknownErr)
}
