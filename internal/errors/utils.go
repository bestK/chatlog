package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
)

// WrapIfErr 如果 err 不为 nil，则包装错误并返回，否则返回 nil
func WrapIfErr(err error, errType, message string, code int) error {
	if err == nil {
		return nil
	}
	return Wrap(err, errType, message, code)
}

// JoinErrors 将多个错误合并为一个错误
// 如果只有一个错误不为 nil，则返回该错误
// 如果有多个错误不为 nil，则创建一个包含所有错误信息的新错误
func JoinErrors(errs ...error) error {
	var nonNilErrs []error
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	if len(nonNilErrs) == 0 {
		return nil
	}

	if len(nonNilErrs) == 1 {
		return nonNilErrs[0]
	}

	// 合并多个错误
	var messages []string
	for _, err := range nonNilErrs {
		messages = append(messages, err.Error())
	}

	return Internal(
		fmt.Sprintf("multiple errors occurred: %s", strings.Join(messages, "; ")),
		nonNilErrs[0],
	)
}

// IsNil 检查错误是否为 nil
func IsNil(err error) bool {
	return err == nil
}

// IsNotNil 检查错误是否不为 nil
func IsNotNil(err error) bool {
	return err != nil
}

// IsType 检查错误是否为指定类型
func IsType(err error, errType string) bool {
	return Is(err, errType)
}

// HasCause 检查错误是否包含指定的原因
func HasCause(err error, cause error) bool {
	if err == nil || cause == nil {
		return false
	}

	var appErr *AppError
	if stderrors.As(err, &appErr) {
		if appErr.Cause == cause {
			return true
		}
		return HasCause(appErr.Cause, cause)
	}

	return err == cause
}

// AsAppError 将错误转换为 AppError 类型
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// FormatErrorChain 格式化错误链，便于调试
func FormatErrorChain(err error) string {
	if err == nil {
		return "<nil>"
	}

	var result strings.Builder
	result.WriteString(err.Error())

	// 获取 AppError 类型的堆栈信息
	var appErr *AppError
	if stderrors.As(err, &appErr) && len(appErr.Stack) > 0 {
		result.WriteString("\nStack Trace:\n")
		for _, frame := range appErr.Stack {
			result.WriteString("  ")
			result.WriteString(frame)
			result.WriteString("\n")
		}
	}

	// 递归处理错误链
	cause := stderrors.Unwrap(err)
	if cause != nil {
		result.WriteString("\nCaused by: ")
		result.WriteString(FormatErrorChain(cause))
	}

	return result.String()
}

// GetErrorDetails 返回错误的详细信息，包括类型、消息、HTTP状态码和请求ID
func GetErrorDetails(err error) (errType string, message string, code int, requestID string) {
	if err == nil {
		return "", "", 0, ""
	}

	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr.Type, appErr.Message, appErr.Code, appErr.RequestID
	}

	return "unknown", err.Error(), 500, ""
}
