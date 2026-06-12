import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { PageHeader } from '@/components/ui/page-header'

interface EnvMqtt { enable: boolean; endpoint: string }
interface EnvConfig { websocket: { url: string }; mqtt: EnvMqtt }
interface OtaForm { signature_key: string; firmware_version: string; firmware_url: string; test: EnvConfig; external: EnvConfig }

const defaults: OtaForm = {
  signature_key: 'xiaozhi_ota_signature_key',
  firmware_version: '',
  firmware_url: '',
  test: { websocket: { url: 'ws://127.0.0.1:8989/xiaozhi/v1/' }, mqtt: { enable: true, endpoint: '127.0.0.1:1883' } },
  external: { websocket: { url: 'ws://127.0.0.1:8989/xiaozhi/v1/' }, mqtt: { enable: false, endpoint: '127.0.0.1:1883' } }
}

function OtaConfigPage() {
  const { t } = useLocale()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState<'test' | 'external' | null>(null)
  const [configId, setConfigId] = useState<number | null>(null)
  const [form, setForm] = useState<OtaForm>(defaults)

  const setSig = (v: string) => setForm((f) => ({ ...f, signature_key: v }))
  const setTest = (patch: Partial<EnvConfig>) => setForm((f) => ({ ...f, test: { ...f.test, ...patch } }))
  const setExt = (patch: Partial<EnvConfig>) => setForm((f) => ({ ...f, external: { ...f.external, ...patch } }))
  const setTestMqtt = (patch: Partial<EnvMqtt>) => setTest({ mqtt: { ...form.test.mqtt, ...patch } })
  const setExtMqtt = (patch: Partial<EnvMqtt>) => setExt({ mqtt: { ...form.external.mqtt, ...patch } })

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/ota-configs')
      const configs = res.data?.data || []
      if (configs.length > 0) {
        const c = configs[0]; setConfigId(c.id)
        const d = JSON.parse(c.json_data || '{}')
        setForm({
          signature_key: d.signature_key || 'xiaozhi_ota_signature_key',
          firmware_version: d.firmware_version || '',
          firmware_url: d.firmware_url || '',
          test: { websocket: { url: d.test?.websocket?.url || 'ws://127.0.0.1:8989/xiaozhi/v1/' }, mqtt: { enable: d.test?.mqtt?.enable !== undefined ? d.test.mqtt.enable : true, endpoint: d.test?.mqtt?.endpoint || '127.0.0.1:1883' } },
          external: { websocket: { url: d.external?.websocket?.url || 'ws://127.0.0.1:8989/xiaozhi/v1/' }, mqtt: { enable: d.external?.mqtt?.enable !== undefined ? d.external.mqtt.enable : false, endpoint: d.external?.mqtt?.endpoint || '127.0.0.1:1883' } }
        })
      } else { setConfigId(null); setForm(defaults) }
    } catch { toast.error(t('load_ota_config_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSave = async () => {
    if (!form.signature_key.trim()) { toast.error(t('enter_signature_key')); return }
    setSaving(true)
    try {
      const payload = { name: t('ota_config_label'), config_id: 'ota_ota_config', enabled: true, is_default: true, json_data: JSON.stringify({ signature_key: form.signature_key.trim(), firmware_version: form.firmware_version.trim(), firmware_url: form.firmware_url.trim(), test: { websocket: { url: form.test.websocket.url.trim() }, mqtt: { enable: form.test.mqtt.enable, endpoint: form.test.mqtt.endpoint.trim() } }, external: { websocket: { url: form.external.websocket.url.trim() }, mqtt: { enable: form.external.mqtt.enable, endpoint: form.external.mqtt.endpoint.trim() } } }) }
      if (configId) { await api.put(`/admin/ota-configs/${configId}`, payload); toast.success(t('ota_config_updated')) }
      else { const r = await api.post('/admin/ota-configs', payload); setConfigId(r.data?.data?.id || null); toast.success(t('ota_config_saved')) }
      await load()
    } catch (e) { toast.error((e as Error).message || t('save_ota_failed')) }
    finally { setSaving(false) }
  }

  const testEnv = async (env: 'test' | 'external') => {
    setTesting(env)
    try {
      const payload = { signature_key: form.signature_key.trim(), test: { websocket: { url: form.test.websocket.url.trim() }, mqtt: { enable: form.test.mqtt.enable, endpoint: form.test.mqtt.endpoint.trim() } }, external: { websocket: { url: form.external.websocket.url.trim() }, mqtt: { enable: form.external.mqtt.enable, endpoint: form.external.mqtt.endpoint.trim() } } }
      const res = await api.post('/admin/configs/test', { types: ['ota'], data: { ota: { ota_ota_config: payload } } }, { timeout: 30000 })
      const d = res.data?.data ?? res.data
      const otaResult = d?.ota?.ota_ota_config
      if (!otaResult) { toast.error(t('label_no_result', { label: env })); return }
      const wsResult = otaResult.websocket || {}
      const wsOk = wsResult.ok || false
      const wsMsg = wsResult.message || ''
      const wsMs = wsResult.first_packet_ms
      const msg = `WebSocket: ${wsMsg}${wsMs != null ? ` (${wsMs}ms)` : ''}`
      if (wsOk) toast.success(`${env}: ${msg}`)
      else toast.warning(`${env}: ${msg}`)
    } catch (e) { toast.error((e as Error).message || t('test_request_failed_v2')) }
    finally { setTesting(null) }
  }

  if (loading) return <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>

  const EnvCard = ({ label, badge, env, wsUrl, setWsUrl, mqttEnabled, setMqttEnabled, mqttEndpoint, setMqttEndpoint, isTest }: { label: string; badge: React.ReactNode; env: 'test' | 'external'; wsUrl: string; setWsUrl: (v: string) => void; mqttEnabled: boolean; setMqttEnabled: (v: boolean) => void; mqttEndpoint: string; setMqttEndpoint: (v: string) => void; isTest?: boolean }) => (
    <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
      <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
        <div>
          <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{isTest ? 'Test' : 'External'}</p>
          <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{label}</h3>
        </div>
        <div className="flex items-center gap-2 shrink-0 mt-1">
          {badge}
          <Button size="sm" variant="outline" disabled={testing === env} onClick={() => testEnv(env)}>{testing === env ? '...' : t('test_env_tag')}</Button>
        </div>
      </div>
      <div className="p-6 grid gap-6">
        <div className="grid gap-3">
          <p className="text-sm font-bold text-[var(--color-text)]">{t('ws_delivery')}</p>
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">WebSocket URL</label>
            <Input value={wsUrl} onChange={(e) => setWsUrl(e.target.value)} placeholder="ws://..." />
          </div>
        </div>
        <div className="grid gap-3 pt-6 border-t border-[var(--color-line)]">
          <p className="text-sm font-bold text-[var(--color-text)]">{t('mqtt_delivery')}</p>
          <div className="flex items-center justify-between gap-4">
            <p className="text-sm font-semibold text-[var(--color-text)]">{t('priority_mqtt')}</p>
            <Switch checked={mqttEnabled} onCheckedChange={setMqttEnabled} />
          </div>
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('mqtt_endpoint')}</label>
            <Input disabled={!mqttEnabled} value={mqttEndpoint} onChange={(e) => setMqttEndpoint(e.target.value)} placeholder="127.0.0.1:1883" />
          </div>
        </div>
      </div>
    </div>
  )

  return (
    <div className="grid gap-6 px-6 pb-8">
      <PageHeader title={t('ota_config')} />

      <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
        <div className="px-6 pt-6 pb-0">
          <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">OTA Base</p>
          <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('signature_constraints')}</h3>
        </div>
        <div className="p-6">
          <div className="grid gap-1.5 max-w-lg">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('signature_key')}</label>
            <Input type="password" value={form.signature_key} onChange={(e) => setSig(e.target.value)} />
            <p className="text-xs text-[var(--color-text-secondary)]">{t('ota_sig_key_hint')}</p>
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
        <div className="px-6 pt-6 pb-0">
          <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">Firmware</p>
          <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('firmware_update')}</h3>
        </div>
        <div className="p-6 grid gap-4 max-w-lg">
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('firmware_version')}</label>
            <Input value={form.firmware_version} onChange={(e) => setForm(f => ({ ...f, firmware_version: e.target.value }))} placeholder="1.0.0" />
          </div>
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('firmware_url')}</label>
            <Input value={form.firmware_url} onChange={(e) => setForm(f => ({ ...f, firmware_url: e.target.value }))} placeholder="https://..." />
            <p className="text-xs text-[var(--color-text-secondary)]">{t('firmware_url_hint')}</p>
          </div>
        </div>
      </div>

      <div className="grid gap-6 grid-cols-2">
        <EnvCard
          label={t('test_env_delivery')} env="test" isTest
          badge={<span className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold status-warning">{t('test_env_tag')}</span>}
          wsUrl={form.test.websocket.url} setWsUrl={(v) => setTest({ websocket: { url: v } })}
          mqttEnabled={form.test.mqtt.enable} setMqttEnabled={(v) => setTestMqtt({ enable: v })}
          mqttEndpoint={form.test.mqtt.endpoint} setMqttEndpoint={(v) => setTestMqtt({ endpoint: v })}
        />
        <EnvCard
          label={t('external_env_delivery')} env="external"
          badge={<span className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold status-success">{t('prod_env_tag')}</span>}
          wsUrl={form.external.websocket.url} setWsUrl={(v) => setExt({ websocket: { url: v } })}
          mqttEnabled={form.external.mqtt.enable} setMqttEnabled={(v) => setExtMqtt({ enable: v })}
          mqttEndpoint={form.external.mqtt.endpoint} setMqttEndpoint={(v) => setExtMqtt({ endpoint: v })}
        />
      </div>

      <div className="flex items-center justify-end gap-3">
        <Button variant="outline" disabled={loading} onClick={load}>{t('reset_to_current')}</Button>
        <Button disabled={saving} onClick={handleSave}>{t('save_config')}</Button>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/ota-config')({
  component: OtaConfigPage,
})
