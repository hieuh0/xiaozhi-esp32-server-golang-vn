import { useEffect, useRef, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { getAudioDurationSeconds, useVoiceRecording } from '@/hooks/use-voice-recording'
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
interface AudioFile { id: number; source_type: string; file_name: string; transcript: string }
interface Capability { enabled: boolean; requires_transcript: boolean; min_text_len: number; max_text_len: number }

const MIN_AUDIO_DURATION_SECONDS = 10
const CLONE_PROVIDERS = ['doubao', 'doubao_ws', 'minimax', 'cosyvoice', 'aliyun_qwen', 'indextts_vllm']
const PENDING = ['queued', 'processing']
const DEFAULT_CAPABILITY: Capability = { enabled: true, requires_transcript: false, min_text_len: 0, max_text_len: 0 }

// --- Status helpers ---
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
  if (s === 'queued') return 'status-primary'
  if (s === 'processing') return 'status-warning'
  if (s === 'active') return 'status-success'
  if (s === 'failed') return 'status-danger'
  return 'status-muted'
}
const lastError = (r: Clone) => {
  if (normalizeStatus(r) !== 'failed') return '-'
  if (r.task_last_error) return r.task_last_error
  try { return JSON.parse(r.meta_json || '').last_error || '-' } catch { return '-' }
}

// --- Provider utils ---
const resolveChargeNotice = (provider: string, scene: 'create' | 'preview') => {
  const p = String(provider || '').trim().toLowerCase()
  if (p === 'aliyun_qwen') return { message: scene === 'create' ? 'billing_qwen_per_voice' : 'billing_qwen_confirm' }
  if (p === 'minimax') return { message: scene === 'create' ? 'billing_minimax_first_fee' : 'billing_minimax_first_preview' }
  if (p === 'cosyvoice') return { message: scene === 'create' ? 'billing_cosyvoice_free' : 'billing_cosyvoice_confirm' }
  return { message: '' }
}
const isAliyunQwen = (p: string) => String(p || '').trim().toLowerCase() === 'aliyun_qwen'
const requiresMinimaxDuration = (p: string) => String(p || '').trim().toLowerCase() === 'minimax'
const uploadAcceptTypes = (p: string) =>
  isAliyunQwen(p) ? '.wav,.mp3,.m4a,audio/wav,audio/wave,audio/mpeg,audio/mp4,audio/x-m4a' : '.wav,audio/wav,audio/wave'

function VoiceClonesPage() {
  const { t } = useLocale()
  const authStore = useAuthStore()

  // List
  const [clones, setClones] = useState<Clone[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const pageSize = 20
  const totalPages = Math.ceil(total / pageSize) || 1

  // TTS configs
  const [ttsConfigs, setTtsConfigs] = useState<TtsConfig[]>([])

  // Create dialog
  const [showCreate, setShowCreate] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState({ name: '', tts_config_id: '', source_type: 'upload', transcript: '', transcript_lang: 'zh-CN' })
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [capability, setCapability] = useState<Capability>(DEFAULT_CAPABILITY)

  // Edit dialog
  const [editTarget, setEditTarget] = useState<Clone | null>(null)
  const [editName, setEditName] = useState('')
  const [editSaving, setEditSaving] = useState(false)

  // Delete
  const [deleteTarget, setDeleteTarget] = useState<Clone | null>(null)
  const [deleting, setDeleting] = useState(false)

  // Charge confirmation dialog (shared for submit + preview)
  const [chargeConfirm, setChargeConfirm] = useState<{ message: string; onConfirm: () => void } | null>(null)

  // Clone preview player
  const [previewUrl, setPreviewUrl] = useState('')
  const [previewLabel, setPreviewLabel] = useState('')
  const [previewSubmitting, setPreviewSubmitting] = useState<Record<number, boolean>>({})

  // Audio list dialog (Original Audio)
  const [audioListTarget, setAudioListTarget] = useState<Clone | null>(null)
  const [audioList, setAudioList] = useState<AudioFile[]>([])
  const [audioListLoading, setAudioListLoading] = useState(false)

  // Refs
  const pollingRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const appendFileRef = useRef<HTMLInputElement>(null)
  const appendTarget = useRef<Clone | null>(null)

  // Recording hook
  const recording = useVoiceRecording({
    onDurationCheck: (d) => {
      const provider = ttsConfigs.find(c => c.config_id === form.tts_config_id)?.provider || ''
      if (requiresMinimaxDuration(provider) && d < MIN_AUDIO_DURATION_SECONDS) {
        toast.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: d.toFixed(2) }))
        return false
      }
      return true
    },
    onError: (msg) => toast.error(msg),
  })

  const setF = (p: Partial<typeof form>) => setForm(f => ({ ...f, ...p }))
  const cloneEnabledConfigs = ttsConfigs.filter(c => CLONE_PROVIDERS.includes(c.provider))
  const selectedProvider = cloneEnabledConfigs.find(c => c.config_id === form.tts_config_id)?.provider || ''
  const chargeNoticeMsg = resolveChargeNotice(selectedProvider, 'create').message ? t(resolveChargeNotice(selectedProvider, 'create').message) : ''
  const audioRequirementText = requiresMinimaxDuration(selectedProvider)
    ? t('wav_min_dur_require', { n: MIN_AUDIO_DURATION_SECONDS })
    : isAliyunQwen(selectedProvider) ? t('audio_duration_requirement') : t('wav_requirement')

  // --- Data loading ---
  const load = async (silent = false, pg?: number) => {
    if (!silent) setLoading(true)
    try {
      const res = await api.get('/user/voice-clones', { params: { page: pg ?? page, page_size: pageSize } })
      const data: Clone[] = res.data.data || []
      setClones(data); setTotal(res.data.total || 0)
      if (data.some(c => PENDING.includes(normalizeStatus(c)))) schedulePolling()
      else clearPolling()
    } finally { if (!silent) setLoading(false) }
  }
  const loadTts = async (): Promise<TtsConfig[]> => {
    try { const res = await api.get('/user/tts-configs'); const data = res.data.data || []; setTtsConfigs(data); return data } catch { return [] }
  }

  const schedulePolling = () => {
    if (pollingRef.current) return
    pollingRef.current = setTimeout(async () => { pollingRef.current = null; await load(true) }, 2000)
  }
  const clearPolling = () => { if (pollingRef.current) { clearTimeout(pollingRef.current); pollingRef.current = null } }

  useEffect(() => { load(); return () => clearPolling() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // --- Create dialog ---
  const onConfigChange = async (configId: string) => {
    setF({ tts_config_id: configId })
    const cfg = cloneEnabledConfigs.find(c => c.config_id === configId)
    if (!cfg) { setCapability(DEFAULT_CAPABILITY); return }
    try {
      const res = await api.get('/user/voice-clone/capabilities', { params: { provider: cfg.provider } })
      setCapability(res.data.data || DEFAULT_CAPABILITY)
    } catch { setCapability(DEFAULT_CAPABILITY) }
  }

  const openCreate = async () => {
    const freshConfigs = await loadTts()
    recording.reset()
    setAudioFile(null)
    if (fileRef.current) fileRef.current.value = ''
    setCapability(DEFAULT_CAPABILITY)
    setForm({ name: '', tts_config_id: '', source_type: 'upload', transcript: '', transcript_lang: 'zh-CN' })
    setShowCreate(true)
    // auto-select first clone-enabled config using fresh data (not stale derived state)
    const firstCfg = freshConfigs.filter(c => CLONE_PROVIDERS.includes(c.provider))[0]?.config_id
    if (firstCfg) await onConfigChange(firstCfg)
  }

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] || null
    if (!file) { setAudioFile(null); return }
    if (!isAliyunQwen(selectedProvider)) {
      const ok = file.type.includes('audio/wav') || file.type.includes('audio/wave') || file.name.toLowerCase().endsWith('.wav')
      if (!ok) { toast.warning(t('wav_only')); setAudioFile(null); e.target.value = ''; return }
    }
    if (requiresMinimaxDuration(selectedProvider)) {
      try {
        const dur = await getAudioDurationSeconds(file)
        if (dur < MIN_AUDIO_DURATION_SECONDS) {
          toast.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: dur.toFixed(2) }))
          setAudioFile(null); e.target.value = ''; return
        }
      } catch (err) { toast.warning((err as Error).message || t('read_audio_duration_fail')); setAudioFile(null); e.target.value = ''; return }
    }
    setAudioFile(file)
  }

  const submitClone = async (bypass = false) => {
    if (!form.tts_config_id) { toast.warning(t('select_clonable_tts')); return }
    if (form.source_type === 'upload' && !audioFile) { toast.warning(t('upload_audio_file')); return }
    if (form.source_type === 'record' && !recording.recordBlob) { toast.warning(t('record_first')); return }
    const noticeKey = resolveChargeNotice(selectedProvider, 'create').message
    if (noticeKey && !bypass) {
      setChargeConfirm({ message: t(noticeKey), onConfirm: () => { setChargeConfirm(null); submitClone(true) } })
      return
    }
    if (capability.requires_transcript && !form.transcript.trim()) { toast.warning(t('provider_requires_audio_text')); return }
    if (form.source_type === 'upload' && audioFile && requiresMinimaxDuration(selectedProvider)) {
      try {
        const dur = await getAudioDurationSeconds(audioFile)
        if (dur < MIN_AUDIO_DURATION_SECONDS) { toast.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: dur.toFixed(2) })); return }
      } catch (err) { toast.warning((err as Error).message || t('read_audio_duration_fail')); return }
    }
    if (form.source_type === 'record' && requiresMinimaxDuration(selectedProvider) && recording.durationSec < MIN_AUDIO_DURATION_SECONDS) {
      toast.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: recording.durationSec.toFixed(2) })); return
    }
    const fd = new FormData()
    fd.append('name', form.name); fd.append('tts_config_id', form.tts_config_id)
    fd.append('transcript', form.transcript); fd.append('transcript_lang', form.transcript_lang)
    if (form.source_type === 'upload') {
      fd.append('source_type', 'upload'); fd.append('audio_file', audioFile!)
    } else {
      fd.append('source_type', 'record'); fd.append('audio_blob', recording.recordBlob!, `recording_${Date.now()}.wav`)
    }
    setSubmitting(true)
    try {
      await api.post('/user/voice-clones', fd, { timeout: 120000 })
      toast.success(t('clone_task_submitted'))
      setShowCreate(false); recording.reset(); await load()
    } catch { toast.error(t('create_failed')) }
    finally { setSubmitting(false) }
  }

  // --- List actions ---
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

  const previewClonedVoice = async (c: Clone, bypass = false) => {
    const noticeKey = resolveChargeNotice(c.provider, 'preview').message
    if (noticeKey && !bypass) {
      setChargeConfirm({ message: t(noticeKey), onConfirm: () => { setChargeConfirm(null); previewClonedVoice(c, true) } })
      return
    }
    setPreviewSubmitting(prev => ({ ...prev, [c.id]: true }))
    try {
      const res = await api.get(`/user/voice-clones/${c.id}/preview`, { responseType: 'blob' })
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      setPreviewUrl(URL.createObjectURL(res.data))
      setPreviewLabel(`${t('preview_clone')} — ${c.name}`)
    } catch (e) { toast.error((e as { response?: { data?: { error?: string } } }).response?.data?.error || t('preview_clone_audio_failed')) }
    finally { setPreviewSubmitting(prev => { const n = { ...prev }; delete n[c.id]; return n }) }
  }

  const toggleShared = async (c: Clone, val: boolean) => {
    try { await api.put(`/user/voice-clones/${c.id}`, { shared_to_all: val }); setClones(prev => prev.map(x => x.id === c.id ? { ...x, shared_to_all: val } : x)); toast.success(val ? t('enabled_for_all') : t('sharing_closed')) }
    catch { toast.error(t('save_failed')) }
  }

  // --- Audio List dialog ---
  const openAudioList = async (c: Clone) => {
    setAudioListTarget(c); setAudioList([]); setAudioListLoading(true)
    try {
      const res = await api.get(`/user/voice-clones/${c.id}/audios`)
      setAudioList(res.data.data || [])
    } catch { toast.error(t('preview_upload_audio_failed')) }
    finally { setAudioListLoading(false) }
  }

  const playAudioFile = async (audio: AudioFile) => {
    try {
      const res = await api.get(`/user/voice-clones/audios/${audio.id}/file`, { responseType: 'blob' })
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      setPreviewUrl(URL.createObjectURL(res.data))
      setPreviewLabel(`${t('original_audio')} — ${audio.file_name || t('clone_original_audio')}`)
    } catch { toast.error(t('preview_upload_audio_failed')) }
  }

  // --- Append audio (indextts_vllm only) ---
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
      <input ref={appendFileRef} type="file" accept=".wav,audio/wav,audio/wave" className="hidden" onChange={handleAppend} />

      {/* Main table */}
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
                      <Button variant="outline" size="sm" onClick={() => openAudioList(c)}>{t('original_audio')}</Button>
                      {normalizeStatus(c) === 'active' && <Button variant="outline" size="sm" disabled={!!previewSubmitting[c.id]} onClick={() => previewClonedVoice(c)}>{t('preview_clone')}</Button>}
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
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => { const prev = page - 1; setPage(prev); load(false, prev) }}>{t('prev')}</Button>
            <span>{page} / {totalPages}</span>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => { const next = page + 1; setPage(next); load(false, next) }}>{t('next')}</Button>
          </div>
        </div>
      )}

      {/* Create dialog */}
      <Dialog open={showCreate} onOpenChange={v => { if (!v) { setShowCreate(false); recording.reset() } }}>
        <DialogContent className="max-w-xl max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('create_clone_voice')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('clone_name_label')}</label>
              <Input value={form.name} onChange={e => setF({ name: e.target.value })} placeholder={t('clone_name_optional_ph')} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('tts_config_label')} <span className="text-destructive">*</span></label>
              <Select value={form.tts_config_id} onValueChange={onConfigChange}>
                <SelectTrigger><SelectValue placeholder={t('select_cloneable_tts_ph')} /></SelectTrigger>
                <SelectContent>{cloneEnabledConfigs.map(c => <SelectItem key={c.config_id} value={c.config_id}>{c.name} ({c.config_id})</SelectItem>)}</SelectContent>
              </Select>
              {isAliyunQwen(selectedProvider) && (
                <p className="text-xs text-[var(--color-text-tertiary)]">{t('qwen_clone_hint', { model: 'qwen3-tts-vc-2026-01-22' })}</p>
              )}
              {chargeNoticeMsg && (
                <div className="rounded-lg border border-yellow-200 bg-yellow-50 text-yellow-800 p-3 text-xs dark:bg-yellow-900/20 dark:border-yellow-800 dark:text-yellow-300">
                  {chargeNoticeMsg}
                </div>
              )}
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('audio_source')}</label>
              <div className="flex gap-2">
                <Button variant={form.source_type === 'upload' ? 'default' : 'outline'} size="sm" onClick={() => setF({ source_type: 'upload' })}>{t('upload_audio')}</Button>
                <Button variant={form.source_type === 'record' ? 'default' : 'outline'} size="sm" onClick={() => setF({ source_type: 'record' })}>{t('browser_record')}</Button>
              </div>
            </div>
            {form.source_type === 'upload' ? (
              <div className="grid gap-1.5">
                <label className="text-sm font-medium">{t('audio_file_label')} <span className="text-destructive">*</span></label>
                <input ref={fileRef} type="file" accept={uploadAcceptTypes(selectedProvider)} className="text-sm" onChange={handleFileChange} />
                <p className="text-xs text-[var(--color-text-tertiary)]">{audioRequirementText}</p>
              </div>
            ) : (
              <div className="grid gap-1.5">
                <label className="text-sm font-medium">{t('browser_record')} <span className="text-destructive">*</span></label>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" disabled={recording.isRecording} onClick={recording.startRecording}>{t('start_recording')}</Button>
                  <Button variant="destructive" size="sm" disabled={!recording.isRecording} onClick={recording.stopRecording}>{t('stop_recording_btn')}</Button>
                </div>
                {recording.recordPreviewUrl && <audio src={recording.recordPreviewUrl} controls className="w-full mt-1" />}
                <p className="text-xs text-[var(--color-text-tertiary)]">{audioRequirementText}</p>
              </div>
            )}
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">
                {capability.requires_transcript ? t('audio_corresponding_text_req') : t('audio_corresponding_text')}
              </label>
              <textarea value={form.transcript} onChange={e => setF({ transcript: e.target.value })} rows={4}
                placeholder={capability.requires_transcript ? t('provider_requires_audio_text') : t('optional_submit')}
                className="dark:bg-input/30 border-input rounded-md border bg-transparent px-2.5 py-2 text-sm resize-none focus-visible:outline-none" />
              <p className="text-xs text-[var(--color-text-tertiary)]">
                {t('text_char_require', { min: capability.min_text_len || 0, max: capability.max_text_len || 4000 })}
              </p>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('text_language')}</label>
              <Select value={form.transcript_lang} onValueChange={v => setF({ transcript_lang: v })}>
                <SelectTrigger className="w-[220px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="zh-CN">{t('chinese_lang_option')}</SelectItem>
                  <SelectItem value="en-US">{t('english_lang_option')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowCreate(false); recording.reset() }}>{t('cancel')}</Button>
            <Button disabled={submitting} onClick={() => submitClone()}>{t('submit_clone')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit dialog */}
      <Dialog open={!!editTarget} onOpenChange={v => { if (!v) setEditTarget(null) }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>{t('edit_clone_voice')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('name')}</label>
              <Input value={editName} onChange={e => setEditName(e.target.value)} maxLength={100} />
            </div>
            {editTarget && <>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('provider')}</label>
                <Input value={editTarget.provider || '-'} readOnly className="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('tts_config_label')}</label>
                <Input value={`${editTarget.tts_config_name || '-'} (${editTarget.tts_config_id || '-'})`} readOnly className="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('clone_voice_id')}</label>
                <Input value={editTarget.provider_voice_id || '-'} readOnly className="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('task_status')}</label>
                <Input value={statusLabel(editTarget, t)} readOnly className="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('created_at')}</label>
                <Input value={editTarget.created_at ? new Date(editTarget.created_at).toLocaleString() : '-'} readOnly className="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
              </div>
              {lastError(editTarget) !== '-' && (
                <div className="grid gap-1.5">
                  <label className="text-sm font-medium text-[var(--color-text-secondary)]">{t('failure_reason')}</label>
                  <textarea value={lastError(editTarget)} readOnly rows={3} className="dark:bg-input/30 border-input rounded-md border bg-[var(--color-surface-muted)] px-2.5 py-2 text-sm text-[var(--color-text-secondary)] resize-none focus-visible:outline-none" />
                </div>
              )}
            </>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>{t('cancel')}</Button>
            <Button disabled={editSaving} onClick={saveEdit}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Audio list dialog */}
      <Dialog open={!!audioListTarget} onOpenChange={v => { if (!v) { setAudioListTarget(null); setAudioList([]) } }}>
        <DialogContent className="max-w-[720px] max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('clone_original_audio')}</DialogTitle></DialogHeader>
          <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-24">{t('source_label')}</TableHead>
                  <TableHead>{t('filename_label')}</TableHead>
                  <TableHead>{t('corresponded_text')}</TableHead>
                  <TableHead className="w-24 text-center">{t('play')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {audioListLoading ? (
                  <TableRow><TableCell colSpan={4} className="py-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</TableCell></TableRow>
                ) : audioList.length === 0 ? (
                  <TableRow><TableCell colSpan={4} className="py-6 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
                ) : audioList.map(audio => (
                  <TableRow key={audio.id}>
                    <TableCell className="text-sm text-[var(--color-text-secondary)]">{audio.source_type}</TableCell>
                    <TableCell className="text-sm">{audio.file_name}</TableCell>
                    <TableCell className="text-sm text-[var(--color-text-secondary)] max-w-[240px] truncate" title={audio.transcript}>{audio.transcript || '-'}</TableCell>
                    <TableCell className="text-center">
                      <Button variant="ghost" size="sm" onClick={() => playAudioFile(audio)}>{t('play')}</Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </DialogContent>
      </Dialog>

      {/* Audio preview player */}
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

      {/* Charge confirmation dialog */}
      {chargeConfirm && (
        <ConfirmDialog
          open={!!chargeConfirm}
          onClose={() => setChargeConfirm(null)}
          onConfirm={chargeConfirm.onConfirm}
          title={t('create_clone_reminder')}
          description={chargeConfirm.message}
          confirmLabel={t('i_understand_continue')}
          confirmVariant="default"
        />
      )}

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_clone_voice', { name: deleteTarget?.name || '' })} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/voice-clones')({
  component: VoiceClonesPage,
})
