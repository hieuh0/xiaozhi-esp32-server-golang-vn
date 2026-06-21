import { RefreshCw } from 'lucide-react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { SectionLabel } from '@/components/ui/section-label'
import { Button } from '@/components/ui/button'
import { DeviceStatusChart } from '@/components/charts/device-status-chart'
import { PoolResourceChart } from '@/components/charts/pool-resource-chart'
import type { PoolStatEntry } from '@/features/dashboard/types'
import { useLocale } from '@/hooks/use-locale'

interface DashboardChartsRowProps {
  online: number
  total: number
  isAdmin: boolean
  poolData: PoolStatEntry[]
  onRefreshPool?: () => void
  refreshing?: boolean
}

export function DashboardChartsRow({ online, total, isAdmin, poolData, onRefreshPool, refreshing }: DashboardChartsRowProps) {
  const { t } = useLocale()

  return (
    <div className={`grid gap-4 ${isAdmin ? 'grid-cols-1 lg:grid-cols-2' : ''}`}>
      <DeviceStatusChart online={online} total={total} />

      {isAdmin && (
        <Card className="border-[var(--color-line)]">
          <CardHeader className="flex-row items-start justify-between gap-3 p-4 pb-3 border-b border-[var(--color-line)]">
            <div>
              <SectionLabel className="mb-0.5">ADMIN</SectionLabel>
              <h3 className="text-[15px] font-semibold text-[var(--color-text)]">{t('pool_utilization')}</h3>
              <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">
                {t('total_resources')} / {t('available_resources')} / {t('in_use')}
              </p>
            </div>
            {onRefreshPool && (
              <Button variant="outline" size="sm" className="shrink-0 mt-0.5" disabled={refreshing} onClick={onRefreshPool}>
                <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
              </Button>
            )}
          </CardHeader>
          <CardContent className="p-4 pt-3">
            {poolData.length > 0 ? (
              <PoolResourceChart data={poolData} />
            ) : (
              <div className="h-[220px] flex items-center justify-center text-sm text-[var(--color-text-tertiary)]">
                {t('no_statistics')}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
