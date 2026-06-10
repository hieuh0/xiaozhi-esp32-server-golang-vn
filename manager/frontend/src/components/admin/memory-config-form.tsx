import { useEffect, useState } from 'react'
import { useLocale } from '@/hooks/use-locale'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { ConfigRow, ConfigForm } from './config-list-page'

interface MemoryFields {
  provider: string; name: string; config_id: string; enabled: boolean; is_default: boolean
  api_key: string; base_url: string
  enable_search: boolean; search_threshold: number; search_top_k: number; timeout_ms: number
}

const DEFAULT_URLS: Record<string, string> = {
  memobase: 'https://api.memobase.dev',
  mem0: 'https://api.mem0.ai',
  memos: 'https://memos.memtensor.cn/api/openmem/v1',
}

const D: MemoryFields = {
  provider: 'memobase', name: '', config_id: '', enabled: true, is_default: false,
  api_key: '', base_url: DEFAULT_URLS.memobase,
  enable_search: true, search_threshold: 0.5, search_top_k: 3, timeout_ms: 10000,
}

function parse(row: ConfigRow | null): MemoryFields {
  if (!row) return { ...D }
  try {
    const d = JSON.parse(row.json_data || '{}')
    return {
      ...D,
      provider: row.provider || 'memobase',
      name: row.name, config_id: row.config_id,
      enabled: row.enabled !== false, is_default: !!row.is_default,
      api_key: d.api_key || '',
      base_url: d.base_url || DEFAULT_URLS[row.provider || 'memobase'] || '',
      enable_search: d.enable_search !== undefined ? !!d.enable_search : true,
      search_threshold: d.search_threshold !== undefined ? d.search_threshold : 0.5,
      search_top_k: d.search_top_k !== undefined ? d.search_top_k : 3,
      timeout_ms: d.timeout_ms !== undefined ? d.timeout_ms : 10000,
    }
  } catch { return { ...D } }
}

function serialize(f: MemoryFields): string {
  const config: Record<string, unknown> = {
    api_key: f.api_key,
    base_url: f.base_url,
    enable_search: f.enable_search,
    search_threshold: f.search_threshold,
    search_top_k: f.search_top_k,
  }
  if (f.provider === 'memos') config.timeout_ms = f.timeout_ms
  return JSON.stringify(config)
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

const MEMORY_PROVIDERS = [
  { label: 'Memobase', value: 'memobase' },
  { label: 'Mem0', value: 'mem0' },
  { label: 'MemOS', value: 'memos' },
]

export function MemoryConfigForm({ form, setForm, editing }: { form: ConfigForm; setForm: (p: Partial<ConfigForm>) => void; editing: ConfigRow | null }) {
  const { t } = useLocale()
  const [f, setF] = useState<MemoryFields>(() => parse(editing))
  useEffect(() => { setF(parse(editing)) }, [editing])

  const upd = (patch: Partial<MemoryFields>) => {
    const next = { ...f, ...patch }
    setF(next)
    setForm({ name: next.name, config_id: next.config_id, provider: next.provider, enabled: next.enabled, is_default: next.is_default, json_data: serialize(next) })
  }

  const onProviderChange = (value: string) => {
    upd({ provider: value, api_key: '', base_url: DEFAULT_URLS[value] || '', enable_search: true, search_threshold: 0.5, search_top_k: 3, timeout_ms: 10000 })
  }

  return (
    <div className="grid gap-3">
      <F label={t('provider')}>
        <Select value={f.provider} onValueChange={onProviderChange}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{MEMORY_PROVIDERS.map(o => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}</SelectContent>
        </Select>
      </F>
      <div className="grid grid-cols-2 gap-3">
        <F label={t('config_name')}><Input value={f.name} onChange={e => upd({ name: e.target.value })} placeholder={t('enter_config_name')} /></F>
        <F label={t('config_id')}><Input value={f.config_id} onChange={e => upd({ config_id: e.target.value })} placeholder={t('enter_unique_config_id')} /></F>
      </div>
      <F label={t('api_key')}>
        <Input type="password" value={f.api_key} onChange={e => upd({ api_key: e.target.value })}
          placeholder={f.provider === 'memos' ? t('enter_memos_api_key') : f.provider === 'mem0' ? t('enter_mem0_api_key') : t('memobase_api_key_ph')} />
      </F>
      <F label={t('base_url')}>
        <Input value={f.base_url} onChange={e => upd({ base_url: e.target.value })}
          placeholder={f.provider === 'memos' ? t('enter_memos_base_url') : f.provider === 'mem0' ? t('enter_mem0_base_url') : t('memobase_base_url_ph')} />
      </F>
      <div className="grid grid-cols-2 gap-3">
        <F label={t('search_threshold')}><N v={f.search_threshold} min={0} max={1} step={0.1} onChange={v => upd({ search_threshold: v })} /></F>
        <F label={t('search_top_k')}><N v={f.search_top_k} min={1} step={1} onChange={v => upd({ search_top_k: v })} /></F>
      </div>
      {f.provider === 'memos' && (
        <F label={t('fetch_timeout_ms')}><N v={f.timeout_ms} min={1000} step={1000} onChange={v => upd({ timeout_ms: v })} /></F>
      )}
      <div className="flex items-center gap-6 pt-1">
        <label className="flex items-center gap-2 cursor-pointer">
          <Switch checked={f.enable_search} onCheckedChange={v => upd({ enable_search: v })} />
          <span className="text-sm">{t('enable_search')}</span>
        </label>
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
