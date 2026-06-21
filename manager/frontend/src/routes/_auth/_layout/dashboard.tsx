import { Suspense, useRef, useState } from 'react'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { useSuspenseQuery, useQuery } from '@tanstack/react-query'
import { User, Monitor, Cpu, Wifi, Wand2, Download, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { dashboardApi } from '@/features/dashboard/api/dashboard-api'
import type { DashboardStats, PoolStatsData, PoolStatEntry } from '@/features/dashboard/types'
import { DashboardServiceCard } from '@/components/dashboard/dashboard-service-card'
import { DashboardChartsRow } from '@/components/dashboard/dashboard-charts-row'
import { DashboardPageHeader } from '@/components/dashboard/dashboard-page-header'
import { DashboardRightCol } from '@/components/dashboard/dashboard-right-col'
import type { DashboardPeriod } from '@/components/dashboard/dashboard-page-header'
import { KpiSparkline } from '@/components/charts'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { SectionLabel } from '@/components/ui/section-label'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

function StatCard({ icon: Icon, iconClass, sparkColor, trend, value, label }: {
  icon: React.ElementType; iconClass: string; sparkColor: string; trend: string; value: number; label: string
}) {
  return (
    <div
      className="relative p-5 rounded-2xl border border-[var(--color-line)] overflow-hidden group transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[var(--shadow-card-hover)]"
      style={{ background: 'linear-gradient(145deg, var(--color-surface-1) 0%, var(--color-bg) 100%)', boxShadow: 'var(--shadow-card)' }}
    >
      <div className="absolute -right-4 -top-4 w-24 h-24 rounded-full blur-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300" style={{ background: 'var(--color-primary-soft)' }} />
      <div className="relative">
        <div className="flex items-center justify-between mb-4">
          <span className={cn('w-10 h-10 rounded-lg inline-flex items-center justify-center', iconClass)}><Icon className="w-5 h-5" /></span>
          <span className="text-[10px] font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] font-mono">{trend}</span>
        </div>
        <strong className="block text-4xl font-bold font-display tracking-tight leading-none text-[var(--color-text)]">{value}</strong>
        <p className="mt-2.5 text-sm text-[var(--color-text-secondary)]">{label}</p>
        <KpiSparkline value={value} color={sparkColor} />
      </div>
    </div>
  )
}

function ConfigActionRow({ icon: Icon, title, desc, highlighted, onClick }: {
  icon: React.ElementType; title: string; desc: string; highlighted?: boolean; onClick: () => void
}) {
  return (
    <button type="button" onClick={onClick}
      className={cn('flex items-center gap-3.5 w-full p-4 rounded-xl border cursor-pointer hover:-translate-y-px hover:shadow-[var(--shadow-card-hover)] active:scale-[0.98] transition-all duration-150 text-left',
        highlighted ? 'border-[var(--color-primary)]/20 bg-[var(--color-primary-soft)]' : 'border-[var(--color-line)] bg-[var(--color-surface-1)] shadow-[var(--shadow-card)]')}>
      <span className="w-10 h-10 rounded-xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><Icon className="w-5 h-5" /></span>
      <span className="flex flex-col gap-0.5">
        <strong className="text-[15px] text-[var(--color-text)]">{title}</strong>
        <small className="text-sm text-[var(--color-text-secondary)]">{desc}</small>
      </span>
    </button>
  )
}

function DashboardInner() {
  const { t } = useLocale()
  const router = useRouter()
  const { isAdmin, token } = useAuthStore()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [importing, setImporting] = useState(false)
  const [poolRefreshing, setPoolRefreshing] = useState(false)
  const [period, setPeriod] = useState<DashboardPeriod>('today')

  const { data: stats } = useSuspenseQuery<DashboardStats>({
    queryKey: ['dashboard-stats'],
    queryFn: dashboardApi.getStats,
    staleTime: 30_000,
  })

  const { data: poolStats, refetch: refetchPool } = useQuery<PoolStatsData | null>({
    queryKey: ['pool-stats'],
    queryFn: dashboardApi.getPoolStats,
    staleTime: 25_000,
    enabled: isAdmin,
  })

  const poolData: PoolStatEntry[] = poolStats?.stats
    ? Object.entries(poolStats.stats)
        .filter(([, v]) => v && typeof v === 'object')
        .map(([poolKey, s]) => ({
          poolKey, total: s.total_resources || 0, available: s.available_resources || 0,
          inUse: s.in_use_resources || 0, maxSize: s.max_size || 0,
          minSize: s.min_size || 0, maxIdle: s.max_idle || 0, isClosed: s.is_closed || false,
        }))
    : []

  const programStartedAt = stats.programStartedAt ? new Date(stats.programStartedAt).toLocaleString() : '—'

  const metricCards = [
    { icon: User,    iconClass: 'text-[var(--color-primary)] bg-[var(--color-primary-soft)]',                                sparkColor: 'var(--color-chart-blue)',  trend: isAdmin ? t('global_user') : t('linked_account'),    value: isAdmin ? stats.totalUsers : 1, label: isAdmin ? t('total_users') : t('current_logged_account') },
    { icon: Monitor, iconClass: 'text-[var(--color-success)] bg-[color-mix(in_srgb,var(--color-success)_12%,transparent)]', sparkColor: 'var(--color-chart-green)', trend: t('online_devices'),                                  value: stats.totalDevices,             label: isAdmin ? t('total_devices') : t('my_devices') },
    { icon: Cpu,     iconClass: 'text-[var(--color-warning)] bg-[color-mix(in_srgb,var(--color-warning)_12%,transparent)]', sparkColor: 'var(--color-chart-amber)', trend: t('active'),                                          value: stats.totalAgents,              label: isAdmin ? t('agent_count') : t('my_agents') },
    { icon: Wifi,    iconClass: 'text-[var(--color-danger)] bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]',   sparkColor: 'var(--color-chart-red)',   trend: t('realtime_monitoring'),                             value: stats.onlineDevices,            label: t('online_devices') },
  ]

  const handleExport = async () => {
    try { await dashboardApi.exportConfig(token ?? ''); toast.success(t('config_export_success')) }
    catch { toast.error(t('config_export_failed')) }
  }

  const handleImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; if (!file) return
    const ext = file.name.toLowerCase().slice(file.name.lastIndexOf('.'))
    if (!['.yaml', '.yml', '.json'].includes(ext)) { toast.error(t('select_yaml_or_json')); return }
    setImporting(true)
    try { await dashboardApi.importConfig(token ?? '', file); toast.success(t('config_import_success')) }
    catch (err) { toast.error((err as Error).message || t('config_import_failed')) }
    finally { setImporting(false); e.target.value = '' }
  }

  const handleRefreshPool = async () => {
    setPoolRefreshing(true)
    await refetchPool()
    setPoolRefreshing(false)
  }

  return (
    <div className="grid gap-4">
      <DashboardPageHeader
        period={period}
        onPeriodChange={setPeriod}
        onExport={handleExport}
        isAdmin={isAdmin}
      />

      <div className="grid gap-4 px-6">
        {/* KPI cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {metricCards.map((m) => <StatCard key={m.label} {...m} />)}
        </div>

        {/* Charts row */}
        <DashboardChartsRow
          online={stats.onlineDevices}
          total={stats.totalDevices}
          isAdmin={isAdmin}
          poolData={poolData}
          onRefreshPool={isAdmin ? handleRefreshPool : undefined}
          refreshing={poolRefreshing}
        />

        {/* Bottom two-column */}
        <div className="flex flex-col xl:flex-row gap-4 items-start pb-6">
          <div className="flex-1 min-w-0 grid gap-4">
            {isAdmin && <DashboardServiceCard />}
            {isAdmin && (
              <Card className="border-[var(--color-line)]">
                <CardHeader className="p-4 pb-2">
                  <div className="flex items-center gap-1.5 mb-1">
                    <span className="w-1 h-3 rounded-full bg-[var(--color-primary)] opacity-60" />
                    <SectionLabel>CONFIGURATION</SectionLabel>
                  </div>
                  <h3 className="text-[15px] font-semibold text-[var(--color-text)]">{t('config_management')}</h3>
                </CardHeader>
                <CardContent className="p-4 pt-0 grid gap-3">
                  <ConfigActionRow icon={Wand2}    title={t('config_wizard')}       desc={t('from_wizard_desc')}   highlighted onClick={() => router.navigate({ to: '/admin/config-wizard' })} />
                  <ConfigActionRow icon={Download} title={t('export_config_title')} desc={t('export_config_desc')}             onClick={handleExport} />
                  <ConfigActionRow icon={Upload}   title={t('import_config_title')} desc={t('import_config_desc')}             onClick={() => fileInputRef.current?.click()} />
                  <input ref={fileInputRef} type="file" accept=".yaml,.yml,.json" className="hidden" onChange={handleImport} disabled={importing} />
                </CardContent>
              </Card>
            )}
          </div>

          <DashboardRightCol
            programStartedAt={programStartedAt}
            poolData={poolData}
            onNavigate={(route) => router.navigate({ to: route as never })}
          />
        </div>
      </div>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div className="grid gap-5 p-6">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-36 rounded-xl" />)}</div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">{Array.from({ length: 2 }).map((_, i) => <Skeleton key={i} className="h-52 rounded-xl" />)}</div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/dashboard')({
  component: () => (
    <Suspense fallback={<DashboardSkeleton />}>
      <DashboardInner />
    </Suspense>
  ),
})
