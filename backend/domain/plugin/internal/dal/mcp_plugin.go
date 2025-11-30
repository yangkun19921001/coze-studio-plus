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

package dal

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/domain/plugin/dto"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/internal/dal/model"
	"github.com/coze-dev/coze-studio/backend/infra/idgen"
)

func NewMcpPluginDAO(db *gorm.DB, idGen idgen.IDGenerator) *McpPluginDAO {
	return &McpPluginDAO{
		idGen: idGen,
		db:    db,
	}
}

type McpPluginDAO struct {
	idGen idgen.IDGenerator
	db    *gorm.DB
}

// Create creates a new MCP plugin
func (m *McpPluginDAO) Create(ctx context.Context, req *dto.CreateMcpPluginRequest, userID int64) (int64, error) {
	id, err := m.idGen.GenID(ctx)
	if err != nil {
		return 0, err
	}

	now := time.Now().UnixMilli()
	mcpPlugin := &model.McpPlugin{
		ID:        id,
		Name:      req.Name,
		PluginID:  0,             // User-created plugins have plugin_id = 0
		UserID:    userID,        // User ID for user-created plugins
		McpConfig: req.McpConfig, // json.RawMessage can be directly assigned
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := m.db.WithContext(ctx).Create(mcpPlugin).Error; err != nil {
		return 0, err
	}

	return id, nil
}

// Update updates an existing MCP plugin
func (m *McpPluginDAO) Update(ctx context.Context, req *dto.UpdateMcpPluginRequest) error {
	now := time.Now().UnixMilli()

	// First, verify the record exists and get current data for logging
	var existing model.McpPlugin
	err := m.db.WithContext(ctx).
		Where("id = ?", req.ID).
		First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return gorm.ErrRecordNotFound
		}
		return err
	}

	// Use map to ensure complete replacement of JSON field, not merge
	// Use Select to explicitly specify fields to update, ensuring JSON field is completely replaced
	result := m.db.WithContext(ctx).
		Model(&model.McpPlugin{}).
		Where("id = ?", req.ID).
		Select("name", "mcp_config", "updated_at").
		Updates(map[string]interface{}{
			"name":       req.Name,
			"mcp_config": req.McpConfig, // Force complete replacement of JSON field
			"updated_at": now,
		})

	if result.Error != nil {
		return result.Error
	}

	// Check if any row was actually updated
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// Get gets an MCP plugin by user_id (for user-created plugins)
// Since each user can only have one MCP plugin, we use user_id to get it
func (m *McpPluginDAO) Get(ctx context.Context, userID int64) (*dto.McpPluginInfo, bool, error) {
	var mcpPlugin model.McpPlugin
	err := m.db.WithContext(ctx).
		Where("plugin_id = ? AND user_id = ?", 0, userID).
		First(&mcpPlugin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	return m.toDTO(mcpPlugin), true, nil
}

// GetByID gets an MCP plugin by ID (for backward compatibility, mainly for system plugins)
func (m *McpPluginDAO) GetByID(ctx context.Context, id int64) (*dto.McpPluginInfo, bool, error) {
	var mcpPlugin model.McpPlugin
	err := m.db.WithContext(ctx).
		Where("id = ?", id).
		First(&mcpPlugin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	return m.toDTO(mcpPlugin), true, nil
}

// GetByPluginID gets MCP plugin by plugin_id (from plugin_meta.yaml)
func (m *McpPluginDAO) GetByPluginID(ctx context.Context, pluginID int64) (*dto.McpPluginInfo, bool, error) {
	var mcpPlugin model.McpPlugin
	err := m.db.WithContext(ctx).
		Where("plugin_id = ?", pluginID).
		First(&mcpPlugin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	return m.toDTO(mcpPlugin), true, nil
}

// List lists MCP plugins with pagination, filtered by userID if provided
func (m *McpPluginDAO) List(ctx context.Context, req *dto.ListMcpPluginsRequest, userID *int64) ([]*dto.McpPluginInfo, int64, error) {
	offset := int((req.Page - 1) * req.PageSize)
	limit := int(req.PageSize)

	query := m.db.WithContext(ctx).Model(&model.McpPlugin{})

	// Filter by userID if provided (for user-created plugins)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get list
	var mcpPlugins []model.McpPlugin
	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&mcpPlugins).Error
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*dto.McpPluginInfo, 0, len(mcpPlugins))
	for _, plugin := range mcpPlugins {
		dtos = append(dtos, m.toDTO(plugin))
	}

	return dtos, total, nil
}

// GetAll gets all MCP plugins
func (m *McpPluginDAO) GetAll(ctx context.Context) ([]*dto.McpPluginInfo, error) {
	var mcpPlugins []model.McpPlugin
	err := m.db.WithContext(ctx).Find(&mcpPlugins).Error
	if err != nil {
		return nil, err
	}

	dtos := make([]*dto.McpPluginInfo, 0, len(mcpPlugins))
	for _, plugin := range mcpPlugins {
		dtos = append(dtos, m.toDTO(plugin))
	}

	return dtos, nil
}

// UpsertByPluginID upserts MCP plugin by plugin_id (for syncing from plugin_meta.yaml)
func (m *McpPluginDAO) UpsertByPluginID(ctx context.Context, pluginID int64, name string, mcpConfig json.RawMessage) error {
	now := time.Now().UnixMilli()

	// Try to get existing
	var existing model.McpPlugin
	err := m.db.WithContext(ctx).
		Where("plugin_id = ?", pluginID).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		id, genErr := m.idGen.GenID(ctx)
		if genErr != nil {
			return genErr
		}
		mcpPlugin := &model.McpPlugin{
			ID:        id,
			Name:      name,
			PluginID:  pluginID,
			UserID:    0, // System plugins have user_id = 0
			McpConfig: mcpConfig,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return m.db.WithContext(ctx).Create(mcpPlugin).Error
	} else if err != nil {
		return err
	}

	// Update existing - use map to ensure complete replacement
	err = m.db.WithContext(ctx).
		Model(&model.McpPlugin{}).
		Where("id = ?", existing.ID).
		Updates(map[string]interface{}{
			"name":       name,
			"mcp_config": mcpConfig, // Force complete replacement of JSON field
			"updated_at": now,
		}).Error
	return err
}

// Delete deletes an MCP plugin
func (m *McpPluginDAO) Delete(ctx context.Context, id int64) error {
	err := m.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.McpPlugin{}).Error
	return err
}

// DeleteAllUserCreated deletes all user-created MCP plugins for a specific user (plugin_id = 0 and user_id = userID)
func (m *McpPluginDAO) DeleteAllUserCreated(ctx context.Context, userID int64) error {
	err := m.db.WithContext(ctx).
		Where("plugin_id = ? AND user_id = ?", 0, userID).
		Delete(&model.McpPlugin{}).Error
	return err
}

// DeleteAllUserCreatedForAllUsers deletes all user-created MCP plugins for all users (plugin_id = 0)
// This is a temporary method for testing, should be removed after testing
func (m *McpPluginDAO) DeleteAllUserCreatedForAllUsers(ctx context.Context) error {
	err := m.db.WithContext(ctx).
		Where("plugin_id = ?", 0).
		Delete(&model.McpPlugin{}).Error
	return err
}

// DeleteAllUserCreatedExcept deletes all user-created MCP plugins for a specific user except the specified ID
func (m *McpPluginDAO) DeleteAllUserCreatedExcept(ctx context.Context, userID int64, exceptID int64) error {
	err := m.db.WithContext(ctx).
		Where("plugin_id = ? AND user_id = ? AND id != ?", 0, userID, exceptID).
		Delete(&model.McpPlugin{}).Error
	return err
}

// delete all MCP plugins
func (m *McpPluginDAO) DeleteAll(ctx context.Context) error {
	err := m.db.WithContext(ctx).
		Delete(&model.McpPlugin{}).Error
	return err
}

func (m *McpPluginDAO) toDTO(plugin model.McpPlugin) *dto.McpPluginInfo {
	return &dto.McpPluginInfo{
		ID:        plugin.ID,
		Name:      plugin.Name,
		PluginID:  plugin.PluginID,
		UserID:    plugin.UserID,
		McpConfig: plugin.McpConfig, // json.RawMessage can be directly assigned
		CreatedAt: plugin.CreatedAt,
		UpdatedAt: plugin.UpdatedAt,
	}
}
