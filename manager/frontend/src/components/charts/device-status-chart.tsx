import { PieChart, Pie, Cell, ResponsiveContainer } from 'recharts'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { SectionLabel } from '@/components/ui/section-label'
import { LiveBadge } from '@/components/ui/live-badge'
import { useLocale } from '@/hooks/use-locale'

interface DeviceStatusChartProps {
  online: number
  total: number
}

export function DeviceStatusChart({ online, total }: DeviceStatusChartProps) {
  const { t } = useLocale()
  const offline = Math.max(0, total - online)
  const pct = total > 0 ? Math.round((online / total) * 100) : 0
  const offlinePct = 100 - pct

  const data = [
    { name: t('online'),  value: online > 0 ? online : 0.001, fill: 'var(--color-chart-green)' },
    { name: t('offline'), value: total === 0 ? 1 : offline > 0 ? offline : 0.001, fill: 'var(--color-line)' },
  ]

  return (
    <Card className="border-[var(--color-line)]">
      <CardHeader className="flex-row items-start justify-between gap-3 p-4 pb-3 border-b border-[var(--color-line)]">
        <div>
          <div className="flex items-center gap-1.5 mb-0.5">
            <div className="w-1 h-3 rounded-full bg-[var(--color-primary)] opacity-60" />
            <SectionLabel>REALTIME</SectionLabel>
          </div>
          <h3 className="text-[15px] font-semibold text-[var(--color-text)]">{t('device_status')}</h3>
          <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{t('online')} / {t('offline')} theo thời gian thực</p>
        </div>
        <LiveBadge />
      </CardHeader>
      <CardContent className="p-5">
        <div className="flex items-center gap-6">
          {/* Donut with center % */}
          <div className="relative shrink-0" style={{ width: 140, height: 140 }}>
            <ResponsiveContainer width={140} height={140}>
              <PieChart>
                <Pie
                  data={data}
                  cx="50%"
                  cy="50%"
                  innerRadius={42}
                  outerRadius={64}
                  dataKey="value"
                  strokeWidth={0}
                  startAngle={90}
                  endAngle={-270}
                >
                  {data.map((entry, i) => <Cell key={i} fill={entry.fill} />)}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
            <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
              <strong className="font-numeric text-[26px] font-bold text-[var(--color-text)] leading-none">{pct}%</strong>
              <span className="text-[10px] text-[var(--color-text-secondary)] mt-0.5">{t('online')}</span>
            </div>
          </div>

          {/* Legend */}
          <div className="flex flex-col gap-4 min-w-0">
            <div className="flex items-start gap-3">
              <span className="mt-1.5 w-2.5 h-2.5 rounded-full shrink-0" style={{ background: 'var(--color-chart-green)' }} />
              <div>
                <p className="text-xs text-[var(--color-text-secondary)]">{t('online')}</p>
                <p className="font-numeric text-[20px] font-bold text-[var(--color-text)] leading-tight">{online}</p>
                <p className="text-[11px] text-[var(--color-text-tertiary)]">{pct}%</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <span className="mt-1.5 w-2.5 h-2.5 rounded-full shrink-0" style={{ background: 'var(--color-line)' }} />
              <div>
                <p className="text-xs text-[var(--color-text-secondary)]">{t('offline')}</p>
                <p className="font-numeric text-[20px] font-bold text-[var(--color-text)] leading-tight">{offline}</p>
                <p className="text-[11px] text-[var(--color-text-tertiary)]">{offlinePct}%</p>
              </div>
            </div>
            <div className="border-t border-[var(--color-line)] pt-2.5">
              <p className="text-[11px] text-[var(--color-text-tertiary)]">
                {t('total')}: <strong className="text-[var(--color-text)] font-semibold">{total}</strong>
              </p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
