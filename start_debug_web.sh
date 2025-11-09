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

echo "🚀 启动 Coze Studio 前端开发服务器..."
echo ""

# 进入前端目录
cd "$(dirname "$0")/frontend"

# 检查 node 版本
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 21 ]; then
    echo "⚠️  警告：Node.js 版本过低（当前：$(node -v)），建议使用 Node.js >= 21"
    echo "   继续尝试启动..."
    echo ""
fi

# 配置国内镜像源（解决网络问题）
echo "🔧 配置 npm 镜像源..."
npm config set registry https://registry.npmmirror.com
pnpm config set registry https://registry.npmmirror.com 2>/dev/null || true
echo "✅ 镜像源配置完成"
echo ""

# 检查依赖是否已安装
if [ ! -d "node_modules" ]; then
    echo "📦 首次运行，正在安装依赖..."
    echo "   这可能需要 10-20 分钟，请耐心等待..."
    echo ""
    
    # 检查是否安装了 Rush
    if ! command -v rush &> /dev/null; then
        echo "📦 安装 Rush..."
        npm install -g @microsoft/rush
        echo ""
    fi
    
    # 清理之前失败的安装
    if [ -d "common/temp" ]; then
        echo "🧹 清理之前的安装缓存..."
        rm -rf common/temp
        echo ""
    fi
    
    # 安装依赖（使用镜像源）
    echo "📦 执行 rush install（使用国内镜像）..."
    rush install --max-install-attempts 3
    echo ""
fi

echo "🎯 启动前端开发服务器..."
echo ""
echo "📝 注意事项："
echo "   - 前端将运行在: http://localhost:5001"
echo "   - 后端 API 地址: http://localhost:8888"
echo "   - 请确保后端服务已启动（在 VS Code 中按 F5）"
echo ""

# 进入应用目录
cd apps/coze-studio

# 启动开发服务器
PORT=5001 npm run dev

