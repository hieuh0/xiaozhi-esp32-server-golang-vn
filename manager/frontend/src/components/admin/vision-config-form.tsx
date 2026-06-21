import { useEffect, useState } from 'react'
import { useLocale } from '@/hooks/use-locale'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { ConfigRow, ConfigForm } from './config-list-page'

interface VisionFields {
  provider: string; name: string; enabled: boolean; is_default: boolean
  type: string; model_name: string; api_key: string; base_url: string
  max_tokens: number; temperature: number; top_p: number; timeout: number
}

const D: VisionFields = {
  provider: 'aliyun_vision', name: '', enabled: true, is_default: false,
  type: 'openai', model_name: 'qwen-vl-max',
  api_key: '', base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  max_tokens: 1000, temperature: 0.1, top_p: 0.1, timeout: 30,
}

function parse(row: ConfigRow | null): VisionFields {
  if (!row) return { ...D }
  try {
    const d = JSON.parse(row.json_data || '{}')
    return {
      ...D,
      provider: row.provider || 'aliyun_vision',
      name: row.name,
      enabled: row.enabled !== false,
      is_default: !!row.is_default,
      type: d.type || 'openai',
      model_name: d.model_name || '',
      api_key: d.api_key || '',
      base_url: d.base_url || '',
      max_tokens: d.max_tokens || 1000,
      temperature: d.temperature !== undefined ? d.temperature : 0.1,
      top_p: d.top_p !== undefined ? d.top_p : 0.1,
      timeout: d.timeout || 30,
    }
  } catch { return { ...D } }
}

function serialize(f: VisionFields): string {
  return JSON.stringify({
    provider: f.provider, type: f.type, model_name: f.model_name,
    api_key: f.api_key, base_url: f.base_url,
    max_tokens: f.max_tokens, temperature: f.temperature, top_p: f.top_p, timeout: f.timeout,
  })
}

const F = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="grid gap-1.5">
    <label className="text-sm font-medium text-[var(--color-text)]">{label}</label>
    {children}
  </div>
)

const N = ({ v, min, max, step, onChange }: { v: number; min?: number; max?: number; step?: number; onChange: (n: number) => void }) => (
  <Input type="number" value={v} min={min} max={max} step={step} onChange={e => onChange(Number(e.target.value))} />
)

const VISION_PROVIDERS = [
  { label: 'aliyun_vision', value: 'aliyun_vision', labelKey: true },
  { label: 'doubao_vision', value: 'doubao_vision', labelKey: true },
]

export function VisionConfigForm({ form: _form, setForm, editing }: { form: ConfigForm; setForm: (p: Partial<ConfigForm>) => void; editing: ConfigRow | null }) {
  const { t } = useLocale()
  const [f, setF] = useState<VisionFields>(() => parse(editing))
  useEffect(() => { setF(parse(editing)) }, [editing])

  const upd = (patch: Partial<VisionFields>) => {
    const next = { ...f, ...patch }
    setF(next)
    setForm({ name: next.name, provider: next.provider, enabled: next.enabled, is_default: next.is_default, json_data: serialize(next) })
  }

  return (
    <div className="grid gap-3">
      <F label={t('provider')}>
        <Select value={f.provider} onValueChange={v => upd({ provider: v })}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{VISION_PROVIDERS.map(o => <SelectItem key={o.value} value={o.value}>{o.labelKey ? t(o.label) : o.label}</SelectItem>)}</SelectContent>
        </Select>
      </F>
      <F label={t('config_name')}>
        <Input value={f.name} onChange={e => upd({ name: e.target.value })} placeholder={t('enter_config_name')} />
      </F>
      <div className="grid grid-cols-2 gap-3">
        <F label={t('type')}>
          <Input value={f.type} onChange={e => upd({ type: e.target.value })} placeholder={t('enter_type')} />
        </F>
        <F label={t('model_name_label')}>
          <Input value={f.model_name} onChange={e => upd({ model_name: e.target.value })} placeholder={t('enter_model_name')} />
        </F>
      </div>
      <F label={t('api_key')}>
        <Input type="password" value={f.api_key} onChange={e => upd({ api_key: e.target.value })} placeholder={t('enter_api_password')} />
      </F>
      <F label={t('base_url')}>
        <Input value={f.base_url} onChange={e => upd({ base_url: e.target.value })} placeholder={t('enter_base_url')} />
      </F>
      <div className="grid grid-cols-3 gap-3">
        <F label={t('max_tokens_label')}><N v={f.max_tokens} min={1} max={100000} onChange={v => upd({ max_tokens: v })} /></F>
        <F label={t('temperature')}><N v={f.temperature} min={0} max={2} step={0.1} onChange={v => upd({ temperature: v })} /></F>
        <F label="Top P"><N v={f.top_p} min={0} max={1} step={0.1} onChange={v => upd({ top_p: v })} /></F>
      </div>
      <F label={t('timeout_seconds')}>
        <N v={f.timeout} min={1} max={300} onChange={v => upd({ timeout: v })} />
      </F>
      <div className="flex items-center gap-6 pt-1">
        <label className="flex items-center gap-2 cursor-pointer">
          <Switch checked={f.enabled} onCheckedChange={v => upd({ enabled: v })} />
          <span className="text-sm">{t('enabled_status')}</span>
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <Switch checked={f.is_default} onCheckedChange={v => upd({ is_default: v })} />
          <span className="text-sm">{t('default_config')}</span>
        </label>
      </div>
    </div>
  )
}
