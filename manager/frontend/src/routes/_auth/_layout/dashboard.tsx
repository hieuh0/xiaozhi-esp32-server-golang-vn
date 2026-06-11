import { Suspense, useRef, useState } from 'react'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { useSuspenseQuery } from '@tanstack/react-query'
import { User, Monitor, Cpu, Wifi, Wand2, Download, Upload, Settings } from 'lucide-react'
import { SectionLabel } from '@/components/ui/section-label'
import { toast } from 'sonner'
import { dashboardApi } from '@/features/dashboard/api/dashboard-api'
import type { DashboardStats } from '@/features/dashboard/types'
import { DashboardServiceCard } from '@/components/dashboard/dashboard-service-card'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

// ---- sub-components ----

function StatCard({ icon: Icon, iconClass, trend, value, label }: {
  icon: React.ElementType; iconClass: string; trend: string; value: number; label: string
}) {
  return (
    <div
      className="relative p-5 rounded-2xl border border-[var(--color-line)] overflow-hidden group transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[var(--shadow-card-hover)]"
      style={{
        background: 'linear-gradient(145deg, var(--color-surface-1) 0%, var(--color-bg) 100%)',
        boxShadow: 'var(--shadow-card)',
      }}
    >
      {/* Ambient glow blob */}
      <div
        className="absolute -right-4 -top-4 w-24 h-24 rounded-full blur-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"
        style={{ background: 'var(--color-primary-soft)' }}
      />
      <div className="relative">
        <div className="flex items-center justify-between mb-4">
          <span className={cn('w-10 h-10 rounded-lg inline-flex items-center justify-center', iconClass)}>
            <Icon className="w-5 h-5" />
          </span>
          <span className="text-[10px] font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] font-mono">{trend}</span>
        </div>
        <strong className="block text-4xl font-bold font-display tracking-tight leading-none text-[var(--color-text)]">{value}</strong>
        <p className="mt-2.5 text-sm text-[var(--color-text-secondary)]">{label}</p>
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
        highlighted
          ? 'border-[var(--color-primary)]/20 bg-[var(--color-primary-soft)]'
          : 'border-[var(--color-line)] bg-[var(--color-surface-1)] shadow-[var(--shadow-card)]')}>
      <span className="w-10 h-10 rounded-xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0">
        <Icon className="w-5 h-5" />
      </span>
      <span className="flex flex-col gap-0.5">
        <strong className="text-[15px] text-[var(--color-text)]">{title}</strong>
        <small className="text-sm text-[var(--color-text-secondary)]">{desc}</small>
      </span>
    </button>
  )
}

// ---- data section (Suspense boundary) ----

function DashboardInner() {
  const { t } = useLocale()
  const router = useRouter()
  const { user, isAdmin, token } = useAuthStore()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [importing, setImporting] = useState(false)

  const { data: stats } = useSuspenseQuery<DashboardStats>({
    queryKey: ['dashboard-stats'],
    queryFn: dashboardApi.getStats,
    staleTime: 30_000,
  })

  const programStartedAt = stats.programStartedAt
    ? new Date(stats.programStartedAt).toLocaleString()
    : '—'

  const metricCards = [
    { icon: User, iconClass: 'text-[var(--color-primary)] bg-[var(--color-primary-soft)]', trend: isAdmin ? t('global_user') : t('linked_account'), value: isAdmin ? stats.totalUsers : 1, label: isAdmin ? t('total_users') : t('current_logged_account') },
    { icon: Monitor, iconClass: 'text-[var(--color-success)] bg-[color-mix(in_srgb,var(--color-success)_12%,transparent)]', trend: t('online_devices'), value: stats.totalDevices, label: isAdmin ? t('total_devices') : t('my_devices') },
    { icon: Cpu, iconClass: 'text-[var(--color-warning)] bg-[color-mix(in_srgb,var(--color-warning)_12%,transparent)]', trend: t('active'), value: stats.totalAgents, label: isAdmin ? t('agent_count') : t('my_agents') },
    { icon: Wifi, iconClass: 'text-[var(--color-danger)] bg-[color-mix(in_srgb,var(--color-danger)_12%,transparent)]', trend: t('realtime_monitoring'), value: stats.onlineDevices, label: t('online_devices') },
  ]

  const adminQuickActions = [
    { icon: User, label: t('user_management'), desc: t('view_account_desc'), route: '/admin/users' },
    { icon: Settings, label: t('llm_config'), desc: t('llm_config_desc'), route: '/admin/llm-config' },
    { icon: Cpu, label: t('vad_config'), desc: t('vad_config_desc'), route: '/admin/vad-config' },
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

  return (
    <div className="grid gap-4 px-6 py-4">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {metricCards.map((m) => <StatCard key={m.label} {...m} />)}
      </div>

      <div className="flex flex-col xl:flex-row gap-4 items-start">
        <div className="flex-1 min-w-0 grid gap-4">
          {isAdmin && <DashboardServiceCard />}
          {isAdmin && (
            <Card className="border-[var(--color-line)]">
              <CardHeader className="pb-3">
                <SectionLabel className="mb-1">CONFIGURATION</SectionLabel>
                <h3 className="text-lg font-semibold text-[var(--color-text)]">{t('config_management')}</h3>
              </CardHeader>
              <CardContent className="grid gap-3">
                <ConfigActionRow icon={Wand2} title={t('config_wizard')} desc={t('from_wizard_desc')} highlighted onClick={() => router.navigate({ to: '/admin/config-wizard' })} />
                <ConfigActionRow icon={Download} title={t('export_config_title')} desc={t('export_config_desc')} onClick={handleExport} />
                <ConfigActionRow icon={Upload} title={t('import_config_title')} desc={t('import_config_desc')} onClick={() => fileInputRef.current?.click()} />
                <input ref={fileInputRef} type="file" accept=".yaml,.yml,.json" className="hidden" onChange={handleImport} disabled={importing} />
              </CardContent>
            </Card>
          )}
        </div>

        <div className="w-full xl:w-[360px] xl:shrink-0 grid gap-3 min-w-0">
          <Card>
            <CardHeader className="p-4 pb-2">
              <SectionLabel>SYSTEM</SectionLabel>
              <h3 className="text-base font-semibold text-[var(--color-text)]">{t('system_info')}</h3>
            </CardHeader>
            <CardContent className="px-4 pb-4 pt-0">
              <dl className="divide-y divide-[var(--color-line)]">
                {[
                  { label: t('system_version'), value: 'v1.0.0' },
                  { label: t('program_start_time'), value: programStartedAt },
                  { label: t('current_user_label'), value: user?.username || '—' },
                ].map((row) => (
                  <div key={row.label} className="flex items-center justify-between gap-3 py-2">
                    <dt className="text-sm text-[var(--color-text-secondary)]">{row.label}</dt>
                    <dd className="text-sm font-semibold text-[var(--color-text)]">{row.value}</dd>
                  </div>
                ))}
                <div className="flex items-center justify-between gap-3 py-2">
                  <dt className="text-sm text-[var(--color-text-secondary)]">{t('user_role_label')}</dt>
                  <dd><Badge variant={isAdmin ? 'destructive' : 'secondary'}>{isAdmin ? t('admin') : t('normal_user')}</Badge></dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <SectionLabel>SHORTCUTS</SectionLabel>
              <h3 className="text-base font-semibold text-[var(--color-text)]">{t('quick_actions')}</h3>
            </CardHeader>
            <CardContent className="px-4 pb-4 pt-0 grid gap-2">
              {isAdmin ? adminQuickActions.map((qa) => (
                <button key={qa.route} type="button" onClick={() => router.navigate({ to: qa.route as never })}
                  className="flex items-center gap-3 w-full p-2.5 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] shadow-[var(--shadow-card)] cursor-pointer hover:-translate-y-px hover:shadow-[var(--shadow-card-hover)] active:scale-[0.98] transition-all duration-150 text-left">
                  <span className="w-9 h-9 rounded-xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><qa.icon className="w-4 h-4" /></span>
                  <span className="flex flex-col gap-0.5 min-w-0">
                    <strong className="text-sm text-[var(--color-text)]">{qa.label}</strong>
                    <small className="text-xs text-[var(--color-text-secondary)] truncate">{qa.desc}</small>
                  </span>
                </button>
              )) : (
                <>
                  <button type="button" onClick={() => router.navigate({ to: '/agents' })}
                    className="flex items-center gap-3 w-full p-2.5 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] shadow-[var(--shadow-card)] cursor-pointer hover:-translate-y-px hover:shadow-[var(--shadow-card-hover)] active:scale-[0.98] transition-all duration-150 text-left">
                    <span className="w-9 h-9 rounded-xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><Monitor className="w-4 h-4" /></span>
                    <span className="flex flex-col gap-0.5"><strong className="text-sm text-[var(--color-text)]">{t('agent_management')}</strong><small className="text-xs text-[var(--color-text-secondary)]">{t('agent_mgmt_desc')}</small></span>
                  </button>
                  <p className="text-sm text-[var(--color-text-secondary)]">{t('normal_user_quick_hint')}</p>
                </>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div className="grid gap-5 p-6">
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-32 rounded-xl" />)}</div>
    </div>
  )
}

function DashboardPage() {
  return (
    <Suspense fallback={<DashboardSkeleton />}>
      <DashboardInner />
    </Suspense>
  )
}

export const Route = createFileRoute('/_auth/_layout/dashboard')({
  component: DashboardPage,
})
