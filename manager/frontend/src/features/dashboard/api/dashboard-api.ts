import api from '@/utils/api'
import type { DashboardStats, ServiceAddress, PoolStatsData, PoolSummary } from '../types'

type OtaConfig = { is_default: boolean; json_data: string }
type UdpConfig = { is_default: boolean; json_data: string }

export const dashboardApi = {
  getStats: async (): Promise<DashboardStats> => {
    const { data } = await api.get<DashboardStats>('/dashboard/stats')
    return { ...{ totalUsers: 0, totalDevices: 0, totalAgents: 0, onlineDevices: 0 }, ...data }
  },

  getServiceAddress: async (): Promise<ServiceAddress> => {
    const [otaRes, udpRes] = await Promise.all([
      api.get<{ data: OtaConfig[] }>('/admin/ota-configs'),
      api.get<{ data: UdpConfig[] }>('/admin/udp-configs'),
    ])
    const result: ServiceAddress = { otaUrl: '', wsUrl: '', mqttEndpoint: '', udpAddress: '' }
    const otaList = otaRes.data?.data || []
    const config = otaList.find((c) => c.is_default) || otaList[0]
    if (config?.json_data) {
      const parsed = JSON.parse(config.json_data || '{}')
      let env = parsed.external || {}
      if (!env.websocket?.url && !env.ota_url) env = parsed.test || {}
      let otaUrl = env.ota_url || ''
      if (!otaUrl && env.websocket?.url) {
        const m = (env.websocket.url as string).match(/^(wss?):\/\/([^:/]+)(?::(\d+))?/)
        if (m) {
          const proto = m[1] === 'wss' ? 'https' : 'http'
          const port = m[3] || (m[1] === 'wss' ? '443' : '80')
          otaUrl = `${proto}://${m[2]}:${port}/xiaozhi/ota/`
        }
      }
      result.otaUrl = otaUrl
      result.wsUrl = env.websocket?.url || ''
      if (env.mqtt?.enable && env.mqtt?.endpoint) result.mqttEndpoint = env.mqtt.endpoint
    }
    const udpList = udpRes.data?.data || []
    const udpConfig = udpList.find((c) => c.is_default) || udpList[0]
    if (udpConfig?.json_data) {
      const d = JSON.parse(udpConfig.json_data || '{}')
      if (d.external_host && d.external_port != null) result.udpAddress = `${d.external_host}:${d.external_port}`
    }
    return result
  },

  getPoolStats: async (): Promise<PoolStatsData | null> => {
    const { data } = await api.get<{ data: PoolStatsData }>('/admin/pool/stats?type=latest')
    return data?.data || null
  },

  getPoolSummary: async (): Promise<PoolSummary> => {
    const { data } = await api.get<{ data: PoolSummary }>('/admin/pool/stats/summary')
    return data?.data || { total_records: 0, oldest_timestamp: null, newest_timestamp: null }
  },

  exportConfig: async (token: string): Promise<void> => {
    const res = await fetch('/api/admin/configs/export', { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) throw new Error('export_failed')
    const url = URL.createObjectURL(await res.blob())
    const a = Object.assign(document.createElement('a'), { href: url, download: 'config.yaml' })
    document.body.appendChild(a); a.click(); URL.revokeObjectURL(url); document.body.removeChild(a)
  },

  importConfig: async (token: string, file: File): Promise<void> => {
    const fd = new FormData(); fd.append('file', file)
    const res = await fetch('/api/admin/configs/import', { method: 'POST', headers: { Authorization: `Bearer ${token}` }, body: fd })
    if (!res.ok) { const err = await res.json().catch(() => ({})); throw new Error((err as { error?: string }).error || 'import_failed') }
  },

  runOtaTest: async (): Promise<{ ok: boolean; message: string; text: string }> => {
    const { data } = await api.post<{ data: unknown }>('/admin/configs/test', { types: ['ota'] }, { timeout: 30000 })
    const d = (data?.data ?? data) as Record<string, unknown>
    const ota = d?.ota as Record<string, unknown> | undefined
    if (ota) {
      const entry = Object.entries(ota).find(([k]) => !k.startsWith('_'))
      if (entry) {
        const v = entry[1] as Record<string, { ok?: boolean; message?: string; first_packet_ms?: number }>
        let txt = ''
        const ws = v.websocket; if (ws) txt += `WebSocket: ${ws.ok ? '✓' : '✗'} ${ws.message}${ws.first_packet_ms != null ? ` (${ws.first_packet_ms}ms)` : ''}\n`
        const m = v.mqtt_udp; if (m) txt += `MQTT UDP: ${m.ok ? '✓' : '✗'} ${m.message}${m.first_packet_ms != null ? ` (${m.first_packet_ms}ms)` : ''}\n`
        const or = (v as unknown as { ota_response?: unknown }).ota_response
        if (or !== undefined && or !== '') { try { txt += `\n--- OTA ---\n${JSON.stringify(JSON.parse(String(or)), null, 2)}` } catch { txt += `\n--- OTA ---\n${String(or)}` } }
        return { ok: !!(v as unknown as { ok?: boolean }).ok, message: (v as unknown as { message?: string }).message || '', text: txt.trim() }
      }
    }
    return { ok: false, message: '', text: typeof d === 'string' ? d : JSON.stringify(d || {}, null, 2) }
  },
}
