export interface Agent {
  id: number
  user_id: number
  name: string
  nickname: string
  custom_prompt: string
  llm_config_id: string | null
  llm_config: { name?: string; provider?: string } | null
  tts_config_id: string | null
  tts_config: { name?: string; provider?: string } | null
  voice: string | null
  asr_speed: string
  memory_mode: string
  speaker_chat_mode: string
  knowledge_base_ids: number[]
  mcp_service_names: string
  openclaw: { allowed: boolean; enter_keywords: string[]; exit_keywords: string[] } | null
  openclaw_config: string | null
  created_at: string
  updated_at: string
}

export interface AgentFormData {
  user_id: number | null
  name: string
  nickname: string
  custom_prompt: string
  llm_config_id: string | null
  tts_config_id: string | null
  voice: string | null
  asr_speed: 'normal' | 'patient' | 'fast'
  memory_mode: 'none' | 'short' | 'long'
  speaker_chat_mode: 'off' | 'identified_only'
  knowledge_base_ids: number[]
  mcp_service_names: string
  openclaw_allowed: boolean
  openclaw_enter_keywords: string[]
  openclaw_exit_keywords: string[]
}

export interface LLMConfig { config_id: string; name: string; is_default: boolean; provider?: string }
export interface TTSConfig { config_id: string; name: string; is_default: boolean; provider?: string }
export interface VoiceOption { value: string; label: string }
export interface CloneVoice { id: number; name: string; tts_config_id: string; tts_config_name?: string; provider_voice_id: string }
export interface KnowledgeBase { id: number; name: string }
export interface UserOption { id: number; username?: string; name?: string }
export interface Role { id: number; name: string; role_type: 'global' | 'user'; status?: string; prompt?: string; llm_config_id?: string; tts_config_id?: string; voice?: string }

export interface AgentHistoryMessage {
  id: number
  role: string
  content: string
  device_id?: string
  created_at: string
  audio_url?: string
}

export const OPENCLAW_DEFAULT_ENTER_KEYWORDS = ['open openclaw', 'enter openclaw']
export const OPENCLAW_DEFAULT_EXIT_KEYWORDS = ['close openclaw', 'exit openclaw']

export const createDefaultAgentForm = ({ isAdmin = false, userId = null as number | null } = {}): AgentFormData => ({
  user_id: isAdmin ? userId : null,
  name: '',
  nickname: '',
  custom_prompt: '',
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
  openclaw_exit_keywords: [...OPENCLAW_DEFAULT_EXIT_KEYWORDS],
})

export const agentToForm = (agent: Partial<Agent>, { isAdmin = false } = {}): AgentFormData => {
  let openclaw = { allowed: false, enter_keywords: [...OPENCLAW_DEFAULT_ENTER_KEYWORDS], exit_keywords: [...OPENCLAW_DEFAULT_EXIT_KEYWORDS] }
  if (agent.openclaw && typeof agent.openclaw === 'object') openclaw = { ...openclaw, ...agent.openclaw }
  else if (typeof agent.openclaw_config === 'string' && agent.openclaw_config.trim()) {
    try { openclaw = { ...openclaw, ...JSON.parse(agent.openclaw_config) } } catch { /* ignore */ }
  }
  return {
    user_id: isAdmin ? (agent.user_id ?? null) : null,
    name: agent.name || '',
    nickname: agent.nickname || agent.name || '',
    custom_prompt: agent.custom_prompt || '',
    llm_config_id: agent.llm_config_id || null,
    tts_config_id: agent.tts_config_id || null,
    voice: agent.voice || null,
    asr_speed: (agent.asr_speed as AgentFormData['asr_speed']) || 'normal',
    memory_mode: (agent.memory_mode as AgentFormData['memory_mode']) || 'short',
    speaker_chat_mode: (agent.speaker_chat_mode as AgentFormData['speaker_chat_mode']) || 'off',
    knowledge_base_ids: Array.isArray(agent.knowledge_base_ids) ? agent.knowledge_base_ids.map(Number).filter(Boolean) : [],
    mcp_service_names: agent.mcp_service_names || '',
    openclaw_allowed: !!openclaw.allowed,
    openclaw_enter_keywords: openclaw.enter_keywords?.length ? openclaw.enter_keywords : [...OPENCLAW_DEFAULT_ENTER_KEYWORDS],
    openclaw_exit_keywords: openclaw.exit_keywords?.length ? openclaw.exit_keywords : [...OPENCLAW_DEFAULT_EXIT_KEYWORDS],
  }
}

export const buildAgentPayload = (form: AgentFormData, { isAdmin = false } = {}) => {
  const name = form.name.trim()
  const payload: Record<string, unknown> = {
    name,
    nickname: form.nickname.trim() || name,
    custom_prompt: form.custom_prompt || '',
    llm_config_id: form.llm_config_id || null,
    tts_config_id: form.tts_config_id || null,
    voice: form.voice || null,
    asr_speed: form.asr_speed || 'normal',
    memory_mode: form.memory_mode || 'short',
    speaker_chat_mode: form.speaker_chat_mode || 'off',
    knowledge_base_ids: form.knowledge_base_ids.map(Number).filter(Boolean),
    mcp_service_names: form.mcp_service_names.trim(),
    openclaw: { allowed: !!form.openclaw_allowed, enter_keywords: form.openclaw_enter_keywords, exit_keywords: form.openclaw_exit_keywords },
  }
  if (isAdmin) payload.user_id = form.user_id
  return payload
}
