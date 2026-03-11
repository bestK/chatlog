//go:build windows

package windows

import (
	"debug/pe"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/sjzar/chatlog/pkg/util"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var wechatProcessNames = []string{"Weixin.exe", "WeChat.exe", "WeChatAppEx.exe", "crashpad_handler.exe"}

var (
	user32DLL                    = windows.NewLazySystemDLL("user32.dll")
	procWaitForInputIdle         = user32DLL.NewProc("WaitForInputIdle")
	procEnumWindows              = user32DLL.NewProc("EnumWindows")
	procIsWindowVisible          = user32DLL.NewProc("IsWindowVisible")
	procGetWindowThreadProcessId = user32DLL.NewProc("GetWindowThreadProcessId")
)

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
	dll, err := loadDLLWithFallback(dllPath)
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

func loadDLLWithFallback(dllPath string) (*syscall.DLL, error) {
	if absPath, absErr := filepath.Abs(dllPath); absErr == nil {
		dllPath = absPath
	}

	pathState := dllPathState(dllPath)
	workingDir, _ := os.Getwd()
	dllArch := peMachineLabel(dllPath)
	exeArch := currentExeMachineLabel()

	dll, err := syscall.LoadDLL(dllPath)
	if err == nil {
		return dll, nil
	}
	primaryErr := describeLoadErr(err)

	dllDir := filepath.Dir(dllPath)
	if dllDir == "" || dllDir == "." {
		return nil, formatDLLLoadError(dllPath, pathState, workingDir, dllArch, exeArch, primaryErr, "", loadHint(err))
	}

	if setErr := windows.SetDllDirectory(dllDir); setErr != nil {
		return nil, formatDLLLoadError(dllPath, pathState, workingDir, dllArch, exeArch, primaryErr, describeLoadErr(setErr), loadHint(err))
	}
	defer windows.SetDllDirectory("")

	dll, retryErr := syscall.LoadDLL(dllPath)
	if retryErr != nil {
		return nil, formatDLLLoadError(dllPath, pathState, workingDir, dllArch, exeArch, primaryErr, describeLoadErr(retryErr), loadHint(retryErr))
	}

	return dll, nil
}

func formatDLLLoadError(dllPath string, pathState string, workingDir string, dllArch string, exeArch string, loadErr string, retryErr string, hint string) error {
	summary := "加载 wx_key.dll 失败"
	suggestion := "请检查 DLL 文件和依赖库是否完整。"

	switch {
	case strings.HasPrefix(pathState, "missing("):
		summary = "未找到 wx_key.dll 文件"
		suggestion = "请确认程序目录下存在 wx_key.dll，并重新启动程序。"
	case strings.Contains(hint, "dependent DLL missing"):
		summary = "wx_key.dll 依赖库缺失"
		suggestion = "请安装 VC++ 运行库，或将依赖 DLL 放到 wx_key.dll 同目录。"
	case strings.Contains(hint, "architecture mismatch"):
		summary = "wx_key.dll 与当前程序架构不匹配"
		suggestion = fmt.Sprintf("请确认程序架构为 %s，DLL 架构为 %s。", exeArch, dllArch)
	}

	details := []string{
		fmt.Sprintf("DLL 路径：%s", dllPath),
		fmt.Sprintf("路径状态：%s", pathState),
		fmt.Sprintf("当前目录：%s", workingDir),
		fmt.Sprintf("程序架构：%s", exeArch),
		fmt.Sprintf("DLL 架构：%s", dllArch),
		fmt.Sprintf("首次加载：%s", loadErr),
	}
	if retryErr != "" {
		details = append(details, fmt.Sprintf("重试加载：%s", retryErr))
	}

	return fmt.Errorf("%s\n%s\n\n诊断信息：\n%s", summary, suggestion, strings.Join(details, "\n"))
}

func normalizeProgressMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"Hook安装成功，现在登录微信...", "Hook 安装成功，请等待微信界面出现后再登录...",
		"请在微信中登录账号，密钥将在登录时自动获取...", "Hook 安装成功，请等待微信界面出现后登录，密钥将在登录时自动获取...",
		"请在微信中登录账号...", "请等待微信界面出现后再登录微信...",
	)

	return replacer.Replace(message)
}

func describeLoadErr(err error) string {
	if err == nil {
		return ""
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return fmt.Sprintf("%v(code=%d)", err, uint32(errno))
	}
	return err.Error()
}

func loadHint(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return "unknown"
	}
	if errno == syscall.Errno(126) {
		return "dependent DLL missing (try installing VC++ runtime or placing dependency DLLs beside wx_key.dll)"
	}
	if errno == syscall.Errno(193) {
		return "architecture mismatch (x86/x64/arm64 mismatch between chatlog.exe and wx_key.dll)"
	}
	return fmt.Sprintf("winerr=%d", uint32(errno))
}

func dllPathState(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("missing(%v)", err)
	}
	return fmt.Sprintf("exists(size=%d)", info.Size())
}

func currentExeMachineLabel() string {
	exePath, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	return peMachineLabel(exePath)
}

func peMachineLabel(filePath string) string {
	file, err := pe.Open(filePath)
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	switch file.FileHeader.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return "x86"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "x64"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "arm64"
	default:
		return fmt.Sprintf("machine_%d", file.FileHeader.Machine)
	}
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

// GetDbKeyResult 数据库密钥获取结果
type GetDbKeyResult struct {
	Success bool
	Key     string
	Error   string
	Message string
}

// GetDbKey 获取数据库密钥（简化版，不重启微信）
func GetDbKey(dllPath string, pid uint32, timeout int, onProgress func(string)) GetDbKeyResult {
	progressMessages := make([]string, 0, 16)
	reportProgress := func(message string) {
		message = normalizeProgressMessage(message)
		if message == "" {
			return
		}
		progressMessages = append(progressMessages, message)
		if onProgress != nil {
			onProgress(message)
		}
	}

	reportProgress("正在加载 Hook DLL...")

	hc, err := NewHookController(dllPath)
	if err != nil {
		return GetDbKeyResult{
			Success: false,
			Error:   err.Error(),
			Message: strings.Join(progressMessages, "\n"),
		}
	}
	defer hc.Release()

	reportProgress(fmt.Sprintf("正在初始化 Hook (PID: %d)...", pid))

	// 状态消息处理回调
	statusHandler := func(msg string, level int) {
		prefix := "[*]"
		switch level {
		case 1:
			prefix = "[+]"
		case 2:
			prefix = "[!]"
		}
		reportProgress(fmt.Sprintf("%s %s", prefix, msg))
	}

	// 初始化 Hook
	if !hc.Initialize(pid) {
		hc.ProcessStatusMessages(statusHandler)
		errMsg := hc.GetLastError()
		if errMsg == "" {
			errMsg = "初始化 Hook 失败"
		}
		return GetDbKeyResult{
			Success: false,
			Error:   errMsg,
			Message: strings.Join(progressMessages, "\n"),
		}
	}

	// 处理初始化阶段的状态消息
	hc.ProcessStatusMessages(statusHandler)

	reportProgress("Hook 安装成功，等待密钥...")
	reportProgress("请在微信中登录账号...")

	// 轮询等待密钥
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		// 处理状态消息
		hc.ProcessStatusMessages(statusHandler)

		// 检查是否有新密钥
		key, hasKey := hc.PollKeyData()
		if hasKey && len(key) > 0 {
			reportProgress("成功获取数据库密钥！")
			hc.Cleanup()
			return GetDbKeyResult{
				Success: true,
				Key:     key,
				Message: strings.Join(progressMessages, "\n"),
			}
		}

		time.Sleep(pollInterval)
	}

	// 超时
	hc.Cleanup()
	return GetDbKeyResult{
		Success: false,
		Error:   fmt.Sprintf("等待密钥超时（%d秒）。请确保在微信中完成登录操作。", timeout),
		Message: strings.Join(progressMessages, "\n"),
	}
}

// GetDbKeyFull 完整的数据库密钥获取流程（自动重启微信）
func GetDbKeyFull(dllPath string, timeout int, onProgress func(string)) GetDbKeyResult {
	progressMessages := make([]string, 0, 24)
	reportProgress := func(message string) {
		message = normalizeProgressMessage(message)
		if message == "" {
			return
		}
		progressMessages = append(progressMessages, message)
		if onProgress != nil {
			onProgress(message)
		}
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
		reportProgress(fmt.Sprintf("%s %s", prefix, msg))
	}

	// 1. 检查 DLL
	reportProgress("正在加载 Hook DLL...")

	hc, err := NewHookController(dllPath)
	if err != nil {
		return GetDbKeyResult{
			Success: false,
			Error:   err.Error(),
			Message: strings.Join(progressMessages, "\n"),
		}
	}
	defer hc.Release()

	// 2. 检查微信是否运行，如果运行则关闭
	reportProgress("正在检查微信进程...")

	reportProgress("正在关闭当前进程并清理残留...")
	if err := RestartPreparationCleanup(reportProgress); err != nil {
		return GetDbKeyResult{
			Success: false,
			Error:   fmt.Sprintf("启动前清理微信进程失败: %v", err),
			Message: strings.Join(progressMessages, "\n"),
		}
	}
	reportProgress("微信相关进程已全部退出，准备重新启动微信")

	// 3. 启动微信
	reportProgress("正在启动微信...")

	err = LaunchWeChat()
	if err != nil {
		return GetDbKeyResult{
			Success: false,
			Error:   err.Error(),
			Message: strings.Join(progressMessages, "\n"),
		}
	}

	reportProgress("微信进程已启动，正在等待微信主进程出现...")

	// 4. 等待微信进程出现
	pid, found := WaitForWeChatProcess(15, reportProgress)
	if !found {
		return GetDbKeyResult{
			Success: false,
			Error:   "等待微信窗口超时",
			Message: strings.Join(progressMessages, "\n"),
		}
	}

	reportProgress(fmt.Sprintf("找到微信进程 PID: %d", pid))
	reportProgress("正在等待微信界面完成初始化...")
	if !WaitForWeChatWindowReady(pid, 25, reportProgress) {
		return GetDbKeyResult{
			Success: false,
			Error:   "等待微信界面就绪超时，请确认微信是否成功启动并显示登录窗口",
			Message: strings.Join(progressMessages, "\n"),
		}
	}

	// 5. 初始化并安装 Hook
	reportProgress("正在安装远程 Hook...")

	if !hc.Initialize(pid) {
		hc.ProcessStatusMessages(statusHandler)
		errMsg := hc.GetLastError()
		if errMsg == "" {
			errMsg = "初始化 Hook 失败"
		}
		return GetDbKeyResult{
			Success: false,
			Error:   errMsg,
			Message: strings.Join(progressMessages, "\n"),
		}
	}

	// 处理初始化阶段的状态消息
	hc.ProcessStatusMessages(statusHandler)

	reportProgress("正在确认微信进程状态...")
	time.Sleep(1500 * time.Millisecond)
	hc.ProcessStatusMessages(statusHandler)
	if !IsPIDRunning(pid) {
		hc.Cleanup()
		return GetDbKeyResult{
			Success: false,
			Error:   "Hook 安装后微信进程已退出，请检查 DLL 兼容性或当前微信版本是否受支持",
			Message: strings.Join(progressMessages, "\n"),
		}
	}

	reportProgress("Hook 安装成功！")
	reportProgress("Hook 安装成功，请等待微信界面出现后登录，密钥将在登录时自动获取...")

	// 6. 轮询等待密钥
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		// 处理状态消息
		hc.ProcessStatusMessages(statusHandler)

		if !IsPIDRunning(pid) {
			hc.Cleanup()
			return GetDbKeyResult{
				Success: false,
				Error:   "等待密钥期间微信进程已退出，请重新启动微信后重试",
				Message: strings.Join(progressMessages, "\n"),
			}
		}

		// 检查是否有新密钥
		key, hasKey := hc.PollKeyData()
		if hasKey && len(key) > 0 {
			reportProgress("成功获取数据库密钥！")
			hc.Cleanup()
			return GetDbKeyResult{
				Success: true,
				Key:     key,
				Message: strings.Join(progressMessages, "\n"),
			}
		}

		time.Sleep(pollInterval)
	}

	// 超时
	hc.Cleanup()
	return GetDbKeyResult{
		Success: false,
		Error:   fmt.Sprintf("等待密钥超时（%d秒）。请确保在微信中完成登录操作。", timeout),
		Message: strings.Join(progressMessages, "\n"),
	}
}

// FindProcessIdsByName find process id by name
func FindProcessIdsByName(name string) ([]uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	var pids []uint32
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}

	for {
		if windows.UTF16ToString(entry.ExeFile[:]) == name {
			pids = append(pids, entry.ProcessID)
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}

	return pids, nil
}

// IsPIDRunning 检查指定 PID 是否仍然存活
func IsPIDRunning(pid uint32) bool {
	const stillActiveExitCode = 259

	if pid == 0 {
		return false
	}

	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(hProcess)

	var code uint32
	if err := windows.GetExitCodeProcess(hProcess, &code); err != nil {
		return false
	}

	return code == stillActiveExitCode
}

// IsProcessRunning check if process is running
func IsProcessRunning(name string) bool {
	pids, err := FindProcessIdsByName(name)
	return err == nil && len(pids) > 0
}

// HasAnyWeChatProcess 检查是否存在任意微信相关进程
func HasAnyWeChatProcess() bool {
	for _, name := range wechatProcessNames {
		if IsProcessRunning(name) {
			return true
		}
	}
	return false
}

// KillWeChatProcesses 关闭所有微信进程
func KillWeChatProcesses() error {
	for _, name := range []string{"Weixin.exe", "WeChat.exe"} {
		if !IsProcessRunning(name) {
			continue
		}
		cmd := util.Command("taskkill", "/F", "/IM", name, "/T")
		_ = cmd.Run()
	}

	for _, name := range wechatProcessNames {
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

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !HasAnyWeChatProcess() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	remaining := make([]string, 0, len(wechatProcessNames))
	for _, name := range wechatProcessNames {
		if IsProcessRunning(name) {
			remaining = append(remaining, name)
		}
	}
	if len(remaining) > 0 {
		return fmt.Errorf("以下进程仍未退出: %s", strings.Join(remaining, ", "))
	}

	return nil
}

// RestartPreparationCleanup 在重启微信前关闭当前进程并清理残留
func RestartPreparationCleanup(onProgress func(string)) error {
	if onProgress != nil {
		onProgress("正在结束当前微信进程...")
	}
	if err := KillWeChatProcesses(); err != nil {
		return err
	}

	if onProgress != nil {
		onProgress("正在确认微信残留进程是否已清理完成...")
	}
	quietDeadline := time.Now().Add(3 * time.Second)
	nextProgressAt := time.Now().Add(1200 * time.Millisecond)
	for time.Now().Before(quietDeadline) {
		if HasAnyWeChatProcess() {
			quietDeadline = time.Now().Add(3 * time.Second)
		}
		if onProgress != nil && time.Now().After(nextProgressAt) {
			onProgress("正在等待微信残留进程完全退出...")
			nextProgressAt = time.Now().Add(1200 * time.Millisecond)
		}
		time.Sleep(300 * time.Millisecond)
	}

	if HasAnyWeChatProcess() {
		return fmt.Errorf("微信残留进程仍未完全退出")
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

// WaitForWeChatProcess 等待微信主进程出现
func WaitForWeChatProcess(maxWaitSeconds int, onProgress func(string)) (uint32, bool) {
	deadline := time.Now().Add(time.Duration(maxWaitSeconds) * time.Second)
	nextProgressAt := time.Now()

	for time.Now().Before(deadline) {
		for _, name := range []string{"Weixin.exe", "WeChat.exe"} {
			pids, _ := FindProcessIdsByName(name)
			if len(pids) > 0 {
				return pids[0], true
			}
		}
		if onProgress != nil && time.Now().After(nextProgressAt) {
			onProgress("正在等待微信主进程启动...")
			nextProgressAt = time.Now().Add(1200 * time.Millisecond)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return 0, false
}

func WaitForWeChatWindowReady(pid uint32, maxWaitSeconds int, onProgress func(string)) bool {
	if pid == 0 {
		return false
	}

	deadline := time.Now().Add(time.Duration(maxWaitSeconds) * time.Second)
	nextProgressAt := time.Now()
	for time.Now().Before(deadline) {
		if !IsPIDRunning(pid) {
			return false
		}

		_ = waitForInputIdle(pid, 1200*time.Millisecond)
		if hasVisibleTopLevelWindow(pid) {
			return true
		}
		if onProgress != nil && time.Now().After(nextProgressAt) {
			onProgress("正在等待微信登录界面显示...")
			nextProgressAt = time.Now().Add(1500 * time.Millisecond)
		}

		time.Sleep(300 * time.Millisecond)
	}

	return false
}

func waitForInputIdle(pid uint32, timeout time.Duration) bool {
	hProcess, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(hProcess)

	ret, _, _ := procWaitForInputIdle.Call(uintptr(hProcess), uintptr(timeout/time.Millisecond))
	return ret == 0
}

func hasVisibleTopLevelWindow(pid uint32) bool {
	found := false
	callback := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var windowPID uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID != pid {
			return 1
		}

		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}

		found = true
		return 0
	})

	procEnumWindows.Call(callback, 0)
	return found
}
