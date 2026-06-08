<template>
  <div class="grid gap-5">
    <!-- Toolbar -->
    <section class="flex flex-wrap items-center justify-between gap-4 px-6 py-5 rounded-[30px] bg-[var(--color-surface)] border border-[var(--color-line)] shadow-sm">
      <div class="flex flex-wrap gap-2">
        <span class="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold bg-[var(--color-primary)] text-white">{{ t('agent') }} {{ agentsCountText }}</span>
        <span class="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold bg-green-500 text-white">{{ t('device') }} {{ devicesCountText }}</span>
        <span class="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)] border border-[var(--color-line)]">{{ t('online') }} {{ onlineDevicesCountText }}</span>
      </div>
      <div class="flex flex-wrap gap-3 justify-end">
        <Button @click="showAddAgentDialog = true">
          <Plus class="w-4 h-4 mr-1.5" />{{ t('add_agent') }}
        </Button>
        <Button variant="outline" @click="openAddDeviceDialog">
          <Monitor class="w-4 h-4 mr-1.5" />{{ t('add_device') }}
        </Button>
        <Button variant="outline" @click="openInjectMessageDialog">
          <MessageSquare class="w-4 h-4 mr-1.5" />{{ t('voice_push') }}
        </Button>
      </div>
    </section>

    <!-- Skeleton loading -->
    <section v-if="initialLoading" class="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(280px,340px))]">
      <article v-for="i in 3" :key="i" class="p-5 rounded-[28px] bg-[var(--color-surface)] border border-[var(--color-line)] min-h-[220px] flex flex-col gap-3.5 pointer-events-none">
        <div class="flex items-start gap-3.5">
          <div class="w-12 h-12 rounded-2xl flex-none animate-pulse bg-[var(--color-surface-muted)]" />
          <div class="flex-1 grid gap-2.5 pt-1">
            <div class="h-5 w-[46%] rounded-full animate-pulse bg-[var(--color-surface-muted)]" />
            <div class="h-3.5 w-[64%] rounded-full animate-pulse bg-[var(--color-surface-muted)]" />
          </div>
          <div class="grid grid-cols-2 gap-2">
            <span v-for="j in 4" :key="j" class="w-7 h-7 rounded-[10px] animate-pulse bg-[var(--color-surface-muted)]" />
          </div>
        </div>
        <div class="grid gap-2.5 mt-1">
          <div v-for="j in 3" :key="j" class="h-[18px] rounded-full animate-pulse bg-[var(--color-surface-muted)]" :class="j === 3 ? 'w-[52%]' : ''" />
        </div>
        <div class="flex gap-2.5 mt-auto">
          <span v-for="j in 3" :key="j" class="flex-1 h-8 rounded-[10px] animate-pulse bg-[var(--color-surface-muted)]" />
        </div>
      </article>
    </section>

    <!-- Empty welcome state -->
    <div v-else-if="agents.length === 0" class="flex justify-start">
      <div class="p-8 rounded-[28px] bg-[var(--color-surface)] border border-[var(--color-line)] text-center">
        <Monitor class="w-16 h-16 mx-auto text-[var(--color-primary)] mb-4" />
        <h3 class="text-2xl font-bold text-[var(--color-text)] mb-2.5">{{ t('create_first_agent') }}</h3>
        <p class="text-sm text-[var(--color-text-secondary)] leading-relaxed mb-6">{{ t('post_create_agent_hint') }}</p>
        <Button size="lg" @click="showAddAgentDialog = true">
          <Plus class="w-4 h-4 mr-2" />{{ t('create_agent_label') }}
        </Button>
      </div>
    </div>

    <!-- Agent cards grid -->
    <section v-else class="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(280px,340px))] content-start justify-start">
      <article v-for="agent in agents" :key="agent.id" class="p-5 rounded-[28px] bg-[var(--color-surface)] border border-[var(--color-line)] flex flex-col gap-3.5 max-w-[340px] w-full">
        <!-- Header -->
        <div class="flex items-start justify-between gap-3.5">
          <div class="flex items-center gap-3.5 min-w-0 flex-1">
            <div class="w-12 h-12 rounded-2xl flex-none inline-flex items-center justify-center text-white bg-gradient-to-b from-blue-400 to-blue-600 shadow-[0_12px_24px_rgba(0,122,255,0.18)]">
              <Monitor class="w-5 h-5" />
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="text-lg font-semibold text-[var(--color-text)] m-0 mb-1 leading-snug">{{ agent.name }}</h3>
              <p class="text-[13px] text-[var(--color-text-secondary)] m-0 leading-relaxed truncate">{{ t('nickname_label') }} {{ agent.nickname || agent.name }}</p>
            </div>
          </div>
          <!-- State badges 2×2 -->
          <div class="flex-none grid grid-cols-2 gap-2.5">
            <div
              :title="t('memory_type_tooltip', { type: getMemoryModeText(agent) })"
              class="w-[30px] h-[30px] rounded-[10px] inline-flex items-center justify-center border cursor-default"
              :class="memoryBadgeClass(agent)"
            >
              <img class="w-3.5 h-3.5 block object-contain pointer-events-none select-none" :src="memoryStatusIcon" alt="" />
            </div>
            <div
              :title="getKnowledgeBaseTooltip(agent)"
              class="w-[30px] h-[30px] rounded-[10px] inline-flex items-center justify-center border cursor-default"
              :class="getKnowledgeBaseCount(agent) > 0 ? 'text-[var(--color-primary)] bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800' : 'text-[var(--color-text-tertiary)] bg-[var(--color-surface-muted)] border-[var(--color-line)]'"
            >
              <img class="w-[15px] h-[15px] block object-contain pointer-events-none select-none" :src="knowledgeBaseStatusIcon" alt="" />
            </div>
            <div
              :title="getMcpStatusTooltip(agent)"
              class="w-[30px] h-[30px] rounded-[10px] inline-flex items-center justify-center border cursor-default"
              :class="mcpBadgeClass(agent)"
              @mouseenter="ensureMcpConnectionStatus(agent.id)"
            >
              <img class="w-[15px] h-[15px] block object-contain pointer-events-none select-none" :src="mcpStatusIcon" alt="" />
            </div>
            <div
              :title="getOpenClawStatusTooltip(agent)"
              class="w-[30px] h-[30px] rounded-[10px] inline-flex items-center justify-center border cursor-default"
              :class="openClawBadgeClass(agent)"
              @mouseenter="ensureOpenClawConnectionStatus(agent.id)"
            >
              <img class="w-[15px] h-[15px] block object-contain pointer-events-none select-none" :src="openClawStatusIcon" alt="" />
            </div>
          </div>
        </div>

        <!-- Summary rows -->
        <div class="grid gap-1.5">
          <div class="flex items-center gap-2 min-w-0 text-[13px] leading-relaxed" :title="`${t('timbre_model')} ${getVoiceModelText(agent)}`">
            <span class="flex-none text-[var(--color-text-secondary)]">{{ t('timbre_model') }}</span>
            <span class="flex-1 min-w-0 font-semibold text-[var(--color-text)] overflow-hidden text-ellipsis whitespace-nowrap">{{ truncateText(getVoiceModelText(agent), 18) }}</span>
          </div>
          <div class="flex items-center gap-2 min-w-0 text-[13px] leading-relaxed" :title="`${t('language_model_label')} ${getLLMModelText(agent)}`">
            <span class="flex-none text-[var(--color-text-secondary)]">{{ t('language_model_label') }}</span>
            <span class="flex-1 min-w-0 font-semibold text-[var(--color-text)] overflow-hidden text-ellipsis whitespace-nowrap">{{ truncateText(getLLMModelText(agent), 16) }}</span>
          </div>
          <div class="flex items-center gap-2 min-w-0 text-[13px] leading-relaxed" :title="`${t('device_count')} ${getDeviceCountText(agent)}`">
            <span class="flex-none text-[var(--color-text-secondary)]">{{ t('device_count') }}</span>
            <span class="flex-1 font-bold text-[var(--color-text)]">{{ getDeviceCountText(agent) }}</span>
          </div>
        </div>

        <!-- Action buttons 2×2 -->
        <div class="grid grid-cols-2 gap-2 mt-auto">
          <Button
            variant="outline" size="sm"
            class="text-xs font-semibold rounded-xl text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800 bg-blue-50/80 dark:bg-blue-900/20 hover:bg-blue-100 dark:hover:bg-blue-900/30"
            @click="editAgent(agent.id)"
          >
            <Settings class="w-3.5 h-3.5 mr-1" />{{ t('config') }}
          </Button>
          <Button variant="outline" size="sm" class="text-xs font-semibold rounded-xl" @click="handleChatHistory(agent.id)">
            <MessageSquare class="w-3.5 h-3.5 mr-1" />{{ t('chat') }}
          </Button>
          <Button variant="outline" size="sm" class="text-xs font-semibold rounded-xl" @click="handleManageDevices(agent.id)">
            <Link2 class="w-3.5 h-3.5 mr-1" />{{ t('device') }}
          </Button>
          <Button
            variant="outline" size="sm"
            class="text-xs font-semibold rounded-xl text-red-600 dark:text-red-400 border-red-200 dark:border-red-800 bg-red-50/80 dark:bg-red-900/20 hover:bg-red-100 dark:hover:bg-red-900/30"
            @click="handleDeleteAgent(agent)"
          >
            <Trash2 class="w-3.5 h-3.5 mr-1" />{{ t('delete') }}
          </Button>
        </div>
      </article>
    </section>

    <!-- Add Agent Dialog -->
    <Dialog v-model:open="showAddAgentDialog" @update:open="v => { if (!v) { agentFormRef?.resetFields?.(); agentForm = createDefaultAgentForm() } }">
      <DialogContent class="max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{{ t('add_agent') }}</DialogTitle>
        </DialogHeader>
        <AgentForm ref="agentFormRef" v-model="agentForm" mode="create" />
        <DialogFooter>
          <Button variant="outline" @click="showAddAgentDialog = false">{{ t('cancel') }}</Button>
          <Button :disabled="adding" @click="handleAddAgent">
            {{ adding ? t('creating') : t('create_agent_label') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Add Device Dialog -->
    <Dialog v-model:open="showAddDeviceDialog" @update:open="v => { if (!v) resetAddDeviceForm() }">
      <DialogContent class="max-w-[520px]">
        <DialogHeader>
          <DialogTitle>{{ t('add_device') }}</DialogTitle>
        </DialogHeader>
        <DeviceForm ref="deviceFormRef" v-model="deviceForm" mode="bind" :agents="agents" />
        <DialogFooter>
          <Button variant="outline" @click="showAddDeviceDialog = false">{{ t('cancel') }}</Button>
          <Button :disabled="addingDevice" @click="handleAddDevice">
            {{ addingDevice ? t('binding') : t('bind_device') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <MessageInjectDialog v-model="showInjectMessageDialog" :devices="allDevices" @success="handleInjectSuccess" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Settings, MessageSquare, Monitor, Trash2, Link2 } from '@lucide/vue'
import api from '../../utils/api'
import AgentForm from '../../components/common/AgentForm.vue'
import DeviceForm from '../../components/common/DeviceForm.vue'
import MessageInjectDialog from '../../components/user/MessageInjectDialog.vue'
import { createDefaultAgentForm, createDefaultDeviceForm } from '../../composables/useAgentFormOptions'
import mcpStatusIcon from '../../assets/agent-status-icons/mcp.png'
import openClawStatusIcon from '../../assets/agent-status-icons/openclaw.png'
import memoryStatusIcon from '../../assets/agent-status-icons/memory.png'
import knowledgeBaseStatusIcon from '../../assets/agent-status-icons/knowledge-base.png'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'

const { t } = useLocale()
const router = useRouter()

const agents = ref([])
const allDevices = ref([])
const knowledgeBases = ref([])

const showAddAgentDialog = ref(false)
const showAddDeviceDialog = ref(false)
const showInjectMessageDialog = ref(false)

const adding = ref(false)
const addingDevice = ref(false)
const agentFormRef = ref()
const deviceFormRef = ref()
const initialLoading = ref(true)

const agentForm = ref(createDefaultAgentForm())
const deviceForm = ref(createDefaultDeviceForm({ mode: 'bind' }))

const onlineDevicesCount = computed(() => allDevices.value.filter(device => isDeviceOnline(device.last_active_at)).length)
const agentsCountText = computed(() => initialLoading.value ? '--' : agents.value.length)
const devicesCountText = computed(() => initialLoading.value ? '--' : allDevices.value.length)
const onlineDevicesCountText = computed(() => initialLoading.value ? '--' : onlineDevicesCount.value)
const knowledgeBaseNameMap = computed(() => {
  const map = new Map()
  for (const kb of knowledgeBases.value) {
    map.set(Number(kb.id), kb.name || t('knowledge_base_id', { id: kb.id }))
  }
  return map
})
const mcpConnectionStatusMap = reactive({})
const openClawConnectionStatusMap = reactive({})
const globalMcpServiceCount = ref(null)
const globalMcpServiceCountError = ref('')

const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  const lastActive = new Date(lastActiveAt)
  return (Date.now() - lastActive.getTime()) < 5 * 60 * 1000
}

const getAgentDevices = (agentId) => {
  return allDevices.value.filter(device => Number(device.agent_id) === Number(agentId))
}

const getAgentDeviceCount = (agentId) => getAgentDevices(agentId).length
const canDeleteAgent = (agent) => getAgentDeviceCount(agent.id) === 0

const loadAgents = async () => {
  try {
    const response = await api.get('/user/agents')
    agents.value = response.data.data || []
  } catch {
    ElMessage.error(t('load_agent_list_failed'))
  }
}

const loadDevices = async () => {
  try {
    const response = await api.get('/user/devices')
    allDevices.value = response.data.data || []
  } catch {
    allDevices.value = []
    ElMessage.error(t('load_device_list_failed'))
  }
}

const loadKnowledgeBases = async () => {
  try {
    const response = await api.get('/user/knowledge-bases')
    knowledgeBases.value = response.data.data || []
  } catch (error) {
    knowledgeBases.value = []
    console.error(t('load_kb_list_failed'), error)
  }
}

const handleAddAgent = async () => {
  if (!agentFormRef.value) return
  try {
    await agentFormRef.value.validate()
  } catch {
    return
  }
  adding.value = true
  try {
    const response = await api.post('/user/agents', agentFormRef.value.buildPayload())
    if (response.data.success) {
      ElMessage.success(t('agent_add_success'))
      showAddAgentDialog.value = false
      await loadAgents()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('add_agent_failed'))
  } finally {
    adding.value = false
  }
}

const resetAddDeviceForm = () => {
  deviceForm.value = createDefaultDeviceForm({ mode: 'bind' })
  deviceFormRef.value?.clearValidate?.()
}

const openAddDeviceDialog = () => {
  if (!agents.value.length) {
    ElMessage.warning(t('create_agent_before_bind'))
    return
  }
  resetAddDeviceForm()
  showAddDeviceDialog.value = true
}

const handleAddDevice = async () => {
  if (!deviceFormRef.value) return
  try {
    await deviceFormRef.value.validate()
  } catch {
    return
  }
  const agentId = deviceForm.value.agent_id
  if (!agentId) {
    ElMessage.warning(t('select_target_agent'))
    return
  }
  addingDevice.value = true
  try {
    const response = await api.post(`/user/agents/${agentId}/devices`, deviceFormRef.value.buildPayload())
    if (response.data?.success) {
      ElMessage.success(t('device_bind_success'))
      showAddDeviceDialog.value = false
      await handleDeviceBound()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('device_bind_failed'))
  } finally {
    addingDevice.value = false
  }
}

const openInjectMessageDialog = () => {
  if (!allDevices.value.length) {
    ElMessage.warning(t('bind_device_before_push'))
    return
  }
  showInjectMessageDialog.value = true
}

const handleDeviceBound = async () => {
  await Promise.all([loadAgents(), loadDevices()])
}

const handleInjectSuccess = async () => {
  await loadDevices()
}

const editAgent = (id) => {
  router.push(`/user/agents/${id}/edit`)
}

const handleChatHistory = (id) => {
  router.push(`/user/agents/${id}/history`)
}

const handleManageDevices = (id) => {
  router.push({ path: '/user/devices', query: { agent_id: id } })
}

const handleDeleteAgent = async (agent) => {
  if (!canDeleteAgent(agent)) {
    ElMessage.warning(t('agent_has_bound_devices'))
    return
  }
  try {
    await ElMessageBox.confirm(
      t('confirm_delete_agent_msg', { name: agent.name }),
      t('confirm_delete'),
      { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' }
    )
    await api.delete(`/user/agents/${agent.id}`)
    ElMessage.success(t('agent_delete_success'))
    await loadAgents()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || t('agent_delete_failed'))
    }
  }
}

const truncateText = (value, maxLength = 14) => {
  const text = String(value || '').trim() || t('not_set')
  return text.length <= maxLength ? text : `${text.slice(0, maxLength)}...`
}

const getVoiceModelText = (agent) => {
  const ttsType = agent.tts_config?.name?.trim() || agent.tts_config?.provider?.trim() || ''
  const voiceName = typeof agent.voice === 'string' ? agent.voice.trim() : ''
  if (ttsType && voiceName) return `${ttsType} · ${voiceName}`
  if (ttsType) return t('default_voice_suffix', { type: ttsType })
  if (voiceName) return voiceName
  return t('not_set')
}

const getLLMModelText = (agent) => {
  return agent.llm_config?.name?.trim() || agent.llm_config?.provider?.trim() || t('not_set')
}

const getDeviceCountText = (agent) => {
  return t('device_count_n', { count: getAgentDeviceCount(agent.id) })
}

const getMemoryModeKey = (agent) => {
  const mode = String(agent.memory_mode || 'short').trim().toLowerCase()
  if (mode === 'none') return 'none'
  if (mode === 'long') return 'long'
  return 'short'
}

const getMemoryModeText = (agent) => {
  const key = getMemoryModeKey(agent)
  if (key === 'none') return t('no_memory')
  if (key === 'long') return t('long_memory')
  return t('short_memory')
}

const getKnowledgeBaseIds = (agent) => {
  return Array.isArray(agent.knowledge_base_ids) ? agent.knowledge_base_ids : []
}

const getKnowledgeBaseCount = (agent) => getKnowledgeBaseIds(agent).length

const getKnowledgeBaseNames = (agent) => {
  return getKnowledgeBaseIds(agent).map((id) => knowledgeBaseNameMap.value.get(Number(id)) || t('knowledge_base_id', { id }))
}

const getKnowledgeBaseTooltip = (agent) => {
  const names = getKnowledgeBaseNames(agent)
  return names.length === 0 ? t('knowledge_base_unlinked') : t('linked_kbs_tooltip', { names: names.join(', ') })
}

const normalizeMcpServiceNames = (raw) => {
  return String(raw || '').split(',').map(item => item.trim()).filter(Boolean)
}

const getConnectionStatusText = (state) => {
  if (!state || state.loading) return t('detecting')
  if (state.connected || state.status === 'online') return t('connected')
  if (state.status === 'offline') return t('not_connected')
  if (state.status_message) return t('connection_unknown')
  return t('not_connected')
}

const getMcpStatusKey = (agent) => {
  const state = mcpConnectionStatusMap[String(agent.id)]
  if (!state || state.loading) return 'checking'
  if (state.connected || state.status === 'online') return 'online'
  if (state.status === 'offline') return 'offline'
  return 'unknown'
}

const getGlobalMcpServiceCountText = () => {
  if (globalMcpServiceCountError.value) return t('detection_failed')
  if (globalMcpServiceCount.value === null) return t('detecting')
  return t('global_mcp_count', { count: globalMcpServiceCount.value })
}

const getMcpServiceScopeText = (agent) => {
  const count = normalizeMcpServiceNames(agent.mcp_service_names).length
  return count > 0 ? t('mcp_selected_services', { count }) : t('follow_global_config')
}

const getMcpClientCountText = (connection) => {
  const count = Number(connection?.client_count || 0)
  return count <= 0 ? '' : t('mcp_client_count_text', { count })
}

const getMcpStatusTooltip = (agent) => {
  const connection = mcpConnectionStatusMap[String(agent.id)]
  const connectionText = getConnectionStatusText(connection)
  return t('mcp_status_tooltip', { ws: connectionText, clients: getMcpClientCountText(connection), mcp: getGlobalMcpServiceCountText(), scope: getMcpServiceScopeText(agent) })
}

const parseOpenClawConfig = (agent) => {
  if (agent?.openclaw && typeof agent.openclaw === 'object') return { allowed: !!agent.openclaw.allowed }
  if (typeof agent?.openclaw_config === 'string' && agent.openclaw_config.trim()) {
    try { return { allowed: !!JSON.parse(agent.openclaw_config)?.allowed } } catch {}
  }
  return { allowed: false }
}

const getOpenClawStatusKey = (agent) => parseOpenClawConfig(agent).allowed ? 'enabled' : 'disabled'

const getOpenClawStatusTooltip = (agent) => {
  const connection = openClawConnectionStatusMap[String(agent.id)]
  const configText = parseOpenClawConfig(agent).allowed ? t('enabled') : t('not_enabled')
  return t('openclaw_status_tooltip', { config: configText, connection: getConnectionStatusText(connection) })
}

const ensureMcpConnectionStatus = async (agentId) => {
  const key = String(agentId)
  const current = mcpConnectionStatusMap[key]
  if (current?.loading || current?.loaded) return
  mcpConnectionStatusMap[key] = { loading: true, loaded: false, connected: false, status: 'unknown', status_message: '', client_count: 0 }
  try {
    const response = await api.get(`/user/agents/${agentId}/mcp-endpoint`)
    const data = response.data.data || {}
    mcpConnectionStatusMap[key] = {
      loading: false, loaded: true, connected: !!data.connected,
      status: String(data.status || 'unknown').toLowerCase(),
      status_message: String(data.status_message || ''),
      client_count: Number(data.client_count || 0)
    }
  } catch (error) {
    mcpConnectionStatusMap[key] = {
      loading: false, loaded: true, connected: false, status: 'unknown',
      status_message: error.response?.data?.error || error.message || t('status_fetch_failed'),
      client_count: 0
    }
  }
}

const loadGlobalMcpServiceCount = async () => {
  globalMcpServiceCountError.value = ''
  try {
    const response = await api.get('/user/mcp-services/options')
    const options = response.data.data?.options
    globalMcpServiceCount.value = Array.isArray(options) ? options.length : 0
  } catch (error) {
    globalMcpServiceCount.value = null
    globalMcpServiceCountError.value = error.response?.data?.error || error.message || t('load_failed')
    console.error(t('load_global_mcp_failed'), error)
  }
}

const loadMcpConnectionStatuses = async () => {
  await Promise.all(agents.value.map(agent => ensureMcpConnectionStatus(agent.id)))
}

const ensureOpenClawConnectionStatus = async (agentId) => {
  const key = String(agentId)
  const current = openClawConnectionStatusMap[key]
  if (current?.loading || current?.loaded) return
  openClawConnectionStatusMap[key] = { loading: true, loaded: false, connected: false, status: 'unknown', status_message: '' }
  try {
    const response = await api.get(`/user/agents/${agentId}/openclaw-endpoint`)
    const data = response.data.data || {}
    openClawConnectionStatusMap[key] = {
      loading: false, loaded: true, connected: !!data.connected,
      status: String(data.status || 'unknown').toLowerCase(),
      status_message: String(data.status_message || '')
    }
  } catch (error) {
    openClawConnectionStatusMap[key] = {
      loading: false, loaded: true, connected: false, status: 'unknown',
      status_message: error.response?.data?.error || error.message || t('status_fetch_failed')
    }
  }
}

// Badge class helpers
const BLUE_BADGE = 'text-[var(--color-primary)] bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800'
const GREEN_BADGE = 'text-green-700 dark:text-green-400 bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800'
const RED_BADGE = 'text-red-700 dark:text-red-400 bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
const MUTED_BADGE = 'text-[var(--color-text-tertiary)] bg-[var(--color-surface-muted)] border-[var(--color-line)]'

const memoryBadgeClass = (agent) => {
  const key = getMemoryModeKey(agent)
  if (key === 'short') return BLUE_BADGE
  if (key === 'long') return GREEN_BADGE
  return MUTED_BADGE
}

const mcpBadgeClass = (agent) => {
  const key = getMcpStatusKey(agent)
  if (key === 'checking') return BLUE_BADGE
  if (key === 'online') return GREEN_BADGE
  if (key === 'offline') return RED_BADGE
  return MUTED_BADGE
}

const openClawBadgeClass = (agent) => {
  return getOpenClawStatusKey(agent) === 'enabled' ? GREEN_BADGE : MUTED_BADGE
}

onMounted(async () => {
  initialLoading.value = true
  try {
    await Promise.all([loadAgents(), loadDevices(), loadKnowledgeBases()])
    void loadGlobalMcpServiceCount()
    void loadMcpConnectionStatuses()
  } finally {
    initialLoading.value = false
  }
})
</script>
