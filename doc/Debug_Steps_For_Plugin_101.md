# 🔍 实时调试：在 Debug Terminal 中查找错误

## 在你的 Debug Terminal（Go Debug Terminal）中执行以下搜索：

### 1. 搜索 "101"
```
Ctrl+F (或 Cmd+F)
输入: 101
```

### 2. 搜索以下错误关键词：
```
"invalid version"
"invalid plugin id"
"duplicate plugin id"
"invalid plugin type"
"plugin manifest validates failed"
"load file"
"validates failed"
"pairat"
"remote_exec"
"not found in doc"
```

### 3. 特别注意这些错误：

#### 可能错误 1：Manifest 验证失败
```
plugin manifest validates failed
```

#### 可能错误 2：OpenAPI 文档加载失败
```
load file 'xxx/pairat_remote_exec.yaml', err=xxx
```

#### 可能错误 3：OpenAPI 文档验证失败
```
the openapi3 doc 'pairat_remote_exec.yaml' validates failed
```

#### 可能错误 4：工具路径不匹配
```
api '[POST]:/remote_exec' not found in doc
api '[POST]:/vmcheck' not found in doc
```

---

## 🎯 最可能的问题

根据之前的分析，最可能的问题是：

### 问题 1：工具没有成功加载，导致插件被删除

```go
// load_plugin.go:251-253
if len(pi.ToolIDs) == 0 {
    delete(pluginProducts, m.PluginID)  // ← 如果没有工具，删除插件！
}
```

这意味着即使插件配置正确，如果所有工具都验证失败，插件也会被删除。

---

## 🔧 手动验证步骤

### 步骤 1：验证文件同步
```bash
cd /Users/devyk/Data/code/AI/coze-studio

# 确认两个文件一致
diff backend/conf/plugin/pluginproduct/plugin_meta.yaml \
     backend/resources/conf/plugin/pluginproduct/plugin_meta.yaml
# 应该输出：无差异

diff backend/conf/plugin/pluginproduct/pairat_remote_exec.yaml \
     backend/resources/conf/plugin/pluginproduct/pairat_remote_exec.yaml
# 应该输出：无差异
```

### 步骤 2：检查配置文件内容
```bash
# 查看 plugin_id: 101 的完整配置
sed -n '909,946p' backend/resources/conf/plugin/pluginproduct/plugin_meta.yaml

# 查看 OpenAPI 文档的关键部分
head -20 backend/resources/conf/plugin/pluginproduct/pairat_remote_exec.yaml
```

### 步骤 3：尝试添加调试日志

在重启前，我们可以暂时修改插件 ID 来测试：

```bash
# 方案：暂时改为更大的 ID 避免冲突
# 比如改为 201，看看能否加载
```

---

## 💡 临时解决方案：使用已知可用的插件 ID

让我检查一下已经加载的插件使用了哪些 ID：

从你的日志中看到：
- entity_id: 5 (创客贴智能设计) - plugin_id 可能是 5
- entity_id: 6 (搜狐热闻) - plugin_id 可能是 6
- entity_id: 11 (板栗看板) - plugin_id 可能是 11
- entity_id: 12 (天眼查) - plugin_id 可能是 12

**建议：改用 plugin_id: 201**

```yaml
- plugin_id: 201  # 改为 201
  product_id: 7600000000000000101
  # ... 其他配置保持不变
  tools:
    - tool_id: 201001  # 改为 201001
      deprecated: false
      method: post
      sub_url: /remote_exec
    - tool_id: 201002  # 改为 201002
      deprecated: false
      method: post
      sub_url: /vmcheck
```

---

## 🚨 紧急检查：确认后端是否真的重启了

```bash
# 在一个新终端中检查
lsof -i :8888 | grep LISTEN

# 或者查看进程启动时间
ps aux | grep coze-studio
```

如果后端没有真正重启，旧的配置还在内存中！

---

## 📝 完整的重启流程

1. **停止后端**
   - 在 Debug Terminal 中按 `Ctrl+C`
   - 等待进程完全停止（看到 "Exit" 信息）

2. **确认进程已停止**
   ```bash
   lsof -i :8888
   # 应该没有输出
   ```

3. **重新启动**
   - 在 VSCode 中点击 Debug 面板的重启按钮
   - 或者运行 `make run`

4. **观察启动日志**
   - 看是否有关于 plugin 加载的日志
   - 特别注意错误信息

---

## 🔍 下一步调试

请在 Debug Terminal 中：

1. **搜索 "plugin" 关键词**
2. **看看启动时的日志**
3. **查找是否有 "101"、"pairat" 或 "remote_exec" 相关的错误**

然后把看到的错误信息告诉我，我可以针对性地解决！

---

**如果实在找不到错误，我们可以：**
1. 改用 plugin_id: 201 试试
2. 或者先参考搜狐热闻的配置，创建一个最简单的测试插件
3. 一步步添加 MCP 配置

你觉得呢？要不要先试试改为 plugin_id: 201？

