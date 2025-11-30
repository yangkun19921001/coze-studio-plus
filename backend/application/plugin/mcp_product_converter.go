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

package plugin

import (
	"context"
	"fmt"
	"time"

	productCommon "github.com/coze-dev/coze-studio/backend/api/model/marketplace/product_common"
	productAPI "github.com/coze-dev/coze-studio/backend/api/model/marketplace/product_public_api"
	common "github.com/coze-dev/coze-studio/backend/api/model/plugin_develop/common"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/consts"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/model"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/conf"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/dto"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/entity"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/service"
	"github.com/coze-dev/coze-studio/backend/infra/storage"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/mcp"
)

// McpProductConverter handles conversion of MCP plugins to product info
type McpProductConverter struct {
	domainSVC service.PluginService
	oss       storage.Storage
}

// NewMcpProductConverter creates a new MCP product converter
func NewMcpProductConverter(domainSVC service.PluginService, oss storage.Storage) *McpProductConverter {
	return &McpProductConverter{
		domainSVC: domainSVC,
		oss:       oss,
	}
}

// GetMcpPluginsAsProducts gets MCP plugins and converts them to product info list
// Each MCP server configuration becomes a separate product
func (c *McpProductConverter) GetMcpPluginsAsProducts(ctx context.Context) ([]*productAPI.ProductInfo, error) {
	// Get all MCP plugins from database
	listReq := &dto.ListMcpPluginsRequest{
		Page:     1,
		PageSize: 1000, // Get all user-created MCP plugins
	}
	listResp, err := c.domainSVC.ListMcpPlugins(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP plugins: %w", err)
	}

	products := make([]*productAPI.ProductInfo, 0)
	for _, mcpPlugin := range listResp.Plugins {
		// Skip plugins from plugin_meta.yaml (they're already in local plugins)
		if mcpPlugin.PluginID > 0 {
			continue
		}

		// Convert each MCP server config to a separate product
		productInfos, err := c.convertMcpPluginToProducts(ctx, mcpPlugin)
		if err != nil {
			logs.CtxErrorf(ctx, "[MCP] Failed to convert MCP plugin %d to products: %v", mcpPlugin.ID, err)
			continue
		}

		products = append(products, productInfos...)
	}

	return products, nil
}

// convertMcpPluginToProducts converts a single MCP plugin to multiple ProductInfo
// Each MCP server configuration becomes a separate product with its own tools
func (c *McpProductConverter) convertMcpPluginToProducts(ctx context.Context, mcpPlugin *dto.McpPluginInfo) ([]*productAPI.ProductInfo, error) {
	// Convert cursor format to internal format
	internalConfigs, err := service.ConvertCursorMcpConfigToInternal(mcpPlugin.McpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}

	if len(internalConfigs) == 0 {
		return nil, fmt.Errorf("no valid MCP config found")
	}

	products := make([]*productAPI.ProductInfo, 0, len(internalConfigs))
	for i, config := range internalConfigs {
		// Get tools for this specific MCP server
		tools, err := c.getMcpPluginTools(ctx, mcpPlugin.ID, config)
		if err != nil {
			logs.CtxWarnf(ctx, "[MCP] Failed to get tools for plugin %d server %d: %v", mcpPlugin.ID, i, err)
			// Continue even if tools loading fails for one server
			tools = []*entity.ToolInfo{}
		}

		// Build plugin entity for this specific server config
		pluginEntity := c.buildPluginEntityFromMcpPlugin(mcpPlugin, config, i)

		// Build product info with server index for unique product ID
		metaInfo, err := c.buildProductMetaInfo(ctx, pluginEntity, mcpPlugin.ID, i)
		if err != nil {
			logs.CtxWarnf(ctx, "[MCP] Failed to build meta info for plugin %d server %d: %v", mcpPlugin.ID, i, err)
			continue
		}

		extraInfo, err := c.buildPluginProductExtraInfo(ctx, pluginEntity, i, tools)
		if err != nil {
			logs.CtxWarnf(ctx, "[MCP] Failed to build extra info for plugin %d server %d: %v", mcpPlugin.ID, i, err)
			continue
		}

		products = append(products, &productAPI.ProductInfo{
			CommercialSetting: &productCommon.CommercialSetting{
				CommercialType: productCommon.ProductPaidType_Free,
			},
			MetaInfo:    metaInfo,
			PluginExtra: extraInfo,
		})
	}

	return products, nil
}

// buildPluginEntityFromMcpPlugin builds entity.PluginInfo from MCP plugin
// mcpConfigs contains the MCP server configuration(s) for this product
// serverIndex is the index of the server in the original config (for naming)
// Note: ID is set to the real MCP plugin ID (mcpPlugin.ID) so that backend can find it when adding plugin
// The product ID (for listing) is generated separately in buildProductMetaInfo
func (c *McpProductConverter) buildPluginEntityFromMcpPlugin(mcpPlugin *dto.McpPluginInfo, mcpConfig *mcp.Config, serverIndex int) *entity.PluginInfo {
	return &entity.PluginInfo{
		PluginInfo: &model.PluginInfo{
			ID:         time.Now().UnixNano() + int64(serverIndex), // Use real MCP plugin ID so backend can find it
			PluginType: common.PluginType_LOCAL,
			Version:    ptr.Of("v1.0.0"),
			IconURI:    ptr.Of("official_plugin_icon/plugin_mcp.png"),
			ServerURL:  ptr.Of("mcp://"),
			Manifest: &model.PluginManifest{
				SchemaVersion:       "v1",
				NameForModel:        mcpConfig.ServerName,
				NameForHuman:        mcpConfig.ServerName,
				DescriptionForModel: fmt.Sprintf("MCP plugin: %s", mcpConfig.ServerName),
				DescriptionForHuman: fmt.Sprintf("MCP plugin: %s", mcpConfig.ServerName),
				LogoURL:             "official_plugin_icon/plugin_mcp.png",
				Auth: &model.AuthV2{
					Type: consts.AuthzTypeOfNone,
				},
				API: model.APIDesc{
					Type: consts.PluginTypeOfMCP,
					Extensions: map[string]interface{}{
						"mcp_config": mcpConfig, // Store the specific server config
					},
				},
			},
		},
	}
}

// getMcpPluginTools gets tools for an MCP plugin
func (c *McpProductConverter) getMcpPluginTools(ctx context.Context, pluginID int64, mcpConfig *mcp.Config) ([]*entity.ToolInfo, error) {
	// Try to get tools from plugin products cache first
	pluginProducts := conf.GetAllPluginProducts()
	for _, pluginProduct := range pluginProducts {
		if pluginProduct.Info != nil && pluginProduct.Info.ID == pluginID {
			// Get tools from cache
			toolInfos := make([]*entity.ToolInfo, 0, len(pluginProduct.ToolIDs))
			for _, toolID := range pluginProduct.ToolIDs {
				toolProduct, exists := conf.GetToolProduct(toolID)
				if exists && toolProduct != nil && toolProduct.Info != nil {
					toolInfos = append(toolInfos, toolProduct.Info)
				}
			}
			return toolInfos, nil
		}
	}

	// If not in cache, load from MCP server
	toolInfos, err := conf.LoadMCPToolsForPlugin(ctx, pluginID, "v1.0.0", mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP tools: %w", err)
	}

	// Convert to entity.ToolInfo
	tools := make([]*entity.ToolInfo, 0, len(toolInfos))
	for _, toolInfo := range toolInfos {
		tools = append(tools, toolInfo.Info)
	}

	return tools, nil
}

// buildProductMetaInfo builds ProductMetaInfo from plugin entity
// For MCP plugins, we generate a unique product ID for each server config
// but keep EntityID as the real MCP plugin ID so backend can find it when adding
func (c *McpProductConverter) buildProductMetaInfo(ctx context.Context, plugin *entity.PluginInfo, mcpPluginID int64, serverIndex int) (*productAPI.ProductMetaInfo, error) {
	iconURL, err := c.oss.GetObjectUrl(ctx, plugin.GetIconURI())
	if err != nil {
		logs.CtxWarnf(ctx, "get icon url failed with '%s', err=%v", plugin.GetIconURI(), err)
	}

	// Generate unique product ID for each server config
	// Use mcpPluginID*1000 + serverIndex to ensure uniqueness
	// productID := mcpPluginID/1000000 + int64(serverIndex)
	productID := time.Now().Unix() + int64(serverIndex)

	return &productAPI.ProductMetaInfo{
		ID:          0,         // Unique product ID for listing
		EntityID:    productID, // Real MCP plugin ID so backend can find it when adding
		EntityType:  productCommon.ProductEntityType_Plugin,
		IconURL:     iconURL,
		Name:        plugin.GetName(),
		Description: plugin.GetDesc(),
		IsFree:      true,
		IsOfficial:  true,
		Status:      productCommon.ProductStatus_Listed,
		ListedAt:    time.Now().Unix(),
		UserInfo: &productCommon.UserInfo{
			Name: "Coze Official",
		},
	}, nil
}

// buildPluginProductExtraInfo builds PluginExtraInfo from plugin entity and tools
func (c *McpProductConverter) buildPluginProductExtraInfo(ctx context.Context, plugin *entity.PluginInfo, serverIndex int, tools []*entity.ToolInfo) (*productAPI.PluginExtraInfo, error) {
	ei := &productAPI.PluginExtraInfo{
		IsOfficial: true,
		PluginType: func() *productCommon.PluginType {
			if plugin.Manifest != nil && plugin.Manifest.API.Type == consts.PluginTypeOfMCP {
				pt := productCommon.PluginType_LocalPlugin
				return &pt
			}
			pt := productCommon.PluginType_LocalPlugin
			return &pt
		}(),
		Tools:      make([]*productAPI.PluginToolInfo, 0, len(tools)),
		Connectors: nil, // MCP plugins don't have channel restrictions, set to nil explicitly
	}

	// Convert tools
	for index, tool := range tools {
		params, err := tool.ToToolParameters()
		if err != nil {
			logs.CtxWarnf(ctx, "[MCP] Failed to convert tool parameters for tool %d: %v", tool.ID, err)
			params = []*productAPI.ToolParameter{}
		}

		toolInfo := &productAPI.PluginToolInfo{
			ID:          time.Now().UnixNano() + int64(serverIndex) + int64(index),
			Name:        tool.GetName(),
			Description: tool.GetDesc(),
			Parameters:  params,
		}
		ei.Tools = append(ei.Tools, toolInfo)
	}

	return ei, nil
}
