# MCP 插件开发快速参考

**5 分钟上手 MCP 插件开发**

---

## 📝 快速对比

### OpenAPI 插件（如搜狐热闻）

```yaml
# 配置示例
manifest:
  api:
    type: openapi                    # ← OpenAPI 类型
servers:
  - url: https://api.example.com    # ← HTTP 服务器地址
```

### MCP 插件

```yaml
# 配置示例
manifest:
  api:
    type: coze-studio-mcp           # ← MCP 类型
    extensions:
      mcp_config:
        transport_type: stdio       # ← stdio/SSE 传输
        stdio_config:
          command: ["node", "/path/to/server.js"]
```

---

## 🚀 三步创建 MCP 插件

### 步骤 1：创建 MCP 服务器 (2 分钟)

```javascript
#!/usr/bin/env node
// mcp-server.js

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');

const server = new Server({
  name: 'my-mcp-server',
  version: '1.0.0',
}, {
  capabilities: { tools: {} },
});

// 1. 列出工具
server.setRequestHandler('tools/list', async () => ({
  tools: [{
    name: 'my_tool',
    description: '我的工具',
    inputSchema: {
      type: 'object',
      properties: {
        param1: { type: 'string', description: '参数1' },
      },
      required: ['param1'],
    },
  }],
}));

// 2. 调用工具
server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === 'my_tool') {
    return {
      content: [{
        type: 'text',
        text: `处理结果: ${args.param1}`,
      }],
    };
  }
  
  throw new Error(`Unknown tool: ${name}`);
});

// 3. 启动
const transport = new StdioServerTransport();
server.connect(transport);
console.error('MCP Server Started');
```

### 步骤 2：配置插件 (2 分钟)

```yaml
# backend/conf/plugin/pluginproduct/plugin_meta.yaml

- plugin_id: 100
  version: v1.0.0
  openapi_doc_file: my_mcp_plugin.yaml
  plugin_type: 1
  manifest:
    schema_version: v1
    name_for_model: my_mcp_plugin
    name_for_human: 我的 MCP 插件
    description_for_model: MCP 插件示例
    description_for_human: MCP 插件示例
    auth:
      type: none
    logo_url: plugin_icon.png
    api:
      type: coze-studio-mcp              # ← 关键：MCP 类型
      extensions:
        mcp_config:                      # ← MCP 配置
          transport_type: stdio
          stdio_config:
            command: ["node", "/absolute/path/to/mcp-server.js"]
            env:
              NODE_ENV: production
            working_dir: /working/dir
  tools:
    - tool_id: 100001
      method: get
      sub_url: /my_tool
```

### 步骤 3：定义 Schema (1 分钟)

```yaml
# backend/conf/plugin/pluginproduct/my_mcp_plugin.yaml

info:
  title: 我的 MCP 插件
  version: v1
openapi: 3.0.1

servers:
  - url: mcp://localhost

paths:
  /my_tool:
    get:
      operationId: my_tool
      summary: 我的工具
      parameters:
        - name: param1
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          content:
            application/json:
              schema:
                properties:
                  result:
                    type: string
```

---

## 🔍 架构对比图

### OpenAPI 插件流程

```
┌─────────────┐
│  Coze       │
│  Studio     │
└──────┬──────┘
       │ HTTP Request
       ▼
┌─────────────┐
│  External   │
│  REST API   │
└─────────────┘
```

### MCP 插件流程

```
┌─────────────┐
│  Coze       │
│  Studio     │
└──────┬──────┘
       │ MCP Protocol
       ▼
┌─────────────┐
│  mcp-go     │
│  Library    │
└──────┬──────┘
       │ stdio/SSE
       ▼
┌─────────────┐
│  MCP Server │
│  (Node.js)  │
└─────────────┘
```

---

## 📊 功能对比表

| 功能 | OpenAPI | MCP |
|------|---------|-----|
| **配置复杂度** | ⭐ 简单 | ⭐⭐ 中等 |
| **开发时间** | 5 分钟 | 15 分钟 |
| **HTTP API 调用** | ✅ | ❌ |
| **本地资源访问** | ❌ | ✅ |
| **自定义逻辑** | ❌ | ✅ |
| **资源管理** | ❌ | ✅ |
| **流式输出** | ❌ | ✅ |
| **会话状态** | ❌ | ✅ |

---

## 🛠️ 常用代码片段

### 1. 带参数验证的工具

```javascript
server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  // 参数验证
  if (name === 'search') {
    if (!args.query || typeof args.query !== 'string') {
      return {
        content: [{
          type: 'text',
          text: 'Error: query parameter is required and must be a string',
        }],
        isError: true,
      };
    }
    
    // 业务逻辑
    const results = await performSearch(args.query);
    
    return {
      content: [{
        type: 'text',
        text: JSON.stringify(results, null, 2),
      }],
    };
  }
  
  throw new Error(`Unknown tool: ${name}`);
});
```

### 2. 错误处理

```javascript
server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  try {
    const result = await someAsyncOperation(args);
    
    return {
      content: [{
        type: 'text',
        text: JSON.stringify(result),
      }],
    };
  } catch (error) {
    console.error('[MCP Server] Error:', error);
    
    return {
      content: [{
        type: 'text',
        text: `Error: ${error.message}`,
      }],
      isError: true,
    };
  }
});
```

### 3. 访问本地文件

```javascript
const fs = require('fs').promises;
const path = require('path');

server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === 'read_file') {
    try {
      const filePath = path.resolve(args.path);
      const content = await fs.readFile(filePath, 'utf-8');
      
      return {
        content: [{
          type: 'text',
          text: content,
        }],
      };
    } catch (error) {
      return {
        content: [{
          type: 'text',
          text: `Error reading file: ${error.message}`,
        }],
        isError: true,
      };
    }
  }
});
```

### 4. 多个工具

```javascript
// 定义工具处理器映射
const toolHandlers = {
  async get_time() {
    return {
      content: [{
        type: 'text',
        text: new Date().toISOString(),
      }],
    };
  },
  
  async search({ query }) {
    const results = await performSearch(query);
    return {
      content: [{
        type: 'text',
        text: JSON.stringify(results),
      }],
    };
  },
  
  async calculate({ expression }) {
    try {
      const result = eval(expression);  // 注意：实际应用中避免使用 eval
      return {
        content: [{
          type: 'text',
          text: String(result),
        }],
      };
    } catch (error) {
      return {
        content: [{
          type: 'text',
          text: `Calculation error: ${error.message}`,
        }],
        isError: true,
      };
    }
  },
};

server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  const handler = toolHandlers[name];
  if (!handler) {
    throw new Error(`Unknown tool: ${name}`);
  }
  
  return await handler(args);
});

server.setRequestHandler('tools/list', async () => ({
  tools: [
    {
      name: 'get_time',
      description: '获取当前时间',
      inputSchema: { type: 'object', properties: {} },
    },
    {
      name: 'search',
      description: '搜索内容',
      inputSchema: {
        type: 'object',
        properties: {
          query: { type: 'string', description: '搜索关键词' },
        },
        required: ['query'],
      },
    },
    {
      name: 'calculate',
      description: '计算数学表达式',
      inputSchema: {
        type: 'object',
        properties: {
          expression: { type: 'string', description: '数学表达式' },
        },
        required: ['expression'],
      },
    },
  ],
}));
```

---

## 🐛 常见问题速查

### 问题 1：服务器启动失败

```bash
# 错误：command not found
# 解决：使用绝对路径
which node  # 找到 node 路径
# 然后在配置中使用：["/usr/local/bin/node", "/path/to/server.js"]
```

### 问题 2：工具未找到

```javascript
// 确保工具名称一致
// tools/list 返回的 name 必须与 tools/call 中使用的 name 匹配

// ✅ 正确
tools: [{ name: 'my_tool', ... }]
if (name === 'my_tool') { ... }

// ❌ 错误
tools: [{ name: 'my_tool', ... }]
if (name === 'myTool') { ... }  // 名称不匹配
```

### 问题 3：参数未传递

```javascript
// MCP 调用时，所有参数会合并到一个对象中

// Coze Studio 调用：
// query: { count: 10 }
// body: { keyword: 'test' }

// MCP 服务器接收：
// arguments: { count: 10, keyword: 'test' }
```

### 问题 4：调试技巧

```javascript
// 在 MCP 服务器中添加日志
console.error('[DEBUG] Received:', JSON.stringify(request, null, 2));

// 日志会输出到 stderr，不影响 MCP 协议通信（stdio 使用 stdout）
```

---

## ✅ 检查清单

创建 MCP 插件前的检查清单：

- [ ] 安装了 Node.js 环境
- [ ] 安装了 `@modelcontextprotocol/sdk` 包
- [ ] MCP 服务器脚本有执行权限 (`chmod +x`)
- [ ] 配置文件中使用了绝对路径
- [ ] 工具名称在 `tools/list` 和 `tools/call` 中一致
- [ ] 已定义 OpenAPI Schema 文件
- [ ] 已添加到 `plugin_meta.yaml`
- [ ] 重启了 Coze Studio 后端

---

## 📚 下一步

- **完整文档**：查看 `MCP_Plugin_Development_Guide.md`
- **MCP 协议**：https://modelcontextprotocol.io/
- **示例项目**：查看 `backend/conf/plugin/pluginproduct/` 目录

---

**文档版本**：v1.0  
**创建日期**：2025-11-09

