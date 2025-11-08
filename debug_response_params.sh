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


echo "==================================="
echo "响应参数配置诊断工具"
echo "==================================="
echo ""
echo "请导出您的工作流 JSON 文件，然后运行："
echo "  ./debug_response_params.sh <导出的json文件路径>"
echo ""

if [ -z "$1" ]; then
    echo "❌ 请提供工作流 JSON 文件路径"
    exit 1
fi

JSON_FILE="$1"

if [ ! -f "$JSON_FILE" ]; then
    echo "❌ 文件不存在: $JSON_FILE"
    exit 1
fi

echo "📋 分析文件: $JSON_FILE"
echo ""

# 查找 LLM 节点中的 fcParam
echo "🔍 查找 Function Calling 配置..."
jq '.nodes[] | select(.type == "3") | .data.inputs.fcParam' "$JSON_FILE" > /tmp/fc_param.json 2>/dev/null

if [ -s /tmp/fc_param.json ] && [ "$(cat /tmp/fc_param.json)" != "null" ]; then
    echo "✅ 找到 fcParam 配置"
    echo ""
    
    # 查找插件的响应参数配置
    echo "🔍 查找插件的响应参数配置..."
    jq '.plugin_list[]? | select(.plugin_id == "6") | .fc_setting.response_params' /tmp/fc_param.json > /tmp/response_params.json 2>/dev/null
    
    if [ -s /tmp/response_params.json ] && [ "$(cat /tmp/response_params.json)" != "null" ]; then
        echo "✅ 找到响应参数配置"
        echo ""
        echo "📝 响应参数配置详情:"
        jq '.' /tmp/response_params.json
        echo ""
        
        # 检查 list 数组的配置
        echo "🔍 检查 list 数组配置..."
        LIST_CONFIG=$(jq '.[] | select(.name == "data") | .sub_parameters[]? | select(.name == "coze_ark_001") | .sub_parameters[]? | select(.name == "list")' /tmp/response_params.json 2>/dev/null)
        
        if [ -n "$LIST_CONFIG" ] && [ "$LIST_CONFIG" != "null" ]; then
            echo "📋 List 数组配置:"
            echo "$LIST_CONFIG" | jq '.'
            echo ""
            
            # 检查 list 元素的子参数
            SUB_PARAMS=$(echo "$LIST_CONFIG" | jq '.sub_parameters')
            if [ "$SUB_PARAMS" != "null" ] && [ -n "$SUB_PARAMS" ]; then
                echo "✅ List 元素子参数:"
                echo "$SUB_PARAMS" | jq '.'
                
                # 检查是否有被禁用的参数
                DISABLED=$(echo "$SUB_PARAMS" | jq '[.[] | select(.local_disable == true or .global_disable == true) | .name]')
                if [ "$DISABLED" != "[]" ]; then
                    echo ""
                    echo "⚠️  以下字段被禁用: $DISABLED"
                fi
            else
                echo "❌ List 元素没有配置子参数（sub_parameters）！这可能是问题所在！"
            fi
        else
            echo "❌ 未找到 list 数组配置"
        fi
    else
        echo "ℹ️  没有配置响应参数（这是推荐的配置）"
    fi
else
    echo "❌ 未找到 fcParam 配置"
fi

rm -f /tmp/fc_param.json /tmp/response_params.json 2>/dev/null
