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

package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// TransportType defines the type of MCP transport
type TransportType string

const (
	TransportTypeStdio TransportType = "stdio"
	TransportTypeSSE   TransportType = "sse"
)

// Config contains configuration for MCP client
type Config struct {
	ServerName    string        `json:"server_name"`
	TransportType TransportType `json:"transport_type"`
	StdioConfig   *StdioConfig  `json:"stdio_config,omitempty"`
	SSEConfig     *SSEConfig    `json:"sse_config,omitempty"`
}

// StdioConfig contains configuration for stdio transport
type StdioConfig struct {
	Command    []string          `json:"command"`
	Env        map[string]string `json:"env,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
}

// SSEConfig contains configuration for SSE transport
type SSEConfig struct {
	URL     string            `json:"url"`
	APIKey  string            `json:"api_key,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Client wraps the mcp-go client with simplified interface
type Client struct {
	client    *client.Client
	transport transport.Interface // Keep transport reference for Start()
}

// NewClient creates a new MCP client based on configuration
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	switch config.TransportType {
	case TransportTypeStdio:
		if config.StdioConfig == nil {
			return nil, fmt.Errorf("stdio config is required")
		}
		return newStdioClient(config.StdioConfig)
	case TransportTypeSSE:
		if config.SSEConfig == nil {
			return nil, fmt.Errorf("sse config is required")
		}
		return newSSEClient(config.SSEConfig)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.TransportType)
	}
}

// newStdioClient creates a stdio-based MCP client
func newStdioClient(config *StdioConfig) (*Client, error) {
	if len(config.Command) == 0 {
		return nil, fmt.Errorf("command is empty")
	}

	// Build environment variables
	envVars := []string{}
	for k, v := range config.Env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Create stdio client using NewStdioMCPClient
	mcpClient, err := client.NewStdioMCPClient(
		config.Command[0],
		envVars,
		config.Command[1:]...,
	)
	if err != nil {
		return nil, fmt.Errorf("create stdio client failed: %w", err)
	}

	return &Client{client: mcpClient}, nil
}

// newSSEClient creates an SSE-based MCP client
func newSSEClient(config *SSEConfig) (*Client, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("url is empty")
	}

	// Build client options using the proper transport package
	var options []transport.ClientOption

	// Add headers if provided
	if len(config.Headers) > 0 {
		options = append(options, client.WithHeaders(config.Headers))
	}

	// Create SSE MCP client
	mcpClient, err := client.NewSSEMCPClient(config.URL, options...)
	if err != nil {
		return nil, fmt.Errorf("create sse client failed: %w", err)
	}

	// Get the transport to call Start()
	sseTransport := mcpClient.GetTransport()

	// IMPORTANT: Start the SSE client connection
	if err := sseTransport.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("start sse client failed: %w", err)
	}

	return &Client{
		client:    mcpClient,
		transport: sseTransport,
	}, nil
}

// Initialize initializes the MCP client connection
func (c *Client) Initialize(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("client is nil")
	}

	// Build initialize request following mcp-go convention
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "coze-studio",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	// Call Initialize with proper request
	_, err := c.client.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	// List tools to verify connectivity
	result, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("list tools failed: %w", err)
	}

	// Log available tools for debugging
	toolCount := len(result.Tools)
	if toolCount > 0 {
		var toolNames []string
		for _, tool := range result.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		fmt.Printf("[MCP] Successfully initialized. Available tools (%d): %v\n", toolCount, toolNames)
	} else {
		fmt.Printf("[MCP] Successfully initialized but no tools found\n")
	}

	return nil
}

// CallTool calls a tool with the given name and arguments
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("client is nil")
	}

	result, err := c.client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	})
	if err != nil {
		return "", fmt.Errorf("call tool failed: %w", err)
	}

	// Convert result to string
	return convertToolResultToString(result), nil
}

// ListTools lists all available tools
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client is nil")
	}

	result, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}

	return result.Tools, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// convertToolResultToString converts MCP tool result to string
func convertToolResultToString(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	// If error, return error message
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				return "Error: " + textContent.Text
			}
		}
		return "Error occurred"
	}

	// Extract text content
	var textParts []string
	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			textParts = append(textParts, c.Text)
		case mcp.ImageContent:
			textParts = append(textParts, fmt.Sprintf("[Image: %s]", c.MIMEType))
		case mcp.EmbeddedResource:
			// Get URI from resource
			uri := ""
			if blob, ok := c.Resource.(mcp.BlobResourceContents); ok {
				uri = blob.URI
			} else if text, ok := c.Resource.(mcp.TextResourceContents); ok {
				uri = text.URI
			}
			textParts = append(textParts, fmt.Sprintf("[Resource: %s]", uri))
		}
	}

	if len(textParts) == 0 {
		return ""
	}

	// Join all text parts
	return strings.Join(textParts, "\n")
}
