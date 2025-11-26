# 工作流数据存储位置深度分析

## 1. 工作流数据存储架构

### 1.1 数据库存储（MySQL）

工作流的核心数据存储在 MySQL 数据库中，数据目录已挂载到 `./data/mysql`，**数据是持久化的**。

#### 主要数据表：

1. **`workflow_draft`** - 工作流草稿表
   - `id`: 工作流 ID
   - `canvas`: **前端画布 JSON 数据**（这是你编辑的工作流内容）
   - `input_params`: 输入参数配置
   - `output_params`: 输出参数配置
   - `commit_id`: 草稿版本标识
   - `modified`: 是否已修改
   - `test_run_success`: 测试运行是否成功
   - `updated_at`: 更新时间

2. **`workflow_meta`** - 工作流元数据表
   - `id`: 工作流 ID
   - `name`: 工作流名称
   - `description`: 描述
   - `icon_uri`: 图标 URI
   - `status`: 发布状态
   - `space_id`: 空间 ID
   - `app_id`: 应用 ID

3. **`workflow_version`** - 工作流版本表
   - 存储已发布的版本信息

4. **`workflow_snapshot`** - 工作流快照表
   - 存储执行时的快照数据

### 1.2 前端缓存（浏览器 localStorage）

前端使用 localStorage 存储一些 UI 状态，**这些是浏览器端的缓存**：

- `workspace-develop-filters`: 项目开发页面的过滤器
- `workspace-library-filters`: 资源库页面的过滤器
- `workspace-spaceId`: 当前工作空间 ID
- `flowide:*`: 工作流 IDE 的本地缓存

**注意**：工作流内容本身**不存储在 localStorage**，只存储 UI 状态。

### 1.3 Redis 缓存

Redis 用于存储：
- 执行状态缓存
- Checkpoint 数据
- 临时数据

数据目录：`./data/bitnami/redis`（已持久化）

### 1.4 MinIO 对象存储

文件、图片等存储在 MinIO：
- 数据目录：`./data/minio`（已持久化）

## 2. 数据丢失原因分析

### 2.1 使用 `--force-recreate` 的影响

```bash
docker compose up -d --force-recreate coze-server
```

**这个命令会**：
1. ✅ 停止旧容器
2. ❌ **删除旧容器**
3. ✅ 创建新容器
4. ✅ MySQL 数据**不会丢失**（因为已挂载）

### 2.2 可能丢失的数据

1. **前端缓存（localStorage）**：
   - 如果浏览器缓存被清除，UI 状态会丢失
   - 但工作流数据在数据库中，不会丢失

2. **Redis 缓存**：
   - 执行状态、临时数据可能丢失
   - 但工作流定义在数据库中，不会丢失

3. **容器内的临时文件**：
   - 如果数据存储在容器内的 `/tmp` 或其他未挂载目录
   - 容器删除后这些数据会丢失

## 3. 如何只更新 coze-server 而不删除缓存

### 方法 1：使用 `--no-deps` 只更新 coze-server（推荐）

```bash
# 1. 先拉取新镜像
docker pull cozedev/coze-studio-server:latest

# 2. 停止并重新创建 coze-server（不删除容器，只更新）
docker compose up -d --no-deps coze-server

# 或者先停止再启动
docker compose stop coze-server
docker compose up -d --no-deps coze-server
```

**优点**：
- 不会删除容器
- 不会影响其他服务
- 保留容器内的临时数据（如果有）

### 方法 2：使用 `restart` 命令（最简单）

```bash
# 1. 拉取新镜像
docker pull cozedev/coze-studio-server:latest

# 2. 重启容器（会使用新镜像）
docker compose restart coze-server
```

**注意**：`restart` 不会自动使用新镜像，需要先 `pull`。

### 方法 3：先 pull 再 up（推荐用于更新镜像）

```bash
# 1. 拉取新镜像
docker pull cozedev/coze-studio-server:latest

# 2. 停止容器
docker compose stop coze-server

# 3. 重新创建（不使用 --force-recreate，会保留容器配置）
docker compose up -d coze-server
```

### 方法 4：使用 `docker compose up` 的 `--pull` 选项

```bash
# 自动拉取最新镜像并更新
docker compose up -d --pull always --no-deps coze-server
```

## 4. 检查数据是否真的丢失

### 4.1 检查 MySQL 中的数据

```bash
# 进入 MySQL 容器
docker exec -it coze-mysql mysql -u coze -pcoze123 opencoze

# 查看工作流草稿数量
SELECT COUNT(*) FROM workflow_draft;

# 查看最近的工作流
SELECT id, name, updated_at FROM workflow_meta ORDER BY updated_at DESC LIMIT 10;

# 查看工作流草稿内容
SELECT id, LEFT(canvas, 100) as canvas_preview, updated_at FROM workflow_draft LIMIT 5;
```

### 4.2 检查前端缓存

打开浏览器开发者工具：
- Application → Local Storage → 查看是否有缓存数据
- 如果 localStorage 被清除，只是 UI 状态丢失，数据还在数据库中

## 5. 最佳实践

### 5.1 更新 coze-server 的标准流程

```bash
# 1. 备份（可选，但推荐）
docker exec coze-mysql mysqldump -u coze -pcoze123 opencoze > backup_$(date +%Y%m%d_%H%M%S).sql

# 2. 拉取新镜像
docker pull cozedev/coze-studio-server:latest

# 3. 更新服务（不删除容器）
docker compose up -d --no-deps coze-server

# 4. 检查服务状态
docker compose ps coze-server
docker compose logs -f coze-server
```

### 5.2 避免数据丢失的建议

1. **不要使用 `--force-recreate`**，除非确定要删除容器
2. **定期备份 MySQL 数据**：
   ```bash
   docker exec coze-mysql mysqldump -u coze -pcoze123 opencoze > backup.sql
   ```
3. **确保数据目录已挂载**：
   - MySQL: `./data/mysql`
   - Redis: `./data/bitnami/redis`
   - MinIO: `./data/minio`
   - Elasticsearch: `./data/bitnami/elasticsearch`

## 6. 数据恢复（如果确实丢失）

如果数据真的丢失了，可以：

1. **从备份恢复**：
   ```bash
   docker exec -i coze-mysql mysql -u coze -pcoze123 opencoze < backup.sql
   ```

2. **检查是否有旧容器数据**：
   ```bash
   # 查看所有容器（包括已停止的）
   docker ps -a | grep coze-server
   
   # 如果有旧容器，可以从中恢复数据
   docker cp <旧容器ID>:/app/data ./data/coze-server/
   ```

3. **检查 MySQL binlog**（如果启用了）：
   ```bash
   docker exec coze-mysql mysqlbinlog /var/lib/mysql/binlog.* > recovery.sql
   ```

