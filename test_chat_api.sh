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
# Quick test script for Chat API
#

API_KEY="pat_4be7a56bc8c6450b6e5d91e0534108cf2c13ca496423a04fd5f03052d0f306a9"
BASE_URL="https://iaas-ops-agent.pyinfra.work"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查参数
if [ $# -lt 3 ]; then
    echo "用法: $0 <workflow_id> <app_id> <message> [conversation_id]"
    echo ""
    echo "示例:"
    echo "  $0 7574700179186515968 7574699996000288768 \"你好\""
    echo "  $0 7574700179186515968 7574699996000288768 \"继续聊天\" 7574707195900592128"
    exit 1
fi

workflow_id=$1
app_id=$2
user_message=$3
conversation_id=${4:-""}

echo "🚀 测试 Chat API"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Workflow ID: $workflow_id"
echo "App ID: $app_id"
echo "Message: $user_message"
if [ -n "$conversation_id" ]; then
    echo "Conversation ID: $conversation_id"
else
    echo "Conversation ID: (自动创建)"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 构建请求体（根据真实 API 格式）
if command -v jq &> /dev/null; then
    additional_messages=$(jq -n \
        --arg content "$user_message" \
        '[{role: "user", content_type: "text", content: $content}]')
    
    if [ -n "$conversation_id" ]; then
        request_body=$(jq -n \
            --arg wid "$workflow_id" \
            --arg aid "$app_id" \
            --arg cid "$conversation_id" \
            --argjson msgs "$additional_messages" \
            '{
                workflow_id: $wid,
                app_id: $aid,
                conversation_id: $cid,
                connector_id: "10000010",
                execute_mode: "DEBUG",
                additional_messages: $msgs,
                ext: {
                    _caller: "CANVAS"
                }
            }')
    else
        request_body=$(jq -n \
            --arg wid "$workflow_id" \
            --arg aid "$app_id" \
            --argjson msgs "$additional_messages" \
            '{
                workflow_id: $wid,
                app_id: $aid,
                connector_id: "10000010",
                execute_mode: "DEBUG",
                additional_messages: $msgs,
                ext: {
                    _caller: "CANVAS"
                }
            }')
    fi
    
    echo "📤 请求体:"
    echo "$request_body" | jq '.'
    echo ""
else
    user_message_escaped=$(echo "$user_message" | sed 's/"/\\"/g')
    if [ -n "$conversation_id" ]; then
        request_body="{\"additional_messages\":[{\"role\":\"user\",\"content_type\":\"text\",\"content\":\"$user_message_escaped\"}],\"connector_id\":\"10000010\",\"workflow_id\":\"$workflow_id\",\"execute_mode\":\"DEBUG\",\"app_id\":\"$app_id\",\"conversation_id\":\"$conversation_id\",\"ext\":{\"_caller\":\"CANVAS\"}}"
    else
        request_body="{\"additional_messages\":[{\"role\":\"user\",\"content_type\":\"text\",\"content\":\"$user_message_escaped\"}],\"connector_id\":\"10000010\",\"workflow_id\":\"$workflow_id\",\"execute_mode\":\"DEBUG\",\"app_id\":\"$app_id\",\"ext\":{\"_caller\":\"CANVAS\"}}"
    fi
    
    echo "📤 请求体:"
    echo "$request_body"
    echo ""
fi

echo -e "${BLUE}🤖 AI 回复:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 发送请求并解析流式响应
curl -N -s -X POST "$BASE_URL/v1/workflows/chat" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d "$request_body" 2>/dev/null | while IFS= read -r line; do
    
    # 解析 SSE 格式
    if [[ $line == event:* ]]; then
        event_type=$(echo "$line" | sed 's/^event: *//')
        echo -e "\n${YELLOW}[事件: $event_type]${NC}" >&2
    elif [[ $line == data:* ]]; then
        data_content=$(echo "$line" | sed 's/^data: *//')
        
        # 跳过空数据或 [DONE]
        if [ -z "$data_content" ] || [ "$data_content" == "[DONE]" ]; then
            continue
        fi
        
        # 尝试提取 content 字段（使用 jq）
        if command -v jq &> /dev/null; then
            # 提取消息内容并打印（不换行）
            message_content=$(echo "$data_content" | jq -r '.message.content // empty' 2>/dev/null)
            if [ -n "$message_content" ]; then
                echo -n "$message_content"
            fi
            
            # 提取事件类型
            event=$(echo "$data_content" | jq -r '.event // empty' 2>/dev/null)
            if [ "$event" == "error" ]; then
                echo -e "\n${RED}❌ 错误: $(echo "$data_content" | jq -r '.message // .')${NC}" >&2
            fi
        else
            # 如果没有 jq，直接输出原始数据
            echo "$data_content"
        fi
    fi
done

echo ""
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ 测试完成${NC}"
echo ""

