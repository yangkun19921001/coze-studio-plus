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

package dto

import (
	"encoding/json"
)

// McpServerConfig represents a single MCP server configuration
// Compatible with cursor's MCP configuration format
type McpServerConfig struct {
	URL           string            `json:"url,omitempty"`            // For SSE transport
	TransportType string            `json:"transport_type,omitempty"` // Transport type: "sse" or "stdio"
	Command       interface{}       `json:"command,omitempty"`        // For stdio transport: can be string or []string
	Args          []string          `json:"args,omitempty"`           // Command arguments (used when command is string)
	Env           map[string]string `json:"env,omitempty"`            // For stdio transport
	WorkingDir    string            `json:"working_dir,omitempty"`    // For stdio transport
	Headers       map[string]string `json:"headers,omitempty"`        // For SSE transport
	APIKey        string            `json:"api_key,omitempty"`        // For SSE transport
}

// GetCommandArray converts Command to []string, handling both string and []string cases
func (m *McpServerConfig) GetCommandArray() []string {
	if m.Command == nil {
		return nil
	}

	switch v := m.Command.(type) {
	case string:
		// If command is a string and args exist, combine them
		if len(m.Args) > 0 {
			result := []string{v}
			result = append(result, m.Args...)
			return result
		}
		return []string{v}
	case []string:
		return v
	case []interface{}:
		// Handle case where JSON unmarshals to []interface{}
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// McpServersConfig represents the MCP servers configuration
// Format: { "mcpServers": { "server-name": { "url": "..." } } }
type McpServersConfig struct {
	McpServers map[string]McpServerConfig `json:"mcpServers"`
}

// CreateMcpPluginRequest request to create MCP plugin
type CreateMcpPluginRequest struct {
	Name      string          `json:"name" validate:"required"`
	McpConfig json.RawMessage `json:"mcp_config" validate:"required"`
}

// UpdateMcpPluginRequest request to update MCP plugin
type UpdateMcpPluginRequest struct {
	ID        int64           `json:"id" validate:"required"`
	Name      string          `json:"name" validate:"required"`
	McpConfig json.RawMessage `json:"mcp_config" validate:"required"`
}

// ListMcpPluginsRequest request to list MCP plugins
type ListMcpPluginsRequest struct {
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
}

// ListMcpPluginsResponse response for listing MCP plugins
type ListMcpPluginsResponse struct {
	Plugins []*McpPluginInfo `json:"plugins"`
	Total   int64            `json:"total"`
}

// McpPluginInfo MCP plugin information
type McpPluginInfo struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	PluginID  int64           `json:"plugin_id"`
	UserID    int64           `json:"user_id"`
	McpConfig json.RawMessage `json:"mcp_config"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}
