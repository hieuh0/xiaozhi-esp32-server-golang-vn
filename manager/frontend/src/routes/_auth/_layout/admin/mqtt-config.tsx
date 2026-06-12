import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PageHeader } from '@/components/ui/page-header'
import { cn } from '@/lib/utils'

interface MqttClientStatus {
  configured: boolean
  broker: string
  type: string
  port: number
  enable: boolean
  connected: boolean | null
  broker_url: string | null
}

interface MqttForm {
  name: string
  broker: string
  type: string
  port: number
  client_id: string
  username: string
  password: string
  enable: boolean
}

const defaults: MqttForm = {
  name: 'MQTT Config', broker: '', type: 'tcp', port: 1883, client_id: '', username: '', password: '', enable: true
}

function MqttConfigPage() {
  const { t } = useLocale()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [configId, setConfigId] = useState<number | null>(null)
  const [form, setForm] = useState<MqttForm>(defaults)

  const setF = (patch: Partial<MqttForm>) => setForm((f) => ({ ...f, ...patch }))

  const { data: statusData, isLoading: statusLoading } = useQuery<{ data: MqttClientStatus }>({
    queryKey: ['mqtt-client-status'],
    queryFn: () => api.get('/admin/mqtt-status').then(r => r.data),
    refetchInterval: 10_000,
  })
  const clientStatus = statusData?.data

  const isCoreComplete = Boolean(form.broker.trim() && form.client_id.trim() && form.type && form.port)
  const hasCredentials = Boolean(form.username.trim() || form.password.trim())

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/mqtt-configs')
      const configs = res.data?.data || []
      if (configs.length > 0) {
        const c = configs[0]
        setConfigId(c.id)
        const d = JSON.parse(c.json_data || '{}')
        setForm({
          name: c.name || 'MQTT Config', broker: d.broker || '', type: d.type || 'tcp',
          port: Number(d.port) > 0 ? Number(d.port) : 1883, client_id: d.client_id || '',
          username: d.username || '', password: d.password || '',
          enable: d.enable !== false
        })
      } else {
        setConfigId(null); setForm(defaults)
      }
    } catch { toast.error(t('load_mqtt_config_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSave = async () => {
    if (!form.broker.trim()) { toast.error(t('enter_mqtt_broker')); return }
    if (!form.client_id.trim()) { toast.error(t('enter_client_id')); return }
    setSaving(true)
    try {
      const name = form.name || 'MQTT Config'
      const configIdSlug = `mqtt_${name.replace(/[^a-zA-Z0-9]/g, '_').toLowerCase()}`
      const payload = {
        name, config_id: configIdSlug, is_default: true,
        json_data: JSON.stringify({ enable: form.enable, broker: form.broker.trim(), type: form.type, port: Number(form.port), client_id: form.client_id.trim(), username: form.username.trim(), password: form.password })
      }
      if (configId) { await api.put(`/admin/mqtt-configs/${configId}`, payload); toast.success(t('mqtt_config_updated')) }
      else { const r = await api.post('/admin/mqtt-configs', payload); setConfigId(r.data?.data?.id || null); toast.success(t('mqtt_config_saved')) }
      await load()
    } catch (e) { toast.error((e as Error).message || t('save_mqtt_failed')) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>

  return (
    <div className="grid gap-6 px-6 pb-8">
      <PageHeader title={t('mqtt_config')} />
      <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] px-6 py-4 flex items-center justify-between gap-4">
        <div>
          <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">
            {t('mqtt_client_config_status')}
          </p>
          <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
            {clientStatus?.broker_url
              || (clientStatus?.configured && clientStatus.broker
                ? `${clientStatus.type}://${clientStatus.broker}:${clientStatus.port}`
                : clientStatus?.configured ? '—' : '—')}
          </p>
        </div>
        {statusLoading && !clientStatus ? (
          <span className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold status-muted">
            {t('status_checking')}
          </span>
        ) : (
          <span className={cn(
            'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold',
            clientStatus?.connected === true ? 'status-success'
              : clientStatus?.connected === false ? 'status-danger'
              : clientStatus?.configured ? 'status-success' : 'status-muted'
          )}>
            {clientStatus?.connected === true ? t('mqtt_client_connected')
              : clientStatus?.connected === false ? t('mqtt_client_disconnected')
              : clientStatus?.configured ? t('mqtt_client_configured') : t('mqtt_client_not_configured')}
          </span>
        )}
      </div>
      <div className="grid gap-6 [grid-template-columns:minmax(0,1.45fr)_minmax(320px,0.95fr)]">
        <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
          <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('connection_label')}</p>
              <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('connection_params')}</h3>
              <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{t('broker_setup_hint')}</p>
            </div>
            <span className={cn('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1', isCoreComplete ? 'status-success' : 'status-warning')}>
              {isCoreComplete ? t('params_complete') : t('pending_fill')}
            </span>
          </div>
          <div className="p-6 grid grid-cols-2 gap-5">
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('config_name')}</label>
              <Input value={form.name} onChange={(e) => setF({ name: e.target.value })} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('broker_address')}</label>
              <Input value={form.broker} onChange={(e) => setF({ broker: e.target.value })} placeholder="mqtt://broker.example.com" />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('connection_type')}</label>
              <Select value={form.type} onValueChange={(v) => setF({ type: v })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="tcp">TCP</SelectItem>
                  <SelectItem value="websocket">WebSocket</SelectItem>
                  <SelectItem value="ssl">SSL/TLS</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('port')}</label>
              <Input type="number" value={form.port} onChange={(e) => setF({ port: Number(e.target.value) })} min={1} max={65535} />
            </div>
            <div className="grid gap-1.5 col-span-2">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('client_id_label')}</label>
              <Input value={form.client_id} onChange={(e) => setF({ client_id: e.target.value })} placeholder={t('client_id_ph')} />
              <p className="text-xs text-[var(--color-text-secondary)]">{t('client_id_help')}</p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
          <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('authentication')}</p>
              <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('auth_info')}</h3>
              <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{t('auth_info_desc')}</p>
            </div>
            <span className={cn('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1', hasCredentials ? 'status-success' : 'status-muted')}>
              {hasCredentials ? t('credentials_filled') : t('can_be_empty')}
            </span>
          </div>
          <div className="p-6 grid gap-5">
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('username')}</label>
              <Input value={form.username} onChange={(e) => setF({ username: e.target.value })} placeholder={t('username_no_auth')} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('password')}</label>
              <Input type="password" value={form.password} onChange={(e) => setF({ password: e.target.value })} placeholder={t('username_no_auth')} />
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

export const Route = createFileRoute('/_auth/_layout/admin/mqtt-config')({
  component: MqttConfigPage,
})
