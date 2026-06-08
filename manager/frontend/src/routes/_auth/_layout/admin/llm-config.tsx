import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'
import { ConfigListPage } from '@/components/admin/config-list-page'

function LLMConfigPage() {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <PageHeader title={t('llm_config')} />
      <ConfigListPage endpoint="/admin/llm-configs" addLabel={t('add_llm_config')} editLabel={t('edit_llm_config')} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/llm-config')({
  component: LLMConfigPage,
})
