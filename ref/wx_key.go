package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// 命令行参数
	mode := flag.String("mode", "image", "获取模式: image(图片密钥) 或 db(数据库密钥)")
	cacheDir := flag.String("dir", "", "微信缓存目录路径（图片密钥模式可选）")
	dllPath := flag.String("dll", "", "wx_key.dll 路径（数据库密钥模式必需）")
	pid := flag.Uint("pid", 0, "微信进程 PID（可选，数据库模式下如不指定则自动重启微信）")
	timeout := flag.Int("timeout", 120, "等待超时时间（秒，仅数据库密钥模式）")
	flag.Parse()

	// 进度回调
	onProgress := func(msg string) {
		fmt.Printf("[*] %s\n", msg)
	}

	fmt.Println("=== 微信密钥获取工具 (Go 版本) ===")
	fmt.Println()

	switch *mode {
	case "image":
		getImageKeyMode(*cacheDir, onProgress)
	case "db":
		getDbKeyMode(*dllPath, uint32(*pid), *timeout, onProgress)
	default:
		fmt.Printf("未知模式: %s\n", *mode)
		fmt.Println("使用 -mode=image 获取图片密钥")
		fmt.Println("使用 -mode=db 获取数据库密钥")
		os.Exit(1)
	}
}

// getImageKeyMode 图片密钥获取模式
func getImageKeyMode(cacheDir string, onProgress func(string)) {
	fmt.Println(">> 图片密钥获取模式")
	fmt.Println()

	result := GetImageKeys(cacheDir, onProgress)

	fmt.Println()

	if result.Success {
		fmt.Println("=== 密钥获取成功 ===")
		fmt.Printf("XOR 密钥: 0x%02X\n", result.XorKey)
		fmt.Printf("AES 密钥: %s\n", result.AesKey)
	} else {
		fmt.Println("=== 密钥获取失败 ===")
		fmt.Printf("错误: %s\n", result.Error)

		if result.NeedManualSelection {
			fmt.Println("\n提示: 请使用 -dir 参数指定微信缓存目录")
		}

		os.Exit(1)
	}
}

// getDbKeyMode 数据库密钥获取模式
func getDbKeyMode(dllPath string, pid uint32, timeout int, onProgress func(string)) {
	fmt.Println(">> 数据库密钥获取模式")
	fmt.Println()

	// 检查 DLL 路径
	if dllPath == "" {
		// 尝试在当前目录查找
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		defaultDll := filepath.Join(exeDir, "wx_key.dll")

		if _, err := os.Stat(defaultDll); err == nil {
			dllPath = defaultDll
		} else {
			// 尝试当前工作目录
			cwd, _ := os.Getwd()
			defaultDll = filepath.Join(cwd, "wx_key.dll")
			if _, err := os.Stat(defaultDll); err == nil {
				dllPath = defaultDll
			} else {
				fmt.Println("错误: 未指定 DLL 路径")
				fmt.Println("请使用 -dll 参数指定 wx_key.dll 的路径")
				os.Exit(1)
			}
		}
	}

	// 检查 DLL 是否存在
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		fmt.Printf("错误: DLL 文件不存在: %s\n", dllPath)
		os.Exit(1)
	}

	fmt.Printf("使用 DLL: %s\n", dllPath)
	fmt.Println()

	var result DbKeyResult

	if pid == 0 {
		// 未指定 PID，使用完整流程（关闭→启动→Hook）
		fmt.Println("未指定 PID，将自动重启微信并获取密钥...")
		fmt.Println()
		result = GetDbKeyFull(dllPath, timeout, onProgress)
	} else {
		// 指定了 PID，直接对该进程安装 Hook
		fmt.Printf("使用指定 PID: %d\n", pid)
		fmt.Println()
		result = GetDbKey(dllPath, pid, timeout, onProgress)
	}

	fmt.Println()

	if result.Success {
		fmt.Println("=== 数据库密钥获取成功 ===")
		fmt.Printf("密钥 (Hex): %s\n", result.Key)
	} else {
		fmt.Println("=== 数据库密钥获取失败 ===")
		fmt.Printf("错误: %s\n", result.Error)
		os.Exit(1)
	}
}
