import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import type { PoolStatEntry } from '@/features/dashboard/types'
import { useLocale } from '@/hooks/use-locale'

interface PoolResourceChartProps {
  data: PoolStatEntry[]
}

interface TooltipEntry {
  name: string
  value: number
  color?: string
  fill?: string
}

function ChartTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipEntry[]; label?: string }) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-1)] px-3 py-2 shadow-[var(--shadow-float)] text-sm space-y-1">
      <p className="font-semibold text-[var(--color-text)] mb-1">{label}</p>
      {payload.map((p) => (
        <div key={p.name} className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full shrink-0" style={{ background: p.color ?? p.fill }} />
          <span className="text-[var(--color-text-secondary)]">{p.name}:</span>
          <span className="font-numeric font-semibold text-[var(--color-text)]">{p.value}</span>
        </div>
      ))}
    </div>
  )
}

function shortenKey(key: string): string {
  return key.length > 14 ? key.slice(0, 13) + '…' : key
}

export function PoolResourceChart({ data }: PoolResourceChartProps) {
  const { t } = useLocale()

  if (!data.length) return null

  const chartData = data.map((p) => ({
    name: shortenKey(p.poolKey),
    [t('total_resources')]:     p.total,
    [t('available_resources')]: p.available,
    [t('in_use')]:              p.inUse,
  }))

  return (
    <ResponsiveContainer width="100%" height={220}>
      <BarChart data={chartData} barSize={8} barCategoryGap="35%">
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-line)" vertical={false} />
        <XAxis
          dataKey="name"
          tick={{ fontSize: 11, fill: 'var(--color-text-secondary)' }}
          axisLine={{ stroke: 'var(--color-line)' }}
          tickLine={false}
        />
        <YAxis
          tick={{ fontSize: 11, fill: 'var(--color-text-secondary)' }}
          axisLine={false}
          tickLine={false}
          width={28}
        />
        <Tooltip content={<ChartTooltip />} cursor={{ fill: 'var(--color-surface-2)' }} />
        <Legend
          wrapperStyle={{ fontSize: 12, color: 'var(--color-text-secondary)', paddingTop: 8 }}
          iconType="circle"
          iconSize={8}
        />
        <Bar dataKey={t('total_resources')}     fill="var(--color-chart-blue)"  radius={[3, 3, 0, 0]} />
        <Bar dataKey={t('available_resources')} fill="var(--color-chart-green)" radius={[3, 3, 0, 0]} />
        <Bar dataKey={t('in_use')}              fill="var(--color-chart-amber)" radius={[3, 3, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
}
