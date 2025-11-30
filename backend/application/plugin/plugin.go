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
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/coze-dev/coze-studio/backend/bizpkg/config"

	"github.com/coze-dev/coze-studio/backend/api/model/app/bot_common"
	productCommon "github.com/coze-dev/coze-studio/backend/api/model/marketplace/product_common"
	productAPI "github.com/coze-dev/coze-studio/backend/api/model/marketplace/product_public_api"
	pluginAPI "github.com/coze-dev/coze-studio/backend/api/model/plugin_develop"
	common "github.com/coze-dev/coze-studio/backend/api/model/plugin_develop/common"
	"github.com/coze-dev/coze-studio/backend/application/base/ctxutil"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/consts"
	"github.com/coze-dev/coze-studio/backend/crossdomain/plugin/convert/api"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/conf"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/dto"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/entity"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/repository"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/service"
	search "github.com/coze-dev/coze-studio/backend/domain/search/service"
	userEntity "github.com/coze-dev/coze-studio/backend/domain/user/entity"
	user "github.com/coze-dev/coze-studio/backend/domain/user/service"
	"github.com/coze-dev/coze-studio/backend/infra/storage"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

var PluginApplicationSVC = &PluginApplicationService{}

type PluginApplicationService struct {
	DomainSVC service.PluginService
	eventbus  search.ResourceEventBus
	oss       storage.Storage
	userSVC   user.User

	toolRepo   repository.ToolRepository
	pluginRepo repository.PluginRepository
}

func (p *PluginApplicationService) CheckAndLockPluginEdit(ctx context.Context, req *pluginAPI.CheckAndLockPluginEditRequest) (resp *pluginAPI.CheckAndLockPluginEditResponse, err error) {
	resp = &pluginAPI.CheckAndLockPluginEditResponse{
		Data: &common.CheckAndLockPluginEditData{
			Seized: true,
		},
	}

	return resp, nil
}

func (p *PluginApplicationService) GetBotDefaultParams(ctx context.Context, req *pluginAPI.GetBotDefaultParamsRequest) (resp *pluginAPI.GetBotDefaultParamsResponse, err error) {

	draftAgentTool, err := p.DomainSVC.GetDraftAgentToolByName(ctx, req.BotID, req.PluginID, req.APIName)
	if err != nil {
		return nil, errorx.Wrapf(err, "GetDraftAgentToolByName failed, agentID=%d, toolName=%s", req.BotID, req.APIName)
	}
	reqAPIParams, err := draftAgentTool.ToReqAPIParameter()
	if err != nil {
		return nil, err
	}
	respAPIParams, err := draftAgentTool.ToRespAPIParameter()
	if err != nil {
		return nil, err
	}

	resp = &pluginAPI.GetBotDefaultParamsResponse{
		RequestParams:  reqAPIParams,
		ResponseParams: respAPIParams,
	}

	return resp, nil
}

func (p *PluginApplicationService) UpdateBotDefaultParams(ctx context.Context, req *pluginAPI.UpdateBotDefaultParamsRequest) (resp *pluginAPI.UpdateBotDefaultParamsResponse, err error) {
	op, err := api.APIParamsToOpenapiOperation(req.RequestParams, req.ResponseParams)
	if err != nil {
		return nil, err
	}

	err = p.DomainSVC.UpdateBotDefaultParams(ctx, &dto.UpdateBotDefaultParamsRequest{
		PluginID:    req.PluginID,
		ToolName:    req.APIName,
		AgentID:     req.BotID,
		Parameters:  op.Parameters,
		RequestBody: op.RequestBody,
		Responses:   op.Responses,
	})
	if err != nil {
		return nil, errorx.Wrapf(err, "UpdateBotDefaultParams failed, agentID=%d, toolName=%s", req.BotID, req.APIName)
	}

	resp = &pluginAPI.UpdateBotDefaultParamsResponse{}

	return resp, nil
}

func (p *PluginApplicationService) UnlockPluginEdit(ctx context.Context, req *pluginAPI.UnlockPluginEditRequest) (resp *pluginAPI.UnlockPluginEditResponse, err error) {
	resp = &pluginAPI.UnlockPluginEditResponse{
		Released: true,
	}
	return resp, nil
}

func (p *PluginApplicationService) PublicGetProductList(ctx context.Context, req *productAPI.GetProductListRequest) (resp *productAPI.GetProductListResponse, err error) {
	//首先处理  mcp 插件
	mcpConverter := NewMcpProductConverter(p.DomainSVC, p.oss)
	if err := p.addMcpPluginsToCache(ctx, mcpConverter); err != nil {
		logs.CtxWarnf(ctx, "[MCP] Failed to add MCP plugins to cache: %v", err)
		// Continue even if cache update fails
	}

	res, err := p.DomainSVC.ListPluginProducts(ctx, &dto.ListPluginProductsRequest{})
	if err != nil {
		return nil, errorx.Wrapf(err, "ListPluginProducts failed")
	}

	products := make([]*productAPI.ProductInfo, 0, len(res.Plugins))
	for _, pl := range res.Plugins {
		tls, err := p.toolRepo.GetPluginAllOnlineTools(ctx, pl.ID)
		if err != nil {
			return nil, errorx.Wrapf(err, "GetPluginAllOnlineTools failed, pluginID=%d", pl.ID)
		}

		pi, err := p.buildProductInfo(ctx, pl, tls)
		if err != nil {
			return nil, err
		}

		products = append(products, pi)
	}

	// mcpProducts, err := mcpConverter.GetMcpPluginsAsProducts(ctx)
	// if err != nil {
	// 	logs.CtxWarnf(ctx, "[MCP] Failed to get MCP plugins as products: %v", err)
	// 	// Continue even if MCP plugins loading fails
	// } else {
	// 	products = append(products, mcpProducts...)
	// 	// Add MCP plugins to in-memory cache so they can be found by GetPlaygroundPluginList
	// 	if err := p.addMcpPluginsToCache(ctx, mcpConverter); err != nil {
	// 		logs.CtxWarnf(ctx, "[MCP] Failed to add MCP plugins to cache: %v", err)
	// 		// Continue even if cache update fails
	// 	}
	// }

	//对比 pairat-remote-exec vs "Pairat Remote Exec" 数据，输出 log
	for _, product := range products {
		if product.MetaInfo.Name == "Pairat Remote Exec" {
			logs.CtxInfof(ctx, "Pairat Remote Exec: metaName=%s, productName=%s, entityID=%d, pluginID=%d, pluginType=%s, tools=%d, jumpSaasURL=%s, isOfficial=%t, id=%d", product.MetaInfo.Name, product.MetaInfo.Name, product.MetaInfo.EntityID, product.MetaInfo.ID, product.PluginExtra.PluginType.String(), len(product.PluginExtra.Tools), product.PluginExtra.JumpSaasURL, product.MetaInfo.IsOfficial, product.MetaInfo.ID)
			for _, tool := range product.PluginExtra.Tools {
				logs.CtxInfof(ctx, "tool: name=%s, description=%s, parameters=%v", tool.Name, tool.Description, tool.Parameters)
			}
		}
		if product.MetaInfo.Name == "pairat-remote-exec" {
			logs.CtxInfof(ctx, "pairat-remote-exec: metaName=%s, productName=%s, entityID=%d, pluginID=%d, pluginType=%s, tools=%d, jumpSaasURL=%s, isOfficial=%t", product.MetaInfo.Name, product.MetaInfo.Name, product.MetaInfo.EntityID, product.MetaInfo.ID, product.PluginExtra.PluginType.String(), len(product.PluginExtra.Tools), product.PluginExtra.JumpSaasURL, product.MetaInfo.IsOfficial)
			for _, tool := range product.PluginExtra.Tools {
				logs.CtxInfof(ctx, "tool: name=%s, description=%s, parameters=%v", tool.Name, tool.Description, tool.Parameters)
			}
		}
	}

	if req.GetKeyword() != "" {
		filterProducts := make([]*productAPI.ProductInfo, 0, len(products))
		for _, _p := range products {
			if strings.Contains(strings.ToLower(_p.MetaInfo.Name), strings.ToLower(req.GetKeyword())) {
				filterProducts = append(filterProducts, _p)
			}
		}
		products = filterProducts
	}

	resp = &productAPI.GetProductListResponse{
		Data: &productAPI.GetProductListData{
			Products: products,
			HasMore:  false, // Finish at one time
			Total:    int32(res.Total),
		},
	}

	return resp, nil
}

func (p *PluginApplicationService) buildProductInfo(ctx context.Context, plugin *entity.PluginInfo, tools []*entity.ToolInfo) (*productAPI.ProductInfo, error) {
	metaInfo, err := p.buildProductMetaInfo(ctx, plugin)
	if err != nil {
		return nil, err
	}

	extraInfo, err := p.buildPluginProductExtraInfo(ctx, plugin, tools)
	if err != nil {
		return nil, err
	}

	pi := &productAPI.ProductInfo{
		CommercialSetting: &productCommon.CommercialSetting{
			CommercialType: productCommon.ProductPaidType_Free,
		},
		MetaInfo:    metaInfo,
		PluginExtra: extraInfo,
	}

	return pi, nil
}

func (p *PluginApplicationService) buildProductMetaInfo(ctx context.Context, plugin *entity.PluginInfo) (*productAPI.ProductMetaInfo, error) {
	iconURL, err := p.oss.GetObjectUrl(ctx, plugin.GetIconURI())
	if err != nil {
		logs.CtxWarnf(ctx, "get icon url failed with '%s', err=%v", plugin.GetIconURI(), err)
	}

	return &productAPI.ProductMetaInfo{
		ID:          plugin.GetRefProductID(),
		EntityID:    plugin.ID,
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

func (p *PluginApplicationService) buildPluginProductExtraInfo(ctx context.Context, plugin *entity.PluginInfo, tools []*entity.ToolInfo) (*productAPI.PluginExtraInfo, error) {
	ei := &productAPI.PluginExtraInfo{
		IsOfficial: true,
		PluginType: func() *productCommon.PluginType {
			if plugin.PluginType == common.PluginType_LOCAL {
				return ptr.Of(productCommon.PluginType_LocalPlugin)
			}
			return ptr.Of(productCommon.PluginType_CLoudPlugin)
		}(),
		JumpSaasURL: func() *string {
			if plugin.SaasPluginExtra != nil && plugin.SaasPluginExtra.JumpSaasURL != nil {
				return plugin.SaasPluginExtra.JumpSaasURL
			}
			return nil
		}(),
	}

	toolInfos := make([]*productAPI.PluginToolInfo, 0, len(tools))
	for _, tl := range tools {
		params, err := tl.ToToolParameters()
		if err != nil {
			return nil, err
		}

		toolInfo := &productAPI.PluginToolInfo{
			ID:          tl.ID,
			Name:        tl.GetName(),
			Description: tl.GetDesc(),
			Parameters:  params,
		}

		example := plugin.GetToolExample(ctx, tl.GetName())
		if example != nil {
			toolInfo.Example = &productAPI.PluginToolExample{
				ReqExample:  example.RequestExample,
				RespExample: example.ResponseExample,
			}
		}

		toolInfos = append(toolInfos, toolInfo)
	}

	ei.Tools = toolInfos

	authInfo := plugin.GetAuthInfo()

	authMode := ptr.Of(productAPI.PluginAuthMode_NoAuth)
	if authInfo != nil {
		if authInfo.Type == consts.AuthzTypeOfService || authInfo.Type == consts.AuthzTypeOfOAuth {
			authMode = ptr.Of(productAPI.PluginAuthMode_Required)
			err := plugin.Manifest.Validate(false)
			if err != nil {
				logs.CtxWarnf(ctx, "validate plugin manifest failed, err=%v", err)
			} else {
				authMode = ptr.Of(productAPI.PluginAuthMode_Configured)
			}
		} else if authInfo.Type == consts.AuthTypeOfSaasInstalled {
			authMode = ptr.Of(productAPI.PluginAuthMode_NeedInstalled)
		}
	}

	ei.AuthMode = authMode

	return ei, nil
}

func (p *PluginApplicationService) PublicGetProductDetail(ctx context.Context, req *productAPI.GetProductDetailRequest) (resp *productAPI.GetProductDetailResponse, err error) {
	plugin, exist, err := p.pluginRepo.GetOnlinePlugin(ctx, req.GetEntityID())
	if err != nil {
		return nil, errorx.Wrapf(err, "GetOnlinePlugin failed, pluginID=%d", req.GetEntityID())
	}
	if !exist {
		return nil, errorx.New(errno.ErrPluginRecordNotFound)
	}

	tools, err := p.toolRepo.GetPluginAllOnlineTools(ctx, plugin.ID)
	if err != nil {
		return nil, errorx.Wrapf(err, "GetPluginAllOnlineTools failed, pluginID=%d", plugin.ID)
	}
	pi, err := p.buildProductInfo(ctx, plugin, tools)
	if err != nil {
		return nil, err
	}

	resp = &productAPI.GetProductDetailResponse{
		Data: &productAPI.GetProductDetailData{
			MetaInfo:    pi.MetaInfo,
			PluginExtra: pi.PluginExtra,
		},
	}

	return resp, nil
}

func (p *PluginApplicationService) validateDraftPluginAccess(ctx context.Context, pluginID int64) (plugin *entity.PluginInfo, err error) {
	uid := ctxutil.GetUIDFromCtx(ctx)
	if uid == nil {
		return nil, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "session is required"))
	}

	plugin, err = p.DomainSVC.GetDraftPlugin(ctx, pluginID)
	if err != nil {
		return nil, errorx.Wrapf(err, "GetDraftPlugin failed, pluginID=%d", pluginID)
	}

	if plugin.DeveloperID != *uid {
		return nil, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "you are not the plugin owner"))
	}

	return plugin, nil
}

// convertPluginToProductInfo converts a plugin entity to ProductInfo
func convertPluginToProductInfo(plugin *entity.PluginInfo) *productAPI.ProductInfo {

	isOfficial := func() bool {
		if plugin.SaasPluginExtra == nil {
			return true
		}
		return plugin.SaasPluginExtra.IsOfficial
	}()
	return &productAPI.ProductInfo{
		MetaInfo: &productAPI.ProductMetaInfo{
			ID:          plugin.GetRefProductID(),
			Name:        plugin.GetName(),
			EntityID:    plugin.ID,
			Description: plugin.GetDesc(),
			IconURL:     plugin.GetIconURI(),
			ListedAt:    plugin.CreatedAt,
			EntityType: func() productCommon.ProductEntityType {
				if ptr.From(plugin.Source) == bot_common.PluginFrom_FromSaas {
					return productCommon.ProductEntityType_SaasPlugin
				}
				return productCommon.ProductEntityType_Plugin
			}(),
			IsOfficial: isOfficial,
			Status:     productCommon.ProductStatus_Listed,
			UserInfo: &productCommon.UserInfo{
				Name: func() string {
					if isOfficial {
						return "Coze Official"
					}
					return "Coze Community"
				}(),
			},
		},
	}
}

func (p *PluginApplicationService) convertPluginsToProductInfos(ctx context.Context, plugins []*entity.PluginInfo, tools map[int64][]*entity.ToolInfo) []*productAPI.ProductInfo {
	products := make([]*productAPI.ProductInfo, 0, len(plugins))
	for _, plugin := range plugins {
		var pExtra *productAPI.PluginExtraInfo
		if tool, exist := tools[plugin.ID]; exist {
			pluginExtra, err := p.buildPluginProductExtraInfo(ctx, plugin, tool)
			if err != nil {
				logs.CtxErrorf(ctx, "buildPluginProductExtraInfo failed: %v", err)
			} else {
				pExtra = pluginExtra
			}
		}
		pi := convertPluginToProductInfo(plugin)
		pi.PluginExtra = pExtra
		products = append(products, pi)
	}
	return products
}
func (p *PluginApplicationService) convertPluginsToMetaInfos(ctx context.Context, plugins []*entity.PluginInfo, tools map[int64][]*entity.ToolInfo) []*productAPI.ProductMetaInfo {
	products := make([]*productAPI.ProductMetaInfo, 0, len(plugins))
	for _, plugin := range plugins {
		pi := &productAPI.ProductMetaInfo{
			ID:          plugin.ID,
			Name:        plugin.GetName(),
			Description: plugin.GetDesc(),
			IconURL:     plugin.GetIconURI(),
			ListedAt:    plugin.CreatedAt,
			EntityID:    plugin.ID,
		}
		products = append(products, pi)
	}
	return products
}

func (p *PluginApplicationService) getSaasPluginList(ctx context.Context, domainReq *dto.ListSaasPluginProductsRequest) (*dto.ListPluginProductsResponse, error) {
	return p.DomainSVC.ListSaasPluginProducts(ctx, domainReq)
}

func (p *PluginApplicationService) getSaasPluginToolsList(ctx context.Context, pluginIDs []int64) (map[int64][]*entity.ToolInfo, error) {
	tools, _, err := p.DomainSVC.BatchGetSaasPluginToolsInfo(ctx, pluginIDs)
	return tools, err
}

func (p *PluginApplicationService) GetCozeSaasPluginList(ctx context.Context, req *productAPI.GetProductListRequest) (resp *productAPI.GetProductListResponse, err error) {
	domainResp, err := p.getSaasPluginList(ctx, &dto.ListSaasPluginProductsRequest{
		PageNum:     ptr.Of(req.PageNum),
		PageSize:    ptr.Of(req.PageSize),
		Keyword:     req.Keyword,
		EntityTypes: req.EntityTypes,
	})
	if err != nil {
		logs.CtxErrorf(ctx, "ListSaasPluginProducts failed: %v", err)
		return nil, err
	}

	// tools
	pluginIDs := make([]int64, 0, len(domainResp.Plugins))
	for _, product := range domainResp.Plugins {
		pluginIDs = append(pluginIDs, product.ID)
	}

	tools, err := p.getSaasPluginToolsList(ctx, pluginIDs)
	if err != nil {
		logs.CtxErrorf(ctx, "BatchGetSaasPluginToolsInfo failed: %v", err)
		return nil, err
	}

	products := p.convertPluginsToProductInfos(ctx, domainResp.Plugins, tools)

	return &productAPI.GetProductListResponse{
		Code:    0,
		Message: "success",
		Data: &productAPI.GetProductListData{
			Products: products,
			Total:    int32(domainResp.Total),
			HasMore:  domainResp.HasMore,
		},
	}, nil
}

func (p *PluginApplicationService) PublicSearchProduct(ctx context.Context, req *productAPI.SearchProductRequest) (resp *productAPI.SearchProductResponse, err error) {
	domainResp, err := p.getSaasPluginList(ctx, &dto.ListSaasPluginProductsRequest{
		PageNum:  ptr.Of(req.PageNum),
		PageSize: ptr.Of(req.PageSize),
		Keyword:  ptr.Of(req.Keyword),
		EntityTypes: func() []productCommon.ProductEntityType {
			if req.EntityTypes == nil {
				return nil
			}
			return p.convertEntityTypesStrToSlice(*req.EntityTypes)
		}(),
		CategoryIDs:     req.CategoryIDs,
		IsOfficial:      req.IsOfficial,
		PluginType:      req.PluginType,
		ProductPaidType: req.ProductPaidType,
		SortType:        req.SortType,
	})
	if err != nil {
		logs.CtxErrorf(ctx, "ListSaasPluginProducts failed: %v", err)
		return nil, err
	}
	// tools
	pluginIDs := make([]int64, 0, len(domainResp.Plugins))
	for _, product := range domainResp.Plugins {
		pluginIDs = append(pluginIDs, product.ID)
	}

	tools, err := p.getSaasPluginToolsList(ctx, pluginIDs)
	if err != nil {
		logs.CtxErrorf(ctx, "BatchGetSaasPluginToolsInfo failed: %v", err)
		return nil, err
	}

	products := p.convertPluginsToProductInfos(ctx, domainResp.Plugins, tools)

	return &productAPI.SearchProductResponse{
		Code:    0,
		Message: "success",
		Data: &productAPI.SearchProductResponseData{
			Products: products,
			Total:    ptr.Of(int32(domainResp.Total)),
			HasMore:  ptr.Of(domainResp.HasMore),
		},
	}, nil
}

func (p *PluginApplicationService) convertEntityTypesStrToSlice(entityTypesStr string) []productCommon.ProductEntityType {
	var entityTypes []productCommon.ProductEntityType
	if entityTypesStr != "" {
		typeStrs := strings.Split(entityTypesStr, ",")
		for _, typeStr := range typeStrs {
			typeStr = strings.TrimSpace(typeStr)
			if typeStr != "" {
				if entityType, err := productCommon.ProductEntityTypeFromString(typeStr); err == nil {
					entityTypes = append(entityTypes, entityType)
				}
			}
		}
	}
	return entityTypes
}
func (p *PluginApplicationService) PublicSearchSuggest(ctx context.Context, req *productAPI.SearchSuggestRequest) (resp *productAPI.SearchSuggestResponse, err error) {
	domainResp, err := p.getSaasPluginList(ctx, &dto.ListSaasPluginProductsRequest{
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
		Keyword:  req.Keyword,
		EntityTypes: func() []productCommon.ProductEntityType {
			if req.EntityTypes == nil {
				return nil
			}
			return p.convertEntityTypesStrToSlice(*req.EntityTypes)
		}(),
	})

	if err != nil {
		logs.CtxErrorf(ctx, "ListSaasPluginProducts for suggestions failed: %v", err)
		return nil, err
	}

	// tools
	pluginIDs := make([]int64, 0, len(domainResp.Plugins))
	for _, product := range domainResp.Plugins {
		pluginIDs = append(pluginIDs, product.ID)
	}

	tools, err := p.getSaasPluginToolsList(ctx, pluginIDs)
	if err != nil {
		logs.CtxErrorf(ctx, "BatchGetSaasPluginToolsInfo failed: %v", err)
		return nil, err
	}

	suggestionProducts := p.convertPluginsToProductInfos(ctx, domainResp.Plugins, tools)

	return &productAPI.SearchSuggestResponse{
		Code:    0,
		Message: "success",
		Data: &productAPI.SearchSuggestResponseData{
			SuggestionV2: suggestionProducts,
			Suggestions:  p.convertPluginsToMetaInfos(ctx, domainResp.Plugins, tools),
			HasMore:      ptr.Of(domainResp.HasMore),
		},
	}, nil
}

func (p *PluginApplicationService) GetSaasProductCategoryList(ctx context.Context, req *productAPI.GetProductCategoryListRequest) (resp *productAPI.GetProductCategoryListResponse, err error) {

	domainReq := &dto.ListPluginCategoriesRequest{}

	if req.GetEntityType() == productCommon.ProductEntityType_SaasPlugin {
		domainReq.EntityType = ptr.Of("plugin")
	}

	domainResp, err := p.DomainSVC.ListSaasPluginCategories(ctx, domainReq)
	if err != nil {
		logs.CtxErrorf(ctx, "ListSaasPluginCategories failed: %v", err)
		return nil, err
	}

	// 转换响应数据
	categories := make([]*productAPI.ProductCategory, 0)
	if domainResp.Data != nil && domainResp.Data.Items != nil {
		for _, item := range domainResp.Data.Items {
			// 将字符串 ID 转换为 int64
			categoryID, _ := strconv.ParseInt(item.ID, 10, 64)
			categories = append(categories, &productAPI.ProductCategory{
				ID:   categoryID,
				Name: item.Name,
			})
		}
	}

	return &productAPI.GetProductCategoryListResponse{
		Code:    0,
		Message: "success",
		Data: &productAPI.GetProductCategoryListData{
			EntityType: req.GetEntityType(),
			Categories: categories,
		},
	}, nil
}

func (p *PluginApplicationService) GetProductCallInfo(ctx context.Context, req *productAPI.GetProductCallInfoRequest) (resp *productAPI.GetProductCallInfoResponse, err error) {
	userInfo, err := p.userSVC.GetSaasUserInfo(ctx)
	if err != nil {
		logs.CtxErrorf(ctx, "GetSaasUserInfo failed: %v", err)
		return nil, err
	}

	benefit, err := p.userSVC.GetUserBenefit(ctx)
	if err != nil {
		logs.CtxErrorf(ctx, "GetUserBenefit failed: %v", err)
		return nil, err
	}

	// Build response data
	data := &productAPI.GetProductCallInfoData{
		UserInfo: &productAPI.UserInfo{
			UserName:  ptr.Of(userInfo.UserName),
			NickName:  ptr.Of(userInfo.NickName),
			AvatarURL: ptr.Of(userInfo.AvatarURL),
		},
	}

	if benefit != nil {
		data.UserLevel = func() productAPI.UserLevel {
			switch benefit.UserLevel {
			case userEntity.UserLevelPro:
				return productAPI.UserLevel_ProPersonal
			case userEntity.UserLevelEnterprise:
				return productAPI.UserLevel_Enterprise
			default:
				return productAPI.UserLevel_Free
			}
		}()
		data.CallCountLimit = &productAPI.ProductCallCountLimit{
			IsUnlimited:   benefit.IsUnlimited,
			UsedCount:     benefit.UsedCount,
			TotalCount:    benefit.TotalCount,
			ResetDatetime: benefit.ResetDatetime,
		}
		data.CallRateLimit = &productAPI.ProductCallRateLimit{
			QPS: benefit.CallQPS,
		}
	}

	return &productAPI.GetProductCallInfoResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

func (p *PluginApplicationService) GetMarketPluginConfig(ctx context.Context, req *productAPI.GetMarketPluginConfigRequest) (resp *productAPI.GetMarketPluginConfigResponse, err error) {

	baseConfig, err := config.Base().GetBaseConfig(ctx)
	if err != nil {
		logs.CtxErrorf(ctx, "GetBaseConfig failed: %v", err)
		return nil, err
	}

	cozeSaasPluginEnabled := baseConfig.PluginConfiguration.CozeSaasPluginEnabled
	saasAPIKey := baseConfig.PluginConfiguration.CozeAPIToken

	enableSaasPlugin := cozeSaasPluginEnabled && len(saasAPIKey) > 0

	resp = &productAPI.GetMarketPluginConfigResponse{
		Code:    0,
		Message: "success",
		Data: &productAPI.Configuration{
			EnableSaasPlugin: &enableSaasPlugin,
		},
	}

	return resp, nil
}

// CreateMcpPlugin creates a new MCP plugin
func (p *PluginApplicationService) CreateMcpPlugin(ctx context.Context, req *dto.CreateMcpPluginRequest) (id int64, err error) {
	return p.DomainSVC.CreateMcpPlugin(ctx, req)
}

// UpdateMcpPlugin updates an existing MCP plugin
func (p *PluginApplicationService) UpdateMcpPlugin(ctx context.Context, req *dto.UpdateMcpPluginRequest) (err error) {
	return p.DomainSVC.UpdateMcpPlugin(ctx, req)
}

// ListMcpPlugins lists MCP plugins with pagination
func (p *PluginApplicationService) ListMcpPlugins(ctx context.Context, req *dto.ListMcpPluginsRequest) (resp *dto.ListMcpPluginsResponse, err error) {
	return p.DomainSVC.ListMcpPlugins(ctx, req)
}

// GetMcpPlugin gets an MCP plugin by user_id
func (p *PluginApplicationService) GetMcpPlugin(ctx context.Context, userID int64) (plugin *dto.McpPluginInfo, err error) {
	return p.DomainSVC.GetMcpPlugin(ctx, userID)
}

// DeleteMcpPlugin deletes an MCP plugin
func (p *PluginApplicationService) DeleteMcpPlugin(ctx context.Context, id int64) (err error) {
	return p.DomainSVC.DeleteMcpPlugin(ctx, id)
}

// string encoding to int64
func stringHashToInt64(s string) int64 {
	hash := fnv.New64a()
	hash.Write([]byte(s))
	return int64(hash.Sum64())
}

// addMcpPluginsToCache adds user-created MCP plugins to the in-memory cache
// so they can be found by GetPlaygroundPluginList when adding tools
func (p *PluginApplicationService) addMcpPluginsToCache(ctx context.Context, mcpConverter *McpProductConverter) error {
	// Get all MCP plugins from database
	listReq := &dto.ListMcpPluginsRequest{
		Page:     1,
		PageSize: 1000, // Get all user-created MCP plugins
	}
	listResp, err := p.DomainSVC.ListMcpPlugins(ctx, listReq)
	if err != nil {
		return fmt.Errorf("failed to get MCP plugins: %w", err)
	}

	for _, mcpPlugin := range listResp.Plugins {
		// Skip plugins from plugin_meta.yaml (they're already in cache)
		if mcpPlugin.PluginID > 0 {
			continue
		}

		delPluginCount, delToolCount := conf.DeleteAllLocalPluginProducts()
		logs.CtxInfof(ctx, "Deleted %d local plugin products, %d local tool products", delPluginCount, delToolCount)

		// Convert cursor format to internal format
		internalConfigs, err := service.ConvertCursorMcpConfigToInternal(mcpPlugin.McpConfig)
		if err != nil {
			logs.CtxWarnf(ctx, "[MCP] Failed to convert config for plugin %d: %v", mcpPlugin.ID, err)
			continue
		}

		if len(internalConfigs) == 0 {
			logs.CtxWarnf(ctx, "[MCP] No valid MCP config found for plugin %d", mcpPlugin.ID)
			continue
		}

		// Collect all tools from all server configs
		// Load directly from MCP server to avoid circular dependency with cache
		for i, config := range internalConfigs {
			allTools := make([]*entity.ToolInfo, 0)
			toolInfos, err := conf.LoadMCPToolsForPlugin(ctx, mcpPlugin.ID, "v1.0.0", config)
			if err != nil {
				logs.CtxWarnf(ctx, "[MCP] Failed to load tools for plugin %d server %d: %v", mcpPlugin.ID, i, err)
				continue
			}
			mcpPlugin.ID = stringHashToInt64(config.ServerName)

			// Convert to entity.ToolInfo
			for _, toolInfo := range toolInfos {
				if toolInfo != nil && toolInfo.Info != nil {
					toolInfo.Info.ID = stringHashToInt64(*toolInfo.Info.SubURL)
					toolInfo.Info.PluginID = mcpPlugin.ID
					allTools = append(allTools, toolInfo.Info)
				}
			}

			if len(allTools) == 0 {
				logs.CtxWarnf(ctx, "[MCP] No tools found for plugin %d, skipping cache", mcpPlugin.ID)
				continue
			}

			// Build plugin info (use the first server config for manifest)
			pluginEntity := mcpConverter.buildPluginEntityFromMcpPlugin(mcpPlugin, config, i)
			pluginEntity.PluginInfo.ID = mcpPlugin.ID
			// Add to cache
			err = conf.AddLocalMcpPlugin(ctx, mcpPlugin.ID, pluginEntity.PluginInfo, allTools)
			if err != nil {
				logs.CtxWarnf(ctx, "[MCP] Failed to add plugin %d to cache: %v", mcpPlugin.ID, err)
				continue
			}
		}

	}

	return nil
}
