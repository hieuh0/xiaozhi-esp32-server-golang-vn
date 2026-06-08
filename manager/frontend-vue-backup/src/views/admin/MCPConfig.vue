<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Trash2 } from '@lucide/vue'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'
import { Switch } from '@/components/ui/switch'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)

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
    merged.set(tool.name, { name: tool.name, description: tool.description || '' })
  })
  ;(server.allowed_tools || []).forEach((name) => {
    if (!name || merged.has(name)) return
    merged.set(name, { name, description: t('currently_selected') })
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

const enabledServerCount = computed(() => form.mcp.global.servers.filter(s => s.enabled).length)

const validate = () => {
  if (form.mcp.global.reconnect_interval < 1 || form.mcp.global.reconnect_interval > 3600) {
    ElMessage.error(t('reconnect_interval_range')); return false
  }
  if (form.mcp.global.max_reconnect_attempts < 1 || form.mcp.global.max_reconnect_attempts > 100) {
    ElMessage.error(t('max_reconnect_range_error')); return false
  }
  return true
}

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

const addGlobalServer = () => { form.mcp.global.servers.push(createGlobalServer()) }
const removeGlobalServer = (index) => { form.mcp.global.servers.splice(index, 1) }

const sanitizeGlobalServers = () => form.mcp.global.servers.map((server) => {
  const sanitized = { ...server }
  delete sanitized._tool_options
  delete sanitized._tools_loading
  return sanitized
})

const generateConfig = () => JSON.stringify({
  mcp: { global: { ...form.mcp.global, servers: sanitizeGlobalServers() } },
  local_mcp: { ...form.local_mcp }
})

const discoverGlobalServerTools = async (server) => {
  if (!server?.url) { ElMessage.warning(t('fill_server_url')); return }
  server._tools_loading = true
  try {
    const response = await api.post('/admin/mcp-configs/discover-tools', {
      transport: server.type,
      url: server.url,
      headers: server.headers || null
    })
    mergeServerToolOptions(server, response.data?.data?.tools || [])
    ElMessage.success(t('probing_tools_count', { count: server._tool_options.length }))
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
            servers: Array.isArray(configData.global?.servers) ? configData.global.servers.map(normalizeGlobalServer) : []
          }
        } else if (configData.mcp?.global) {
          form.mcp.global = {
            ...form.mcp.global,
            ...configData.mcp.global,
            servers: Array.isArray(configData.mcp.global?.servers) ? configData.mcp.global.servers.map(normalizeGlobalServer) : []
          }
        }
        if (configData.local_mcp) Object.assign(form.local_mcp, configData.local_mcp)
      } catch {
        ElMessage.warning(t('mcp_config_format_error'))
      }
    } else {
      configId.value = null
    }
  } catch {
    ElMessage.error(t('load_mcp_config_failed'))
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!validate()) return
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

const toggleAllowedTool = (server, toolName) => {
  const idx = server.allowed_tools.indexOf(toolName)
  if (idx >= 0) server.allowed_tools.splice(idx, 1)
  else server.allowed_tools.push(toolName)
}

onMounted(() => { loadConfig() })
</script>

<template>
  <div class="px-6 pb-8">
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <div v-else class="grid gap-6">
      <!-- Two-column layout -->
      <div class="grid gap-6" style="grid-template-columns: minmax(0, 1.45fr) minmax(340px, 0.9fr);">

        <!-- Global MCP card -->
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] flex flex-col">
          <div class="px-6 pt-6 pb-4 border-b border-[var(--color-line)] flex items-start justify-between gap-4">
            <div>
              <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-1">Global MCP</p>
              <h3 class="text-xl font-semibold tracking-tight text-[var(--color-text)] m-0">{{ t('global_mcp_service') }}</h3>
              <p class="text-sm text-[var(--color-text-secondary)] mt-1 m-0">{{ t('mcp_server_management_desc') }}</p>
            </div>
            <span :class="form.mcp.global.enabled
              ? 'inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800 shrink-0'
              : 'inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700 shrink-0'">
              {{ form.mcp.global.enabled ? t('enabled_server_count', { count: enabledServerCount }) : t('global_mcp_disabled') }}
            </span>
          </div>

          <div class="p-6 grid gap-5">
            <!-- Global switches/inputs -->
            <div class="grid gap-5" style="grid-template-columns: repeat(3, minmax(0,1fr));">
              <div class="col-span-3 flex items-start justify-between gap-4">
                <div>
                  <p class="text-sm font-semibold text-[var(--color-text)] m-0">{{ t('allow_mcp_server_connection') }}</p>
                  <p class="text-xs text-[var(--color-text-secondary)] mt-1 m-0">{{ t('allow_mcp_close_help') }}</p>
                </div>
                <Switch :checked="form.mcp.global.enabled" @update:checked="v => form.mcp.global.enabled = v" />
              </div>
              <div class="grid gap-1.5">
                <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('reconnect_interval_sec') }}</label>
                <NumberInput v-model="form.mcp.global.reconnect_interval" :min="1" :max="3600" :step="10" :precision="0" />
              </div>
              <div class="grid gap-1.5">
                <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('max_reconnect_attempts') }}</label>
                <NumberInput v-model="form.mcp.global.max_reconnect_attempts" :min="1" :max="100" :step="1" :precision="0" />
              </div>
            </div>

            <!-- Server list -->
            <div class="pt-5 border-t border-[var(--color-line)]">
              <div class="flex items-center justify-between mb-4">
                <h4 class="text-base font-semibold text-[var(--color-text)] m-0">{{ t('mcp_server_list_title') }}</h4>
                <Button size="sm" @click="addGlobalServer">
                  <Plus class="w-4 h-4 mr-1" />{{ t('add_server') }}
                </Button>
              </div>

              <!-- Empty state -->
              <div v-if="!form.mcp.global.servers.length" class="rounded-xl border border-dashed border-[var(--color-line)] bg-[var(--color-surface-muted)] p-5 text-center">
                <p class="font-semibold text-[var(--color-text)] m-0">{{ t('mcp_no_servers_title') }}</p>
                <p class="text-sm text-[var(--color-text-secondary)] mt-1 m-0">{{ t('mcp_no_servers_hint') }}</p>
              </div>

              <!-- Server items -->
              <div class="grid gap-4">
                <div v-for="(server, index) in form.mcp.global.servers" :key="index"
                  class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-muted)] p-4">
                  <!-- Server header -->
                  <div class="flex items-start justify-between gap-3 mb-4">
                    <div class="flex items-center flex-wrap gap-1.5">
                      <strong class="text-sm text-[var(--color-text)]">{{ t('server_label_n', { n: index + 1 }) }}</strong>
                      <span :class="server.enabled
                        ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
                        : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'">
                        {{ server.enabled ? t('enabled') : t('deactivated') }}
                      </span>
                      <span :class="server.allowed_tools?.length
                        ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800'
                        : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'">
                        {{ server.allowed_tools?.length ? t('tools_count', { count: server.allowed_tools.length }) : t('all_tools') }}
                      </span>
                    </div>
                    <div class="flex items-center gap-2 shrink-0">
                      <Button size="sm" variant="outline" :disabled="server._tools_loading" @click="discoverGlobalServerTools(server)">{{ t('probe_tools') }}</Button>
                      <Button size="sm" variant="outline" class="text-destructive border-destructive/30 hover:bg-destructive/10" @click="removeGlobalServer(index)">
                        <Trash2 class="w-3.5 h-3.5" />
                      </Button>
                    </div>
                  </div>

                  <!-- Server fields -->
                  <div class="grid gap-3" style="grid-template-columns: repeat(2, minmax(0,1fr));">
                    <div class="grid gap-1.5">
                      <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('server_name') }}</label>
                      <Input v-model="server.name" :placeholder="t('mcp_server_name_ph')" />
                    </div>
                    <div class="grid gap-1.5">
                      <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('server_type') }}</label>
                      <Select :model-value="server.type" @update:model-value="v => server.type = v">
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="sse">SSE</SelectItem>
                          <SelectItem value="streamablehttp">StreamableHTTP</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div class="col-span-2 grid gap-1.5">
                      <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('server_url') }}</label>
                      <Input v-model="server.url" :placeholder="t('mcp_server_url_ph')" />
                    </div>
                    <div class="col-span-2 flex items-center justify-between gap-4">
                      <div>
                        <p class="text-sm font-semibold text-[var(--color-text)] m-0">{{ t('allow_connect_server') }}</p>
                        <p class="text-xs text-[var(--color-text-secondary)] mt-0.5 m-0">{{ t('allow_connect_help') }}</p>
                      </div>
                      <Switch :checked="server.enabled" @update:checked="v => server.enabled = v" />
                    </div>
                  </div>

                  <!-- Allowed tools -->
                  <div v-if="server._tool_options.length > 0" class="mt-3 pt-3 border-t border-[var(--color-line)]">
                    <p class="text-xs text-[var(--color-text-secondary)] mb-2">{{ t('allowed_tools_hint') }}</p>
                    <div class="grid gap-1.5" style="grid-template-columns: repeat(auto-fill, minmax(160px,1fr));">
                      <label v-for="tool in server._tool_options" :key="tool.name"
                        class="flex items-start gap-2 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] p-2 cursor-pointer hover:bg-[var(--color-surface-muted)] transition-colors">
                        <input type="checkbox"
                          :checked="server.allowed_tools.includes(tool.name)"
                          class="mt-0.5 h-4 w-4 rounded accent-[var(--color-primary)] shrink-0"
                          @change="toggleAllowedTool(server, tool.name)" />
                        <div class="min-w-0">
                          <p class="text-xs font-semibold text-[var(--color-text)] truncate m-0">{{ tool.name }}</p>
                          <p v-if="tool.description" class="text-[10px] text-[var(--color-text-secondary)] truncate m-0">{{ tool.description }}</p>
                        </div>
                      </label>
                    </div>
                  </div>
                  <div v-else-if="server._tool_options.length === 0 && !server._tools_loading" class="mt-3 pt-3 border-t border-[var(--color-line)]">
                    <p class="text-xs text-[var(--color-text-secondary)]">{{ t('allowed_tools_hint') }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Local MCP card -->
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] h-fit">
          <div class="px-6 pt-6 pb-4 border-b border-[var(--color-line)]">
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-1">Local MCP</p>
            <h3 class="text-xl font-semibold tracking-tight text-[var(--color-text)] m-0">{{ t('local_mcp_capabilities') }}</h3>
            <p class="text-sm text-[var(--color-text-secondary)] mt-1 m-0">{{ t('local_mcp_desc') }}</p>
          </div>
          <div class="p-6 grid gap-5">
            <div class="flex items-start justify-between gap-4">
              <div>
                <p class="text-sm font-semibold text-[var(--color-text)] m-0">{{ t('allow_model_end_session') }}</p>
                <p class="text-xs text-[var(--color-text-secondary)] mt-1 m-0">{{ t('allow_model_end_help') }}</p>
              </div>
              <Switch :checked="form.local_mcp.exit_conversation" @update:checked="v => form.local_mcp.exit_conversation = v" />
            </div>
            <div class="flex items-start justify-between gap-4">
              <div>
                <p class="text-sm font-semibold text-[var(--color-text)] m-0">{{ t('allow_model_clear_context') }}</p>
                <p class="text-xs text-[var(--color-text-secondary)] mt-1 m-0">{{ t('allow_model_clear_help') }}</p>
              </div>
              <Switch :checked="form.local_mcp.clear_conversation_history" @update:checked="v => form.local_mcp.clear_conversation_history = v" />
            </div>
            <div class="flex items-start justify-between gap-4">
              <div>
                <p class="text-sm font-semibold text-[var(--color-text)] m-0">{{ t('allow_model_play_music') }}</p>
                <p class="text-xs text-[var(--color-text-secondary)] mt-1 m-0">{{ t('allow_model_play_help') }}</p>
              </div>
              <Switch :checked="form.local_mcp.play_music" @update:checked="v => form.local_mcp.play_music = v" />
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between gap-4 px-1">
        <p class="text-sm text-[var(--color-text-secondary)] max-w-[680px] m-0">{{ t('mcp_save_hint') }}</p>
        <div class="flex items-center gap-3 shrink-0">
          <Button variant="outline" :disabled="loading" @click="loadConfig">{{ t('reset_to_current') }}</Button>
          <Button :disabled="saving" @click="handleSave">{{ t('save_config') }}</Button>
        </div>
      </div>
    </div>
  </div>
</template>
