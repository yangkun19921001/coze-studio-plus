# 插件节点 vs Function Calling 对比

## 🎯 核心问题

你的流程是：**开始 → 大模型 → top_news(插件) → 结束**

这是一个**显式调用**的流程，**不是** React Agent！

---

## 📊 两种完全不同的架构

### 架构 1：插件节点（你的当前流程）❌ 不是 React Agent

```
┌─────────┐      ┌─────────┐      ┌──────────┐      ┌─────────┐
│  开始   │ ───→ │ 大模型  │ ───→ │ top_news │ ───→ │  结束   │
│  Start  │      │   LLM   │      │  (插件)  │      │   End   │
└─────────┘      └─────────┘      └──────────┘      └─────────┘
                     │                   ↑
                     └──── output ───────┘
                    (强制传递参数)
```

**特点**：
- ✅ 插件**总是执行**（无论是否需要）
- ✅ 这是**固定的线性流程**
- ❌ LLM **无法决定**是否调用插件
- ❌ **不使用** React Agent
- ❌ **不是** Function Calling

**执行逻辑**：
```python
# 伪代码
llm_output = LLM.invoke(user_input)           # 大模型生成内容
plugin_output = top_news.invoke(llm_output)   # 插件被强制调用
return plugin_output
```

---

### 架构 2：Function Calling（React Agent）✅ 智能调用

```
┌─────────┐      ┌───────────────────────────────────┐      ┌─────────┐
│  开始   │ ───→ │       大模型 (带工具能力)          │ ───→ │  结束   │
│  Start  │      │                                   │      │   End   │
└─────────┘      │  ┌─────────────────────────────┐ │      └─────────┘
                 │  │   React Agent Loop          │ │
                 │  │                             │ │
                 │  │  1. LLM 思考               │ │
                 │  │  2. 决定是否调用工具？      │ │
                 │  │     ├─ YES → 调用 top_news │ │
                 │  │     │        ↓              │ │
                 │  │     │    获取结果           │ │
                 │  │     │        ↓              │ │
                 │  │     │    回到步骤 1         │ │
                 │  │     │                       │ │
                 │  │     └─ NO → 直接输出答案    │ │
                 │  └─────────────────────────────┘ │
                 └───────────────────────────────────┘
```

**特点**：
- ✅ LLM **自己决定**是否调用插件
- ✅ 插件**可能不执行**（如果 LLM 认为不需要）
- ✅ 使用 **React Agent**
- ✅ 这是 **Function Calling**
- ✅ 支持**多轮对话**和**推理**

**执行逻辑**：
```python
# 伪代码
def react_agent(user_input):
    while True:
        decision = LLM.invoke(user_input, tools=[top_news])
        
        if decision.need_tool:
            tool_result = top_news.invoke(decision.tool_params)
            user_input = f"Tool result: {tool_result}"  # 继续循环
        else:
            return decision.final_answer  # 结束循环
```

---

## 🔍 你的流程深度分析

### 当前流程结构

```yaml
Workflow:
  nodes:
    - id: start
      type: Start
      
    - id: llm_node
      type: LLM
      name: "大模型"
      inputs:
        - input: "来自 Start"
      outputs:
        - output
        - reasoning_content
      config:
        model: "Deepseek-V3-0324"
        # ❌ 没有 FCParam (Function Calling 参数)
        # ❌ 没有配置 tools
        
    - id: top_news
      type: Plugin      # ⚠️ 这是一个独立的插件节点！
      name: "top_news"
      inputs:
        - count: 10
        - q: "来自 大模型.output"  # 强制接收 LLM 的输出
      outputs:
        - code
        - data
        - message
        # ... 更多输出
      config:
        pluginID: "xxx"
        apiID: "xxx"
        
    - id: end
      type: End
      
  edges:
    - from: start → llm_node
    - from: llm_node → top_news   # ⚠️ 固定边，总是执行
    - from: top_news → end
```

### 执行流程（当前）

```
用户输入: "查询今天的新闻"
    ↓
┌──────────────────────────────────────┐
│ Step 1: 大模型节点执行               │
├──────────────────────────────────────┤
│ LLM.Invoke(input="查询今天的新闻")  │
│   ↓                                  │
│ output = "今日新闻摘要"              │
│ reasoning_content = "..."            │
└──────────────────────────────────────┘
    ↓ (强制传递)
┌──────────────────────────────────────┐
│ Step 2: top_news 插件节点执行        │
├──────────────────────────────────────┤
│ Plugin.Invoke(                       │
│   count=10,                          │
│   q="今日新闻摘要"  ← 来自 LLM 输出 │
│ )                                    │
│   ↓                                  │
│ 调用真实的新闻 API                   │
│   ↓                                  │
│ 返回新闻列表                         │
└──────────────────────────────────────┘
    ↓
最终输出
```

**问题分析**：
1. ❌ **LLM 节点和插件节点是分离的**
2. ❌ **插件总是被执行**（无论是否需要）
3. ❌ **不使用 React Agent**
4. ✅ **插件会执行**（如果流程正常）

---

## 💡 为什么你说"没有执行到"？

可能的原因：

### 原因 1：LLM 的 output 为空或格式不对

```javascript
// 如果大模型节点输出：
{
  output: "",  // ❌ 空字符串
  reasoning_content: "..."
}

// top_news 接收到：
{
  count: 10,
  q: ""  // ❌ 空的查询参数
}

// 插件可能会：
// - 报错
// - 返回空结果
// - 被跳过（取决于插件实现）
```

### 原因 2：插件执行报错

```javascript
// 可能的错误：
1. 插件 API 认证失败
2. 参数格式不正确
3. 网络超时
4. 插件版本不匹配
```

### 原因 3：调试模式下没有看到插件输出

在 Coze Studio 的调试界面，你需要：
1. 点击 `top_news` 节点
2. 查看右侧面板的 **"常规"** 选项卡
3. 检查是否有执行日志

---

## 🔧 如何改成 React Agent（LLM 自动调用插件）

### 步骤 1：删除插件节点

```
删除：大模型 → top_news → 结束
改为：大模型 → 结束
```

### 步骤 2：在大模型节点中配置 Function Calling

1. 点击"大模型"节点
2. 在配置面板中找到 **"函数调用 (Function Calling)"** 或 **"工具"**
3. 点击 **"添加插件"**
4. 选择 `top_news` 插件
5. 配置插件参数映射（可选）
6. 保存

### 步骤 3：修改 Prompt

在大模型的 **User Prompt** 中添加：

```
你是一个新闻助手，可以使用 top_news 工具查询最新新闻。

当用户问到新闻相关的问题时，请调用 top_news 工具获取最新信息。
```

### 最终流程

```
┌─────────┐      ┌─────────────────────────┐      ┌─────────┐
│  开始   │ ───→ │   大模型 (React Agent)   │ ───→ │  结束   │
│  Start  │      │                         │      │   End   │
└─────────┘      │  内部可调用 top_news     │      └─────────┘
                 │  (自动决策)              │
                 └─────────────────────────┘
```

### 执行效果对比

**用户输入 1**: "你好"

| 当前架构（插件节点） | React Agent 架构 |
|---------------------|-----------------|
| LLM: "你好！"<br>Plugin: 调用 top_news(q="你好！")<br>❌ 浪费资源 | LLM: "你好！有什么可以帮你的？"<br>✅ 不调用插件<br>✅ 智能决策 |

**用户输入 2**: "今天有什么新闻？"

| 当前架构（插件节点） | React Agent 架构 |
|---------------------|-----------------|
| LLM: "今日新闻查询"<br>Plugin: 调用 top_news(q="今日新闻查询")<br>✅ 获取新闻 | LLM: 思考 → 需要工具<br>Plugin: 调用 top_news(q="今天")<br>LLM: 整理结果<br>✅ 获取新闻 + 智能摘要 |

---

## 🐛 调试你的当前流程

### 方法 1：查看执行日志

1. 在 VS Code 中启动后端调试（已配置）
2. 在 Coze Studio Web 中运行工作流
3. 观察终端输出：

```bash
# 应该看到：
[INFO] executing node: llm_node (LLM)
[INFO] llm output: {"output": "...", "reasoning_content": "..."}
[INFO] executing node: top_news (Plugin)
[INFO] plugin input: {"count": 10, "q": "..."}
[INFO] calling plugin API: pluginID=xxx, toolID=xxx
[INFO] plugin result: {"code": 0, "data": [...]}
```

如果看不到 `executing node: top_news`，说明插件节点被跳过了。

### 方法 2：在插件节点打断点

```go
// backend/domain/workflow/internal/nodes/plugin/plugin.go
func (p *Plugin) Invoke(ctx context.Context, parameters map[string]any) (ret map[string]any, err error) {
    // 👈 在这里打断点
    logs.CtxInfof(ctx, "Plugin.Invoke called, parameters: %+v", parameters)
    
    // ... 插件执行逻辑
}
```

### 方法 3：检查边连接

在 Coze Studio Web 中：
1. 检查 `大模型` → `top_news` 之间的连线是否正确
2. 检查 `top_news` 节点的输入配置：
   ```
   q: 大模型.output  ← 确保这个连接正确
   ```

---

## 📊 架构对比总结

| 特性 | 插件节点（当前） | React Agent（推荐） |
|-----|----------------|-------------------|
| **架构** | 开始 → LLM → Plugin → 结束 | 开始 → LLM(带工具) → 结束 |
| **执行模式** | 固定流程，插件总是执行 | 智能决策，按需调用 |
| **LLM 控制** | ❌ 无法控制插件调用 | ✅ 完全控制 |
| **适用场景** | 固定的数据处理流程 | 智能对话、复杂决策 |
| **资源效率** | ❌ 低（总是执行） | ✅ 高（按需执行） |
| **调试复杂度** | ✅ 简单（线性流程） | ⚠️ 中等（需理解 Agent） |
| **使用 React Agent** | ❌ 否 | ✅ 是 |
| **是否是 FC** | ❌ 否 | ✅ 是 |

---

## 🎯 结论

### 你的当前流程

```
开始 → 大模型 → top_news(插件) → 结束
```

- ❌ **不是** React Agent
- ❌ **不是** Function Calling
- ✅ **是** 显式的插件节点调用
- ✅ 插件**应该会执行**（如果流程正常）

### 如果"没有执行到"

**可能原因**：
1. LLM 输出为空或格式错误
2. 插件参数配置错误
3. 插件 API 调用失败
4. 边连接配置错误

**调试建议**：
1. 打开 VS Code 调试，查看日志
2. 在 `plugin.go` 的 `Invoke` 方法打断点
3. 检查 LLM 节点的输出是否正常
4. 检查 top_news 节点的输入配置

### 如果想要 React Agent

**需要改造流程**：
1. 删除独立的 `top_news` 插件节点
2. 在"大模型"节点中配置 Function Calling
3. 添加 `top_news` 为可调用工具
4. 修改 Prompt 引导 LLM 使用工具

---

希望这个对比能帮你理解两种架构的区别！🎉

