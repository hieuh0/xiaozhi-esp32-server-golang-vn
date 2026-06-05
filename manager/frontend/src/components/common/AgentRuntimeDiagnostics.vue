<template>
  <div class="agent-runtime-diagnostics">
    <el-collapse v-model="activePanels" class="diagnostics-collapse">
      <el-collapse-item name="mcp">
        <template #title>
          <div class="collapse-title">
            <strong>{{ t('mcp_endpoint_debug') }}</strong>
            <span>{{ mcpSummaryText }}</span>
          </div>
        </template>

        <div class="diagnostic-section" v-loading="mcpLoading">
          <div class="status-row">
            <div>
              <div class="field-label">{{ t('agent_websocket') }}</div>
              <div class="status-inline">
                <el-tag :type="mcpStatusTagType">{{ mcpStatusText }}</el-tag>
                <span>{{ mcpStatusDetailText }}</span>
              </div>
            </div>
            <div class="action-row">
              <el-button size="small" @click="refreshMcpDebugInfo" :loading="mcpLoading">
                <el-icon><Refresh /></el-icon>{{ t('refresh_data') }}</el-button>
              <el-button size="small" type="primary" @click="copyMcpEndpoint" :disabled="!mcpEndpointData.endpoint">{{ t('copy_url') }}</el-button>
            </div>
          </div>

          <div class="result-block">
            <div class="field-label">{{ t('mcp_endpoint_url') }}</div>
            <pre class="code-box">{{ mcpEndpointData.endpoint || t('no_endpoints_save') }}</pre>
          </div>

          <div class="tool-header">
            <div class="field-label">{{ t('mcp_tool_list') }}</div>
            <el-button size="small" type="primary" @click="refreshMcpTools" :loading="toolsLoading">
              <el-icon><Refresh /></el-icon>{{ t('refresh_tool_list') }}</el-button>
          </div>
          <div v-if="mcpTools.length === 0" class="empty-box">{{ t('no_tool_data') }}</div>
          <div v-else class="tool-list">
            <el-tag
              v-for="tool in mcpTools"
              :key="tool.name"
              :type="tool.schema || tool.input_schema ? 'success' : 'info'"
              class="tool-tag"
            >
              {{ tool.name }}
            </el-tag>
          </div>

          <el-form label-position="top" class="diagnostic-form">
            <el-form-item :label="t('tool')">
              <el-select
                v-model="mcpCallForm.tool_name"
                :placeholder="t('select_tool')"
                style="width: 100%"
                filterable
                @change="handleMcpToolChange"
              >
                <el-option v-for="tool in mcpTools" :key="tool.name" :label="tool.name" :value="tool.name" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('args_json_label')">
              <el-input v-model="mcpCallForm.argumentsText" type="textarea" :rows="6" :placeholder="t('mcp_args_placeholder')" />
            </el-form-item>
          </el-form>
          <el-button type="primary" @click="callAgentMcpTool" :loading="callingTool">{{ t('call_tool') }}</el-button>
          <pre class="code-box result-box">{{ mcpCallResult || t('no_call_results') }}</pre>
        </div>
      </el-collapse-item>

      <el-collapse-item name="openclaw">
        <template #title>
          <div class="collapse-title">
            <strong>OpenClaw</strong>
            <span>{{ openClawSummaryText }}</span>
          </div>
        </template>

        <div class="diagnostic-section">
          <div class="status-row">
            <div>
              <div class="field-label">{{ t('connection_status') }}</div>
              <div class="status-inline">
                <el-tag :type="openClawStatusTagType">{{ openClawStatusText }}</el-tag>
                <span>{{ openClawEndpointData.status_message || t('role_command_realtime') }}</span>
              </div>
            </div>
            <div class="action-row">
              <el-link :href="openClawDocURL" target="_blank" type="primary" :underline="false">{{ t('view_docs') }}</el-link>
              <el-button size="small" @click="fetchOpenClawEndpoint" :loading="openClawEndpointLoading">{{ t('refresh_status') }}</el-button>
              <el-button size="small" type="primary" @click="copyOpenClawCommands" :disabled="!openClawCommandData.ready">{{ t('copy_command') }}</el-button>
            </div>
          </div>

          <div class="result-block" v-loading="openClawEndpointLoading">
            <div class="field-label">{{ t('openclaw_role_config_cmd') }}</div>
            <div v-if="openClawCommandData.ready" class="command-hint">{{ t('openclaw_execute_hint') }}</div>
            <div v-if="openClawCommandData.ready" class="command-steps">
              <div v-for="(step, index) in openClawCommandData.steps" :key="`${step.title}-${index}`" class="command-step">
                <div class="command-step-title">{{ t('step_line_label', { num: index + 1, title: step.title }) }}</div>
                <pre class="code-box">{{ step.command }}</pre>
              </div>
            </div>
            <pre v-else class="code-box">{{ openClawCommandDisplayText }}</pre>
          </div>

          <div class="result-block">
            <div class="field-label">{{ t('openclaw_chat_test') }}</div>
            <el-form label-position="top" class="diagnostic-form">
              <el-form-item :label="t('test_message_label')">
                <el-input
                  v-model="openClawChatTestForm.message"
                  type="textarea"
                  :rows="3"
                  :placeholder="t('openclaw_input_ph')"
                />
              </el-form-item>
            </el-form>
            <el-button type="primary" @click="testOpenClawChat" :loading="openClawChatTesting">{{ t('send_test_btn') }}</el-button>
            <pre class="code-box result-box">{{ openClawChatTestResult || t('no_test_results') }}</pre>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { postJSONWithSSE } from '../../utils/sse'
import { buildOpenClawCommands } from '../../utils/openclaw'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

const props = defineProps({
  agentId: {
    type: [Number, String],
    required: true
  },
  scope: {
    type: String,
    default: 'user'
  },
  defaultPanels: {
    type: Array,
    default: () => []
  },
  preloadStatus: {
    type: Boolean,
    default: false
  }
})

const activePanels = ref([...props.defaultPanels])
const mcpEndpointLoaded = ref(false)
const mcpLoaded = ref(false)
const openClawLoaded = ref(false)

const mcpLoading = ref(false)
const toolsLoading = ref(false)
const callingTool = ref(false)
const mcpEndpointData = ref({
  endpoint: '',
  connected: false,
  status: 'unknown',
  status_message: '',
  client_count: 0
})
const mcpTools = ref([])
const mcpCallResult = ref('')
const mcpCallForm = ref({ tool_name: '', argumentsText: '{}' })

const openClawEndpointLoading = ref(false)
const openClawEndpointData = ref({
  endpoint: '',
  connected: false,
  status: 'unknown',
  status_message: ''
})
const openClawChatTesting = ref(false)
const openClawChatTestResult = ref('')
const openClawChatTestForm = ref({ message: '' })

const safeScope = computed(() => (props.scope === 'admin' ? 'admin' : 'user'))
const agentPath = computed(() => `/${safeScope.value}/agents/${props.agentId}`)
const openClawDocURL = 'https://github.com/hackers365/xiaozhi-esp32-server-golang/blob/main/doc/openclaw_integration.md'

const mcpStatusText = computed(() => {
  const status = String(mcpEndpointData.value.status || '').toLowerCase()
  if (mcpEndpointData.value.connected || status === 'online') return t('connected')
  if (status === 'offline') return t('not_connected')
  return t('status_unknown')
})

const mcpStatusTagType = computed(() => {
  const status = String(mcpEndpointData.value.status || '').toLowerCase()
  if (mcpEndpointData.value.connected || status === 'online') return 'success'
  if (status === 'offline') return 'danger'
  return 'info'
})

const mcpStatusDetailText = computed(() => {
  const count = Number(mcpEndpointData.value.client_count || 0)
  if (count > 0) return t('clients_online_count', { count })
  return mcpEndpointData.value.status_message || t('no_online_clients')
})

const mcpSummaryText = computed(() => {
  if (!mcpEndpointLoaded.value) return t('expand_load_endpoints')
  if (!mcpLoaded.value) return t('expand_load_tools', { status: mcpStatusText.value })
  return t('tools_count_short', { status: mcpStatusText.value, count: mcpTools.value.length })
})

const openClawStatusText = computed(() => {
  const status = String(openClawEndpointData.value.status || '').toLowerCase()
  if (openClawEndpointData.value.connected || status === 'online') return t('connected')
  if (status === 'offline') return t('not_connected')
  return t('status_unknown')
})

const openClawStatusTagType = computed(() => {
  const status = String(openClawEndpointData.value.status || '').toLowerCase()
  if (openClawEndpointData.value.connected || status === 'online') return 'success'
  if (status === 'offline') return 'danger'
  return 'info'
})

const openClawCommandData = computed(() => buildOpenClawCommands(openClawEndpointData.value.endpoint))
const openClawCommandDisplayText = computed(() => {
  if (openClawCommandData.value.ready) return openClawCommandData.value.copyText
  if (!props.agentId) return t('no_install_command_save')
  return t('no_install_command_refresh')
})
const openClawSummaryText = computed(() => {
  if (!openClawLoaded.value) return t('expand_load_status')
  return openClawStatusText.value
})

const resetState = () => {
  mcpEndpointLoaded.value = false
  mcpLoaded.value = false
  openClawLoaded.value = false
  mcpEndpointData.value = {
    endpoint: '',
    connected: false,
    status: 'unknown',
    status_message: '',
    client_count: 0
  }
  mcpTools.value = []
  mcpCallResult.value = ''
  mcpCallForm.value = { tool_name: '', argumentsText: '{}' }
  openClawEndpointData.value = {
    endpoint: '',
    connected: false,
    status: 'unknown',
    status_message: ''
  }
  openClawChatTestResult.value = ''
  openClawChatTestForm.value = { message: '' }
}

const loadMcpEndpoint = async ({ showError = false } = {}) => {
  try {
    const response = await api.get(`${agentPath.value}/mcp-endpoint`)
    const data = response.data?.data || {}
    const status = String(data.status || '').trim().toLowerCase()
    const connected = !!data.connected
    mcpEndpointData.value = {
      endpoint: data.endpoint || '',
      connected,
      status: status || (connected ? 'online' : 'offline'),
      status_message: typeof data.status_message === 'string' ? data.status_message : '',
      client_count: Number(data.client_count || 0)
    }
    mcpEndpointLoaded.value = true
    return true
  } catch (error) {
    mcpEndpointData.value = {
      endpoint: '',
      connected: false,
      status: 'unknown',
      status_message: error.response?.data?.error || '',
      client_count: 0
    }
    mcpEndpointLoaded.value = true
    if (showError) ElMessage.error(error.response?.data?.error || t('get_mcp_endpoint_failed'))
    return false
  }
}

const refreshMcpTools = async () => {
  toolsLoading.value = true
  try {
    const response = await api.get(`${agentPath.value}/mcp-tools`)
    mcpTools.value = response.data?.data?.tools || []
    if (mcpTools.value.length > 0) {
      if (!mcpCallForm.value.tool_name) {
        mcpCallForm.value.tool_name = mcpTools.value[0].name
      }
      updateMcpExampleByTool(mcpCallForm.value.tool_name)
    }
  } catch (error) {
    mcpTools.value = []
    ElMessage.error(error.response?.data?.error || t('get_tool_list_failed'))
  } finally {
    toolsLoading.value = false
  }
}

const refreshMcpDebugInfo = async () => {
  mcpLoading.value = true
  mcpCallResult.value = ''
  try {
    const endpointLoaded = await loadMcpEndpoint({ showError: true })
    if (endpointLoaded) {
      await refreshMcpTools()
      mcpLoaded.value = true
    }
  } finally {
    mcpLoading.value = false
  }
}

const buildExampleFromSchema = (schema = {}) => {
  if (!schema || typeof schema !== 'object') return {}
  if (Array.isArray(schema.enum) && schema.enum.length > 0) return schema.enum[0]

  const type = schema.type || 'object'
  if (type === 'object') {
    const props = schema.properties || {}
    const result = {}
    Object.keys(props).sort().forEach((key) => {
      result[key] = buildExampleFromSchema(props[key])
    })
    return result
  }
  if (type === 'array') return [buildExampleFromSchema(schema.items || {})]
  if (type === 'number') return 0.1
  if (type === 'integer') return 0
  if (type === 'boolean') return false
  return ''
}

const updateMcpExampleByTool = (toolName) => {
  const selectedTool = mcpTools.value.find((item) => item.name === toolName)
  if (!selectedTool) return
  const schema = selectedTool.input_schema || selectedTool.schema || {}
  mcpCallForm.value.argumentsText = JSON.stringify(buildExampleFromSchema(schema), null, 2)
}

const handleMcpToolChange = (toolName) => updateMcpExampleByTool(toolName)

const formatMcpCallResult = (payload) => {
  const maxDepth = 8
  const tryParseJSONString = (value) => {
    if (typeof value !== 'string') return { parsed: false, value }
    let text = value.trim()
    if (!text) return { parsed: false, value }
    const fenced = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i)
    if (fenced) text = fenced[1].trim()
    const looksLikeJSON = (text.startsWith('{') && text.endsWith('}')) || (text.startsWith('[') && text.endsWith(']'))
    if (!looksLikeJSON) return { parsed: false, value }
    try {
      return { parsed: true, value: JSON.parse(text) }
    } catch (_) {
      return { parsed: false, value }
    }
  }

  const deepParseJSONStrings = (value, depth = 0) => {
    if (depth >= maxDepth || value == null) return value
    if (typeof value === 'string') {
      const parsed = tryParseJSONString(value)
      return parsed.parsed ? deepParseJSONStrings(parsed.value, depth + 1) : value
    }
    if (Array.isArray(value)) return value.map((item) => deepParseJSONStrings(item, depth + 1))
    if (typeof value === 'object') {
      const out = {}
      Object.keys(value).forEach((key) => {
        out[key] = deepParseJSONStrings(value[key], depth + 1)
      })
      if (Array.isArray(out.content) && out.content.length === 1) {
        const first = out.content[0]
        if (first && typeof first === 'object' && first.type === 'text' && Object.prototype.hasOwnProperty.call(first, 'text')) {
          return first.text && typeof first.text === 'object' ? first.text : out
        }
      }
      return out
    }
    return value
  }

  const data = payload ?? {}
  const raw = data && typeof data === 'object' && !Array.isArray(data) && Object.prototype.hasOwnProperty.call(data, 'result')
    ? data.result
    : data
  return JSON.stringify(deepParseJSONStrings(raw), null, 2)
}

const callAgentMcpTool = async () => {
  if (!mcpCallForm.value.tool_name) {
    ElMessage.warning(t('select_tool'))
    return
  }

  let argumentsObj = {}
  try {
    argumentsObj = mcpCallForm.value.argumentsText ? JSON.parse(mcpCallForm.value.argumentsText) : {}
  } catch (_) {
    ElMessage.error(t('params_json_format_error'))
    return
  }

  callingTool.value = true
  try {
    const response = await api.post(`${agentPath.value}/mcp-call`, {
      tool_name: mcpCallForm.value.tool_name,
      arguments: argumentsObj
    })
    mcpCallResult.value = formatMcpCallResult(response.data?.data || {})
    ElMessage.success(t('mcp_tool_call_success'))
  } catch (error) {
    mcpCallResult.value = JSON.stringify(error.response?.data || { error: error.message }, null, 2)
    ElMessage.error(t('mcp_tool_call_failed'))
  } finally {
    callingTool.value = false
  }
}

const copyMcpEndpoint = async () => {
  if (!mcpEndpointData.value.endpoint) {
    ElMessage.warning(t('no_mcp_endpoint_to_copy'))
    return
  }
  try {
    await navigator.clipboard.writeText(mcpEndpointData.value.endpoint)
    ElMessage.success(t('mcp_endpoint_url_copied'))
  } catch (_) {
    ElMessage.error(t('copy_failed'))
  }
}

const fetchOpenClawEndpoint = async ({ showError = true } = {}) => {
  openClawEndpointLoading.value = true
  try {
    const response = await api.get(`${agentPath.value}/openclaw-endpoint`)
    const data = response.data?.data || {}
    const status = String(data.status || '').trim().toLowerCase()
    const connected = !!data.connected
    openClawEndpointData.value = {
      endpoint: data.endpoint || '',
      connected,
      status: status || (connected ? 'online' : 'offline'),
      status_message: typeof data.status_message === 'string' ? data.status_message : ''
    }
    openClawLoaded.value = true
  } catch (error) {
    openClawEndpointData.value = {
      endpoint: '',
      connected: false,
      status: 'unknown',
      status_message: error.response?.data?.error || ''
    }
    if (showError) ElMessage.error(error.response?.data?.error || t('get_openclaw_endpoint_failed'))
  } finally {
    openClawEndpointLoading.value = false
  }
}

const copyOpenClawCommands = async () => {
  const commands = openClawCommandData.value.copyText
  if (!commands) {
    ElMessage.warning(t('no_openclaw_config_to_copy'))
    return
  }
  try {
    await navigator.clipboard.writeText(commands)
    ElMessage.success(t('openclaw_role_config_copied'))
  } catch (_) {
    ElMessage.error(t('copy_failed_manual'))
  }
}

const formatOpenClawChatResult = (reply, latency) => {
  const lines = [t('reply_prefix', { text: String(reply || '') || t('empty_parens') })]
  if (Number.isFinite(latency)) lines.push(t('elapsed_prefix', { ms: latency }))
  return lines.join('\n')
}

const testOpenClawChat = async () => {
  const message = String(openClawChatTestForm.value.message || '').trim()
  if (!message) {
    ElMessage.warning(t('enter_test_message'))
    return
  }

  openClawChatTesting.value = true
  openClawChatTestResult.value = t('connecting')
  try {
    const requestTimeoutMs = 610000
    const timeoutMs = 600000
    const token = String(localStorage.getItem('token') || '')
    const chunks = []
    let finalData = null
    let streamError = ''
    const normalizePayload = (payload) => (payload && typeof payload === 'object' ? payload : {})

    const response = await postJSONWithSSE({
      url: `/api/${safeScope.value}/agents/${props.agentId}/openclaw-chat-test?stream=1`,
      body: { message, timeout_ms: timeoutMs },
      timeoutMs: requestTimeoutMs,
      token,
      onEvent: (event, payload) => {
        const envelope = normalizePayload(payload)
        if (event === 'start') {
          openClawChatTestResult.value = t('connected_waiting')
          return
        }
        if (event === 'chunk') {
          const data = normalizePayload(envelope.data)
          const chunk = typeof data.chunk === 'string' ? data.chunk : ''
          if (chunk) chunks.push(chunk)
          const reply = String(data.reply || chunks.join(''))
          const latency = Number(data.latency_ms)
          openClawChatTestResult.value = `${t('stream_reply')}\n${formatOpenClawChatResult(reply, latency)}`
          return
        }
        if (event === 'result') {
          finalData = normalizePayload(envelope.data)
          const reply = String(finalData.reply || chunks.join(''))
          const latency = Number(finalData.latency_ms)
          openClawChatTestResult.value = formatOpenClawChatResult(reply, latency)
          return
        }
        if (event === 'error') {
          const data = normalizePayload(envelope.data)
          const messageText = String(envelope.error || data.error || t('openclaw_test_failed'))
          const partialReply = String(data.reply || chunks.join(''))
          streamError = messageText
          openClawChatTestResult.value = partialReply
            ? t('error_received_msg', { msg: messageText, text: partialReply })
            : t('error_prefix', { msg: messageText })
          return
        }
        if (event === 'done') {
          if (!finalData) finalData = normalizePayload(envelope.data)
          if (envelope.ok === false && !streamError) streamError = t('openclaw_test_failed')
        }
      }
    })

    if (response.mode === 'json') {
      const data = response.payload?.data || {}
      const reply = String(data.reply || '')
      const latency = Number(data.latency_ms)
      openClawChatTestResult.value = formatOpenClawChatResult(reply, latency)
      ElMessage.success(t('openclaw_chat_test_success'))
      return
    }

    if (streamError) throw new Error(streamError)
    if (finalData && typeof finalData === 'object') {
      const reply = String(finalData.reply || chunks.join(''))
      const latency = Number(finalData.latency_ms)
      openClawChatTestResult.value = formatOpenClawChatResult(reply, latency)
    } else if (chunks.length > 0) {
      openClawChatTestResult.value = formatOpenClawChatResult(chunks.join(''), Number.NaN)
    } else {
      throw new Error(t('openclaw_no_response'))
    }
    ElMessage.success(t('openclaw_chat_test_success'))
  } catch (error) {
    const msg = error.response?.data?.error || error.message || t('openclaw_test_failed')
    openClawChatTestResult.value = t('error_prefix', { msg })
    ElMessage.error(msg)
  } finally {
    openClawChatTesting.value = false
    await fetchOpenClawEndpoint({ showError: false })
  }
}

watch(
  () => props.defaultPanels,
  (panels) => {
    activePanels.value = [...panels]
  },
  { deep: true }
)

watch(
  () => props.agentId,
  () => {
    resetState()
    if (props.preloadStatus) {
      void preloadRuntimeStatus()
    }
  }
)

watch(
  activePanels,
  async (panels) => {
    const active = Array.isArray(panels) ? panels : [panels]
    if (active.includes('mcp') && !mcpLoaded.value && !mcpLoading.value) {
      await refreshMcpDebugInfo()
    }
    if (active.includes('openclaw') && !openClawLoaded.value && !openClawEndpointLoading.value) {
      await fetchOpenClawEndpoint({ showError: false })
    }
  },
  { immediate: true }
)

const preloadRuntimeStatus = async () => {
  await Promise.all([
    loadMcpEndpoint({ showError: false }),
    fetchOpenClawEndpoint({ showError: false })
  ])
}

onMounted(() => {
  if (props.preloadStatus) {
    void preloadRuntimeStatus()
  }
})
</script>

<style scoped>
.agent-runtime-diagnostics {
  width: 100%;
}

.diagnostics-collapse {
  border-top: 1px solid #ebeef5;
  border-bottom: 1px solid #ebeef5;
}

.collapse-title {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.collapse-title strong {
  color: #303133;
  font-size: 14px;
}

.collapse-title span {
  color: #909399;
  font-size: 12px;
}

.diagnostic-section {
  display: grid;
  gap: 16px;
  padding: 4px 0 10px;
}

.status-row,
.tool-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.status-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  color: #606266;
  font-size: 13px;
}

.action-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.field-label {
  color: #374151;
  font-size: 14px;
  font-weight: 600;
}

.result-block,
.diagnostic-form {
  min-width: 0;
}

.code-box {
  margin: 8px 0 0;
  min-height: 48px;
  padding: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  color: #1f2937;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-all;
}

.result-box {
  margin-top: 12px;
}

.empty-box {
  padding: 14px;
  border: 1px dashed #dcdfe6;
  border-radius: 8px;
  color: #909399;
  text-align: center;
}

.tool-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tool-tag {
  border-radius: 8px;
}

.command-hint {
  margin-top: 8px;
  color: #606266;
  font-size: 13px;
}

.command-steps {
  display: grid;
  gap: 10px;
  margin-top: 8px;
}

.command-step-title {
  color: #374151;
  font-size: 13px;
  font-weight: 600;
}

@media (max-width: 760px) {
  .status-row,
  .tool-header {
    align-items: stretch;
    flex-direction: column;
  }

  .action-row {
    justify-content: flex-start;
  }
}
</style>
