# MCP 插件配置指南 - 在 Function Calling 中使用

## 🎯 目标

将 MCP 服务器配置为 Coze Studio 插件，在大模型的 Function Calling 中使用，就像 `top_news` 那样。

---

## 📋 配置流程

### 步骤 1：准备 MCP 配置

根据您的 MCP 服务器类型，准备相应的配置：

#### SSE 方式（您的场景）

```json
{
  "transport_type": "sse",
  "sse_config": {
    "url": "http://10.1.16.4:8000/mcp/sse"
  }
}
```

#### Stdio 方式

```json
{
  "transport_type": "stdio",
  "stdio_config": {
    "command": ["node", "/path/to/mcp-server.js"],
    "env": {
      "NODE_ENV": "production"
    }
  }
}
```

---

## 🔧 方式 1：通过数据库直接配置（快速测试）

### 1. 连接到数据库

```sql
-- 假设使用 PostgreSQL
psql -h localhost -U your_user -d coze_studio
```

### 2. 创建 MCP 插件

```sql
-- 插入插件记录
INSERT INTO plugin (
    space_id,
    project_id,
    name_for_model,
    name_for_human,
    description_for_model,
    description_for_human,
    api_type,
    created_at,
    updated_at
) VALUES (
    7547331761605181440,  -- 您的 space_id
    7569925343595724800,  -- 您的 project_id
    'mcp_remote_tools',
    'MCP 远程工具',
    '{"transport_type":"sse","sse_config":{"url":"http://10.1.16.4:8000/mcp/sse"}}',
    'MCP SSE 服务器提供的工具',
    'coze-studio-mcp',
    NOW(),
    NOW()
) RETURNING id;

-- 记录返回的 plugin_id，例如：123456789
```

### 3. 从 MCP 服务器获取工具列表

```bash
# 手动测试 MCP 服务器
curl -N -H "Accept: text/event-stream" http://10.1.16.4:8000/mcp/sse

# 发送 tools/list 请求
curl -X POST http://10.1.16.4:8000/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }'
```

### 4. 为每个工具创建记录

```sql
-- 为 MCP 服务器提供的每个工具创建记录
INSERT INTO tool (
    plugin_id,
    name,
    description,
    parameters_schema,
    activated_status,
    created_at,
    updated_at
) VALUES (
    123456789,  -- 上面返回的 plugin_id
    'execute_command',  -- 工具名称
    '执行远程命令',  -- 工具描述
    '{"type":"object","properties":{"command":{"type":"string","description":"要执行的命令"}}}',
    'activated',
    NOW(),
    NOW()
);
```

---

## 🖥️ 方式 2：通过 API 配置（推荐）

创建一个脚本来配置 MCP 插件：

```bash
#!/bin/bash

SPACE_ID="7547331761605181440"
PROJECT_ID="7569925343595724800"
MCP_URL="http://10.1.16.4:8000/mcp/sse"
API_BASE="http://localhost:8080"

# 1. 创建 MCP 插件
PLUGIN_RESPONSE=$(curl -s -X POST "${API_BASE}/api/v1/plugin/create" \
  -H "Content-Type: application/json" \
  -d "{
    \"space_id\": \"${SPACE_ID}\",
    \"project_id\": \"${PROJECT_ID}\",
    \"name_for_model\": \"mcp_remote_tools\",
    \"name_for_human\": \"MCP 远程工具\",
    \"description_for_model\": \"{\\\"transport_type\\\":\\\"sse\\\",\\\"sse_config\\\":{\\\"url\\\":\\\"${MCP_URL}\\\"}}\",
    \"description_for_human\": \"通过 MCP 连接的远程工具\",
    \"api\": {
      \"type\": \"coze-studio-mcp\"
    }
  }")

echo "Plugin created: $PLUGIN_RESPONSE"

# 提取 plugin_id
PLUGIN_ID=$(echo $PLUGIN_RESPONSE | jq -r '.data.plugin_id')

# 2. 获取 MCP 工具列表
MCP_TOOLS=$(curl -s -X POST "${MCP_URL%/sse}/message" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }' | jq -r '.result.tools')

echo "MCP Tools: $MCP_TOOLS"

# 3. 为每个工具创建记录
echo "$MCP_TOOLS" | jq -c '.[]' | while read tool; do
  TOOL_NAME=$(echo $tool | jq -r '.name')
  TOOL_DESC=$(echo $tool | jq -r '.description')
  TOOL_SCHEMA=$(echo $tool | jq -r '.inputSchema')
  
  curl -s -X POST "${API_BASE}/api/v1/tool/create" \
    -H "Content-Type: application/json" \
    -d "{
      \"plugin_id\": \"${PLUGIN_ID}\",
      \"name\": \"${TOOL_NAME}\",
      \"description\": \"${TOOL_DESC}\",
      \"parameters_schema\": ${TOOL_SCHEMA}
    }"
done

echo "MCP Plugin configured with ID: ${PLUGIN_ID}"
```

---

## 📝 在 Workflow 中使用

### 步骤 1：配置大模型节点

1. 打开您的 Workflow
2. 编辑大模型节点
3. 在 Function Calling 配置中：
   - 点击 "添加插件"
   - 选择您创建的 MCP 插件：`mcp_remote_tools`
   - 选择要使用的工具

### 步骤 2：配置用户提示词

在大模型节点的"用户提示词"中填入：

```
{{input}}

请使用可用的工具来帮助回答用户的问题。
```

### 步骤 3：测试

运行 Workflow，输入：

```
请执行命令查看当前时间
```

大模型会：
1. 识别需要使用 `execute_command` 工具
2. 通过 MCP 连接到您的服务器
3. 执行命令
4. 返回结果

---

## 🔍 完整示例：配置您的 MCP SSE 服务器

### 1. 创建配置脚本

```bash
cat > /tmp/setup_mcp_plugin.sh << 'EOF'
#!/bin/bash

# 配置
SPACE_ID="7547331761605181440"
PROJECT_ID="7569925343595724800"
MCP_SSE_URL="http://10.1.16.4:8000/mcp/sse"
MCP_MESSAGE_URL="http://10.1.16.4:8000/mcp/message"

# 创建 MCP 配置 JSON
MCP_CONFIG=$(cat <<JSON
{
  "transport_type": "sse",
  "sse_config": {
    "url": "$MCP_SSE_URL"
  }
}
JSON
)

echo "MCP Config: $MCP_CONFIG"

# 在 Coze Studio 中创建插件记录
# 注意：需要根据实际 API 调整
echo "请在 Coze Studio 中创建插件，配置如下："
echo "- 名称：MCP 远程工具"
echo "- 类型：coze-studio-mcp"
echo "- 描述（DescriptionForModel）：$MCP_CONFIG"

# 测试连接
echo ""
echo "测试 MCP 服务器连接..."
curl -X POST "$MCP_MESSAGE_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }' | jq .

EOF

chmod +x /tmp/setup_mcp_plugin.sh
bash /tmp/setup_mcp_plugin.sh
```

### 2. 在 Coze Studio UI 中配置

由于当前没有专门的 MCP 插件配置界面，您需要：

**临时方案**：
1. 进入插件开发界面
2. 创建新插件
3. 在"名称"中输入：`mcp_remote_tools`
4. 在"描述（给模型）"字段中粘贴 MCP 配置 JSON：
   ```json
   {"transport_type":"sse","sse_config":{"url":"http://10.1.16.4:8000/mcp/sse"}}
   ```
5. 插件类型选择：`coze-studio-mcp`
6. 保存插件

### 3. 手动添加工具

为 MCP 服务器提供的每个工具：
1. 点击"添加工具"
2. 填写工具名称（从 MCP `tools/list` 获取）
3. 填写工具描述
4. 配置参数 Schema

---

## 🎨 与 top_news 对比

| 项目 | top_news（HTTP 插件） | MCP SSE 插件 |
|------|---------------------|-------------|
| **插件类型** | `openapi` | `coze-studio-mcp` |
| **连接方式** | HTTP REST API | MCP SSE |
| **配置位置** | OpenAPI Schema | DescriptionForModel |
| **工具发现** | 手动配置 | MCP tools/list |
| **调用方式** | HTTP POST | MCP tools/call |
| **在 FC 中使用** | ✅ 相同 | ✅ 相同 |

两者在 Function Calling 中的使用方式**完全一致**，大模型会根据工具描述自动选择合适的工具调用。

---

## 🚀 快速测试

```bash
# 1. 测试您的 MCP 服务器
curl -N -H "Accept: text/event-stream" http://10.1.16.4:8000/mcp/sse &
SSE_PID=$!

# 2. 发送 tools/list 请求
curl -X POST http://10.1.16.4:8000/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }' | jq .

# 3. 测试工具调用
curl -X POST http://10.1.16.4:8000/mcp/message \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "your_tool_name",
      "arguments": {}
    }
  }' | jq .

# 清理
kill $SSE_PID
```

---

## 🐛 故障排查

### 问题 1：无法连接到 MCP 服务器

```bash
# 检查服务器是否运行
curl -I http://10.1.16.4:8000/mcp/sse

# 检查防火墙
telnet 10.1.16.4 8000
```

### 问题 2：工具未显示在 Function Calling 中

- 确认插件已保存
- 检查工具是否激活
- 查看后端日志：`grep MCP /path/to/coze-studio.log`

### 问题 3：工具调用失败

```bash
# 启用调试日志
export LOG_LEVEL=debug

# 查看详细错误
tail -f /path/to/coze-studio.log | grep -i mcp
```

---

## 📚 参考

- MCP 协议：https://modelcontextprotocol.io/
- mcp-go 库：https://github.com/mark3labs/mcp-go
- SSE 规范：https://html.spec.whatwg.org/multipage/server-sent-events.html

---

**文档版本**：v1.0  
**创建日期**：2025-11-09  
**适用场景**：将 MCP SSE 服务器配置为 Coze Studio Function Calling 插件

