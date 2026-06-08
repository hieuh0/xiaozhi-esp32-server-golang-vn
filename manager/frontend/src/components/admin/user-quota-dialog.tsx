import { useState } from 'react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

interface QuotaRow {
  tts_config_id: string; tts_config_name: string; provider: string
  used_count: number; remaining_count: number; max_count: number
}

interface Props {
  open: boolean
  user: { id: number; username: string } | null
  onClose: () => void
}

export function UserQuotaDialog({ open, user, onClose }: Props) {
  const { t } = useLocale()
  const [rows, setRows] = useState<QuotaRow[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [original, setOriginal] = useState<Record<string, number>>({})

  const load = async (id: number) => {
    setLoading(true)
    try {
      const res = await api.get(`/admin/users/${id}/voice-clone-quotas`)
      const quotas: QuotaRow[] = (res.data?.data?.quotas || []).map((q: QuotaRow) => ({
        ...q,
        max_count: Number.isFinite(Number(q.max_count)) ? Number(q.max_count) : -1,
        used_count: Number(q.used_count || 0),
        remaining_count: Number.isFinite(Number(q.remaining_count)) ? Number(q.remaining_count) : -1,
      }))
      setRows(quotas)
      setOriginal(quotas.reduce<Record<string, number>>((acc, r) => { acc[r.tts_config_id] = r.max_count; return acc }, {}))
    } catch { toast.error(t('load_clone_quota_failed')); setRows([]) }
    finally { setLoading(false) }
  }

  const handleOpen = (isOpen: boolean) => {
    if (isOpen && user) load(user.id)
    else { setRows([]); setOriginal({}) }
    if (!isOpen) onClose()
  }

  const setMax = (configId: string, val: number) =>
    setRows(prev => prev.map(r => r.tts_config_id === configId ? { ...r, max_count: val } : r))

  const save = async () => {
    if (!user) return
    const changed = rows.filter(r => original[r.tts_config_id] !== r.max_count)
    if (!changed.length) { toast.info(t('quota_unchanged')); return }
    setSaving(true)
    try {
      await api.put(`/admin/users/${user.id}/voice-clone-quotas`, { items: changed.map(r => ({ tts_config_id: r.tts_config_id, max_count: r.max_count })) })
      toast.success(t('clone_quota_save_success'))
      onClose()
    } catch { toast.error(t('save_clone_quota_failed')) }
    finally { setSaving(false) }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpen}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>{t('voice_clone_quota_title', { name: user?.username || '' })}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-[var(--color-text-secondary)]">{t('quota_hint')}</p>
        <div className="rounded-xl border border-[var(--color-line)] overflow-hidden mt-2">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('tts_config_name_col')}</TableHead>
                <TableHead>TTS Config ID</TableHead>
                <TableHead className="w-28">Provider</TableHead>
                <TableHead className="w-24">{t('used_count_col')}</TableHead>
                <TableHead className="w-32">{t('remaining_count_col')}</TableHead>
                <TableHead className="w-40">{t('max_count_col')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow><TableCell colSpan={6} className="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell></TableRow>
              ) : rows.length === 0 ? (
                <TableRow><TableCell colSpan={6} className="py-8 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
              ) : rows.map(r => (
                <TableRow key={r.tts_config_id}>
                  <TableCell className="font-medium">{r.tts_config_name}</TableCell>
                  <TableCell className="font-mono text-xs text-[var(--color-text-secondary)]">{r.tts_config_id}</TableCell>
                  <TableCell>{r.provider}</TableCell>
                  <TableCell>{r.used_count}</TableCell>
                  <TableCell>{r.remaining_count < 0 ? t('unlimited') : r.remaining_count}</TableCell>
                  <TableCell>
                    <Input type="number" value={r.max_count} onChange={e => setMax(r.tts_config_id, Number(e.target.value))} min={-1} step={1} className="w-28" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>{t('cancel')}</Button>
          <Button disabled={saving} onClick={save}>{t('save_quota')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
