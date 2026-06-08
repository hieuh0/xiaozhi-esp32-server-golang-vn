import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { PageHeader } from '@/components/ui/page-header'

interface SpeakerForm { base_url: string; threshold: number; enabled: boolean }

const defaults: SpeakerForm = { base_url: 'http://192.168.208.214:8080', threshold: 0.4, enabled: true }

function SpeakerConfigPage() {
  const { t } = useLocale()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [currentId, setCurrentId] = useState<number | null>(null)
  const [form, setForm] = useState<SpeakerForm>(defaults)

  const setF = (patch: Partial<SpeakerForm>) => setForm((f) => ({ ...f, ...patch }))

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/speaker-configs')
      const configs = res.data?.data || []
      if (configs.length > 0) {
        const c = configs[0]; setCurrentId(c.id)
        const d = JSON.parse(c.json_data || '{}')
        setForm({
          base_url: d.service?.base_url || d.base_url || 'http://192.168.208.214:8080',
          threshold: d.service?.threshold !== undefined ? d.service.threshold : (d.threshold ?? 0.4),
          enabled: d.enable !== false
        })
      } else { setCurrentId(null); setForm(defaults) }
    } catch { toast.error(t('load_config_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSave = async () => {
    if (!form.base_url.trim()) { toast.error(t('enter_service_address')); return }
    if (form.threshold < 0 || form.threshold > 1) { toast.error(t('threshold_0_to_1_decimal')); return }
    setSaving(true)
    try {
      const payload = {
        name: t('voiceprint_recognition_config'), config_id: 'asr_server', provider: 'asr_server', is_default: true, enabled: form.enabled,
        json_data: JSON.stringify({ service: { base_url: form.base_url.trim(), threshold: form.threshold }, enable: form.enabled })
      }
      if (currentId) { await api.put(`/admin/speaker-configs/${currentId}`, payload); toast.success(t('config_update_success')) }
      else { const r = await api.post('/admin/speaker-configs', payload); setCurrentId(r.data?.data?.id || null); toast.success(t('config_create_success')) }
      await load()
    } catch (e) { toast.error((e as Error).message || t('save_failed')) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>

  return (
    <div className="grid gap-6 px-6 pb-8">
      <PageHeader title={t('voiceprint_recognition_config')} />
      <div className="max-w-[640px] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
        <div className="p-6 grid gap-5">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-semibold text-[var(--color-text)]">{t('enabled_status')}</p>
              <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{t('enable_voiceprint_service')}</p>
            </div>
            <Switch checked={form.enabled} onCheckedChange={(v) => setF({ enabled: v })} />
          </div>
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('service_address')}</label>
            <Input value={form.base_url} onChange={(e) => setF({ base_url: e.target.value })} placeholder="http://192.168.0.1:8080" />
            <p className="text-xs text-[var(--color-text-secondary)]">{t('speaker_service_url_hint')}</p>
          </div>
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('recognition_threshold')}</label>
            <Input type="number" value={form.threshold} onChange={(e) => setF({ threshold: Number(e.target.value) })} min={0} max={1} step={0.01} />
            <p className="text-xs text-[var(--color-text-secondary)]">{t('threshold_0_to_1_hint')}</p>
          </div>
        </div>
      </div>

      <div className="flex items-center justify-end gap-3">
        <Button variant="outline" disabled={loading} onClick={load}>{t('reset_to_current')}</Button>
        <Button disabled={saving} onClick={handleSave}>{t('save_config')}</Button>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/speaker-config')({
  component: SpeakerConfigPage,
})
