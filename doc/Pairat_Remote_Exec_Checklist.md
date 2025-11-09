# ✅ Pairat Remote Exec 插件注册检查清单

## 📋 重启后能否在前端搜索到？

**答案：理论上可以，但需要满足以下条件** ⚠️

---

## 🔍 关键检查点

### 1. ✅ 配置文件路径（已确认）

插件加载路径：`backend/resources/conf/plugin/pluginproduct/`

**当前配置**：
- ✅ `plugin_meta.yaml` - 已添加 plugin_id: 101
- ✅ `pairat_remote_exec.yaml` - OpenAPI Schema 已创建

**需要确认的路径**：
```bash
# 检查文件是否在正确位置
ls -la backend/conf/plugin/pluginproduct/pairat_remote_exec.yaml
ls -la backend/resources/conf/plugin/pluginproduct/pairat_remote_exec.yaml

# ⚠️ 注意：代码从 resources/conf/plugin/ 加载插件
# 你需要确保文件在 resources 目录下！
```

---

### 2. ⚠️ 文件位置问题（需要处理）

根据代码 `backend/domain/plugin/conf/config.go:38`：

```go
basePath := path.Join(cwd, "resources", "conf", "plugin")
```

**问题**：你的配置文件在 `backend/conf/` 而不是 `backend/resources/conf/`

**解决方案**：

```bash
# 方案 1：复制文件到正确位置
cp backend/conf/plugin/pluginproduct/pairat_remote_exec.yaml \
   backend/resources/conf/plugin/pluginproduct/

cp backend/conf/plugin/pluginproduct/plugin_meta.yaml \
   backend/resources/conf/plugin/pluginproduct/

# 方案 2：创建符号链接（推荐）
cd backend/resources/conf/plugin/pluginproduct
ln -s ../../../conf/plugin/pluginproduct/pairat_remote_exec.yaml .

# 方案 3：修改 plugin_meta.yaml（如果 resources 目录下也有）
# 在 backend/resources/conf/plugin/pluginproduct/plugin_meta.yaml 中添加配置
```

---

### 3. ✅ 配置验证（需要检查）

插件加载时会进行以下验证（`load_plugin.go:152-175`）：

#### 3.1 基础验证

```yaml
# ✅ plugin_id: 101 - 有效
# ✅ version: v1.0.0 - 有效 (符合 semver)
# ✅ plugin_type: 1 - 有效 (PLUGIN 类型)
# ✅ deprecated: false - 未弃用
```

#### 3.2 Manifest 验证

```yaml
# ✅ schema_version: v1
# ✅ name_for_model: pairat_remote_exec
# ✅ name_for_human: Pairat Remote Exec
# ✅ api.type: coze-studio-mcp
```

#### 3.3 OpenAPI 文档验证

```yaml
# ✅ pairat_remote_exec.yaml 存在
# ✅ servers[0].url: mcp://localhost
# ✅ paths 定义完整
```

---

### 4. ⚠️ 可能的错误情况

如果插件没有加载，检查以下日志：

```bash
# 查看后端启动日志
tail -f backend/logs/app.log | grep -i "plugin\|pairat\|remote_exec"

# 可能的错误信息：
# ❌ "read file 'xxx' failed" - 文件路径错误
# ❌ "plugin manifest validates failed" - Manifest 验证失败
# ❌ "load file 'xxx', err=xxx" - OpenAPI 文档加载失败
# ❌ "the openapi3 doc 'xxx' validates failed" - OpenAPI 文档验证失败
# ❌ "duplicate plugin id '101'" - 插件 ID 冲突
```

---

## 🚀 完整操作步骤

### 步骤 1：确保文件在正确位置

```bash
cd /Users/devyk/Data/code/AI/coze-studio

# 方案 A：复制文件（推荐）
cp backend/conf/plugin/pluginproduct/pairat_remote_exec.yaml \
   backend/resources/conf/plugin/pluginproduct/

# 如果 resources 目录下的 plugin_meta.yaml 不同步，也需要更新
# 检查是否已经有这个文件
diff backend/conf/plugin/pluginproduct/plugin_meta.yaml \
     backend/resources/conf/plugin/pluginproduct/plugin_meta.yaml

# 如果不同，复制或手动同步
cp backend/conf/plugin/pluginproduct/plugin_meta.yaml \
   backend/resources/conf/plugin/pluginproduct/
```

### 步骤 2：验证配置文件

```bash
# 验证 YAML 语法
cd backend/resources/conf/plugin/pluginproduct

# 检查 pairat_remote_exec.yaml
python3 -c "import yaml; print('✅ Valid YAML' if yaml.safe_load(open('pairat_remote_exec.yaml')) else '❌ Invalid')"

# 检查 plugin_meta.yaml
python3 -c "import yaml; data=yaml.safe_load(open('plugin_meta.yaml')); print(f'✅ Found {len(data)} plugins')"

# 检查是否包含 plugin_id: 101
grep -A 5 "plugin_id: 101" plugin_meta.yaml
```

### 步骤 3：重启后端

```bash
cd /Users/devyk/Data/code/AI/coze-studio/backend

# 停止后端
make stop

# 或者
pkill -f "coze-studio"

# 启动后端（前台运行，方便看日志）
make run

# 或者后台运行
make start
```

### 步骤 4：查看启动日志

```bash
# 实时查看日志
tail -f backend/logs/app.log | grep -i "plugin\|pairat"

# 查看插件加载情况
grep "plugin" backend/logs/app.log | tail -20

# 如果有错误，查看完整错误信息
grep "error\|failed" backend/logs/app.log | grep -i plugin
```

### 步骤 5：测试 API 接口

```bash
# 测试获取插件列表 API
curl http://localhost:8080/api/plugin/list \
  -H "Content-Type: application/json" \
  -d '{"page": 1, "size": 100}' | jq '.data.plugin_list[] | select(.id==101)'

# 或者使用 jq 美化输出
curl -s http://localhost:8080/api/plugin/list | jq '.data.plugin_list[] | {id, name: .name_for_human, type: .manifest.api.type}'
```

### 步骤 6：前端搜索测试

1. 打开浏览器，访问 Coze Studio
2. 进入插件市场或 Workflow 编辑器
3. 搜索 "Pairat" 或 "Remote Exec"
4. 应该能看到插件

---

## 🐛 故障排查

### 问题 1：前端搜不到插件

**可能原因**：

1. **文件路径错误**
   ```bash
   # 检查文件是否在 resources 目录
   ls backend/resources/conf/plugin/pluginproduct/pairat_remote_exec.yaml
   ```

2. **插件 ID 冲突**
   ```bash
   # 检查是否有重复的 plugin_id: 101
   grep -n "plugin_id: 101" backend/resources/conf/plugin/pluginproduct/plugin_meta.yaml
   ```

3. **YAML 语法错误**
   ```bash
   # 验证 YAML 格式
   yamllint backend/resources/conf/plugin/pluginproduct/pairat_remote_exec.yaml
   ```

4. **后端没有重启**
   ```bash
   # 确认进程
   ps aux | grep coze-studio
   
   # 如果还在运行旧进程，强制停止
   pkill -9 -f coze-studio
   ```

---

### 问题 2：插件加载失败

查看日志中的具体错误：

```bash
# 查看所有与 pairat 相关的日志
grep -i "pairat\|remote_exec\|plugin_id.*101" backend/logs/app.log

# 常见错误及解决方案：

# 错误 1: "read file failed"
# 解决：确保文件在 backend/resources/conf/plugin/pluginproduct/

# 错误 2: "manifest validates failed"
# 解决：检查 manifest 字段是否完整

# 错误 3: "openapi3 doc validates failed"
# 解决：检查 OpenAPI Schema 格式

# 错误 4: "duplicate plugin id"
# 解决：修改 plugin_id 为未使用的 ID
```

---

### 问题 3：MCP 连接失败

如果插件能搜到但执行失败：

```bash
# 1. 测试 MCP 服务器是否运行
curl -N -H "Accept: text/event-stream" http://10.1.16.4:8000/mcp/sse

# 2. 测试 tools/list
curl -X POST http://10.1.16.4:8000/mcp/sse \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'

# 3. 检查网络连接
ping 10.1.16.4
telnet 10.1.16.4 8000
```

---

## ✅ 验证成功的标志

### 1. 后端日志

启动时应该看到：

```
[INFO] Loading plugin products from: backend/resources/conf/plugin/pluginproduct
[INFO] Loaded plugin: pairat_remote_exec (ID: 101)
[INFO] Plugin count: XX
```

### 2. API 响应

```json
{
  "code": 0,
  "data": {
    "plugin_list": [
      {
        "id": 101,
        "name_for_model": "pairat_remote_exec",
        "name_for_human": "Pairat Remote Exec",
        "manifest": {
          "api": {
            "type": "coze-studio-mcp"
          }
        }
      }
    ]
  }
}
```

### 3. 前端界面

- ✅ 搜索框输入 "Pairat" 能找到插件
- ✅ 插件卡片显示 "Pairat Remote Exec"
- ✅ 插件详情显示工具列表（remote_exec, vmcheck）
- ✅ 能添加到 Workflow

---

## 📝 总结

**关键点**：

1. ⚠️ **文件必须在 `backend/resources/conf/plugin/pluginproduct/` 目录**
2. ✅ 配置文件语法正确
3. ✅ 插件 ID 不冲突（101 应该是可用的）
4. ✅ 重启后端服务
5. ✅ 查看启动日志确认加载成功

**推荐操作顺序**：

```bash
# 1. 复制文件到正确位置
cp backend/conf/plugin/pluginproduct/pairat_remote_exec.yaml \
   backend/resources/conf/plugin/pluginproduct/

# 2. 同步 plugin_meta.yaml（如果需要）
# 检查并合并配置

# 3. 重启后端
cd backend && make restart

# 4. 查看日志
tail -f logs/app.log | grep -i "plugin\|101"

# 5. 测试前端搜索
# 打开浏览器，搜索 "Pairat"
```

如果以上步骤都正确，重启后就能在前端搜索到你的 MCP 插件了！🎉

---

**文档版本**：v1.0  
**创建日期**：2025-11-09

