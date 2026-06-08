import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useLocale } from '@/hooks/use-locale'
import { cn } from '@/lib/utils'

interface Agent { id: number; name: string }
interface TtsConfig { config_id: string; name: string; is_default: boolean }
interface VoiceOption { value: string; label: string }
interface CloneVoicePreset { id: string; name?: string; provider_voice_id: string; tts_config_id: string; tts_config_name?: string }

export interface SpeakerGroupForm {
  agent_id: number | ''
  name: string
  prompt: string
  description: string
  tts_config_id: string
  voice: string
}

interface Props {
  open: boolean
  mode: 'add' | 'edit'
  form: SpeakerGroupForm
  onChange: (patch: Partial<SpeakerGroupForm>) => void
  onClose: () => void
  onSubmit: () => void
  agents: Agent[]
  ttsConfigs: TtsConfig[]
  currentVoiceOptions: VoiceOption[]
  cloneVoicePresets: CloneVoicePreset[]
  cloneVoicesLoading?: boolean
  submitting?: boolean
  currentTtsConfigName?: string
  currentTtsConfigInfo?: string
  isCloneVoiceSelected: (clone: CloneVoicePreset) => boolean
  onApplyCloneVoice: (clone: CloneVoicePreset) => void
  onTtsConfigChange: (id: string) => void
}

export function SpeakerGroupDialog({
  open, mode, form, onChange, onClose, onSubmit,
  agents, ttsConfigs, currentVoiceOptions, cloneVoicePresets,
  cloneVoicesLoading, submitting, currentTtsConfigName, currentTtsConfigInfo,
  isCloneVoiceSelected, onApplyCloneVoice, onTtsConfigChange,
}: Props) {
  const { t } = useLocale()

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="max-w-[580px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {mode === 'add' ? t('create_voiceprint_group') : t('edit_voiceprint_group')}
          </DialogTitle>
        </DialogHeader>

        <div className="grid gap-4 py-2">
          {/* Agent */}
          <div className="grid gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text)]">{t('link_agent')}</label>
            <Select value={form.agent_id ? String(form.agent_id) : ''} onValueChange={(v) => onChange({ agent_id: Number(v) })}>
              <SelectTrigger className="w-full"><SelectValue placeholder={t('select_agent')} /></SelectTrigger>
              <SelectContent>
                {agents.map((a) => <SelectItem key={a.id} value={String(a.id)}>{a.name}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>

          {/* Name */}
          <div className="grid gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text)]">{t('voiceprint_name')}</label>
            <Input value={form.name} onChange={(e) => onChange({ name: e.target.value })}
              placeholder={t('enter_voiceprint_name')} maxLength={100} />
          </div>

          {/* Prompt */}
          <div className="grid gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text)]">Prompt</label>
            <Textarea value={form.prompt} onChange={(e) => onChange({ prompt: e.target.value })}
              placeholder={t('role_prompt_ph')} rows={4} />
          </div>

          {/* Description */}
          <div className="grid gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text)]">{t('description')}</label>
            <Textarea value={form.description} onChange={(e) => onChange({ description: e.target.value })}
              placeholder={t('desc_optional_ph')} rows={3} maxLength={200} />
          </div>

          {/* Clone voice presets */}
          {cloneVoicePresets.length > 0 && (
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">{t('my_cloned_voice')}</label>
              <div className={cn('flex flex-wrap gap-1.5', cloneVoicesLoading && 'opacity-50')}>
                {cloneVoicePresets.map((clone) => (
                  <button key={clone.id} type="button"
                    title={`${clone.tts_config_name ?? clone.tts_config_id} · ${clone.provider_voice_id}`}
                    onClick={() => onApplyCloneVoice(clone)}
                    className={cn(
                      'inline-flex items-center px-3 py-1 rounded-full text-xs border transition-colors cursor-pointer',
                      isCloneVoiceSelected(clone)
                        ? 'border-[var(--color-primary)] bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                        : 'border-[var(--color-line)] bg-[var(--color-surface-1)] text-[var(--color-text-secondary)] hover:border-[var(--color-primary)]',
                    )}>
                    {clone.name ?? clone.provider_voice_id}
                  </button>
                ))}
              </div>
              <p className="text-xs text-[var(--color-text-tertiary)]">{t('click_auto_fill')}</p>
            </div>
          )}

          {/* TTS config */}
          <div className="grid gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text)]">{t('tts_config_label')}</label>
            <Select value={form.tts_config_id} onValueChange={(v) => { onChange({ tts_config_id: v }); onTtsConfigChange(v) }}>
              <SelectTrigger className="w-full"><SelectValue placeholder={t('select_tts_config_opt')} /></SelectTrigger>
              <SelectContent>
                {ttsConfigs.map((cfg) => (
                  <SelectItem key={cfg.config_id} value={cfg.config_id}>
                    {cfg.is_default ? t('tts_default_label', { name: cfg.name }) : cfg.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {form.tts_config_id && currentTtsConfigInfo && (
              <p className="text-xs text-[var(--color-text-tertiary)]">{currentTtsConfigInfo}</p>
            )}
          </div>

          {/* Voice (datalist) */}
          {form.tts_config_id && (
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">{t('voice_timbre')}</label>
              <Input value={form.voice} onChange={(e) => onChange({ voice: e.target.value })}
                list="speaker-group-voice-datalist" placeholder={t('select_or_enter_voice')} />
              <datalist id="speaker-group-voice-datalist">
                {currentVoiceOptions.map((v) => <option key={v.value} value={v.value}>{v.label}</option>)}
              </datalist>
              <p className="text-xs text-[var(--color-text-tertiary)]">
                {t('current_tts_config_hint', { name: currentTtsConfigName })}
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={submitting}>{t('cancel')}</Button>
          <Button onClick={onSubmit} disabled={submitting}>
            {submitting && <Loader2 className="w-4 h-4 mr-1.5 animate-spin" />}
            {mode === 'add' ? t('create') : t('save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
