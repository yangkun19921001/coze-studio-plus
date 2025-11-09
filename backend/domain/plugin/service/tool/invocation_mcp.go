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

package tool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/bytedance/sonic"

	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/mcp"
)

type mcpCallImpl struct {
	clients map[string]*mcp.Client // key: config hash
	mu      sync.RWMutex
}

func NewMcpCallImpl() Invocation {
	return &mcpCallImpl{
		clients: make(map[string]*mcp.Client),
	}
}

func (m *mcpCallImpl) Do(ctx context.Context, args *InvocationArgs) (request string, resp string, err error) {
	logs.Infof("[MCP] Do() called: tool_id=%d, tool_name=%s", args.Tool.ID, args.Tool.GetName())

	// 1. Parse MCP config from plugin manifest
	mcpConfig, err := m.parseMCPConfig(args)
	if err != nil {
		logs.Errorf("[MCP] Failed to parse config: %v", err)
		return "", "", fmt.Errorf("parse mcp config failed: %w", err)
	}

	// 2. Get or create MCP client
	client, err := m.getOrCreateClient(ctx, mcpConfig)
	if err != nil {
		return "", "", fmt.Errorf("get mcp client failed: %w", err)
	}

	// 3. Build tool call arguments
	toolName := args.Tool.GetName()
	if toolName == "" {
		toolName = fmt.Sprintf("tool_%d", args.Tool.ID)
	}

	// Merge all arguments
	arguments := make(map[string]any)
	for k, v := range args.Header {
		arguments[k] = v
	}
	for k, v := range args.Query {
		arguments[k] = v
	}
	for k, v := range args.Path {
		arguments[k] = v
	}
	for k, v := range args.Body {
		arguments[k] = v
	}

	logs.Infof("[MCP] Tool call name: %s, arguments: %v", toolName, arguments)

	// 4. Call tool via mcp-go library
	resultStr, err := client.CallTool(ctx, toolName, arguments)
	if err != nil {
		// Serialize error response
		requestJSON, _ := sonic.MarshalString(map[string]any{
			"tool":      toolName,
			"arguments": arguments,
		})
		return requestJSON, "", fmt.Errorf("call mcp tool failed: %w", err)
	}

	// 5. Serialize request
	requestJSON, _ := sonic.MarshalString(map[string]any{
		"tool":      toolName,
		"arguments": arguments,
	})

	logs.Infof("[MCP] Tool call succeeded: tool=%s, result_len=%d", toolName, len(resultStr))

	// 6. Wrap response in JSON format
	// Coze Studio expects all plugin responses to be JSON objects
	// MCP tools return text, so we wrap it in a standard structure
	responseJSON, err := sonic.MarshalString(map[string]any{
		"output": resultStr,
	})
	if err != nil {
		return requestJSON, "", fmt.Errorf("marshal mcp response failed: %w", err)
	}

	return requestJSON, responseJSON, nil
}

// parseMCPConfig parses MCP config from invocation args
func (m *mcpCallImpl) parseMCPConfig(args *InvocationArgs) (*mcp.Config, error) {
	if args.PluginManifest == nil {
		return nil, fmt.Errorf("plugin manifest is nil")
	}

	// Get MCP config from manifest.api.extensions.mcp_config
	if args.PluginManifest.API.Extensions == nil {
		return nil, fmt.Errorf("manifest.api.extensions is nil")
	}

	mcpConfigData, ok := args.PluginManifest.API.Extensions["mcp_config"]
	if !ok {
		return nil, fmt.Errorf("mcp_config not found in manifest.api.extensions")
	}

	// Convert to JSON and parse
	configJSON, err := json.Marshal(mcpConfigData)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp_config failed: %w", err)
	}

	logs.Infof("[MCP] Parsing config: %s", string(configJSON))

	var config mcp.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("unmarshal mcp_config failed: %w", err)
	}

	logs.Infof("[MCP] Config parsed: transport=%s", config.TransportType)

	return &config, nil
}

// getOrCreateClient gets or creates MCP client
func (m *mcpCallImpl) getOrCreateClient(ctx context.Context, config *mcp.Config) (*mcp.Client, error) {
	configHash := m.hashConfig(config)

	// Try to get from cache
	m.mu.RLock()
	if client, ok := m.clients[configHash]; ok {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	// Create new client
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check
	if client, ok := m.clients[configHash]; ok {
		return client, nil
	}

	// Create client using mcp-go library
	client, err := mcp.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create mcp client failed: %w", err)
	}

	// Initialize client
	if err := client.Initialize(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize mcp client failed: %w", err)
	}

	// Cache client
	m.clients[configHash] = client

	logs.Infof("[MCP] Created new client: transport=%s, hash=%s", config.TransportType, configHash[:8])

	return client, nil
}

// hashConfig generates a hash for the config
func (m *mcpCallImpl) hashConfig(config *mcp.Config) string {
	data, _ := json.Marshal(config)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
