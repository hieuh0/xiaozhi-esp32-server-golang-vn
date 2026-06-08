import { useCallback, useRef, useState } from 'react'
import { agentsApi } from '@/features/agents/api/agents-api'
import type { LLMConfig, TTSConfig, VoiceOption, CloneVoice, KnowledgeBase, UserOption } from '@/features/agents/types'

export interface AgentFormOptions {
  llmConfigs: LLMConfig[]
  ttsConfigs: TTSConfig[]
  voiceOptions: VoiceOption[]
  filteredVoiceOptions: VoiceOption[]
  cloneVoices: CloneVoice[]
  knowledgeBases: KnowledgeBase[]
  mcpServiceOptions: string[]
  users: UserOption[]
  loading: boolean
  voiceSearch: string
  setVoiceSearch: (q: string) => void
  load: (opts: { isAdmin?: boolean; userId?: number | null; ttsConfigId?: string | null }) => Promise<void>
  refreshVoices: (opts: { ttsConfigId?: string | null; provider?: string; clearInvalid?: boolean; previousConfigId?: string | null }) => Promise<VoiceOption[]>
}

const MAX_VOICE = 300

function filterVoices(all: VoiceOption[], keyword: string, selected: string | null): VoiceOption[] {
  const kw = keyword.trim().toLowerCase()
  const list = kw ? all.filter((v) => v.label.toLowerCase().includes(kw) || v.value.toLowerCase().includes(kw)) : all
  const visible = list.slice(0, MAX_VOICE)
  if (selected && !visible.some((v) => v.value === selected)) {
    const opt = list.find((v) => v.value === selected)
    if (opt) visible.unshift(opt)
  }
  return visible
}

export function useAgentFormOptions(): AgentFormOptions {
  const [llmConfigs, setLlmConfigs] = useState<LLMConfig[]>([])
  const [ttsConfigs, setTtsConfigs] = useState<TTSConfig[]>([])
  const [voiceOptions, setVoiceOptions] = useState<VoiceOption[]>([])
  const [filteredVoiceOptions, setFilteredVoiceOptions] = useState<VoiceOption[]>([])
  const [cloneVoices, setCloneVoices] = useState<CloneVoice[]>([])
  const [knowledgeBases, setKnowledgeBases] = useState<KnowledgeBase[]>([])
  const [mcpServiceOptions, setMcpServiceOptions] = useState<string[]>([])
  const [users, setUsers] = useState<UserOption[]>([])
  const [loading, setLoading] = useState(false)
  const [voiceSearch, setVoiceSearchState] = useState('')
  const selectedVoiceRef = useRef<string | null>(null)

  const setVoiceSearch = useCallback((q: string) => {
    setVoiceSearchState(q)
    setFilteredVoiceOptions(filterVoices(voiceOptions, q, selectedVoiceRef.current))
  }, [voiceOptions])

  const refreshVoices = useCallback(async ({ ttsConfigId, provider }: {
    ttsConfigId?: string | null; provider?: string; clearInvalid?: boolean; previousConfigId?: string | null
  }): Promise<VoiceOption[]> => {
    if (!ttsConfigId || !provider) {
      setVoiceOptions([]); setFilteredVoiceOptions([]); setCloneVoices([])
      return []
    }
    const [voices, clones] = await Promise.all([
      agentsApi.getVoiceOptions(provider, ttsConfigId).catch(() => []),
      agentsApi.getCloneVoices(ttsConfigId).catch(() => []),
    ])
    setVoiceOptions(voices)
    setFilteredVoiceOptions(filterVoices(voices, voiceSearch, selectedVoiceRef.current))
    setCloneVoices(clones)
    return voices
  }, [voiceSearch])

  const load = useCallback(async ({ isAdmin = false, userId = null }: {
    isAdmin?: boolean; userId?: number | null; ttsConfigId?: string | null
  }) => {
    setLoading(true)
    try {
      const [llm, tts, kb, mcp, userList] = await Promise.all([
        agentsApi.getLLMConfigs().catch(() => []),
        agentsApi.getTTSConfigs().catch(() => []),
        agentsApi.getKnowledgeBases(userId).catch(() => []),
        agentsApi.getMcpServiceOptions().catch(() => []),
        isAdmin ? agentsApi.getAdminUsers().catch(() => []) : Promise.resolve<UserOption[]>([]),
      ])
      setLlmConfigs(llm)
      setTtsConfigs(tts)
      setKnowledgeBases(kb)
      setMcpServiceOptions(mcp)
      setUsers(userList)
    } finally {
      setLoading(false)
    }
  }, [])

  return {
    llmConfigs, ttsConfigs, voiceOptions, filteredVoiceOptions, cloneVoices,
    knowledgeBases, mcpServiceOptions, users, loading, voiceSearch,
    setVoiceSearch, load, refreshVoices,
  }
}
