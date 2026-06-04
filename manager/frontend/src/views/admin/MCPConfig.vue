<template>
  <div class="mcp-config">
    <el-form ref="formRef" :model="form" :rules="rules" class="config-form" v-loading="loading">
      <div class="config-layout">
        <el-card class="config-card config-card-main" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Global MCP</p>
                <h3>{{ t('global_mcp_service') }}全局 MCP 服务</h3>
                <p class="card-description">{{ t('mcp_server_management_desc') }}维护服务端统一可用的 MCP 服务器、重连策略和允许工具范围。</p>
              </div>
              <el-tag :type="form.mcp.global.enabled ? 'success' : 'info'" effect="plain" round>
                {{ form.mcp.global.enabled ? `${enabledServerCount} 个启用服务` : t('global_mcp_disabled') }}
              </el-tag>
            </div>
          </template>

          <div class="field-grid field-grid-main">
            <el-form-item label="启用全局 MCP" prop="mcp.global.enabled">
              <div class="switch-field">
                <div>
                  <div class="switch-title">允许服务端统一连接 MCP</div>
                  <div class="field-help">关闭后不会主动建立全局 MCP 连接，但本地 MCP 仍可单独控制。</div>
                </div>
                <el-switch v-model="form.mcp.global.enabled" />
              </div>
            </el-form-item>

            <el-form-item label="重连间隔（秒）" prop="mcp.global.reconnect_interval">
              <el-input-number
                v-model="form.mcp.global.reconnect_interval"
                :min="1"
                :max="3600"
                controls-position="right"
                style="width: 100%"
              />
            </el-form-item>

            <el-form-item label="最大重连次数" prop="mcp.global.max_reconnect_attempts">
              <el-input-number
                v-model="form.mcp.global.max_reconnect_attempts"
                :min="1"
                :max="100"
                controls-position="right"
                style="width: 100%"
              />
            </el-form-item>
          </div>

          <div class="server-list">
            <div class="server-list-header">
              <div>
                <h4>服务器列表</h4>
                <p>每个服务器都可以单独启停、探测工具，并限制只暴露给主程序的工具集合。</p>
              </div>
              <el-button type="primary" @click="addGlobalServer">
                <el-icon><Plus /></el-icon>
                添加服务器
              </el-button>
            </div>

            <div v-if="form.mcp.global.servers.length === 0" class="empty-state">
              <strong>还没有 MCP 服务器</strong>
              <p>先添加一台服务器，再填写名称、类型和 URL；留空允许工具则表示该服务器的全部工具可用。</p>
            </div>

            <div v-for="(server, index) in form.mcp.global.servers" :key="index" class="server-item">
              <div class="server-item-header">
                <div class="server-title-row">
                  <strong>服务器 {{ index + 1 }}</strong>
                  <el-tag size="small" :type="server.enabled ? 'success' : 'info'" effect="plain" round>
                    {{ server.enabled ? t('enabled') : t('deactivated') }}
                  </el-tag>
                  <el-tag size="small" :type="server.allowed_tools?.length ? 'warning' : 'info'" effect="plain" round>
                    {{ server.allowed_tools?.length ? `${server.allowed_tools.length} 个工具` : t('all_tools') }}
                  </el-tag>
                </div>

                <div class="server-actions">
                  <el-button size="small" :loading="server._tools_loading" @click="discoverGlobalServerTools(server)">
                    探测工具
                  </el-button>
                  <el-button size="small" type="danger" @click="removeGlobalServer(index)">
                    <el-icon><Delete /></el-icon>
                    {{ t('delete') }}
                  </el-button>
                </div>
              </div>

              <div class="field-grid server-grid">
                <el-form-item :label="t('server_name')" :prop="`mcp.global.servers.${index}.name`">
                  <el-input v-model="server.name" placeholder="例如：Amap MCP" />
                </el-form-item>

                <el-form-item :label="t('server_type')" :prop="`mcp.global.servers.${index}.type`">
                  <el-select v-model="server.type" placeholder="选择服务器类型" style="width: 100%">
                    <el-option label="SSE" value="sse" />
                    <el-option label="StreamableHTTP" value="streamablehttp" />
                  </el-select>
                </el-form-item>

                <el-form-item :label="t('server_url')" :prop="`mcp.global.servers.${index}.url`" class="field-span-full">
                  <el-input v-model="server.url" placeholder="例如：https://example.com/mcp" />
                </el-form-item>

                <el-form-item :label="t('enabled_status')" :prop="`mcp.global.servers.${index}.enabled`">
                  <div class="switch-field">
                    <div>
                      <div class="switch-title">允许主程序连接该服务</div>
                      <div class="field-help">停用后该服务不会参与全局工具发现与调用。</div>
                    </div>
                    <el-switch v-model="server.enabled" />
                  </div>
                </el-form-item>
              </div>

              <el-form-item :label="t('allow_tools')" class="tool-form-item">
                <div class="tool-picker">
                  <div class="field-help">
                    留空表示允许该服务器的全部工具。探测工具时会使用当前填写的类型与 URL。
                  </div>
                  <el-select
                    v-model="server.allowed_tools"
                    multiple
                    filterable
                    clearable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    placeholder="不选择则允许全部工具"
                    :loading="server._tools_loading"
                  >
                    <el-option v-for="tool in server._tool_options" :key="tool.name" :label="tool.name" :value="tool.name">
                      <div class="tool-option-row">
                        <span class="tool-option-name">{{ tool.name }}</span>
                        <span class="tool-option-desc">{{ tool.description || t('no_description') }}</span>
                      </div>
                    </el-option>
                  </el-select>
                </div>
              </el-form-item>
            </div>
          </div>
        </el-card>

        <el-card class="config-card config-card-side" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Local MCP</p>
                <h3>本地 MCP 能力</h3>
                <p class="card-description">这些是主程序本地暴露给模型的基础能力开关，可以按场景逐项控制。</p>
              </div>
            </div>
          </template>

          <div class="field-stack">
            <el-form-item label="退出对话" prop="local_mcp.exit_conversation">
              <div class="switch-field">
                <div>
                  <div class="switch-title">允许模型结束当前会话</div>
                  <div class="field-help">适合需要主动收尾、关闭会话的工具链场景。</div>
                </div>
                <el-switch v-model="form.local_mcp.exit_conversation" />
              </div>
            </el-form-item>

            <el-form-item label="清除对话历史" prop="local_mcp.clear_conversation_history">
              <div class="switch-field">
                <div>
                  <div class="switch-title">允许模型清空当前上下文</div>
                  <div class="field-help">适合切换任务或重置上下文时主动调用。</div>
                </div>
                <el-switch v-model="form.local_mcp.clear_conversation_history" />
              </div>
            </el-form-item>

            <el-form-item label="播放音乐" prop="local_mcp.play_music">
              <div class="switch-field">
                <div>
                  <div class="switch-title">允许模型触发音乐播放</div>
                  <div class="field-help">如果你的产品场景不需要音频娱乐能力，可以关闭。</div>
                </div>
                <el-switch v-model="form.local_mcp.play_music" />
              </div>
            </el-form-item>
          </div>
        </el-card>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          保存后会更新默认 MCP 全局配置；如果某台服务器只希望暴露部分工具，请先探测工具后再限制允许列表。
        </p>
        <div class="footer-actions">
          <el-button plain :loading="loading" @click="loadConfig">重置为当前配置</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
        </div>
      </div>
    </el-form>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)
const formRef = ref()

const createDefaultState = () => ({
  mcp: {
    global: {
      enabled: true,
      servers: [],
      reconnect_interval: 300,
      max_reconnect_attempts: 10
    }
  },
  local_mcp: {
    exit_conversation: true,
    clear_conversation_history: true,
    play_music: false
  }
})

const form = reactive(createDefaultState())

const rules = {
  'mcp.global.reconnect_interval': [
    { required: true, message: t('enter_reconnect_interval'), trigger: 'blur' },
    { type: 'number', min: 1, max: 3600, message: t('reconnect_interval_range'), trigger: 'blur' }
  ],
  'mcp.global.max_reconnect_attempts': [
    { required: true, message: t('enter_max_reconnects'), trigger: 'blur' },
    { type: 'number', min: 1, max: 100, message: t('max_reconnect_range_error'), trigger: 'blur' }
  ]
}

const createGlobalServer = () => ({
  name: '',
  type: 'streamablehttp',
  url: '',
  enabled: true,
  allowed_tools: [],
  _tool_options: [],
  _tools_loading: false
})

const mergeServerToolOptions = (server, tools = []) => {
  const merged = new Map()

  ;(tools || []).forEach((tool) => {
    if (!tool?.name) return
    merged.set(tool.name, {
      name: tool.name,
      description: tool.description || ''
    })
  })

  ;(server.allowed_tools || []).forEach((name) => {
    if (!name || merged.has(name)) return
    merged.set(name, {
      name,
      description: t('currently_selected')
    })
  })

  server._tool_options = Array.from(merged.values()).sort((a, b) => a.name.localeCompare(b.name))
}

const normalizeGlobalServer = (server = {}) => {
  const normalized = {
    ...server,
    name: server.name || '',
    type: server.type || 'streamablehttp',
    url: server.url || '',
    enabled: server.enabled !== false,
    allowed_tools: Array.isArray(server.allowed_tools) ? [...server.allowed_tools] : [],
    _tool_options: [],
    _tools_loading: false
  }
  mergeServerToolOptions(normalized)
  return normalized
}

const enabledServerCount = computed(() => form.mcp.global.servers.filter(server => server.enabled).length)

const resetForm = () => {
  const defaults = createDefaultState()
  form.mcp.global.enabled = defaults.mcp.global.enabled
  form.mcp.global.reconnect_interval = defaults.mcp.global.reconnect_interval
  form.mcp.global.max_reconnect_attempts = defaults.mcp.global.max_reconnect_attempts
  form.mcp.global.servers = defaults.mcp.global.servers
  form.local_mcp.exit_conversation = defaults.local_mcp.exit_conversation
  form.local_mcp.clear_conversation_history = defaults.local_mcp.clear_conversation_history
  form.local_mcp.play_music = defaults.local_mcp.play_music
}

const addGlobalServer = () => {
  form.mcp.global.servers.push(createGlobalServer())
}

const removeGlobalServer = (index) => {
  form.mcp.global.servers.splice(index, 1)
}

const sanitizeGlobalServers = () => {
  return form.mcp.global.servers.map((server) => {
    const sanitized = { ...server }
    delete sanitized._tool_options
    delete sanitized._tools_loading
    return sanitized
  })
}

const generateConfig = () => {
  return JSON.stringify({
    mcp: {
      global: {
        ...form.mcp.global,
        servers: sanitizeGlobalServers()
      }
    },
    local_mcp: { ...form.local_mcp }
  })
}

const discoverGlobalServerTools = async (server) => {
  if (!server?.url) {
    ElMessage.warning(t('fill_server_url'))
    return
  }

  server._tools_loading = true
  try {
    const response = await api.post('/admin/mcp-configs/discover-tools', {
      transport: server.type,
      url: server.url,
      headers: server.headers || null
    })
    mergeServerToolOptions(server, response.data?.data?.tools || [])
    ElMessage.success(`探测到 ${server._tool_options.length} 个工具`)
  } catch (error) {
    mergeServerToolOptions(server)
    ElMessage.error(error.response?.data?.error || t('probe_tools_failed'))
  } finally {
    server._tools_loading = false
  }
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/mcp-configs')
    const configs = response.data?.data || []

    resetForm()

    if (configs.length > 0) {
      const config = configs.find(item => item.is_default) || configs[0]
      configId.value = config.id

      try {
        const configData = JSON.parse(config.json_data || '{}')
        if (configData.global && !configData.mcp) {
          form.mcp.global = {
            ...form.mcp.global,
            ...configData.global,
            servers: Array.isArray(configData.global?.servers)
              ? configData.global.servers.map(normalizeGlobalServer)
              : []
          }
        } else if (configData.mcp?.global) {
          form.mcp.global = {
            ...form.mcp.global,
            ...configData.mcp.global,
            servers: Array.isArray(configData.mcp.global?.servers)
              ? configData.mcp.global.servers.map(normalizeGlobalServer)
              : []
          }
        }

        if (configData.local_mcp) {
          Object.assign(form.local_mcp, configData.local_mcp)
        }
      } catch (error) {
        ElMessage.warning(t('mcp_config_format_error'))
      }
    } else {
      configId.value = null
    }
  } catch (error) {
    ElMessage.error(t('load_mcp_config_failed'))
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    const payload = {
      name: t('mcp_global_config'),
      config_id: 'mcp_global_config',
      is_default: true,
      json_data: generateConfig()
    }

    if (configId.value) {
      await api.put(`/admin/mcp-configs/${configId.value}`, payload)
      ElMessage.success(t('mcp_config_updated'))
    } else {
      const response = await api.post('/admin/mcp-configs', payload)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success(t('mcp_config_saved'))
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('save_mcp_failed'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.mcp-config {
  padding: 0 24px 32px;
}

.config-form {
  display: grid;
  gap: 24px;
}

.config-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(340px, 0.9fr);
  gap: 24px;
}

.config-card {
  border: 1px solid rgba(255, 255, 255, 0.88);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: var(--apple-shadow-md);
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.card-kicker {
  display: block;
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--apple-text-tertiary);
}

.card-head h3 {
  margin: 8px 0 0;
  font-size: 22px;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: var(--apple-text);
}

.card-description,
.field-help,
.footer-note,
.server-list-header p,
.empty-state p {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--apple-text-secondary);
}

.field-grid {
  display: grid;
  gap: 20px 18px;
}

.field-grid-main {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.field-stack {
  display: grid;
  gap: 20px;
}

.field-span-full {
  grid-column: 1 / -1;
}

.switch-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px 18px;
  align-items: center;
}

.switch-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--apple-text);
}

.server-list {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid rgba(229, 229, 234, 0.72);
}

.server-list-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 18px;
}

.server-list-header h4,
.empty-state strong {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--apple-text);
}

.empty-state {
  padding: 18px;
  border-radius: 18px;
  border: 1px dashed rgba(229, 229, 234, 0.9);
  background: rgba(248, 250, 252, 0.72);
}

.server-item {
  padding: 18px;
  border-radius: 18px;
  border: 1px solid rgba(229, 229, 234, 0.88);
  background: rgba(248, 250, 252, 0.82);
}

.server-item + .server-item {
  margin-top: 16px;
}

.server-item-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 18px;
}

.server-title-row,
.server-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.server-title-row strong {
  font-size: 15px;
  color: var(--apple-text);
}

.server-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.tool-form-item {
  margin-top: 16px;
}

.tool-picker {
  width: 100%;
}

.tool-option-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.35;
}

.tool-option-name {
  color: var(--apple-text);
}

.tool-option-desc {
  color: var(--apple-text-secondary);
  font-size: 12px;
}

.footer-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 0 4px;
}

.footer-note {
  max-width: 680px;
  margin: 0;
}

.footer-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
}

:deep(.el-card__header) {
  padding: 24px 24px 0;
  border-bottom: none;
  background: transparent;
}

:deep(.el-card__body) {
  padding: 24px;
}

:deep(.el-form-item) {
  margin-bottom: 0;
}

:deep(.el-form-item__label) {
  font-size: 14px;
  font-weight: 600;
  color: var(--apple-text);
}

@media (max-width: 1180px) {
  .config-layout,
  .field-grid-main {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .mcp-config {
    padding: 0 16px 24px;
  }

  :deep(.el-card__body) {
    padding: 20px;
  }

  :deep(.el-card__header) {
    padding: 20px 20px 0;
  }

  .server-list-header,
  .server-item-header,
  .footer-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .server-grid {
    grid-template-columns: 1fr;
  }

  .footer-actions {
    justify-content: stretch;
  }

  .footer-actions :deep(.el-button) {
    flex: 1;
  }
}
</style>
