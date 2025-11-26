#!/bin/bash
# 修复 coze-web 容器挂载问题

set -e

echo "🔧 修复 coze-web 容器挂载问题..."
echo ""

# 1. 停止并删除旧容器
echo "1️⃣ 停止并删除旧容器..."
docker stop coze-web 2>/dev/null || true
docker rm coze-web 2>/dev/null || true
echo "✅ 旧容器已删除"
echo ""

# 2. 检查配置文件是否存在
echo "2️⃣ 检查配置文件..."
if [ ! -f "./nginx/nginx.conf" ]; then
    echo "❌ 错误: ./nginx/nginx.conf 不存在"
    echo "当前目录: $(pwd)"
    exit 1
fi

if [ ! -f "./nginx/conf.d/docker.conf" ]; then
    echo "❌ 错误: ./nginx/conf.d/docker.conf 不存在"
    echo "当前目录: $(pwd)"
    exit 1
fi

echo "✅ 配置文件存在"
echo "   - $(pwd)/nginx/nginx.conf"
echo "   - $(pwd)/nginx/conf.d/docker.conf"
echo ""

# 3. 验证配置文件路径（绝对路径）
echo "3️⃣ 验证配置文件路径..."
NGINX_CONF=$(realpath ./nginx/nginx.conf)
DOCKER_CONF=$(realpath ./nginx/conf.d/docker.conf)

echo "绝对路径:"
echo "   - $NGINX_CONF"
echo "   - $DOCKER_CONF"
echo ""

# 4. 重新创建容器
echo "4️⃣ 重新创建容器..."
docker compose up -d coze-web

# 5. 等待容器启动
echo "5️⃣ 等待容器启动..."
sleep 3

# 6. 检查容器状态
echo "6️⃣ 检查容器状态..."
if docker ps | grep -q coze-web; then
    echo "✅ 容器启动成功"
    docker ps | grep coze-web
    echo ""
    
    # 7. 验证配置
    echo "7️⃣ 验证配置..."
    docker exec coze-web nginx -t
    echo ""
    
    echo "🎉 修复完成！"
else
    echo "❌ 容器启动失败"
    echo "查看日志:"
    docker logs coze-web
    exit 1
fi

