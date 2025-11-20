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
# 快速测试 Chat API - 一键运行
#


# 本地运行的传入的 json 
curl -N -X POST https://iaas-ops-agent.pyinfra.work/v1/workflows/chat \
  -H "Authorization: Bearer pat_4be7a56bc8c6450b6e5d91e0534108cf2c13ca496423a04fd5f03052d0f306a9" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "additional_messages": [{
      "role": "user",
      "content_type": "text",
      "content": "你好"
    }],
    "connector_id": "10000010",
    "workflow_id": "7574700179186515968",
    "execute_mode": "DEBUG",
    "app_id": "7574699996000288768",
    "conversation_id": "123",
    "ext": {"_caller": "CANVAS"}
  }'

# sdk 传入的 json
  curl -N -X POST https://iaas-ops-agent.pyinfra.work/v1/workflows/chat \
  -H "Authorization: Bearer pat_4be7a56bc8c6450b6e5d91e0534108cf2c13ca496423a04fd5f03052d0f306a9" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"additional_messages":[{"role":"user","content_type":"text","content":"1"}],"connector_id":"999","workflow_id":"7574700179186515968","parameters":{},"app_id":"7574699996000288768","conversation_id":"7575170106058080256","ext":{"user_id":"12334"},"suggest_reply_info":{"suggest_reply_mode":null,"customized_suggest_prompt":null}}'
