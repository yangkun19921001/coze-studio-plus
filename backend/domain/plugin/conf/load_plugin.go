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
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mohae/deepcopy"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"

	"encoding/json"

	common "github.com/coze-dev/coze-studio/backend/api/model/plugin_develop/common"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/consts"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/model"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/dto"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/entity"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/mcp"
)

type pluginProductMeta struct {
	PluginID       int64                 `yaml:"plugin_id" validate:"required"`
	Deprecated     bool                  `yaml:"deprecated"`
	Version        string                `yaml:"version" validate:"required"`
	PluginType     common.PluginType     `yaml:"plugin_type" validate:"required"`
	OpenapiDocFile string                `yaml:"openapi_doc_file" validate:"required"`
	Manifest       *model.PluginManifest `yaml:"manifest" validate:"required"`
	Tools          []*toolProductMeta    `yaml:"tools" validate:"required"`
}

type toolProductMeta struct {
	ToolID     int64  `yaml:"tool_id" validate:"required"`
	Deprecated bool   `yaml:"deprecated"`
	Method     string `yaml:"method" validate:"required"`
	SubURL     string `yaml:"sub_url" validate:"required"`
}

var (
	pluginProducts map[int64]*PluginInfo
	toolProducts   map[int64]*ToolInfo
)

func GetToolProduct(toolID int64) (*ToolInfo, bool) {
	ti, ok := toolProducts[toolID]
	if !ok {
		return nil, false
	}

	ti_ := deepcopy.Copy(ti).(*ToolInfo)

	return ti_, true
}

func MGetToolProducts(toolIDs []int64) []*ToolInfo {
	tools := make([]*ToolInfo, 0, len(toolIDs))
	for _, toolID := range toolIDs {
		ti, ok := GetToolProduct(toolID)
		if !ok {
			continue
		}

		tools = append(tools, ti)
	}

	return tools
}

func GetPluginProduct(pluginID int64) (*PluginInfo, bool) {
	pl, ok := pluginProducts[pluginID]
	return pl, ok
}

func MGetPluginProducts(pluginIDs []int64) []*PluginInfo {
	plugins := make([]*PluginInfo, 0, len(pluginIDs))
	for _, pluginID := range pluginIDs {
		pl, ok := pluginProducts[pluginID]
		if !ok {
			continue
		}
		plugins = append(plugins, pl)
	}
	return plugins
}

func GetAllPluginProducts() []*PluginInfo {
	plugins := make([]*PluginInfo, 0, len(pluginProducts))
	for _, pl := range pluginProducts {
		plugins = append(plugins, pl)
	}
	return plugins
}

type PluginInfo struct {
	Info    *model.PluginInfo
	ToolIDs []int64
}

func (pi PluginInfo) GetPluginAllTools() (tools []*ToolInfo) {
	tools = make([]*ToolInfo, 0, len(pi.ToolIDs))
	for _, toolID := range pi.ToolIDs {
		ti, ok := toolProducts[toolID]
		if !ok {
			continue
		}
		tools = append(tools, ti)
	}
	return tools
}

type ToolInfo struct {
	Info *entity.ToolInfo
}

func loadPluginProductMeta(ctx context.Context, basePath string) (err error) {
	logs.CtxInfof(ctx, "basePath=%s", basePath)
	root := path.Join(basePath, "pluginproduct")
	metaFile := path.Join(root, "plugin_meta.yaml")

	file, err := os.ReadFile(metaFile)
	if err != nil {
		return fmt.Errorf("read file '%s' failed, err=%v", metaFile, err)
	}

	var pluginsMeta []*pluginProductMeta
	err = yaml.Unmarshal(file, &pluginsMeta)
	if err != nil {
		return fmt.Errorf("unmarshal file '%s' failed, err=%v", metaFile, err)
	}

	pluginProducts = make(map[int64]*PluginInfo, len(pluginsMeta))
	toolProducts = map[int64]*ToolInfo{}

	for _, m := range pluginsMeta {
		logs.CtxInfof(ctx, "load plugin meta, openapi_doc_file=%s", m.OpenapiDocFile)
		if !checkPluginMetaInfo(ctx, m) {
			logs.CtxInfof(ctx, "checkPluginMetaInfo failed, openapi_doc_file=%s", m.OpenapiDocFile)
			continue
		}

		err = m.Manifest.Validate(true)
		if err != nil {
			logs.CtxErrorf(ctx, "plugin manifest validates failed, openapi_doc_file=%s, err=%v", m.OpenapiDocFile, err)
			continue
		}

		// Check if this is an MCP plugin
		isMCPPlugin := m.Manifest.API.Type == consts.PluginTypeOfMCP

		var doc *model.Openapi3T
		var pi *PluginInfo

		if isMCPPlugin {
			// For MCP plugins, load tools from MCP server
			logs.CtxInfof(ctx, "[MCP] Loading MCP plugin: plugin_id=%d", m.PluginID)

			// Parse MCP config
			mcpConfig, err := parseMCPConfigFromManifest(m.Manifest)
			if err != nil {
				logs.CtxErrorf(ctx, "[MCP] Failed to parse MCP config for plugin_id=%d: %v", m.PluginID, err)
				continue
			}

			// Create a minimal OpenAPI doc for MCP plugins
			doc = createMinimalOpenAPIDocForMCP(m.Manifest)

			pi = &PluginInfo{
				Info: &model.PluginInfo{
					ID:         m.PluginID,
					PluginType: m.PluginType,
					Version:    ptr.Of(m.Version),
					IconURI:    ptr.Of(m.Manifest.LogoURL),
					ServerURL:  ptr.Of("mcp://"), // MCP plugins don't have HTTP server URL
					Manifest:   m.Manifest,
					OpenapiDoc: doc,
				},
				ToolIDs: make([]int64, 0),
			}

			if pluginProducts[m.PluginID] != nil {
				logs.CtxErrorf(ctx, "duplicate plugin id '%d', openapi_doc_file=%s", m.PluginID, m.OpenapiDocFile)
				continue
			}

			pluginProducts[m.PluginID] = pi

			// Load tools from MCP server
			loader := &mcpToolLoader{}
			mcpTools, err := loader.LoadMCPTools(ctx, m.PluginID, m.Version, mcpConfig)
			if err != nil {
				logs.CtxErrorf(ctx, "[MCP] Failed to load tools for plugin_id=%d: %v", m.PluginID, err)
				delete(pluginProducts, m.PluginID)
				continue
			}

			// Add MCP tools to toolProducts
			for _, toolInfo := range mcpTools {
				// Check for duplicate tool ID
				if _, ok := toolProducts[toolInfo.Info.ID]; ok {
					logs.CtxWarnf(ctx, "[MCP] Duplicate tool id '%d' for plugin_id=%d, skipping", toolInfo.Info.ID, m.PluginID)
					continue
				}

				pi.ToolIDs = append(pi.ToolIDs, toolInfo.Info.ID)
				toolProducts[toolInfo.Info.ID] = toolInfo
			}

			// Also add tools from meta.yaml if specified (for backward compatibility)
			for _, t := range m.Tools {
				if t.Deprecated {
					continue
				}

				_, ok := toolProducts[t.ToolID]
				if ok {
					logs.CtxWarnf(ctx, "[MCP] Duplicate tool id '%d' in meta.yaml for plugin_id=%d, skipping", t.ToolID, m.PluginID)
					continue
				}

				// For MCP plugins, tools from meta.yaml are optional and may not have OpenAPI doc
				// We'll skip them if they're not found in MCP server response
				logs.CtxInfof(ctx, "[MCP] Tool from meta.yaml (tool_id=%d) will be ignored, using MCP server tools", t.ToolID)
			}
		} else {
			// For regular OpenAPI plugins, use existing logic
			docPath := path.Join(root, m.OpenapiDocFile)
			loader := openapi3.NewLoader()
			_doc, err := loader.LoadFromFile(docPath)
			if err != nil {
				logs.CtxErrorf(ctx, "load openapi3 doc file '%s', err=%v", docPath, err)
				continue
			}

			doc = ptr.Of(model.Openapi3T(*_doc))

			err = doc.Validate(ctx)
			if err != nil {
				logs.CtxErrorf(ctx, "the openapi3 doc '%s' validates failed, err=%v", m.OpenapiDocFile, err)
				continue
			}

			pi = &PluginInfo{
				Info: &model.PluginInfo{
					ID:         m.PluginID,
					PluginType: m.PluginType,
					Version:    ptr.Of(m.Version),
					IconURI:    ptr.Of(m.Manifest.LogoURL),
					ServerURL:  ptr.Of(doc.Servers[0].URL),
					Manifest:   m.Manifest,
					OpenapiDoc: doc,
				},
				ToolIDs: make([]int64, 0, len(m.Tools)),
			}

			if pluginProducts[m.PluginID] != nil {
				logs.CtxErrorf(ctx, "duplicate plugin id '%d', openapi_doc_file=%s", m.PluginID, m.OpenapiDocFile)
				continue
			}

			pluginProducts[m.PluginID] = pi

			apis := make(map[dto.UniqueToolAPI]*model.Openapi3Operation, len(doc.Paths))
			for subURL, pathItem := range doc.Paths {
				for method, op := range pathItem.Operations() {
					api := dto.UniqueToolAPI{
						SubURL: subURL,
						Method: strings.ToUpper(method),
					}
					apis[api] = model.NewOpenapi3Operation(op)
				}
			}

			for _, t := range m.Tools {
				if t.Deprecated {
					continue
				}

				_, ok := toolProducts[t.ToolID]
				if ok {
					logs.CtxErrorf(ctx, "duplicate tool id '%d', openapi_doc_file=%s", t.ToolID, m.OpenapiDocFile)
					continue
				}

				api := dto.UniqueToolAPI{
					SubURL: t.SubURL,
					Method: strings.ToUpper(t.Method),
				}
				op, ok := apis[api]
				if !ok {
					logs.CtxErrorf(ctx, "api '[%s]:%s' not found in doc '%s', openapi_doc_file=%s", api.Method, api.SubURL, docPath, m.OpenapiDocFile)
					continue
				}
				if err = op.Validate(ctx); err != nil {
					logs.CtxErrorf(ctx, "the openapi3 operation of tool '[%s]:%s' in '%s' validates failed, openapi_doc_file=%s, err=%v",
						t.Method, t.SubURL, docPath, m.OpenapiDocFile, err)
					continue
				}

				pi.ToolIDs = append(pi.ToolIDs, t.ToolID)

				toolProducts[t.ToolID] = &ToolInfo{
					Info: &entity.ToolInfo{
						ID:              t.ToolID,
						PluginID:        m.PluginID,
						Version:         ptr.Of(m.Version),
						Method:          ptr.Of(t.Method),
						SubURL:          ptr.Of(t.SubURL),
						Operation:       op,
						ActivatedStatus: ptr.Of(consts.ActivateTool),
						DebugStatus:     ptr.Of(common.APIDebugStatus_DebugPassed),
					},
				}
			}
		}

		if len(pi.ToolIDs) == 0 {
			delete(pluginProducts, m.PluginID)
		}
	}

	return nil
}

func checkPluginMetaInfo(ctx context.Context, m *pluginProductMeta) (continued bool) {
	if m.Deprecated {
		return false
	}

	if !semver.IsValid(m.Version) {
		logs.CtxErrorf(ctx, "invalid version '%s'", m.Version)
		return false
	}
	if m.PluginID <= 0 {
		logs.CtxErrorf(ctx, "invalid plugin id '%d'", m.PluginID)
		return false
	}

	_, ok := toolProducts[m.PluginID]
	if ok {
		logs.CtxErrorf(ctx, "duplicate plugin id '%d'", m.PluginID)
		return false
	}
	if m.PluginType != common.PluginType_PLUGIN {
		logs.CtxErrorf(ctx, "invalid plugin type '%s'", m.PluginType)
		return false
	}

	return true
}

// parseMCPConfigFromManifest parses MCP config from plugin manifest
func parseMCPConfigFromManifest(manifest *model.PluginManifest) (*mcp.Config, error) {
	if manifest == nil {
		return nil, fmt.Errorf("plugin manifest is nil")
	}

	if manifest.API.Extensions == nil {
		return nil, fmt.Errorf("manifest.api.extensions is nil")
	}

	mcpConfigData, ok := manifest.API.Extensions["mcp_config"]
	if !ok {
		return nil, fmt.Errorf("mcp_config not found in manifest.api.extensions")
	}

	// Convert to JSON and parse
	configJSON, err := json.Marshal(mcpConfigData)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp_config failed: %w", err)
	}

	var config mcp.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("unmarshal mcp_config failed: %w", err)
	}

	return &config, nil
}

// createMinimalOpenAPIDocForMCP creates a minimal OpenAPI document for MCP plugins
func createMinimalOpenAPIDocForMCP(manifest *model.PluginManifest) *model.Openapi3T {
	doc := &openapi3.T{
		OpenAPI: "3.0.1",
		Info: &openapi3.Info{
			Title:       manifest.NameForHuman,
			Description: manifest.DescriptionForHuman,
			Version:     "1.0.0",
		},
		Servers: openapi3.Servers{
			&openapi3.Server{
				URL: "mcp://",
			},
		},
		Paths: make(openapi3.Paths),
	}

	return ptr.Of(model.Openapi3T(*doc))
}
