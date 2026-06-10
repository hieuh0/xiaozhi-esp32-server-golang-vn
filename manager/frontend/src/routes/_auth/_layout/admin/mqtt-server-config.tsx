import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { PageHeader } from '@/components/ui/page-header'
import { cn } from '@/lib/utils'

interface TlsConfig { enable: boolean; port: number; pem: string; key: string }
interface MqttServerForm {
  enable: boolean
  listen_host: string
  listen_port: number
  username: string
  password: string
  signature_key: string
  enable_auth: boolean
  tls: TlsConfig
}

const defaults: MqttServerForm = {
  enable: true, listen_host: '0.0.0.0', listen_port: 1883, username: '', password: '',
  signature_key: 'xiaozhi_ota_signature_key', enable_auth: false,
  tls: { enable: false, port: 8883, pem: '', key: '' }
}

function MqttServerConfigPage() {
  const { t } = useLocale()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [configId, setConfigId] = useState<number | null>(null)
  const [form, setForm] = useState<MqttServerForm>(defaults)

  const setF = (patch: Partial<MqttServerForm>) => setForm((f) => ({ ...f, ...patch }))
  const setTls = (patch: Partial<TlsConfig>) => setForm((f) => ({ ...f, tls: { ...f.tls, ...patch } }))

  const serverReady = Boolean(form.listen_host.trim() && form.listen_port)

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/mqtt-server-configs')
      const configs = res.data?.data || []
      if (configs.length > 0) {
        const c = configs[0]; setConfigId(c.id)
        const d = JSON.parse(c.json_data || '{}')
        setForm({
          enable: d.enable !== false, listen_host: d.listen_host || '0.0.0.0',
          listen_port: Number(d.listen_port) > 0 ? Number(d.listen_port) : 1883,
          username: d.username || '', password: d.password || '',
          signature_key: d.signature_key || 'xiaozhi_ota_signature_key',
          enable_auth: !!d.enable_auth,
          tls: { enable: !!d.tls?.enable, port: Number(d.tls?.port) > 0 ? Number(d.tls.port) : 8883, pem: d.tls?.pem || '', key: d.tls?.key || '' }
        })
      } else { setConfigId(null); setForm(defaults) }
    } catch { toast.error(t('load_mqtt_server_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSave = async () => {
    if (!form.listen_host.trim()) { toast.error(t('enter_listen_host')); return }
    if (!form.signature_key.trim()) { toast.error(t('enter_signature_key')); return }
    setSaving(true)
    try {
      const payload = {
        name: t('mqtt_server_config_label'), config_id: 'mqtt_server_mqtt_server_config', provider: 'mqtt_server', enabled: true, is_default: true,
        json_data: JSON.stringify({ enable: form.enable, listen_host: form.listen_host.trim(), listen_port: Number(form.listen_port), username: form.username.trim(), password: form.password, signature_key: form.signature_key.trim(), enable_auth: form.enable_auth, tls: { enable: form.tls.enable, port: Number(form.tls.port), pem: form.tls.pem.trim(), key: form.tls.key.trim() } })
      }
      if (configId) { await api.put(`/admin/mqtt-server-configs/${configId}`, payload); toast.success(t('mqtt_server_config_updated')) }
      else { const r = await api.post('/admin/mqtt-server-configs', payload); setConfigId(r.data?.data?.id || null); toast.success(t('mqtt_server_config_saved')) }
      await load()
    } catch (e) { toast.error((e as Error).message || t('save_mqtt_server_failed')) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>

  const badge = (ok: boolean, yes: string, no: string) => (
    <span className={cn('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1', ok ? 'status-success' : 'status-warning')}>
      {ok ? yes : no}
    </span>
  )

  return (
    <div className="grid gap-6 px-6 pb-8">
      <PageHeader title={t('mqtt_server_config')} />
      <div className="grid gap-6 [grid-template-columns:minmax(0,1.25fr)_minmax(340px,0.95fr)]">
        <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
          <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">MQTT Server</p>
              <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('listen_and_access')}</h3>
              <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{t('mqtt_server_config_desc')}</p>
            </div>
            {badge(serverReady, t('service_params_complete'), t('pending_fill'))}
          </div>
          <div className="p-6 grid gap-5">
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="text-sm font-semibold text-[var(--color-text)]">{t('enable_mqtt_server')}</p>
                <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{t('enable_mqtt_server_help')}</p>
              </div>
              <Switch checked={form.enable} onCheckedChange={(v) => setF({ enable: v })} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('listen_host')}</label>
              <Input value={form.listen_host} onChange={(e) => setF({ listen_host: e.target.value })} placeholder="0.0.0.0" />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('listen_port')}</label>
              <Input type="number" value={form.listen_port} onChange={(e) => setF({ listen_port: Number(e.target.value) })} min={1} max={65535} />
            </div>
          </div>
        </div>

        <div className="grid gap-6 content-start">
          <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
            <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
              <div>
                <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('authentication')}</p>
                <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('auth_and_signing')}</h3>
              </div>
              <span className={cn('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1', form.enable_auth ? 'status-warning' : 'status-muted')}>
                {form.enable_auth ? t('auth_enabled') : t('anonymous_access')}
              </span>
            </div>
            <div className="p-6 grid gap-5">
              <div className="flex items-center justify-between gap-4">
                <p className="text-sm font-semibold text-[var(--color-text)]">{t('verify_mqtt_auth')}</p>
                <Switch checked={form.enable_auth} onCheckedChange={(v) => setF({ enable_auth: v })} />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('admin_user')}</label>
                <Input value={form.username} onChange={(e) => setF({ username: e.target.value })} />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('admin_password')}</label>
                <Input type="password" value={form.password} onChange={(e) => setF({ password: e.target.value })} />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('signature_key')}</label>
                <Input value={form.signature_key} onChange={(e) => setF({ signature_key: e.target.value })} />
                <p className="text-xs text-[var(--color-text-secondary)]">{t('signature_key_note_ota')}</p>
              </div>
            </div>
          </div>

          <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
            <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
              <div>
                <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">MQTTS</p>
                <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('tls_config')}</h3>
              </div>
              <span className={cn('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1', form.tls.enable ? 'status-success' : 'status-muted')}>
                {form.tls.enable ? t('tls_enabled') : t('tls_not_enabled')}
              </span>
            </div>
            <div className="p-6 grid gap-5">
              <div className="flex items-center justify-between gap-4">
                <p className="text-sm font-semibold text-[var(--color-text)]">{t('allow_devices_mqtts')}</p>
                <Switch checked={form.tls.enable} onCheckedChange={(v) => setTls({ enable: v })} />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('tls_port')}</label>
                <Input type="number" value={form.tls.port} onChange={(e) => setTls({ port: Number(e.target.value) })} min={1} max={65535} />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('cert_file')}</label>
                <Input value={form.tls.pem} onChange={(e) => setTls({ pem: e.target.value })} placeholder="/path/to/cert.pem" />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold text-[var(--color-text)]">{t('key_file')}</label>
                <Input value={form.tls.key} onChange={(e) => setTls({ key: e.target.value })} placeholder="/path/to/key.pem" />
              </div>
            </div>
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

export const Route = createFileRoute('/_auth/_layout/admin/mqtt-server-config')({
  component: MqttServerConfigPage,
})
