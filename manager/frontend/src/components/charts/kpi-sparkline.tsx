import { AreaChart, Area, ResponsiveContainer } from 'recharts'

const PATTERN = [0.5, 0.58, 0.52, 0.68, 0.63, 0.78, 0.88, 1.0]

interface KpiSparklineProps {
  value: number
  color: string
}

export function KpiSparkline({ value, color }: KpiSparklineProps) {
  const data = PATTERN.map(f => ({ v: Math.max(1, Math.round(value * f)) }))
  return (
    <div className="h-9 mt-3 opacity-70">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data}>
          <defs>
            <linearGradient id={`spark-${color.replace(/[^a-z0-9]/gi, '')}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity={0.2} />
              <stop offset="100%" stopColor={color} stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="v"
            stroke={color}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
            fill={`url(#spark-${color.replace(/[^a-z0-9]/gi, '')})`}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
