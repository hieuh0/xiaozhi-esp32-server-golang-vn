import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { ConfigListPage } from '@/components/admin/config-list-page'

function KnowledgeSearchConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('knowledge_search_config')} />
      <ConfigListPage endpoint="/admin/knowledge-search-configs" addLabel={t('add_knowledge_config')} editLabel={t('edit_knowledge_config')} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/knowledge-search-config')({
  component: KnowledgeSearchConfigPage,
})
