import { computed, ref, unref } from 'vue'
import api from '../utils/api'
import { tl as _tl } from '../utils/i18n-helper'

export const getDefaultAgentPrompt = () => _tl('default_agent_prompt')
export const DEFAULT_AGENT_PROMPT = getDefaultAgentPrompt()

export const OPENCLAW_DEFAULT_ENTER_KEYWORDS = ['open openclaw', 'enter openclaw']
export const OPENCLAW_DEFAULT_EXIT_KEYWORDS = ['close openclaw', 'exit openclaw']

const normalizeDataList = (response) => {
  const data = response?.data?.data
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.items)) return data.items
  return []
}

const normalizeNullableString = (value) => {
  const text = String(value ?? '').trim()
  return text || null
}

export const normalizeKeywordList = (keywords) => {
  if (!Array.isArray(keywords)) return []
  const seen = new Set()
  const result = []
  for (const item of keywords) {
    const keyword = String(item ?? '').trim()
    if (!keyword || seen.has(keyword)) continue
    seen.add(keyword)
    result.push(keyword)
  }
  return result
}

export const normalizeOpenClawConfig = (raw) => {
  const enterKeywords = normalizeKeywordList(raw?.enter_keywords)
  const exitKeywords = normalizeKeywordList(raw?.exit_keywords)
  return {
    allowed: !!raw?.allowed,
    enter_keywords: enterKeywords.length ? enterKeywords : [...OPENCLAW_DEFAULT_ENTER_KEYWORDS],
    exit_keywords: exitKeywords.length ? exitKeywords : [...OPENCLAW_DEFAULT_EXIT_KEYWORDS]
  }
}

export const parseOpenClawConfigFromAgent = (agent = {}) => {
  if (agent.openclaw && typeof agent.openclaw === 'object') {
    return normalizeOpenClawConfig(agent.openclaw)
  }
  if (typeof agent.openclaw_config !== 'string' || !agent.openclaw_config.trim()) {
    return normalizeOpenClawConfig(null)
  }
  try {
    return normalizeOpenClawConfig(JSON.parse(agent.openclaw_config))
  } catch (_) {
    return normalizeOpenClawConfig(null)
  }
}

export const createDefaultAgentForm = ({ isAdmin = false, userId = null } = {}) => ({
  user_id: isAdmin ? userId : null,
  name: '',
  nickname: '',
  custom_prompt: getDefaultAgentPrompt(),
  llm_config_id: null,
  tts_config_id: null,
  voice: null,
  asr_speed: 'normal',
  memory_mode: 'short',
  speaker_chat_mode: 'off',
  knowledge_base_ids: [],
  mcp_service_names: '',
  openclaw_allowed: false,
  openclaw_enter_keywords: [...OPENCLAW_DEFAULT_ENTER_KEYWORDS],
  openclaw_exit_keywords: [...OPENCLAW_DEFAULT_EXIT_KEYWORDS]
})

export const agentToForm = (agent = {}, { isAdmin = false } = {}) => {
  const openclaw = parseOpenClawConfigFromAgent(agent)
  return {
    user_id: isAdmin ? agent.user_id || null : null,
    name: agent.name || '',
    nickname: agent.nickname || agent.name || '',
    custom_prompt: agent.custom_prompt || getDefaultAgentPrompt(),
    llm_config_id: agent.llm_config_id || null,
    tts_config_id: agent.tts_config_id || null,
    voice: agent.voice || null,
    asr_speed: agent.asr_speed || 'normal',
    memory_mode: agent.memory_mode || 'short',
    speaker_chat_mode: agent.speaker_chat_mode || 'off',
    knowledge_base_ids: Array.isArray(agent.knowledge_base_ids) ? agent.knowledge_base_ids.map(Number).filter(Boolean) : [],
    mcp_service_names: agent.mcp_service_names || '',
    openclaw_allowed: !!openclaw.allowed,
    openclaw_enter_keywords: normalizeKeywordList(openclaw.enter_keywords),
    openclaw_exit_keywords: normalizeKeywordList(openclaw.exit_keywords)
  }
}

export const buildAgentPayload = (form = {}, { isAdmin = false } = {}) => {
  const name = String(form.name || '').trim()
  const nickname = String(form.nickname || '').trim() || name
  const payload = {
    name,
    nickname,
    custom_prompt: form.custom_prompt || '',
    llm_config_id: normalizeNullableString(form.llm_config_id),
    tts_config_id: normalizeNullableString(form.tts_config_id),
    voice: normalizeNullableString(form.voice),
    asr_speed: form.asr_speed || 'normal',
    memory_mode: form.memory_mode || 'short',
    speaker_chat_mode: form.speaker_chat_mode || 'off',
    knowledge_base_ids: Array.isArray(form.knowledge_base_ids) ? form.knowledge_base_ids.map(Number).filter(Boolean) : [],
    mcp_service_names: String(form.mcp_service_names || '').trim(),
    openclaw: {
      allowed: !!form.openclaw_allowed,
      enter_keywords: normalizeKeywordList(form.openclaw_enter_keywords),
      exit_keywords: normalizeKeywordList(form.openclaw_exit_keywords)
    }
  }
  if (isAdmin) {
    payload.user_id = Number(form.user_id || 0)
  }
  return payload
}

export const createDefaultDeviceForm = ({ isAdmin = false, userId = null, mode = 'create', fixedAgentId = null } = {}) => ({
  user_id: isAdmin ? userId : null,
  nick_name: '',
  device_code: '',
  device_name: '',
  identifier: '',
  activated: true,
  agent_id: fixedAgentId || 0,
  mode
})

export const deviceToForm = (device = {}, { isAdmin = false } = {}) => ({
  user_id: isAdmin ? device.user_id || null : null,
  nick_name: device.nick_name || '',
  device_code: device.device_code || '',
  device_name: device.device_name || '',
  identifier: '',
  activated: device.activated !== false,
  agent_id: device.agent_id || 0,
  mode: 'edit'
})

export const buildDevicePayload = (form = {}, { isAdmin = false, mode = 'create' } = {}) => {
  if (mode === 'bind') {
    const identifier = String(form.identifier || '').trim()
    const payload = /^\d{6}$/.test(identifier)
      ? { code: identifier }
      : { device_mac: identifier }
    const nickName = String(form.nick_name || '').trim()
    if (nickName) payload.nick_name = nickName
    return payload
  }

  const payload = {
    nick_name: String(form.nick_name || '').trim(),
    device_code: String(form.device_code || '').trim(),
    device_name: String(form.device_name || '').trim(),
    agent_id: Number(form.agent_id || 0)
  }
  if (isAdmin) {
    payload.user_id = Number(form.user_id || 0)
    payload.activated = form.activated !== false
  }
  return payload
}

export function useAgentFormOptions(options = {}) {
  const isAdmin = computed(() => !!unref(options.isAdmin))
  const targetUserId = computed(() => Number(unref(options.targetUserId) || 0))

  const users = ref([])
  const agents = ref([])
  const llmConfigs = ref([])
  const ttsConfigs = ref([])
  const knowledgeBases = ref([])
  const mcpServiceOptions = ref([])
  const voiceOptions = ref([])
  const cloneVoices = ref([])
  const loading = ref({
    users: false,
    agents: false,
    configs: false,
    knowledgeBases: false,
    mcpServices: false,
    voices: false,
    cloneVoices: false
  })

  const apiBase = computed(() => isAdmin.value ? '/admin' : '/user')

  const sortConfigs = (items) => {
    return [...items].sort((a, b) => {
      if (!!a.is_default !== !!b.is_default) return a.is_default ? -1 : 1
      return String(a.name || '').localeCompare(String(b.name || ''), 'zh-CN')
    })
  }

  const loadUsers = async () => {
    if (!isAdmin.value) {
      users.value = []
      return []
    }
    loading.value.users = true
    try {
      const response = await api.get('/admin/users')
      users.value = normalizeDataList(response)
      return users.value
    } finally {
      loading.value.users = false
    }
  }

  const loadConfigs = async () => {
    loading.value.configs = true
    try {
      const [llmResponse, ttsResponse] = await Promise.all([
        api.get(`${apiBase.value}/llm-configs`),
        api.get(`${apiBase.value}/tts-configs`)
      ])
      llmConfigs.value = sortConfigs(normalizeDataList(llmResponse))
      ttsConfigs.value = sortConfigs(normalizeDataList(ttsResponse))
      return { llmConfigs: llmConfigs.value, ttsConfigs: ttsConfigs.value }
    } finally {
      loading.value.configs = false
    }
  }

  const loadAgents = async () => {
    loading.value.agents = true
    try {
      const response = await api.get(`${apiBase.value}/agents`)
      const items = normalizeDataList(response)
      agents.value = isAdmin.value && targetUserId.value
        ? items.filter((agent) => Number(agent.user_id) === targetUserId.value)
        : items
      return agents.value
    } finally {
      loading.value.agents = false
    }
  }

  const loadKnowledgeBases = async () => {
    loading.value.knowledgeBases = true
    try {
      if (isAdmin.value) {
        if (!targetUserId.value) {
          knowledgeBases.value = []
          return knowledgeBases.value
        }
        const response = await api.get(`/admin/users/${targetUserId.value}/knowledge-bases`)
        knowledgeBases.value = normalizeDataList(response)
        return knowledgeBases.value
      }
      const response = await api.get('/user/knowledge-bases')
      knowledgeBases.value = normalizeDataList(response)
      return knowledgeBases.value
    } finally {
      loading.value.knowledgeBases = false
    }
  }

  const loadMcpServiceOptions = async () => {
    loading.value.mcpServices = true
    try {
      const response = await api.get('/user/mcp-services/options')
      mcpServiceOptions.value = response.data?.data?.options || []
      return mcpServiceOptions.value
    } finally {
      loading.value.mcpServices = false
    }
  }

  const loadVoiceOptions = async ({ provider, configId }) => {
    const normalizedProvider = String(provider || '').trim()
    if (!normalizedProvider) {
      voiceOptions.value = []
      return []
    }
    loading.value.voices = true
    try {
      const params = { provider: normalizedProvider }
      if (configId) params.config_id = configId
      const url = isAdmin.value && targetUserId.value
        ? `/admin/users/${targetUserId.value}/voice-options`
        : '/user/voice-options'
      const response = await api.get(url, { params })
      voiceOptions.value = normalizeDataList(response)
      return voiceOptions.value
    } finally {
      loading.value.voices = false
    }
  }

  const loadCloneVoices = async (ttsConfigId = '') => {
    loading.value.cloneVoices = true
    try {
      const params = {}
      if (ttsConfigId) params.tts_config_id = ttsConfigId
      const url = isAdmin.value && targetUserId.value
        ? `/admin/users/${targetUserId.value}/voice-clones`
        : '/user/voice-clones'
      const response = await api.get(url, { params })
      cloneVoices.value = normalizeDataList(response)
        .filter((clone) => {
          const status = String(clone.status || '').toLowerCase()
          const taskStatus = String(clone.task_status || '').toLowerCase()
          return status === 'active' || taskStatus === 'succeeded'
        })
        .filter((clone) => clone.provider_voice_id && clone.tts_config_id)
      return cloneVoices.value
    } finally {
      loading.value.cloneVoices = false
    }
  }

  return {
    users,
    agents,
    llmConfigs,
    ttsConfigs,
    knowledgeBases,
    mcpServiceOptions,
    voiceOptions,
    cloneVoices,
    loading,
    loadUsers,
    loadConfigs,
    loadAgents,
    loadKnowledgeBases,
    loadMcpServiceOptions,
    loadVoiceOptions,
    loadCloneVoices
  }
}
