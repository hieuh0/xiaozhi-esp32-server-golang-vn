import { useEffect } from 'react'
import { useLocale } from '@/hooks/use-locale'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { DeviceFormData } from '@/features/devices/types'
import type { Agent } from '@/features/agents/types'

interface DeviceFormProps {
  value: DeviceFormData
  onChange: (v: DeviceFormData) => void
  agents: Agent[]
  fixedAgentId?: number | null
}

export function DeviceForm({ value, onChange, agents, fixedAgentId }: DeviceFormProps) {
  const { t } = useLocale()
  const hasFixedAgent = fixedAgentId !== null && fixedAgentId !== undefined

  const set = (patch: Partial<DeviceFormData>) => onChange({ ...value, ...patch })

  useEffect(() => {
    if (hasFixedAgent) set({ agent_id: Number(fixedAgentId) })
  }, [fixedAgentId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="grid gap-4">
      {!hasFixedAgent && (
        <div className="grid gap-1.5">
          <label className="text-sm font-medium text-[var(--color-text)]">{t('target_agent')}</label>
          <Select value={value.agent_id ? String(value.agent_id) : ''} onValueChange={(v) => set({ agent_id: Number(v) })}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder={t('select_agent_to_bind')} />
            </SelectTrigger>
            <SelectContent>
              {agents.map((a) => (
                <SelectItem key={a.id} value={String(a.id)}>{a.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      <div className="grid gap-1.5">
        <label className="text-sm font-medium text-[var(--color-text)]">{t('device_verify_code_mac')}</label>
        <Input
          value={value.identifier}
          onChange={(e) => set({ identifier: e.target.value })}
          placeholder={t('enter_6digit_or_mac')}
          autoComplete="off"
        />
        <div className="flex items-center gap-2 flex-wrap text-xs text-[var(--color-text-tertiary)]">
          <span>{t('example')}</span>
          <code className="px-1.5 py-0.5 rounded bg-[var(--color-surface-2)] font-mono">123456</code>
          <code className="px-1.5 py-0.5 rounded bg-[var(--color-surface-2)] font-mono">28:0A:C6:1D:3B:E8</code>
        </div>
      </div>

      <div className="grid gap-1.5">
        <label className="text-sm font-medium text-[var(--color-text)]">{t('device_nickname')}</label>
        <Input
          value={value.nick_name}
          onChange={(e) => set({ nick_name: e.target.value })}
          placeholder={t('device_name_example')}
          maxLength={50}
        />
      </div>
    </div>
  )
}
