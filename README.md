# Chatlog

Chatlog 是一个聊天记录收集、分析的开源工具，旨在帮助用户更好地利用自己的聊天数据。  
目前支持微信聊天记录的解密和查询，提供 Terminal UI 界面和 HTTP API 服务，让您可以方便地访问和分析聊天数据。

## 功能特点

- **数据收集**：从本地数据库文件中获取聊天数据
- **终端界面**：提供简洁的 Terminal UI，方便直接操作
- **HTTP API**：提供 API 接口，支持查询聊天记录、联系人和群聊信息
- **MCP 支持**：实现 Model Context Protocol，可与支持 MCP 的 AI 助手无缝集成
- **多格式输出**：支持 JSON、CSV、纯文本等多种输出格式

## 安装

### 从源码安装

```bash
go install github.com/sjzar/chatlog@latest
```

### 下载预编译版本

访问 [Releases](https://github.com/sjzar/chatlog/releases) 页面下载适合您系统的预编译版本。

## 快速开始

### 终端 UI 模式

1. 启动程序：

```bash
./chatlog
```

2. 使用界面操作：
   - 使用方向键导航菜单
   - 按 Enter 选择菜单项
   - 按 Esc 返回上一级菜单
   - 按 Ctrl+C 退出程序

### 命令行模式

获取微信进程密钥：

```bash
./chatlog key
```

解密数据库文件：

```bash
./chatlog decrypt --data-dir "微信数据目录" --work-dir "输出目录" --key "密钥" --version 3
```

## HTTP API

启动 HTTP 服务后，可以通过以下 API 访问数据：

### 聊天记录

```
GET /api/v1/chatlog?time=2023-01-01&talker=wxid_xxx&limit=100&offset=0&format=json
```

参数说明：
- `time`: 时间范围，格式为 `YYYY-MM-DD` 或 `YYYY-MM-DD~YYYY-MM-DD`
- `talker`: 聊天对象的 ID，不知道 ID 的话也可以尝试备注名、昵称、群聊 ID等
- `limit`: 返回记录数量限制
- `offset`: 分页偏移量
- `format`: 输出格式，支持 `json`、`csv` 或纯文本

### 联系人列表

```
GET /api/v1/contact?format=json
```

### 群聊列表

```
GET /api/v1/chatroom?format=json
```

### 会话列表

```
GET /api/v1/session?limit=100&format=json
```

## MCP 集成

Chatlog 实现了 Model Context Protocol (MCP)，可以与支持 MCP 的 AI 助手集成。通过 MCP，AI 助手可以：

1. 查询联系人信息
2. 获取群聊列表和成员
3. 检索最近的聊天记录
4. 按时间和联系人搜索聊天记录

## 未来规划

Chatlog 希望成为最好用的聊天记录工具，帮助用户充分挖掘自己聊天数据的价值。我们的路线图包括：

- **多平台支持**：计划支持 MacOS 平台的微信聊天记录解密
- **全文索引**：实现聊天记录的全文检索，提供更快速的搜索体验
- **统计与可视化**：提供聊天数据的统计分析和可视化 Dashboard
- **CS 架构**：将数据收集和统计分析功能分离，支持将服务部署在 NAS 或家庭服务器上
- **增量更新**：支持聊天记录的增量采集和更新，减少资源消耗
- **关键词监控**：提供关键词监控和实时提醒功能
- **更多聊天工具支持**：计划支持更多主流聊天工具的数据采集和分析

## 数据安全声明

Chatlog 高度重视用户数据安全和隐私保护：

- 所有数据处理均在本地完成，不会上传到任何外部服务器
- 解密后的数据存储在用户指定的工作目录中，用户对数据有完全控制权
- 建议定期备份重要的聊天记录，并妥善保管解密后的数据
- 请勿将本工具用于未经授权访问他人聊天记录等非法用途

## 贡献

我们欢迎社区的贡献！无论是代码贡献、问题报告还是功能建议，都将帮助 Chatlog 变得更好：

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开一个 Pull Request

## 许可证

本项目采用 Apache-2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

- [tview](https://github.com/rivo/tview) - 终端 UI 库
- [gin](https://github.com/gin-gonic/gin) - HTTP 框架
- [Model Context Protocol](https://github.com/modelcontextprotocol) - AI 助手集成协议
- 以及所有贡献者和用户的支持与反馈