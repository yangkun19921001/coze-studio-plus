# LLM 输出重复保存问题分析

## 问题描述
当前 LLM 节点的流式输出会被保存**两次**到数据库：
1. React Agent 的 callback 中通过 `realtimeWriter` 发送消息后保存一次
2. `tConvert` 输出通过 `NodeStreamingOutput` 事件后保存一次

## 完整数据流分析

### 第一条路径：realtimeWriter 实时流 (React Agent Callback)

```
llm.go L873-926: ChatModel Callback OnEndWithStreamOutput
  └─> 读取 React Agent 的输出流
  └─> frame.Message.Content 有内容时:
      └─> L918: llmRef.realtimeWriter.Send(msg, nil)  // 发送到 Workflow 的 StreamWriter
          └─> msg = &entity.Message{
                  DataMessage: {
                      Type: entity.Answer,
                      Content: frame.Message.Content,
                      Role: schema.Assistant,
                      NodeType: entity.NodeTypeLLM,
                      ExecuteID: exeCtx.RootExecuteID,
                      NodeID: string(exeCtx.NodeKey),
                      NodeTitle: exeCtx.NodeName
                  }
              }

chatflow.go L715: GetWorkflowDomainSVC().StreamExecute(ctx, exeCfg, parameters)
  └─> 返回 *schema.StreamReader[*entity.Message]
  └─> L720: schema.StreamReaderWithConvert(sr, w.convertToChatFlowRunResponseList(...))
      └─> convertToChatFlowRunResponseList 逐条读取消息
          └─> L1032-1103: 处理消息
              ├─> L1077: intermediateMessage.Content += msg.Content  // 积累内容
              ├─> L1091-1097: 如果 !msg.Last -> 只发送 message_delta，不保存
              └─> L1100-1103: 如果 msg.Last -> 调用 crossmessage.DefaultSVC().Create()
                              🔴 **第一次保存到数据库**
```

**关键点：**
- `realtimeWriter` 发送的消息会实时通过 SSE 发送到前端（`message_delta` 事件）
- 当消息完成（`msg.Last = true`）时，会在 `chatflow.go:1100` 保存到数据库

### 第二条路径：tConvert 输出流 (Node Callback)

```
llm.go L790-837: tConvert function
  └─> 作为 output_convert 节点的 Lambda 函数
  └─> L802: msg, err := s.Recv()  // 读取 React Agent 的输出
  └─> L831: sw.Send(map[string]any{outputKey: msg.Content}, nil)
      └─> 发送到新创建的 Pipe 的 StreamWriter
      └─> L836: return sr (返回 Pipe 的 StreamReader)

llm.go L847-848: AddEdge
  └─> llmNodeKey -> outputConvertNodeKey -> compose.END

callback.go L1177-1199: NodeHandler.OnEndWithStreamOutput
  └─> L1186-1194: incrementalEndProcessor(c, output)
      └─> output 就是 tConvert 返回的 StreamReader
      └─> L1113: chunk, err := output.Recv()  // 读取 tConvert 输出的 chunk
      └─> L1156: buildStreamDeltaEvent(c, chunk, accumulated)
          └─> 构建 Event{Type: NodeStreamingOutput, ...}
      └─> L1163: n.ch <- firstEvent  // 发送到 event channel

event_handle.go L559-588: handleEvent(NodeStreamingOutput)
  ├─> L560-573: 如果有 StreamWriter，发送消息到 SSE 前端
  └─> L586-588: repo.UpdateNodeExecutionStreaming(ctx, nodeExec)
                🔴 **第二次保存到数据库**
```

**关键点：**
- `tConvert` 读取 React Agent 的输出，转换格式后发送到新的 Pipe
- Workflow 框架的 `NodeHandler` 读取 `tConvert` 的输出
- 生成 `NodeStreamingOutput` 事件，在 `event_handle.go:586` 保存到数据库

## 数据源对比

| 特征 | realtimeWriter 路径 | tConvert 路径 |
|------|---------------------|---------------|
| 数据源 | React Agent 原始输出 | React Agent 原始输出（同一个 StreamReader） |
| 内容 | `frame.Message.Content` | `msg.Content`（相同） |
| 保存时机 | 消息完成时（msg.Last = true） | 每个 chunk 都更新 NodeExecution |
| 保存位置 | `conversation.message` 表（chatflow.go:1100） | `node_execution` 表（event_handle.go:586） |
| 保存方法 | `crossmessage.DefaultSVC().Create()` | `repo.UpdateNodeExecutionStreaming()` |

## 为什么会读取同一个 StreamReader 两次？

关键在于 React Agent 的输出流会被**多个 callback** 同时监听：

```go
// llm.go L873: ChatModel Callback (优先触发)
OnEndWithStreamOutput: func(..., output *schema.StreamReader[*model.CallbackOutput]) {
    // 这个 output 是 React Agent 的原始输出流
    // 在这里读取并通过 realtimeWriter 发送
}

// llm.go L1478: Graph Stream 执行
out, err = l.r.Stream(ctx, in, composeOpts...)
// 这个 out 也是同一个流的引用（或克隆）
// tConvert 会读取它并转换格式
```

**实际上，这两个 callback 读取的是同一个数据源的不同副本或通过不同的 callback 机制传递的副本。**

## 解决方案

### 方案 1：只在 realtimeWriter 保存（推荐）✅

**优点：**
- 保留实时性，消息立即发送到前端
- 只在消息完成时保存一次
- 语义清晰：realtimeWriter 负责实时输出+最终保存

**实现：**
```go
// event_handle.go L559-588
case NodeStreamingOutput:
    // 只发送到 SSE 前端，不保存到数据库
    if sw != nil && len(event.Answer) > 0 {
        sw.Send(&entity.Message{...}, nil)
    }
    
    // 🔴 移除或注释掉这段保存逻辑
    // if err = repo.UpdateNodeExecutionStreaming(ctx, nodeExec); err != nil {
    //     return noTerminate, fmt.Errorf("failed to save node execution: %v", err)
    // }
```

### 方案 2：只在 NodeStreamingOutput 保存

**优点：**
- 统一的 Workflow 事件处理机制
- 所有节点的输出都通过 NodeStreamingOutput 保存

**缺点：**
- 需要修改 realtimeWriter 的逻辑，可能影响其他节点

**实现：**
```go
// chatflow.go L1100-1103
// 🔴 移除 realtimeWriter 路径的保存逻辑
// if msg.Last {
//     _, err = crossmessage.DefaultSVC().Create(ctx, intermediateMessage)
//     if err != nil {
//         return nil, err
//     }
// }

// 只保留 SSE 发送，不保存
if msg.Last {
    completeData, _ := sonic.MarshalString(&vo.MessageDetail{...})
    return []*workflow.ChatFlowRunResponse{{
        Event: string(vo.ChatFlowMessageCompleted),
        Data:  completeData,
    }}, nil
}
```

### 方案 3：条件保存（最灵活）

只在其中一条路径保存，通过标志位避免重复：

```go
// llm.go L918: realtimeWriter.Send 时添加标志
msg := &entity.Message{
    DataMessage: dataMsg,
    SavedByRealtimeWriter: true,  // 添加标志
}

// event_handle.go L559: 检查标志
case NodeStreamingOutput:
    if sw != nil && len(event.Answer) > 0 {
        sw.Send(&entity.Message{...}, nil)
    }
    
    // 只有 realtimeWriter 未保存时才保存
    if !event.SavedByRealtimeWriter {
        if err = repo.UpdateNodeExecutionStreaming(ctx, nodeExec); err != nil {
            return noTerminate, fmt.Errorf("failed to save node execution: %v", err)
        }
    }
```

## 推荐实现

**推荐方案 1**，理由：
1. **语义最清晰**：realtimeWriter 就是为了实时输出设计的，让它负责最终保存符合其定位
2. **最小改动**：只需要在 `event_handle.go` 中移除一处保存逻辑
3. **保持一致性**：不影响其他节点的 NodeStreamingOutput 逻辑（如果其他节点不使用 realtimeWriter）

## 具体修改建议

**文件：** `coze-studio/backend/domain/workflow/internal/execute/event_handle.go`

```go
// L559-588
case NodeStreamingOutput:
    // 🎯 只发送到 SSE，不保存到数据库（避免重复保存）
    // realtimeWriter 路径已经在 chatflow.go:1100 保存过了
    if sw != nil && len(event.Answer) > 0 {
        sw.Send(&entity.Message{
            DataMessage: &entity.DataMessage{
                ExecuteID: event.RootExecuteID,
                Role:      schema.Assistant,
                Type:      entity.Answer,
                Content:   event.Answer,
                NodeID:    string(event.NodeKey),
                NodeType:  event.NodeType,
                NodeTitle: event.NodeName,
                Last:      event.StreamEnd,
            },
        }, nil)
    }

    // 🔴 注释掉数据库保存逻辑，避免与 realtimeWriter 路径重复保存
    /*
    nodeExec := &entity.NodeExecution{
        ID:    event.NodeExecuteID,
        Extra: event.extra,
    }

    if event.outputStr != nil {
        nodeExec.Output = event.outputStr
    } else {
        nodeExec.Output = ptr.Of(mustMarshalToString(event.Output))
    }

    if err = repo.UpdateNodeExecutionStreaming(ctx, nodeExec); err != nil {
        return noTerminate, fmt.Errorf("failed to save node execution: %v", err)
    }
    */
```

## 验证方法

1. **修改前：** 查看数据库，同一条消息会有两条记录
   - `conversation.message` 表中一条（来自 realtimeWriter）
   - `node_execution` 表中一条（来自 NodeStreamingOutput）

2. **修改后：** 
   - `conversation.message` 表中有一条（来自 realtimeWriter）
   - `node_execution` 表中无记录（或只有元数据，没有重复的 content）

3. **功能测试：**
   - 前端仍能正常接收 SSE 消息流
   - 数据库中消息只保存一次
   - 历史记录查询正常

## 相关文件

- `coze-studio/backend/domain/workflow/internal/nodes/llm/llm.go`（L790-837, L873-926）
- `coze-studio/backend/application/workflow/chatflow.go`（L1031-1127）
- `coze-studio/backend/domain/workflow/internal/execute/event_handle.go`（L559-588）
- `coze-studio/backend/domain/workflow/internal/execute/callback.go`（L1177-1199）

