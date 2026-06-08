import { useEffect, useRef, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'
import { useAuthStore } from '@/stores/auth'

interface Clone { id: number; name: string; provider: string; tts_config_id: string; tts_config_name?: string; provider_voice_id?: string; status?: string; task_status?: string; task_last_error?: string; meta_json?: string; shared_to_all?: boolean; created_at: string }
interface TtsConfig { id: number; name: string; config_id: string; provider: string; enabled: boolean }

const PENDING = ['queued', 'processing']
const normalizeStatus = (r: Clone) => {
  const s = String(r.status || '').toLowerCase(), ts = String(r.task_status || '').toLowerCase()
  if (s === 'failed' || ts === 'failed') return 'failed'
  if (s === 'active' || ts === 'succeeded') return 'active'
  if (PENDING.includes(ts)) return ts
  if (PENDING.includes(s)) return s
  return s || ts || 'unknown'
}
const statusLabel = (r: Clone, t: (k: string) => string) => {
  const s = normalizeStatus(r)
  return { queued: t('queuing'), processing: t('processing'), active: t('success'), failed: t('failed') }[s] || t('unknown')
}
const statusBadge = (r: Clone) => {
  const s = normalizeStatus(r)
  if (s === 'queued') return 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800'
  if (s === 'processing') return 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800'
  if (s === 'active') return 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
  if (s === 'failed') return 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
  return 'bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'
}
const lastError = (r: Clone) => {
  if (normalizeStatus(r) !== 'failed') return '-'
  if (r.task_last_error) return r.task_last_error
  try { return JSON.parse(r.meta_json || '').last_error || '-' } catch { return '-' }
}

const CLONE_PROVIDERS = ['doubao_ws', 'minimax', 'cosyvoice', 'aliyun_qwen', 'indextts_vllm']

function VoiceClonesPage() {
  const { t } = useLocale()
  const authStore = useAuthStore()
  const [clones, setClones] = useState<Clone[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const pageSize = 20
  const totalPages = Math.ceil(total / pageSize) || 1

  const [ttsConfigs, setTtsConfigs] = useState<TtsConfig[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState({ name: '', tts_config_id: '', source_type: 'upload', transcript: '', transcript_lang: 'zh-CN' })
  const [audioFile, setAudioFile] = useState<File | null>(null)

  const [editTarget, setEditTarget] = useState<Clone | null>(null)
  const [editName, setEditName] = useState('')
  const [editSaving, setEditSaving] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState<Clone | null>(null)
  const [deleting, setDeleting] = useState(false)

  const [previewUrl, setPreviewUrl] = useState('')
  const [previewLabel, setPreviewLabel] = useState('')
  const [previewSubmitting, setPreviewSubmitting] = useState<Record<number, string>>({})

  const pollingRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const appendFileRef = useRef<HTMLInputElement>(null)
  const appendTarget = useRef<Clone | null>(null)

  const setF = (p: Partial<typeof form>) => setForm(f => ({ ...f, ...p }))

  const cloneEnabledConfigs = ttsConfigs.filter(c => CLONE_PROVIDERS.includes(c.provider))

  const load = async (silent = false) => {
    if (!silent) setLoading(true)
    try {
      const res = await api.get('/user/voice-clones', { params: { page, page_size: pageSize } })
      const data: Clone[] = res.data.data || []
      setClones(data); setTotal(res.data.total || 0)
      if (data.some(c => PENDING.includes(normalizeStatus(c)))) schedulePolling()
      else clearPolling()
    } finally { if (!silent) setLoading(false) }
  }

  const loadTts = async () => { try { const res = await api.get('/user/tts-configs'); setTtsConfigs(res.data.data || []) } catch {} }

  const schedulePolling = () => {
    if (pollingRef.current) return
    pollingRef.current = setTimeout(async () => { pollingRef.current = null; await load(true) }, 2000)
  }
  const clearPolling = () => { if (pollingRef.current) { clearTimeout(pollingRef.current); pollingRef.current = null } }

  useEffect(() => { load(); return () => clearPolling() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openCreate = async () => {
    await loadTts()
    const first = cloneEnabledConfigs[0]?.config_id || ''
    setForm({ name: '', tts_config_id: first, source_type: 'upload', transcript: '', transcript_lang: 'zh-CN' })
    setAudioFile(null)
    if (fileRef.current) fileRef.current.value = ''
    setShowCreate(true)
  }

  const submitClone = async () => {
    if (!form.tts_config_id) { toast.warning(t('select_clonable_tts')); return }
    if (!audioFile) { toast.warning(t('upload_audio_file')); return }
    const fd = new FormData()
    fd.append('name', form.name); fd.append('tts_config_id', form.tts_config_id)
    fd.append('source_type', 'upload'); fd.append('transcript', form.transcript); fd.append('transcript_lang', form.transcript_lang)
    fd.append('audio_file', audioFile)
    setSubmitting(true)
    try {
      await api.post('/user/voice-clones', fd, { timeout: 120000 })
      toast.success(t('clone_task_submitted'))
      setShowCreate(false); await load()
    } catch { toast.error(t('create_failed')) }
    finally { setSubmitting(false) }
  }

  const retryClone = async (c: Clone) => {
    try { await api.post(`/user/voice-clones/${c.id}/retry`); toast.success(t('reclone_task_submitted')); await load(true) }
    catch { toast.error(t('retry_failed')) }
  }

  const saveEdit = async () => {
    if (!editTarget || !editName.trim()) { toast.error(t('name_required')); return }
    if (editName.trim() === editTarget.name) { setEditTarget(null); return }
    setEditSaving(true)
    try { await api.put(`/user/voice-clones/${editTarget.id}`, { name: editName.trim() }); toast.success(t('name_update_success')); setEditTarget(null); await load(true) }
    catch { toast.error(t('save_failed')) }
    finally { setEditSaving(false) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try { await api.delete(`/user/voice-clones/${deleteTarget.id}`); toast.success(t('delete_success')); setDeleteTarget(null); await load(true) }
    catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  const previewAudio = async (c: Clone, type: 'upload' | 'clone') => {
    setPreviewSubmitting(prev => ({ ...prev, [c.id]: type }))
    try {
      let blobRes
      if (type === 'upload') {
        const audios = await api.get(`/user/voice-clones/${c.id}/audios`)
        const first = (audios.data.data || [])[0]
        if (!first) { toast.warning(t('no_uploaded_audio')); return }
        blobRes = await api.get(`/user/voice-clones/audios/${first.id}/file`, { responseType: 'blob' })
      } else {
        blobRes = await api.get(`/user/voice-clones/${c.id}/preview`, { responseType: 'blob' })
      }
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      setPreviewUrl(URL.createObjectURL(blobRes.data))
      setPreviewLabel(`${type === 'upload' ? t('original_audio') : t('preview_clone')} — ${c.name}`)
    } catch (e) { toast.error((e as { response?: { data?: { error?: string } } }).response?.data?.error || t('preview_failed')) }
    finally { setPreviewSubmitting(prev => { const n = { ...prev }; delete n[c.id]; return n }) }
  }

  const toggleShared = async (c: Clone, val: boolean) => {
    try { await api.put(`/user/voice-clones/${c.id}`, { shared_to_all: val }); setClones(prev => prev.map(x => x.id === c.id ? { ...x, shared_to_all: val } : x)); toast.success(val ? t('enabled_for_all') : t('sharing_closed')) }
    catch { toast.error(t('save_failed')) }
  }

  const openAppendAudio = (c: Clone) => { appendTarget.current = c; appendFileRef.current?.click() }
  const handleAppend = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; const c = appendTarget.current
    if (!file || !c) { appendTarget.current = null; return }
    const fd = new FormData(); fd.append('source_type', 'upload'); fd.append('audio_file', file)
    try { await api.post(`/user/voice-clones/${c.id}/append-audio`, fd, { timeout: 120000 }); toast.success(t('append_ref_audio_success')); await load(true) }
    catch (e) { toast.error((e as { response?: { data?: { error?: string } } }).response?.data?.error || t('append_ref_audio_failed')) }
    finally { appendTarget.current = null; if (e.target) (e.target as HTMLInputElement).value = '' }
  }

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('voice_clone')} />
      <div className="flex justify-end">
        <Button onClick={openCreate}>{t('create_clone_voice')}</Button>
      </div>
      <input ref={appendFileRef} type="file" accept=".wav,audio/wav" className="hidden" onChange={handleAppend} />

      <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('name')}</TableHead>
                <TableHead className="w-24">{t('provider')}</TableHead>
                <TableHead>{t('tts_config_label')}</TableHead>
                <TableHead>{t('clone_voice_id')}</TableHead>
                {authStore.isAdmin && <TableHead className="w-32 text-center">{t('share_to_all_col')}</TableHead>}
                <TableHead className="w-24">{t('task_status')}</TableHead>
                <TableHead>{t('failure_reason')}</TableHead>
                <TableHead className="w-40">{t('created_at')}</TableHead>
                <TableHead>{t('actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow><TableCell colSpan={authStore.isAdmin ? 9 : 8} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</TableCell></TableRow>
              ) : clones.length === 0 ? (
                <TableRow><TableCell colSpan={authStore.isAdmin ? 9 : 8} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
              ) : clones.map(c => (
                <TableRow key={c.id}>
                  <TableCell className="font-medium text-sm">{c.name}</TableCell>
                  <TableCell className="text-sm text-[var(--color-text-secondary)]">{c.provider}</TableCell>
                  <TableCell className="text-sm text-[var(--color-text-secondary)] max-w-[180px] truncate">{`${c.tts_config_name || '-'} (${c.tts_config_id || '-'})`}</TableCell>
                  <TableCell className="text-xs font-mono text-[var(--color-text-secondary)] max-w-[160px] truncate" title={c.provider_voice_id}>{c.provider_voice_id}</TableCell>
                  {authStore.isAdmin && (
                    <TableCell className="text-center">
                      <Switch checked={!!c.shared_to_all} disabled={normalizeStatus(c) !== 'active'} onCheckedChange={v => toggleShared(c, v)} />
                    </TableCell>
                  )}
                  <TableCell><span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${statusBadge(c)}`}>{statusLabel(c, t)}</span></TableCell>
                  <TableCell className="text-xs text-[var(--color-text-secondary)] max-w-[140px] truncate" title={lastError(c)}>{lastError(c)}</TableCell>
                  <TableCell className="text-xs text-[var(--color-text-secondary)]">{c.created_at ? new Date(c.created_at).toLocaleString() : '-'}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      <Button variant="outline" size="sm" disabled={!!previewSubmitting[c.id]} onClick={() => previewAudio(c, 'upload')}>{t('original_audio')}</Button>
                      {normalizeStatus(c) === 'active' && <Button variant="outline" size="sm" disabled={!!previewSubmitting[c.id]} onClick={() => previewAudio(c, 'clone')}>{t('preview_clone')}</Button>}
                      <Button variant="outline" size="sm" onClick={() => { setEditTarget(c); setEditName(c.name) }}>{t('edit')}</Button>
                      {normalizeStatus(c) === 'failed' && <Button variant="outline" size="sm" onClick={() => retryClone(c)}>{t('re_clone')}</Button>}
                      {normalizeStatus(c) === 'active' && c.provider === 'indextts_vllm' && <Button variant="outline" size="sm" onClick={() => openAppendAudio(c)}>{t('append_reference_audio')}</Button>}
                      <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setDeleteTarget(c)}>{t('delete')}</Button>
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
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => { setPage(p => p - 1); load() }}>{t('prev')}</Button>
            <span>{page} / {totalPages}</span>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => { setPage(p => p + 1); load() }}>{t('next')}</Button>
          </div>
        </div>
      )}

      {/* Create dialog */}
      <Dialog open={showCreate} onOpenChange={v => { if (!v) setShowCreate(false) }}>
        <DialogContent className="max-w-xl max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('create_clone_voice')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('clone_name_label')}</label><Input value={form.name} onChange={e => setF({ name: e.target.value })} placeholder={t('clone_name_optional_ph')} /></div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('tts_config_label')} <span className="text-destructive">*</span></label>
              <Select value={form.tts_config_id} onValueChange={v => setF({ tts_config_id: v })}>
                <SelectTrigger><SelectValue placeholder={t('select_cloneable_tts_ph')} /></SelectTrigger>
                <SelectContent>{cloneEnabledConfigs.map(c => <SelectItem key={c.config_id} value={c.config_id}>{c.name} ({c.config_id})</SelectItem>)}</SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('audio_file_label')} <span className="text-destructive">*</span></label>
              <input ref={fileRef} type="file" accept=".wav,.mp3,.m4a,audio/wav,audio/mpeg,audio/mp4" className="text-sm" onChange={e => setAudioFile(e.target.files?.[0] || null)} />
              <p className="text-xs text-[var(--color-text-tertiary)]">{t('wav_requirement')}</p>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('audio_corresponding_text')}</label>
              <textarea value={form.transcript} onChange={e => setF({ transcript: e.target.value })} rows={4} placeholder={t('optional_submit')} className="dark:bg-input/30 border-input rounded-md border bg-transparent px-2.5 py-2 text-sm resize-none focus-visible:outline-none" />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('text_language')}</label>
              <Select value={form.transcript_lang} onValueChange={v => setF({ transcript_lang: v })}>
                <SelectTrigger className="w-[220px]"><SelectValue /></SelectTrigger>
                <SelectContent><SelectItem value="zh-CN">{t('chinese_lang_option')}</SelectItem><SelectItem value="en-US">{t('english_lang_option')}</SelectItem></SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>{t('cancel')}</Button>
            <Button disabled={submitting} onClick={submitClone}>{t('submit_clone')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit name dialog */}
      <Dialog open={!!editTarget} onOpenChange={v => { if (!v) setEditTarget(null) }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>{t('edit_clone_voice')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('name')}</label><Input value={editName} onChange={e => setEditName(e.target.value)} maxLength={100} /></div>
            {editTarget && <div className="grid gap-1.5"><label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('task_status')}</label><Input value={statusLabel(editTarget, t)} readOnly className="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" /></div>}
            {editTarget && lastError(editTarget) !== '-' && (
              <div className="grid gap-1.5"><label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('failure_reason')}</label><textarea value={lastError(editTarget)} readOnly rows={3} className="dark:bg-input/30 border-input rounded-md border bg-[var(--color-surface-muted)] px-2.5 py-2 text-sm text-[var(--color-text-secondary)] resize-none focus-visible:outline-none" /></div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>{t('cancel')}</Button>
            <Button disabled={editSaving} onClick={saveEdit}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Audio preview */}
      {previewUrl && (
        <Dialog open={!!previewUrl} onOpenChange={v => { if (!v) { URL.revokeObjectURL(previewUrl); setPreviewUrl('') } }}>
          <DialogContent className="max-w-lg">
            <DialogHeader><DialogTitle>{t('audio_preview_title')}</DialogTitle></DialogHeader>
            <div className="grid gap-3 py-2">
              <p className="text-sm text-[var(--color-text-secondary)]">{previewLabel}</p>
              <audio src={previewUrl} controls autoPlay className="w-full" />
            </div>
          </DialogContent>
        </Dialog>
      )}

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_clone_voice', { name: deleteTarget?.name || '' })} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/voice-clones')({
  component: VoiceClonesPage,
})
