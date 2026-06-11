import { useCallback, useState } from 'react'
import { agentsApi } from '@/features/agents/api/agents-api'
import type { LLMConfig, TTSConfig, VoiceOption, CloneVoice, KnowledgeBase, UserOption } from '@/features/agents/types'

export interface AgentFormOptions {
  llmConfigs: LLMConfig[]
  ttsConfigs: TTSConfig[]
  voiceOptions: VoiceOption[]
  cloneVoices: CloneVoice[]
  knowledgeBases: KnowledgeBase[]
  mcpServiceOptions: string[]
  users: UserOption[]
  loading: boolean
  load: (opts: { isAdmin?: boolean; userId?: number | null; ttsConfigId?: string | null }) => Promise<void>
  refreshVoices: (opts: { ttsConfigId?: string | null; provider?: string; clearInvalid?: boolean; previousConfigId?: string | null }) => Promise<VoiceOption[]>
}

export function useAgentFormOptions(): AgentFormOptions {
  const [llmConfigs, setLlmConfigs] = useState<LLMConfig[]>([])
  const [ttsConfigs, setTtsConfigs] = useState<TTSConfig[]>([])
  const [voiceOptions, setVoiceOptions] = useState<VoiceOption[]>([])
  const [cloneVoices, setCloneVoices] = useState<CloneVoice[]>([])
  const [knowledgeBases, setKnowledgeBases] = useState<KnowledgeBase[]>([])
  const [mcpServiceOptions, setMcpServiceOptions] = useState<string[]>([])
  const [users, setUsers] = useState<UserOption[]>([])
  const [loading, setLoading] = useState(false)

  const refreshVoices = useCallback(async ({ ttsConfigId, provider }: {
    ttsConfigId?: string | null; provider?: string; clearInvalid?: boolean; previousConfigId?: string | null
  }): Promise<VoiceOption[]> => {
    if (!ttsConfigId || !provider) {
      setVoiceOptions([]); setCloneVoices([])
      return []
    }
    const [voices, clones] = await Promise.all([
      agentsApi.getVoiceOptions(provider, ttsConfigId).catch(() => []),
      agentsApi.getCloneVoices(ttsConfigId).catch(() => []),
    ])
    setVoiceOptions(voices)
    setCloneVoices(clones)
    return voices
  }, [])

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
    llmConfigs, ttsConfigs, voiceOptions, cloneVoices,
    knowledgeBases, mcpServiceOptions, users, loading, load, refreshVoices,
  }
}
