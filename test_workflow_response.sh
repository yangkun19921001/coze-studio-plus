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
API_KEY="pat_9c5877dc08a472cdea1ac0429441833aec9d8fc92e06a8ef4aac542ac1d18c5e"
BASE_URL="http://localhost:8888"

echo "测试 1: 今天星期几"
./start_api_test.sh $WORKFLOW_ID "今天星期几" | jq -r '.data'
echo ""
echo "---"
echo ""

echo "测试 2: 新能源车推荐"
./start_api_test.sh $WORKFLOW_ID "新能源车推荐" | jq -r '.data'
echo ""
echo "---"
echo ""

echo "测试 3: 什么是AI"
./start_api_test.sh $WORKFLOW_ID "什么是AI" | jq -r '.data'
