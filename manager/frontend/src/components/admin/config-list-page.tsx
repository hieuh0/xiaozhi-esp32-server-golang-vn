import { useEffect, useState } from 'react'
import { MoreHorizontal, Plus, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'

export interface ConfigRow {
  id: number
  name: string
  config_id: string
  provider?: string
  enabled?: boolean
  is_default?: boolean
  created_at: string
  json_data?: string
}

interface ConfigForm {
  name: string
  config_id: string
  provider: string
  enabled: boolean
  is_default: boolean
  json_data: string
}

const defaultForm = (): ConfigForm => ({
  name: '', config_id: '', provider: '', enabled: true, is_default: false, json_data: '{}'
})

interface Props {
  endpoint: string
  addLabel?: string
  editLabel?: string
  extraColumns?: Array<{ key: string; header: string; render?: (row: ConfigRow) => React.ReactNode }>
}

export function ConfigListPage({ endpoint, addLabel, editLabel, extraColumns = [] }: Props) {
  const { t } = useLocale()

  const [rows, setRows] = useState<ConfigRow[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const [showDialog, setShowDialog] = useState(false)
  const [editing, setEditing] = useState<ConfigRow | null>(null)
  const [form, setForm] = useState<ConfigForm>(defaultForm())
  const [saving, setSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<ConfigRow | null>(null)
  const [deleting, setDeleting] = useState(false)

  const setF = (patch: Partial<ConfigForm>) => setForm((f) => ({ ...f, ...patch }))

  const load = async (p = page) => {
    setLoading(true)
    try {
      const res = await api.get(endpoint, { params: { page: p, page_size: pageSize } })
      setRows(res.data.data || [])
      setTotal(res.data.total || 0)
    } catch { toast.error(t('load_config_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load(1) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openAdd = () => { setEditing(null); setForm(defaultForm()); setShowDialog(true) }
  const openEdit = (row: ConfigRow) => {
    setEditing(row)
    setForm({
      name: row.name, config_id: row.config_id, provider: row.provider || '',
      enabled: row.enabled !== false, is_default: !!row.is_default, json_data: row.json_data || '{}'
    })
    setShowDialog(true)
  }

  const handleSave = async () => {
    if (!form.name.trim()) { toast.error(t('enter_config_name')); return }
    if (!form.config_id.trim()) { toast.error(t('enter_config_id')); return }
    setSaving(true)
    try {
      const payload = { name: form.name, config_id: form.config_id, provider: form.provider, enabled: form.enabled, is_default: form.is_default, json_data: form.json_data }
      if (editing) {
        await api.put(`${endpoint}/${editing.id}`, payload)
        toast.success(t('config_update_success'))
      } else {
        await api.post(endpoint, payload)
        toast.success(t('config_create_success'))
      }
      setShowDialog(false); await load(page)
    } catch (e) { toast.error((e as Error).message || t('save_failed')) }
    finally { setSaving(false) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await api.delete(`${endpoint}/${deleteTarget.id}`)
      toast.success(t('delete_success'))
      setDeleteTarget(null); await load(page)
    } catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  const toggleEnabled = async (row: ConfigRow, val: boolean) => {
    setRows((prev) => prev.map((r) => r.id === row.id ? { ...r, enabled: val } : r))
    try { await api.put(`${endpoint}/${row.id}`, { ...row, enabled: val }) }
    catch { setRows((prev) => prev.map((r) => r.id === row.id ? { ...r, enabled: !val } : r)); toast.error(t('operation_failed')) }
  }

  const toggleDefault = async (row: ConfigRow, val: boolean) => {
    setRows((prev) => prev.map((r) => r.id === row.id ? { ...r, is_default: val } : r))
    try { await api.put(`${endpoint}/${row.id}`, { ...row, is_default: val }); await load(page) }
    catch { setRows((prev) => prev.map((r) => r.id === row.id ? { ...r, is_default: !val } : r)); toast.error(t('operation_failed')) }
  }

  const totalPages = Math.ceil(total / pageSize) || 1

  return (
    <div className="grid gap-4 px-6 pb-8">
      <div className="flex justify-end items-center gap-2">
        <Button variant="outline" onClick={() => load(page)}><RefreshCw className="w-4 h-4 mr-1.5" />{t('refresh')}</Button>
        <Button onClick={openAdd}><Plus className="w-4 h-4 mr-1.5" />{addLabel || t('add_config')}</Button>
      </div>

      <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
        {loading ? (
          <div className="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-14">ID</TableHead>
                  <TableHead>{t('config_name')}</TableHead>
                  <TableHead className="w-36">{t('config_id')}</TableHead>
                  {extraColumns.map((c) => <TableHead key={c.key}>{c.header}</TableHead>)}
                  <TableHead className="w-20 text-center">{t('enabled_status')}</TableHead>
                  <TableHead className="w-20 text-center">{t('default_config')}</TableHead>
                  <TableHead className="w-40">{t('created_at')}</TableHead>
                  <TableHead className="w-14 text-center">{t('actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.length === 0 ? (
                  <TableRow><TableCell colSpan={7 + extraColumns.length} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
                ) : rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="text-xs text-[var(--color-text-secondary)]">{row.id}</TableCell>
                    <TableCell className="font-medium">{row.name}</TableCell>
                    <TableCell className="text-sm text-[var(--color-text-secondary)] font-mono">{row.config_id}</TableCell>
                    {extraColumns.map((c) => <TableCell key={c.key}>{c.render ? c.render(row) : String((row as never)[c.key] ?? '—')}</TableCell>)}
                    <TableCell className="text-center"><Switch checked={row.enabled !== false} onCheckedChange={(v) => toggleEnabled(row, v)} /></TableCell>
                    <TableCell className="text-center"><Switch checked={!!row.is_default} onCheckedChange={(v) => toggleDefault(row, v)} /></TableCell>
                    <TableCell className="text-sm text-[var(--color-text-secondary)]">{new Date(row.created_at).toLocaleString()}</TableCell>
                    <TableCell className="text-center">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-7 w-7"><MoreHorizontal className="w-4 h-4" /></Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(row)}>{t('edit')}</DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(row)}>{t('delete')}</DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {total > pageSize && (
        <div className="flex items-center justify-end gap-2 text-sm text-[var(--color-text-secondary)]">
          <span>{total} {t('total_items')}</span>
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => { setPage(page - 1); load(page - 1) }}>{t('prev')}</Button>
          <span>{page} / {totalPages}</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => { setPage(page + 1); load(page + 1) }}>{t('next')}</Button>
        </div>
      )}

      <Dialog open={showDialog} onOpenChange={(v) => { if (!v) setShowDialog(false) }}>
        <DialogContent className="max-w-[620px]">
          <DialogHeader><DialogTitle>{editing ? (editLabel || t('edit_config')) : (addLabel || t('add_config'))}</DialogTitle></DialogHeader>
          <div className="max-h-[65vh] overflow-y-auto pr-1 grid gap-3 py-2">
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">{t('config_name')}</label>
              <Input value={form.name} onChange={(e) => setF({ name: e.target.value })} placeholder={t('enter_config_name')} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">{t('config_id')}</label>
              <Input value={form.config_id} onChange={(e) => setF({ config_id: e.target.value })} placeholder={t('enter_unique_config_id')} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">{t('provider')}</label>
              <Input value={form.provider} onChange={(e) => setF({ provider: e.target.value })} placeholder={t('provider')} />
            </div>
            <div className="flex items-center gap-6">
              <div className="flex items-center gap-2">
                <Switch checked={form.enabled} onCheckedChange={(v) => setF({ enabled: v })} />
                <span className="text-sm">{t('enabled_status')}</span>
              </div>
              <div className="flex items-center gap-2">
                <Switch checked={form.is_default} onCheckedChange={(v) => setF({ is_default: v })} />
                <span className="text-sm">{t('default_config')}</span>
              </div>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">JSON {t('config')}</label>
              <Textarea value={form.json_data} onChange={(e) => setF({ json_data: e.target.value })} rows={8} className="font-mono text-xs resize-y" placeholder="{}" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>{t('cancel')}</Button>
            <Button disabled={saving} onClick={handleSave}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        isLoading={deleting}
        title={t('confirm_delete')}
        description={t('confirm_delete_config')}
      />
    </div>
  )
}
