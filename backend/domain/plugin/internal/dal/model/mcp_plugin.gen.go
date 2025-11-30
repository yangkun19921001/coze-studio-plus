/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"encoding/json"
)

const TableNameMcpPlugin = "mcp_plugin"

// McpPlugin MCP Plugin Configuration
type McpPlugin struct {
	ID        int64           `gorm:"column:id;primaryKey;comment:Primary Key ID" json:"id"`
	Name      string          `gorm:"column:name;not null;comment:MCP Server Name" json:"name"`
	PluginID  int64           `gorm:"column:plugin_id;not null;comment:Plugin ID from plugin_meta.yaml, 0 if user-created" json:"plugin_id"`
	UserID    int64           `gorm:"column:user_id;not null;default:0;comment:User ID, 0 for system plugins" json:"user_id"`
	McpConfig json.RawMessage `gorm:"column:mcp_config;type:json;not null;comment:MCP Configuration (JSON format, compatible with SSE and stdio)" json:"mcp_config"`
	CreatedAt int64           `gorm:"column:created_at;not null;autoCreateTime:milli;comment:Create Time in Milliseconds" json:"created_at"`
	UpdatedAt int64           `gorm:"column:updated_at;not null;autoUpdateTime:milli;comment:Update Time in Milliseconds" json:"updated_at"`
}

// TableName McpPlugin's table name
func (*McpPlugin) TableName() string {
	return TableNameMcpPlugin
}
