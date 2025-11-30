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

import { useState, useEffect, useCallback, useMemo } from 'react';

import { Modal, Button, TextArea, Toast } from '@coze-arch/coze-design';

interface McpConfigModalProps {
  visible: boolean;
  onClose: () => void;
}

interface McpServerConfig {
  url?: string;
  command?: string | string[];
  args?: string[];
  env?: Record<string, string>;
  transport_type?: string;
}

interface McpConfig {
  mcpServers: Record<string, McpServerConfig>;
}

interface McpPlugin {
  id: number;
  name: string;
  mcp_config: string | McpConfig;
}

interface McpPluginListResponse {
  code: number;
  data?: {
    plugins?: McpPlugin[];
  };
  message?: string;
}

const JSON_INDENT = 2;
const MIN_SERVER_COUNT = 0;

// Default MCP config template (cursor format)
const getDefaultConfig = (): McpConfig => ({
  mcpServers: {
    'remote-exec': {
      url: 'http://10.1.48.133:8001/sse',
    },
  },
});

const parsePluginConfig = (plugin: McpPlugin): McpConfig | null => {
  try {
    let pluginConfig: McpConfig;
    if (typeof plugin.mcp_config === 'string') {
      pluginConfig = JSON.parse(plugin.mcp_config) as McpConfig;
    } else {
      pluginConfig = plugin.mcp_config as McpConfig;
    }
    return pluginConfig;
  } catch (e) {
    console.error('Failed to parse plugin config:', e);
    return null;
  }
};

const mergePluginConfigs = (plugins: McpPlugin[]): McpConfig => {
  const mergedConfig: McpConfig = { mcpServers: {} };
  plugins.forEach(plugin => {
    const pluginConfig = parsePluginConfig(plugin);
    if (pluginConfig?.mcpServers) {
      Object.assign(mergedConfig.mcpServers, pluginConfig.mcpServers);
    }
  });
  return mergedConfig;
};

/* eslint-disable @coze-arch/max-line-per-function -- MCP config modal has complex logic for loading, merging, and saving configs */
export const McpConfigModal: React.FC<McpConfigModalProps> = ({
  visible,
  onClose,
}) => {
  const [configJson, setConfigJson] = useState('');
  const [loading, setLoading] = useState(false);

  // Default MCP config template (cursor format)
  const defaultConfig = useMemo(() => getDefaultConfig(), []);

  const loadMcpPlugins = useCallback(async () => {
    try {
      // Use direct fetch until API schema is updated
      const res = await fetch('/api/plugin_api/list_mcp_plugins', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ page: 1, page_size: 100 }),
      });
      const data = (await res.json()) as McpPluginListResponse;
      if (
        data.code === 0 &&
        data.data?.plugins &&
        data.data.plugins.length > MIN_SERVER_COUNT
      ) {
        const mergedConfig = mergePluginConfigs(data.data.plugins);
        if (Object.keys(mergedConfig.mcpServers).length > MIN_SERVER_COUNT) {
          setConfigJson(JSON.stringify(mergedConfig, null, JSON_INDENT));
        }
      }
    } catch (error) {
      console.error('Failed to load MCP plugins:', error);
      // Don't show error message, just use default config
    }
  }, []);

  useEffect(() => {
    if (visible) {
      // Load existing MCP plugins
      loadMcpPlugins();
      // Set default config only if no existing plugins (will be overwritten by loadMcpPlugins if found)
      setConfigJson(JSON.stringify(defaultConfig, null, JSON_INDENT));
    } else {
      // Reset when modal closes
      setConfigJson('');
      setLoading(false);
    }
  }, [visible, loadMcpPlugins, defaultConfig]);

  const validateConfig = (configJsonStr: string): McpConfig => {
    if (!configJsonStr.trim()) {
      throw new Error('请输入MCP配置');
    }

    const config = JSON.parse(configJsonStr) as McpConfig;
    if (!config.mcpServers || typeof config.mcpServers !== 'object') {
      throw new Error('配置格式错误：必须包含 mcpServers 对象');
    }

    return config;
  };

  const findExistingPlugin = async (
    pluginName: string,
  ): Promise<McpPlugin | null> => {
    const listRes = await fetch('/api/plugin_api/list_mcp_plugins', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ page: 1, page_size: 100 }),
    });

    if (!listRes.ok) {
      throw new Error(`获取插件列表失败: ${listRes.status}`);
    }

    const listData = (await listRes.json()) as McpPluginListResponse;

    if (listData.code !== 0) {
      throw new Error(listData.message || '获取插件列表失败');
    }

    return listData.data?.plugins?.find(p => p.name === pluginName) || null;
  };

  const updatePlugin = async (
    pluginId: number,
    pluginName: string,
    config: McpConfig,
  ): Promise<void> => {
    const updateRes = await fetch('/api/plugin_api/update_mcp_plugin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: pluginId,
        name: pluginName,
        mcp_config: config,
      }),
    });

    if (!updateRes.ok) {
      throw new Error(`更新失败: ${updateRes.status}`);
    }

    const updateData = (await updateRes.json()) as McpPluginListResponse;
    if (updateData.code !== 0) {
      throw new Error(updateData.message || '更新失败');
    }
  };

  const createPlugin = async (
    pluginName: string,
    config: McpConfig,
  ): Promise<void> => {
    const createRes = await fetch('/api/plugin_api/create_mcp_plugin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: pluginName,
        mcp_config: config,
      }),
    });

    if (!createRes.ok) {
      throw new Error(`创建失败: ${createRes.status}`);
    }

    const createData = (await createRes.json()) as McpPluginListResponse;
    if (createData.code !== 0) {
      const errorMsg =
        createData.message || createData.data?.error || '创建失败';
      throw new Error(errorMsg);
    }
  };

  const handleSave = async () => {
    setLoading(true);
    try {
      const config = validateConfig(configJson);
      const pluginName = 'mcp-config';

      const existingPlugin = await findExistingPlugin(pluginName);

      if (existingPlugin) {
        await updatePlugin(existingPlugin.id, pluginName, config);
      } else {
        await createPlugin(pluginName, config);
      }

      Toast.success('MCP配置保存成功');
      onClose();
    } catch (error) {
      console.error('Failed to save MCP config:', error);
      const errorMsg =
        error instanceof Error
          ? error.message
          : String(error) || '配置格式错误';
      Toast.error(`保存失败: ${errorMsg}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="配置MCP工具"
      visible={visible}
      onCancel={onClose}
      width={800}
      footer={[
        <Button key="cancel" onClick={onClose}>
          取消
        </Button>,
        <Button
          key="save"
          type="primary"
          loading={loading}
          onClick={handleSave}
        >
          确认
        </Button>,
      ]}
    >
      <div style={{ marginBottom: 16 }}>
        <p style={{ marginBottom: 8, color: '#666' }}>
          请输入 MCP 服务器配置（JSON 格式，仅支持 SSE 和 stdio ）：
        </p>
        {/* <p style={{ marginBottom: 16, fontSize: 12, color: '#999' }}>
          格式示例：
          <br />
          {`{
  "mcpServers": {
    "server-name": {
      "url": "http://example.com/sse"  // SSE传输
    },
    "another-server": {
      "command": ["node", "/path/to/server.js"],  // stdio传输
      "env": {"NODE_ENV": "production"}
    }
  }
}`}
        </p> */}
      </div>
      <TextArea
        value={configJson}
        onChange={value => setConfigJson(value)}
        autosize={{ minRows: 20, maxRows: 30 }}
        placeholder="请输入MCP配置JSON"
        style={{ fontFamily: 'monospace', fontSize: 12 }}
      />
    </Modal>
  );
};
