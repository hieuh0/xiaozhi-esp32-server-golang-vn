import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'

interface Group { id: number; name: string }
interface Sample { id: number; file_name?: string; duration_sec?: number; created_at?: string }
interface VerifyResult { verified: boolean; score?: number; message?: string }

interface Props { group: Group | null; open: boolean; onClose: () => void }

export function SpeakerSamplesDialog({ group, open, onClose }: Props) {
  const { t } = useLocale()
  const [samples, setSamples] = useState<Sample[]>([])
  const [loading, setLoading] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Sample | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [previewUrl, setPreviewUrl] = useState('')
  const [uploading, setUploading] = useState(false)

  const [showVerify, setShowVerify] = useState(false)
  const [verifyFile, setVerifyFile] = useState<File | null>(null)
  const [verifying, setVerifying] = useState(false)
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null)

  const uploadRef = useRef<HTMLInputElement>(null)
  const verifyRef = useRef<HTMLInputElement>(null)

  const load = async () => {
    if (!group) return
    setLoading(true)
    try { const res = await api.get(`/user/speaker-groups/${group.id}/samples`); setSamples(res.data.data || []) }
    catch { toast.error(t('load_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { if (open && group) load() }, [open, group?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !group) return
    setUploading(true)
    const fd = new FormData(); fd.append('audio_file', file)
    try { await api.post(`/user/speaker-groups/${group.id}/samples`, fd, { timeout: 60000 }); toast.success(t('upload_success')); await load() }
    catch (err) { toast.error((err as { response?: { data?: { error?: string } } }).response?.data?.error || t('upload_failed')) }
    finally { setUploading(false); if (e.target) (e.target as HTMLInputElement).value = '' }
  }

  const playSample = async (s: Sample) => {
    if (!group) return
    try {
      const res = await api.get(`/user/speaker-groups/${group.id}/samples/${s.id}/file`, { responseType: 'blob' })
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      setPreviewUrl(URL.createObjectURL(res.data))
    } catch { toast.error(t('play_failed')) }
  }

  const handleDelete = async () => {
    if (!deleteTarget || !group) return
    setDeleting(true)
    try { await api.delete(`/user/speaker-groups/${group.id}/samples/${deleteTarget.id}`); toast.success(t('delete_success')); setDeleteTarget(null); await load() }
    catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  const runVerify = async () => {
    if (!verifyFile || !group) { toast.warning(t('upload_audio_file')); return }
    setVerifying(true); setVerifyResult(null)
    const fd = new FormData(); fd.append('audio_file', verifyFile)
    try { const res = await api.post(`/user/speaker-groups/${group.id}/verify`, fd, { timeout: 60000 }); setVerifyResult(res.data.data || res.data) }
    catch (err) { toast.error((err as { response?: { data?: { error?: string } } }).response?.data?.error || t('verify_failed')) }
    finally { setVerifying(false) }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={v => { if (!v) { onClose(); if (previewUrl) { URL.revokeObjectURL(previewUrl); setPreviewUrl('') } } }}>
        <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{t('voiceprint_samples_title', { name: group?.name || '' })}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="flex items-center justify-between gap-2 flex-wrap">
              <span className="text-sm text-[var(--color-text-secondary)]">{t('sample_count_info', { count: samples.length })}</span>
              <div className="flex gap-2">
                <Button variant="outline" disabled={uploading} onClick={() => uploadRef.current?.click()}>
                  {uploading ? t('uploading') : t('upload_sample')}
                </Button>
                <input ref={uploadRef} type="file" accept=".wav,.mp3,audio/wav,audio/mpeg" className="hidden" onChange={handleUpload} />
                <Button variant="outline" onClick={() => { setVerifyFile(null); setVerifyResult(null); setShowVerify(true) }}>{t('verify_speaker')}</Button>
              </div>
            </div>

            {previewUrl && (
              <div className="rounded-lg border border-[var(--color-line)] p-3 bg-[var(--color-surface-muted)]">
                <p className="text-xs font-semibold text-[var(--color-text-secondary)] mb-2">{t('audio_preview_title')}</p>
                <audio src={previewUrl} controls autoPlay className="w-full" />
              </div>
            )}

            <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">ID</TableHead>
                    <TableHead>{t('filename_label')}</TableHead>
                    <TableHead className="w-28">{t('duration_col')}</TableHead>
                    <TableHead className="w-40">{t('created_at')}</TableHead>
                    <TableHead className="w-32 text-center">{t('actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loading ? (
                    <TableRow><TableCell colSpan={5} className="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell></TableRow>
                  ) : samples.length === 0 ? (
                    <TableRow><TableCell colSpan={5} className="py-8 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
                  ) : samples.map(s => (
                    <TableRow key={s.id}>
                      <TableCell className="text-xs font-mono text-[var(--color-text-secondary)]">{s.id}</TableCell>
                      <TableCell className="text-sm">{s.file_name || '-'}</TableCell>
                      <TableCell className="text-sm text-[var(--color-text-secondary)]">{s.duration_sec != null ? `${Number(s.duration_sec).toFixed(1)}s` : '-'}</TableCell>
                      <TableCell className="text-xs text-[var(--color-text-secondary)]">{s.created_at ? new Date(s.created_at).toLocaleString() : '-'}</TableCell>
                      <TableCell className="text-center">
                        <div className="flex justify-center gap-1">
                          <Button variant="ghost" size="sm" onClick={() => playSample(s)}>{t('play')}</Button>
                          <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setDeleteTarget(s)}>{t('delete')}</Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Verify dialog */}
      <Dialog open={showVerify} onOpenChange={v => { if (!v) setShowVerify(false) }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>{t('verify_speaker')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <p className="text-sm text-[var(--color-text-secondary)]">{t('verify_speaker_hint')}</p>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('audio_file_label')}</label>
              <input ref={verifyRef} type="file" accept=".wav,.mp3,audio/wav,audio/mpeg" className="text-sm" onChange={e => setVerifyFile(e.target.files?.[0] || null)} />
            </div>
            {verifyResult && (
              <div className={`rounded-lg border p-3 text-sm ${verifyResult.verified ? 'status-success' : 'status-danger'}`}>
                <p className="font-semibold">{verifyResult.verified ? t('verify_passed') : t('verify_failed_result')}</p>
                {verifyResult.score != null && <p className="text-xs mt-1">{t('verify_score', { score: Number(verifyResult.score).toFixed(4) })}</p>}
                {verifyResult.message && <p className="text-xs mt-1">{verifyResult.message}</p>}
              </div>
            )}
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setShowVerify(false)}>{t('close')}</Button>
            <Button disabled={verifying || !verifyFile} onClick={runVerify}>{verifying ? t('verifying') : t('start_verify')}</Button>
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_sample')} />
    </>
  )
}
