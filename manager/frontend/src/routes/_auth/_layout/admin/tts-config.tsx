import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { ConfigListPage } from '@/components/admin/config-list-page'

function TTSConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('tts_config')} />
      <ConfigListPage endpoint="/admin/tts-configs" addLabel={t('add_tts_config')} editLabel={t('edit_tts_config')} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/tts-config')({
  component: TTSConfigPage,
})
