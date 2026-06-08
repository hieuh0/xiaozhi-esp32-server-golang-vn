import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PageHeader } from '@/components/ui/page-header'
import { cn } from '@/lib/utils'

interface UdpForm {
  name: string
  listen_host: string
  listen_port: number
  external_host: string
  external_port: number
}

const defaults: UdpForm = { name: 'UDP Config', listen_host: '0.0.0.0', listen_port: 8990, external_host: '192.168.0.1', external_port: 8990 }

function UdpConfigPage() {
  const { t } = useLocale()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [configId, setConfigId] = useState<number | null>(null)
  const [form, setForm] = useState<UdpForm>(defaults)

  const setF = (patch: Partial<UdpForm>) => setForm((f) => ({ ...f, ...patch }))

  const listenReady = Boolean(form.listen_host.trim() && form.listen_port)
  const externalReady = Boolean(form.external_host.trim() && form.external_port)

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/udp-configs')
      const configs = res.data?.data || []
      if (configs.length > 0) {
        const c = configs[0]; setConfigId(c.id)
        const d = JSON.parse(c.json_data || '{}')
        setForm({ name: c.name || 'UDP Config', listen_host: d.listen_host || '0.0.0.0', listen_port: Number(d.listen_port) || 8990, external_host: d.external_host || '192.168.0.1', external_port: Number(d.external_port) || 8990 })
      } else { setConfigId(null); setForm(defaults) }
    } catch { toast.error(t('load_udp_config_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSave = async () => {
    if (!form.listen_host.trim()) { toast.error(t('enter_listen_host')); return }
    if (!form.external_host.trim()) { toast.error(t('enter_external_host')); return }
    setSaving(true)
    try {
      const payload = {
        name: form.name,
        config_id: `udp_${form.name.replace(/[^a-zA-Z0-9]/g, '_').toLowerCase()}`,
        is_default: true,
        json_data: JSON.stringify({ listen_host: form.listen_host.trim(), listen_port: Number(form.listen_port), external_host: form.external_host.trim(), external_port: Number(form.external_port) })
      }
      if (configId) { await api.put(`/admin/udp-configs/${configId}`, payload); toast.success(t('udp_config_updated')) }
      else { const r = await api.post('/admin/udp-configs', payload); setConfigId(r.data?.data?.id || null); toast.success(t('udp_config_saved')) }
      await load()
    } catch (e) { toast.error((e as Error).message || t('save_udp_failed')) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>

  const badgeClass = (ok: boolean) => cn('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1', ok ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800' : 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800')

  return (
    <div className="grid gap-6 px-6 pb-8">
      <PageHeader title={t('udp_config')} />
      <div className="grid gap-6 [grid-template-columns:minmax(0,1.4fr)_minmax(320px,0.9fr)]">
        <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
          <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">UDP Server</p>
              <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('service_listen')}</h3>
              <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{t('udp_server_config_desc')}</p>
            </div>
            <span className={badgeClass(listenReady)}>{listenReady ? t('listen_params_complete') : t('pending_fill')}</span>
          </div>
          <div className="p-6 grid grid-cols-2 gap-5">
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('config_name')}</label>
              <Input value={form.name} onChange={(e) => setF({ name: e.target.value })} />
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

        <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)]">
          <div className="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('announce_address')}</p>
              <h3 className="mt-2 text-xl font-semibold text-[var(--color-text)]">{t('terminal_publish_addr')}</h3>
              <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{t('terminal_publish_desc')}</p>
            </div>
            <span className={badgeClass(externalReady)}>{externalReady ? t('address_complete') : t('pending_fill')}</span>
          </div>
          <div className="p-6 grid gap-5">
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('external_host')}</label>
              <Input value={form.external_host} onChange={(e) => setF({ external_host: e.target.value })} placeholder="192.168.0.1" />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold text-[var(--color-text)]">{t('external_port')}</label>
              <Input type="number" value={form.external_port} onChange={(e) => setF({ external_port: Number(e.target.value) })} min={1} max={65535} />
              <p className="text-xs text-[var(--color-text-secondary)]">{t('terminal_addr_note')}</p>
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

export const Route = createFileRoute('/_auth/_layout/admin/udp-config')({
  component: UdpConfigPage,
})
