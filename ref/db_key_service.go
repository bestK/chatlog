package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// DbKeyResult 数据库密钥获取结果
type DbKeyResult struct {
	Success bool
	Key     string
	Error   string
}

// HookController 封装 wx_key.dll 的调用
type HookController struct {
	dll              *syscall.DLL
	initializeHook   *syscall.Proc
	pollKeyData      *syscall.Proc
	getStatusMessage *syscall.Proc
	cleanupHook      *syscall.Proc
	getLastErrorMsg  *syscall.Proc
	initialized      bool
}

// NewHookController 创建新的 HookController 实例
func NewHookController(dllPath string) (*HookController, error) {
	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		return nil, fmt.Errorf("加载 DLL 失败: %v", err)
	}

	hc := &HookController{dll: dll}

	// 查找导出函数
	hc.initializeHook, err = dll.FindProc("InitializeHook")
	if err != nil {
		dll.Release()
		return nil, fmt.Errorf("找不到 InitializeHook 函数: %v", err)
	}

	hc.pollKeyData, err = dll.FindProc("PollKeyData")
	if err != nil {
		dll.Release()
		return nil, fmt.Errorf("找不到 PollKeyData 函数: %v", err)
	}

	hc.getStatusMessage, err = dll.FindProc("GetStatusMessage")
	if err != nil {
		dll.Release()
		return nil, fmt.Errorf("找不到 GetStatusMessage 函数: %v", err)
	}

	hc.cleanupHook, err = dll.FindProc("CleanupHook")
	if err != nil {
		dll.Release()
		return nil, fmt.Errorf("找不到 CleanupHook 函数: %v", err)
	}

	hc.getLastErrorMsg, err = dll.FindProc("GetLastErrorMsg")
	if err != nil {
		dll.Release()
		return nil, fmt.Errorf("找不到 GetLastErrorMsg 函数: %v", err)
	}

	return hc, nil
}

// Initialize 初始化 Hook
func (hc *HookController) Initialize(targetPid uint32) bool {
	ret, _, _ := hc.initializeHook.Call(uintptr(targetPid))
	hc.initialized = ret != 0
	return hc.initialized
}

// PollKeyData 轮询密钥数据
func (hc *HookController) PollKeyData() (string, bool) {
	buffer := make([]byte, 65)
	ret, _, _ := hc.pollKeyData.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)

	if ret == 0 {
		return "", false
	}

	// 找到字符串结束位置
	end := 0
	for i, b := range buffer {
		if b == 0 {
			end = i
			break
		}
		end = i + 1
	}

	return string(buffer[:end]), true
}

// GetStatusMessage 获取状态消息
func (hc *HookController) GetStatusMessage() (message string, level int, hasMessage bool) {
	statusBuffer := make([]byte, 256)
	var outLevel int32

	ret, _, _ := hc.getStatusMessage.Call(
		uintptr(unsafe.Pointer(&statusBuffer[0])),
		uintptr(len(statusBuffer)),
		uintptr(unsafe.Pointer(&outLevel)),
	)

	if ret == 0 {
		return "", 0, false
	}

	// 找到字符串结束位置
	end := 0
	for i, b := range statusBuffer {
		if b == 0 {
			end = i
			break
		}
		end = i + 1
	}

	return string(statusBuffer[:end]), int(outLevel), true
}

// Cleanup 清理 Hook
func (hc *HookController) Cleanup() bool {
	if !hc.initialized {
		return true
	}
	ret, _, _ := hc.cleanupHook.Call()
	hc.initialized = false
	return ret != 0
}

// GetLastError 获取最后的错误信息
func (hc *HookController) GetLastError() string {
	ptr, _, _ := hc.getLastErrorMsg.Call()
	if ptr == 0 {
		return ""
	}

	// 读取 C 字符串
	var result []byte
	for i := 0; ; i++ {
		b := *(*byte)(unsafe.Pointer(ptr + uintptr(i)))
		if b == 0 {
			break
		}
		result = append(result, b)
		if i > 512 { // 防止无限循环
			break
		}
	}

	return string(result)
}

// Release 释放 DLL
func (hc *HookController) Release() {
	if hc.initialized {
		hc.Cleanup()
	}
	if hc.dll != nil {
		hc.dll.Release()
	}
}

// ProcessStatusMessages 处理所有待处理的状态消息
func (hc *HookController) ProcessStatusMessages(onStatus func(message string, level int)) {
	for i := 0; i < 10; i++ { // 最多处理10条
		msg, level, hasMsg := hc.GetStatusMessage()
		if !hasMsg {
			break
		}
		if onStatus != nil {
			onStatus(msg, level)
		}
	}
}

// ========== 微信进程管理 ==========

// KillWeChatProcesses 关闭所有微信进程
func KillWeChatProcesses() error {
	processNames := []string{"Weixin.exe", "WeChat.exe"}

	for _, name := range processNames {
		pids, _ := FindProcessIdsByName(name)
		for _, pid := range pids {
			hProcess, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
			if err != nil {
				continue
			}
			windows.TerminateProcess(hProcess, 0)
			windows.CloseHandle(hProcess)
		}
	}

	return nil
}

// GetWeChatPath 获取微信安装路径
func GetWeChatPath() string {
	// 1. 从注册表查找
	registryPaths := []struct {
		key  registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\Tencent\WeChat`},
		{registry.CURRENT_USER, `Software\Tencent\Weixin`},
		{registry.LOCAL_MACHINE, `SOFTWARE\Tencent\WeChat`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Tencent\WeChat`},
	}

	for _, rp := range registryPaths {
		k, err := registry.OpenKey(rp.key, rp.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer k.Close()

		for _, valName := range []string{"InstallPath", "Install", "Path"} {
			val, _, err := k.GetStringValue(valName)
			if err == nil && val != "" {
				// 检查是否是 exe 路径
				if strings.HasSuffix(strings.ToLower(val), ".exe") {
					if _, err := os.Stat(val); err == nil {
						return val
					}
				}
				// 尝试拼接 exe
				for _, exeName := range []string{"Weixin.exe", "WeChat.exe"} {
					exePath := filepath.Join(val, exeName)
					if _, err := os.Stat(exePath); err == nil {
						return exePath
					}
				}
			}
		}
	}

	// 2. 尝试常见路径
	drives := []string{"C", "D", "E", "F"}
	commonPaths := []string{
		`\Program Files\Tencent\WeChat\WeChat.exe`,
		`\Program Files (x86)\Tencent\WeChat\WeChat.exe`,
		`\Program Files\Tencent\Weixin\Weixin.exe`,
		`\Program Files (x86)\Tencent\Weixin\Weixin.exe`,
	}

	for _, drive := range drives {
		for _, path := range commonPaths {
			fullPath := drive + ":" + path
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	return ""
}

// LaunchWeChat 启动微信
func LaunchWeChat() error {
	wechatPath := GetWeChatPath()
	if wechatPath == "" {
		return fmt.Errorf("未找到微信安装路径")
	}

	cmd := exec.Command(wechatPath)
	cmd.Dir = filepath.Dir(wechatPath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("启动微信失败: %v", err)
	}

	return nil
}

// WaitForWeChatProcess 等待微信进程出现
func WaitForWeChatProcess(maxWaitSeconds int) (uint32, bool) {
	deadline := time.Now().Add(time.Duration(maxWaitSeconds) * time.Second)

	for time.Now().Before(deadline) {
		for _, name := range []string{"Weixin.exe", "WeChat.exe"} {
			pids, _ := FindProcessIdsByName(name)
			if len(pids) > 0 {
				return pids[0], true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return 0, false
}

// ========== 完整的数据库密钥获取流程 ==========

// GetDbKeyFull 完整的数据库密钥获取流程（按原版逻辑）
// 1. 关闭微信
// 2. 启动微信
// 3. 等待窗口
// 4. 安装 Hook
// 5. 等待登录获取密钥
func GetDbKeyFull(dllPath string, timeout int, onProgress func(string)) DbKeyResult {
	// 状态消息处理回调
	statusHandler := func(msg string, level int) {
		prefix := "[*]"
		switch level {
		case 1:
			prefix = "[+]"
		case 2:
			prefix = "[!]"
		}
		if onProgress != nil {
			onProgress(fmt.Sprintf("%s %s", prefix, msg))
		}
	}

	// 1. 检查 DLL
	if onProgress != nil {
		onProgress("正在加载 Hook DLL...")
	}

	hc, err := NewHookController(dllPath)
	if err != nil {
		return DbKeyResult{Success: false, Error: err.Error()}
	}
	defer hc.Release()

	// 2. 检查微信是否运行，如果运行则关闭
	if onProgress != nil {
		onProgress("正在检查微信进程...")
	}

	if IsProcessRunning("Weixin.exe") || IsProcessRunning("WeChat.exe") {
		if onProgress != nil {
			onProgress("检测到微信正在运行，正在关闭...")
		}
		KillWeChatProcesses()
		time.Sleep(2 * time.Second)
		if onProgress != nil {
			onProgress("已关闭微信进程")
		}
	}

	// 3. 启动微信
	if onProgress != nil {
		onProgress("正在启动微信...")
	}

	err = LaunchWeChat()
	if err != nil {
		return DbKeyResult{Success: false, Error: err.Error()}
	}

	if onProgress != nil {
		onProgress("微信启动成功，等待窗口出现...")
	}

	// 4. 等待微信进程出现
	pid, found := WaitForWeChatProcess(15)
	if !found {
		return DbKeyResult{Success: false, Error: "等待微信窗口超时"}
	}

	if onProgress != nil {
		onProgress(fmt.Sprintf("找到微信进程 PID: %d", pid))
	}

	// 等待微信界面加载
	time.Sleep(3 * time.Second)

	// 5. 初始化并安装 Hook
	if onProgress != nil {
		onProgress("正在安装远程 Hook...")
	}

	if !hc.Initialize(pid) {
		hc.ProcessStatusMessages(statusHandler)
		errMsg := hc.GetLastError()
		if errMsg == "" {
			errMsg = "初始化 Hook 失败"
		}
		return DbKeyResult{Success: false, Error: errMsg}
	}

	// 处理初始化阶段的状态消息
	hc.ProcessStatusMessages(statusHandler)

	if onProgress != nil {
		onProgress("Hook 安装成功！")
		onProgress("请在微信中登录账号，密钥将在登录时自动获取...")
	}

	// 6. 轮询等待密钥
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		// 处理状态消息
		hc.ProcessStatusMessages(statusHandler)

		// 检查是否有新密钥
		key, hasKey := hc.PollKeyData()
		if hasKey && len(key) > 0 {
			if onProgress != nil {
				onProgress("成功获取数据库密钥！")
			}
			hc.Cleanup()
			return DbKeyResult{Success: true, Key: key}
		}

		time.Sleep(pollInterval)
	}

	// 超时
	hc.Cleanup()
	return DbKeyResult{
		Success: false,
		Error:   fmt.Sprintf("等待密钥超时（%d秒）。请确保在微信中完成登录操作。", timeout),
	}
}

// GetDbKey 获取数据库密钥（简化版，不重启微信）
func GetDbKey(dllPath string, pid uint32, timeout int, onProgress func(string)) DbKeyResult {
	if onProgress != nil {
		onProgress("正在加载 Hook DLL...")
	}

	hc, err := NewHookController(dllPath)
	if err != nil {
		return DbKeyResult{Success: false, Error: err.Error()}
	}
	defer hc.Release()

	if onProgress != nil {
		onProgress(fmt.Sprintf("正在初始化 Hook (PID: %d)...", pid))
	}

	// 状态消息处理回调
	statusHandler := func(msg string, level int) {
		prefix := "[*]"
		switch level {
		case 1:
			prefix = "[+]"
		case 2:
			prefix = "[!]"
		}
		if onProgress != nil {
			onProgress(fmt.Sprintf("%s %s", prefix, msg))
		}
	}

	// 初始化 Hook
	if !hc.Initialize(pid) {
		hc.ProcessStatusMessages(statusHandler)
		errMsg := hc.GetLastError()
		if errMsg == "" {
			errMsg = "初始化 Hook 失败"
		}
		return DbKeyResult{Success: false, Error: errMsg}
	}

	// 处理初始化阶段的状态消息
	hc.ProcessStatusMessages(statusHandler)

	if onProgress != nil {
		onProgress("Hook 安装成功，等待密钥...")
		onProgress("请在微信中登录账号...")
	}

	// 轮询等待密钥
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		// 处理状态消息
		hc.ProcessStatusMessages(statusHandler)

		// 检查是否有新密钥
		key, hasKey := hc.PollKeyData()
		if hasKey && len(key) > 0 {
			if onProgress != nil {
				onProgress("成功获取数据库密钥！")
			}
			hc.Cleanup()
			return DbKeyResult{Success: true, Key: key}
		}

		time.Sleep(pollInterval)
	}

	// 超时
	hc.Cleanup()
	return DbKeyResult{
		Success: false,
		Error:   fmt.Sprintf("等待密钥超时（%d秒）。请确保在微信中完成登录操作。", timeout),
	}
}
