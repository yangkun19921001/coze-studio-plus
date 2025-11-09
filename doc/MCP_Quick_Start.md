# MCP 集成快速开始指南

本指南帮助您快速在 Coze Studio 中集成和测试 MCP (Model Context Protocol) 功能。

## 📋 目录

- [前置要求](#前置要求)
- [MCP 服务器准备](#mcp-服务器准备)
- [配置 MCP 插件](#配置-mcp-插件)
- [测试 MCP 集成](#测试-mcp-集成)
- [故障排查](#故障排查)

---

## 前置要求

1. **Go 环境**：Go 1.21+
2. **Node.js 环境**：Node.js 18+ （用于运行示例 MCP 服务器）
3. **已启动 Coze Studio 后端服务**

---

## MCP 服务器准备

### 方式 1：使用官方示例服务器（推荐测试）

```bash
# 安装 @modelcontextprotocol/server-everything 示例服务器
npm install -g @modelcontextprotocol/server-everything

# 或者使用本地安装
mkdir ~/mcp-server-test
cd ~/mcp-server-test
npm init -y
npm install @modelcontextprotocol/server-everything
```

### 方式 2：创建简单的测试 MCP 服务器

创建文件 `mcp-test-server.js`：

```javascript
#!/usr/bin/env node

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');

// 创建 MCP 服务器
const server = new Server(
  {
    name: 'test-mcp-server',
    version: '1.0.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// 注册工具
server.setRequestHandler('tools/list', async () => {
  return {
    tools: [
      {
        name: 'get_current_time',
        description: '获取当前时间',
        inputSchema: {
          type: 'object',
          properties: {
            timezone: {
              type: 'string',
              description: '时区，例如 Asia/Shanghai',
            },
          },
        },
      },
      {
        name: 'calculate',
        description: '执行简单的数学计算',
        inputSchema: {
          type: 'object',
          properties: {
            expression: {
              type: 'string',
              description: '数学表达式，例如 "2 + 2"',
            },
          },
          required: ['expression'],
        },
      },
    ],
  };
});

// 处理工具调用
server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;

  if (name === 'get_current_time') {
    const timezone = args.timezone || 'UTC';
    const now = new Date().toLocaleString('zh-CN', { timeZone: timezone });
    return {
      content: [
        {
          type: 'text',
          text: `当前时间（${timezone}）: ${now}`,
        },
      ],
    };
  }

  if (name === 'calculate') {
    try {
      // 注意：这只是示例，实际生产环境不应该使用 eval
      const result = eval(args.expression);
      return {
        content: [
          {
            type: 'text',
            text: `计算结果: ${args.expression} = ${result}`,
          },
        ],
      };
    } catch (error) {
      return {
        content: [
          {
            type: 'text',
            text: `计算错误: ${error.message}`,
          },
        ],
        isError: true,
      };
    }
  }

  throw new Error(`Unknown tool: ${name}`);
});

// 启动服务器
const transport = new StdioServerTransport();
server.connect(transport);
console.error('MCP Test Server Started');
```

保存并添加执行权限：

```bash
chmod +x mcp-test-server.js

# 测试服务器
node mcp-test-server.js
# 输入: {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
# 应该看到初始化响应
```

---

## 配置 MCP 插件

### 1. 创建 MCP 插件配置

在 Coze Studio 中，MCP 配置需要存储在插件 manifest 的 `api.extensions.mcp_config` 字段中。

#### Stdio 方式配置示例：

```json
{
  "schema_version": "v1",
  "name_for_model": "test_mcp_tools",
  "name_for_human": "测试 MCP 工具",
  "description_for_model": "通过 MCP 协议提供的测试工具",
  "description_for_human": "用于测试 MCP 集成的工具",
  "api": {
    "type": "coze-studio-mcp",
    "extensions": {
      "mcp_config": {
        "transport_type": "stdio",
        "stdio_config": {
          "command": ["node", "/path/to/mcp-test-server.js"],
          "env": {
            "NODE_ENV": "production"
          },
          "working_dir": "/path/to/working/directory"
        }
      }
    }
  }
}
```

#### SSE 方式配置示例：

```json
{
  "schema_version": "v1",
  "name_for_model": "remote_mcp_tools",
  "name_for_human": "远程 MCP 工具",
  "description_for_model": "通过 SSE 连接的远程 MCP 工具",
  "description_for_human": "远程 MCP 服务器提供的工具",
  "api": {
    "type": "coze-studio-mcp",
    "extensions": {
      "mcp_config": {
        "transport_type": "sse",
        "sse_config": {
          "url": "https://your-mcp-server.com",
          "api_key": "your-api-key",
          "headers": {
            "X-Custom-Header": "value"
          }
        }
      }
    }
  }
}
```

### 2. 工具定义

MCP 工具会自动从 MCP 服务器获取，您需要在 Coze Studio 中创建对应的工具定义：

```json
{
  "id": "get_current_time",
  "name": "get_current_time",
  "description": "获取当前时间",
  "parameters": {
    "timezone": {
      "type": "string",
      "description": "时区"
    }
  }
}
```

---

## 测试 MCP 集成

### 方式 1：使用 Go 测试代码

创建 `backend/pkg/mcp/client_test.go`：

```go
package mcp_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/coze-dev/coze-studio/backend/pkg/mcp"
)

func TestStdioMCP(t *testing.T) {
	// 创建配置
	config := &mcp.Config{
		TransportType: mcp.TransportTypeStdio,
		StdioConfig: &mcp.StdioConfig{
			Command: []string{"node", "/path/to/mcp-test-server.js"},
		},
	}

	// 创建客户端
	client, err := mcp.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 初始化
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Logf("Connected to: %s v%s", client.GetServerInfo().Name, client.GetServerInfo().Version)

	// 列出工具
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	t.Logf("Found %d tools:", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description)
	}

	// 调用工具
	result, err := client.CallTool(ctx, "get_current_time", map[string]any{
		"timezone": "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}

	t.Logf("Tool result: %+v", result)
}

func TestSSEMCP(t *testing.T) {
	// 检查是否设置了 SSE 服务器 URL
	sseURL := os.Getenv("MCP_SSE_URL")
	if sseURL == "" {
		t.Skip("MCP_SSE_URL not set, skipping SSE test")
	}

	config := &mcp.Config{
		TransportType: mcp.TransportTypeSSE,
		SSEConfig: &mcp.SSEConfig{
			URL:    sseURL,
			APIKey: os.Getenv("MCP_API_KEY"),
		},
	}

	client, err := mcp.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Logf("Connected to: %s v%s", client.GetServerInfo().Name, client.GetServerInfo().Version)

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	t.Logf("Found %d tools", len(tools))
}
```

运行测试：

```bash
cd backend
go test -v ./pkg/mcp/...
```

### 方式 2：通过 Workflow 测试

1. 在 Coze Studio 中创建 Workflow
2. 添加 LLM 节点
3. 添加 MCP 插件节点
4. 配置插件工具调用
5. 运行测试

---

## 故障排查

### 问题 1：stdio 服务器无法启动

**症状**：`start command failed` 错误

**解决方案**：
```bash
# 检查命令路径
which node

# 检查脚本权限
chmod +x /path/to/mcp-test-server.js

# 手动测试脚本
node /path/to/mcp-test-server.js
```

### 问题 2：无法连接到 SSE 服务器

**症状**：`connect sse failed` 错误

**解决方案**：
```bash
# 测试 SSE 端点
curl -N -H "Accept: text/event-stream" https://your-mcp-server.com/sse

# 检查防火墙设置
# 检查 API Key 是否正确
```

### 问题 3：工具调用失败

**症状**：`call mcp tool failed` 错误

**解决方案**：
- 检查工具名称是否匹配
- 检查参数格式
- 查看服务器日志（stderr）
- 启用调试日志：设置环境变量 `LOG_LEVEL=debug`

### 问题 4：配置解析失败

**症状**：`parse mcp config failed` 错误

**解决方案**：
- 检查 `mcp_config` JSON 格式
- 确保 `transport_type` 正确
- 验证必需字段（如 `command` 或 `url`）

---

## 下一步

1. **生产环境部署**：参考 `doc/MCP_Integration_Guide.md`
2. **自定义 MCP 服务器**：查看 MCP 官方文档
3. **性能优化**：配置连接池和缓存
4. **安全加固**：添加认证和授权

---

## 资源

- [MCP 官方文档](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [示例 MCP 服务器](https://github.com/modelcontextprotocol/servers)

---

**文档版本**：v1.0  
**创建日期**：2025-11-09  
**维护者**：开发团队

