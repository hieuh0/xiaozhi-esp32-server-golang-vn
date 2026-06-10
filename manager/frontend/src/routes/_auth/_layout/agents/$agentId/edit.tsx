import { useRef, useState } from 'react'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { toast } from 'sonner'
import { agentsApi } from '@/features/agents/api/agents-api'
import { agentToForm } from '@/features/agents/types'
import type { AgentFormData, Role } from '@/features/agents/types'
import { AgentForm, type AgentFormHandle } from '@/components/agents/agent-form'
import { AgentRuntimeDiagnostics } from '@/components/agents/agent-runtime-diagnostics'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useEffect } from 'react'
import { cn } from '@/lib/utils'

function AgentEditPage() {
  const { t } = useLocale()
  const router = useRouter()
  const { agentId } = Route.useParams()
  const formRef = useRef<AgentFormHandle>(null)

  const [form, setForm] = useState<AgentFormData | null>(null)
  const [loadingAgent, setLoadingAgent] = useState(true)
  const [saving, setSaving] = useState(false)
  const [roles, setRoles] = useState<Role[]>([])
  const [selectedRoleId, setSelectedRoleId] = useState<number | null>(null)
  const [applyingRole, setApplyingRole] = useState(false)

  useEffect(() => {
    Promise.all([
      agentsApi.getAgent(agentId).then((agent) => setForm(agentToForm(agent))).catch((e) => { toast.error(e.message || t('load_agent_config_failed')) }),
      agentsApi.getRoles().then(({ global_roles, user_roles }) => setRoles([...global_roles, ...user_roles].filter((r) => !r.status || r.status === 'active'))).catch(() => setRoles([])),
    ]).finally(() => setLoadingAgent(false))
  }, [agentId]) // eslint-disable-line react-hooks/exhaustive-deps

  const applyRole = async (role: Role) => {
    if (!form) return
    setApplyingRole(true)
    setSelectedRoleId(role.id)
    try {
      setForm((prev) => prev ? { ...prev, custom_prompt: role.prompt || prev.custom_prompt, llm_config_id: role.llm_config_id || prev.llm_config_id, tts_config_id: role.tts_config_id || prev.tts_config_id, voice: role.voice || prev.voice } : prev)
      toast.info(t('role_config_applied'))
    } finally { setApplyingRole(false) }
  }

  const handleSave = async () => {
    if (!form || applyingRole) { if (applyingRole) toast.info(t('filling_role_config')); return }
    const valid = await formRef.current?.validate()
    if (!valid) return
    setSaving(true)
    try {
      await agentsApi.updateAgent(agentId, formRef.current!.buildPayload())
      toast.success(t('save_success'))
      router.navigate({ to: '/agents' as never })
    } catch (e) { toast.error((e as Error).message || t('save_failed')) }
    finally { setSaving(false) }
  }

  return (
    <div className="min-h-full py-2 pb-6">
      <div className="max-w-[1120px] mx-auto mb-3 flex items-center justify-between gap-4">
        <div className="flex items-center gap-2.5 min-w-0">
          <Button variant="ghost" size="sm" onClick={() => router.navigate({ to: '/agents' as never })}>
            <ArrowLeft className="w-4 h-4 mr-1" />{t('back')}
          </Button>
          <h2 className="text-xl font-bold text-[var(--color-text)] truncate">{form?.name || t('edit_agent')}</h2>
        </div>
        <Button disabled={saving || loadingAgent} onClick={handleSave}>{t('save_config')}</Button>
      </div>

      <div className={cn('max-w-[1120px] mx-auto mb-3 min-h-[42px] flex items-center gap-2 overflow-x-auto pb-1', !roles.length && 'opacity-60')}>
        {roles.map((role) => (
          <button key={role.id} type="button" onClick={() => applyRole(role)}
            className={cn('inline-flex items-center gap-2 px-2.5 py-2 rounded-lg text-xs border flex-none cursor-pointer transition-colors', selectedRoleId === role.id ? 'border-[var(--color-primary)] status-primary' : 'border-[var(--color-line)] bg-[var(--color-surface-1)] text-[var(--color-text)] hover:border-[var(--color-primary)]')}>
            <span>{role.name}</span>
            <small className="text-[var(--color-text-secondary)]">{role.role_type === 'global' ? t('global') : t('mine')}</small>
          </button>
        ))}
        {!roles.length && <span className="text-sm text-[var(--color-text-secondary)]">{t('no_roles_available')}</span>}
      </div>

      <div className={cn('max-w-[1120px] mx-auto p-5 border border-[var(--color-line)] rounded-xl bg-[var(--color-surface-1)]', loadingAgent && 'opacity-60')}>
        {loadingAgent || !form ? (
          <div className="grid gap-4">{Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}</div>
        ) : (
          <AgentForm ref={formRef} value={form} onChange={setForm} mode="edit" />
        )}
      </div>

      <div className="max-w-[1120px] mx-auto mt-3 p-5 border border-[var(--color-line)] rounded-xl bg-[var(--color-surface-1)]">
        <AgentRuntimeDiagnostics agentId={agentId} scope="user" />
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/agents/$agentId/edit')({
  component: AgentEditPage,
})
