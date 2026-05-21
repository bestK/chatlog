<div align="center">

# Chatlog

_微信聊天记录管理工具_

[![GitHub release](https://img.shields.io/github/release/bestK/chatlog.svg)](https://github.com/bestK/chatlog/releases)
[![GitHub license](https://img.shields.io/github/license/bestK/chatlog.svg)](https://github.com/bestK/chatlog/blob/main/LICENSE)

> Fork 自 [sjzar/chatlog](https://github.com/sjzar/chatlog)，在原项目基础上增加了 AI 总结、聊天记录浏览、纯 Go SQLite 等功能。

</div>

## 功能

- 微信聊天记录本地解密与浏览
- 支持 Windows / macOS，兼容微信 4.x
- 桌面 GUI（Wails）+ 命令行工具
- 联系人聊天记录浏览，图片/视频/语音/表情渲染
- 图片点击放大预览，聊天记录无限滚动加载
- AI 聊天记录总结（流式输出，支持 OpenAI / Anthropic / Google）
- 自定义总结提示词，AI 提供商配置持久化
- HTTP API + MCP 协议，可与 AI 助手集成
- Webhook 新消息回调
- 自动解密，多账号切换

## 快速开始

### 下载

从 [Releases](https://github.com/bestK/chatlog/releases) 下载：

| 平台 | 架构 |
|------|------|
| Windows | amd64 |
| macOS | amd64 (Intel) |
| macOS | arm64 (Apple Silicon) |

### 运行

```bash
# 启动桌面 GUI
chatlog

# 或命令行模式
chatlog key       # 获取密钥
chatlog decrypt   # 解密数据库
chatlog server    # 启动 HTTP 服务
```

### 基本流程

1. 启动程序，检测微信进程
2. 获取数据库密钥（设置页 → 自动获取）
3. 解密数据库
4. 浏览联系人聊天记录 / 启动 HTTP 服务

## 桌面 GUI

- **账号页**：切换微信账号，点击联系人查看聊天记录
- **服务页**：启动/停止 HTTP API 和 MCP 服务
- **AI 页**：配置 AI 提供商（OpenAI、Anthropic、Google 等）
- **设置页**：目录配置、密钥管理、自动解密开关、总结提示词
- **Webhook 页**：配置新消息回调规则

### AI 总结

在联系人聊天记录面板中：
1. 选择时间范围（今天/最近7天/自定义）
2. 选择 AI 提供商
3. 点击"总结"，流式输出 Markdown 结果

## HTTP API

启动服务后默认地址 `http://127.0.0.1:5030`。

### 聊天记录

```
GET /api/v1/chatlog?time=today&talker=wxid_xxx&sort=desc&format=json
```

| 参数 | 说明 |
|------|------|
| `time` | 时间范围：`today`、`yesterday`、`last-7d`、`2024-01-01~2024-01-31` |
| `talker` | 聊天对象（wxid、群 ID、备注名、昵称） |
| `sort` | `desc`（默认，新→旧）或 `asc` |
| `limit` | 返回数量 |
| `offset` | 分页偏移 |
| `format` | `json`、`csv` 或纯文本 |

### 其他接口

| 接口 | 说明 |
|------|------|
| `GET /api/v1/contact` | 联系人列表 |
| `GET /api/v1/chatroom` | 群聊列表 |
| `GET /api/v1/session` | 最近会话 |
| `GET /api/v1/ai/providers` | AI 提供商列表 |
| `POST /api/v1/ai/summary/stream` | AI 总结（SSE 流式） |
| `GET /image/<key>` | 图片 |
| `GET /video/<key>` | 视频 |
| `GET /voice/<key>` | 语音 |

## MCP 集成

支持 MCP (Model Context Protocol)，启动 HTTP 服务后：

```
Streamable HTTP: http://127.0.0.1:5030/mcp
SSE Endpoint:    http://127.0.0.1:5030/sse
```

兼容 ChatWise、Cherry Studio 等支持 MCP 的 AI 客户端。

## 从源码构建

```bash
go install github.com/sjzar/chatlog@latest
```

> 音频转码（语音转 MP3）依赖 CGO。SQLite 使用纯 Go 实现（modernc.org/sqlite），无需额外 C 依赖。

## 平台说明

### macOS

获取密钥前需临时关闭 SIP：

```bash
# 恢复模式下执行
csrutil disable
# 获取密钥后可重新启用
csrutil enable
```

### Windows

遇到命令行显示异常请使用 [Windows Terminal](https://github.com/microsoft/terminal)。

## 免责声明

本项目仅供学习和个人合法使用。仅限处理您自己合法拥有的聊天数据。详见 [DISCLAIMER.md](./DISCLAIMER.md)。

## License

[Apache-2.0](./LICENSE)

## 致谢

- [sjzar/chatlog](https://github.com/sjzar/chatlog) — 原始项目
- [wechat-dump-rs](https://github.com/0xlane/wechat-dump-rs)
- [PyWxDump](https://github.com/xaoyaoo/PyWxDump)
- [Wails](https://wails.io/)
- [modernc.org/sqlite](https://modernc.org/sqlite)
