# MCP (Model Context Protocol) 集成方案

## 📋 目录

1. [当前状态分析](#当前状态分析)
2. [MCP 协议概述](#mcp-协议概述)
3. [架构设计](#架构设计)
4. [实现方案](#实现方案)
5. [实施步骤](#实施步骤)
6. [代码示例](#代码示例)

---

## 当前状态分析

### ✅ 已有基础设施

1. **插件类型定义**：
   - `PluginTypeOfMCP = "coze-studio-mcp"` 已定义
   - 插件执行系统已支持 MCP 类型路由

2. **框架代码**：
   - `backend/domain/plugin/service/tool/invocation_mcp.go` 已存在
   - 实现了 `Invocation` 接口，但 `Do` 方法返回 "not implemented"

3. **前端支持**：
   - UI 组件已存在（`mcp-config-btn.tsx`）
   - 发布配置界面已支持 MCP

### ❌ 缺失部分

1. **MCP 协议实现**：
   - 缺少 MCP 客户端实现
   - 缺少与 MCP 服务器的通信逻辑
   - 缺少工具发现和调用机制

2. **配置管理**：
   - MCP 服务器连接配置
   - 认证信息管理

---

## MCP 协议概述

### 什么是 MCP？

**Model Context Protocol (MCP)** 是一个开放标准，允许 AI 应用安全地访问外部数据源和工具。

### 核心概念

1. **MCP Server**：提供工具和资源的服务器
2. **MCP Client**：Coze Studio（我们的系统）
3. **Transport**：通信方式（stdio、HTTP、SSE）

### 标准方法

```json
// 1. 初始化
{"jsonrpc": "2.0", "method": "initialize", "params": {...}}

// 2. 列出可用工具
{"jsonrpc": "2.0", "method": "tools/list", "params": {}}

// 3. 调用工具
{"jsonrpc": "2.0", "method": "tools/call", "params": {
  "name": "tool_name",
  "arguments": {...}
}}

// 4. 列出资源
{"jsonrpc": "2.0", "method": "resources/list", "params": {}}

// 5. 读取资源
{"jsonrpc": "2.0", "method": "resources/read", "params": {
  "uri": "resource_uri"
}}
```

### 通信方式

#### 方式 1：stdio（推荐用于本地服务器）
- 通过标准输入/输出通信
- 适合本地进程

#### 方式 2：HTTP/SSE
- 通过 HTTP 请求通信
- 适合远程服务器

---

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    Coze Studio                          │
│                                                          │
│  ┌──────────────┐      ┌──────────────┐                │
│  │  Workflow    │─────▶│   Plugin     │                │
│  │   Engine     │      │   Service    │                │
│  └──────────────┘      └──────────────┘                │
│                              │                          │
│                              ▼                          │
│  ┌──────────────────────────────────────────┐        │
│  │         Plugin Execution Layer             │        │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐ │        │
│  │  │  HTTP    │  │   MCP    │  │ Custom   │ │        │
│  │  │ CallImpl │  │ CallImpl │  │ CallImpl │ │        │
│  │  └──────────┘  └──────────┘  └──────────┐ │        │
│  └──────────────────────────────────────────┘        │
│                              │                          │
└──────────────────────────────┼──────────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │   MCP Client SDK     │
                    │  (封装 MCP 协议)      │
                    └──────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    │                     │
                    ▼                     ▼
            ┌──────────────┐    ┌──────────────┐
            │  stdio       │    │  HTTP/SSE     │
            │  Transport   │    │  Transport   │
            └──────────────┘    └──────────────┘
                    │                     │
                    └──────────┬──────────┘
                               ▼
                    ┌──────────────────────┐
                    │    MCP Server        │
                    │  (外部工具提供者)      │
                    └──────────────────────┘
```

### 目录结构

```
backend/
├── domain/
│   └── plugin/
│       └── service/
│           └── tool/
│               ├── invocation_mcp.go          # MCP 调用实现（待完善）
│               └── ...
├── pkg/
│   └── mcp/                                   # 新建：MCP 客户端 SDK
│       ├── client.go                          # MCP 客户端核心
│       ├── transport.go                      # 传输层抽象
│       ├── transport_stdio.go                 # stdio 传输实现
│       ├── transport_http.go                  # HTTP 传输实现
│       ├── protocol.go                        # MCP 协议定义
│       └── errors.go                          # MCP 错误定义
└── crossdomain/
    └── plugin/
        └── model/
            └── mcp_config.go                  # MCP 配置模型
```

---

## 实现方案

### 方案 A：使用现有 Go MCP SDK（推荐）

**优点**：
- 快速实现
- 经过测试
- 符合标准

**推荐库**：
- `github.com/modelcontextprotocol/go-sdk`（如果存在）
- 或参考 Anthropic 的官方实现

### 方案 B：自研轻量级实现

**优点**：
- 完全控制
- 可定制化
- 无外部依赖

**缺点**：
- 开发时间长
- 需要维护

### 方案 C：混合方案（推荐）

**策略**：
1. 先实现核心功能（工具调用）
2. 使用标准 JSON-RPC 2.0
3. 逐步完善功能

---

## 实施步骤

### Phase 1: 基础框架（1-2 天）

#### 1.1 创建 MCP 客户端包

```bash
mkdir -p backend/pkg/mcp
```

#### 1.2 定义核心接口

```go
// backend/pkg/mcp/client.go
type Client interface {
    // 初始化连接
    Initialize(ctx context.Context, params InitializeParams) error
    
    // 列出可用工具
    ListTools(ctx context.Context) ([]Tool, error)
    
    // 调用工具
    CallTool(ctx context.Context, name string, arguments map[string]any) (*ToolResult, error)
    
    // 关闭连接
    Close() error
}
```

#### 1.3 实现传输层接口

```go
// backend/pkg/mcp/transport.go
type Transport interface {
    SendRequest(ctx context.Context, method string, params any) (*Response, error)
    Close() error
}
```

### Phase 2: 实现 stdio 传输（2-3 天）

#### 2.1 stdio 传输实现

```go
// backend/pkg/mcp/transport_stdio.go
type StdioTransport struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    // ...
}
```

#### 2.2 进程管理

- 启动 MCP 服务器进程
- 管理进程生命周期
- 错误处理和重连

### Phase 3: 实现 HTTP 传输（2-3 天）

#### 3.1 HTTP 传输实现

```go
// backend/pkg/mcp/transport_http.go
type HTTPTransport struct {
    baseURL string
    client  *http.Client
    // ...
}
```

### Phase 4: 完善 invocation_mcp.go（1-2 天）

#### 4.1 实现 Do 方法

```go
func (m *mcpCallImpl) Do(ctx context.Context, args *InvocationArgs) (request string, resp string, err error) {
    // 1. 从配置获取 MCP 服务器信息
    // 2. 创建/获取 MCP 客户端
    // 3. 调用工具
    // 4. 返回结果
}
```

### Phase 5: 配置管理（1 天）

#### 5.1 MCP 配置模型

```go
// backend/crossdomain/plugin/model/mcp_config.go
type MCPConfig struct {
    TransportType string // "stdio" | "http" | "sse"
    
    // stdio 配置
    Command      []string
    Env          map[string]string
    WorkingDir   string
    
    // HTTP 配置
    BaseURL      string
    APIKey       string
    Headers      map[string]string
    
    // 通用配置
    Timeout      time.Duration
    RetryPolicy  *RetryPolicy
}
```

### Phase 6: 测试和文档（2-3 天）

---

## 代码示例

### 示例 1: MCP 客户端核心实现

```go
// backend/pkg/mcp/client.go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
)

type Client struct {
    transport Transport
    initialized bool
}

func NewClient(transport Transport) *Client {
    return &Client{
        transport: transport,
    }
}

func (c *Client) Initialize(ctx context.Context, params InitializeParams) error {
    req := &Request{
        JSONRPC: "2.0",
        Method:  "initialize",
        Params:  params,
    }
    
    resp, err := c.transport.SendRequest(ctx, req)
    if err != nil {
        return fmt.Errorf("initialize failed: %w", err)
    }
    
    c.initialized = true
    return nil
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
    if !c.initialized {
        return nil, fmt.Errorf("client not initialized")
    }
    
    req := &Request{
        JSONRPC: "2.0",
        Method:  "tools/list",
        Params:  map[string]any{},
    }
    
    resp, err := c.transport.SendRequest(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("list tools failed: %w", err)
    }
    
    var result struct {
        Tools []Tool `json:"tools"`
    }
    
    if err := json.Unmarshal(resp.Result, &result); err != nil {
        return nil, fmt.Errorf("unmarshal tools failed: %w", err)
    }
    
    return result.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (*ToolResult, error) {
    if !c.initialized {
        return nil, fmt.Errorf("client not initialized")
    }
    
    req := &Request{
        JSONRPC: "2.0",
        Method:  "tools/call",
        Params: map[string]any{
            "name":      name,
            "arguments": arguments,
        },
    }
    
    resp, err := c.transport.SendRequest(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("call tool failed: %w", err)
    }
    
    var result ToolResult
    if err := json.Unmarshal(resp.Result, &result); err != nil {
        return nil, fmt.Errorf("unmarshal result failed: %w", err)
    }
    
    return &result, nil
}
```

### 示例 2: stdio 传输实现

```go
// backend/pkg/mcp/transport_stdio.go
package mcp

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os/exec"
    "sync"
)

type StdioTransport struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    scanner *bufio.Scanner
    mu     sync.Mutex
    reqID  int64
    pending map[int64]chan *Response
}

func NewStdioTransport(command []string, env map[string]string, workingDir string) (*StdioTransport, error) {
    if len(command) == 0 {
        return nil, fmt.Errorf("command is empty")
    }
    
    cmd := exec.Command(command[0], command[1:]...)
    cmd.Env = os.Environ()
    for k, v := range env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }
    cmd.Dir = workingDir
    
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, fmt.Errorf("create stdin pipe failed: %w", err)
    }
    
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, fmt.Errorf("create stdout pipe failed: %w", err)
    }
    
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("start command failed: %w", err)
    }
    
    t := &StdioTransport{
        cmd:     cmd,
        stdin:   stdin,
        stdout:  stdout,
        scanner: bufio.NewScanner(stdout),
        pending: make(map[int64]chan *Response),
    }
    
    // 启动响应读取 goroutine
    go t.readResponses()
    
    return t, nil
}

func (t *StdioTransport) SendRequest(ctx context.Context, req *Request) (*Response, error) {
    t.mu.Lock()
    req.ID = t.reqID
    t.reqID++
    
    ch := make(chan *Response, 1)
    t.pending[req.ID] = ch
    t.mu.Unlock()
    
    data, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal request failed: %w", err)
    }
    
    data = append(data, '\n')
    
    if _, err := t.stdin.Write(data); err != nil {
        return nil, fmt.Errorf("write request failed: %w", err)
    }
    
    select {
    case resp := <-ch:
        if resp.Error != nil {
            return nil, fmt.Errorf("mcp error: %s", resp.Error.Message)
        }
        return resp, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (t *StdioTransport) readResponses() {
    for t.scanner.Scan() {
        line := t.scanner.Bytes()
        
        var resp Response
        if err := json.Unmarshal(line, &resp); err != nil {
            continue
        }
        
        t.mu.Lock()
        ch, ok := t.pending[resp.ID]
        if ok {
            delete(t.pending, resp.ID)
        }
        t.mu.Unlock()
        
        if ok {
            ch <- &resp
        }
    }
}

func (t *StdioTransport) Close() error {
    if t.stdin != nil {
        t.stdin.Close()
    }
    if t.cmd != nil && t.cmd.Process != nil {
        return t.cmd.Process.Kill()
    }
    return nil
}
```

### 示例 3: 完善 invocation_mcp.go

```go
// backend/domain/plugin/service/tool/invocation_mcp.go
package tool

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    
    "github.com/bytedance/sonic"
    "github.com/coze-dev/coze-studio/backend/pkg/mcp"
    "github.com/coze-dev/coze-studio/backend/pkg/logs"
)

type mcpCallImpl struct {
    clients map[string]*mcp.Client // 缓存客户端，key 为配置 hash
    mu      sync.RWMutex
}

func NewMcpCallImpl() Invocation {
    return &mcpCallImpl{
        clients: make(map[string]*mcp.Client),
    }
}

func (m *mcpCallImpl) Do(ctx context.Context, args *InvocationArgs) (request string, resp string, err error) {
    // 1. 解析 MCP 配置
    mcpConfig, err := m.parseMCPConfig(args.PluginManifest)
    if err != nil {
        return "", "", fmt.Errorf("parse mcp config failed: %w", err)
    }
    
    // 2. 获取或创建 MCP 客户端
    client, err := m.getOrCreateClient(ctx, mcpConfig)
    if err != nil {
        return "", "", fmt.Errorf("get mcp client failed: %w", err)
    }
    
    // 3. 构建工具调用参数
    toolName := args.Tool.GetName()
    arguments := args.Body // MCP 工具参数通常在 Body 中
    
    // 4. 调用工具
    result, err := client.CallTool(ctx, toolName, arguments)
    if err != nil {
        return "", "", fmt.Errorf("call mcp tool failed: %w", err)
    }
    
    // 5. 序列化请求和响应
    requestJSON, _ := sonic.MarshalString(map[string]any{
        "tool":      toolName,
        "arguments": arguments,
    })
    
    responseJSON, _ := sonic.MarshalString(result)
    
    return requestJSON, responseJSON, nil
}

func (m *mcpCallImpl) parseMCPConfig(manifest *model.PluginManifest) (*mcp.Config, error) {
    // 从 manifest 中解析 MCP 配置
    // 配置可能存储在 manifest.API 的某个字段中
    
    if manifest.API == nil {
        return nil, fmt.Errorf("api config is nil")
    }
    
    // 假设配置存储在 manifest.API 的扩展字段中
    // 实际实现需要根据您的数据结构调整
    configJSON, ok := manifest.API.Extensions["mcp_config"]
    if !ok {
        return nil, fmt.Errorf("mcp config not found")
    }
    
    var config mcp.Config
    if err := json.Unmarshal([]byte(configJSON.(string)), &config); err != nil {
        return nil, fmt.Errorf("unmarshal mcp config failed: %w", err)
    }
    
    return &config, nil
}

func (m *mcpCallImpl) getOrCreateClient(ctx context.Context, config *mcp.Config) (*mcp.Client, error) {
    // 生成配置 hash 作为缓存 key
    configHash := m.hashConfig(config)
    
    // 先尝试从缓存获取
    m.mu.RLock()
    if client, ok := m.clients[configHash]; ok {
        m.mu.RUnlock()
        return client, nil
    }
    m.mu.RUnlock()
    
    // 创建新客户端
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // 双重检查
    if client, ok := m.clients[configHash]; ok {
        return client, nil
    }
    
    // 创建传输层
    var transport mcp.Transport
    var err error
    
    switch config.TransportType {
    case "stdio":
        transport, err = mcp.NewStdioTransport(
            config.Command,
            config.Env,
            config.WorkingDir,
        )
    case "http":
        transport, err = mcp.NewHTTPTransport(
            config.BaseURL,
            config.APIKey,
            config.Headers,
        )
    default:
        return nil, fmt.Errorf("unsupported transport type: %s", config.TransportType)
    }
    
    if err != nil {
        return nil, fmt.Errorf("create transport failed: %w", err)
    }
    
    // 创建客户端
    client := mcp.NewClient(transport)
    
    // 初始化
    initParams := mcp.InitializeParams{
        ProtocolVersion: "2024-11-05",
        Capabilities: mcp.ClientCapabilities{
            Tools: map[string]any{},
        },
        ClientInfo: mcp.ClientInfo{
            Name:    "coze-studio",
            Version: "1.0.0",
        },
    }
    
    if err := client.Initialize(ctx, initParams); err != nil {
        transport.Close()
        return nil, fmt.Errorf("initialize mcp client failed: %w", err)
    }
    
    // 缓存客户端
    m.clients[configHash] = client
    
    logs.CtxInfof(ctx, "created new mcp client, transport=%s", config.TransportType)
    
    return client, nil
}

func (m *mcpCallImpl) hashConfig(config *mcp.Config) string {
    // 简单的 hash 实现，实际可以使用更复杂的算法
    data, _ := json.Marshal(config)
    return fmt.Sprintf("%x", data)
}
```

---

## 推荐实施路径

### 🎯 快速启动（MVP）

**目标**：1 周内实现基础功能

1. **Day 1-2**：实现 stdio 传输和基础客户端
2. **Day 3-4**：完善 `invocation_mcp.go`
3. **Day 5**：配置管理和测试

### 🚀 完整实现

**目标**：2-3 周实现完整功能

1. **Week 1**：基础框架 + stdio 传输
2. **Week 2**：HTTP 传输 + 错误处理
3. **Week 3**：资源访问 + 测试 + 文档

---

## 注意事项

### 1. 进程管理

- MCP 服务器进程的生命周期管理
- 进程崩溃时的自动重启
- 资源清理

### 2. 错误处理

- 网络错误
- 协议错误
- 超时处理

### 3. 安全性

- 命令注入防护（stdio）
- API Key 安全存储
- 输入验证

### 4. 性能

- 客户端连接池
- 请求超时设置
- 并发控制

---

## 参考资源

1. **MCP 官方文档**：
   - https://modelcontextprotocol.io/

2. **Go JSON-RPC 2.0 库**：
   - https://github.com/gorilla/rpc

3. **进程管理**：
   - Go `os/exec` 包文档

---

## 总结

**推荐方案**：**方案 C（混合方案）**

**实施顺序**：
1. ✅ 先实现 stdio 传输（最常用）
2. ✅ 完善 `invocation_mcp.go`
3. ✅ 添加配置管理
4. ✅ 后续添加 HTTP 传输支持

**预计时间**：1-2 周（MVP），2-3 周（完整版）

**关键文件**：
- `backend/pkg/mcp/`（新建）
- `backend/domain/plugin/service/tool/invocation_mcp.go`（完善）
- `backend/crossdomain/plugin/model/mcp_config.go`（新建）

---

**文档版本**：v1.0  
**创建日期**：2025-11-08  
**维护者**：开发团队

