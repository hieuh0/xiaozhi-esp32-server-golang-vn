import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { AlertTriangle, MoreHorizontal, Plus } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'
import { KnowledgeBaseDocumentsDialog } from '@/components/user/knowledge-base-documents-dialog'

interface KB { id: number; name: string; description?: string; sync_provider?: string; doc_count?: number; sync_status?: string; sync_error?: string; last_synced_at?: string; status: string; retrieval_threshold?: number | null }

const FILE_UPLOAD_PREFIX = '__KB_FILE_UPLOAD_V1__:'
const DEFAULT_THRESHOLD = 0.2

const syncBadge = (s: string) => {
  if (['upload_failed', 'parse_failed', 'failed'].includes(s)) return 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
  if (['uploading', 'parsing'].includes(s)) return 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800'
  if (s === 'synced') return 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
  return 'bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'
}
const syncText = (s: string, t: (k: string) => string) => ({ uploading: t('uploading'), uploaded: t('uploaded'), parsing: t('parsing'), upload_failed: t('upload_failed'), parse_failed: t('parse_failed'), synced: t('synced'), failed: t('failed') })[s] || t('pending_sync')
const providerLabel = (p: string) => ({ ragflow: 'RAGFlow', weknora: 'WeKnora', dify: 'Dify' }[p?.toLowerCase()] || p || '-')
const fmtThreshold = (v: number | null | undefined, t: (k: string) => string) => (v == null) ? t('global') : Number(v).toFixed(2)

const defForm = () => ({ name: '', description: '', status: 'active', retrieval_threshold_text: String(DEFAULT_THRESHOLD) })

function KnowledgeBasesPage() {
  const { t } = useLocale()
  const [items, setItems] = useState<KB[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const pageSize = 20
  const totalPages = Math.ceil(total / pageSize) || 1

  const [showDialog, setShowDialog] = useState(false)
  const [editing, setEditing] = useState<KB | null>(null)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState(defForm())
  const [deleteTarget, setDeleteTarget] = useState<KB | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [docKb, setDocKb] = useState<KB | null>(null)
  const [globalProvider, setGlobalProvider] = useState('dify')

  const setF = (p: Partial<typeof form>) => setForm(f => ({ ...f, ...p }))

  const load = async (p = page) => {
    setLoading(true)
    try {
      const res = await api.get('/user/knowledge-bases', { params: { page: p, page_size: pageSize } })
      setItems(res.data.data || [])
      setTotal(res.data.total || 0)
      setGlobalProvider(res.data.knowledge?.default_provider || 'dify')
    } catch { toast.error(t('load_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load(1) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openCreate = () => {
    setEditing(null)
    setForm({ ...defForm(), retrieval_threshold_text: String(DEFAULT_THRESHOLD) })
    setShowDialog(true)
  }
  const openEdit = (kb: KB) => {
    setEditing(kb)
    setForm({ name: kb.name, description: kb.description || '', status: kb.status || 'active', retrieval_threshold_text: kb.retrieval_threshold != null ? String(kb.retrieval_threshold) : String(DEFAULT_THRESHOLD) })
    setShowDialog(true)
  }

  const submit = async () => {
    if (!form.name.trim()) { toast.error(t('name_required')); return }
    const threshold = Number(form.retrieval_threshold_text)
    if (isNaN(threshold) || threshold < 0 || threshold > 1) { toast.error(t('retrieval_threshold_range')); return }
    setSaving(true)
    try {
      const payload = { name: form.name, description: form.description, status: form.status, retrieval_threshold: threshold }
      if (editing) { await api.put(`/user/knowledge-bases/${editing.id}`, payload); toast.success(t('save_success')) }
      else { await api.post('/user/knowledge-bases', payload); toast.success(t('save_success')) }
      setShowDialog(false); await load(page)
    } catch { toast.error(t('save_failed')) }
    finally { setSaving(false) }
  }

  const toggleStatus = async (kb: KB, checked: boolean) => {
    const nextStatus = checked ? 'active' : 'inactive'
    setItems(prev => prev.map(i => i.id === kb.id ? { ...i, status: nextStatus } : i))
    try {
      await api.put(`/user/knowledge-bases/${kb.id}`, { name: kb.name, status: nextStatus })
      toast.success(checked ? t('enabled') : t('deactivate'))
      await load(page)
    } catch { await load(page); toast.error(t('status_update_failed')) }
  }

  const syncKb = async (id: number) => {
    try { const res = await api.post(`/user/knowledge-bases/${id}/sync`); toast.success(res.data?.message || t('sync_submitted')); await load(page) }
    catch (e) { toast.error((e as { response?: { data?: { error?: string } } }).response?.data?.error || t('sync_failed')); await load(page) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try { await api.delete(`/user/knowledge-bases/${deleteTarget.id}`); toast.success(t('delete_success')); setDeleteTarget(null); await load(page) }
    catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('my_knowledge_bases')} />
      <div className="flex justify-end">
        <Button onClick={openCreate}><Plus className="w-4 h-4 mr-1.5" />{t('add_knowledge_base')}</Button>
      </div>
      <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-14">ID</TableHead>
                <TableHead className="w-32">{t('name')}</TableHead>
                <TableHead>{t('description')}</TableHead>
                <TableHead className="w-24">{t('provider')}</TableHead>
                <TableHead className="w-20 text-center">{t('doc_count_col')}</TableHead>
                <TableHead className="w-36">{t('sync_status')}</TableHead>
                <TableHead className="w-28 text-center">{t('status')}</TableHead>
                <TableHead className="w-56">{t('actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow><TableCell colSpan={8} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</TableCell></TableRow>
              ) : items.length === 0 ? (
                <TableRow><TableCell colSpan={8} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
              ) : items.map(kb => (
                <TableRow key={kb.id}>
                  <TableCell className="text-xs font-mono text-[var(--color-text-secondary)]">{kb.id}</TableCell>
                  <TableCell className="font-medium text-sm" title={kb.name}>{kb.name}</TableCell>
                  <TableCell className="text-sm text-[var(--color-text-secondary)] max-w-[180px] truncate" title={kb.description || '-'}>{(kb.description || '').trim() || '-'}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700">{providerLabel(kb.sync_provider || globalProvider)}</span>
                  </TableCell>
                  <TableCell className="text-center text-sm">{kb.doc_count ?? 0}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5">
                      <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${syncBadge(kb.sync_status || '')}`}>{syncText(kb.sync_status || '', t)}</span>
                      {kb.sync_error && ['failed', 'upload_failed', 'parse_failed'].includes(kb.sync_status || '') && (
                        <span title={kb.sync_error}><AlertTriangle className="w-3.5 h-3.5 text-red-500 shrink-0 cursor-help" /></span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-center">
                    <Switch checked={kb.status === 'active'} onCheckedChange={v => toggleStatus(kb, v)} />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1 flex-wrap">
                      <Button variant="outline" size="sm" onClick={() => setDocKb(kb)}>{t('document_btn')}</Button>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild><Button variant="outline" size="sm"><MoreHorizontal className="w-4 h-4" /></Button></DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(kb)}>{t('edit')}</DropdownMenuItem>
                          <DropdownMenuItem onClick={() => syncKb(kb.id)}>{t('retry_sync')}</DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(kb)}>{t('delete')}</DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
      {total > pageSize && (
        <div className="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
          <span>{total} {t('total_items')}</span>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => { setPage(p => p - 1); load(page - 1) }}>{t('prev')}</Button>
            <span>{page} / {totalPages}</span>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => { setPage(p => p + 1); load(page + 1) }}>{t('next')}</Button>
          </div>
        </div>
      )}

      <Dialog open={showDialog} onOpenChange={v => { if (!v) setShowDialog(false) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{editing ? t('edit_knowledge_base') : t('add_knowledge_base')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('name')}</label><Input value={form.name} onChange={e => setF({ name: e.target.value })} maxLength={100} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('description')}</label><Input value={form.description} onChange={e => setF({ description: e.target.value })} /></div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('retrieve_threshold')}</label>
              <Input value={form.retrieval_threshold_text} onChange={e => setF({ retrieval_threshold_text: e.target.value })} placeholder={t('threshold_test_ph')} />
              <p className="text-xs text-[var(--color-text-tertiary)]">{t('threshold_hint', { provider: globalProvider, threshold: fmtThreshold(DEFAULT_THRESHOLD, t) })}</p>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('status')}</label>
              <Select value={form.status} onValueChange={v => setF({ status: v })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent><SelectItem value="active">active</SelectItem><SelectItem value="inactive">inactive</SelectItem></SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>{t('cancel')}</Button>
            <Button disabled={saving} onClick={submit}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {docKb && <KnowledgeBaseDocumentsDialog kb={docKb} open={!!docKb} onClose={() => setDocKb(null)} onRefresh={() => load(page)} />}
      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_knowledge_base')} />
    </div>
  )
}

export { FILE_UPLOAD_PREFIX, syncBadge, syncText }
export const Route = createFileRoute('/_auth/_layout/user/knowledge-bases')({
  component: KnowledgeBasesPage,
})
