import { useRef, useState, useEffect } from 'react'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { useSuspenseQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Settings, MessageSquare, Monitor, Trash2, Link2, Brain, BookOpen, Plug, Zap } from 'lucide-react'
import { Suspense } from 'react'
import { toast } from 'sonner'
import { agentsApi } from '@/features/agents/api/agents-api'
import type { Agent, AgentFormData, Role } from '@/features/agents/types'
import { createDefaultAgentForm } from '@/features/agents/types'
import { AgentForm, type AgentFormHandle } from '@/components/agents/agent-form'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

function memoryBadgeClass(mode: string) {
  if (mode === 'long') return 'status-success border text-xs'
  if (mode === 'short') return 'status-primary border text-xs'
  return 'status-muted border text-xs'
}

function hasMcp(agent: Agent): boolean {
  return (agent.mcp_service_names?.split(',').filter(Boolean).length ?? 0) > 0
}

function isOpenClawActive(agent: Agent): boolean {
  return !!(agent.openclaw?.allowed || (typeof agent.openclaw_config === 'string' && (() => {
    try { return !!JSON.parse(agent.openclaw_config!).allowed } catch { return false }
  })()))
}

function AgentCard({ agent, onEdit, onHistory, onDevices, onDelete }: { agent: Agent; onEdit: () => void; onHistory: () => void; onDevices: () => void; onDelete: () => void }) {
  const { t } = useLocale()
  const voice = agent.tts_config?.name && agent.voice ? `${agent.tts_config.name} · ${agent.voice}` : agent.tts_config?.name || agent.voice || t('not_set')
  const llm = agent.llm_config?.name || t('not_set')
  const kbCount = Array.isArray(agent.knowledge_base_ids) ? agent.knowledge_base_ids.length : 0

  return (
    <article className="p-5 rounded-xl bg-[var(--color-surface-1)] border border-[var(--color-line)] flex flex-col gap-3.5 max-w-[340px] w-full shadow-[var(--shadow-card)] transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[var(--shadow-card-hover)] hover:border-[var(--color-primary)]/30">
      <div className="flex items-center gap-3.5 min-w-0">
        <div
          className="w-12 h-12 rounded-xl flex-none inline-flex items-center justify-center"
          style={{
            background: 'linear-gradient(145deg, var(--color-primary-soft), color-mix(in srgb, var(--color-primary) 20%, transparent))',
            color: 'var(--color-primary)',
            border: '1px solid color-mix(in srgb, var(--color-primary) 25%, transparent)',
            boxShadow: 'var(--shadow-primary-glow)',
          }}
        >
          <Monitor className="w-5 h-5" />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="text-lg font-semibold text-[var(--color-text)] leading-snug mb-1">{agent.name}</h3>
          <p className="text-[13px] text-[var(--color-text-secondary)] truncate">{t('nickname_label')} {agent.nickname || agent.name}</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-1.5">
        <span className={cn('inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full font-medium', memoryBadgeClass(agent.memory_mode))}>
          <Brain className="w-3 h-3" />
          {agent.memory_mode === 'long' ? t('memory_long') : agent.memory_mode === 'short' ? t('memory_short') : t('memory_none')}
        </span>
        <span className={cn(
          'inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full font-medium border text-xs',
          kbCount > 0 ? 'status-primary' : 'status-muted'
        )}>
          <BookOpen className="w-3 h-3" />
          {kbCount > 0 ? `KB: ${kbCount}` : 'KB: —'}
        </span>
        {hasMcp(agent) && (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full font-medium border text-xs status-primary">
            <Plug className="w-3 h-3" />MCP
          </span>
        )}
        {isOpenClawActive(agent) && (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full font-medium border text-xs status-success">
            <Zap className="w-3 h-3" />OpenClaw
          </span>
        )}
      </div>

      <div className="grid gap-1.5">
        {[
          [t('timbre_model'), voice],
          [t('language_model_label'), llm],
        ].map(([lbl, val]) => (
          <div key={lbl} className="flex items-center gap-2 min-w-0 text-[13px]">
            <span className="flex-none text-[var(--color-text-secondary)]">{lbl}</span>
            <span className="flex-1 min-w-0 font-semibold text-[var(--color-text)] truncate">{String(val).length > 18 ? String(val).slice(0, 18) + '…' : val}</span>
          </div>
        ))}
      </div>
      <div className="grid grid-cols-2 gap-2 mt-auto">
        <Button variant="outline" size="sm" className="text-xs font-semibold rounded-md status-primary border" onClick={onEdit}>
          <Settings className="w-3.5 h-3.5 mr-1" />{t('config')}
        </Button>
        <Button variant="outline" size="sm" className="text-xs font-semibold rounded-md" onClick={onHistory}>
          <MessageSquare className="w-3.5 h-3.5 mr-1" />{t('chat')}
        </Button>
        <Button variant="outline" size="sm" className="text-xs font-semibold rounded-md" onClick={onDevices}>
          <Link2 className="w-3.5 h-3.5 mr-1" />{t('device')}
        </Button>
        <Button variant="outline" size="sm" className="text-xs font-semibold rounded-md status-danger border" onClick={onDelete}>
          <Trash2 className="w-3.5 h-3.5 mr-1" />{t('delete')}
        </Button>
      </div>
    </article>
  )
}

function AgentsInner() {
  const { t } = useLocale()
  const router = useRouter()
  const qc = useQueryClient()
  const formRef = useRef<AgentFormHandle>(null)

  const { data: agents } = useSuspenseQuery<Agent[]>({ queryKey: ['user-agents'], queryFn: agentsApi.getUserAgents, staleTime: 30_000 })

  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState<AgentFormData>(createDefaultAgentForm())
  const [adding, setAdding] = useState(false)
  const [roles, setRoles] = useState<Role[]>([])
  const [selectedRoleId, setSelectedRoleId] = useState<number | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Agent | null>(null)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    if (showAdd) {
      agentsApi.getRoles()
        .then(({ global_roles, user_roles }) =>
          setRoles([...global_roles, ...user_roles].filter((r) => !r.status || r.status === 'active'))
        )
        .catch(() => setRoles([]))
    }
  }, [showAdd]) // eslint-disable-line react-hooks/exhaustive-deps

  const applyRole = (role: Role) => {
    setSelectedRoleId(role.id)
    setForm((prev) => ({
      ...prev,
      custom_prompt: role.prompt || prev.custom_prompt,
      llm_config_id: role.llm_config_id || prev.llm_config_id,
      tts_config_id: role.tts_config_id || prev.tts_config_id,
      voice: role.voice || prev.voice,
    }))
    toast.info(t('role_config_applied'))
  }

  const refresh = () => qc.invalidateQueries({ queryKey: ['user-agents'] })

  const handleAdd = async () => {
    const valid = await formRef.current?.validate()
    if (!valid) return
    setAdding(true)
    try {
      await agentsApi.createAgent(formRef.current!.buildPayload())
      toast.success(t('agent_add_success'))
      setShowAdd(false); setForm(createDefaultAgentForm()); setSelectedRoleId(null)
      await refresh()
    } catch (e) { toast.error((e as Error).message || t('add_agent_failed')) }
    finally { setAdding(false) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await agentsApi.deleteAgent(deleteTarget.id)
      toast.success(t('agent_delete_success'))
      setDeleteTarget(null); await refresh()
    } catch (e) { toast.error((e as Error).message || t('agent_delete_failed')) }
    finally { setDeleting(false) }
  }

  return (
    <div className="grid gap-5 p-6">
      <section className="flex flex-wrap items-center justify-between gap-4 px-6 py-4 rounded-xl bg-[var(--color-surface-1)] border border-[var(--color-line)] shadow-sm">
        <div className="flex flex-wrap gap-2">
          <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold font-mono status-primary border">{t('agent')} {agents.length}</span>
        </div>
        <Button onClick={() => { setForm(createDefaultAgentForm()); setShowAdd(true) }}>
          <Plus className="w-4 h-4 mr-1.5" />{t('add_agent')}
        </Button>
      </section>

      {agents.length === 0 ? (
        <div className="flex justify-center">
          <div className="p-8 rounded-xl bg-[var(--color-surface-1)] border border-[var(--color-line)] text-center">
            <Monitor className="w-16 h-16 mx-auto text-[var(--color-primary)] mb-4" />
            <h3 className="text-2xl font-bold font-display tracking-tight text-[var(--color-text)] mb-2.5">{t('create_first_agent')}</h3>
            <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed mb-6">{t('post_create_agent_hint')}</p>
            <Button size="lg" onClick={() => { setForm(createDefaultAgentForm()); setShowAdd(true) }}>
              <Plus className="w-4 h-4 mr-2" />{t('create_agent_label')}
            </Button>
          </div>
        </div>
      ) : (
        <section className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(280px,340px))]">
          {agents.map((agent) => (
            <AgentCard key={agent.id} agent={agent}
              onEdit={() => router.navigate({ to: '/agents/$agentId/edit', params: { agentId: String(agent.id) } })}
              onHistory={() => router.navigate({ to: '/agents/$agentId/history', params: { agentId: String(agent.id) } })}
              onDevices={() => router.navigate({ to: '/agents' as never })}
              onDelete={() => setDeleteTarget(agent)}
            />
          ))}
        </section>
      )}

      <Dialog open={showAdd} onOpenChange={(v) => { if (!v) { setForm(createDefaultAgentForm()); setSelectedRoleId(null) } setShowAdd(v) }}>
        <DialogContent className="max-w-[560px] max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('add_agent')}</DialogTitle></DialogHeader>
          {roles.length > 0 && (
            <div className="flex items-center gap-2 overflow-x-auto pb-1 -mx-1 px-1">
              {roles.map((role) => (
                <button key={role.id} type="button" onClick={() => applyRole(role)}
                  className={cn(
                    'inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs border flex-none cursor-pointer transition-colors',
                    selectedRoleId === role.id
                      ? 'border-[var(--color-primary)] status-primary'
                      : 'border-[var(--color-line)] bg-[var(--color-surface-1)] text-[var(--color-text)] hover:border-[var(--color-primary)]'
                  )}>
                  <span>{role.name}</span>
                  <small className="text-[var(--color-text-secondary)]">{role.role_type === 'global' ? t('global') : t('mine')}</small>
                </button>
              ))}
            </div>
          )}
          <AgentForm ref={formRef} value={form} onChange={setForm} mode="create" />
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowAdd(false); setSelectedRoleId(null) }}>{t('cancel')}</Button>
            <Button disabled={adding} onClick={handleAdd}>{adding ? t('creating') : t('create_agent_label')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting}
        title={t('confirm_delete')} description={t('confirm_delete_agent_msg', { name: deleteTarget?.name || '' })} />
    </div>
  )
}

function AgentsSkeleton() {
  return (
    <div className="p-6 grid gap-5">
      <Skeleton className="h-16 rounded-xl" />
      <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(280px,340px))]">
        {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-52 rounded-xl" />)}
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/agents/')({
  component: () => (
    <Suspense fallback={<AgentsSkeleton />}>
      <AgentsInner />
    </Suspense>
  ),
})
