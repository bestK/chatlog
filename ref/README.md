# 微信密钥获取工具 (Go 版本)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat&logo=windows)](https://www.microsoft.com/windows)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

基于 [wx_key](https://github.com/ycccccccy/wx_key) 项目的 Go 语言移植版本，用于获取微信 4.0 及以上版本的数据库密钥和图片密钥。

## ✨ 功能特性

-   🔐 **数据库密钥获取** - 通过 DLL Hook 技术获取微信数据库加密密钥
-   🖼️ **图片密钥获取** - 从进程内存中提取图片缓存的 XOR 和 AES 解密密钥
-   🚀 **纯 Go 实现** - 图片密钥获取模块使用纯 Go 实现，无需额外依赖
-   📦 **单文件部署** - 编译后生成单个可执行文件，便于分发

## 📋 支持版本

支持所有微信 4.x 版本，已测试版本包括：

-   4.1.5.11
-   4.1.4.17
-   4.1.4.15
-   4.1.2.18
-   4.1.2.17
-   4.1.0.30
-   4.0.5.17

## 🚀 快速开始

### 编译

```bash
go build -o wx_key.exe .
```

### 使用方法

#### 图片密钥模式（默认）

```bash
# 自动检测微信缓存目录
wx_key.exe -mode=image

# 手动指定缓存目录
wx_key.exe -mode=image -dir="C:\Users\xxx\Documents\WeChat Files\wxid_xxx"
```

#### 数据库密钥模式

```bash
# 自动重启微信并获取密钥
wx_key.exe -mode=db

# 指定已运行的微信进程 PID
wx_key.exe -mode=db -pid=12345

# 自定义超时时间（默认 120 秒）
wx_key.exe -mode=db -timeout=180
```

### 命令行参数

| 参数       | 说明                                             | 默认值   |
| ---------- | ------------------------------------------------ | -------- |
| `-mode`    | 获取模式：`image`(图片密钥) 或 `db`(数据库密钥)  | `image`  |
| `-dir`     | 微信缓存目录路径（图片密钥模式可选）             | 自动检测 |
| `-dll`     | wx_key.dll 路径（数据库密钥模式）                | 当前目录 |
| `-pid`     | 微信进程 PID（数据库模式，不指定则自动重启微信） | 自动检测 |
| `-timeout` | 等待超时时间（秒，仅数据库密钥模式）             | `120`    |

## 📁 项目结构

```
wx_key_go/
├── wx_key.go              # 主程序入口
├── image_key_service.go   # 图片密钥获取服务（纯 Go）
├── db_key_service.go      # 数据库密钥获取服务
├── process_helper.go      # 进程辅助工具
├── windows_api.go         # Windows API 封装
├── wx_key.dll             # 数据库密钥 Hook DLL
├── go.mod                 # Go 模块定义
└── go.sum                 # 依赖校验
```

## ⚠️ 注意事项

1. **管理员权限** - 获取数据库密钥需要管理员权限运行
2. **目录路径** - 请勿将工具放在包含中文字符的目录下，否则可能导致 DLL 加载失败
3. **微信状态** - 数据库密钥模式需要在微信登录过程中捕获，工具会自动重启微信

## 📜 许可证

本项目采用 MIT 许可证。

## 🙏 鸣谢

-   [ycccccccy/wx_key](https://github.com/ycccccccy/wx_key) - 原始项目，本项目基于其核心逻辑进行 Go 语言移植

## ⚠️ 免责声明

本工具仅供学习研究和数据备份使用，请勿用于非法用途。使用本工具所产生的一切后果由使用者自行承担。
