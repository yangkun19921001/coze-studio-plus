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


WORKFLOW_ID="7569925483370905600"
SPACE_ID="7547331761605181440"
API_URL="http://localhost:8888"

echo "==================================="
echo "导出工作流配置"
echo "==================================="
echo ""

# 使用内部 API 获取工作流配置（需要 session_key，但我们可以先尝试）
curl -s "${API_URL}/api/workflow_api/canvas" \
  -H "Content-Type: application/json" \
  -d "{\"space_id\":\"${SPACE_ID}\",\"workflow_id\":\"${WORKFLOW_ID}\"}" | jq '.data.workflow.schema_json' -r > /tmp/workflow_schema.json 2>/dev/null

if [ -f /tmp/workflow_schema.json ] && [ -s /tmp/workflow_schema.json ]; then
    echo "✅ 工作流配置已导出到 /tmp/workflow_schema.json"
    echo ""
    
    # 解析并查找 fcParam
    echo "🔍 分析 Function Calling 配置..."
    cat /tmp/workflow_schema.json | jq '.' > /tmp/workflow_formatted.json 2>/dev/null
    
    # 查找 LLM 节点的 fcParam
    FC_PARAM=$(cat /tmp/workflow_formatted.json | jq '.nodes[] | select(.type == "3") | .data.inputs.fcParam' 2>/dev/null)
    
    if [ "$FC_PARAM" != "null" ] && [ -n "$FC_PARAM" ]; then
        echo "✅ 找到 fcParam 配置"
        echo ""
        echo "$FC_PARAM" > /tmp/fc_param.json
        
        # 查找插件配置
        PLUGIN_CONFIG=$(cat /tmp/fc_param.json | jq '.plugin_list[]? | select(.plugin_id == "6")' 2>/dev/null)
        
        if [ -n "$PLUGIN_CONFIG" ] && [ "$PLUGIN_CONFIG" != "null" ]; then
            echo "📋 插件配置详情:"
            echo "$PLUGIN_CONFIG" | jq '.'
            echo ""
            
            # 检查是否有 fc_setting
            FC_SETTING=$(echo "$PLUGIN_CONFIG" | jq '.fc_setting' 2>/dev/null)
            
            if [ "$FC_SETTING" != "null" ] && [ -n "$FC_SETTING" ]; then
                echo "⚠️  发现 fc_setting 配置!"
                echo ""
                
                # 检查 response_params
                RESPONSE_PARAMS=$(echo "$FC_SETTING" | jq '.response_params' 2>/dev/null)
                
                if [ "$RESPONSE_PARAMS" != "null" ] && [ "$RESPONSE_PARAMS" != "[]" ] && [ -n "$RESPONSE_PARAMS" ]; then
                    echo "❌ 发现响应参数配置（这是问题所在）:"
                    echo "$RESPONSE_PARAMS" | jq '.'
                    echo ""
                    echo "💡 解决方案：需要手动删除这个配置"
                else
                    echo "✅ 没有响应参数配置（response_params 为空）"
                fi
            else
                echo "✅ 没有 fc_setting 配置（这是正确的）"
            fi
        else
            echo "❌ 未找到 plugin_id=6 的配置"
        fi
    else
        echo "❌ 未找到 fcParam 配置"
    fi
    
    echo ""
    echo "📄 完整的格式化 JSON 已保存到: /tmp/workflow_formatted.json"
    echo "📄 fcParam 配置已保存到: /tmp/fc_param.json"
else
    echo "❌ 无法导出工作流配置"
    echo "ℹ️  您可以在 Coze Studio UI 中手动导出工作流 JSON"
fi
