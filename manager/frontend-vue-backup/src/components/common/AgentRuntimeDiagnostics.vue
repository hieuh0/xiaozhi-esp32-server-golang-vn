<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { RefreshCw, ChevronDown } from '@lucide/vue'
import api from '../../utils/api'
import { postJSONWithSSE } from '../../utils/sse'
import { buildOpenClawCommands } from '../../utils/openclaw'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const props = defineProps({
  agentId: { type: [Number, String], required: true },
  scope: { type: String, default: 'user' },
  defaultPanels: { type: Array, default: () => [] },
  preloadStatus: { type: Boolean, default: false }
})

const activePanels = ref([...props.defaultPanels])
const mcpEndpointLoaded = ref(false)
const mcpLoaded = ref(false)
const openClawLoaded = ref(false)

const mcpLoading = ref(false)
const toolsLoading = ref(false)
const callingTool = ref(false)
const mcpEndpointData = ref({ endpoint: '', connected: false, status: 'unknown', status_message: '', client_count: 0 })
const mcpTools = ref([])
const mcpCallResult = ref('')
const mcpCallForm = ref({ tool_name: '', argumentsText: '{}' })

const openClawEndpointLoading = ref(false)
const openClawEndpointData = ref({ endpoint: '', connected: false, status: 'unknown', status_message: '' })
const openClawChatTesting = ref(false)
const openClawChatTestResult = ref('')
const openClawChatTestForm = ref({ message: '' })

const safeScope = computed(() => (props.scope === 'admin' ? 'admin' : 'user'))
const agentPath = computed(() => `/${safeScope.value}/agents/${props.agentId}`)
const openClawDocURL = 'https://github.com/hackers365/xiaozhi-esp32-server-golang/blob/main/doc/openclaw_integration.md'

const statusTag = (connected, status) => {
  const s = String(status || '').toLowerCase()
  if (connected || s === 'online') return 'inline-flex items-center px-2 py-0.5 rounded text-xs bg-green-100 text-green-700 border border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800'
  if (s === 'offline') return 'inline-flex items-center px-2 py-0.5 rounded text-xs bg-red-100 text-red-600 border border-red-200 dark:bg-red-900/30 dark:text-red-400 dark:border-red-800'
  return 'inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-[var(--color-text-secondary)]'
}

const mcpStatusText = computed(() => {
  const s = String(mcpEndpointData.value.status || '').toLowerCase()
  if (mcpEndpointData.value.connected || s === 'online') return t('connected')
  if (s === 'offline') return t('not_connected')
  return t('status_unknown')
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
  const s = String(openClawEndpointData.value.status || '').toLowerCase()
  if (openClawEndpointData.value.connected || s === 'online') return t('connected')
  if (s === 'offline') return t('not_connected')
  return t('status_unknown')
})

const openClawCommandData = computed(() => buildOpenClawCommands(openClawEndpointData.value.endpoint))
const openClawCommandDisplayText = computed(() => {
  if (openClawCommandData.value.ready) return openClawCommandData.value.copyText
  if (!props.agentId) return t('no_install_command_save')
  return t('no_install_command_refresh')
})
const openClawSummaryText = computed(() => !openClawLoaded.value ? t('expand_load_status') : openClawStatusText.value)

const togglePanel = (name) => {
  const idx = activePanels.value.indexOf(name)
  if (idx >= 0) activePanels.value.splice(idx, 1)
  else activePanels.value.push(name)
}
const isPanelOpen = (name) => activePanels.value.includes(name)

const resetState = () => {
  mcpEndpointLoaded.value = false; mcpLoaded.value = false; openClawLoaded.value = false
  mcpEndpointData.value = { endpoint: '', connected: false, status: 'unknown', status_message: '', client_count: 0 }
  mcpTools.value = []; mcpCallResult.value = ''; mcpCallForm.value = { tool_name: '', argumentsText: '{}' }
  openClawEndpointData.value = { endpoint: '', connected: false, status: 'unknown', status_message: '' }
  openClawChatTestResult.value = ''; openClawChatTestForm.value = { message: '' }
}

const loadMcpEndpoint = async ({ showError = false } = {}) => {
  try {
    const res = await api.get(`${agentPath.value}/mcp-endpoint`)
    const data = res.data?.data || {}
    const status = String(data.status || '').trim().toLowerCase()
    const connected = !!data.connected
    mcpEndpointData.value = { endpoint: data.endpoint || '', connected, status: status || (connected ? 'online' : 'offline'), status_message: typeof data.status_message === 'string' ? data.status_message : '', client_count: Number(data.client_count || 0) }
    mcpEndpointLoaded.value = true; return true
  } catch (error) {
    mcpEndpointData.value = { endpoint: '', connected: false, status: 'unknown', status_message: error.response?.data?.error || '', client_count: 0 }
    mcpEndpointLoaded.value = true
    if (showError) ElMessage.error(error.response?.data?.error || t('get_mcp_endpoint_failed'))
    return false
  }
}

const buildExampleFromSchema = (schema = {}) => {
  if (!schema || typeof schema !== 'object') return {}
  if (Array.isArray(schema.enum) && schema.enum.length > 0) return schema.enum[0]
  const type = schema.type || 'object'
  if (type === 'object') {
    const result = {}
    Object.keys(schema.properties || {}).sort().forEach((key) => { result[key] = buildExampleFromSchema(schema.properties[key]) })
    return result
  }
  if (type === 'array') return [buildExampleFromSchema(schema.items || {})]
  if (type === 'number') return 0.1
  if (type === 'integer') return 0
  if (type === 'boolean') return false
  return ''
}

const updateMcpExampleByTool = (toolName) => {
  const tool = mcpTools.value.find((i) => i.name === toolName)
  if (!tool) return
  mcpCallForm.value.argumentsText = JSON.stringify(buildExampleFromSchema(tool.input_schema || tool.schema || {}), null, 2)
}

const handleMcpToolChange = (toolName) => updateMcpExampleByTool(toolName)

const refreshMcpTools = async () => {
  toolsLoading.value = true
  try {
    const res = await api.get(`${agentPath.value}/mcp-tools`)
    mcpTools.value = res.data?.data?.tools || []
    if (mcpTools.value.length > 0) {
      if (!mcpCallForm.value.tool_name) mcpCallForm.value.tool_name = mcpTools.value[0].name
      updateMcpExampleByTool(mcpCallForm.value.tool_name)
    }
  } catch (error) {
    mcpTools.value = []; ElMessage.error(error.response?.data?.error || t('get_tool_list_failed'))
  } finally {
    toolsLoading.value = false
  }
}

const refreshMcpDebugInfo = async () => {
  mcpLoading.value = true; mcpCallResult.value = ''
  try {
    const ok = await loadMcpEndpoint({ showError: true })
    if (ok) { await refreshMcpTools(); mcpLoaded.value = true }
  } finally { mcpLoading.value = false }
}

const formatMcpCallResult = (payload) => {
  const tryParse = (v) => { if (typeof v !== 'string') return { parsed: false, value: v }; let text = v.trim(); const m = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i); if (m) text = m[1].trim(); if (!((text.startsWith('{') && text.endsWith('}')) || (text.startsWith('[') && text.endsWith(']')))) return { parsed: false, value: v }; try { return { parsed: true, value: JSON.parse(text) } } catch { return { parsed: false, value: v } } }
  const deep = (v, d = 0) => { if (d >= 8 || v == null) return v; if (typeof v === 'string') { const r = tryParse(v); return r.parsed ? deep(r.value, d + 1) : v }; if (Array.isArray(v)) return v.map((i) => deep(i, d + 1)); if (typeof v === 'object') { const out = {}; Object.keys(v).forEach((k) => { out[k] = deep(v[k], d + 1) }); if (Array.isArray(out.content) && out.content.length === 1) { const f = out.content[0]; if (f?.type === 'text' && Object.prototype.hasOwnProperty.call(f, 'text')) return f.text && typeof f.text === 'object' ? f.text : out }; return out }; return v }
  const raw = payload && typeof payload === 'object' && !Array.isArray(payload) && Object.prototype.hasOwnProperty.call(payload, 'result') ? payload.result : payload
  return JSON.stringify(deep(raw), null, 2)
}

const callAgentMcpTool = async () => {
  if (!mcpCallForm.value.tool_name) { ElMessage.warning(t('select_tool')); return }
  let argumentsObj = {}
  try { argumentsObj = mcpCallForm.value.argumentsText ? JSON.parse(mcpCallForm.value.argumentsText) : {} } catch { ElMessage.error(t('params_json_format_error')); return }
  callingTool.value = true
  try {
    const res = await api.post(`${agentPath.value}/mcp-call`, { tool_name: mcpCallForm.value.tool_name, arguments: argumentsObj })
    mcpCallResult.value = formatMcpCallResult(res.data?.data || {}); ElMessage.success(t('mcp_tool_call_success'))
  } catch (error) {
    mcpCallResult.value = JSON.stringify(error.response?.data || { error: error.message }, null, 2); ElMessage.error(t('mcp_tool_call_failed'))
  } finally { callingTool.value = false }
}

const copyMcpEndpoint = async () => {
  if (!mcpEndpointData.value.endpoint) { ElMessage.warning(t('no_mcp_endpoint_to_copy')); return }
  try { await navigator.clipboard.writeText(mcpEndpointData.value.endpoint); ElMessage.success(t('mcp_endpoint_url_copied')) } catch { ElMessage.error(t('copy_failed')) }
}

const fetchOpenClawEndpoint = async ({ showError = true } = {}) => {
  openClawEndpointLoading.value = true
  try {
    const res = await api.get(`${agentPath.value}/openclaw-endpoint`)
    const data = res.data?.data || {}; const s = String(data.status || '').trim().toLowerCase(); const connected = !!data.connected
    openClawEndpointData.value = { endpoint: data.endpoint || '', connected, status: s || (connected ? 'online' : 'offline'), status_message: typeof data.status_message === 'string' ? data.status_message : '' }
    openClawLoaded.value = true
  } catch (error) {
    openClawEndpointData.value = { endpoint: '', connected: false, status: 'unknown', status_message: error.response?.data?.error || '' }
    if (showError) ElMessage.error(error.response?.data?.error || t('get_openclaw_endpoint_failed'))
  } finally { openClawEndpointLoading.value = false }
}

const copyOpenClawCommands = async () => {
  const cmds = openClawCommandData.value.copyText
  if (!cmds) { ElMessage.warning(t('no_openclaw_config_to_copy')); return }
  try { await navigator.clipboard.writeText(cmds); ElMessage.success(t('openclaw_role_config_copied')) } catch { ElMessage.error(t('copy_failed_manual')) }
}

const formatOpenClawChatResult = (reply, latency) => {
  const lines = [t('reply_prefix', { text: String(reply || '') || t('empty_parens') })]
  if (Number.isFinite(latency)) lines.push(t('elapsed_prefix', { ms: latency }))
  return lines.join('\n')
}

const testOpenClawChat = async () => {
  const message = String(openClawChatTestForm.value.message || '').trim()
  if (!message) { ElMessage.warning(t('enter_test_message')); return }
  openClawChatTesting.value = true; openClawChatTestResult.value = t('connecting')
  try {
    const timeoutMs = 600000; const token = String(localStorage.getItem('token') || '')
    const chunks = []; let finalData = null; let streamError = ''
    const norm = (p) => (p && typeof p === 'object' ? p : {})
    const response = await postJSONWithSSE({ url: `/api/${safeScope.value}/agents/${props.agentId}/openclaw-chat-test?stream=1`, body: { message, timeout_ms: timeoutMs }, timeoutMs: 610000, token,
      onEvent: (event, payload) => {
        const env = norm(payload)
        if (event === 'start') { openClawChatTestResult.value = t('connected_waiting'); return }
        if (event === 'chunk') { const data = norm(env.data); const chunk = typeof data.chunk === 'string' ? data.chunk : ''; if (chunk) chunks.push(chunk); openClawChatTestResult.value = `${t('stream_reply')}\n${formatOpenClawChatResult(String(data.reply || chunks.join('')), Number(data.latency_ms))}`; return }
        if (event === 'result') { finalData = norm(env.data); openClawChatTestResult.value = formatOpenClawChatResult(String(finalData.reply || chunks.join('')), Number(finalData.latency_ms)); return }
        if (event === 'error') { const data = norm(env.data); streamError = String(env.error || data.error || t('openclaw_test_failed')); openClawChatTestResult.value = String(data.reply || chunks.join('')) ? t('error_received_msg', { msg: streamError, text: String(data.reply || chunks.join('')) }) : t('error_prefix', { msg: streamError }); return }
        if (event === 'done') { if (!finalData) finalData = norm(env.data); if (env.ok === false && !streamError) streamError = t('openclaw_test_failed') }
      }
    })
    if (response.mode === 'json') { const data = response.payload?.data || {}; openClawChatTestResult.value = formatOpenClawChatResult(String(data.reply || ''), Number(data.latency_ms)); ElMessage.success(t('openclaw_chat_test_success')); return }
    if (streamError) throw new Error(streamError)
    if (finalData) openClawChatTestResult.value = formatOpenClawChatResult(String(finalData.reply || chunks.join('')), Number(finalData.latency_ms))
    else if (chunks.length) openClawChatTestResult.value = formatOpenClawChatResult(chunks.join(''), Number.NaN)
    else throw new Error(t('openclaw_no_response'))
    ElMessage.success(t('openclaw_chat_test_success'))
  } catch (error) {
    const msg = error.response?.data?.error || error.message || t('openclaw_test_failed')
    openClawChatTestResult.value = t('error_prefix', { msg }); ElMessage.error(msg)
  } finally {
    openClawChatTesting.value = false
    await fetchOpenClawEndpoint({ showError: false })
  }
}

watch(() => props.defaultPanels, (panels) => { activePanels.value = [...panels] }, { deep: true })
watch(() => props.agentId, () => { resetState(); if (props.preloadStatus) void preloadRuntimeStatus() })
watch(activePanels, async (panels) => {
  const active = Array.isArray(panels) ? panels : [panels]
  if (active.includes('mcp') && !mcpLoaded.value && !mcpLoading.value) await refreshMcpDebugInfo()
  if (active.includes('openclaw') && !openClawLoaded.value && !openClawEndpointLoading.value) await fetchOpenClawEndpoint({ showError: false })
}, { immediate: true })

const preloadRuntimeStatus = async () => {
  await Promise.all([loadMcpEndpoint({ showError: false }), fetchOpenClawEndpoint({ showError: false })])
}

onMounted(() => { if (props.preloadStatus) void preloadRuntimeStatus() })
</script>

<template>
  <div class="w-full grid gap-1">
    <!-- MCP panel -->
    <div class="border border-[var(--color-line)] rounded-xl overflow-hidden">
      <button type="button" class="w-full flex items-center justify-between gap-3 px-4 py-3 text-left bg-[var(--color-surface)] hover:bg-[var(--color-surface-muted)] transition-colors" @click="togglePanel('mcp')">
        <div class="flex items-center gap-2 min-w-0">
          <strong class="text-sm font-semibold text-[var(--color-text)]">{{ t('mcp_endpoint_debug') }}</strong>
          <span class="text-xs text-[var(--color-text-tertiary)] truncate">{{ mcpSummaryText }}</span>
        </div>
        <ChevronDown :class="['w-4 h-4 text-[var(--color-text-tertiary)] transition-transform shrink-0', isPanelOpen('mcp') && 'rotate-180']" />
      </button>
      <div v-show="isPanelOpen('mcp')" :class="['border-t border-[var(--color-line)] p-4 grid gap-4', mcpLoading && 'opacity-60 pointer-events-none']">
        <!-- Status row -->
        <div class="flex items-start justify-between gap-3 flex-wrap">
          <div>
            <p class="text-xs font-semibold text-[var(--color-text-secondary)] mb-1">{{ t('agent_websocket') }}</p>
            <div class="flex items-center gap-2">
              <span :class="statusTag(mcpEndpointData.connected, mcpEndpointData.status)">{{ mcpStatusText }}</span>
              <span class="text-xs text-[var(--color-text-secondary)]">{{ mcpStatusDetailText }}</span>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-wrap">
            <Button size="sm" variant="outline" :disabled="mcpLoading" @click="refreshMcpDebugInfo">
              <RefreshCw class="w-3 h-3 mr-1" />{{ t('refresh_data') }}
            </Button>
            <Button size="sm" variant="outline" :disabled="!mcpEndpointData.endpoint" @click="copyMcpEndpoint">{{ t('copy_url') }}</Button>
          </div>
        </div>
        <!-- Endpoint URL -->
        <div>
          <p class="text-xs font-semibold text-[var(--color-text)] mb-1.5">{{ t('mcp_endpoint_url') }}</p>
          <pre class="min-h-[40px] p-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-xs font-mono whitespace-pre-wrap break-all">{{ mcpEndpointData.endpoint || t('no_endpoints_save') }}</pre>
        </div>
        <!-- Tools -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <p class="text-xs font-semibold text-[var(--color-text)]">{{ t('mcp_tool_list') }}</p>
            <Button size="sm" variant="outline" :disabled="toolsLoading" @click="refreshMcpTools">
              <RefreshCw class="w-3 h-3 mr-1" />{{ t('refresh_tool_list') }}
            </Button>
          </div>
          <div v-if="!mcpTools.length" class="p-3 border border-dashed border-[var(--color-line)] rounded-lg text-xs text-[var(--color-text-tertiary)] text-center">{{ t('no_tool_data') }}</div>
          <div v-else class="flex flex-wrap gap-1.5">
            <span v-for="tool in mcpTools" :key="tool.name" :class="['inline-flex items-center px-2 py-0.5 rounded text-xs border', tool.schema || tool.input_schema ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800' : 'bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)] border-[var(--color-line)]']">{{ tool.name }}</span>
          </div>
        </div>
        <!-- Tool call form -->
        <div class="grid gap-3">
          <div class="grid gap-1.5">
            <label class="text-xs font-semibold text-[var(--color-text)]">{{ t('tool') }}</label>
            <Select v-model="mcpCallForm.tool_name" @update:model-value="handleMcpToolChange">
              <SelectTrigger class="w-full"><SelectValue :placeholder="t('select_tool')" /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="tool in mcpTools" :key="tool.name" :value="tool.name">{{ tool.name }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid gap-1.5">
            <label class="text-xs font-semibold text-[var(--color-text)]">{{ t('args_json_label') }}</label>
            <Textarea v-model="mcpCallForm.argumentsText" :placeholder="t('mcp_args_placeholder')" rows="6" class="font-mono text-xs" />
          </div>
          <Button :disabled="callingTool" @click="callAgentMcpTool">{{ t('call_tool') }}</Button>
          <pre class="min-h-[60px] p-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-xs font-mono whitespace-pre-wrap break-all">{{ mcpCallResult || t('no_call_results') }}</pre>
        </div>
      </div>
    </div>

    <!-- OpenClaw panel -->
    <div class="border border-[var(--color-line)] rounded-xl overflow-hidden">
      <button type="button" class="w-full flex items-center justify-between gap-3 px-4 py-3 text-left bg-[var(--color-surface)] hover:bg-[var(--color-surface-muted)] transition-colors" @click="togglePanel('openclaw')">
        <div class="flex items-center gap-2 min-w-0">
          <strong class="text-sm font-semibold text-[var(--color-text)]">OpenClaw</strong>
          <span class="text-xs text-[var(--color-text-tertiary)] truncate">{{ openClawSummaryText }}</span>
        </div>
        <ChevronDown :class="['w-4 h-4 text-[var(--color-text-tertiary)] transition-transform shrink-0', isPanelOpen('openclaw') && 'rotate-180']" />
      </button>
      <div v-show="isPanelOpen('openclaw')" class="border-t border-[var(--color-line)] p-4 grid gap-4">
        <!-- Status row -->
        <div class="flex items-start justify-between gap-3 flex-wrap">
          <div>
            <p class="text-xs font-semibold text-[var(--color-text-secondary)] mb-1">{{ t('connection_status') }}</p>
            <div class="flex items-center gap-2">
              <span :class="statusTag(openClawEndpointData.connected, openClawEndpointData.status)">{{ openClawStatusText }}</span>
              <span class="text-xs text-[var(--color-text-secondary)]">{{ openClawEndpointData.status_message || t('role_command_realtime') }}</span>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-wrap">
            <a :href="openClawDocURL" target="_blank" class="text-xs text-[var(--color-primary)] hover:underline">{{ t('view_docs') }}</a>
            <Button size="sm" variant="outline" :disabled="openClawEndpointLoading" @click="fetchOpenClawEndpoint">{{ t('refresh_status') }}</Button>
            <Button size="sm" variant="outline" :disabled="!openClawCommandData.ready" @click="copyOpenClawCommands">{{ t('copy_command') }}</Button>
          </div>
        </div>
        <!-- Commands -->
        <div :class="openClawEndpointLoading && 'opacity-60'">
          <p class="text-xs font-semibold text-[var(--color-text)] mb-1.5">{{ t('openclaw_role_config_cmd') }}</p>
          <p v-if="openClawCommandData.ready" class="text-xs text-[var(--color-text-secondary)] mb-2">{{ t('openclaw_execute_hint') }}</p>
          <div v-if="openClawCommandData.ready" class="grid gap-2">
            <div v-for="(step, index) in openClawCommandData.steps" :key="`${step.title}-${index}`">
              <p class="text-xs font-semibold text-[var(--color-text)] mb-1">{{ t('step_line_label', { num: index + 1, title: step.title }) }}</p>
              <pre class="p-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-xs font-mono whitespace-pre-wrap break-all">{{ step.command }}</pre>
            </div>
          </div>
          <pre v-else class="p-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-xs font-mono whitespace-pre-wrap break-all">{{ openClawCommandDisplayText }}</pre>
        </div>
        <!-- Chat test -->
        <div>
          <p class="text-xs font-semibold text-[var(--color-text)] mb-2">{{ t('openclaw_chat_test') }}</p>
          <div class="grid gap-2">
            <Textarea v-model="openClawChatTestForm.message" :placeholder="t('openclaw_input_ph')" rows="3" />
            <Button :disabled="openClawChatTesting" @click="testOpenClawChat">{{ t('send_test_btn') }}</Button>
            <pre class="min-h-[60px] p-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-xs font-mono whitespace-pre-wrap break-all">{{ openClawChatTestResult || t('no_test_results') }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
