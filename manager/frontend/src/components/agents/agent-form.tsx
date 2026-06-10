import { forwardRef, useEffect, useImperativeHandle, useRef } from 'react'
import { toast } from 'sonner'
import { useAgentFormOptions } from '@/hooks/use-agent-form-options'
import type { AgentFormData, LLMConfig, TTSConfig } from '@/features/agents/types'
import { buildAgentPayload } from '@/features/agents/types'
import { useLocale } from '@/hooks/use-locale'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

export interface AgentFormHandle {
  validate: () => Promise<boolean>
  buildPayload: () => Record<string, unknown>
  reloadOptions: () => Promise<void>
}

interface Props {
  value: AgentFormData
  onChange: (data: AgentFormData) => void
  isAdmin?: boolean
  mode?: 'create' | 'edit'
}

const Field = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="grid gap-1.5">
    <label className="text-sm font-medium text-[var(--color-text)]">{label}</label>
    {children}
  </div>
)

export const AgentForm = forwardRef<AgentFormHandle, Props>(({ value, onChange, isAdmin = false, mode = 'create' }, ref) => {
  const { t } = useLocale()
  const opts = useAgentFormOptions()
  const ttsRef = useRef<string | null>(value.tts_config_id)

  const set = (patch: Partial<AgentFormData>) => onChange({ ...value, ...patch })

  const getProvider = (cfgs: TTSConfig[], id: string | null) => cfgs.find((c) => c.config_id === id)?.provider ?? ''

  const handleTtsChange = async (newId: string | null) => {
    const prev = ttsRef.current
    ttsRef.current = newId
    set({ tts_config_id: newId, voice: null })
    const provider = getProvider(opts.ttsConfigs, newId)
    if (newId && provider) {
      const voices = await opts.refreshVoices({ ttsConfigId: newId, provider, clearInvalid: true, previousConfigId: prev })
      if (mode === 'create' && !value.voice && voices.length) set({ tts_config_id: newId, voice: null })
    } else {
      await opts.refreshVoices({ ttsConfigId: null })
    }
  }

  const applyClone = async (clone: { tts_config_id: string; provider_voice_id: string }) => {
    if (clone.tts_config_id !== value.tts_config_id) await handleTtsChange(clone.tts_config_id)
    set({ voice: clone.provider_voice_id })
  }

  const toggleMcp = (svc: string) => {
    const curr = value.mcp_service_names.split(',').map((s) => s.trim()).filter(Boolean)
    const next = curr.includes(svc) ? curr.filter((s) => s !== svc) : [...curr, svc]
    set({ mcp_service_names: next.join(',') })
  }

  const addKeyword = (field: 'openclaw_enter_keywords' | 'openclaw_exit_keywords', kw: string) => {
    const trimmed = kw.trim()
    if (!trimmed || value[field].includes(trimmed)) return
    set({ [field]: [...value[field], trimmed] })
  }
  const removeKeyword = (field: 'openclaw_enter_keywords' | 'openclaw_exit_keywords', kw: string) => {
    set({ [field]: value[field].filter((k) => k !== kw) })
  }

  useEffect(() => {
    opts.load({ isAdmin, userId: isAdmin ? value.user_id : null, ttsConfigId: value.tts_config_id }).then(async () => {
      if (value.tts_config_id) {
        const provider = getProvider(opts.ttsConfigs, value.tts_config_id)
        if (provider) await opts.refreshVoices({ ttsConfigId: value.tts_config_id, provider, clearInvalid: false })
      }
      if (mode === 'create') {
        const patch: Partial<AgentFormData> = {}
        if (!value.llm_config_id) { const d = opts.llmConfigs.find((c: LLMConfig) => c.is_default); if (d) patch.llm_config_id = d.config_id }
        if (!value.tts_config_id) { const d = opts.ttsConfigs.find((c: TTSConfig) => c.is_default); if (d) patch.tts_config_id = d.config_id }
        if (Object.keys(patch).length) set(patch)
      }
    })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useImperativeHandle(ref, () => ({
    validate: async () => {
      if (isAdmin && !value.user_id) { toast.error(t('select_owner_user')); return false }
      if (!value.name.trim()) { toast.error(t('enter_agent_name')); return false }
      if (!value.nickname.trim()) { toast.error(t('enter_agent_nickname')); return false }
      return true
    },
    buildPayload: () => buildAgentPayload(value, { isAdmin }),
    reloadOptions: async () => opts.load({ isAdmin, userId: isAdmin ? value.user_id : null, ttsConfigId: value.tts_config_id }),
  }))

  const selectedMcpSet = new Set(value.mcp_service_names.split(',').map((s) => s.trim()).filter(Boolean))

  return (
    <div className="grid gap-4">
      {isAdmin && (
        <Field label={t('owner_user')}>
          <Select value={value.user_id ? String(value.user_id) : ''} onValueChange={(v) => set({ user_id: Number(v) || null })}>
            <SelectTrigger><SelectValue placeholder={t('select_owner_user')} /></SelectTrigger>
            <SelectContent>{opts.users.map((u) => <SelectItem key={u.id} value={String(u.id)}>{u.username || u.name} (ID: {u.id})</SelectItem>)}</SelectContent>
          </Select>
        </Field>
      )}
      <div className="grid grid-cols-2 gap-4">
        <Field label={t('agent_name')}><Input value={value.name} onChange={(e) => set({ name: e.target.value })} maxLength={50} placeholder={t('enter_admin_display_name')} /></Field>
        <Field label={t('agent_nickname')}><Input value={value.nickname} onChange={(e) => set({ nickname: e.target.value })} maxLength={50} placeholder={t('model_display_name_hint')} /></Field>
      </div>
      <Field label={t('role_description')}>
        <Textarea value={value.custom_prompt} onChange={(e) => set({ custom_prompt: e.target.value })} rows={4} maxLength={10000} placeholder={t('enter_role_prompt')} />
      </Field>
      <Field label={t('link_knowledge_base')}>
        <div className="border border-[var(--color-line)] rounded-lg max-h-32 overflow-y-auto p-1.5">
          {opts.knowledgeBases.length === 0 ? (
            <p className="text-xs text-[var(--color-text-tertiary)] p-2 text-center">{t('select_linked_knowledge_base')}</p>
          ) : opts.knowledgeBases.map((kb) => (
            <label key={kb.id} className="flex items-center gap-2 p-1.5 rounded hover:bg-[var(--color-surface-muted)] cursor-pointer text-sm">
              <input type="checkbox" checked={value.knowledge_base_ids.includes(kb.id)} onChange={(e) => set({ knowledge_base_ids: e.target.checked ? [...value.knowledge_base_ids, kb.id] : value.knowledge_base_ids.filter((id) => id !== kb.id) })} className="accent-[var(--color-primary)]" />
              <span>{kb.name}</span>
            </label>
          ))}
        </div>
      </Field>
      <div className="grid grid-cols-2 gap-4">
        <Field label={t('language_model')}>
          <Select value={value.llm_config_id ?? ''} onValueChange={(v) => set({ llm_config_id: v || null })}>
            <SelectTrigger><SelectValue placeholder={t('select_language_model')} /></SelectTrigger>
            <SelectContent>{opts.llmConfigs.map((c) => <SelectItem key={c.config_id} value={c.config_id}>{c.is_default ? t('tts_default_label', { name: c.name }) : c.name}</SelectItem>)}</SelectContent>
          </Select>
        </Field>
        <Field label={t('tts_config_label')}>
          <Select value={value.tts_config_id ?? ''} onValueChange={(v) => handleTtsChange(v || null)}>
            <SelectTrigger><SelectValue placeholder={t('select_tts_config')} /></SelectTrigger>
            <SelectContent>{opts.ttsConfigs.map((c) => <SelectItem key={c.config_id} value={c.config_id}>{c.is_default ? t('tts_default_label', { name: c.name }) : c.name}</SelectItem>)}</SelectContent>
          </Select>
        </Field>
      </div>
      {value.tts_config_id && (
        <Field label={t('tts_voice')}>
          <Input value={value.voice ?? ''} onChange={(e) => set({ voice: e.target.value || null })} list="agent-voice-list" placeholder={t('select_or_enter_timbre')} />
          <datalist id="agent-voice-list">{opts.filteredVoiceOptions.map((v) => <option key={v.value} value={v.value}>{v.label || v.value}</option>)}</datalist>
        </Field>
      )}
      {opts.cloneVoices.length > 0 && (
        <div className="flex flex-wrap gap-1.5 -mt-2">
          {opts.cloneVoices.map((clone) => (
            <button key={clone.id} type="button" onClick={() => applyClone(clone)}
              className={`inline-flex items-center px-2.5 py-1 rounded-lg text-xs border transition-colors cursor-pointer ${value.tts_config_id === clone.tts_config_id && value.voice === clone.provider_voice_id ? 'status-primary' : 'border-[var(--color-line)] text-[var(--color-text-secondary)] hover:border-[var(--color-primary)]'}`}>
              {clone.name || clone.provider_voice_id}
            </button>
          ))}
        </div>
      )}
      <div className="grid grid-cols-3 gap-4">
        {([['asr_speed', [['normal', t('normal')], ['patient', t('patience')], ['fast', t('fast')]], t('asr_speed')],
          ['memory_mode', [['none', t('no_memory')], ['short', t('short_memory')], ['long', t('long_memory')]], t('memory_mode')],
          ['speaker_chat_mode', [['off', t('close')], ['identified_only', t('voiceprint_only_chat')]], t('voiceprint_chat_limit')],
        ] as [keyof AgentFormData, [string, string][], string][]).map(([field, opts2, label]) => (
          <Field key={field} label={label}>
            <Select value={String(value[field])} onValueChange={(v) => set({ [field]: v } as Partial<AgentFormData>)}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>{opts2.map(([val, lbl]) => <SelectItem key={val} value={val}>{lbl}</SelectItem>)}</SelectContent>
            </Select>
          </Field>
        ))}
      </div>
      <Field label={t('mcp_service')}>
        <div className="border border-[var(--color-line)] rounded-lg max-h-32 overflow-y-auto p-1.5">
          {opts.mcpServiceOptions.length === 0 ? (
            <p className="text-xs text-[var(--color-text-tertiary)] p-2 text-center">{t('leave_blank_all_enabled')}</p>
          ) : opts.mcpServiceOptions.map((svc) => (
            <label key={svc} className="flex items-center gap-2 p-1.5 rounded hover:bg-[var(--color-surface-muted)] cursor-pointer text-sm">
              <input type="checkbox" checked={selectedMcpSet.has(svc)} onChange={() => toggleMcp(svc)} className="accent-[var(--color-primary)]" />
              <span>{svc}</span>
            </label>
          ))}
        </div>
      </Field>
      <div className="border border-[var(--color-line)] rounded-xl p-4 bg-[var(--color-surface-2)] grid gap-3">
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm font-semibold text-[var(--color-text)]">{t('allow_openclaw_mode')}</span>
          <Switch checked={!!value.openclaw_allowed} onCheckedChange={(v) => set({ openclaw_allowed: v })} />
        </div>
        <div className="grid grid-cols-2 gap-4">
          {(['openclaw_enter_keywords', 'openclaw_exit_keywords'] as const).map((field) => (
            <div key={field} className="grid gap-1.5">
              <label className="text-xs font-medium text-[var(--color-text-secondary)]">{field === 'openclaw_enter_keywords' ? t('openclaw_enter_keyword') : t('openclaw_exit_keyword')}</label>
              <div className="flex flex-wrap gap-1 min-h-[28px]">
                {value[field].map((kw) => (
                  <span key={kw} className="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full border text-xs status-primary">
                    {kw}<button type="button" className="hover:opacity-70 ml-0.5" onClick={() => removeKeyword(field, kw)}>×</button>
                  </span>
                ))}
              </div>
              <Input className="h-7 text-xs" placeholder={t('enter_press_add_keywords')}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addKeyword(field, (e.target as HTMLInputElement).value); (e.target as HTMLInputElement).value = '' } }} />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
})

AgentForm.displayName = 'AgentForm'
