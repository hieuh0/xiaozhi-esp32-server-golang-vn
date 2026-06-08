import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { ConfigListPage } from '@/components/admin/config-list-page'

function VisionConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('vision_config')} />
      <ConfigListPage endpoint="/admin/vision-configs" addLabel={t('add_vision_config')} editLabel={t('edit_vision_config')} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/vision-config')({
  component: VisionConfigPage,
})
