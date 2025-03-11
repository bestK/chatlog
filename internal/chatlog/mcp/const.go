package mcp

import (
	"github.com/sjzar/chatlog/internal/mcp"
)

// MCPTools 和资源定义
var (
	InitializeResponse = mcp.InitializeResponse{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities:    mcp.DefaultCapabilities,
		ServerInfo: mcp.ServerInfo{
			Name:    "chatlog",
			Version: "0.0.1",
		},
	}

	ToolContact = mcp.Tool{
		Name:        "query_contact",
		Description: "查询用户的联系人信息。可以通过姓名、备注名或ID进行查询，返回匹配的联系人列表。当用户询问某人的联系方式、想了解联系人信息或需要查找特定联系人时使用此工具。参数为空时，将返回联系人列表",
		InputSchema: mcp.ToolSchema{
			Type: "object",
			Properties: mcp.M{
				"query": mcp.M{
					"type":        "string",
					"description": "联系人的搜索关键词，可以是姓名、备注名或ID。",
				},
			},
		},
	}

	ToolChatRoom = mcp.Tool{
		Name:        "query_chat_room",
		Description: "查询用户参与的群聊信息。可以通过群名称、群ID或相关关键词进行查询，返回匹配的群聊列表。当用户询问群聊信息、想了解某个群的详情或需要查找特定群聊时使用此工具。",
		InputSchema: mcp.ToolSchema{
			Type: "object",
			Properties: mcp.M{
				"query": mcp.M{
					"type":        "string",
					"description": "群聊的搜索关键词，可以是群名称、群ID或相关描述",
				},
			},
		},
	}

	ToolRecentChat = mcp.Tool{
		Name:        "query_recent_chat",
		Description: "查询最近会话列表，包括个人聊天和群聊。当用户想了解最近的聊天记录、查看最近联系过的人或群组时使用此工具。不需要参数，直接返回最近的会话列表。",
		InputSchema: mcp.ToolSchema{
			Type:       "object",
			Properties: mcp.M{},
		},
	}

	ToolChatLog = mcp.Tool{
		Name:        "chatlog",
		Description: "查询特定时间或时间段内与特定联系人或群组的聊天记录。当用户需要回顾过去的对话内容、查找特定信息或想了解与某人/某群的历史交流时使用此工具。",
		InputSchema: mcp.ToolSchema{
			Type: "object",
			Properties: mcp.M{
				"time": mcp.M{
					"type":        "string",
					"description": "查询的时间点或时间段。可以是具体时间，例如 YYYY-MM-DD，也可以是时间段，例如 YYYY-MM-DD~YYYY-MM-DD，时间段之间用\"~\"分隔。",
				},
				"talker": mcp.M{
					"type":        "string",
					"description": "交谈对象，可以是联系人或群聊。支持使用ID、昵称、备注名等进行查询。",
				},
			},
		},
	}

	ResourceRecentChat = mcp.Resource{
		Name:        "最近会话",
		URI:         "session://recent",
		Description: "获取最近的聊天会话列表",
	}

	ResourceTemplateContact = mcp.ResourceTemplate{
		Name:        "联系人信息",
		URITemplate: "contact://{username}",
		Description: "获取指定联系人的详细信息",
	}

	ResourceTemplateChatRoom = mcp.ResourceTemplate{
		Name:        "群聊信息",
		URITemplate: "chatroom://{roomid}",
		Description: "获取指定群聊的详细信息",
	}

	ResourceTemplateChatlog = mcp.ResourceTemplate{
		Name:        "聊天记录",
		URITemplate: "chatlog://{talker}/{timeframe}?limit,offset",
		Description: "获取与特定联系人或群聊的聊天记录",
	}
)
