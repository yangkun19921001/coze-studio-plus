# "exceeds max steps" 错误分析与解决方案

## 问题描述

工作流执行时出现错误：
```json
{
  "$error": "Workflow execution failure: exceeds max steps"
}
```

## 问题根源

### 1. 错误来源
错误来自 `eino-compose` 库（`eino-com/eino/compose/graph_run.go:241-242`）：
```go
if !r.dag && step >= maxSteps {
    return nil, newGraphRunError(ErrExceedMaxSteps)
}
```

### 2. Max Steps 默认值计算
在 `eino-com/eino/compose/graph.go:842-843` 中，默认值设置为：
```go
} else if !r.dag && r.options.maxRunSteps == 0 {
    r.options.maxRunSteps = len(r.chanSubscribeTo) + 10
}
```

**默认 max steps = 节点数量 + 10**

### 3. 当前实现问题
在 `coze-studio/backend/domain/workflow/internal/compose/workflow.go:143-151` 中，编译工作流时**没有设置 `WithMaxRunSteps`**：

```go
var compileOpts []compose.GraphCompileOption
if wf.requireCheckpoint {
    compileOpts = append(compileOpts, compose.WithCheckPointStore(workflow2.GetRepository()))
}
if wfOpts.idAsName {
    compileOpts = append(compileOpts, compose.WithGraphName(strconv.FormatInt(wfOpts.wfID, 10)))
}
// 编译成可执行的 Runner
r, err := wf.Compile(ctx, compileOpts...)
```

这意味着所有工作流都使用默认值 `节点数 + 10`，这对于包含循环、重试或复杂分支的工作流来说可能不够。

## 为什么会超过限制？

1. **循环节点（Loop）**：每次循环迭代都会增加步骤数
2. **条件分支**：某些分支路径可能执行更多步骤
3. **重试机制**：节点失败重试会增加步骤数
4. **复杂工作流**：节点数量多，但实际执行步骤远超 `节点数 + 10`
5. **Agent/ReAct 模式**：Agent 节点可能进行多轮工具调用，导致步骤数增加

## 解决方案

### 方案 1：在编译时设置更大的 Max Steps（推荐）

修改 `coze-studio/backend/domain/workflow/internal/compose/workflow.go`，在编译时添加 `WithMaxRunSteps`：

```go
var compileOpts []compose.GraphCompileOption
if wf.requireCheckpoint {
    compileOpts = append(compileOpts, compose.WithCheckPointStore(workflow2.GetRepository()))
}
if wfOpts.idAsName {
    compileOpts = append(compileOpts, compose.WithGraphName(strconv.FormatInt(wfOpts.wfID, 10)))
}

// 设置更大的 max steps，避免复杂工作流超过限制
// 默认值：节点数 * 10（为循环和重试预留足够空间）
nodeCount := sc.NodeCount()
maxSteps := int(nodeCount) * 10
if maxSteps < 100 {
    maxSteps = 100 // 最小保证 100 步
}
compileOpts = append(compileOpts, compose.WithMaxRunSteps(maxSteps))

// 编译成可执行的 Runner
r, err := wf.Compile(ctx, compileOpts...)
```

### 方案 2：通过配置项设置（更灵活）

1. 在 `execute/consts.go` 中添加配置：
```go
const (
    // ... 其他常量
    defaultMaxRunStepsMultiplier = 10 // 默认倍数：节点数 * 10
    minMaxRunSteps              = 100 // 最小 max steps
)
```

2. 在 `workflow.go` 中使用配置：
```go
nodeCount := sc.NodeCount()
maxSteps := int(nodeCount) * defaultMaxRunStepsMultiplier
if maxSteps < minMaxRunSteps {
    maxSteps = minMaxRunSteps
}
compileOpts = append(compileOpts, compose.WithMaxRunSteps(maxSteps))
```

### 方案 3：针对特定工作流类型设置

对于包含循环或 Agent 节点的工作流，设置更大的值：

```go
var maxSteps int
nodeCount := int(sc.NodeCount())

// 检查是否包含循环节点或 Agent 节点
hasLoop := sc.HasNodeType(entity.NodeTypeLoop)
hasAgent := sc.HasNodeType(entity.NodeTypeAgent)

if hasLoop || hasAgent {
    // 循环和 Agent 节点需要更多步骤
    maxSteps = nodeCount * 50
} else {
    maxSteps = nodeCount * 10
}

if maxSteps < 100 {
    maxSteps = 100
}

compileOpts = append(compileOpts, compose.WithMaxRunSteps(maxSteps))
```

## 实施建议

1. **立即修复**：采用方案 1，设置 `节点数 * 10` 作为默认值
2. **长期优化**：
   - 添加配置项，允许管理员调整倍数
   - 监控工作流执行步骤数，识别异常情况
   - 对于特别复杂的工作流，考虑优化结构

## 注意事项

1. **不要设置过大**：过大的 max steps 可能导致无限循环无法及时终止
2. **监控执行步骤**：建议添加日志记录实际执行的步骤数
3. **DAG 图不受影响**：DAG（有向无环图）模式不受 max steps 限制

## 相关代码位置

- 错误定义：`eino-com/eino/compose/error.go:27`
- 错误检查：`eino-com/eino/compose/graph_run.go:241-242`
- 默认值设置：`eino-com/eino/compose/graph.go:842-843`
- 编译选项：`coze-studio/backend/domain/workflow/internal/compose/workflow.go:143-151`

