package errors

import (
	"fmt"
	"net/http"
	"testing"
)

func TestErrorCreation(t *testing.T) {
	// 测试创建基本错误
	err := New("test", "test message", nil, http.StatusBadRequest)
	if err.Type != "test" || err.Message != "test message" || err.Code != http.StatusBadRequest {
		t.Errorf("New() created incorrect error: %v", err)
	}

	// 测试创建带原因的错误
	cause := fmt.Errorf("original error")
	err = New("test", "test with cause", cause, http.StatusInternalServerError)
	if err.Cause != cause {
		t.Errorf("New() did not set cause correctly: %v", err)
	}

	// 测试错误消息格式
	expected := "test: test with cause: original error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestErrorWrapping(t *testing.T) {
	// 测试包装普通错误
	original := fmt.Errorf("original error")
	wrapped := Wrap(original, "wrapped", "wrapped message", http.StatusBadRequest)

	if wrapped.Type != "wrapped" || wrapped.Message != "wrapped message" {
		t.Errorf("Wrap() created incorrect error: %v", wrapped)
	}

	if wrapped.Cause != original {
		t.Errorf("Wrap() did not set cause correctly")
	}

	// 测试包装 AppError
	appErr := New("app", "app error", nil, http.StatusNotFound)
	rewrapped := Wrap(appErr, "ignored", "new message", http.StatusBadRequest)

	if rewrapped.Type != "app" {
		t.Errorf("Wrap() did not preserve original AppError type: got %s, want %s",
			rewrapped.Type, appErr.Type)
	}

	if rewrapped.Message != "new message" {
		t.Errorf("Wrap() did not update message: got %s, want %s",
			rewrapped.Message, "new message")
	}

	if rewrapped.Code != appErr.Code {
		t.Errorf("Wrap() did not preserve original status code: got %d, want %d",
			rewrapped.Code, appErr.Code)
	}
}

func TestErrorTypeChecking(t *testing.T) {
	// 创建不同类型的错误
	dbErr := Database("db error", nil)
	httpErr := HTTP("http error", nil)

	// 测试 Is 函数
	if !Is(dbErr, ErrTypeDatabase) {
		t.Errorf("Is() failed to identify database error")
	}

	if Is(dbErr, ErrTypeHTTP) {
		t.Errorf("Is() incorrectly identified database error as HTTP error")
	}

	if !Is(httpErr, ErrTypeHTTP) {
		t.Errorf("Is() failed to identify HTTP error")
	}

	// 测试 GetType 函数
	if GetType(dbErr) != ErrTypeDatabase {
		t.Errorf("GetType() returned incorrect type: got %s, want %s",
			GetType(dbErr), ErrTypeDatabase)
	}

	if GetType(httpErr) != ErrTypeHTTP {
		t.Errorf("GetType() returned incorrect type: got %s, want %s",
			GetType(httpErr), ErrTypeHTTP)
	}

	// 测试普通错误
	stdErr := fmt.Errorf("standard error")
	if GetType(stdErr) != "unknown" {
		t.Errorf("GetType() for standard error should return 'unknown', got %s",
			GetType(stdErr))
	}
}

func TestErrorUnwrapping(t *testing.T) {
	// 创建嵌套错误
	innermost := fmt.Errorf("innermost error")
	inner := Wrap(innermost, "inner", "inner error", http.StatusBadRequest)
	outer := Wrap(inner, "outer", "outer error", http.StatusInternalServerError)

	// 测试 Unwrap
	if unwrapped := outer.Unwrap(); unwrapped != inner.Cause {
		t.Errorf("Unwrap() did not return correct inner error")
	}

	// 测试 RootCause
	if root := RootCause(outer); root != innermost {
		t.Errorf("RootCause() did not return innermost error")
	}
}

func TestErrorHelperFunctions(t *testing.T) {
	// 测试辅助函数
	invalidArg := ErrInvalidArg("username")
	if invalidArg.Type != ErrTypeInvalidArg {
		t.Errorf("ErrInvalidArg() created error with wrong type: %s", invalidArg.Type)
	}

	dbErr := Database("query failed", nil)
	if dbErr.Type != ErrTypeDatabase {
		t.Errorf("Database() created error with wrong type: %s", dbErr.Type)
	}

	notFound := NotFound("user", nil)
	if notFound.Type != ErrTypeNotFound || notFound.Code != http.StatusNotFound {
		t.Errorf("NotFound() created error with wrong type or code: %s, %d",
			notFound.Type, notFound.Code)
	}
}

func TestErrorUtilityFunctions(t *testing.T) {
	// 测试 JoinErrors
	err1 := fmt.Errorf("error 1")
	err2 := fmt.Errorf("error 2")

	// 单个错误
	if joined := JoinErrors(err1); joined != err1 {
		t.Errorf("JoinErrors() with single error should return that error")
	}

	// 多个错误
	joined := JoinErrors(err1, err2)
	if joined == nil {
		t.Errorf("JoinErrors() returned nil for multiple errors")
	}

	// nil 错误
	if joined := JoinErrors(nil, nil); joined != nil {
		t.Errorf("JoinErrors() with all nil should return nil")
	}

	// 测试 WrapIfErr
	if wrapped := WrapIfErr(nil, "test", "message", http.StatusOK); wrapped != nil {
		t.Errorf("WrapIfErr() with nil should return nil")
	}

	if wrapped := WrapIfErr(err1, "test", "message", http.StatusBadRequest); wrapped == nil {
		t.Errorf("WrapIfErr() with non-nil error should return non-nil")
	}
}
