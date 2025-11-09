# Pairat Remote Exec MCP 插件使用指南

## 📋 概述

这是一个基于 MCP SSE 传输协议的远程命令执行插件，允许你通过 Coze Studio 在远程机器上安全地执行命令和脚本。

---

## 🔧 配置文件

### 1. OpenAPI Schema

文件：`backend/conf/plugin/pluginproduct/pairat_remote_exec.yaml`

定义了两个工具：
- `remote_exec` - 远程执行命令
- `vmcheck` - 检查虚拟机硬件要求

### 2. 插件元数据

文件：`backend/conf/plugin/pluginproduct/plugin_meta.yaml`

```yaml
- plugin_id: 101
  version: v1.0.0
  openapi_doc_file: pairat_remote_exec.yaml
  manifest:
    name_for_model: pairat_remote_exec
    name_for_human: Pairat Remote Exec
    api:
      type: coze-studio-mcp          # ← MCP 类型
      extensions:
        mcp_config:
          transport_type: sse         # ← SSE 传输
          sse_config:
            url: http://10.1.16.4:8000/mcp/sse
            headers:
              Content-Type: application/json
  tools:
    - tool_id: 101001               # remote_exec
    - tool_id: 101002               # vmcheck
```

---

## 🚀 使用方法

### 方式 1：在 Workflow 中使用

#### 步骤 1：创建 Workflow

1. 打开 Coze Studio
2. 创建新的 Workflow
3. 添加 LLM 节点

#### 步骤 2：添加插件节点

1. 点击"添加节点" → "插件"
2. 搜索"Pairat Remote Exec"
3. 选择工具：
   - `remote_exec` - 执行远程命令
   - `vmcheck` - 检查虚拟机

#### 步骤 3：配置参数

**remote_exec 工具**：
```json
{
  "machineId": "75590566982b48729186ce5be91f2352",
  "script": "ls -la"
}
```

**vmcheck 工具**：
```json
{
  "machine_ids": "id1,id2,id3"
}
```

#### 步骤 4：运行测试

点击"运行"按钮，查看执行结果。

---

### 方式 2：在 Agent 中使用（Function Calling）

创建一个 Agent，添加 Pairat Remote Exec 插件作为工具：

```
用户提问：请帮我在服务器 75590566982b48729186ce5be91f2352 上查看磁盘使用情况

Agent 自动调用：
- 工具：remote_exec
- 参数：{
    "machineId": "75590566982b48729186ce5be91f2352",
    "script": "df -h"
  }
- 结果：返回磁盘使用情况
```

---

## 📝 API 参考

### 1. remote_exec - 远程执行命令

**请求参数**：
```json
{
  "machineId": "string",    // 机器 ID (必填)
  "script": "string"        // 要执行的命令 (必填，需要在白名单中)
}
```

**响应**：
```json
{
  "success": true,
  "output": "命令输出内容",
  "error": null,
  "exitCode": 0
}
```

**示例 1：查看系统信息**
```json
{
  "machineId": "75590566982b48729186ce5be91f2352",
  "script": "uname -a"
}
```

**示例 2：检查进程**
```json
{
  "machineId": "75590566982b48729186ce5be91f2352",
  "script": "ps aux | grep nginx"
}
```

**示例 3：查看网络状态**
```json
{
  "machineId": "75590566982b48729186ce5be91f2352",
  "script": "netstat -tuln"
}
```

---

### 2. vmcheck - 检查虚拟机硬件

**请求参数**：
```json
{
  "machine_ids": "string"   // 机器 ID 列表，逗号分隔 (必填)
}
```

**响应**：
```json
{
  "success": true,
  "results": {
    "id1": {
      "cpu": "ok",
      "memory": "ok",
      "disk": "ok"
    },
    "id2": {
      "cpu": "insufficient",
      "memory": "ok",
      "disk": "ok"
    }
  },
  "message": "检查完成"
}
```

**示例：检查多台机器**
```json
{
  "machine_ids": "machine1,machine2,machine3"
}
```

---

## 🔍 调试技巧

### 1. 查看 MCP 连接状态

```bash
# 测试 SSE 端点
curl -N -H "Accept: text/event-stream" http://10.1.16.4:8000/mcp/sse
```

### 2. 查看后端日志

```bash
# 查看 Coze Studio 后端日志
cd /path/to/coze-studio/backend
tail -f logs/app.log | grep -i "mcp\|remote_exec"
```

### 3. 验证 MCP 服务器响应

使用 MCP Inspector 或者直接测试：

```bash
# 发送 tools/list 请求
curl -X POST http://10.1.16.4:8000/mcp/sse \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

---

## ⚠️ 安全注意事项

### 1. 命令白名单

MCP 服务器应该实现命令白名单机制，只允许执行预定义的安全命令。

### 2. 机器 ID 验证

确保 `machineId` 参数经过验证，防止未授权访问。

### 3. 输出过滤

对命令输出进行适当的过滤，避免泄露敏感信息。

### 4. 日志记录

所有远程命令执行都应该被记录，用于审计。

---

## 🐛 常见问题

### 问题 1：连接失败

**症状**：
```
Error: connect to SSE server failed
```

**解决方案**：
1. 检查 URL 是否正确：`http://10.1.16.4:8000/mcp/sse`
2. 确认 MCP 服务器正在运行
3. 检查网络连接和防火墙设置

```bash
# 测试连接
ping 10.1.16.4
curl http://10.1.16.4:8000/mcp/sse
```

---

### 问题 2：工具未找到

**症状**：
```
Error: tool 'remote_exec' not found
```

**解决方案**：
1. 确认 MCP 服务器已实现该工具
2. 检查工具名称是否匹配

```bash
# 列出可用工具
curl -X POST http://10.1.16.4:8000/mcp/sse \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

---

### 问题 3：命令执行失败

**症状**：
```
Error: command not in whitelist
```

**解决方案**：
- 确认命令在白名单中
- 联系管理员添加命令到白名单
- 使用预定义的安全命令

---

## 📊 性能优化

### 1. 连接复用

MCP 客户端会自动缓存 SSE 连接，避免重复建立连接。

### 2. 超时设置

```yaml
# 在 mcp_config 中添加超时设置
sse_config:
  url: http://10.1.16.4:8000/mcp/sse
  timeout: 30s                    # 连接超时
  read_timeout: 60s               # 读取超时
```

### 3. 重试策略

```yaml
# 添加重试配置
sse_config:
  url: http://10.1.16.4:8000/mcp/sse
  retry:
    max_attempts: 3
    backoff: exponential
```

---

## 🔄 与其他插件对比

| 特性 | Pairat Remote Exec (MCP) | 传统 HTTP API 插件 |
|------|-------------------------|-------------------|
| **协议** | MCP SSE | HTTP/REST |
| **连接方式** | 持久连接 | 短连接 |
| **实时性** | 高（事件流） | 低 |
| **状态管理** | ✅ 支持 | ❌ 无状态 |
| **工具发现** | 动态 | 静态 |
| **资源访问** | ✅ 支持 | ❌ 不支持 |

---

## 📚 相关文档

- [MCP 插件开发完整指南](./MCP_Plugin_Development_Guide.md)
- [MCP 插件快速参考](./MCP_Plugin_Quick_Reference.md)
- [MCP 集成指南](./MCP_Integration_Guide.md)
- [MCP 协议官方文档](https://modelcontextprotocol.io/)

---

## 📞 技术支持

如有问题，请参考：

1. 查看日志：`backend/logs/app.log`
2. 检查配置：`backend/conf/plugin/pluginproduct/`
3. 测试连接：使用 curl 测试 SSE 端点
4. 查阅文档：参考上述相关文档

---

**文档版本**：v1.0  
**创建日期**：2025-11-09  
**MCP 服务器**：http://10.1.16.4:8000/mcp/sse

