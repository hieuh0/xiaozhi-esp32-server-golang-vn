import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ConfigListPage } from '@/components/admin/config-list-page'
import { VisionConfigForm } from '@/components/admin/vision-config-form'

function VisionBaseConfigCard() {
  const { t } = useLocale()
  const [enableAuth, setEnableAuth] = useState(false)
  const [visionUrl, setVisionUrl] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.get('/admin/vision-base-config').then(res => {
      const d = res.data.data || {}
      setEnableAuth(!!d.enable_auth)
      setVisionUrl(d.vision_url || '')
    }).catch(() => {})
  }, [])

  const handleSave = async () => {
    if (!visionUrl.trim()) { toast.error(t('enter_vision_url')); return }
    setSaving(true)
    try {
      await api.put('/admin/vision-base-config', { enable_auth: enableAuth, vision_url: visionUrl })
      toast.success(t('basic_config_save_success'))
    } catch { toast.error(t('save_failed_check_network')) }
    finally { setSaving(false) }
  }

  return (
    <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
      <div className="px-6 py-4 border-b border-[var(--color-line)]">
        <p className="text-sm font-semibold text-[var(--color-text)]">{t('basic_config')}</p>
      </div>
      <div className="p-6 grid gap-5 max-w-[600px]">
        <div className="flex items-center justify-between gap-4">
          <label className="text-sm font-semibold text-[var(--color-text)]">{t('enable_auth')}</label>
          <Switch checked={enableAuth} onCheckedChange={setEnableAuth} />
        </div>
        <div className="grid gap-1.5">
          <label className="text-sm font-semibold text-[var(--color-text)]">Vision URL</label>
          <Input value={visionUrl} onChange={e => setVisionUrl(e.target.value)} placeholder={t('vision_url_ph')} />
          <p className="text-xs text-[var(--color-text-secondary)]">{t('vision_url_hint')}</p>
        </div>
        <div>
          <Button disabled={saving} onClick={handleSave}>{t('save_basic_config')}</Button>
        </div>
      </div>
    </div>
  )
}

function VisionConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('vision_config')} />
      <VisionBaseConfigCard />
      <ConfigListPage
        endpoint="/admin/vision-configs"
        addLabel={t('add_vision_config')}
        editLabel={t('edit_vision_config')}
        renderForm={(props) => <VisionConfigForm {...props} />}
      />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/vision-config')({
  component: VisionConfigPage,
})
