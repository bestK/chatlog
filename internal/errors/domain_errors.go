package errors

import (
	"fmt"
	"net/http"
)

// 微信相关错误

// WeChatProcessNotFound 创建微信进程未找到错误
func WeChatProcessNotFound() *AppError {
	return New(ErrTypeWeChat, "wechat process not found", nil, http.StatusNotFound).WithStack()
}

// WeChatKeyExtractFailed 创建微信密钥提取失败错误
func WeChatKeyExtractFailed(cause error) *AppError {
	return New(ErrTypeWeChat, "failed to extract wechat key", cause, http.StatusInternalServerError).WithStack()
}

// WeChatDecryptFailed 创建微信解密失败错误
func WeChatDecryptFailed(cause error) *AppError {
	return New(ErrTypeWeChat, "failed to decrypt wechat database", cause, http.StatusInternalServerError).WithStack()
}

// WeChatAccountNotSelected 创建未选择微信账号错误
func WeChatAccountNotSelected() *AppError {
	return New(ErrTypeWeChat, "no wechat account selected", nil, http.StatusBadRequest).WithStack()
}

// 数据库相关错误

// DBConnectionFailed 创建数据库连接失败错误
func DBConnectionFailed(cause error) *AppError {
	return New(ErrTypeDatabase, "database connection failed", cause, http.StatusInternalServerError).WithStack()
}

// DBQueryFailed 创建数据库查询失败错误
func DBQueryFailed(operation string, cause error) *AppError {
	return New(ErrTypeDatabase, fmt.Sprintf("database query failed: %s", operation), cause, http.StatusInternalServerError).WithStack()
}

// DBRecordNotFound 创建数据库记录未找到错误
func DBRecordNotFound(resource string) *AppError {
	return New(ErrTypeNotFound, fmt.Sprintf("record not found: %s", resource), nil, http.StatusNotFound).WithStack()
}

// 配置相关错误

// ConfigInvalid 创建配置无效错误
func ConfigInvalid(field string, cause error) *AppError {
	return New(ErrTypeConfig, fmt.Sprintf("invalid configuration: %s", field), cause, http.StatusInternalServerError).WithStack()
}

// ConfigMissing 创建配置缺失错误
func ConfigMissing(field string) *AppError {
	return New(ErrTypeConfig, fmt.Sprintf("missing configuration: %s", field), nil, http.StatusBadRequest).WithStack()
}

// 平台相关错误

// PlatformUnsupported 创建不支持的平台错误
func PlatformUnsupported(platform string, version int) *AppError {
	return New(ErrTypeInvalidArg, fmt.Sprintf("unsupported platform: %s v%d", platform, version), nil, http.StatusBadRequest).WithStack()
}

// 文件系统错误

// FileNotFound 创建文件未找到错误
func FileNotFound(path string) *AppError {
	return New(ErrTypeNotFound, fmt.Sprintf("file not found: %s", path), nil, http.StatusNotFound).WithStack()
}

// FileReadFailed 创建文件读取失败错误
func FileReadFailed(path string, cause error) *AppError {
	return New(ErrTypeInternal, fmt.Sprintf("failed to read file: %s", path), cause, http.StatusInternalServerError).WithStack()
}

// FileWriteFailed 创建文件写入失败错误
func FileWriteFailed(path string, cause error) *AppError {
	return New(ErrTypeInternal, fmt.Sprintf("failed to write file: %s", path), cause, http.StatusInternalServerError).WithStack()
}

// 参数验证错误

// RequiredParam 创建必需参数缺失错误
func RequiredParam(param string) *AppError {
	return New(ErrTypeInvalidArg, fmt.Sprintf("required parameter missing: %s", param), nil, http.StatusBadRequest).WithStack()
}

// InvalidParam 创建参数无效错误
func InvalidParam(param string, reason string) *AppError {
	message := fmt.Sprintf("invalid parameter: %s", param)
	if reason != "" {
		message = fmt.Sprintf("%s (%s)", message, reason)
	}
	return New(ErrTypeInvalidArg, message, nil, http.StatusBadRequest).WithStack()
}

// 解密相关错误

// DecryptInvalidKey 创建无效密钥格式错误
func DecryptInvalidKey(cause error) *AppError {
	return New(ErrTypeWeChat, "invalid key format", cause, http.StatusBadRequest).
		WithStack()
}

// DecryptCreateCipherFailed 创建无法创建加密器错误
func DecryptCreateCipherFailed(cause error) *AppError {
	return New(ErrTypeWeChat, "failed to create cipher", cause, http.StatusInternalServerError).
		WithStack()
}

// DecryptDecodeKeyFailed 创建无法解码十六进制密钥错误
func DecryptDecodeKeyFailed(cause error) *AppError {
	return New(ErrTypeWeChat, "failed to decode hex key", cause, http.StatusBadRequest).
		WithStack()
}

// DecryptWriteOutputFailed 创建无法写入输出错误
func DecryptWriteOutputFailed(cause error) *AppError {
	return New(ErrTypeWeChat, "failed to write decryption output", cause, http.StatusInternalServerError).
		WithStack()
}

// DecryptOperationCanceled 创建解密操作被取消错误
func DecryptOperationCanceled() *AppError {
	return New(ErrTypeWeChat, "decryption operation was canceled", nil, http.StatusBadRequest).
		WithStack()
}

// DecryptOpenFileFailed 创建无法打开数据库文件错误
func DecryptOpenFileFailed(path string, cause error) *AppError {
	return New(ErrTypeWeChat, fmt.Sprintf("failed to open database file: %s", path), cause, http.StatusInternalServerError).
		WithStack()
}

// DecryptReadFileFailed 创建无法读取数据库文件错误
func DecryptReadFileFailed(path string, cause error) *AppError {
	return New(ErrTypeWeChat, fmt.Sprintf("failed to read database file: %s", path), cause, http.StatusInternalServerError).
		WithStack()
}

// DecryptIncompleteRead 创建不完整的头部读取错误
func DecryptIncompleteRead(cause error) *AppError {
	return New(ErrTypeWeChat, "incomplete header read during decryption", cause, http.StatusInternalServerError).
		WithStack()
}

var ErrAlreadyDecrypted = New(ErrTypeWeChat, "database file is already decrypted", nil, http.StatusBadRequest)
var ErrDecryptHashVerificationFailed = New(ErrTypeWeChat, "hash verification failed during decryption", nil, http.StatusBadRequest)
var ErrDecryptIncorrectKey = New(ErrTypeWeChat, "incorrect decryption key", nil, http.StatusBadRequest)
