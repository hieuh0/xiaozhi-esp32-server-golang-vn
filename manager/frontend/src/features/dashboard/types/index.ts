export interface DashboardStats {
  totalUsers: number
  totalDevices: number
  totalAgents: number
  onlineDevices: number
  programStartedAt?: string
}

export interface ServiceAddress {
  otaUrl: string
  wsUrl: string
  mqttEndpoint: string
  udpAddress: string
}

export interface PoolSummary {
  total_records: number
  oldest_timestamp: string | null
  newest_timestamp: string | null
}

export interface PoolStatEntry {
  poolKey: string
  total: number
  available: number
  inUse: number
  maxSize: number
  minSize: number
  maxIdle: number
  isClosed: boolean
}

export interface PoolStatsData {
  timestamp: string
  stats: Record<string, {
    total_resources: number
    available_resources: number
    in_use_resources: number
    max_size: number
    min_size: number
    max_idle: number
    is_closed: boolean
  }>
}
