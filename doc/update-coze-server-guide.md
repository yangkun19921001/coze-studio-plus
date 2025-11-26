# 使用编译好的镜像更新 coze-server 服务指南

## 场景说明

当你下载了编译好的镜像 tar 文件（如 `coze-studio-server-latest.tar`），需要更新 `coze-server` 服务，但**不想删除缓存和数据**。

## ✅ 正确的操作步骤

### 方法 1：使用 docker-compose（推荐）

```bash
# 1. 进入 docker-compose 目录
cd /root/coze-studio/coze-studio-plus/docker

# 2. 加载新镜像（如果镜像文件在其他位置，请修改路径）
docker load -i /root/coze-studio/coze-studio-server-latest.tar

# 3. 停止 coze-server 服务（不删除容器）
docker compose stop coze-server

# 4. 重新启动 coze-server（使用新镜像，不删除容器）
docker compose up -d --no-deps coze-server

# 5. 查看日志确认启动成功
docker compose logs -f coze-server
```

**关键点**：
- ✅ 使用 `stop` 而不是 `down`（`down` 会删除容器）
- ✅ 使用 `--no-deps` 只更新 coze-server，不影响其他服务
- ✅ **不使用** `--force-recreate`（会删除容器）

### 方法 2：一行命令更新

```bash
cd /root/coze-studio/coze-studio-plus/docker && \
docker load -i /root/coze-studio/coze-studio-server-latest.tar && \
docker compose stop coze-server && \
docker compose up -d --no-deps coze-server
```

### 方法 3：如果镜像已加载，直接更新

如果镜像已经加载过了，只需要：

```bash
cd /root/coze-studio/coze-studio-plus/docker
docker compose stop coze-server
docker compose up -d --no-deps coze-server
```

## ❌ 错误操作（会导致数据丢失）

### 错误示例 1：使用 --force-recreate

```bash
# ❌ 错误：会删除容器，可能导致数据丢失
docker compose up -d --force-recreate coze-server
```

### 错误示例 2：先删除容器再创建

```bash
# ❌ 错误：会删除容器
docker compose rm -f coze-server
docker compose up -d coze-server
```

### 错误示例 3：使用 down 命令

```bash
# ❌ 错误：会删除所有容器（包括 coze-server）
docker compose down
docker compose up -d
```

## 📋 完整操作流程（带检查）

```bash
#!/bin/bash
set -e

# 1. 进入 docker-compose 目录
cd /root/coze-studio/coze-studio-plus/docker

# 2. 检查当前服务状态
echo "📊 当前服务状态："
docker compose ps coze-server

# 3. 检查镜像文件是否存在
IMAGE_TAR="/root/coze-studio/coze-studio-server-latest.tar"
if [ ! -f "$IMAGE_TAR" ]; then
    echo "❌ 镜像文件不存在: $IMAGE_TAR"
    exit 1
fi

# 4. 加载新镜像
echo "📦 加载镜像..."
docker load -i "$IMAGE_TAR"

# 5. 验证镜像是否加载成功
if docker images | grep -q "cozedev/coze-studio-server.*latest"; then
    echo "✅ 镜像加载成功"
else
    echo "❌ 镜像加载失败"
    exit 1
fi

# 6. 停止服务（不删除容器）
echo "🛑 停止 coze-server..."
docker compose stop coze-server

# 7. 重新启动服务（使用新镜像）
echo "🚀 启动 coze-server..."
docker compose up -d --no-deps coze-server

# 8. 等待服务启动
echo "⏳ 等待服务启动..."
sleep 5

# 9. 检查服务状态
echo "📊 服务状态："
docker compose ps coze-server

# 10. 查看日志（最后 20 行）
echo "📋 最近日志："
docker compose logs --tail=20 coze-server

echo "✅ 更新完成！"
```

## 🔍 验证更新是否成功

### 1. 检查容器状态

```bash
docker compose ps coze-server
```

应该看到状态为 `Up` 或 `Up (healthy)`

### 2. 检查镜像版本

```bash
# 查看镜像 ID
docker images cozedev/coze-studio-server:latest

# 查看容器使用的镜像
docker inspect coze-server | grep Image
```

### 3. 检查日志

```bash
# 查看实时日志
docker compose logs -f coze-server

# 查看最近 50 行日志
docker compose logs --tail=50 coze-server
```

### 4. 检查数据是否还在

```bash
# 检查 MySQL 中的工作流数据
docker exec -it coze-mysql mysql -u coze -pcoze123 opencoze -e "SELECT COUNT(*) as workflow_count FROM workflow_draft;"
```

## 🛠️ 故障排查

### 问题 1：镜像加载失败

```bash
# 检查镜像文件完整性
file /root/coze-studio/coze-studio-server-latest.tar

# 检查磁盘空间
df -h

# 手动加载并查看详细错误
docker load -i /root/coze-studio/coze-studio-server-latest.tar
```

### 问题 2：容器启动失败

```bash
# 查看详细错误日志
docker compose logs coze-server

# 检查容器配置
docker compose config coze-server

# 检查端口占用
netstat -tulpn | grep 8888
```

### 问题 3：服务无法访问

```bash
# 检查容器是否运行
docker ps | grep coze-server

# 检查网络连接
docker network ls
docker network inspect coze-studio_coze-network

# 检查服务健康状态
curl http://localhost:8888/health
```

## 📝 快速参考命令

```bash
# 加载镜像
docker load -i /path/to/coze-studio-server-latest.tar

# 停止服务（不删除）
docker compose stop coze-server

# 启动服务（使用新镜像）
docker compose up -d --no-deps coze-server

# 查看状态
docker compose ps coze-server

# 查看日志
docker compose logs -f coze-server

# 重启服务
docker compose restart coze-server
```

## 💡 最佳实践

1. **更新前备份**（可选但推荐）：
   ```bash
   # 备份 MySQL 数据
   docker exec coze-mysql mysqldump -u coze -pcoze123 opencoze > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **在低峰期更新**：避免影响正在使用服务的用户

3. **更新后验证**：
   - 检查服务是否正常启动
   - 检查数据是否完整
   - 测试关键功能

4. **保留旧镜像**（可选）：
   ```bash
   # 给旧镜像打标签
   docker tag cozedev/coze-studio-server:latest cozedev/coze-studio-server:backup-$(date +%Y%m%d)
   ```

## 🎯 总结

**正确的更新流程**：
1. `docker load -i <镜像文件>` - 加载镜像
2. `docker compose stop coze-server` - 停止服务
3. `docker compose up -d --no-deps coze-server` - 启动服务

**关键原则**：
- ✅ 使用 `stop` + `up`，不要使用 `--force-recreate`
- ✅ 使用 `--no-deps` 只更新目标服务
- ✅ 数据在 MySQL 中，不会丢失（已挂载）

