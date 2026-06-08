import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Input } from '@/components/ui/input'
import { PageHeader } from '@/components/ui/page-header'

interface ChatSettingsData {
  auth: { enable: boolean; login_captcha_enabled: boolean }
  chat: { max_idle_duration: number; chat_max_silence_duration: number; realtime_mode: number; global_system_prompt: string }
}

const defaults: ChatSettingsData = {
  auth: { enable: false, login_captcha_enabled: true },
  chat: { max_idle_duration: 30000, chat_max_silence_duration: 400, realtime_mode: 4, global_system_prompt: '' },
}

function ChatSettingsPage() {
  const { t } = useLocale()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState<ChatSettingsData>(defaults)

  const setAuth = (patch: Partial<ChatSettingsData['auth']>) => setForm((f) => ({ ...f, auth: { ...f.auth, ...patch } }))
  const setChat = (patch: Partial<ChatSettingsData['chat']>) => setForm((f) => ({ ...f, chat: { ...f.chat, ...patch } }))

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/chat-settings')
      const d = res.data?.data || {}
      setForm({
        auth: { enable: !!d.auth?.enable, login_captcha_enabled: d.auth?.login_captcha_enabled !== false },
        chat: {
          max_idle_duration: Number(d.chat?.max_idle_duration ?? 30000),
          chat_max_silence_duration: Number(d.chat?.chat_max_silence_duration ?? 400),
          realtime_mode: Number(d.chat?.realtime_mode ?? 4),
          global_system_prompt: String(d.chat?.global_system_prompt ?? ''),
        },
      })
    } catch { toast.error(t('load_chat_settings_failed')) }
    finally { setLoading(false) }
  }

  const save = async () => {
    setSaving(true)
    try {
      await api.put('/admin/chat-settings', form)
      toast.success(t('chat_settings_save_success'))
    } catch { toast.error(t('chat_settings_save_failed')) }
    finally { setSaving(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  if (loading) return <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('chat_settings')} />
      <div className="flex justify-end items-center gap-2">
        <Button variant="outline" onClick={load}>{t('refresh')}</Button>
        <Button disabled={saving} onClick={save}>{t('save_settings')}</Button>
      </div>

      <div className="max-w-[720px] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
        <div className="p-6 grid gap-6">
          <div>
            <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-4">{t('identity_verification')}</p>
            <div className="grid gap-5">
              <div className="flex items-center justify-between gap-4">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('enable_device_activation')}</label>
                <Switch checked={form.auth.enable} onCheckedChange={(v) => setAuth({ enable: v })} />
              </div>
              <div className="grid gap-1.5">
                <div className="flex items-center justify-between gap-4">
                  <label className="text-sm font-semibold text-[var(--color-text)]">{t('login_digit_verify')}</label>
                  <Switch checked={form.auth.login_captcha_enabled} onCheckedChange={(v) => setAuth({ login_captcha_enabled: v })} />
                </div>
                <p className="text-xs text-[var(--color-text-secondary)]">{t('captcha_enabled_hint')}</p>
              </div>
            </div>
          </div>

          <div className="pt-6 border-t border-[var(--color-line)]">
            <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-4">{t('chat_params')}</p>
            <div className="grid gap-5">
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('session_max_idle_time')}</label>
                <Input type="number" value={form.chat.max_idle_duration} onChange={(e) => setChat({ max_idle_duration: Number(e.target.value) })} />
                <p className="text-xs text-[var(--color-text-secondary)]">{t('session_idle_hint')}</p>
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('sentence_end_silence_threshold')}</label>
                <Input type="number" value={form.chat.chat_max_silence_duration} onChange={(e) => setChat({ chat_max_silence_duration: Number(e.target.value) })} />
                <p className="text-xs text-[var(--color-text-secondary)]">{t('chat_silence_hint')}</p>
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('realtime_interrupt_mode')}</label>
                <Select value={String(form.chat.realtime_mode)} onValueChange={(v) => setChat({ realtime_mode: Number(v) })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1">{t('vad_interrupt_mode_1')}</SelectItem>
                    <SelectItem value="2">{t('asr_interrupt_mode')}</SelectItem>
                    <SelectItem value="3">{t('asr_voiceprint_interrupt')}</SelectItem>
                    <SelectItem value="4">{t('asr_result_interrupt')}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('global_system_prompt_desc')}</label>
                <Textarea
                  value={form.chat.global_system_prompt}
                  onChange={(e) => setChat({ global_system_prompt: e.target.value })}
                  rows={6}
                  maxLength={8000}
                  placeholder={t('system_prompt_prefix_hint')}
                />
                <p className="text-xs text-[var(--color-text-secondary)] text-right">{form.chat.global_system_prompt.length} / 8000</p>
                <p className="text-xs text-[var(--color-text-secondary)]">{t('system_prompt_order_hint')}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/chat-settings')({
  component: ChatSettingsPage,
})
