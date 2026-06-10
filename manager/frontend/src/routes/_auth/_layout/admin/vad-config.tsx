import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { ConfigListPage } from '@/components/admin/config-list-page'
import { VadConfigForm } from '@/components/admin/vad-config-form'

function VADConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('vad_config')} />
      <ConfigListPage
        endpoint="/admin/vad-configs"
        addLabel={t('add_vad_config')}
        editLabel={t('edit_vad_config')}
        renderForm={(props) => <VadConfigForm {...props} />}
      />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/vad-config')({
  component: VADConfigPage,
})
