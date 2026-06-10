import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'

interface KB { id: number; name: string; sync_provider?: string }
interface Doc { id: number; name: string; external_doc_id?: string; content?: string; sync_status?: string; last_synced_at?: string }

interface Props { kb: KB; open: boolean; onClose: () => void; onRefresh: () => void }

const FILE_UPLOAD_PREFIX = '__KB_FILE_UPLOAD_V1__:'

const syncBadge = (s: string) => {
  if (['upload_failed', 'parse_failed', 'failed'].includes(s)) return 'status-danger'
  if (['uploading', 'parsing'].includes(s)) return 'status-warning'
  if (s === 'synced') return 'status-success'
  return 'status-muted'
}

const UPLOAD_ACCEPT = '.txt,.md,.markdown,.pdf,.html,.htm,.xlsx,.xls,.docx,.csv,.eml,.msg,.pptx,.ppt,.xml,.epub'

export function KnowledgeBaseDocumentsDialog({ kb, open, onClose, onRefresh }: Props) {
  const { t } = useLocale()
  const [docs, setDocs] = useState<Doc[]>([])
  const [loading, setLoading] = useState(false)

  const [showDocForm, setShowDocForm] = useState(false)
  const [editingDoc, setEditingDoc] = useState<Doc | null>(null)
  const [docForm, setDocForm] = useState({ name: '', content: '' })
  const [savingDoc, setSavingDoc] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState<Doc | null>(null)
  const [deleting, setDeleting] = useState(false)

  const fileInputRef = useRef<HTMLInputElement>(null)

  const isFileDoc = (doc: Doc) => typeof doc.content === 'string' && doc.content.startsWith(FILE_UPLOAD_PREFIX)
  const docPreview = (doc: Doc) => {
    if (isFileDoc(doc)) {
      try { const p = JSON.parse(doc.content!.slice(FILE_UPLOAD_PREFIX.length)); return t('file_document_name', { name: p?.file_name || doc.name || t('upload_file') }) } catch { return t('file_document_name', { name: doc.name || t('upload_file') }) }
    }
    const text = doc.content || ''
    return text.slice(0, 120) + (text.length > 120 ? '...' : '')
  }

  const loadDocs = async () => {
    setLoading(true)
    try { const res = await api.get(`/user/knowledge-bases/${kb.id}/documents`); setDocs(res.data.data || []) }
    catch { toast.error(t('load_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { if (open) loadDocs() }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  const openCreate = () => { setEditingDoc(null); setDocForm({ name: '', content: '' }); setShowDocForm(true) }
  const openEditDoc = (doc: Doc) => {
    if (isFileDoc(doc)) { toast.warning(t('file_doc_no_online_edit')); return }
    setEditingDoc(doc); setDocForm({ name: doc.name, content: doc.content || '' }); setShowDocForm(true)
  }

  const saveDoc = async () => {
    if (!docForm.name.trim()) { toast.error(t('doc_name_required')); return }
    if (!docForm.content.trim()) { toast.error(t('doc_content_required')); return }
    setSavingDoc(true)
    try {
      if (editingDoc) { await api.put(`/user/knowledge-bases/${kb.id}/documents/${editingDoc.id}`, docForm) }
      else { await api.post(`/user/knowledge-bases/${kb.id}/documents`, docForm) }
      toast.success(t('doc_save_success'))
      setShowDocForm(false); await loadDocs(); onRefresh()
    } catch (e) { toast.error((e as { response?: { data?: { error?: string } } }).response?.data?.error || t('doc_save_failed')) }
    finally { setSavingDoc(false) }
  }

  const syncDoc = async (docId: number) => {
    try { const res = await api.post(`/user/knowledge-bases/${kb.id}/documents/${docId}/sync`); toast.success(res.data?.message || t('sync_submitted')); await loadDocs() }
    catch (e) { toast.error((e as { response?: { data?: { error?: string } } }).response?.data?.error || t('sync_failed')) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try { await api.delete(`/user/knowledge-bases/${kb.id}/documents/${deleteTarget.id}`); toast.success(t('delete_success')); setDeleteTarget(null); await loadDocs(); onRefresh() }
    catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const fd = new FormData()
    fd.append('file', file)
    const fileName = file.name.replace(/\.[^/.]+$/, '')
    if (fileName) fd.append('name', fileName)
    try {
      const res = await api.post(`/user/knowledge-bases/${kb.id}/documents/upload`, fd)
      toast.success(res.data?.message || t('file_upload_success'))
      await loadDocs(); onRefresh()
    } catch (e) { toast.error((e as { response?: { data?: { error?: string } } }).response?.data?.error || t('file_upload_failed')) }
    finally { if (e.target) (e.target as HTMLInputElement).value = '' }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={v => { if (!v) onClose() }}>
        <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('document_management')}</DialogTitle></DialogHeader>
          <div className="grid gap-3 py-2">
            <div className="flex items-center justify-between gap-2 flex-wrap">
              <span className="text-sm text-[var(--color-text-secondary)]">{t('current_kb_info', { name: kb.name })}</span>
              <div className="flex gap-2">
                <Button variant="outline" onClick={() => fileInputRef.current?.click()}>{t('upload_file')}</Button>
                <input ref={fileInputRef} type="file" accept={UPLOAD_ACCEPT} className="hidden" onChange={handleFileUpload} />
                <Button onClick={openCreate}>{t('add_document')}</Button>
              </div>
            </div>
            <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-20">ID</TableHead>
                      <TableHead className="w-44">{t('document_name')}</TableHead>
                      <TableHead className="w-52">Document ID</TableHead>
                      <TableHead>{t('content_preview')}</TableHead>
                      <TableHead className="w-28">{t('sync_status')}</TableHead>
                      <TableHead className="w-48">{t('actions')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {loading ? (
                      <TableRow><TableCell colSpan={6} className="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell></TableRow>
                    ) : docs.length === 0 ? (
                      <TableRow><TableCell colSpan={6} className="py-8 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
                    ) : docs.map(doc => (
                      <TableRow key={doc.id}>
                        <TableCell className="text-xs font-mono text-[var(--color-text-secondary)]">{doc.id}</TableCell>
                        <TableCell className="text-sm font-medium">{doc.name}</TableCell>
                        <TableCell className="text-xs font-mono text-[var(--color-text-secondary)]">{doc.external_doc_id}</TableCell>
                        <TableCell className="text-xs text-[var(--color-text-secondary)] max-w-[180px] truncate">{docPreview(doc)}</TableCell>
                        <TableCell><span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${syncBadge(doc.sync_status || '')}`}>{doc.sync_status || '-'}</span></TableCell>
                        <TableCell>
                          <div className="flex gap-1 flex-wrap">
                            <Button variant="outline" size="sm" disabled={isFileDoc(doc)} onClick={() => openEditDoc(doc)}>{t('edit')}</Button>
                            <Button variant="outline" size="sm" onClick={() => syncDoc(doc.id)}>{t('retry_sync')}</Button>
                            <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setDeleteTarget(doc)}>{t('delete')}</Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={showDocForm} onOpenChange={v => { if (!v) setShowDocForm(false) }}>
        <DialogContent className="max-w-2xl">
          <DialogHeader><DialogTitle>{editingDoc ? t('edit_document') : t('add_document')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('document_name')}</label><Input value={docForm.name} onChange={e => setDocForm(f => ({ ...f, name: e.target.value }))} maxLength={200} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('content_label')}</label><Textarea value={docForm.content} onChange={e => setDocForm(f => ({ ...f, content: e.target.value }))} rows={12} placeholder={t('content_ph')} /></div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDocForm(false)}>{t('cancel')}</Button>
            <Button disabled={savingDoc} onClick={saveDoc}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_document')} />
    </>
  )
}
