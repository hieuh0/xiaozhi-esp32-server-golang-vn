import { useState, useRef, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

interface Option { label: string; value: string }

interface ComboInputProps {
  value: string
  onChange: (v: string) => void
  options: Option[]
  placeholder?: string
  loading?: boolean
  disabled?: boolean
  className?: string
}

export function ComboInput({ value, onChange, options, placeholder, loading, disabled, className }: ComboInputProps) {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const filtered = options.filter(o =>
    !value || o.value.toLowerCase().includes(value.toLowerCase()) || o.label.toLowerCase().includes(value.toLowerCase())
  )

  useEffect(() => {
    const handleOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener('mousedown', handleOutside)
    return () => document.removeEventListener('mousedown', handleOutside)
  }, [open])

  return (
    <div ref={containerRef} className="relative">
      <Input
        value={value}
        onChange={e => { onChange(e.target.value); setOpen(true) }}
        onFocus={() => { if (filtered.length > 0) setOpen(true) }}
        placeholder={loading ? '' : placeholder}
        disabled={disabled || loading}
        className={className}
      />
      {loading && (
        <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground animate-pulse">
          loading...
        </span>
      )}
      {open && !loading && filtered.length > 0 && (
        <ul className="absolute z-50 top-full left-0 right-0 mt-1 max-h-48 overflow-y-auto rounded-md border bg-popover text-popover-foreground shadow-md py-1">
          {filtered.map(o => (
            <li
              key={o.value}
              className={cn(
                'px-3 py-1.5 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground',
                o.value === value && 'bg-accent text-accent-foreground'
              )}
              onMouseDown={e => { e.preventDefault(); onChange(o.value); setOpen(false) }}
            >
              {o.label && o.label !== o.value
                ? <><span className="font-medium">{o.value}</span><span className="text-muted-foreground text-xs ml-2">{o.label}</span></>
                : <span>{o.value}</span>}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
