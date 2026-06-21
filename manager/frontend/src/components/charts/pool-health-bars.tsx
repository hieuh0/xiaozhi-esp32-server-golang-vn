import type { PoolStatEntry } from '@/features/dashboard/types'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { SectionLabel } from '@/components/ui/section-label'
import { useLocale } from '@/hooks/use-locale'

function barColor(pct: number): string {
  if (pct >= 80) return 'var(--color-chart-red)'
  if (pct >= 50) return 'var(--color-chart-amber)'
  return 'var(--color-chart-green)'
}

interface PoolHealthBarsProps {
  data: PoolStatEntry[]
}

export function PoolHealthBars({ data }: PoolHealthBarsProps) {
  const { t } = useLocale()
  if (!data.length) return null

  return (
    <Card className="border-[var(--color-line)]">
      <CardHeader className="p-4 pb-2">
        <div className="flex items-center gap-1.5 mb-1">
          <div className="w-1 h-3 rounded-full bg-[var(--color-primary)] opacity-60" />
          <SectionLabel>POOL HEALTH</SectionLabel>
        </div>
        <h3 className="text-[15px] font-semibold text-[var(--color-text)]">{t('resource_pool_stats')}</h3>
      </CardHeader>
      <CardContent className="px-4 pb-4 pt-2 flex flex-col gap-3">
        {data.slice(0, 5).map((pool) => {
          const pct = pool.maxSize > 0 ? Math.round((pool.inUse / pool.maxSize) * 100) : 0
          return (
            <div key={pool.poolKey}>
              <div className="flex items-center justify-between text-xs mb-1.5">
                <span className="font-mono text-[var(--color-text-secondary)] truncate max-w-[55%]">{pool.poolKey}</span>
                <span className="font-semibold text-[var(--color-text)]">{pool.inUse}/{pool.maxSize} {t('in_use').toLowerCase()}</span>
              </div>
              <div className="h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--color-surface-2)' }}>
                <div
                  className="h-full rounded-full transition-[width] duration-300"
                  style={{ width: `${Math.min(100, pct)}%`, background: barColor(pct) }}
                />
              </div>
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}
