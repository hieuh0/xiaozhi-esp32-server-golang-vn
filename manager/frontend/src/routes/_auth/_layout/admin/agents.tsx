import { useRef, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Plus, MoreHorizontal, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { agentsApi } from '@/features/agents/api/agents-api'
import type { Agent, AgentFormData } from '@/features/agents/types'
import { createDefaultAgentForm, agentToForm } from '@/features/agents/types'
import { AgentForm, type AgentFormHandle } from '@/components/agents/agent-form'
import { AgentRuntimeDiagnostics } from '@/components/agents/agent-runtime-diagnostics'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'
import { useEffect } from 'react'

type DiagPanel = 'mcp' | 'openclaw'

function AdminAgentsPage() {
  const { t } = useLocale()
  const formRef = useRef<AgentFormHandle>(null)

  const [agents, setAgents] = useState<Agent[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Agent | null>(null)
  const [form, setForm] = useState<AgentFormData>(createDefaultAgentForm({ isAdmin: true }))
  const [saving, setSaving] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState<Agent | null>(null)
  const [deleting, setDeleting] = useState(false)

  const [diagAgent, setDiagAgent] = useState<Agent | null>(null)
  const [diagPanel, setDiagPanel] = useState<DiagPanel>('mcp')

  const load = async (p = page) => {
    setLoading(true)
    try {
      const res = await agentsApi.getAdminAgents(p, pageSize)
      setAgents(res.items); setTotal(res.total)
    } catch { toast.error(t('load_agent_list_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load(1) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openAdd = () => { setEditing(null); setForm(createDefaultAgentForm({ isAdmin: true })); setShowForm(true) }
  const openEdit = (agent: Agent) => { setEditing(agent); setForm(agentToForm(agent, { isAdmin: true })); setShowForm(true) }

  const handleSave = async () => {
    const valid = await formRef.current?.validate()
    if (!valid) return
    setSaving(true)
    try {
      const payload = formRef.current!.buildPayload()
      if (editing) { await agentsApi.adminUpdateAgent(editing.id, payload); toast.success(t('agent_update_success')) }
      else { await agentsApi.createAgent(payload); toast.success(t('agent_add_success')) }
      setShowForm(false); await load(1)
    } catch (e) { toast.error((e as Error).message || t('save_failed')) }
    finally { setSaving(false) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await agentsApi.adminDeleteAgent(deleteTarget.id)
      toast.success(t('agent_delete_success'))
      setDeleteTarget(null); await load(1)
    } catch (e) { toast.error((e as Error).message || t('agent_delete_failed')) }
    finally { setDeleting(false) }
  }

  const columns: ColumnDef<Agent>[] = [
    { accessorKey: 'id', header: 'ID', size: 60 },
    { accessorKey: 'user_id', header: t('owner_user'), size: 80 },
    { accessorKey: 'name', header: t('agent_name') },
    { accessorKey: 'nickname', header: t('agent_nickname') },
    { accessorKey: 'memory_mode', header: t('memory_mode'), size: 100 },
    {
      accessorKey: 'created_at',
      header: t('created_at'),
      cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString(),
      size: 120,
    },
    {
      id: 'actions',
      header: '',
      size: 60,
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="w-8 h-8"><MoreHorizontal className="w-4 h-4" /></Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => openEdit(row.original)}>{t('edit')}</DropdownMenuItem>
            <DropdownMenuItem onClick={() => { setDiagAgent(row.original); setDiagPanel('mcp') }}>MCP</DropdownMenuItem>
            <DropdownMenuItem onClick={() => { setDiagAgent(row.original); setDiagPanel('openclaw') }}>OpenClaw</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-[var(--color-danger)]" onClick={() => setDeleteTarget(row.original)}>{t('delete')}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ]

  return (
    <div className="p-6 grid gap-5">
      <div className="flex items-center justify-between gap-4">
        <PageHeader eyebrow="ADMIN" title={t('agent_management')} />
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled={loading} onClick={() => load(1)}><RefreshCw className={`w-4 h-4 mr-1.5 ${loading ? 'animate-spin' : ''}`} />{t('refresh')}</Button>
          <Button size="sm" onClick={openAdd}><Plus className="w-4 h-4 mr-1.5" />{t('add_agent')}</Button>
        </div>
      </div>

      <DataTable data={agents} columns={columns} isLoading={loading} emptyMessage={t('no_agents')} pageSize={pageSize} />

      {total > pageSize && (
        <div className="flex items-center justify-center gap-3">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => { const p = page - 1; setPage(p); load(p) }}>{t('prev')}</Button>
          <span className="text-sm text-[var(--color-text-secondary)]">{page} / {Math.ceil(total / pageSize)}</span>
          <Button variant="outline" size="sm" disabled={page >= Math.ceil(total / pageSize)} onClick={() => { const p = page + 1; setPage(p); load(p) }}>{t('next')}</Button>
        </div>
      )}

      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent className="max-w-[580px] max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{editing ? t('edit_agent') : t('add_agent')}</DialogTitle></DialogHeader>
          <AgentForm ref={formRef} value={form} onChange={setForm} isAdmin mode={editing ? 'edit' : 'create'} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowForm(false)}>{t('cancel')}</Button>
            <Button disabled={saving} onClick={handleSave}>{saving ? t('saving') : t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!diagAgent} onOpenChange={(v) => { if (!v) setDiagAgent(null) }}>
        <DialogContent className="max-w-[580px]">
          <DialogHeader><DialogTitle>{diagAgent?.name} — {diagPanel === 'openclaw' ? 'OpenClaw' : 'MCP'}</DialogTitle></DialogHeader>
          {diagAgent && <AgentRuntimeDiagnostics agentId={diagAgent.id} scope="admin" />}
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting}
        title={t('confirm_delete')} description={t('confirm_delete_agent_msg', { name: deleteTarget?.name || '' })} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/agents')({
  component: AdminAgentsPage,
})
