import { Download } from 'lucide-react'
import { useLocale } from '@/hooks/use-locale'
import { cn } from '@/lib/utils'

export type DashboardPeriod = 'today' | '7d' | '30d'

interface DashboardPageHeaderProps {
  period: DashboardPeriod
  onPeriodChange: (p: DashboardPeriod) => void
  onExport: () => void
  isAdmin: boolean
}

export function DashboardPageHeader({ period, onPeriodChange, onExport, isAdmin }: DashboardPageHeaderProps) {
  const { t } = useLocale()

  const periods: { key: DashboardPeriod; label: string }[] = [
    { key: 'today', label: t('today') },
    { key: '7d',    label: t('last_7_days') },
    { key: '30d',   label: t('last_30_days') },
  ]

  return (
    <div className="flex items-start justify-between gap-4 px-6 pt-5 pb-4">
      <div className="min-w-0">
        <div className="flex items-center gap-1.5 mb-0.5">
          <span className="inline-block w-1 h-3 rounded-full bg-[var(--color-primary)] opacity-60" />
          <span className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-primary)] font-mono">OVERVIEW</span>
        </div>
        <h2 className="text-xl font-semibold font-display text-[var(--color-text)] leading-tight">
          {t('dashboard_overview_title')}
        </h2>
        <p className="mt-0.5 text-sm text-[var(--color-text-secondary)]">
          {t('dashboard_overview_desc')}
        </p>
      </div>

      <div className="flex items-center gap-2 shrink-0 mt-1">
        {/* Period tabs */}
        <div className="flex gap-0.5 p-0.5 rounded-lg bg-[var(--color-surface-2)] border border-[var(--color-line)]">
          {periods.map((p) => (
            <button
              key={p.key}
              type="button"
              onClick={() => onPeriodChange(p.key)}
              className={cn(
                'px-3 py-1 rounded-md text-xs font-medium transition-all duration-150 cursor-pointer',
                period === p.key
                  ? 'bg-[var(--color-surface-1)] text-[var(--color-primary)] font-semibold shadow-[var(--shadow-card)]'
                  : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
              )}
            >
              {p.label}
            </button>
          ))}
        </div>

        {/* Export — admin only */}
        {isAdmin && (
          <button
            type="button"
            onClick={onExport}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-1)] text-sm font-medium text-[var(--color-text)] hover:bg-[var(--color-surface-2)] shadow-[var(--shadow-card)] hover:shadow-[var(--shadow-card-hover)] transition-all duration-150 cursor-pointer"
          >
            <Download className="w-3.5 h-3.5" />
            Export
          </button>
        )}
      </div>
    </div>
  )
}
