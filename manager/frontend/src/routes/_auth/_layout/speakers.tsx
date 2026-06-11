import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { MoreHorizontal, Plus } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'
import { SpeakerSamplesDialog } from '@/components/speakers/speaker-samples-dialog'

interface Group { id: number; name: string; description?: string; agent_id?: number; agent_name?: string; tts_config_id?: string; voice?: string; sample_count?: number; created_at?: string }
interface Agent { id: number; name: string }
interface TtsConfig { id: number; name: string; config_id: string; provider: string; enabled: boolean }
interface VoiceOption { value: string; label: string }

const defForm = () => ({ name: '', description: '', agent_id: '', tts_config_id: '', voice: '' })

function SpeakersPage() {
  const { t } = useLocale()
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [filterAgent, setFilterAgent] = useState('')

  const [agents, setAgents] = useState<Agent[]>([])
  const [ttsConfigs, setTtsConfigs] = useState<TtsConfig[]>([])
  const [voices, setVoices] = useState<VoiceOption[]>([])
  const [voiceLoading, setVoiceLoading] = useState(false)

  const [showDialog, setShowDialog] = useState(false)
  const [editing, setEditing] = useState<Group | null>(null)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState(defForm())

  const [deleteTarget, setDeleteTarget] = useState<Group | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [samplesGroup, setSamplesGroup] = useState<Group | null>(null)

  const setF = (p: Partial<typeof form>) => setForm(f => ({ ...f, ...p }))

  const load = async () => {
    setLoading(true)
    try {
      const params: Record<string, string> = {}
      if (filterAgent) params.agent_id = filterAgent
      if (search) params.keyword = search
      const res = await api.get('/user/speaker-groups', { params })
      setGroups(res.data.data || [])
    } catch { toast.error(t('load_failed')) }
    finally { setLoading(false) }
  }

  const loadMeta = async () => {
    try {
      const [aRes, tRes] = await Promise.all([api.get('/user/agents'), api.get('/user/tts-configs')])
      setAgents(aRes.data.data || [])
      setTtsConfigs(tRes.data.data || [])
    } catch {}
  }

  useEffect(() => { load(); loadMeta() }, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [filterAgent]) // eslint-disable-line react-hooks/exhaustive-deps

  const loadVoices = async (configId: string) => {
    if (!configId) { setVoices([]); return }
    const cfg = ttsConfigs.find(c => c.config_id === configId)
    if (!cfg?.provider) { setVoices([]); return }
    setVoiceLoading(true)
    try { const res = await api.get('/user/voice-options', { params: { provider: cfg.provider, config_id: configId } }); setVoices(res.data.data || []) }
    catch { setVoices([]) }
    finally { setVoiceLoading(false) }
  }

  const openCreate = () => { setEditing(null); setForm(defForm()); setVoices([]); setShowDialog(true) }
  const openEdit = (g: Group) => {
    setEditing(g)
    setForm({ name: g.name, description: g.description || '', agent_id: g.agent_id ? String(g.agent_id) : '', tts_config_id: g.tts_config_id || '', voice: g.voice || '' })
    if (g.tts_config_id) loadVoices(g.tts_config_id)
    else setVoices([])
    setShowDialog(true)
  }
  const closeDialog = () => { setShowDialog(false); setEditing(null); setVoices([]) }

  const handleSave = async () => {
    if (!form.name.trim()) { toast.error(t('enter_voiceprint_name')); return }
    if (!form.agent_id) { toast.error(t('select_linked_agent')); return }
    setSaving(true)
    try {
      const payload = { name: form.name, description: form.description, agent_id: Number(form.agent_id), tts_config_id: form.tts_config_id || null, voice: form.voice || null }
      if (editing) { await api.put(`/user/speaker-groups/${editing.id}`, payload); toast.success(t('update_success')) }
      else { await api.post('/user/speaker-groups', payload); toast.success(t('create_success')) }
      closeDialog(); await load()
    } catch (e) { toast.error((e as Error).message) }
    finally { setSaving(false) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try { await api.delete(`/user/speaker-groups/${deleteTarget.id}`); toast.success(t('delete_success')); setDeleteTarget(null); await load() }
    catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  const filtered = search ? groups.filter(g => g.name.toLowerCase().includes(search.toLowerCase()) || (g.agent_name || '').toLowerCase().includes(search.toLowerCase())) : groups

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('voiceprint_management')} />
      <div className="flex items-center justify-between gap-3 flex-wrap px-0 py-2 rounded-xl">
        <div className="flex items-center gap-2 flex-wrap">
          <Input value={search} onChange={e => setSearch(e.target.value)} placeholder={t('search_speaker')} className="w-52" />
          <Select value={filterAgent || '__all__'} onValueChange={(v) => setFilterAgent(v === '__all__' ? '' : v)}>
            <SelectTrigger className="w-44"><SelectValue placeholder={t('all_agents')} /></SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{t('all_agents')}</SelectItem>
              {agents.map(a => <SelectItem key={a.id} value={String(a.id)}>{a.name}</SelectItem>)}
            </SelectContent>
          </Select>
          {groups.length > 0 && (
            <span className="text-xs font-mono font-semibold text-[var(--color-text-tertiary)] px-2 py-0.5 rounded border border-[var(--color-line)] bg-[var(--color-surface-2)]">
              {filtered.length} / {groups.length}
            </span>
          )}
        </div>
        <Button onClick={openCreate}><Plus className="w-4 h-4 mr-1.5" />{t('add_voiceprint_group')}</Button>
      </div>

      <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('name')}</TableHead>
              <TableHead>{t('description')}</TableHead>
              <TableHead>{t('linked_agent')}</TableHead>
              <TableHead>{t('tts_config_label')}</TableHead>
              <TableHead className="w-20 text-center">{t('sample_count')}</TableHead>
              <TableHead className="w-40">{t('created_at')}</TableHead>
              <TableHead className="w-16 text-center">{t('actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow><TableCell colSpan={7} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</TableCell></TableRow>
            ) : filtered.length === 0 ? (
              <TableRow><TableCell colSpan={7} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
            ) : filtered.map(g => (
              <TableRow key={g.id}>
                <TableCell className="font-medium">{g.name}</TableCell>
                <TableCell className="text-sm text-[var(--color-text-secondary)] max-w-[160px] truncate">{g.description || '-'}</TableCell>
                <TableCell className="text-sm">{g.agent_name || '-'}</TableCell>
                <TableCell className="text-sm text-[var(--color-text-secondary)]">{g.tts_config_id || t('default')}</TableCell>
                <TableCell className="text-center text-sm">{g.sample_count ?? 0}</TableCell>
                <TableCell className="text-xs text-[var(--color-text-secondary)]">{g.created_at ? new Date(g.created_at).toLocaleString() : '-'}</TableCell>
                <TableCell className="text-center">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild><Button variant="ghost" size="icon" className="h-8 w-8"><MoreHorizontal className="h-4 w-4" /></Button></DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => setSamplesGroup(g)}>{t('view_samples')}</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => openEdit(g)}>{t('edit')}</DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(g)}>{t('delete')}</DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={showDialog} onOpenChange={v => { if (!v) closeDialog() }}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{editing ? t('edit_voiceprint_group') : t('add_voiceprint_group')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('linked_agent')} <span className="text-destructive">*</span></label>
              <Select value={form.agent_id} onValueChange={v => setF({ agent_id: v })}>
                <SelectTrigger><SelectValue placeholder={t('select_linked_agent')} /></SelectTrigger>
                <SelectContent>{agents.map(a => <SelectItem key={a.id} value={String(a.id)}>{a.name}</SelectItem>)}</SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('voiceprint_name')} <span className="text-destructive">*</span></label><Input value={form.name} onChange={e => setF({ name: e.target.value })} maxLength={100} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('description')}</label><Input value={form.description} onChange={e => setF({ description: e.target.value })} /></div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('tts_config_label')}</label>
              <Select value={form.tts_config_id || '__none__'} onValueChange={v => { const val = v === '__none__' ? '' : v; setF({ tts_config_id: val, voice: '' }); loadVoices(val) }}>
                <SelectTrigger><SelectValue placeholder={t('select_tts_config_opt')} /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">{t('default')}</SelectItem>
                  {ttsConfigs.map(c => <SelectItem key={c.id} value={c.config_id} disabled={!c.enabled}>{c.name} ({c.config_id})</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            {form.tts_config_id && (
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold">{t('voice_timbre')}</label>
                <Input value={form.voice} onChange={e => setF({ voice: e.target.value })} list="speaker-voice-list" placeholder={t('select_or_enter_voice_custom')} disabled={voiceLoading} />
                <datalist id="speaker-voice-list">{voices.map(v => <option key={v.value} value={v.value}>{v.label}</option>)}</datalist>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog}>{t('cancel')}</Button>
            <Button disabled={saving} onClick={handleSave}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SpeakerSamplesDialog group={samplesGroup} open={!!samplesGroup} onClose={() => { setSamplesGroup(null); load() }} />
      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_speaker_group', { name: deleteTarget?.name || '' })} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/speakers')({
  component: SpeakersPage,
})
