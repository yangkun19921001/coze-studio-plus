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
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/application/base/ctxutil"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/consts"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/conf"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/dto"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/mcp"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

// CreateMcpPlugin creates a new MCP plugin
// Ensures only one user-created MCP plugin exists per user by deleting others before create
func (p *pluginServiceImpl) CreateMcpPlugin(ctx context.Context, req *dto.CreateMcpPluginRequest) (id int64, err error) {
	// Get user ID from context
	userID := ctxutil.MustGetUIDFromCtx(ctx)

	// Validate MCP config format - json.RawMessage is already []byte
	mcpConfigBytes := req.McpConfig
	var cursorConfig dto.McpServersConfig
	if err := json.Unmarshal(mcpConfigBytes, &cursorConfig); err != nil {
		return 0, errorx.Wrapf(err, "invalid MCP config format")
	}

	if len(cursorConfig.McpServers) == 0 {
		return 0, errorx.New(errno.ErrPluginInvalidParamCode, errorx.KV(errno.PluginMsgKey, "MCP config must contain at least one server"))
	}

	// Convert cursor format to internal format for validation
	_, err = ConvertCursorMcpConfigToInternal(mcpConfigBytes)
	if err != nil {
		return 0, errorx.Wrapf(err, "invalid MCP config: %v", err)
	}

	// Delete all existing user-created MCP plugins for this user before creating new one
	// This ensures only one user-created config exists per user
	err = p.mcpPluginDAO.DeleteAllUserCreated(ctx, userID)
	if err != nil {
		return 0, errorx.Wrapf(err, "failed to delete existing user-created MCP plugins")
	}

	id, err = p.mcpPluginDAO.Create(ctx, req, userID)
	if err != nil {
		return 0, errorx.Wrapf(err, "failed to create MCP plugin")
	}

	logs.CtxInfof(ctx, "[MCP] Created MCP plugin: id=%d, name=%s, user_id=%d", id, req.Name, userID)
	return id, nil
}

// UpdateMcpPlugin updates an existing MCP plugin
// Ensures only one user-created MCP plugin exists per user by deleting others before update
func (p *pluginServiceImpl) UpdateMcpPlugin(ctx context.Context, req *dto.UpdateMcpPluginRequest) (err error) {
	// Get user ID from context
	userID := ctxutil.MustGetUIDFromCtx(ctx)

	// Get plugin by user_id (since each user can only have one MCP plugin)
	existingPlugin, exists, err := p.mcpPluginDAO.Get(ctx, userID)
	if err != nil {
		return errorx.Wrapf(err, "failed to check if plugin exists")
	}
	if !exists {
		return errorx.New(errno.ErrPluginRecordNotFound, errorx.KV(errno.PluginMsgKey, fmt.Sprintf("MCP plugin for user_id=%d not found", userID)))
	}

	// Use the existing plugin's ID for update
	req.ID = existingPlugin.ID

	logs.CtxInfof(ctx, "[MCP] Updating plugin: id=%d, name=%s, existing_name=%s, user_id=%d", req.ID, req.Name, existingPlugin.Name, userID)
	logs.CtxInfof(ctx, "[MCP] Old config: %s", string(existingPlugin.McpConfig))
	logs.CtxInfof(ctx, "[MCP] New config: %s", string(req.McpConfig))

	// Validate MCP config format - json.RawMessage is already []byte
	mcpConfigBytes := req.McpConfig
	var cursorConfig dto.McpServersConfig
	if err := json.Unmarshal(mcpConfigBytes, &cursorConfig); err != nil {
		return errorx.Wrapf(err, "invalid MCP config format")
	}

	if len(cursorConfig.McpServers) == 0 {
		return errorx.New(errno.ErrPluginInvalidParamCode, errorx.KV(errno.PluginMsgKey, "MCP config must contain at least one server"))
	}

	// Convert cursor format to internal format for validation
	_, err = ConvertCursorMcpConfigToInternal(mcpConfigBytes)
	if err != nil {
		return errorx.Wrapf(err, "invalid MCP config: %v", err)
	}

	// Delete all other user-created MCP plugins for this user except the one being updated
	// This ensures only one user-created config exists per user
	err = p.mcpPluginDAO.DeleteAllUserCreatedExcept(ctx, userID, req.ID)
	if err != nil {
		return errorx.Wrapf(err, "failed to delete other user-created MCP plugins")
	}

	err = p.mcpPluginDAO.Update(ctx, req)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errorx.New(errno.ErrPluginRecordNotFound, errorx.KV(errno.PluginMsgKey, fmt.Sprintf("MCP plugin with id=%d not found", req.ID)))
		}
		return errorx.Wrapf(err, "failed to update MCP plugin")
	}

	// Verify update by reading back the record
	updatedPlugin, exists, err := p.mcpPluginDAO.Get(ctx, userID)
	if err != nil {
		logs.CtxErrorf(ctx, "[MCP] Failed to verify update: %v", err)
	} else if exists {
		logs.CtxInfof(ctx, "[MCP] Verified update: id=%d, updated config: %s", updatedPlugin.ID, string(updatedPlugin.McpConfig))
	}

	logs.CtxInfof(ctx, "[MCP] Updated MCP plugin: id=%d, name=%s, user_id=%d", req.ID, req.Name, userID)
	return nil
}

// ListMcpPlugins lists MCP plugins with pagination, filtered by current user
func (p *pluginServiceImpl) ListMcpPlugins(ctx context.Context, req *dto.ListMcpPluginsRequest) (resp *dto.ListMcpPluginsResponse, err error) {
	// TODO: 临时删除所有用户创建的 MCP 插件，测试完成后请注释掉
	// err = p.mcpPluginDAO.DeleteAllUserCreatedForAllUsers(ctx)
	// if err != nil {
	// 	logs.CtxWarnf(ctx, "[MCP] Failed to delete all user-created plugins: %v", err)
	// } else {
	// 	logs.CtxInfof(ctx, "[MCP] Deleted all user-created MCP plugins")
	// }

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.PageSize < 0 {
		req.PageSize = 20
	}

	// Get user ID from context to filter user-created plugins
	userID := ctxutil.MustGetUIDFromCtx(ctx)
	userIDPtr := &userID

	plugins, total, err := p.mcpPluginDAO.List(ctx, req, userIDPtr)
	if err != nil {
		return nil, errorx.Wrapf(err, "failed to list MCP plugins")
	}

	return &dto.ListMcpPluginsResponse{
		Plugins: plugins,
		Total:   total,
	}, nil
}

// GetMcpPlugin gets an MCP plugin by user_id
func (p *pluginServiceImpl) GetMcpPlugin(ctx context.Context, userID int64) (plugin *dto.McpPluginInfo, err error) {
	plugin, exist, err := p.mcpPluginDAO.Get(ctx, userID)
	if err != nil {
		return nil, errorx.Wrapf(err, "failed to get MCP plugin")
	}
	if !exist {
		return nil, errorx.New(errno.ErrPluginRecordNotFound, errorx.KV(errno.PluginMsgKey, "MCP plugin not found"))
	}
	return plugin, nil
}

// DeleteMcpPlugin deletes an MCP plugin
func (p *pluginServiceImpl) DeleteMcpPlugin(ctx context.Context, id int64) (err error) {
	err = p.mcpPluginDAO.Delete(ctx, id)
	if err != nil {
		return errorx.Wrapf(err, "failed to delete MCP plugin")
	}

	logs.CtxInfof(ctx, "[MCP] Deleted MCP plugin: id=%d", id)
	return nil
}

// SyncMcpPluginsFromYaml syncs MCP plugins from plugin_meta.yaml to database
func (p *pluginServiceImpl) SyncMcpPluginsFromYaml(ctx context.Context) (err error) {
	// Get all plugin products from conf
	pluginProducts := conf.GetAllPluginProducts()

	for _, pluginInfo := range pluginProducts {
		if pluginInfo.Info == nil || pluginInfo.Info.Manifest == nil {
			continue
		}

		// Check if this is an MCP plugin
		if pluginInfo.Info.Manifest.API.Type != consts.PluginTypeOfMCP {
			continue
		}

		pluginID := pluginInfo.Info.ID
		manifest := pluginInfo.Info.Manifest

		// Parse MCP config from manifest
		if manifest.API.Extensions == nil {
			logs.CtxWarnf(ctx, "[MCP] No extensions found for plugin_id=%d", pluginID)
			continue
		}

		mcpConfigData, ok := manifest.API.Extensions["mcp_config"]
		if !ok {
			logs.CtxWarnf(ctx, "[MCP] No mcp_config found for plugin_id=%d", pluginID)
			continue
		}

		// Convert to JSON
		mcpConfigJSON, err := json.Marshal(mcpConfigData)
		if err != nil {
			logs.CtxErrorf(ctx, "[MCP] Failed to marshal mcp_config for plugin_id=%d: %v", pluginID, err)
			continue
		}

		// Convert internal format to cursor format for storage
		var internalConfig mcp.Config
		if err := json.Unmarshal(mcpConfigJSON, &internalConfig); err != nil {
			logs.CtxErrorf(ctx, "[MCP] Failed to unmarshal mcp_config for plugin_id=%d: %v", pluginID, err)
			continue
		}

		// Convert to cursor format
		cursorConfigJSON, err := ConvertInternalMcpConfigToCursor([]*mcp.Config{&internalConfig})
		if err != nil {
			logs.CtxErrorf(ctx, "[MCP] Failed to convert config to cursor format for plugin_id=%d: %v", pluginID, err)
			continue
		}

		// Upsert to database
		name := manifest.NameForHuman
		if name == "" {
			name = manifest.NameForModel
		}
		if name == "" {
			name = fmt.Sprintf("mcp-plugin-%d", pluginID)
		}

		err = p.mcpPluginDAO.UpsertByPluginID(ctx, pluginID, name, cursorConfigJSON)
		if err != nil {
			logs.CtxErrorf(ctx, "[MCP] Failed to upsert MCP plugin plugin_id=%d: %v", pluginID, err)
			continue
		}

		logs.CtxInfof(ctx, "[MCP] Synced MCP plugin to database: plugin_id=%d, name=%s", pluginID, name)
	}

	return nil
}
