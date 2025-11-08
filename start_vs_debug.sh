#!/bin/bash
#
# Copyright 2025 coze-dev Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

echo "🚀 启动 Coze Studio 调试环境..."
echo ""

# 进入项目目录
cd "$(dirname "$0")"

# 检查 Docker
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行！请先启动 Docker Desktop"
    exit 1
fi

echo "✅ Docker 正在运行"
echo ""

# 创建环境变量文件（如果不存在）
if [ ! -f "docker/.env.debug" ]; then
    echo "📄 创建 docker 环境配置文件..."
    cp docker/.env.debug.example docker/.env.debug
    echo "✅ docker/.env.debug 已创建"
    echo ""
fi

# 为 backend 目录创建 .env.debug（如果不存在）
if [ ! -f "backend/.env.debug" ]; then
    echo "📄 创建 backend 环境配置文件..."
    cp docker/.env.debug backend/.env.debug
    echo "✅ backend/.env.debug 已创建"
    echo ""
fi

# 创建 resources 目录的软链接（如果不存在）
if [ ! -d "backend/resources/conf" ]; then
    echo "📄 创建 resources 目录软链接..."
    mkdir -p backend/resources
    ln -sf ../conf backend/resources/conf
    echo "✅ backend/resources/conf -> conf 软链接已创建"
    echo ""
fi

# 启动中间件
echo "🐳 启动中间件容器（MySQL, Redis, ES, Milvus, MinIO, Etcd）..."
echo "   这可能需要几分钟时间，首次运行需要下载 Docker 镜像..."
echo ""

docker compose -f docker/docker-compose-debug.yml \
    --env-file docker/.env.debug \
    --profile middleware \
    up -d --wait

echo ""
echo "✅ 中间件启动成功！"
echo ""

# 显示运行中的容器
echo "📊 运行中的容器："
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "NAMES|coze-"

echo ""
echo "🎉 环境准备完成！现在可以："


echo "   1. 在 VS Code 中按 F5 启动调试"
echo "   2. 或在终端运行: cd backend && go run main.go"
echo ""
echo "📝 访问地址："
echo "   - API Server:  http://localhost:8888"
echo "   - MinIO Console: http://localhost:19001 (minioadmin/minioadmin)"
echo ""

# 询问是否启动前端
read -p "是否启动前端开发服务器？[y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo "🌐 启动前端开发服务器..."
    ./start_debug_web.sh
else
    echo ""
    echo "💡 你可以稍后手动启动前端："
    echo "   ./start_debug_web.sh"
    echo ""
fi


