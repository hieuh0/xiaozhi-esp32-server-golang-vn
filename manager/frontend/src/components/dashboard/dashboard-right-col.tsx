import { User, Monitor, Cpu, Settings } from 'lucide-react'
import { useLocale } from '@/hooks/use-locale'
import { useAuthStore } from '@/stores/auth'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { SectionLabel } from '@/components/ui/section-label'
import { PoolHealthBars } from '@/components/charts'
import type { PoolStatEntry } from '@/features/dashboard/types'
import { cn } from '@/lib/utils'

interface DashboardRightColProps {
  programStartedAt: string
  poolData: PoolStatEntry[]
  onNavigate: (route: string) => void
}

export function DashboardRightCol({ programStartedAt, poolData, onNavigate }: DashboardRightColProps) {
  const { t } = useLocale()
  const { user, isAdmin } = useAuthStore()

  const adminQuickActions = [
    { icon: User,     label: t('user_management'), desc: t('view_account_desc'), route: '/admin/users' },
    { icon: Settings, label: t('llm_config'),      desc: t('llm_config_desc'),  route: '/admin/llm-config' },
    { icon: Cpu,      label: t('vad_config'),      desc: t('vad_config_desc'),  route: '/admin/vad-config' },
  ]

  return (
    <div className="w-full xl:w-[360px] xl:shrink-0 grid gap-3 min-w-0">
      {/* System Info */}
      <Card>
        <CardHeader className="p-4 pb-2">
          <div className="flex items-center gap-1.5 mb-1">
            <span className="w-1 h-3 rounded-full bg-[var(--color-primary)] opacity-60" />
            <SectionLabel>SYSTEM</SectionLabel>
          </div>
          <h3 className="text-[15px] font-semibold text-[var(--color-text)]">{t('system_info')}</h3>
        </CardHeader>
        <CardContent className="px-4 pb-4 pt-0">
          <dl className="divide-y divide-[var(--color-line)]">
            {[
              { label: t('system_version'),     value: 'v1.0.0' },
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
              <dd>
                <Badge className={cn('border text-xs', isAdmin ? 'status-primary' : 'status-muted')}>
                  {isAdmin ? t('admin') : t('normal_user')}
                </Badge>
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {/* Quick Actions */}
      <Card>
        <CardHeader className="p-4 pb-2">
          <div className="flex items-center gap-1.5 mb-1">
            <span className="w-1 h-3 rounded-full bg-[var(--color-primary)] opacity-60" />
            <SectionLabel>SHORTCUTS</SectionLabel>
          </div>
          <h3 className="text-[15px] font-semibold text-[var(--color-text)]">{t('quick_actions')}</h3>
        </CardHeader>
        <CardContent className="px-4 pb-4 pt-0 flex flex-col gap-2">
          {isAdmin ? adminQuickActions.map((qa) => (
            <button key={qa.route} type="button" onClick={() => onNavigate(qa.route)}
              className="flex items-center gap-3 w-full px-3 py-2.5 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] shadow-[var(--shadow-card)] cursor-pointer hover:-translate-y-px hover:shadow-[var(--shadow-card-hover)] active:scale-[0.98] transition-all duration-150 text-left">
              <span className="w-9 h-9 rounded-xl flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0 flex-shrink-0">
                <qa.icon className="w-4 h-4" />
              </span>
              <span className="flex flex-col min-w-0 flex-1">
                <span className="block text-sm font-semibold text-[var(--color-text)] leading-snug">{qa.label}</span>
                <span className="block text-xs text-[var(--color-text-secondary)] truncate leading-snug">{qa.desc}</span>
              </span>
            </button>
          )) : (
            <>
              <button type="button" onClick={() => onNavigate('/agents')}
                className="flex items-center gap-3 w-full px-3 py-2.5 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] shadow-[var(--shadow-card)] cursor-pointer hover:-translate-y-px hover:shadow-[var(--shadow-card-hover)] active:scale-[0.98] transition-all duration-150 text-left">
                <span className="w-9 h-9 rounded-xl flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0 flex-shrink-0">
                  <Monitor className="w-4 h-4" />
                </span>
                <span className="flex flex-col min-w-0 flex-1">
                  <span className="block text-sm font-semibold text-[var(--color-text)] leading-snug">{t('agent_management')}</span>
                  <span className="block text-xs text-[var(--color-text-secondary)] leading-snug">{t('agent_mgmt_desc')}</span>
                </span>
              </button>
              <p className="text-sm text-[var(--color-text-secondary)]">{t('normal_user_quick_hint')}</p>
            </>
          )}
        </CardContent>
      </Card>

      {/* Pool Health — admin + data present */}
      {isAdmin && poolData.length > 0 && <PoolHealthBars data={poolData} />}
    </div>
  )
}
