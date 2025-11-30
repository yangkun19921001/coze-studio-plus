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

package conf

import (
	"context"
	"encoding/json"
	"fmt"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/coze-dev/coze-studio/backend/api/model/plugin_develop/common"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/consts"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/convert/api"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/model"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/dto"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/entity"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/mcp"
)

// mcpToolLoader handles loading tools from MCP servers
type mcpToolLoader struct{}

// LoadMCPToolsForPlugin is a public wrapper to load MCP tools
func LoadMCPToolsForPlugin(
	ctx context.Context,
	pluginID int64,
	pluginVersion string,
	mcpConfig *mcp.Config,
) ([]*ToolInfo, error) {
	loader := &mcpToolLoader{}
	return loader.LoadMCPTools(ctx, pluginID, pluginVersion, mcpConfig)
}

// LoadMCPTools loads tools from MCP server and converts them to ToolInfo
func (l *mcpToolLoader) LoadMCPTools(
	ctx context.Context,
	pluginID int64,
	pluginVersion string,
	mcpConfig *mcp.Config,
) ([]*ToolInfo, error) {
	logs.CtxInfof(ctx, "[MCP] Loading tools for plugin_id=%d, transport=%s", pluginID, mcpConfig.TransportType)

	// Create MCP client
	client, err := mcp.NewClient(mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("create mcp client failed: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logs.CtxErrorf(ctx, "[MCP] Failed to close client: %v", closeErr)
		}
	}()

	// Initialize client
	if err := client.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize mcp client failed: %w", err)
	}

	// List tools from MCP server
	mcpTools, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list mcp tools failed: %w", err)
	}

	if len(mcpTools) == 0 {
		logs.CtxWarnf(ctx, "[MCP] No tools found for plugin_id=%d", pluginID)
		return []*ToolInfo{}, nil
	}

	logs.CtxInfof(ctx, "[MCP] Found %d tools for plugin_id=%d", len(mcpTools), pluginID)

	// Convert MCP tools to ToolInfo
	toolInfos := make([]*ToolInfo, 0, len(mcpTools))
	for idx, mcpTool := range mcpTools {
		toolInfo, err := l.convertMCPToolToToolInfo(ctx, pluginID, pluginVersion, mcpTool, idx)
		if err != nil {
			logs.CtxErrorf(ctx, "[MCP] Failed to convert tool '%s': %v", mcpTool.Name, err)
			continue
		}
		toolInfos = append(toolInfos, toolInfo)
	}

	logs.CtxInfof(ctx, "[MCP] Successfully loaded %d tools for plugin_id=%d", len(toolInfos), pluginID)
	return toolInfos, nil
}

// convertMCPToolToToolInfo converts an MCP tool to ToolInfo
func (l *mcpToolLoader) convertMCPToolToToolInfo(
	ctx context.Context,
	pluginID int64,
	pluginVersion string,
	mcpTool mcpgo.Tool,
	toolIndex int,
) (*ToolInfo, error) {
	// Generate tool ID: plugin_id * 1000 + index
	// This ensures uniqueness while keeping it readable
	toolID := pluginID*1000 + int64(toolIndex+1)

	// Convert MCP input schema to OpenAPI operation
	operation, err := l.convertMCPToolToOpenAPIOperation(mcpTool)
	if err != nil {
		return nil, fmt.Errorf("convert to openapi operation failed: %w", err)
	}

	// Validate operation
	if err := operation.Validate(ctx); err != nil {
		return nil, fmt.Errorf("validate operation failed: %w", err)
	}

	// Create ToolInfo
	toolInfo := &ToolInfo{
		Info: &entity.ToolInfo{
			ID:              toolID,
			PluginID:        pluginID,
			Version:         ptr.Of(pluginVersion),
			Method:          ptr.Of("POST"),                               // MCP tools use POST by default
			SubURL:          ptr.Of(fmt.Sprintf("/mcp/%s", mcpTool.Name)), // Use tool name as sub URL
			Operation:       operation,
			ActivatedStatus: ptr.Of(consts.ActivateTool),
			DebugStatus:     ptr.Of(common.APIDebugStatus_DebugPassed),
		},
	}

	return toolInfo, nil
}

// convertMCPToolToOpenAPIOperation converts MCP tool to OpenAPI operation
func (l *mcpToolLoader) convertMCPToolToOpenAPIOperation(mcpTool mcpgo.Tool) (*model.Openapi3Operation, error) {
	// Convert MCP input schema to JSON schema format
	jsonSchema, err := l.convertMCPInputSchemaToJSONSchema(mcpTool.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("convert input schema failed: %w", err)
	}

	// Convert JSON schema to API parameters
	apiParams := l.convertJSONSchemaToAPIParameters(jsonSchema)

	// Create OpenAPI operation from API parameters
	openapiOp, err := api.APIParamsToOpenapiOperation(apiParams, []*common.APIParameter{})
	if err != nil {
		return nil, fmt.Errorf("create openapi operation failed: %w", err)
	}

	// Set operation metadata
	openapiOp.OperationID = mcpTool.Name
	openapiOp.Summary = mcpTool.Description

	// Ensure default response if not set
	if openapiOp.Responses == nil || len(openapiOp.Responses) == 0 {
		openapiOp.Responses = model.DefaultOpenapi3Responses()
	}

	return &model.Openapi3Operation{
		Operation: openapiOp,
	}, nil
}

// convertMCPInputSchemaToJSONSchema converts MCP input schema to JSON schema format
func (l *mcpToolLoader) convertMCPInputSchemaToJSONSchema(inputSchema interface{}) (*dto.JsonSchema, error) {
	if inputSchema == nil {
		// Return empty object schema if no input schema
		return &dto.JsonSchema{
			Type:       dto.JsonSchemaType_OBJECT,
			Properties: make(map[string]*dto.JsonSchema),
			Required:   []string{},
		}, nil
	}

	// Marshal to JSON first
	jsonBytes, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal input schema failed: %w", err)
	}

	// Unmarshal to JsonSchema
	var jsonSchema dto.JsonSchema
	if err := json.Unmarshal(jsonBytes, &jsonSchema); err != nil {
		return nil, fmt.Errorf("unmarshal json schema failed: %w", err)
	}

	return &jsonSchema, nil
}

// convertJSONSchemaToAPIParameters converts JSON schema to API parameters
func (l *mcpToolLoader) convertJSONSchemaToAPIParameters(schema *dto.JsonSchema) []*common.APIParameter {
	if schema == nil {
		return []*common.APIParameter{}
	}

	return l.convertJsonSchemaToParameters(schema, common.ParameterLocation_Body)
}

// convertJsonSchemaToParameters recursively converts JSON schema to API parameters
// This is similar to the implementation in tool_impl.go
func (l *mcpToolLoader) convertJsonSchemaToParameters(
	schema *dto.JsonSchema,
	location common.ParameterLocation,
) []*common.APIParameter {
	if schema == nil {
		return []*common.APIParameter{}
	}

	var parameters []*common.APIParameter

	// Handle object type with properties
	if schema.Type == dto.JsonSchemaType_OBJECT && len(schema.Properties) > 0 {
		// Create a set of required fields for quick lookup
		requiredFields := make(map[string]bool)
		for _, field := range schema.Required {
			requiredFields[field] = true
		}

		// Convert each property to a parameter
		for name, propSchema := range schema.Properties {
			if propSchema == nil {
				continue
			}

			param := &common.APIParameter{
				Name:       name,
				Desc:       propSchema.Description,
				IsRequired: requiredFields[name],
				Type:       l.mapJsonSchemaTypeToParameterType(propSchema.Type),
				Location:   location,
			}

			// Handle nested object properties
			if propSchema.Type == dto.JsonSchemaType_OBJECT && len(propSchema.Properties) > 0 {
				param.SubParameters = l.convertJsonSchemaToParameters(propSchema, location)
			}

			// Handle array properties
			if propSchema.Type == dto.JsonSchemaType_ARRAY && propSchema.Items != nil {
				// Create a parameter for the array item
				arrayItemParam := &common.APIParameter{
					Name:       "[Array Item]",
					Desc:       propSchema.Items.Description,
					IsRequired: true,
					Type:       l.mapJsonSchemaTypeToParameterType(propSchema.Items.Type),
					Location:   location,
				}

				// If array item is an object, recursively convert its properties
				if propSchema.Items.Type == dto.JsonSchemaType_OBJECT && len(propSchema.Items.Properties) > 0 {
					arrayItemParam.SubParameters = l.convertJsonSchemaToParameters(propSchema.Items, location)
				}

				param.SubParameters = []*common.APIParameter{arrayItemParam}
			}

			parameters = append(parameters, param)
		}
	} else if schema.Type == dto.JsonSchemaType_ARRAY && schema.Items != nil {
		// Handle top-level array
		arrayItemParam := &common.APIParameter{
			Name:       "[Array Item]",
			Desc:       schema.Items.Description,
			IsRequired: true,
			Type:       l.mapJsonSchemaTypeToParameterType(schema.Items.Type),
			Location:   location,
		}

		if schema.Items.Type == dto.JsonSchemaType_OBJECT && len(schema.Items.Properties) > 0 {
			arrayItemParam.SubParameters = l.convertJsonSchemaToParameters(schema.Items, location)
		}

		parameters = append(parameters, arrayItemParam)
	}

	return parameters
}

// mapJsonSchemaTypeToParameterType maps JSON schema type to API parameter type
func (l *mcpToolLoader) mapJsonSchemaTypeToParameterType(jsonType dto.JsonSchemaType) common.ParameterType {
	switch jsonType {
	case dto.JsonSchemaType_STRING:
		return common.ParameterType_String
	case dto.JsonSchemaType_INTEGER:
		return common.ParameterType_Integer
	case dto.JsonSchemaType_NUMBER:
		return common.ParameterType_Number
	case dto.JsonSchemaType_BOOLEAN:
		return common.ParameterType_Bool
	case dto.JsonSchemaType_ARRAY:
		return common.ParameterType_Array
	case dto.JsonSchemaType_OBJECT:
		return common.ParameterType_Object
	default:
		return common.ParameterType_String
	}
}
