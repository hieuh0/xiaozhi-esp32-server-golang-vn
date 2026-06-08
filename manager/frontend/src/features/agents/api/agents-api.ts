import api from '@/utils/api'
import type { Agent, LLMConfig, TTSConfig, VoiceOption, CloneVoice, KnowledgeBase, UserOption } from '../types'

const list = <T>(res: { data: { data?: T[] | { items?: T[] } } }): T[] => {
  const d = res.data?.data
  if (Array.isArray(d)) return d
  if (d && Array.isArray((d as { items?: T[] }).items)) return (d as { items: T[] }).items
  return []
}

export const agentsApi = {
  // ── User agents ─────────────────────────────────────────────────────────
  getUserAgents: async (): Promise<Agent[]> => list(await api.get('/user/agents')),

  getAgent: async (id: number | string): Promise<Agent> => {
    const { data } = await api.get<{ data: Agent }>(`/user/agents/${id}`)
    return data.data
  },

  createAgent: async (payload: Record<string, unknown>): Promise<Agent> => {
    const { data } = await api.post<{ data: Agent }>('/user/agents', payload)
    return data.data
  },

  updateAgent: async (id: number | string, payload: Record<string, unknown>): Promise<void> => {
    await api.put(`/user/agents/${id}`, payload)
  },

  deleteAgent: async (id: number | string): Promise<void> => {
    await api.delete(`/user/agents/${id}`)
  },

  // ── Admin agents ─────────────────────────────────────────────────────────
  getAdminAgents: async (page = 1, pageSize = 20): Promise<{ items: Agent[]; total: number }> => {
    const { data } = await api.get<{ data: Agent[]; total: number }>('/admin/agents', { params: { page, page_size: pageSize } })
    return { items: data.data || [], total: data.total || 0 }
  },

  adminUpdateAgent: async (id: number | string, payload: Record<string, unknown>): Promise<void> => {
    await api.put(`/admin/agents/${id}`, payload)
  },

  adminDeleteAgent: async (id: number | string): Promise<void> => {
    await api.delete(`/admin/agents/${id}`)
  },

  // ── History ──────────────────────────────────────────────────────────────
  getAgentHistory: async (
    agentId: number | string,
    params: { page?: number; page_size?: number; role?: string; device_id?: string; start_date?: string; end_date?: string } = {}
  ): Promise<{ messages: { id: number; role: string; content: string; device_id?: string; created_at: string; audio_url?: string }[]; total: number }> => {
    const { data } = await api.get(`/user/agents/${agentId}/history`, { params })
    return { messages: data.data || [], total: data.total || 0 }
  },

  exportHistory: async (agentId: number | string, params: Record<string, string> = {}): Promise<Blob> => {
    const res = await api.get(`/user/agents/${agentId}/history/export`, { params, responseType: 'blob' })
    return res.data as Blob
  },

  getAgentDevices: async (agentId: number | string): Promise<{ id: number; mac: string; name?: string }[]> => {
    const { data } = await api.get<{ data: { id: number; mac: string; name?: string }[] }>(`/user/agents/${agentId}/devices`)
    return data.data || []
  },

  // ── Form option loaders ──────────────────────────────────────────────────
  getLLMConfigs: async (): Promise<LLMConfig[]> => list(await api.get('/user/llm-configs/options')),

  getTTSConfigs: async (): Promise<TTSConfig[]> => list(await api.get('/user/tts-configs/options')),

  getVoiceOptions: async (provider: string, configId: string): Promise<VoiceOption[]> => {
    const { data } = await api.get<{ data: { options?: VoiceOption[] } }>('/user/tts-configs/voices', { params: { provider, config_id: configId } })
    return data.data?.options || []
  },

  getCloneVoices: async (ttsConfigId?: string | null): Promise<CloneVoice[]> => {
    const params = ttsConfigId ? { tts_config_id: ttsConfigId } : {}
    const { data } = await api.get<{ data: CloneVoice[] }>('/user/voice-clones', { params })
    return data.data || []
  },

  getKnowledgeBases: async (userId?: number | null): Promise<KnowledgeBase[]> => {
    const params = userId ? { user_id: userId } : {}
    return list(await api.get('/user/knowledge-bases', { params }))
  },

  getMcpServiceOptions: async (): Promise<string[]> => {
    const { data } = await api.get<{ data: { options?: string[] } }>('/user/mcp-services/options')
    return data.data?.options || []
  },

  getAdminUsers: async (): Promise<UserOption[]> => list(await api.get('/admin/users/options')),

  getRoles: async (): Promise<{ global_roles: import('../types').Role[]; user_roles: import('../types').Role[] }> => {
    const { data } = await api.get<{ data: { global_roles: import('../types').Role[]; user_roles: import('../types').Role[] } }>('/user/roles')
    return data.data || { global_roles: [], user_roles: [] }
  },
}
