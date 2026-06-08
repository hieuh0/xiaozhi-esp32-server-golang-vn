export interface Device {
  id: number
  user_id?: number
  agent_id?: number | null
  agent_name?: string
  device_name: string
  nick_name?: string
  device_code?: string
  activated?: boolean
  last_active_at?: string | null
  role_id?: number | null
  created_at: string
  updated_at?: string
}

export interface DeviceFormData {
  user_id?: number | null
  agent_id: number | null
  identifier: string
  nick_name: string
  device_name?: string
  device_code?: string
}

export interface McpTool {
  name: string
  description?: string
  input_schema?: Record<string, unknown>
}

export interface Role {
  id: number
  name: string
  role_type: 'global' | 'user'
  status?: string
  prompt?: string
}

export const createDefaultDeviceForm = ({ fixedAgentId = null as number | null } = {}): DeviceFormData => ({
  user_id: null,
  agent_id: fixedAgentId,
  identifier: '',
  nick_name: '',
})

export const isDeviceOnline = (lastActiveAt?: string | null): boolean => {
  if (!lastActiveAt) return false
  return Date.now() - new Date(lastActiveAt).getTime() < 5 * 60 * 1000
}

export const getDeviceDisplayName = (device: Device, fallback = 'Unnamed device'): string => {
  const nick = String(device.nick_name || '').trim()
  if (nick) return nick
  return String(device.device_name || '').trim() || fallback
}
