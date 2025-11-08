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


echo "=========================================="
echo "快速重启后端（保留之前的调试信息）"
echo "=========================================="
echo ""

echo "📋 请在 VS Code 中按以下步骤操作："
echo ""
echo "1️⃣  停止当前的调试会话（点击红色停止按钮）"
echo "2️⃣  按 F5 重新启动调试"
echo "3️⃣  等待后端启动完成（看到 'Hertz server listening on...' 日志）"
echo "4️⃣  在 Coze Studio 中重新运行工作流（输入'新能源车推荐'）"
echo "5️⃣  查看 VS Code 调试控制台中的新日志，特别关注："
echo "     - '🔧 processResponse called' 开头的日志"
echo "     - '🔧 Trimmed response' 的内容"
echo ""
echo "=========================================="
