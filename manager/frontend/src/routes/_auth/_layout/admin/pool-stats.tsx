import { Suspense, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useSuspenseQuery, useQueryClient } from '@tanstack/react-query'
import { RefreshCw, Database, Layers, Clock, CheckCircle2 } from 'lucide-react'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { dashboardApi } from '@/features/dashboard/api/dashboard-api'
import { PoolResourceChart } from '@/components/charts'
import type { PoolStatEntry, PoolStatsData, PoolSummary } from '@/features/dashboard/types'
import { useLocale } from '@/hooks/use-locale'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { DataTable } from '@/components/ui/data-table'
import { PageHeader } from '@/components/ui/page-header'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

function formatTime(ts: string | null): string {
  if (!ts) return '—'
  return new Date(ts).toLocaleString()
}

function PoolSummaryCards({ summary }: { summary: PoolSummary }) {
  const { t } = useLocale()
  const cards = [
    {
      icon: Database,
      iconClass: 'kpi-icon-primary',
      value: String(summary.total_records || 0),
      label: t('total_records'),
      large: true,
    },
    {
      icon: Layers,
      iconClass: 'kpi-icon-muted',
      value: t('latest_only'),
      label: t('storage_mode'),
      large: true,
    },
    {
      icon: Clock,
      iconClass: 'kpi-icon-warning',
      value: formatTime(summary.oldest_timestamp),
      label: t('earliest_time'),
      large: false,
    },
    {
      icon: CheckCircle2,
      iconClass: 'kpi-icon-success',
      value: formatTime(summary.newest_timestamp),
      label: t('latest_time'),
      large: false,
    },
  ]

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((item) => (
        <div key={item.label} className="p-4 rounded-xl bg-[var(--color-surface-1)] border border-[var(--color-line)] shadow-[var(--shadow-card)] flex flex-col gap-3">
          <div className={cn('w-9 h-9 rounded-lg flex items-center justify-center shrink-0', item.iconClass)}>
            <item.icon className="w-4 h-4" />
          </div>
          <div>
            <div className={cn(
              'font-numeric leading-tight',
              item.large
                ? 'text-2xl font-bold text-[var(--color-text)]'
                : 'text-sm font-semibold text-[var(--color-text)] break-all'
            )}>
              {item.value}
            </div>
            <div className="text-xs text-[var(--color-text-secondary)] mt-0.5">{item.label}</div>
          </div>
        </div>
      ))}
    </div>
  )
}

function PoolStatsContent() {
  const { t } = useLocale()
  const queryClient = useQueryClient()
  const [refreshing, setRefreshing] = useState(false)

  const { data: stats } = useSuspenseQuery<PoolStatsData | null>({
    queryKey: ['pool-stats'],
    queryFn: dashboardApi.getPoolStats,
    staleTime: 25_000,
    refetchInterval: 30_000,
  })

  const { data: summary } = useSuspenseQuery<PoolSummary>({
    queryKey: ['pool-summary'],
    queryFn: dashboardApi.getPoolSummary,
    staleTime: 25_000,
  })

  const tableData: PoolStatEntry[] = stats?.stats
    ? Object.entries(stats.stats)
        .filter(([, v]) => v && typeof v === 'object')
        .map(([poolKey, s]) => ({
          poolKey,
          total: s.total_resources || 0,
          available: s.available_resources || 0,
          inUse: s.in_use_resources || 0,
          maxSize: s.max_size || 0,
          minSize: s.min_size || 0,
          maxIdle: s.max_idle || 0,
          isClosed: s.is_closed || false,
        }))
    : []

  const columns: ColumnDef<PoolStatEntry>[] = [
    { accessorKey: 'poolKey', header: t('pool_key_col') },
    { accessorKey: 'total', header: t('total_resources') },
    { accessorKey: 'available', header: t('available_resources') },
    { accessorKey: 'inUse', header: t('in_use') },
    { accessorKey: 'maxSize', header: t('max_capacity') },
    { accessorKey: 'minSize', header: t('min_capacity') },
    { accessorKey: 'maxIdle', header: t('max_idle') },
    {
      accessorKey: 'isClosed',
      header: t('status'),
      cell: ({ row }) => (
        <Badge className={cn('border text-xs', row.original.isClosed ? 'status-danger' : 'status-success')}>
          {row.original.isClosed ? t('closed') : t('running')}
        </Badge>
      ),
    },
  ]

  const handleRefresh = async () => {
    setRefreshing(true)
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['pool-stats'] }),
      queryClient.invalidateQueries({ queryKey: ['pool-summary'] }),
    ])
    setRefreshing(false)
    toast.success(t('refresh_success'))
  }

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between pb-4">
        <h3 className="text-lg font-semibold text-[var(--color-text)]">{t('resource_pool_stats')}</h3>
        <Button variant="outline" size="sm" disabled={refreshing} onClick={handleRefresh}>
          <RefreshCw className={`w-4 h-4 mr-1.5 ${refreshing ? 'animate-spin' : ''}`} />
          {t('refresh')}
        </Button>
      </CardHeader>
      <CardContent className="space-y-6">
        <PoolSummaryCards summary={summary} />
        {tableData.length > 0 && (
          <div className="border-t border-[var(--color-line)] pt-4">
            <p className="text-xs font-semibold uppercase tracking-widest text-[var(--color-text-tertiary)] mb-3">{t('pool_utilization')}</p>
            <PoolResourceChart data={tableData} />
          </div>
        )}
        {stats ? (
          <div className="border-t border-[var(--color-line)] pt-4">
            <p className="text-xs text-[var(--color-text-secondary)] mb-4">
              {t('latest_stats_title', { time: formatTime(stats.timestamp) })}
            </p>
            <DataTable data={tableData} columns={columns} emptyMessage={t('no_statistics')} />
          </div>
        ) : (
          <div className="py-12 text-center text-[var(--color-text-secondary)]">{t('no_statistics')}</div>
        )}
      </CardContent>
    </Card>
  )
}

function PoolStatsSkeleton() {
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between pb-4">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-8 w-24" />
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-20 rounded-xl" />)}
        </div>
        <Skeleton className="h-48 w-full" />
      </CardContent>
    </Card>
  )
}

function PoolStatsPage() {
  const { t } = useLocale()
  return (
    <div className="p-6 grid gap-5">
      <PageHeader eyebrow="ADMIN" title={t('resource_pool_stats')} />
      <Suspense fallback={<PoolStatsSkeleton />}>
        <PoolStatsContent />
      </Suspense>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/pool-stats')({
  component: PoolStatsPage,
})
