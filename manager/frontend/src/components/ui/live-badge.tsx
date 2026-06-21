export function LiveBadge() {
  return (
    <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full border text-[11px] font-semibold status-success shrink-0">
      <span className="relative flex w-2 h-2">
        <span
          className="animate-pulse-ring absolute inline-flex h-full w-full rounded-full"
          style={{ background: 'var(--color-chart-green)' }}
        />
        <span
          className="relative inline-flex w-2 h-2 rounded-full"
          style={{ background: 'var(--color-chart-green)' }}
        />
      </span>
      Live
    </span>
  )
}
