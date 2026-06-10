import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { ConfigListPage } from '@/components/admin/config-list-page'
import { AsrConfigForm } from '@/components/admin/asr-config-form'

function ASRConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('asr_config')} />
      <ConfigListPage
        endpoint="/admin/asr-configs"
        addLabel={t('add_asr_config')}
        editLabel={t('edit_asr_config')}
        renderForm={(props) => <AsrConfigForm {...props} />}
      />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/asr-config')({
  component: ASRConfigPage,
})
