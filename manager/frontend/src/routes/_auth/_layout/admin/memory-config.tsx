import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { ConfigListPage } from '@/components/admin/config-list-page'

function MemoryConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('memory_config')} />
      <ConfigListPage endpoint="/admin/memory-configs" addLabel={t('add_memory_config')} editLabel={t('edit_memory_config')} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/memory-config')({
  component: MemoryConfigPage,
})
