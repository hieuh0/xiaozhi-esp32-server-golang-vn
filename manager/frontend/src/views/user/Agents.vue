<template>
  <div class="agents-page">
    <section class="page-toolbar apple-surface">
      <div class="toolbar-chips">
        <span class="apple-chip is-primary">{{ t('agent') }} {{ agentsCountText }}</span>
        <span class="apple-chip is-success">{{ t('device') }} {{ devicesCountText }}</span>
        <span class="apple-chip">{{ t('online') }} {{ onlineDevicesCountText }}</span>
      </div>

      <div class="toolbar-actions">
        <el-button type="primary" @click="showAddAgentDialog = true">
          <el-icon><Plus /></el-icon>
          {{ t('add_agent') }}
        </el-button>
        <el-button plain @click="openAddDeviceDialog">
          <el-icon><Monitor /></el-icon>
          {{ t('add_device') }}</el-button>
        <el-button plain @click="openInjectMessageDialog">
          <el-icon><ChatDotRound /></el-icon>
          {{ t('voice_push') }}
        </el-button>
      </div>
    </section>

    <section v-if="initialLoading" class="agents-grid agents-grid-loading" aria-:label="t('agent_loading')">
      <article v-for="index in 3" :key="index" class="agent-card agent-card-skeleton apple-surface">
        <div class="skeleton-card-header">
          <div class="skeleton-avatar skeleton-shimmer"></div>
          <div class="skeleton-title-block">
            <div class="skeleton-line skeleton-line-title skeleton-shimmer"></div>
            <div class="skeleton-line skeleton-line-subtitle skeleton-shimmer"></div>
          </div>
          <div class="skeleton-icons">
            <span class="skeleton-icon skeleton-shimmer"></span>
            <span class="skeleton-icon skeleton-shimmer"></span>
            <span class="skeleton-icon skeleton-shimmer"></span>
            <span class="skeleton-icon skeleton-shimmer"></span>
          </div>
        </div>
        <div class="skeleton-summary">
          <div class="skeleton-line skeleton-shimmer"></div>
          <div class="skeleton-line skeleton-shimmer"></div>
          <div class="skeleton-line skeleton-line-short skeleton-shimmer"></div>
        </div>
        <div class="skeleton-actions">
          <span class="skeleton-button skeleton-shimmer"></span>
          <span class="skeleton-button skeleton-shimmer"></span>
          <span class="skeleton-button skeleton-shimmer"></span>
        </div>
      </article>
    </section>

    <div v-else-if="agents.length === 0" class="welcome-section">
      <el-card class="welcome-card apple-surface">
        <div class="welcome-content">
          <el-icon size="64" color="var(--apple-primary)"><Monitor /></el-icon>
          <h3>{{ t('create_first_agent') }}</h3>
          <p>{{ t('post_create_agent_hint') }}</p>
          <div class="welcome-actions">
            <el-button type="primary" size="large" @click="showAddAgentDialog = true">
              <el-icon><Plus /></el-icon>
              {{ t('create_agent_label') }}</el-button>
          </div>
        </div>
      </el-card>
    </div>

    <section v-else class="agents-grid">
      <article v-for="agent in agents" :key="agent.id" class="agent-card apple-surface">
        <div class="agent-header">
          <div class="agent-heading">
            <div class="agent-avatar">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="agent-info">
              <h3 class="agent-name">{{ agent.name }}</h3>
              <p class="agent-desc">{{ t('nickname_label') }} {{ agent.nickname || agent.name }}</p>
            </div>
          </div>

          <div class="agent-state-grid">
            <el-tooltip :content="t('memory_type_tooltip', { type: getMemoryModeText(agent) })" placement="top" :show-after="200">
              <div class="agent-state-badge is-icon-only" :class="`is-memory-${getMemoryModeKey(agent)}`">
                <img class="state-image-icon state-image-icon--memory" :src="memoryStatusIcon" alt="" />
              </div>
            </el-tooltip>

            <el-tooltip :content="getKnowledgeBaseTooltip(agent)" placement="top" :show-after="200">
              <div class="agent-state-badge is-icon-only" :class="{ 'is-active': getKnowledgeBaseCount(agent) > 0 }">
                <img class="state-image-icon state-image-icon--knowledge" :src="knowledgeBaseStatusIcon" alt="" />
              </div>
            </el-tooltip>

            <el-tooltip :content="getMcpStatusTooltip(agent)" placement="top" :show-after="200" @show="ensureMcpConnectionStatus(agent.id)">
              <div class="agent-state-badge is-icon-only" :class="`is-mcp-${getMcpStatusKey(agent)}`">
                <img class="state-image-icon state-image-icon--mcp" :src="mcpStatusIcon" alt="" />
              </div>
            </el-tooltip>

            <el-tooltip :content="getOpenClawStatusTooltip(agent)" placement="top" :show-after="200" @show="ensureOpenClawConnectionStatus(agent.id)">
              <div class="agent-state-badge is-icon-only" :class="`is-openclaw-${getOpenClawStatusKey(agent)}`">
                <img class="state-image-icon state-image-icon--openclaw" :src="openClawStatusIcon" alt="" />
              </div>
            </el-tooltip>
          </div>
        </div>

        <div class="agent-summary">
          <div class="summary-row" :title="`${t('timbre_model')} ${getVoiceModelText(agent)}`">
            <span class="summary-label">{{ t('timbre_model') }}</span>
            <span class="summary-text">{{ truncateText(getVoiceModelText(agent), 18) }}</span>
          </div>
          <div class="summary-row" :title="`${t('language_model_label')} ${getLLMModelText(agent)}`">
            <span class="summary-label">{{ t('language_model_label') }}</span>
            <span class="summary-text">{{ truncateText(getLLMModelText(agent), 16) }}</span>
          </div>
          <div class="summary-row" :title="`${t('device_count')} ${getDeviceCountText(agent)}`">
            <span class="summary-label">{{ t('device_count') }}</span>
            <span class="summary-text is-count">{{ getDeviceCountText(agent) }}</span>
          </div>
        </div>

        <div class="agent-actions">
          <el-button class="agent-action-button agent-action-button-feature" size="small" @click="editAgent(agent.id)">
            <el-icon><Setting /></el-icon>
            {{ t('config') }}</el-button>
          <el-button class="agent-action-button" size="small" @click="handleChatHistory(agent.id)">
            <el-icon><ChatDotRound /></el-icon>
            {{ t('chat') }}</el-button>
          <el-button class="agent-action-button" size="small" @click="handleManageDevices(agent.id)">
            <el-icon><Connection /></el-icon>
            {{ t('device') }}
          </el-button>
          <el-button class="agent-action-button agent-action-button-danger" size="small" @click="handleDeleteAgent(agent)">
            <el-icon><Delete /></el-icon>
            {{ t('delete') }}
          </el-button>
        </div>
      </article>
    </section>

    <el-dialog
      v-model="showAddAgentDialog"
      :title="t('add_agent')"
      width="560px"
      class="agent-dialog"
      :before-close="handleCloseAddAgent"
    >
      <AgentForm
        ref="agentFormRef"
        v-model="agentForm"
        mode="create"
      />

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleCloseAddAgent">{{ t('cancel') }}</el-button>
          <el-button type="primary" :loading="adding" @click="handleAddAgent">
            {{ adding ? t('creating') : t('create_agent_label') }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showAddDeviceDialog"
      :title="t('add_device')"
      width="520px"
      :close-on-click-modal="false"
      @closed="resetAddDeviceForm"
    >
      <DeviceForm
        ref="deviceFormRef"
        v-model="deviceForm"
        mode="bind"
        :agents="agents"
      />
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDeviceDialog = false">{{ t('cancel') }}</el-button>
          <el-button type="primary" :loading="addingDevice" @click="handleAddDevice">
            {{ addingDevice ? t('binding') : t('bind_device') }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <MessageInjectDialog
      v-model="showInjectMessageDialog"
      :devices="allDevices"
      @success="handleInjectSuccess"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Setting, ChatDotRound, Monitor, Delete, Connection } from '@element-plus/icons-vue'
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
  } catch (error) {
    ElMessage.error(t('load_agent_list_failed'))
  }
}

const loadDevices = async () => {
  try {
    const response = await api.get('/user/devices')
    allDevices.value = response.data.data || []
  } catch (error) {
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
      handleCloseAddAgent()
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

const handleCloseAddAgent = () => {
  showAddAgentDialog.value = false
  agentFormRef.value?.resetFields?.()
  agentForm.value = createDefaultAgentForm()
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
      {
        confirmButtonText: t('confirm'),
        cancelButtonText: t('cancel'),
        type: 'warning'
      }
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
  if (text.length <= maxLength) {
    return text
  }
  return `${text.slice(0, maxLength)}...`
}

const getVoiceModelText = (agent) => {
  const ttsType = agent.tts_config?.name?.trim() || agent.tts_config?.provider?.trim() || ''
  const voiceName = typeof agent.voice === 'string' ? agent.voice.trim() : ''

  if (ttsType && voiceName) {
    return `${ttsType} · ${voiceName}`
  }
  if (ttsType) {
    return t('default_voice_suffix', { type: ttsType })
  }
  if (voiceName) {
    return voiceName
  }
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

const getKnowledgeBaseCount = (agent) => {
  return getKnowledgeBaseIds(agent).length
}

const getKnowledgeBaseNames = (agent) => {
  return getKnowledgeBaseIds(agent).map((id) => knowledgeBaseNameMap.value.get(Number(id)) || t('knowledge_base_id', { id }))
}

const getKnowledgeBaseTooltip = (agent) => {
  const names = getKnowledgeBaseNames(agent)
  if (names.length === 0) {
    return t('knowledge_base_unlinked')
  }
  return t('linked_kbs_tooltip', { names: names.join(', ') })
}

const normalizeMcpServiceNames = (raw) => {
  return String(raw || '')
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
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
  if (count <= 0) return ''
  return t('mcp_client_count_text', { count })
}

const getMcpStatusTooltip = (agent) => {
  const connection = mcpConnectionStatusMap[String(agent.id)]
  const connectionText = getConnectionStatusText(connection)
  return t('mcp_status_tooltip', { ws: connectionText, clients: getMcpClientCountText(connection), mcp: getGlobalMcpServiceCountText(), scope: getMcpServiceScopeText(agent) })
}

const parseOpenClawConfig = (agent) => {
  if (agent?.openclaw && typeof agent.openclaw === 'object') {
    return { allowed: !!agent.openclaw.allowed }
  }

  if (typeof agent?.openclaw_config === 'string' && agent.openclaw_config.trim()) {
    try {
      const parsed = JSON.parse(agent.openclaw_config)
      return { allowed: !!parsed?.allowed }
    } catch {}
  }

  return { allowed: false }
}

const getOpenClawStatusKey = (agent) => {
  return parseOpenClawConfig(agent).allowed ? 'enabled' : 'disabled'
}

const getOpenClawStatusTooltip = (agent) => {
  const connection = openClawConnectionStatusMap[String(agent.id)]
  const configText = parseOpenClawConfig(agent).allowed ? t('enabled') : t('not_enabled')
  return t('openclaw_status_tooltip', { config: configText, connection: getConnectionStatusText(connection) })
}

const ensureMcpConnectionStatus = async (agentId) => {
  const key = String(agentId)
  const current = mcpConnectionStatusMap[key]
  if (current?.loading || current?.loaded) return

  mcpConnectionStatusMap[key] = {
    loading: true,
    loaded: false,
    connected: false,
    status: 'unknown',
    status_message: '',
    client_count: 0
  }

  try {
    const response = await api.get(`/user/agents/${agentId}/mcp-endpoint`)
    const data = response.data.data || {}
    mcpConnectionStatusMap[key] = {
      loading: false,
      loaded: true,
      connected: !!data.connected,
      status: String(data.status || 'unknown').toLowerCase(),
      status_message: String(data.status_message || ''),
      client_count: Number(data.client_count || 0)
    }
  } catch (error) {
    mcpConnectionStatusMap[key] = {
      loading: false,
      loaded: true,
      connected: false,
      status: 'unknown',
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

  openClawConnectionStatusMap[key] = {
    loading: true,
    loaded: false,
    connected: false,
    status: 'unknown',
    status_message: ''
  }

  try {
    const response = await api.get(`/user/agents/${agentId}/openclaw-endpoint`)
    const data = response.data.data || {}
    openClawConnectionStatusMap[key] = {
      loading: false,
      loaded: true,
      connected: !!data.connected,
      status: String(data.status || 'unknown').toLowerCase(),
      status_message: String(data.status_message || '')
    }
  } catch (error) {
    openClawConnectionStatusMap[key] = {
      loading: false,
      loaded: true,
      connected: false,
      status: 'unknown',
      status_message: error.response?.data?.error || error.message || t('status_fetch_failed')
    }
  }
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

<style scoped>
.agents-page {
  display: grid;
  gap: 20px;
}

.page-toolbar {
  padding: 22px 24px;
  border-radius: 30px;
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
}

.toolbar-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.welcome-card {
  border-radius: 28px;
}

.welcome-content {
  text-align: center;
  padding: 44px 24px;
}

.welcome-content h3 {
  margin: 18px 0 10px;
  font-size: 26px;
  color: var(--apple-text);
}

.welcome-content p {
  margin: 0 0 24px;
  color: var(--apple-text-secondary);
  line-height: 1.7;
}

.welcome-actions {
  display: flex;
  justify-content: center;
}

.agents-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 340px));
  gap: 16px;
  align-content: flex-start;
  justify-content: flex-start;
}

.agent-card {
  padding: 20px;
  border-radius: 28px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-width: 340px;
  width: 100%;
}

.agent-card-skeleton {
  min-height: 220px;
  pointer-events: none;
}

.skeleton-card-header {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.skeleton-avatar {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  flex: none;
}

.skeleton-title-block {
  min-width: 0;
  flex: 1;
  display: grid;
  gap: 10px;
  padding-top: 4px;
}

.skeleton-icons {
  display: grid;
  grid-template-columns: repeat(2, 30px);
  gap: 8px;
}

.skeleton-icon {
  width: 30px;
  height: 30px;
  border-radius: 10px;
}

.skeleton-summary {
  display: grid;
  gap: 10px;
  margin-top: 4px;
}

.skeleton-line {
  height: 18px;
  border-radius: 999px;
}

.skeleton-line-title {
  width: 46%;
  height: 20px;
}

.skeleton-line-subtitle {
  width: 64%;
  height: 14px;
}

.skeleton-line-short {
  width: 52%;
}

.skeleton-actions {
  display: flex;
  gap: 10px;
  margin-top: auto;
}

.skeleton-button {
  width: 70px;
  height: 32px;
  border-radius: 10px;
}

.skeleton-shimmer {
  background: linear-gradient(90deg, rgba(226, 232, 240, 0.62) 0%, rgba(248, 250, 252, 0.94) 50%, rgba(226, 232, 240, 0.62) 100%);
  background-size: 220% 100%;
  animation: agents-skeleton-shimmer 1.15s ease-in-out infinite;
}

@keyframes agents-skeleton-shimmer {
  0% {
    background-position: 120% 0;
  }

  100% {
    background-position: -120% 0;
  }
}

.agent-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.agent-heading {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 14px;
}

.agent-avatar {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: linear-gradient(180deg, #2e90ff 0%, #007aff 100%);
  box-shadow: 0 12px 24px rgba(0, 122, 255, 0.18);
}

.agent-info {
  flex: 1;
  min-width: 0;
}

.agent-name {
  margin: 0 0 4px;
  font-size: 18px;
  color: var(--apple-text);
}

.agent-desc {
  margin: 0;
  font-size: 13px;
  color: var(--apple-text-secondary);
  line-height: 1.6;
}

.agent-state-grid {
  flex: none;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, auto));
  gap: 10px;
}

.agent-state-badge {
  min-width: 36px;
  min-height: 28px;
  padding: 0 8px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  color: var(--apple-text-secondary);
  position: relative;
}

.agent-state-badge.is-icon-only {
  min-width: 30px;
  width: 30px;
  min-height: 30px;
  height: 30px;
  padding: 0;
}

.state-image-icon {
  width: 15px;
  height: 15px;
  display: block;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.state-image-icon--memory {
  width: 14px;
  height: 14px;
}

.state-image-icon--knowledge {
  width: 15px;
  height: 15px;
}

.state-image-icon--mcp {
  width: 15px;
  height: 15px;
}

.state-image-icon--openclaw {
  width: 15px;
  height: 15px;
}

.agent-state-badge.is-memory-short,
.agent-state-badge.is-active,
.agent-state-badge.is-mcp-checking {
  color: var(--apple-primary);
  background: rgba(0, 122, 255, 0.08);
  border-color: rgba(0, 122, 255, 0.12);
}

.agent-state-badge.is-memory-long,
.agent-state-badge.is-mcp-online,
.agent-state-badge.is-openclaw-enabled {
  color: #176a31;
  background: rgba(52, 199, 89, 0.12);
  border-color: rgba(52, 199, 89, 0.16);
}

.agent-state-badge.is-mcp-offline {
  color: #b42318;
  background: rgba(255, 59, 48, 0.1);
  border-color: rgba(255, 59, 48, 0.16);
}

.agent-state-badge.is-memory-none,
.agent-state-badge.is-mcp-unknown,
.agent-state-badge.is-openclaw-disabled {
  color: var(--apple-text-tertiary);
  background: rgba(248, 250, 252, 0.92);
  border-color: rgba(229, 229, 234, 0.72);
}

.agent-summary {
  display: grid;
  gap: 6px;
}

.summary-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 13px;
  line-height: 1.6;
}

.summary-label {
  flex: none;
  color: var(--apple-text-secondary);
}

.summary-text {
  flex: 1;
  min-width: 0;
  color: var(--apple-text);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.summary-text.is-count {
  font-weight: 700;
}

.agent-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: auto;
}

.agent-actions .el-button {
  min-width: 0;
  width: 100%;
  min-height: 34px;
  margin-left: 0;
  padding: 0 10px;
  border-radius: 12px;
  border: 1px solid rgba(214, 219, 228, 0.9);
  background: rgba(248, 250, 252, 0.92);
  color: #4b5563;
  box-shadow: none;
  font-size: 12px;
  font-weight: 600;
}

.agent-actions .el-button + .el-button {
  margin-left: 0;
}

.agent-actions :deep(.el-button > span) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-actions :deep(.el-button .el-icon) {
  font-size: 13px;
}

.agent-action-button-danger {
  border-color: rgba(244, 191, 191, 0.95);
  background: rgba(255, 245, 245, 0.96);
  color: #b42318;
}

.agent-action-button-feature {
  border-color: rgba(147, 197, 253, 0.85);
  background: linear-gradient(180deg, rgba(239, 246, 255, 0.98) 0%, rgba(219, 234, 254, 0.9) 100%);
  color: #1d4ed8;
}

.agent-actions .el-button:hover {
  border-color: rgba(148, 163, 184, 0.82);
  background: rgba(241, 245, 249, 0.98);
  color: #334155;
}

.agent-action-button-feature:hover {
  border-color: rgba(96, 165, 250, 0.95);
  background: linear-gradient(180deg, rgba(219, 234, 254, 0.98) 0%, rgba(191, 219, 254, 0.92) 100%);
  color: #1e40af;
}

.agent-action-button-danger:hover {
  border-color: rgba(248, 113, 113, 0.78);
  background: rgba(254, 242, 242, 0.98);
  color: #991b1b;
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.form-tip {
  margin-top: 6px;
  color: var(--apple-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

@media (max-width: 1024px) {
  .page-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .toolbar-actions {
    width: 100%;
    justify-content: flex-start;
  }

}

@media (min-width: 769px) and (max-width: 1180px) {
  .agents-grid {
    grid-template-columns: repeat(auto-fill, minmax(280px, 340px));
  }
}

@media (max-width: 768px) {
  .agents-page {
    gap: 16px;
  }

  .page-toolbar,
  .agent-card {
    border-radius: 24px;
  }

  .page-toolbar {
    padding: 20px 18px;
  }

  .toolbar-actions,
  .dialog-footer {
    width: 100%;
    flex-wrap: wrap;
  }

  .toolbar-actions .el-button,
  .dialog-footer .el-button {
    flex: 1;
    min-width: 120px;
  }

  .dialog-grid {
    grid-template-columns: 1fr;
  }

  .agents-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .agent-actions {
    grid-template-columns: 1fr;
  }
}
</style>
