import api from '@/utils/api'
import type { Device, McpTool, Role } from '../types'

// Backend may return null for optional string fields; coerce to undefined so
// downstream code can safely use || and ?? without crashing on .trim() etc.
const norm = (d: Device): Device => ({ ...d, nick_name: d.nick_name ?? undefined })

export const devicesApi = {
  // ── User devices ─────────────────────────────────────────────────────────
  getUserDevices: async (): Promise<Device[]> => {
    const { data } = await api.get<{ data: Device[] }>('/user/devices')
    return (data.data || []).map(norm)
  },

  addDevice: async (agentId: number | string, payload: { code?: string; device_mac?: string; nick_name?: string }): Promise<void> => {
    await api.post(`/user/agents/${agentId}/devices`, payload)
  },

  updateDevice: async (deviceId: number | string, payload: Record<string, unknown>): Promise<Device> => {
    const { data } = await api.put<{ data: Device }>(`/user/devices/${deviceId}`, payload)
    return data.data
  },

  deleteDevice: async (deviceId: number | string): Promise<void> => {
    await api.delete(`/user/devices/${deviceId}`)
  },

  applyRole: async (deviceId: number | string, roleId: number | null): Promise<void> => {
    await api.post(`/devices/${deviceId}/apply-role`, { role_id: roleId })
  },

  injectMessage: async (payload: { device_id: string; message: string; skip_llm: boolean; auto_listen: boolean }): Promise<void> => {
    await api.post('/user/devices/inject-message', payload)
  },

  // ── MCP ──────────────────────────────────────────────────────────────────
  getDeviceMcpTools: async (deviceId: number | string): Promise<McpTool[]> => {
    const { data } = await api.get<{ data: { tools: McpTool[] } }>(`/user/devices/${deviceId}/mcp-tools`)
    return data.data?.tools || []
  },

  callDeviceMcpTool: async (deviceId: number | string, toolName: string, args: Record<string, unknown>): Promise<unknown> => {
    const { data } = await api.post<{ data: unknown }>(`/user/devices/${deviceId}/mcp-call`, { tool_name: toolName, arguments: args })
    return data.data
  },

  // ── Roles ─────────────────────────────────────────────────────────────────
  getRoles: async (): Promise<Role[]> => {
    const { data } = await api.get<{ data: { global_roles: Role[]; user_roles: Role[] } }>('/user/roles')
    return [...(data.data?.global_roles || []), ...(data.data?.user_roles || [])].filter((r) => !r.status || r.status === 'active')
  },

  // ── Admin devices ─────────────────────────────────────────────────────────
  getAdminDevices: async (page = 1, pageSize = 20): Promise<{ items: Device[]; total: number }> => {
    const { data } = await api.get<{ data: Device[]; total: number }>('/admin/devices', { params: { page, page_size: pageSize } })
    return { items: (data.data || []).map(norm), total: data.total || 0 }
  },

  adminUpdateDevice: async (deviceId: number | string, payload: Record<string, unknown>): Promise<void> => {
    await api.put(`/admin/devices/${deviceId}`, payload)
  },

  adminDeleteDevice: async (deviceId: number | string): Promise<void> => {
    await api.delete(`/admin/devices/${deviceId}`)
  },
}
