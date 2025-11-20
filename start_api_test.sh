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


# 🔑 API 配置
# API_KEY="pat_9c5877dc08a472cdea1ac0429441833aec9d8fc92e06a8ef4aac542ac1d18c5e"
# BASE_URL="http://localhost:8888"
# WORKFLOW_ID="7569925483370905600"

API_KEY="pat_4be7a56bc8c6450b6e5d91e0534108cf2c13ca496423a04fd5f03052d0f306a9"
BASE_URL="https://iaas-ops-agent.pyinfra.work"
WORKFLOW_ID="7574700179186515968"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 快速模式：通过命令行参数直接测试
# 用法: ./start_api_test.sh <workflow_id> <text>
if [ $# -eq 2 ]; then
    workflow_id=$WORKFLOW_ID
    user_text=$2
    
    echo "🚀 快速测试模式"
    echo "Workflow ID: $workflow_id"
    echo "输入内容: $user_text"
    echo ""
    
    if command -v jq &> /dev/null; then
        request_body=$(jq -n \
            --arg wid "$workflow_id" \
            --arg text "$user_text" \
            '{workflow_id: $wid, parameters: {USER_INPUT: $text}}')
        
        echo "$request_body"

        
        curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "$request_body" | jq '.'

    else
        user_text_escaped=$(echo "$user_text" | sed 's/"/\\"/g')
        curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"workflow_id":"'$workflow_id'","parameters":{"USER_INPUT":"'$user_text_escaped'"}}'

        echo '{"workflow_id":"'$workflow_id'","parameters":{"USER_INPUT":"'$user_text_escaped'"}}'
    fi
    
    exit 0
fi

# curl -s -X POST "http://localhost:8888/v1/workflow/stream_run/v1/workflow/run" \
#     -H "Authorization: Bearer pat_9c5877dc08a472cdea1ac0429441833aec9d8fc92e06a8ef4aac542ac1d18c5e" \
#     -H "Content-Type: application/json" \
#     -d '{"workflow_id":"7569925483370905600","parameters":{"input":"帮我看下这台设备 55fc63b4e88da07c191b1642f86fe05d CPU多少核?"}}'


clear
echo "🧪 Coze Studio OpenAPI 测试工具"
echo "==========================================="
echo ""
echo "📝 API Key: ${API_KEY:0:20}...${API_KEY: -10}"
echo "🌐 Base URL: $BASE_URL"
echo "🔑 认证方式: Bearer Token (OpenAPI)"
echo ""

# 检查后端服务是否运行（使用 OpenAPI 接口测试）
test_response=$(curl -s -X POST "$BASE_URL/v1/workflow/run" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"workflow_id":"test"}' 2>&1)

if echo "$test_response" | grep -q "Connection refused\|Could not resolve host"; then
    echo -e "${RED}❌ 错误：后端服务未运行！${NC}"
    echo "   请先启动后端服务："
    echo "   1. 运行: ./start_vs_debug.sh"
    echo "   2. 或在 VS Code 中按 F5"
    echo ""
    exit 1
fi

echo -e "${GREEN}✅ 后端服务运行中${NC}"
echo ""
echo "💡 重要提示："
echo "   1. 先选择「0️⃣ 列出所有工作流」获取正确的 workflow ID"
echo "   2. 或启动前端创建工作流: ./start_debug_web.sh"
echo "   3. 快速命令: ./start_api_test.sh <workflow_id> <文本内容>"
echo ""

# 菜单选项
show_menu() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  📋 OpenAPI 测试菜单"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "  0️⃣  列出所有工作流（必看！）"
    echo "  1️⃣  运行工作流（同步）"
    echo "  2️⃣  运行工作流（异步）"
    echo "  3️⃣  运行工作流（流式响应）⭐ 新"
    echo "  4️⃣  获取工作流信息"
    echo "  5️⃣  快速测试（简化版）"
    echo "  6️⃣  查看使用说明"
    echo "  7️⃣  聊天模式（Chat API）🔥 推荐"
    echo "  9️⃣  退出"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    read -p "👉 请输入选项 [0-7,9]: " choice
    echo ""
}

# 0. 列出所有工作流
list_workflows() {
    echo -e "${YELLOW}📋 列出所有工作流${NC}"
    echo ""
    echo "⚠️  注意：OpenAPI 不支持列出工作流"
    echo ""
    echo "📝 获取 Workflow ID 的两种方法："
    echo ""
    echo "方法 1: 通过 Web 界面（推荐）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  1. 启动前端: ./start_debug_web.sh"
    echo "  2. 访问: http://localhost:5001"
    echo "  3. 创建或打开一个工作流"
    echo "  4. 查看浏览器 URL"
    echo "  5. 例如: http://localhost:5001/workflow/123456"
    echo "     其中 123456 就是你的 workflow_id"
    echo ""
    echo "方法 2: 使用内部 API（需要登录）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  步骤："
    echo "  1. 在浏览器中访问 http://localhost:5001 并登录"
    echo "  2. 按 F12 打开开发者工具"
    echo "  3. 在 Console 中输入: document.cookie"
    echo "  4. 复制 session_key 的值"
    echo ""
    read -p "已获取 session_key？输入它来查看工作流列表（直接回车跳过）: " session_key
    echo ""
    
    if [ -n "$session_key" ]; then
        echo "🔍 查询中..."
        echo ""
        
        response=$(curl -s "$BASE_URL/api/workflow_api/list" \
            -H "Cookie: session_key=$session_key" \
            -H "Content-Type: application/json" \
            -d '{"page_num":1,"page_size":50}')
        
        if command -v jq &> /dev/null; then
            # 美化输出
            echo "$response" | jq '.'
            echo ""
            
            # 提取工作流 ID 列表
            workflow_ids=$(echo "$response" | jq -r '.data.items[]? | "\(.id)\t\(.name)"' 2>/dev/null)
            
            if [ -n "$workflow_ids" ]; then
                echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo "📌 可用的工作流："
                echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo "$workflow_ids" | while IFS=$'\t' read -r id name; do
                    echo "  ID: $id"
                    echo "  名称: $name"
                    echo ""
                done
            else
                echo -e "${YELLOW}⚠️  未找到工作流，请先在 Web 界面创建${NC}"
            fi
        else
            echo "$response"
            echo ""
            echo "💡 安装 jq 获得更好的显示: brew install jq"
        fi
    else
        echo -e "${BLUE}💡 快速开始：${NC}"
        echo "   1. 启动前端: ./start_debug_web.sh"
        echo "   2. 打开浏览器访问: http://localhost:5001"
        echo "   3. 创建一个简单的工作流（开始 -> LLM -> 结束）"
        echo "   4. 点击右上角「发布」"
        echo "   5. 从 URL 获取 workflow ID 来测试"
    fi
    
    echo ""
}

# 1. 运行工作流（同步）
test_run_workflow_sync() {
    echo "▶️  运行工作流（同步模式）..."
    echo ""
    
    read -p "请输入 Workflow ID: " workflow_id
    echo ""
    echo "💡 参数输入方式："
    echo "   1) 直接输入文本（推荐，自动作为 user_input）"
    echo "   2) 输入完整 JSON（高级用户）"
    echo ""
    read -p "选择方式 [1/2，默认1]: " input_mode
    input_mode=${input_mode:-1}
    
    echo ""
    if [ "$input_mode" = "1" ]; then
        read -p "请输入内容: " user_text
        
        # 使用 jq 构建 JSON（自动转义）
        if command -v jq &> /dev/null; then
            parameters=$(jq -n --arg text "$user_text" '{input: $text}')
        else
            # 简单转义（可能不完美）
            user_text_escaped=$(echo "$user_text" | sed 's/"/\\"/g' | sed "s/'/\\'/g")
            parameters="{\"input\":\"$user_text_escaped\"}"
        fi
    else
        read -p "请输入 JSON 参数: " parameters
        if [ -z "$parameters" ]; then
            parameters="{}"
        fi
    fi
    
    echo ""
    echo "🚀 执行中..."
    echo ""
    
    # 构建请求
    if command -v jq &> /dev/null; then
        request_body=$(jq -n \
            --arg wid "$workflow_id" \
            --argjson params "$parameters" \
            '{workflow_id: $wid, parameters: $params}')
        
        curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "$request_body" | jq '.'
    else
        curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "{\"workflow_id\":\"$workflow_id\",\"parameters\":$parameters}"
        echo ""
        echo "💡 建议安装 jq 工具: brew install jq"
    fi
    
    echo ""
}

# 2. 运行工作流（异步）
test_run_workflow_async() {
    echo -e "${BLUE}▶️  运行工作流（异步模式）${NC}"
    echo ""
    
    read -p "请输入 Workflow ID: " workflow_id
    echo ""
    echo "💡 参数输入方式："
    echo "   1) 直接输入文本（推荐，自动作为 user_input）"
    echo "   2) 输入完整 JSON（高级用户）"
    echo ""
    read -p "选择方式 [1/2，默认1]: " input_mode
    input_mode=${input_mode:-1}
    
    echo ""
    if [ "$input_mode" = "1" ]; then
        read -p "请输入内容: " user_text
        
        # 使用 jq 构建 JSON（自动转义）
        if command -v jq &> /dev/null; then
            parameters=$(jq -n --arg text "$user_text" '{input: $text}')
        else
            user_text_escaped=$(echo "$user_text" | sed 's/"/\\"/g' | sed "s/'/\\'/g")
            parameters="{\"input\":\"$user_text_escaped\"}"
        fi
    else
        read -p "请输入 JSON 参数: " parameters
        if [ -z "$parameters" ]; then
            parameters="{}"
        fi
    fi
    
    echo ""
    echo "🚀 提交异步任务..."
    echo ""
    
    # 使用 jq 来构建正确的 JSON
    if command -v jq &> /dev/null; then
        request_body=$(jq -n \
            --arg wid "$workflow_id" \
            --argjson params "$parameters" \
            '{workflow_id: $wid, parameters: $params, is_async: true}')
        
        response=$(curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "$request_body")
    else
        response=$(curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "{\"workflow_id\":\"$workflow_id\",\"parameters\":$parameters,\"is_async\":true}")
    fi
    
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
    
    # 提取 execute_id
    execute_id=$(echo "$response" | jq -r '.data.execute_id' 2>/dev/null)
    if [ "$execute_id" != "null" ] && [ -n "$execute_id" ]; then
        echo ""
        echo -e "${GREEN}✅ 任务已提交，Execute ID: $execute_id${NC}"
        echo ""
        read -p "是否查询执行结果？[y/N] " check_result
        if [[ $check_result =~ ^[Yy]$ ]]; then
            check_execution_history "$execute_id"
        fi
    fi
    
    echo ""
}

# 3. 运行工作流（流式响应）
test_run_workflow_stream() {
    echo -e "${GREEN}▶️  运行工作流（流式响应 SSE）${NC}"
    echo ""
    
    read -p "请输入 Workflow ID: " workflow_id
    echo ""
    echo "💡 参数输入方式："
    echo "   1) 直接输入文本（推荐，自动作为 input）"
    echo "   2) 输入完整 JSON（高级用户）"
    echo ""
    read -p "选择方式 [1/2，默认1]: " input_mode
    input_mode=${input_mode:-1}
    
    echo ""
    if [ "$input_mode" = "1" ]; then
        read -p "请输入内容: " user_text
        
        # 使用 jq 构建 JSON（自动转义）
        if command -v jq &> /dev/null; then
            parameters=$(jq -n --arg text "$user_text" '{input: $text}')
        else
            user_text_escaped=$(echo "$user_text" | sed 's/"/\\"/g' | sed "s/'/\\'/g")
            parameters="{\"input\":\"$user_text_escaped\"}"
        fi
    else
        read -p "请输入 JSON 参数: " parameters
        if [ -z "$parameters" ]; then
            parameters="{}"
        fi
    fi
    
    echo ""
    echo "🚀 开始流式执行..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # 构建请求
    if command -v jq &> /dev/null; then
        request_body=$(jq -n \
            --arg wid "$workflow_id" \
            --argjson params "$parameters" \
            '{workflow_id: $wid, parameters: $params}')
    else
        request_body="{\"workflow_id\":\"$workflow_id\",\"parameters\":$parameters}"
    fi
    
    # 使用 curl 的 -N 参数禁用缓冲，实时显示 SSE 流
    # 使用 --no-buffer 确保立即输出
    echo -e "${BLUE}📡 接收流式数据...${NC}"
    echo ""
    
    # 临时文件存储原始 SSE 数据
    temp_file=$(mktemp)
    
    curl -N -X POST "$BASE_URL/v1/workflow/stream_run" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -H "Accept: text/event-stream" \
        -d "$request_body" 2>/dev/null | while IFS= read -r line; do
        
        # 保存原始行
        echo "$line" >> "$temp_file"
        
        # 解析 SSE 格式：event: xxx 和 data: xxx
        if [[ $line == event:* ]]; then
            event_type=$(echo "$line" | sed 's/^event: *//')
            echo -e "${YELLOW}📌 事件: $event_type${NC}"
        elif [[ $line == data:* ]]; then
            data_content=$(echo "$line" | sed 's/^data: *//')
            
            # 尝试格式化 JSON
            if command -v jq &> /dev/null && [ "$data_content" != "[DONE]" ]; then
                echo "$data_content" | jq '.' 2>/dev/null || echo "$data_content"
            else
                echo "$data_content"
            fi
            echo ""
        elif [ -z "$line" ]; then
            # SSE 的空行分隔符
            echo "---"
        fi
    done
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${GREEN}✅ 流式响应完成${NC}"
    echo ""
    echo "💾 完整原始数据已保存到: $temp_file"
    echo ""
    read -p "是否查看完整原始 SSE 数据？[y/N] " view_raw
    if [[ $view_raw =~ ^[Yy]$ ]]; then
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "📄 完整 SSE 原始数据："
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        cat "$temp_file"
        echo ""
    fi
    
    echo ""
}

# 4. 查询执行结果
check_execution_history() {
    local execute_id=$1
    
    if [ -z "$execute_id" ]; then
        read -p "请输入 Execute ID: " execute_id
    fi
    
    echo -e "${BLUE}🔍 查询执行结果（Execute ID: $execute_id）${NC}"
    echo ""
    
    curl -X GET "$BASE_URL/v1/workflow/execute/$execute_id" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" | jq '.' 2>/dev/null || echo "请安装 jq 工具以格式化 JSON 输出"
    
    echo ""
}

# 5. 获取执行历史
test_execution_history() {
    echo -e "${BLUE}📊 获取执行历史${NC}"
    echo ""
    
    read -p "请输入 Workflow ID（可选，直接回车查看所有）: " workflow_id
    
    echo ""
    echo "🔍 查询中..."
    echo ""
    
    if [ -n "$workflow_id" ]; then
        curl -X POST "$BASE_URL/v1/workflow/executions" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "{\"workflow_id\": \"$workflow_id\", \"page_num\": 1, \"page_size\": 10}" | jq '.' 2>/dev/null || echo "请安装 jq 工具以格式化 JSON 输出"
    else
        curl -X POST "$BASE_URL/v1/workflow/executions" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"page_num": 1, "page_size": 10}' | jq '.' 2>/dev/null || echo "请安装 jq 工具以格式化 JSON 输出"
    fi
    
    echo ""
}

# 6. 快速测试（简化版）
test_quick() {
    echo -e "${GREEN}⚡ 快速测试${NC}"
    echo ""
    
    read -p "请输入 Workflow ID: " workflow_id
    
    if [ -z "$workflow_id" ]; then
        echo -e "${RED}❌ Workflow ID 不能为空${NC}"
        return
    fi
    
    read -p "请输入测试内容（直接回车使用默认: 你好，请介绍一下自己）: " test_text
    
    # 如果用户没输入，使用默认
    if [ -z "$test_text" ]; then
        test_text="你好，请介绍一下自己"
    fi
    
    echo ""
    echo "🚀 执行测试工作流..."
    echo "📝 发送内容: $test_text"
    echo ""
    
    # 使用 jq 构建正确的 JSON
    if command -v jq &> /dev/null; then
        request_body=$(jq -n \
            --arg wid "$workflow_id" \
            --arg text "$test_text" \
            '{workflow_id: $wid, parameters: {user_input: $text}}')
        
        curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d "$request_body" | jq '.'
    else
        test_text_escaped=$(echo "$test_text" | sed 's/"/\\"/g')
        curl -s -X POST "$BASE_URL/v1/workflow/run" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"workflow_id":"'$workflow_id'","parameters":{"user_input":"'$test_text_escaped'"}}'
        echo ""
    fi
    
    echo ""
}

# 7. 聊天模式（Chat API）
test_chat_api() {
    echo -e "${GREEN}💬 聊天模式（Chat API）${NC}"
    echo ""
    
    read -p "请输入 Workflow ID: " workflow_id
    
    if [ -z "$workflow_id" ]; then
        echo -e "${RED}❌ Workflow ID 不能为空${NC}"
        return
    fi
    
    read -p "请输入 App ID（必需）: " app_id
    
    if [ -z "$app_id" ]; then
        echo -e "${RED}❌ App ID 不能为空${NC}"
        return
    fi
    
    read -p "请输入 Conversation ID（可选，直接回车自动创建）: " conversation_id
    
    echo ""
    echo "🎯 进入聊天模式（输入 'exit' 或 'quit' 退出）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    while true; do
        read -p "👤 你: " user_message
        
        if [ -z "$user_message" ]; then
            continue
        fi
        
        if [[ "$user_message" == "exit" ]] || [[ "$user_message" == "quit" ]]; then
            echo -e "${YELLOW}👋 退出聊天模式${NC}"
            break
        fi
        
        echo ""
        echo -e "${BLUE}🤖 AI:${NC} "
        
        # 构建 additional_messages JSON（根据真实 API 格式）
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
        else
            # 简单转义（不完美但足够用）
            user_message_escaped=$(echo "$user_message" | sed 's/"/\\"/g' | sed "s/'/\\'/g")
            if [ -n "$conversation_id" ]; then
                request_body="{\"additional_messages\":[{\"role\":\"user\",\"content_type\":\"text\",\"content\":\"$user_message_escaped\"}],\"connector_id\":\"10000010\",\"workflow_id\":\"$workflow_id\",\"execute_mode\":\"DEBUG\",\"app_id\":\"$app_id\",\"conversation_id\":\"$conversation_id\",\"ext\":{\"_caller\":\"CANVAS\"}}"
            else
                request_body="{\"additional_messages\":[{\"role\":\"user\",\"content_type\":\"text\",\"content\":\"$user_message_escaped\"}],\"connector_id\":\"10000010\",\"workflow_id\":\"$workflow_id\",\"execute_mode\":\"DEBUG\",\"app_id\":\"$app_id\",\"ext\":{\"_caller\":\"CANVAS\"}}"
            fi
        fi
        
        # 临时文件存储响应
        temp_file=$(mktemp)
        response_text=""
        
        # 发送请求并解析流式响应
        curl -N -s -X POST "$BASE_URL/v1/workflows/chat" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -H "Accept: text/event-stream" \
            -d "$request_body" 2>/dev/null | while IFS= read -r line; do
            
            # 保存原始行
            echo "$line" >> "$temp_file"
            
            # 解析 SSE 格式
            if [[ $line == data:* ]]; then
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
                else
                    # 如果没有 jq，直接输出原始数据
                    echo "$data_content"
                fi
            fi
        done
        
        echo ""
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
    done
    
    echo ""
}

# 8. 查看使用说明
show_help() {
    clear
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  📖 Coze Studio OpenAPI 使用说明${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "🔑 认证方式："
    echo "   使用 Bearer Token（Personal Access Token）进行认证"
    echo "   格式: Authorization: Bearer YOUR_API_KEY"
    echo ""
    echo "📋 API 端点说明："
    echo ""
    echo "   1. 运行工作流（同步）"
    echo "      POST /v1/workflow/run"
    echo "      参数: workflow_id, parameters"
    echo ""
    echo "   2. 运行工作流（异步）"
    echo "      POST /v1/workflow/run"
    echo "      参数: workflow_id, parameters, is_async: true"
    echo ""
    echo "   3. 运行工作流（流式响应）⭐"
    echo "      POST /v1/workflow/stream_run"
    echo "      参数: workflow_id, parameters"
    echo "      响应: Server-Sent Events (SSE) 流式数据"
    echo ""
    echo "   4. 聊天模式（Chat API）🔥"
    echo "      POST /v1/workflows/chat"
    echo "      必需参数: workflow_id, app_id, additional_messages"
    echo "      可选参数: conversation_id, connector_id, execute_mode, ext"
    echo "      响应: Server-Sent Events (SSE) 流式对话"
    echo "      特点: 支持多轮对话，保持上下文"
    echo ""
    echo "   5. 查询执行结果"
    echo "      GET /v1/workflow/execute/{execute_id}"
    echo ""
    echo "   5. 获取执行历史"
    echo "      POST /v1/workflow/executions"
    echo "      参数: workflow_id(可选), page_num, page_size"
    echo ""
    echo "💡 如何获取 Workflow ID："
    echo "   1. 启动前端: ./start_debug_web.sh"
    echo "   2. 访问 Web 界面: http://localhost:5001"
    echo "   3. 创建或打开工作流"
    echo "   4. 在 URL 中查看 ID"
    echo "   5. 例如: http://localhost:5001/workflow/123"
    echo "      其中 123 就是 workflow_id"
    echo ""
    echo "✨ 本工具的两种输入方式："
    echo ""
    echo "   方式1: 简单模式（推荐）"
    echo "   ├─ 直接输入文本内容"
    echo "   └─ 自动构建为: {\"input\": \"你的内容\"}"
    echo ""
    echo "   方式2: 高级模式"
    echo "   ├─ 手动输入完整 JSON"
    echo "   └─ 例如: {\"question\": \"什么是AI?\", \"lang\": \"zh\"}"
    echo ""
    echo "🔧 参数格式示例（仅高级模式需要）："
    echo "   {\"input\": \"你好\"}"
    echo "   {\"question\": \"什么是AI?\"}"
    echo "   {\"text\": \"要处理的文本\", \"max_length\": \"100\"}"
    echo ""
    echo "📚 第三方调用示例（curl - 同步）："
    echo ""
    echo "   curl -X POST $BASE_URL/v1/workflow/run \\"
    echo "     -H 'Authorization: Bearer YOUR_API_KEY' \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -d '{"
    echo "       \"workflow_id\": \"123\","
    echo "       \"parameters\": {"
    echo "         \"input\": \"你好\""
    echo "       }"
    echo "     }'"
    echo ""
    echo "📡 第三方调用示例（curl - 流式）："
    echo ""
    echo "   curl -N -X POST $BASE_URL/v1/workflow/stream_run \\"
    echo "     -H 'Authorization: Bearer YOUR_API_KEY' \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -H 'Accept: text/event-stream' \\"
    echo "     -d '{"
    echo "       \"workflow_id\": \"123\","
    echo "       \"parameters\": {"
    echo "         \"input\": \"你好\""
    echo "       }"
    echo "     }'"
    echo ""
    echo "   注意：-N 参数禁用缓冲，实时显示流式数据"
    echo ""
    echo "💬 第三方调用示例（curl - Chat API）："
    echo ""
    echo "   curl -N -X POST $BASE_URL/v1/workflows/chat \\"
    echo "     -H 'Authorization: Bearer YOUR_API_KEY' \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -H 'Accept: text/event-stream' \\"
    echo "     -d '{"
    echo "       \"additional_messages\": ["
    echo "         {"
    echo "           \"role\": \"user\","
    echo "           \"content_type\": \"text\","
    echo "           \"content\": \"你好\""
    echo "         }"
    echo "       ],"
    echo "       \"connector_id\": \"10000010\","
    echo "       \"workflow_id\": \"123\","
    echo "       \"execute_mode\": \"DEBUG\","
    echo "       \"app_id\": \"456\","
    echo "       \"conversation_id\": \"789\","
    echo "       \"ext\": {"
    echo "         \"_caller\": \"CANVAS\""
    echo "       }"
    echo "     }'"
    echo ""
    echo "   注意："
    echo "   - app_id 是必需的"
    echo "   - conversation_id 用于保持多轮对话上下文"
    echo "   - connector_id 默认 10000010"
    echo "   - execute_mode: DEBUG 或 留空"
    echo ""
    echo "📱 第三方调用示例（Python - 同步）："
    echo ""
    echo "   import requests"
    echo ""
    echo "   url = '$BASE_URL/v1/workflow/run'"
    echo "   headers = {"
    echo "       'Authorization': 'Bearer YOUR_API_KEY',"
    echo "       'Content-Type': 'application/json'"
    echo "   }"
    echo "   data = {"
    echo "       'workflow_id': '123',"
    echo "       'parameters': {"
    echo "           'input': '你好'"
    echo "       }"
    echo "   }"
    echo "   response = requests.post(url, headers=headers, json=data)"
    echo "   print(response.json())"
    echo ""
    echo "🐍 第三方调用示例（Python - 流式）："
    echo ""
    echo "   import requests"
    echo "   import json"
    echo ""
    echo "   url = '$BASE_URL/v1/workflow/stream_run'"
    echo "   headers = {"
    echo "       'Authorization': 'Bearer YOUR_API_KEY',"
    echo "       'Content-Type': 'application/json',"
    echo "       'Accept': 'text/event-stream'"
    echo "   }"
    echo "   data = {"
    echo "       'workflow_id': '123',"
    echo "       'parameters': {'input': '你好'}"
    echo "   }"
    echo "   "
    echo "   with requests.post(url, headers=headers, json=data, stream=True) as r:"
    echo "       for line in r.iter_lines():"
    echo "           if line:"
    echo "               decoded_line = line.decode('utf-8')"
    echo "               if decoded_line.startswith('data: '):"
    echo "                   data = decoded_line[6:]  # 去掉 'data: ' 前缀"
    echo "                   if data != '[DONE]':"
    echo "                       print(json.loads(data))"
    echo ""
    echo "💬 第三方调用示例（Python - Chat API）："
    echo ""
    echo "   import requests"
    echo "   import json"
    echo ""
    echo "   url = '$BASE_URL/v1/workflows/chat'"
    echo "   headers = {"
    echo "       'Authorization': 'Bearer YOUR_API_KEY',"
    echo "       'Content-Type': 'application/json',"
    echo "       'Accept': 'text/event-stream'"
    echo "   }"
    echo "   data = {"
    echo "       'additional_messages': ["
    echo "           {'role': 'user', 'content_type': 'text', 'content': '你好'}"
    echo "       ],"
    echo "       'connector_id': '10000010',"
    echo "       'workflow_id': '123',"
    echo "       'execute_mode': 'DEBUG',"
    echo "       'app_id': '456',"
    echo "       'conversation_id': '789',"
    echo "       'ext': {'_caller': 'CANVAS'}"
    echo "   }"
    echo "   "
    echo "   with requests.post(url, headers=headers, json=data, stream=True) as r:"
    echo "       for line in r.iter_lines():"
    echo "           if line:"
    echo "               decoded_line = line.decode('utf-8')"
    echo "               if decoded_line.startswith('data: '):"
    echo "                   data = json.loads(decoded_line[6:])"
    echo "                   # 提取消息内容"
    echo "                   if 'message' in data and 'content' in data['message']:"
    echo "                       print(data['message']['content'], end='')"
    echo ""
    echo "⚠️  注意："
    echo "   - parameters 是 JSON 对象，不是字符串"
    echo "   - 参数名取决于你的工作流入口节点定义"
    echo "   - 常见参数名: input, query, text, user_input 等"
    echo "   - 本工具默认使用 'input' 作为参数名"
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# 主循环
while true; do
    show_menu
    
    case $choice in
        0)
            list_workflows
            ;;
        1)
            test_run_workflow_sync
            ;;
        2)
            test_run_workflow_async
            ;;
        3)
            test_run_workflow_stream
            ;;
        4)
            check_execution_history ""
            ;;
        5)
            test_quick
            ;;
        6)
            show_help
            ;;
        7)
            test_chat_api
            ;;
        9)
            echo -e "${GREEN}👋 感谢使用！再见！${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ 无效选项，请重新选择${NC}"
            echo ""
            ;;
    esac
    
    read -p "按回车键继续..." dummy
    clear
done

