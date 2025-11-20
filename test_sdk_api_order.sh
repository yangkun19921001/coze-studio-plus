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


# 测试 SDK API 请求顺序问题
# 用于对比 npm run dev (正常) 和编译 SDK (异常) 的请求顺序

# 不要使用 set -e，手动处理错误
# set -e

# 配置
BASE_URL="https://iaas-ops-agent.pyinfra.work"
TOKEN="pat_4be7a56bc8c6450b6e5d91e0534108cf2c13ca496423a04fd5f03052d0f306a9"
APP_ID="7574699996000288768"
WORKFLOW_ID="7574700179186515968"
CONNECTOR_ID="999"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Coze SDK API 请求顺序测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 函数：获取 workflow 信息
get_workflow_info() {
    echo -e "${YELLOW}[1] GET workflow 信息...${NC}"
    WORKFLOW_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Accept-Language: zh" \
        "${BASE_URL}/v1/workflows/${WORKFLOW_ID}?connector_id=${CONNECTOR_ID}&is_debug=&caller=")
    
    HTTP_CODE=$(echo "$WORKFLOW_RESPONSE" | tail -n1)
    RESPONSE_BODY=$(echo "$WORKFLOW_RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✅ Workflow 获取成功 (HTTP $HTTP_CODE)${NC}"
        echo "$RESPONSE_BODY" | jq -r '.data.role.name // "N/A"' | head -1 | xargs -I {} echo "   Workflow名称: {}"
    else
        echo -e "${RED}❌ Workflow 获取失败 (HTTP $HTTP_CODE)${NC}"
        echo "$RESPONSE_BODY" | jq '.' 2>/dev/null || echo "$RESPONSE_BODY"
    fi
    echo ""
}

# 函数：创建会话
create_conversation() {
    # 不使用颜色代码，避免污染输出
    echo "[2] POST 创建会话..." >&2
    CREATE_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -H "Accept-Language: zh" \
        -d "{
            \"app_id\": \"${APP_ID}\",
            \"conversation_name\": \"Default\",
            \"get_or_create\": true,
            \"workflow_id\": \"${WORKFLOW_ID}\",
            \"connector_id\": \"${CONNECTOR_ID}\"
        }" \
        "${BASE_URL}/v1/workflow/conversation/create" 2>&1)
    
    HTTP_CODE=$(echo "$CREATE_RESPONSE" | tail -n1)
    RESPONSE_BODY=$(echo "$CREATE_RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        CONVERSATION_ID=$(echo "$RESPONSE_BODY" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$CONVERSATION_ID" ]; then
            echo "✅ 会话创建成功 (HTTP $HTTP_CODE)" >&2
            echo "   Conversation ID: $CONVERSATION_ID" >&2
            # 只输出纯净的 conversation_id 到 stdout
            printf "%s" "$CONVERSATION_ID"
        else
            echo "❌ 会话创建失败：返回数据中无 conversation_id" >&2
            echo "$RESPONSE_BODY" | jq '.' 2>&1 >&2 || echo "$RESPONSE_BODY" >&2
        fi
    else
        echo "❌ 会话创建失败 (HTTP $HTTP_CODE)" >&2
        echo "$RESPONSE_BODY" | jq '.' 2>&1 >&2 || echo "$RESPONSE_BODY" >&2
    fi
}

# 函数：获取消息列表
get_message_list() {
    local CONV_ID=$1
    echo -e "${YELLOW}[3] GET 消息列表...${NC}"
    
    if [ -z "$CONV_ID" ]; then
        echo -e "${RED}❌ 错误：Conversation ID 为空${NC}"
        return 1
    fi
    
    LIST_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Accept-Language: zh" \
        "${BASE_URL}/v1/conversation/message/list?conversation_id=${CONV_ID}") || {
        echo -e "${RED}❌ curl 请求失败${NC}"
        return 1
    }
    
    HTTP_CODE=$(echo "$LIST_RESPONSE" | tail -n1)
    RESPONSE_BODY=$(echo "$LIST_RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        MESSAGE_COUNT=$(echo "$RESPONSE_BODY" | jq -r '.data | length // 0' 2>/dev/null || echo "0")
        echo -e "${GREEN}✅ 消息列表获取成功 (HTTP $HTTP_CODE)${NC}"
        echo "   消息数量: $MESSAGE_COUNT"
    else
        echo -e "${RED}❌ 消息列表获取失败 (HTTP $HTTP_CODE)${NC}"
        echo "$RESPONSE_BODY" | jq '.' 2>/dev/null || echo "$RESPONSE_BODY"
    fi
    echo ""
}

# 函数：发送消息
send_message() {
    local CONV_ID=$1
    echo -e "${YELLOW}[4] POST 发送消息 (你好)...${NC}"
    
    if [ -z "$CONV_ID" ]; then
        echo -e "${RED}❌ 错误：Conversation ID 为空${NC}"
        return 1
    fi
    
    CHAT_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Authorization: Bearer ${TOKEN}" \
        -H "Content-Type: application/json" \
        -H "Accept-Language: zh" \
        -d "{
            \"conversation_id\": \"${CONV_ID}\",
            \"workflow_id\": \"${WORKFLOW_ID}\",
            \"app_id\": \"${APP_ID}\",
            \"additional_messages\": [{
                \"role\": \"user\",
                \"content\": \"你好\",
                \"content_type\": \"text\"
            }],
            \"connector_id\": \"${CONNECTOR_ID}\"
        }" \
        "${BASE_URL}/v1/workflows/chat") || {
        echo -e "${RED}❌ curl 请求失败${NC}"
        return 1
    }
    
    HTTP_CODE=$(echo "$CHAT_RESPONSE" | tail -n1)
    RESPONSE_BODY=$(echo "$CHAT_RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✅ 消息发送成功 (HTTP $HTTP_CODE)${NC}"
        # 只显示第一行响应（流式响应）
        echo "$RESPONSE_BODY" | head -1 | jq '.' 2>/dev/null || echo "$RESPONSE_BODY" | head -1
    else
        echo -e "${RED}❌ 消息发送失败 (HTTP $HTTP_CODE)${NC}"
        echo "$RESPONSE_BODY" | jq '.' 2>/dev/null || echo "$RESPONSE_BODY"
    fi
    echo ""
}

# ====================
# 测试 1: 正常顺序（npm run dev）
# ====================
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  测试 1: 正常顺序 (npm run dev)       ║${NC}"
echo -e "${BLUE}║  workflows → create → list → chat     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Step 1: 获取 workflow
get_workflow_info

# Step 2: 创建会话
CONVERSATION_ID_1=$(create_conversation)
echo ""
if [ -z "$CONVERSATION_ID_1" ]; then
    echo -e "${RED}❌ 无法继续测试，会话创建失败${NC}"
else
    # Step 3: 获取消息列表
    get_message_list "$CONVERSATION_ID_1" || echo -e "${YELLOW}⚠️  消息列表获取跳过${NC}"

    # Step 4: 发送消息
    send_message "$CONVERSATION_ID_1" || echo -e "${YELLOW}⚠️  消息发送跳过${NC}"
fi

echo -e "${GREEN}✅ 正常顺序测试完成${NC}"
echo ""
echo ""

sleep 2

# ====================
# 测试 2: 异常顺序（编译 SDK）
# ====================
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  测试 2: 异常顺序 (编译 SDK)          ║${NC}"
echo -e "${BLUE}║  create → workflows → list → chat     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Step 1: 创建会话（先创建）
CONVERSATION_ID_2=$(create_conversation)
echo ""
if [ -z "$CONVERSATION_ID_2" ]; then
    echo -e "${RED}❌ 无法继续测试，会话创建失败${NC}"
else
    # Step 2: 获取 workflow（后获取）
    get_workflow_info

    # Step 3: 获取消息列表
    get_message_list "$CONVERSATION_ID_2" || echo -e "${YELLOW}⚠️  消息列表获取跳过${NC}"

    # Step 4: 发送消息
    send_message "$CONVERSATION_ID_2" || echo -e "${YELLOW}⚠️  消息发送跳过${NC}"
fi

echo -e "${GREEN}✅ 异常顺序测试完成${NC}"
echo ""
echo ""

# ====================
# 结果对比
# ====================
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  测试结果总结${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "正常顺序 Conversation ID: ${GREEN}${CONVERSATION_ID_1}${NC}"
echo -e "异常顺序 Conversation ID: ${GREEN}${CONVERSATION_ID_2}${NC}"
echo ""
echo -e "${YELLOW}💡 关键观察点：${NC}"
echo "1. 两种顺序的消息发送是否都成功？"
echo "2. 如果异常顺序失败，错误信息是什么？"
echo "3. conversation_id 是否正确返回并使用？"
echo ""

