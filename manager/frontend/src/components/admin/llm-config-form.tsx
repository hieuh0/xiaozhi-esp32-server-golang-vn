import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ComboInput } from '@/components/ui/combo-input'
import type { ConfigRow } from './config-list-page'
import type { ConfigForm } from './config-list-page'
import {
  resolveLLMProvider, getProviderFixedType, isProviderBaseURLEditable, getProviderQuickUrl,
  getProviderModelOptions, getProviderModelHint, getProviderModelFieldLabel,
  getProviderModelPlaceholder, getProviderRequestConfig, getProviderThinkingConfig, LLM_PROVIDERS,
} from './llm-catalog'

interface LlmFields {
  provider: string; name: string; config_id: string; enabled: boolean; is_default: boolean
  model_name: string; api_key: string; base_url: string
  max_tokens: number; temperature: number; top_p: number
  thinking_mode: string; thinking_budget: number | null; thinking_effort: string; thinking_clear: string
  bot_id: string; user_prefix: string; connector_id: string
}

const defaults: LlmFields = {
  provider: 'openai', name: '', config_id: '', enabled: true, is_default: false,
  model_name: '', api_key: '', base_url: '',
  max_tokens: 4000, temperature: 0.7, top_p: 0.9,
  thinking_mode: 'default', thinking_budget: null, thinking_effort: 'medium', thinking_clear: 'default',
  bot_id: '', user_prefix: '', connector_id: '1024',
}

function serialize(f: LlmFields): string {
  const type = getProviderFixedType(f.provider)
  const thinking = f.thinking_mode !== 'default' || f.thinking_budget != null
    ? { mode: f.thinking_mode, ...(f.thinking_budget != null ? { budget_tokens: f.thinking_budget } : {}), effort: f.thinking_effort, clear_thinking: f.thinking_clear }
    : undefined
  const data: Record<string, unknown> = { type, model_name: f.model_name, api_key: f.api_key, base_url: f.base_url }
  if (type === 'openai' || type === 'ollama') { data.max_tokens = f.max_tokens; data.temperature = f.temperature; data.top_p = f.top_p }
  if (thinking) data.thinking = thinking
  if (type === 'coze') { data.bot_id = f.bot_id; data.user_prefix = f.user_prefix; data.connector_id = f.connector_id }
  if (type === 'dify') data.user_prefix = f.user_prefix
  return JSON.stringify(data)
}

function parse(row: ConfigRow | null): LlmFields {
  if (!row) return { ...defaults }
  try {
    const d = JSON.parse(row.json_data || '{}')
    const provider = resolveLLMProvider(row.provider || '', d.type)
    return {
      ...defaults, provider, name: row.name, config_id: row.config_id,
      enabled: row.enabled !== false, is_default: !!row.is_default,
      model_name: d.model_name || '', api_key: d.api_key || '', base_url: d.base_url || '',
      max_tokens: d.max_tokens || 4000, temperature: d.temperature ?? 0.7, top_p: d.top_p ?? 0.9,
      thinking_mode: d.thinking?.mode || 'default', thinking_budget: d.thinking?.budget_tokens ?? null,
      thinking_effort: d.thinking?.effort || 'medium', thinking_clear: String(d.thinking?.clear_thinking ?? 'default'),
      bot_id: d.bot_id || '', user_prefix: d.user_prefix || '', connector_id: d.connector_id || '1024',
    }
  } catch { return { ...defaults } }
}

const F = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="grid gap-1.5"><label className="text-sm font-medium text-[var(--color-text)]">{label}</label>{children}</div>
)

export function LlmConfigForm({ form, setForm, editing }: { form: ConfigForm; setForm: (p: Partial<ConfigForm>) => void; editing: ConfigRow | null }) {
  const { t } = useLocale()
  const [f, setF] = useState<LlmFields>(() => parse(editing))
  const [testing, setTesting] = useState(false)
  const [fetchedModels, setFetchedModels] = useState<string[]>([])
  const [fetchingModels, setFetchingModels] = useState(false)

  useEffect(() => { const parsed = parse(editing); setF(parsed) }, [editing])

  const upd = (patch: Partial<LlmFields>) => {
    const next = { ...f, ...patch }
    setF(next)
    setForm({ name: next.name, config_id: next.config_id, provider: next.provider, enabled: next.enabled, is_default: next.is_default, json_data: serialize(next) })
  }

  const testConnection = async () => {
    setTesting(true)
    try {
      const { data } = await api.post<{ ok: boolean; error?: string }>('/admin/llm-configs/test-connection', {
        type: getProviderFixedType(f.provider),
        api_key: f.api_key,
        base_url: f.base_url,
        model_name: f.model_name,
      })
      if (data.ok) {
        toast.success(t('connection_test_success'))
      } else {
        toast.error(`${t('connection_test_failed')}: ${data.error || 'unknown error'}`)
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      toast.error(`${t('connection_test_failed')}: ${msg}`)
    } finally {
      setTesting(false)
    }
  }

  const fetchModels = async () => {
    setFetchingModels(true)
    try {
      const { data } = await api.post<{ ok: boolean; models: string[]; error?: string }>(
        '/admin/llm-configs/fetch-models',
        { type: getProviderFixedType(f.provider), base_url: f.base_url, api_key: f.api_key }
      )
      if (data.ok && data.models.length > 0) {
        setFetchedModels(data.models)
        toast.success(t('fetch_models_success', { count: data.models.length }))
      } else {
        toast.error(`${t('fetch_models_failed')}: ${data.error || 'no models returned'}`)
      }
    } catch (e: unknown) {
      toast.error(`${t('fetch_models_failed')}: ${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setFetchingModels(false)
    }
  }

  const onProviderChange = (p: string) => {
    const url = isProviderBaseURLEditable(p) ? getProviderQuickUrl(p) : ''
    setFetchedModels([])
    upd({ provider: p, base_url: url, model_name: '' })
  }

  const type = getProviderFixedType(f.provider)
  const isOpenAIType = type === 'openai' || type === 'ollama'
  const isDify = type === 'dify'
  const isCoze = type === 'coze'
  const showBaseURL = isProviderBaseURLEditable(f.provider)
  const catalogModelOptions = getProviderModelOptions(f.provider)
  const modelOptions = fetchedModels.length > 0
    ? fetchedModels.map(id => ({ value: id, label: id }))
    : catalogModelOptions
  const modelHint = getProviderModelHint(f.provider, f.model_name, t)
  const requestCfg = getProviderRequestConfig(f.provider, f.model_name)
  const thinkingCfg = getProviderThinkingConfig(f.provider, f.model_name, t)
  const showBudget = thinkingCfg.visible && thinkingCfg.showBudgetFor.includes(f.thinking_mode)
  const showEffort = thinkingCfg.visible && thinkingCfg.showEffortFor.includes(f.thinking_mode)
  const showClear = thinkingCfg.visible && thinkingCfg.showClearThinkingFor.includes(f.thinking_mode)

  return (
    <div className="grid gap-3">
      <F label={t('provider')}>
        <Select value={f.provider} onValueChange={onProviderChange}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{LLM_PROVIDERS.map(p => <SelectItem key={p.value} value={p.value}>{p.labelKey ? t(p.label) : p.label}</SelectItem>)}</SelectContent>
        </Select>
      </F>
      <div className="grid grid-cols-2 gap-3">
        <F label={t('config_name')}><Input value={f.name} onChange={e => upd({ name: e.target.value })} placeholder={t('enter_config_name')} /></F>
        <F label={t('config_id')}><Input value={f.config_id} onChange={e => upd({ config_id: e.target.value })} placeholder={t('enter_unique_config_id')} /></F>
      </div>
      {isOpenAIType && (
        <F label={getProviderModelFieldLabel(f.provider, t)}>
          <div className="flex gap-2">
            <ComboInput
              value={f.model_name}
              onChange={v => upd({ model_name: v })}
              options={modelOptions}
              placeholder={getProviderModelPlaceholder(f.provider, t)}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={fetchModels}
              disabled={fetchingModels}
              title={t('fetch_models')}
            >
              {fetchingModels ? (
                <span className="animate-spin text-sm">⟳</span>
              ) : (
                <span className="text-sm">🔄</span>
              )}
            </Button>
          </div>
          {modelHint && <p className="text-xs text-[var(--color-text-secondary)]">{modelHint}</p>}
          {fetchedModels.length > 0 && (
            <p className="text-xs text-green-600">{t('fetch_models_loaded', { count: fetchedModels.length })}</p>
          )}
        </F>
      )}
      {type !== 'ollama' && (
        <F label="API Key"><Input type="password" value={f.api_key} onChange={e => upd({ api_key: e.target.value })} placeholder={t('enter_api_password')} /></F>
      )}
      {showBaseURL && (
        <F label={t('base_url')}><Input value={f.base_url} onChange={e => upd({ base_url: e.target.value })} placeholder={t('enter_base_url')} /></F>
      )}
      {isCoze && <>
        <F label="Bot ID"><Input value={f.bot_id} onChange={e => upd({ bot_id: e.target.value })} placeholder={t('enter_coze_bot_id_v2')} /></F>
        <F label="Connector ID"><Input value={f.connector_id} onChange={e => upd({ connector_id: e.target.value })} placeholder="1024" /></F>
      </>}
      {(isDify || isCoze) && <F label={t('user_prefix')}><Input value={f.user_prefix} onChange={e => upd({ user_prefix: e.target.value })} placeholder="xiaozhi" /></F>}
      {isOpenAIType && requestCfg.allowMaxTokens && (
        <div className="grid grid-cols-3 gap-3">
          <F label="max_tokens"><Input type="number" value={f.max_tokens} min={1} max={100000} onChange={e => upd({ max_tokens: Number(e.target.value) })} /></F>
          {requestCfg.allowTemperature && <F label={t('temperature')}><Input type="number" value={f.temperature} min={0} max={requestCfg.temperatureMax} step={0.1} onChange={e => upd({ temperature: Number(e.target.value) })} /></F>}
          {requestCfg.allowTopP && <F label="Top P"><Input type="number" value={f.top_p} min={0} max={1} step={0.1} onChange={e => upd({ top_p: Number(e.target.value) })} /></F>}
        </div>
      )}
      {thinkingCfg.visible && (
        <F label={thinkingCfg.label}>
          <Select value={f.thinking_mode} onValueChange={v => upd({ thinking_mode: v })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>{thinkingCfg.options.map(o => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}</SelectContent>
          </Select>
        </F>
      )}
      {showBudget && <F label={t('thinking_budget')}><Input type="number" value={f.thinking_budget ?? ''} min={thinkingCfg.budgetMin} max={thinkingCfg.budgetMax} step={thinkingCfg.budgetStep} onChange={e => upd({ thinking_budget: e.target.value ? Number(e.target.value) : null })} /></F>}
      {showEffort && (
        <F label={t('thinking_level')}>
          <Select value={f.thinking_effort} onValueChange={v => upd({ thinking_effort: v })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>{thinkingCfg.effortOptions.map(o => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}</SelectContent>
          </Select>
        </F>
      )}
      {showClear && (
        <F label={t('history_thinking_chain')}>
          <Select value={String(f.thinking_clear)} onValueChange={v => upd({ thinking_clear: v })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>{thinkingCfg.clearThinkingOptions.map(o => <SelectItem key={String(o.value)} value={String(o.value)}>{o.label}</SelectItem>)}</SelectContent>
          </Select>
        </F>
      )}
      {thinkingCfg.visible && thinkingCfg.hint && (
        <p className="text-xs text-amber-600 bg-amber-50 rounded-md px-3 py-2">{thinkingCfg.hint}</p>
      )}
      <div className="flex items-center gap-6 pt-1">
        <label className="flex items-center gap-2 cursor-pointer">
          <Switch checked={f.enabled} onCheckedChange={v => upd({ enabled: v })} /><span className="text-sm">{t('enabled_status')}</span>
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <Switch checked={f.is_default} onCheckedChange={v => upd({ is_default: v })} /><span className="text-sm">{t('default_config')}</span>
        </label>
        <Button type="button" variant="outline" size="sm" onClick={testConnection} disabled={testing} className="ml-auto">
          {testing ? t('connecting') : t('test_connection')}
        </Button>
      </div>
    </div>
  )
}
