# Nginx 配置文件说明

## 配置文件说明

### 1. `docker.conf` - Docker 容器专用配置
- **用途**：Docker 容器 `coze-web` 使用
- **监听端口**：80（映射到宿主机 8888）
- **代理目标**：`coze-server:8888`（Docker 网络内）
- **挂载方式**：通过 `docker-compose.yml` 自动挂载到容器内的 `/etc/nginx/conf.d/default.conf`
- **超时设置**：已配置 300 秒（5 分钟）超时

### 2. `default.conf` - 物理机专用配置
- **用途**：物理机上的 nginx 使用
- **监听端口**：8101
- **代理目标**：`localhost:8888`（宿主机上的 Docker 端口）
- **安装位置**：`/etc/nginx/conf.d/default.conf`
- **超时设置**：已配置 300 秒（5 分钟）超时

## 配置架构

```
客户端请求
    ↓
物理机 nginx (8101) ← default.conf
    ↓
localhost:8888
    ↓
Docker coze-web (80) ← docker.conf (挂载为 default.conf)
    ↓
coze-server:8888
```

## 使用方法

### Docker 容器配置（自动）

Docker 容器会自动挂载 `docker.conf`，无需手动操作：

```bash
# 修改配置后，重新加载 nginx
docker exec coze-web nginx -s reload

# 或者重启容器
docker compose restart coze-web
```

### 物理机配置（手动）

需要将 `default.conf` 复制到物理机的 nginx 配置目录：

```bash
# 1. 备份现有配置
sudo cp /etc/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf.backup

# 2. 复制新配置
sudo cp /path/to/coze-studio/docker/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf

# 3. 测试配置
sudo nginx -t

# 4. 重新加载配置
sudo systemctl reload nginx
```

## 配置更新流程

### 更新 Docker 容器配置

```bash
# 1. 修改配置文件
vi docker/nginx/conf.d/docker.conf

# 2. 重新加载容器内的 nginx
docker exec coze-web nginx -t  # 先测试
docker exec coze-web nginx -s reload  # 重新加载
```

### 更新物理机配置

```bash
# 1. 修改配置文件
vi docker/nginx/conf.d/default.conf

# 2. 复制到物理机
sudo cp docker/nginx/conf.d/default.conf /etc/nginx/conf.d/default.conf

# 3. 测试并重新加载
sudo nginx -t
sudo systemctl reload nginx
```

## 验证配置

### 验证 Docker 容器配置

```bash
# 检查容器内的配置
docker exec coze-web cat /etc/nginx/conf.d/default.conf | grep -A 3 "proxy_read_timeout"

# 检查 nginx 配置语法
docker exec coze-web nginx -t
```

### 验证物理机配置

```bash
# 检查物理机配置
sudo cat /etc/nginx/conf.d/default.conf | grep -A 3 "proxy_read_timeout"

# 检查 nginx 配置语法
sudo nginx -t
```

## 注意事项

1. **配置文件分离**：两个配置文件互不干扰，可以独立修改
2. **超时设置**：两个配置都已设置 300 秒超时，支持长时间运行的工作流
3. **只读挂载**：Docker 容器内的配置文件是只读挂载（`:ro`），必须在宿主机上修改
4. **配置同步**：修改配置文件后，记得重新加载 nginx 使配置生效

## 故障排查

### 问题：Docker 容器配置不生效

```bash
# 检查挂载是否正确
docker inspect coze-web | grep -A 5 "Mounts"

# 检查配置文件内容
docker exec coze-web cat /etc/nginx/conf.d/default.conf

# 检查 nginx 错误日志
docker logs coze-web
```

### 问题：物理机配置不生效

```bash
# 检查配置文件是否存在
ls -la /etc/nginx/conf.d/default.conf

# 检查 nginx 配置语法
sudo nginx -t

# 检查 nginx 错误日志
sudo tail -f /var/log/nginx/error.log
```

