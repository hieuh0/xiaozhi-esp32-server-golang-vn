import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { MoreHorizontal, Plus, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import type { ColumnDef } from '@tanstack/react-table'
import { devicesApi } from '@/features/devices/api/devices-api'
import type { Device, DeviceFormData } from '@/features/devices/types'
import { createDefaultDeviceForm, getDeviceDisplayName, isDeviceOnline } from '@/features/devices/types'
import { DeviceForm } from '@/components/devices/device-form'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'
import { cn } from '@/lib/utils'
import type { McpTool } from '@/features/devices/types'

function AdminDevicesPage() {
  const { t } = useLocale()

  const [devices, setDevices] = useState<Device[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Device | null>(null)
  const [form, setForm] = useState<DeviceFormData>(createDefaultDeviceForm())
  const [saving, setSaving] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState<Device | null>(null)
  const [deleting, setDeleting] = useState(false)

  const [showMcpDialog, setShowMcpDialog] = useState(false)
  const [mcpLoading, setMcpLoading] = useState(false)
  const [toolsLoading, setToolsLoading] = useState(false)
  const [callingTool, setCallingTool] = useState(false)
  const [currentDeviceId, setCurrentDeviceId] = useState<number | null>(null)
  const [mcpTools, setMcpTools] = useState<McpTool[]>([])
  const [mcpCallResult, setMcpCallResult] = useState('')
  const [mcpToolName, setMcpToolName] = useState('')
  const [mcpToolArgs, setMcpToolArgs] = useState('{}')

  const load = async (p = page) => {
    setLoading(true)
    try {
      const res = await devicesApi.getAdminDevices(p, pageSize)
      setDevices(res.items); setTotal(res.total)
    } catch { toast.error(t('load_device_list_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load(1) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openAdd = () => { setEditing(null); setForm(createDefaultDeviceForm()); setShowForm(true) }
  const openEdit = (d: Device) => { setEditing(d); setForm({ agent_id: d.agent_id || null, identifier: '', nick_name: d.nick_name || '', device_name: d.device_name, device_code: d.device_code }); setShowForm(true) }

  const handleSave = async () => {
    setSaving(true)
    try {
      if (editing) {
        await devicesApi.adminUpdateDevice(editing.id, { nick_name: form.nick_name, agent_id: form.agent_id })
        toast.success(t('device_update_success'))
      } else {
        if (!form.agent_id) { toast.error(t('select_target_agent')); setSaving(false); return }
        if (!form.identifier?.trim()) { toast.error(t('enter_6digit_or_mac')); setSaving(false); return }
        await devicesApi.addDevice(form.agent_id, { identifier: form.identifier.trim(), nick_name: form.nick_name.trim() || undefined })
        toast.success(t('device_bind_success'))
      }
      setShowForm(false); await load(page)
    } catch (e) { toast.error((e as Error).message || t('update_failed')) }
    finally { setSaving(false) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await devicesApi.adminDeleteDevice(deleteTarget.id)
      toast.success(t('device_delete_success'))
      setDeleteTarget(null); await load(page)
    } catch (e) { toast.error((e as Error).message || t('device_delete_failed')) }
    finally { setDeleting(false) }
  }

  const openMcp = async (device: Device) => {
    setCurrentDeviceId(device.id); setMcpCallResult(''); setMcpToolName(''); setMcpToolArgs('{}')
    setShowMcpDialog(true); setMcpLoading(true)
    try {
      const tools = await devicesApi.getDeviceMcpTools(device.id)
      setMcpTools(tools)
      if (tools.length) setMcpToolName(tools[0].name)
    } catch { toast.error(t('fetch_device_mcp_failed')) }
    finally { setMcpLoading(false) }
  }

  const refreshMcp = async () => {
    if (!currentDeviceId) return
    setToolsLoading(true)
    try {
      const tools = await devicesApi.getDeviceMcpTools(currentDeviceId)
      setMcpTools(tools)
    } catch { toast.error(t('fetch_device_mcp_failed')) }
    finally { setToolsLoading(false) }
  }

  const callMcp = async () => {
    if (!currentDeviceId || !mcpToolName) { toast.error(t('select_tool')); return }
    let args: Record<string, unknown> = {}
    try { args = JSON.parse(mcpToolArgs) } catch { toast.error(t('mcp_config_format_error')); return }
    setCallingTool(true)
    try {
      const result = await devicesApi.callDeviceMcpTool(currentDeviceId, mcpToolName, args)
      setMcpCallResult(typeof result === 'string' ? result : JSON.stringify(result, null, 2))
      toast.success(t('mcp_tool_call_success'))
    } catch (e) { setMcpCallResult((e as Error).message); toast.error(t('mcp_tool_call_failed')) }
    finally { setCallingTool(false) }
  }

  const totalPages = Math.ceil(total / pageSize) || 1

  const badgeClass = (active: boolean) => cn(
    'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium',
    active ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
           : 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
  )

  const columns: ColumnDef<Device>[] = [
    { accessorKey: 'id', header: 'ID', cell: ({ row }) => <span className="text-xs font-mono text-[var(--color-text-secondary)]">{row.original.id}</span> },
    { accessorKey: 'nick_name', header: t('device_nickname'), cell: ({ row }) => <span className="font-bold">{getDeviceDisplayName(row.original)}</span> },
    { accessorKey: 'device_code', header: t('activation_code'), cell: ({ row }) => <span className="text-xs">{row.original.device_code || '—'}</span> },
    { accessorKey: 'device_name', header: t('device_id'), cell: ({ row }) => <span className="font-mono text-xs text-[var(--color-text-secondary)]">{row.original.device_name || '—'}</span> },
    { accessorKey: 'user_id', header: t('user_prefix') + 'ID' },
    {
      accessorKey: 'agent_id', header: t('link_agent'),
      cell: ({ row }) => row.original.agent_id
        ? <span className="text-sm">{row.original.agent_name || `#${row.original.agent_id}`}</span>
        : <span className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-500 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700">{t('no_agent_linked')}</span>
    },
    {
      accessorKey: 'activated', header: t('activation_status'),
      cell: ({ row }) => <span className={cn('inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium', row.original.activated ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800' : 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800')}>{row.original.activated ? t('activated') : t('not_activated')}</span>
    },
    {
      accessorKey: 'last_active_at', header: t('online_devices'),
      cell: ({ row }) => <span className={badgeClass(isDeviceOnline(row.original.last_active_at))}>{isDeviceOnline(row.original.last_active_at) ? t('online') : t('offline')}</span>
    },
    { accessorKey: 'last_active_at_fmt', header: t('latest_data'), cell: ({ row }) => <span className="text-sm text-[var(--color-text-secondary)]">{row.original.last_active_at ? new Date(row.original.last_active_at).toLocaleString() : t('never_active')}</span> },
    { accessorKey: 'created_at', header: t('start_date'), cell: ({ row }) => <span className="text-sm text-[var(--color-text-secondary)]">{new Date(row.original.created_at).toLocaleString()}</span> },
    {
      id: 'actions', header: () => <span className="sr-only">{t('actions')}</span>,
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-8 w-8"><MoreHorizontal className="h-4 w-4" /></Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => openEdit(row.original)}>{t('edit')}</DropdownMenuItem>
            <DropdownMenuItem onClick={() => openMcp(row.original)}>MCP</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(row.original)}>{t('delete')}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )
    },
  ]

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('device_management')} />

      <div className="flex items-center justify-end gap-2">
        <Button variant="outline" onClick={() => load(page)}>
          <RefreshCw className="w-4 h-4 mr-1.5" />{t('refresh')}
        </Button>
        <Button onClick={openAdd}>
          <Plus className="w-4 h-4 mr-1.5" />{t('add_device')}
        </Button>
      </div>

      <DataTable columns={columns} data={devices} isLoading={loading} />

      <div className="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
        <span>{total} {t('total_items')}</span>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => { setPage(page - 1); load(page - 1) }}>{t('prev')}</Button>
          <span>{page} / {totalPages}</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => { setPage(page + 1); load(page + 1) }}>{t('next')}</Button>
        </div>
      </div>

      {/* Add/Edit dialog */}
      <Dialog open={showForm} onOpenChange={(v) => { if (!v) setShowForm(false) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{editing ? t('edit') : t('add_device')}</DialogTitle></DialogHeader>
          <DeviceForm value={form} onChange={setForm} agents={[]} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowForm(false)}>{t('cancel')}</Button>
            <Button disabled={saving} onClick={handleSave}>{editing ? t('save') : t('add')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* MCP tools dialog */}
      <Dialog open={showMcpDialog} onOpenChange={setShowMcpDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader><DialogTitle>{t('device_mcp_tools')}</DialogTitle></DialogHeader>
          {mcpLoading ? (
            <div className="py-8 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>
          ) : (
            <div className="grid gap-4">
              <div className="flex items-center justify-between">
                <div className="flex flex-wrap gap-1.5">
                  {mcpTools.length === 0 ? (
                    <span className="text-sm text-[var(--color-text-secondary)]">{t('no_tool_data')}</span>
                  ) : mcpTools.map((tool) => (
                    <span key={tool.name} className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-[var(--color-surface-2)] text-[var(--color-text-secondary)] border-[var(--color-line)]">{tool.name}</span>
                  ))}
                </div>
                <Button size="sm" variant="outline" disabled={toolsLoading} onClick={refreshMcp}>{t('refresh')}</Button>
              </div>
              <div className="border-t border-[var(--color-line)]" />
              <div className="grid gap-3">
                <div className="grid gap-1.5">
                  <label className="text-sm font-semibold text-[var(--color-text)]">{t('tool_name')}</label>
                  <Select value={mcpToolName} onValueChange={setMcpToolName}>
                    <SelectTrigger><SelectValue placeholder={t('select_tool')} /></SelectTrigger>
                    <SelectContent>
                      {mcpTools.map((tool) => <SelectItem key={tool.name} value={tool.name}>{tool.name}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1.5">
                  <label className="text-sm font-semibold text-[var(--color-text)]">{t('args_json_label')}</label>
                  <Textarea value={mcpToolArgs} onChange={(e) => setMcpToolArgs(e.target.value)} rows={6} placeholder={t('mcp_args_placeholder')} className="font-mono text-sm resize-y" />
                </div>
                <Button disabled={callingTool} onClick={callMcp}>{t('call_device_tool')}</Button>
              </div>
              <div className="border-t border-[var(--color-line)]" />
              <pre className="whitespace-pre-wrap font-mono text-xs bg-[var(--color-surface-2)] border border-[var(--color-line)] rounded-lg p-3 min-h-[80px] max-h-[200px] overflow-auto">
                {mcpCallResult || t('no_call_results')}
              </pre>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        isLoading={deleting}
        title={t('confirm_delete')}
        description={t('confirm_delete_device_msg', { name: getDeviceDisplayName(deleteTarget!) })}
      />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/devices')({
  component: AdminDevicesPage,
})
