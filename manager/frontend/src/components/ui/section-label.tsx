import { cn } from '@/lib/utils'

interface SectionLabelProps {
  children: React.ReactNode
  className?: string
}

export function SectionLabel({ children, className }: SectionLabelProps) {
  return (
    <p className={cn('text-[11px] font-bold uppercase tracking-widest text-[var(--color-text-tertiary)]', className)}>
      {children}
    </p>
  )
}
