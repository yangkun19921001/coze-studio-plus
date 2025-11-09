# MCP 插件开发完整指南

**基于 Coze Studio 的 MCP 插件开发解决方案**

---

## 📋 目录

1. [插件系统架构分析](#插件系统架构分析)
2. [传统 OpenAPI 插件开发流程](#传统-openapi-插件开发流程)
3. [MCP 插件开发流程](#mcp-插件开发流程)
4. [开发实战：创建一个 MCP 新闻插件](#开发实战创建一个-mcp-新闻插件)
5. [MCP vs OpenAPI 插件对比](#mcp-vs-openapi-插件对比)
6. [故障排查与最佳实践](#故障排查与最佳实践)

---

## 插件系统架构分析

### 1. 插件执行流程

Coze Studio 的插件执行系统采用分层架构设计：

```
┌──────────────────────────────────────────────────────────────┐
│                    应用层 (Application)                        │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Workflow Engine / Agent / Function Calling             │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────┬───────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│              领域层 (Domain) - Plugin Service                 │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  ExecuteTool (backend/domain/plugin/service/exec_tool.go│ │
│  │  ├─ buildToolExecutor()    // 构建执行器                  │ │
│  │  ├─ acquireAccessToken()   // 获取授权令牌                │ │
│  │  └─ executor.execute()     // 执行工具                    │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────┬───────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│           工具调用层 (Tool Invocation) - 策略模式             │
│  ┌────────────┐   ┌────────────┐   ┌────────────┐          │
│  │   HTTP     │   │    MCP     │   │  Custom    │          │
│  │ CallImpl   │   │ CallImpl   │   │  CallImpl  │          │
│  └────────────┘   └────────────┘   └────────────┘          │
│       │                  │                 │                  │
└───────┼──────────────────┼─────────────────┼─────────────────┘
        │                  │                 │
        ▼                  ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  HTTP API    │  │  MCP Server  │  │ Custom Logic │
│  (REST/RPC)  │  │  (stdio/SSE) │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
```

### 2. 核心类型定义

```go
// backend/crossdomain/plugin/consts/consts.go

type PluginType string

const (
    PluginTypeOfCloud  PluginType = "openapi"              // 标准 OpenAPI 插件
    PluginTypeOfMCP    PluginType = "coze-studio-mcp"      // MCP 协议插件
    PluginTypeOfCustom PluginType = "coze-studio-custom"   // 自定义插件
)
```

### 3. 插件路由机制

```go
// backend/domain/plugin/service/exec_tool.go:597

func newToolInvocation(t *toolExecutor) tool.Invocation {
    switch t.plugin.Manifest.API.Type {
    case consts.PluginTypeOfCloud:
        // HTTP API 调用（如搜狐热闻）
        return tool.NewHttpCallImpl(t.conversationID)
    case consts.PluginTypeOfMCP:
        // MCP 协议调用
        return tool.NewMcpCallImpl()
    case consts.PluginTypeOfCustom:
        // 自定义调用
        return tool.NewCustomCallImpl()
    default:
        return tool.NewHttpCallImpl(t.conversationID)
    }
}
```

---

## 传统 OpenAPI 插件开发流程

### 示例：搜狐热闻插件分析

#### 1. 插件配置文件结构

```yaml
# backend/conf/plugin/pluginproduct/plugin_meta.yaml

- plugin_id: 6
  product_id: 7343894357063385100
  deprecated: false
  version: v1.0.0
  openapi_doc_file: sohu_hot_news.yaml      # OpenAPI 文档文件
  plugin_type: 1                             # OpenAPI 类型
  manifest:
    schema_version: v1
    name_for_model: souhurewen               # AI 模型识别名称
    name_for_human: 搜狐热闻                  # 用户显示名称
    description_for_model: 帮助用户获取搜狐网上的每日热闻
    description_for_human: 帮助用户获取搜狐网上的每日热闻
    auth:
      type: none                             # 无需认证
    logo_url: official_plugin_icon/plugin_sohu_hot_news.png
    api:
      type: openapi                          # API 类型
    common_params:
      body: []
      header:
        - name: User-Agent
          value: Coze/1.0
      path: []
      query: []
  tools:
    - tool_id: 60001
      deprecated: false
      method: get
      sub_url: /blog/outer/temp/feeds/ark
```

#### 2. OpenAPI 文档定义

```yaml
# backend/conf/plugin/pluginproduct/sohu_hot_news.yaml

info:
  description: 帮助用户获取搜狐网上的每日热闻
  title: 搜狐热闻
  version: v1
openapi: 3.0.1

servers:
  - url: https://uis.mp.sohu.com           # API 服务器地址

paths:
  /blog/outer/temp/feeds/ark:
    get:
      operationId: top_news                 # 工具唯一标识符
      summary: 帮助用户获取搜狐网上的每日热闻
      
      parameters:                           # 请求参数
        - description: 获取新闻条数
          in: query
          name: count
          required: true
          schema:
            description: 获取新闻条数
            type: integer
        
        - description: 搜索关键词
          in: query
          name: q
          schema:
            description: 搜索关键词
            type: string
      
      responses:                            # 响应定义
        "200":
          content:
            application/json:
              schema:
                properties:
                  code:
                    type: number
                  data:
                    properties:
                      coze_ark_001:
                        properties:
                          list:
                            items:
                              properties:
                                brief:
                                  type: string
                                title:
                                  type: string
                                url:
                                  type: string
                              type: object
                            type: array
                        type: object
                    type: object
                  message:
                    type: string
                  success:
                    type: boolean
                type: object
```

#### 3. HTTP 调用实现机制

```go
// backend/domain/plugin/service/tool/invocation_http.go

type httpCallImpl struct {
    ConversationID int64
}

func (h *httpCallImpl) Do(ctx context.Context, args *InvocationArgs) (request string, resp string, err error) {
    // 1. 构建 HTTP 请求
    httpReq, err := h.buildHTTPRequest(ctx, args)
    if err != nil {
        return "", "", err
    }
    
    // 2. 注入认证信息（OAuth/API Key等）
    errMsg, err := h.injectAuthInfo(ctx, httpReq, args)
    if err != nil {
        return "", "", err
    }
    
    // 3. 发送 HTTP 请求
    httpResp, err := restyReq.Send()
    if err != nil {
        return "", "", errorx.New(errno.ErrPluginExecuteToolFailed, ...)
    }
    
    // 4. 处理响应
    if httpResp.StatusCode() != http.StatusOK {
        return "", "", errorx.New(errno.ErrPluginExecuteToolFailed, ...)
    }
    
    return requestStr, httpResp.String(), nil
}

func (h *httpCallImpl) buildHTTPRequest(ctx context.Context, args *InvocationArgs) (*http.Request, error) {
    // 1. 构建请求 URL
    rawURL := args.ServerURL + tool.GetSubURL()
    reqURL, err := h.buildHTTPRequestURL(ctx, rawURL, args)
    
    // 2. 构建请求体
    bodyBytes, contentType, err := h.buildRequestBody(ctx, tool.Operation, args.Body)
    
    // 3. 创建 HTTP 请求
    httpReq, err := http.NewRequestWithContext(ctx, tool.GetMethod(), reqURL.String(), bytes.NewBuffer(bodyBytes))
    
    // 4. 添加请求头
    httpReq.Header, err = h.buildHTTPRequestHeader(ctx, args)
    
    return httpReq, nil
}
```

#### 4. OpenAPI 插件开发步骤总结

1. **创建 OpenAPI 文档** (`sohu_hot_news.yaml`)
   - 定义 API 服务器地址
   - 定义接口路径和方法
   - 定义请求参数和响应格式

2. **配置插件元数据** (`plugin_meta.yaml`)
   - 设置插件基本信息
   - 配置认证方式
   - 关联 OpenAPI 文档
   - 定义工具列表

3. **系统自动处理**
   - HTTP 请求构建
   - 参数序列化
   - 认证信息注入
   - 响应解析

---

## MCP 插件开发流程

### MCP 插件架构

```
┌──────────────────────────────────────────────────────────────┐
│                    Coze Studio (MCP Client)                   │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  backend/pkg/mcp/client.go (封装层)                     │  │
│  │  ├─ NewClient(config)      // 创建客户端                 │  │
│  │  ├─ Initialize(ctx)        // 初始化连接                 │  │
│  │  ├─ CallTool(name, args)   // 调用工具                   │  │
│  │  ├─ ListTools(ctx)         // 列出工具                   │  │
│  │  └─ Close()                // 关闭连接                   │  │
│  └────────────────────────────────────────────────────────┘  │
│                           │                                   │
│                           ▼                                   │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  mcp-go Library (github.com/mark3labs/mcp-go)          │  │
│  │  ├─ StdioServerTransport   // stdio 传输                │  │
│  │  ├─ SSEMCPClient           // SSE 传输                  │  │
│  │  └─ MCP Protocol Handler   // 协议处理                  │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────┬───────────────────────────────────┘
                           │
                ┌──────────┴──────────┐
                │                     │
                ▼                     ▼
        ┌──────────────┐      ┌──────────────┐
        │   stdio      │      │     SSE      │
        │  (子进程)     │      │  (HTTP/SSE)  │
        └──────────────┘      └──────────────┘
                │                     │
                └──────────┬──────────┘
                           ▼
                ┌──────────────────────┐
                │    MCP Server        │
                │  (Node.js/Python)    │
                │  ├─ tools/list       │
                │  ├─ tools/call       │
                │  ├─ resources/list   │
                │  └─ resources/read   │
                └──────────────────────┘
```

### MCP 插件配置格式

```yaml
# 假设：backend/conf/plugin/pluginproduct/plugin_meta.yaml

- plugin_id: 100
  product_id: 7500000000000000000
  deprecated: false
  version: v1.0.0
  openapi_doc_file: mcp_news_tool.yaml      # 仍需要定义工具 schema
  plugin_type: 1
  manifest:
    schema_version: v1
    name_for_model: mcp_news_plugin
    name_for_human: MCP 新闻插件
    description_for_model: 通过 MCP 协议获取新闻数据
    description_for_human: 基于 MCP 协议的新闻获取工具
    auth:
      type: none
    logo_url: official_plugin_icon/plugin_mcp_news.png
    api:
      type: coze-studio-mcp                   # MCP 类型！
      extensions:
        mcp_config:                           # MCP 配置
          transport_type: stdio               # 传输类型
          stdio_config:
            command: ["node", "/path/to/mcp-news-server.js"]
            env:
              NODE_ENV: production
              API_KEY: "your_api_key_here"
            working_dir: /path/to/working/dir
    common_params:
      body: []
      header: []
      path: []
      query: []
  tools:
    - tool_id: 100001
      deprecated: false
      method: get                            # 仅用于 schema，实际调用走 MCP
      sub_url: /get_news                     # 仅用于 schema
```

### MCP 调用实现

```go
// backend/domain/plugin/service/tool/invocation_mcp.go

type mcpCallImpl struct {
    clients map[string]*mcp.Client   // 客户端缓存池
    mu      sync.RWMutex
}

func NewMcpCallImpl() Invocation {
    return &mcpCallImpl{
        clients: make(map[string]*mcp.Client),
    }
}

func (m *mcpCallImpl) Do(ctx context.Context, args *InvocationArgs) (request string, resp string, err error) {
    // 1. 解析 MCP 配置
    mcpConfig, err := m.parseMCPConfig(args)
    if err != nil {
        return "", "", fmt.Errorf("parse mcp config failed: %w", err)
    }
    
    // 2. 获取或创建 MCP 客户端（带缓存）
    client, err := m.getOrCreateClient(ctx, mcpConfig)
    if err != nil {
        return "", "", fmt.Errorf("get mcp client failed: %w", err)
    }
    
    // 3. 构建工具调用参数
    toolName := args.Tool.GetName()
    if toolName == "" {
        toolName = fmt.Sprintf("tool_%d", args.Tool.ID)
    }
    
    // 合并所有参数（Header, Query, Path, Body）
    arguments := make(map[string]any)
    for k, v := range args.Header {
        arguments[k] = v
    }
    for k, v := range args.Query {
        arguments[k] = v
    }
    for k, v := range args.Path {
        arguments[k] = v
    }
    for k, v := range args.Body {
        arguments[k] = v
    }
    
    // 4. 调用 MCP 工具
    resultStr, err := client.CallTool(ctx, toolName, arguments)
    if err != nil {
        requestJSON, _ := sonic.MarshalString(map[string]any{
            "tool":      toolName,
            "arguments": arguments,
        })
        return requestJSON, "", fmt.Errorf("call mcp tool failed: %w", err)
    }
    
    // 5. 序列化请求和响应
    requestJSON, _ := sonic.MarshalString(map[string]any{
        "tool":      toolName,
        "arguments": arguments,
    })
    
    return requestJSON, resultStr, nil
}

func (m *mcpCallImpl) parseMCPConfig(args *InvocationArgs) (*mcp.Config, error) {
    if args.PluginManifest == nil {
        return nil, fmt.Errorf("plugin manifest is nil")
    }
    
    // 从 manifest.api.extensions.mcp_config 获取配置
    desc := args.PluginManifest.DescriptionForModel
    if desc == "" {
        return nil, fmt.Errorf("mcp_config not found in manifest")
    }
    
    // 解析为 JSON
    var config mcp.Config
    if err := json.Unmarshal([]byte(desc), &config); err != nil {
        // 如果不是 JSON，使用默认 stdio 配置
        return &mcp.Config{
            TransportType: mcp.TransportTypeStdio,
            StdioConfig: &mcp.StdioConfig{
                Command: []string{"node", desc},
            },
        }, nil
    }
    
    return &config, nil
}

func (m *mcpCallImpl) getOrCreateClient(ctx context.Context, config *mcp.Config) (*mcp.Client, error) {
    configHash := m.hashConfig(config)
    
    // 尝试从缓存获取
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
    
    // 使用 mcp-go 库创建客户端
    client, err := mcp.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("create mcp client failed: %w", err)
    }
    
    // 初始化客户端
    if err := client.Initialize(ctx); err != nil {
        client.Close()
        return nil, fmt.Errorf("initialize mcp client failed: %w", err)
    }
    
    // 缓存客户端
    m.clients[configHash] = client
    
    logs.Infof("[MCP] Created new client: transport=%s, hash=%s", config.TransportType, configHash[:8])
    
    return client, nil
}

func (m *mcpCallImpl) hashConfig(config *mcp.Config) string {
    data, _ := json.Marshal(config)
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}
```

### MCP 客户端封装层

```go
// backend/pkg/mcp/client.go

package mcp

import (
    "context"
    "fmt"
    "strings"
    
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
)

type TransportType string

const (
    TransportTypeStdio TransportType = "stdio"
    TransportTypeSSE   TransportType = "sse"
)

type Config struct {
    TransportType TransportType `json:"transport_type"`
    StdioConfig   *StdioConfig  `json:"stdio_config,omitempty"`
    SSEConfig     *SSEConfig    `json:"sse_config,omitempty"`
}

type StdioConfig struct {
    Command    []string          `json:"command"`
    Env        map[string]string `json:"env,omitempty"`
    WorkingDir string            `json:"working_dir,omitempty"`
}

type SSEConfig struct {
    URL     string            `json:"url"`
    APIKey  string            `json:"api_key,omitempty"`
    Headers map[string]string `json:"headers,omitempty"`
}

type Client struct {
    client *client.Client
}

func NewClient(config *Config) (*Client, error) {
    if config == nil {
        return nil, fmt.Errorf("config is nil")
    }
    
    switch config.TransportType {
    case TransportTypeStdio:
        if config.StdioConfig == nil {
            return nil, fmt.Errorf("stdio config is required")
        }
        return newStdioClient(config.StdioConfig)
    case TransportTypeSSE:
        if config.SSEConfig == nil {
            return nil, fmt.Errorf("sse config is required")
        }
        return newSSEClient(config.SSEConfig)
    default:
        return nil, fmt.Errorf("unsupported transport type: %s", config.TransportType)
    }
}

func newStdioClient(config *StdioConfig) (*Client, error) {
    if len(config.Command) == 0 {
        return nil, fmt.Errorf("command is empty")
    }
    
    // 构建环境变量
    envVars := []string{}
    for k, v := range config.Env {
        envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
    }
    
    // 使用 mcp-go 创建 stdio 客户端
    mcpClient, err := client.NewStdioMCPClient(
        config.Command[0],
        envVars,
        config.Command[1:]...,
    )
    if err != nil {
        return nil, fmt.Errorf("create stdio client failed: %w", err)
    }
    
    return &Client{client: mcpClient}, nil
}

func newSSEClient(config *SSEConfig) (*Client, error) {
    if config.URL == "" {
        return nil, fmt.Errorf("url is empty")
    }
    
    mcpClient, err := client.NewSSEMCPClient(config.URL)
    if err != nil {
        return nil, fmt.Errorf("create sse client failed: %w", err)
    }
    
    return &Client{client: mcpClient}, nil
}

func (c *Client) Initialize(ctx context.Context) error {
    if c.client == nil {
        return fmt.Errorf("client is nil")
    }
    
    // 验证连接（通过列出工具）
    _, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
    if err != nil {
        return fmt.Errorf("initialize failed: %w", err)
    }
    
    return nil
}

func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (string, error) {
    if c.client == nil {
        return "", fmt.Errorf("client is nil")
    }
    
    result, err := c.client.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name:      name,
            Arguments: arguments,
        },
    })
    if err != nil {
        return "", fmt.Errorf("call tool failed: %w", err)
    }
    
    // 转换结果为字符串
    return convertToolResultToString(result), nil
}

func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
    if c.client == nil {
        return nil, fmt.Errorf("client is nil")
    }
    
    result, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
    if err != nil {
        return nil, err
    }
    
    return result.Tools, nil
}

func (c *Client) Close() error {
    if c.client != nil {
        return c.client.Close()
    }
    return nil
}

func convertToolResultToString(result *mcp.CallToolResult) string {
    if result == nil {
        return ""
    }
    
    // 处理错误
    if result.IsError {
        if len(result.Content) > 0 {
            if textContent, ok := result.Content[0].(mcp.TextContent); ok {
                return "Error: " + textContent.Text
            }
        }
        return "Error occurred"
    }
    
    // 提取文本内容
    var textParts []string
    for _, content := range result.Content {
        switch c := content.(type) {
        case mcp.TextContent:
            textParts = append(textParts, c.Text)
        case mcp.ImageContent:
            textParts = append(textParts, fmt.Sprintf("[Image: %s]", c.MIMEType))
        case mcp.EmbeddedResource:
            uri := ""
            if blob, ok := c.Resource.(mcp.BlobResourceContents); ok {
                uri = blob.URI
            } else if text, ok := c.Resource.(mcp.TextResourceContents); ok {
                uri = text.URI
            }
            textParts = append(textParts, fmt.Sprintf("[Resource: %s]", uri))
        }
    }
    
    if len(textParts) == 0 {
        return ""
    }
    
    return strings.Join(textParts, "\n")
}
```

---

## 开发实战：创建一个 MCP 新闻插件

### 场景描述

我们要创建一个类似"搜狐热闻"功能的 MCP 插件，但使用 MCP 协议实现。

### 步骤 1：创建 MCP 服务器

```javascript
#!/usr/bin/env node
// mcp-news-server.js

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');
const https = require('https');

// 创建 MCP 服务器
const server = new Server(
  {
    name: 'news-mcp-server',
    version: '1.0.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// 注册工具列表处理器
server.setRequestHandler('tools/list', async () => {
  return {
    tools: [
      {
        name: 'get_hot_news',
        description: '获取热门新闻列表',
        inputSchema: {
          type: 'object',
          properties: {
            count: {
              type: 'number',
              description: '要获取的新闻数量',
              default: 10,
            },
            keyword: {
              type: 'string',
              description: '搜索关键词（可选）',
            },
          },
          required: ['count'],
        },
      },
    ],
  };
});

// 注册工具调用处理器
server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === 'get_hot_news') {
    const count = args.count || 10;
    const keyword = args.keyword || '';
    
    try {
      // 调用搜狐热闻 API
      const newsData = await fetchSohuNews(count, keyword);
      
      return {
        content: [
          {
            type: 'text',
            text: JSON.stringify(newsData, null, 2),
          },
        ],
      };
    } catch (error) {
      return {
        content: [
          {
            type: 'text',
            text: `Error fetching news: ${error.message}`,
          },
        ],
        isError: true,
      };
    }
  }
  
  throw new Error(`Unknown tool: ${name}`);
});

// 辅助函数：获取搜狐新闻
function fetchSohuNews(count, keyword) {
  return new Promise((resolve, reject) => {
    const url = `https://uis.mp.sohu.com/blog/outer/temp/feeds/ark?count=${count}${keyword ? `&q=${encodeURIComponent(keyword)}` : ''}`;
    
    https.get(url, (res) => {
      let data = '';
      
      res.on('data', (chunk) => {
        data += chunk;
      });
      
      res.on('end', () => {
        try {
          const json = JSON.parse(data);
          resolve(json);
        } catch (e) {
          reject(e);
        }
      });
    }).on('error', reject);
  });
}

// 启动服务器
const transport = new StdioServerTransport();
server.connect(transport);

// 输出到 stderr（stdout 用于 MCP 通信）
console.error('MCP News Server Started');
```

### 步骤 2：配置插件元数据

```yaml
# backend/conf/plugin/pluginproduct/plugin_meta.yaml

- plugin_id: 100
  product_id: 7500000000000000100
  deprecated: false
  version: v1.0.0
  openapi_doc_file: mcp_news_plugin.yaml
  plugin_type: 1
  manifest:
    schema_version: v1
    name_for_model: mcp_hot_news
    name_for_human: MCP 热门新闻
    description_for_model: 通过 MCP 协议获取搜狐热门新闻
    description_for_human: 基于 MCP 协议的新闻获取插件
    auth:
      type: none
    logo_url: official_plugin_icon/plugin_mcp_news.png
    api:
      type: coze-studio-mcp
      extensions:
        mcp_config:
          transport_type: stdio
          stdio_config:
            command: ["node", "/absolute/path/to/mcp-news-server.js"]
            env:
              NODE_ENV: production
            working_dir: /absolute/path/to/working/dir
    common_params:
      body: []
      header: []
      path: []
      query: []
  tools:
    - tool_id: 100001
      deprecated: false
      method: get
      sub_url: /get_hot_news
```

### 步骤 3：定义工具 Schema

```yaml
# backend/conf/plugin/pluginproduct/mcp_news_plugin.yaml

info:
  description: 通过 MCP 协议获取热门新闻
  title: MCP 热门新闻
  version: v1
openapi: 3.0.1

servers:
  - url: mcp://localhost  # MCP 不需要真实 URL

paths:
  /get_hot_news:
    get:
      operationId: get_hot_news
      summary: 获取热门新闻列表
      
      parameters:
        - description: 获取新闻数量
          in: query
          name: count
          required: true
          schema:
            type: integer
            default: 10
        
        - description: 搜索关键词
          in: query
          name: keyword
          required: false
          schema:
            type: string
      
      responses:
        "200":
          content:
            application/json:
              schema:
                properties:
                  code:
                    type: number
                  data:
                    type: object
                  message:
                    type: string
                  success:
                    type: boolean
                type: object
          description: 成功响应
```

### 步骤 4：测试插件

#### 4.1 在 Workflow 中测试

1. 创建新的 Workflow
2. 添加 LLM 节点
3. 添加插件节点，选择"MCP 热门新闻"
4. 配置参数：`count=5`
5. 运行测试

#### 4.2 使用 Go 测试代码

```go
package main

import (
    "context"
    "fmt"
    "testing"
    
    "github.com/coze-dev/coze-studio/backend/pkg/mcp"
)

func TestMCPNewsPlugin(t *testing.T) {
    // 创建 MCP 客户端配置
    config := &mcp.Config{
        TransportType: mcp.TransportTypeStdio,
        StdioConfig: &mcp.StdioConfig{
            Command: []string{"node", "/path/to/mcp-news-server.js"},
            Env: map[string]string{
                "NODE_ENV": "production",
            },
        },
    }
    
    // 创建客户端
    client, err := mcp.NewClient(config)
    if err != nil {
        t.Fatalf("create client failed: %v", err)
    }
    defer client.Close()
    
    // 初始化
    ctx := context.Background()
    if err := client.Initialize(ctx); err != nil {
        t.Fatalf("initialize failed: %v", err)
    }
    
    // 列出工具
    tools, err := client.ListTools(ctx)
    if err != nil {
        t.Fatalf("list tools failed: %v", err)
    }
    
    fmt.Printf("Available tools: %+v\n", tools)
    
    // 调用工具
    result, err := client.CallTool(ctx, "get_hot_news", map[string]any{
        "count":   5,
        "keyword": "科技",
    })
    if err != nil {
        t.Fatalf("call tool failed: %v", err)
    }
    
    fmt.Printf("Result: %s\n", result)
}
```

### 步骤 5：部署和监控

```bash
# 1. 确保 MCP 服务器脚本可执行
chmod +x /path/to/mcp-news-server.js

# 2. 测试 MCP 服务器
node /path/to/mcp-news-server.js

# 3. 重启 Coze Studio 后端
cd /path/to/coze-studio/backend
make restart

# 4. 查看日志
tail -f logs/app.log | grep -i mcp
```

---

## MCP vs OpenAPI 插件对比

| 特性 | OpenAPI 插件 (搜狐热闻) | MCP 插件 |
|------|------------------------|----------|
| **API 类型** | `"openapi"` | `"coze-studio-mcp"` |
| **通信协议** | HTTP/HTTPS | stdio / SSE |
| **服务器** | 外部 REST API | 本地子进程或远程 MCP 服务器 |
| **认证方式** | OAuth / API Key / None | 在 MCP 服务器内部处理 |
| **工具发现** | OpenAPI Schema | MCP `tools/list` |
| **工具调用** | HTTP Request | MCP `tools/call` |
| **响应格式** | JSON (自定义) | MCP 标准格式 |
| **开发难度** | 低（仅需配置） | 中（需实现 MCP 服务器） |
| **灵活性** | 低 | 高（可自定义逻辑） |
| **资源访问** | ❌ | ✅ (`resources/read`) |
| **流式输出** | ❌ | ✅ (MCP 支持) |
| **适用场景** | 标准 REST API 集成 | 复杂逻辑、本地资源、需要状态管理 |

### 何时选择 OpenAPI 插件？

- ✅ 已有标准 REST API
- ✅ 无需复杂业务逻辑
- ✅ 快速集成第三方服务
- ✅ 团队熟悉 OpenAPI 规范

### 何时选择 MCP 插件？

- ✅ 需要访问本地资源（文件、数据库）
- ✅ 需要复杂的业务逻辑
- ✅ 需要状态管理和会话
- ✅ 需要流式输出
- ✅ 想要更好的工具组合能力

---

## 故障排查与最佳实践

### 常见问题

#### 1. MCP 服务器无法启动

**症状**：
```
Error: create stdio client failed: exec: "node": executable file not found in $PATH
```

**解决方案**：
```bash
# 方案 1：使用绝对路径
which node
# 输出：/usr/local/bin/node

# 在配置中使用绝对路径
command: ["/usr/local/bin/node", "/path/to/server.js"]

# 方案 2：修改脚本 shebang
#!/usr/bin/env node
```

#### 2. MCP 工具调用失败

**症状**：
```
Error: call mcp tool failed: tool not found
```

**解决方案**：
```go
// 1. 检查工具名称是否匹配
tools, _ := client.ListTools(ctx)
fmt.Printf("Available tools: %+v\n", tools)

// 2. 验证参数格式
arguments := map[string]any{
    "count": 10,  // 确保类型匹配
}

// 3. 查看 MCP 服务器日志
console.error('[MCP Server] Received tool call:', name, args);
```

#### 3. 客户端缓存问题

**症状**：配置更新后仍使用旧配置

**解决方案**：
```go
// 在 mcpCallImpl 中添加清除缓存方法
func (m *mcpCallImpl) ClearCache() {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    for _, client := range m.clients {
        client.Close()
    }
    m.clients = make(map[string]*mcp.Client)
}

// 或者修改配置 hash 算法，包含时间戳
func (m *mcpCallImpl) hashConfig(config *mcp.Config) string {
    config.Timestamp = time.Now().Unix()  // 强制刷新
    data, _ := json.Marshal(config)
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}
```

### 最佳实践

#### 1. MCP 服务器开发

```javascript
// ✅ 好的做法：详细的错误处理
server.setRequestHandler('tools/call', async (request) => {
  const { name, arguments: args } = request.params;
  
  try {
    // 参数验证
    if (!args.count || typeof args.count !== 'number') {
      return {
        content: [{ type: 'text', text: 'Invalid count parameter' }],
        isError: true,
      };
    }
    
    // 业务逻辑
    const result = await fetchData(args);
    
    return {
      content: [{ type: 'text', text: JSON.stringify(result) }],
    };
  } catch (error) {
    console.error('[MCP Server] Error:', error);
    return {
      content: [{ type: 'text', text: `Error: ${error.message}` }],
      isError: true,
    };
  }
});

// ❌ 不好的做法：直接抛出异常
server.setRequestHandler('tools/call', async (request) => {
  const result = await fetchData(request.params.arguments);  // 可能抛出异常
  return { content: [{ type: 'text', text: result }] };
});
```

#### 2. 配置管理

```yaml
# ✅ 好的做法：使用环境变量
api:
  type: coze-studio-mcp
  extensions:
    mcp_config:
      transport_type: stdio
      stdio_config:
        command: ["node", "${MCP_SERVER_PATH}/news-server.js"]
        env:
          NODE_ENV: "${NODE_ENV}"
          API_KEY: "${NEWS_API_KEY}"
        working_dir: "${MCP_SERVER_PATH}"

# ❌ 不好的做法：硬编码路径
command: ["node", "/home/user/projects/server.js"]
```

#### 3. 错误处理

```go
// ✅ 好的做法：详细的错误信息
func (m *mcpCallImpl) Do(ctx context.Context, args *InvocationArgs) (string, string, error) {
    client, err := m.getOrCreateClient(ctx, config)
    if err != nil {
        logs.CtxErrorf(ctx, "[MCP] Failed to create client: config=%+v, err=%v", config, err)
        return "", "", errorx.Wrapf(err, "create mcp client failed")
    }
    
    result, err := client.CallTool(ctx, toolName, arguments)
    if err != nil {
        logs.CtxErrorf(ctx, "[MCP] Tool call failed: tool=%s, args=%+v, err=%v", toolName, arguments, err)
        return "", "", errorx.Wrapf(err, "call mcp tool '%s' failed", toolName)
    }
    
    return requestJSON, result, nil
}

// ❌ 不好的做法：丢失上下文信息
func (m *mcpCallImpl) Do(ctx context.Context, args *InvocationArgs) (string, string, error) {
    client, err := m.getOrCreateClient(ctx, config)
    if err != nil {
        return "", "", err  // 无法追踪问题
    }
    // ...
}
```

#### 4. 性能优化

```go
// ✅ 好的做法：连接复用和超时控制
func (m *mcpCallImpl) getOrCreateClient(ctx context.Context, config *mcp.Config) (*mcp.Client, error) {
    configHash := m.hashConfig(config)
    
    // 从缓存获取
    m.mu.RLock()
    if client, ok := m.clients[configHash]; ok {
        // 健康检查
        if err := m.healthCheck(ctx, client); err == nil {
            m.mu.RUnlock()
            return client, nil
        }
        // 不健康，需要重新创建
        logs.CtxWarnf(ctx, "[MCP] Client unhealthy, recreating")
    }
    m.mu.RUnlock()
    
    // 创建新客户端（带超时）
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    client, err := mcp.NewClient(config)
    if err != nil {
        return nil, err
    }
    
    // 缓存
    m.mu.Lock()
    m.clients[configHash] = client
    m.mu.Unlock()
    
    return client, nil
}

func (m *mcpCallImpl) healthCheck(ctx context.Context, client *mcp.Client) error {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    
    _, err := client.ListTools(ctx)
    return err
}
```

#### 5. 监控和日志

```go
// ✅ 好的做法：结构化日志
logs.CtxInfof(ctx, "[MCP] Tool call started: tool=%s, args=%+v", toolName, arguments)

start := time.Now()
result, err := client.CallTool(ctx, toolName, arguments)
duration := time.Since(start)

if err != nil {
    logs.CtxErrorf(ctx, "[MCP] Tool call failed: tool=%s, duration=%v, err=%v", toolName, duration, err)
} else {
    logs.CtxInfof(ctx, "[MCP] Tool call succeeded: tool=%s, duration=%v, result_len=%d", toolName, duration, len(result))
}

// 可选：上报到监控系统
metrics.RecordMCPToolCall(toolName, duration, err == nil)
```

### 调试技巧

#### 1. 启用 MCP 服务器调试日志

```javascript
// mcp-server.js
const DEBUG = process.env.DEBUG === 'true';

function debugLog(...args) {
  if (DEBUG) {
    console.error('[DEBUG]', ...args);
  }
}

server.setRequestHandler('tools/call', async (request) => {
  debugLog('Received tool call:', request);
  // ...
});

// 运行时启用
// DEBUG=true node mcp-server.js
```

#### 2. 使用 MCP Inspector 工具

```bash
# 安装 MCP Inspector
npm install -g @modelcontextprotocol/inspector

# 启动 Inspector
mcp-inspector node /path/to/mcp-server.js

# 浏览器打开 http://localhost:6274
```

#### 3. 单独测试 MCP 服务器

```bash
# 使用 echo 测试
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | node mcp-server.js

# 预期输出：
# {"jsonrpc":"2.0","id":1,"result":{"tools":[...]}}
```

---

## 总结

### 开发流程对比

#### OpenAPI 插件（如搜狐热闻）

1. 定义 OpenAPI 文档 (YAML)
2. 配置插件元数据
3. 系统自动处理 HTTP 调用
4. ✅ 简单快速

#### MCP 插件

1. 实现 MCP 服务器（Node.js/Python）
2. 定义工具 Schema (YAML)
3. 配置插件元数据（包含 MCP 配置）
4. 系统通过 mcp-go 库调用
5. ⚙️ 灵活强大

### 核心差异

| 维度 | OpenAPI | MCP |
|------|---------|-----|
| **实现复杂度** | ⭐ | ⭐⭐⭐ |
| **灵活性** | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **资源访问** | ❌ | ✅ |
| **本地能力** | ❌ | ✅ |
| **学习成本** | 低 | 中 |
| **维护成本** | 低 | 中 |

### 推荐选择

- **快速集成第三方 API** → 选择 OpenAPI 插件
- **需要本地资源访问或复杂逻辑** → 选择 MCP 插件
- **混合使用** → 根据具体工具特点选择

---

## 附录

### A. 完整项目结构

```
coze-studio/
├── backend/
│   ├── pkg/
│   │   └── mcp/
│   │       └── client.go                    # MCP 客户端封装
│   ├── domain/
│   │   └── plugin/
│   │       └── service/
│   │           ├── exec_tool.go             # 插件执行入口
│   │           └── tool/
│   │               ├── invocation_http.go   # HTTP 调用实现
│   │               └── invocation_mcp.go    # MCP 调用实现
│   ├── crossdomain/
│   │   └── plugin/
│   │       ├── consts/
│   │       │   └── consts.go               # 插件类型定义
│   │       └── model/
│   │           └── plugin_manifest.go      # Manifest 结构
│   └── conf/
│       └── plugin/
│           └── pluginproduct/
│               ├── plugin_meta.yaml         # 插件元数据
│               ├── sohu_hot_news.yaml       # OpenAPI 示例
│               └── mcp_news_plugin.yaml     # MCP 示例
├── mcp-servers/                             # MCP 服务器
│   └── news-server/
│       ├── package.json
│       ├── mcp-news-server.js               # 新闻服务器
│       └── README.md
└── doc/
    ├── MCP_Integration_Guide.md             # MCP 集成指南
    ├── MCP_Quick_Start.md                   # 快速开始
    └── MCP_Plugin_Development_Guide.md      # 本文档
```

### B. 参考资源

- **MCP 协议文档**：https://modelcontextprotocol.io/
- **mcp-go 库**：https://github.com/mark3labs/mcp-go
- **MCP SDK (Node.js)**：https://github.com/modelcontextprotocol/sdk
- **OpenAPI 规范**：https://swagger.io/specification/
- **Coze Studio 文档**：参见项目 `docs/` 目录

---

**文档版本**：v1.0  
**创建日期**：2025-11-09  
**作者**：开发团队  
**维护者**：开发团队

**变更记录**：
- 2025-11-09：初始版本，基于搜狐热闻插件分析和 MCP 集成实现

