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

#
# 预拉取 Docker 镜像脚本
# 解决镜像拉取失败的问题
#

set -e

echo "📥 预拉取 Docker 镜像..."
echo ""

# 需要拉取的镜像列表
IMAGES=(
    "arigaio/atlas:0.35.0-community-alpine"
    "minio/mc:RELEASE.2025-05-21T01-59-54Z-cpuv1"
)

# 镜像加速器列表（按优先级）
MIRRORS=(
    "https://docker.mirrors.ustc.edu.cn"
    "https://hub-mirror.c.163.com"
    "https://mirror.baidubce.com"
)

# 尝试从镜像加速器拉取
pull_from_mirror() {
    local image=$1
    local mirror=$2
    
    # 构建镜像加速器 URL
    local mirror_image="${mirror}/${image}"
    
    echo "  尝试从镜像加速器拉取: ${mirror_image}"
    if docker pull "${mirror_image}" 2>/dev/null; then
        # 重新标记为原始镜像名
        docker tag "${mirror_image}" "${image}"
        echo "  ✅ 成功从镜像加速器拉取: ${image}"
        return 0
    fi
    return 1
}

# 拉取单个镜像
pull_image() {
    local image=$1
    echo ""
    echo "📦 拉取镜像: ${image}"
    
    # 先尝试直接拉取（会使用配置的镜像加速器）
    if docker pull "${image}" 2>/dev/null; then
        echo "  ✅ 成功拉取: ${image}"
        return 0
    fi
    
    # 如果失败，尝试从各个镜像加速器拉取
    for mirror in "${MIRRORS[@]}"; do
        if pull_from_mirror "${image}" "${mirror}"; then
            return 0
        fi
    done
    
    echo "  ⚠️  无法拉取镜像: ${image}"
    echo "  💡 提示：可能需要配置代理或使用其他网络"
    return 1
}

# 检查镜像是否已存在
check_image_exists() {
    local image=$1
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${image}$"; then
        return 0
    fi
    return 1
}

# 主逻辑
SUCCESS_COUNT=0
FAILED_COUNT=0

for image in "${IMAGES[@]}"; do
    # 检查镜像是否已存在
    if check_image_exists "${image}"; then
        echo "✅ 镜像已存在: ${image}"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        continue
    fi
    
    # 尝试拉取镜像
    if pull_image "${image}"; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
done

echo ""
echo "📊 拉取结果："
echo "  ✅ 成功: ${SUCCESS_COUNT}"
echo "  ❌ 失败: ${FAILED_COUNT}"

if [ ${FAILED_COUNT} -gt 0 ]; then
    echo ""
    echo "⚠️  部分镜像拉取失败，但可以继续尝试启动服务"
    echo "   如果启动时仍然失败，请："
    echo "   1. 重启 Docker Desktop"
    echo "   2. 检查网络连接"
    echo "   3. 配置代理（如果有）"
    exit 1
fi

echo ""
echo "✅ 所有镜像拉取完成！"

