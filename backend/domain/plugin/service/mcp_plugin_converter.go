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

package service

import (
	"encoding/json"
	"fmt"

	"github.com/coze-dev/coze-studio/backend/domain/plugin/dto"
	"github.com/coze-dev/coze-studio/backend/pkg/mcp"
)

// ConvertCursorMcpConfigToInternal converts cursor MCP config format to internal format
// Cursor format: { "mcpServers": { "server-name": { "url": "..." } } }
// Internal format: { "transport_type": "sse", "sse_config": { "url": "..." } }
func ConvertCursorMcpConfigToInternal(cursorConfig json.RawMessage) ([]*mcp.Config, error) {
	var cursorFormat dto.McpServersConfig
	if err := json.Unmarshal(cursorConfig, &cursorFormat); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor config: %w", err)
	}

	configs := make([]*mcp.Config, 0, len(cursorFormat.McpServers))
	for serverName, serverConfig := range cursorFormat.McpServers {
		var config *mcp.Config

		// Determine transport type: use transport_type field if present, otherwise infer from fields
		transportType := serverConfig.TransportType
		if transportType == "" {
			// Infer transport type from fields
			if serverConfig.URL != "" {
				transportType = "sse"
			} else if serverConfig.GetCommandArray() != nil && len(serverConfig.GetCommandArray()) > 0 {
				transportType = "stdio"
			}
		}

		// Create config based on transport type
		switch transportType {
		case "sse":
			if serverConfig.URL == "" {
				return nil, fmt.Errorf("invalid MCP server config for '%s': SSE transport requires 'url' field", serverName)
			}
			config = &mcp.Config{
				ServerName:    serverName,
				TransportType: mcp.TransportTypeSSE,
				SSEConfig: &mcp.SSEConfig{
					URL:     serverConfig.URL,
					Headers: serverConfig.Headers,
					APIKey:  serverConfig.APIKey,
				},
			}
		case "stdio":
			commandArray := serverConfig.GetCommandArray()
			if len(commandArray) == 0 {
				return nil, fmt.Errorf("invalid MCP server config for '%s': stdio transport requires 'command' field", serverName)
			}
			config = &mcp.Config{
				ServerName:    serverName,
				TransportType: mcp.TransportTypeStdio,
				StdioConfig: &mcp.StdioConfig{
					Command:    commandArray,
					Env:        serverConfig.Env,
					WorkingDir: serverConfig.WorkingDir,
				},
			}
		default:
			return nil, fmt.Errorf("invalid MCP server config for '%s': must have either 'url' (SSE) or 'command' (stdio), or specify 'transport_type'", serverName)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// ConvertInternalMcpConfigToCursor converts internal MCP config format to cursor format
func ConvertInternalMcpConfigToCursor(configs []*mcp.Config) (json.RawMessage, error) {
	mcpServers := make(map[string]dto.McpServerConfig)

	for i, config := range configs {
		serverName := fmt.Sprintf("mcp-server-%d", i)
		var serverConfig dto.McpServerConfig

		switch config.TransportType {
		case mcp.TransportTypeSSE:
			if config.SSEConfig != nil {
				serverConfig.URL = config.SSEConfig.URL
				serverConfig.Headers = config.SSEConfig.Headers
				serverConfig.APIKey = config.SSEConfig.APIKey
			}
		case mcp.TransportTypeStdio:
			if config.StdioConfig != nil {
				serverConfig.Command = config.StdioConfig.Command
				serverConfig.Env = config.StdioConfig.Env
				serverConfig.WorkingDir = config.StdioConfig.WorkingDir
			}
		}

		mcpServers[serverName] = serverConfig
	}

	cursorFormat := dto.McpServersConfig{
		McpServers: mcpServers,
	}

	return json.Marshal(cursorFormat)
}
