import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface PageHeaderProps {
  eyebrow?: string
  title: string
  description?: string
  children?: ReactNode
  className?: string
}

export function PageHeader({ eyebrow, title, description, children, className }: PageHeaderProps) {
  return (
    <div className={cn('flex items-start justify-between gap-4 px-6 pt-6 pb-4', className)}>
      <div className="min-w-0">
        {eyebrow && (
          <div className="flex items-center gap-1.5 mb-0.5">
            <span className="inline-block w-1 h-3 rounded-full bg-[var(--color-primary)] opacity-60" />
            <p className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-primary)]">
              {eyebrow}
            </p>
          </div>
        )}
        <h2 className="text-xl font-semibold font-display text-[var(--color-text)] leading-tight truncate">{title}</h2>
        {description && (
          <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{description}</p>
        )}
      </div>
      {children && <div className="flex items-center gap-2 shrink-0">{children}</div>}
    </div>
  )
}
