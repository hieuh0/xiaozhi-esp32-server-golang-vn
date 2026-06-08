import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { MoreHorizontal, Plus } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'

interface Role { id: number; name: string; description?: string; prompt?: string; llm_config_id?: string | null; tts_config_id?: string | null; voice?: string; status: string; sort_order: number; is_default: boolean }
interface Config { id: number; name: string; config_id: string; provider: string; enabled: boolean; is_default: boolean }

const defForm = () => ({ name: '', description: '', prompt: '', llm_config_id: '', tts_config_id: '', voice: '', status: 'active', sort_order: 0, is_default: false })

function GlobalRolesPage() {
  const { t } = useLocale()
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editing, setEditing] = useState<Role | null>(null)
  const [showDialog, setShowDialog] = useState(false)
  const [form, setForm] = useState(defForm())
  const [llmConfigs, setLlmConfigs] = useState<Config[]>([])
  const [ttsConfigs, setTtsConfigs] = useState<Config[]>([])
  const [deleteTarget, setDeleteTarget] = useState<Role | null>(null)
  const [deleting, setDeleting] = useState(false)

  const setF = (p: Partial<typeof form>) => setForm(f => ({ ...f, ...p }))
  const isActive = (r: Role) => r.status !== 'inactive'

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/roles/global')
      setRoles(res.data.data || [])
    } catch { toast.error(t('load_roles_failed')) }
    finally { setLoading(false) }
  }

  const loadConfigs = async () => {
    try {
      const [llm, tts] = await Promise.all([api.get('/admin/llm-configs'), api.get('/admin/tts-configs')])
      setLlmConfigs(llm.data.data || []); setTtsConfigs(tts.data.data || [])
    } catch { /* non-critical */ }
  }

  useEffect(() => { load(); loadConfigs() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openCreate = () => { setEditing(null); setForm(defForm()); setShowDialog(true) }
  const openEdit = (r: Role) => {
    setEditing(r)
    setForm({ name: r.name, description: r.description || '', prompt: r.prompt || '', llm_config_id: r.llm_config_id || '', tts_config_id: r.tts_config_id || '', voice: r.voice || '', status: r.status || 'active', sort_order: r.sort_order || 0, is_default: r.is_default || false })
    setShowDialog(true)
  }
  const openDuplicate = (r: Role) => {
    setEditing(null)
    setForm({ name: `${r.name} ${t('duplicate_suffix')}`, description: r.description || '', prompt: r.prompt || '', llm_config_id: r.llm_config_id || '', tts_config_id: r.tts_config_id || '', voice: r.voice || '', status: r.status || 'active', sort_order: r.sort_order || 0, is_default: false })
    setShowDialog(true)
  }

  const handleSave = async () => {
    if (!form.name.trim()) { toast.error(t('enter_role_name')); return }
    if (!form.prompt.trim()) { toast.error(t('enter_system_prompt')); return }
    setSaving(true)
    try {
      const payload = { ...form, llm_config_id: form.llm_config_id || null, tts_config_id: form.tts_config_id || null }
      if (editing) { await api.put(`/admin/roles/global/${editing.id}`, payload); toast.success(t('update_success')) }
      else { await api.post('/admin/roles/global', payload); toast.success(t('create_success')) }
      setShowDialog(false); await load()
    } catch (e) { toast.error(t('save_failed_colon') + (e as Error).message) }
    finally { setSaving(false) }
  }

  const toggleStatus = async (r: Role) => {
    try { await api.patch(`/admin/roles/global/${r.id}/toggle`); toast.success(t('role_action_success', { action: isActive(r) ? t('close') : t('enable') })); await load() }
    catch (e) { toast.error((e as Error).message) }
  }

  const setDefault = async (r: Role) => {
    if (r.is_default) return
    try { await api.patch(`/admin/roles/global/${r.id}/default`); toast.success(t('set_as_default_role')); await load() }
    catch (e) { toast.error((e as Error).message) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try { await api.delete(`/admin/roles/global/${deleteTarget.id}`); toast.success(t('delete_success')); setDeleteTarget(null); await load() }
    catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  const badge = (cls: string, text: string) => <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${cls}`}>{text}</span>

  return (
    <div className="px-6 pb-8">
      <PageHeader title={t('global_roles')} />
      <div className="flex justify-end mb-5">
        <Button onClick={openCreate}><Plus className="w-4 h-4 mr-1.5" />{t('create_global_role')}</Button>
      </div>

      {loading && <div className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>}
      {!loading && roles.length === 0 && (
        <div className="flex flex-col items-center gap-3 py-16 text-[var(--color-text-secondary)]">
          <p className="text-sm">{t('no_global_role')}</p>
          <Button onClick={openCreate}>{t('create_first_global_role')}</Button>
        </div>
      )}
      {!loading && roles.length > 0 && (
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 340px))' }}>
          {roles.map(r => (
            <div key={r.id} className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:-translate-y-0.5 transition-transform duration-200">
              <div className="flex items-center justify-between gap-3 px-4 pt-4 pb-3 border-b border-[var(--color-line)]">
                <span className="font-bold text-[15px] text-[var(--color-text)] truncate">{r.name}</span>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="flex items-center justify-center w-7 h-7 rounded-full text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] transition-colors cursor-pointer">
                      <MoreHorizontal className="w-4 h-4" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => openEdit(r)}>{t('edit')}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => openDuplicate(r)}>{t('copy')}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => toggleStatus(r)}>{isActive(r) ? t('close') : t('enable')}</DropdownMenuItem>
                    <DropdownMenuItem disabled={r.is_default} onClick={() => setDefault(r)}>{r.is_default ? t('set_as_default_done') : t('set_as_default')}</DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(r)}>{t('delete')}</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <div className="px-4 py-3 flex flex-col gap-2.5 min-h-[170px]">
                <p className="text-sm text-[var(--color-text-secondary)] line-clamp-2">{r.description || t('no_description_alt')}</p>
                <div className="flex flex-wrap gap-1.5">
                  {badge(isActive(r) ? 'status-success' : 'status-muted', isActive(r) ? t('enable') : t('close'))}
                  {r.is_default && badge('status-warning', t('default_role_tag'))}
                  {badge('status-primary', `LLM: ${r.llm_config_id || t('default')}`)}
                  {badge('status-success', `TTS: ${r.tts_config_id || t('default')}`)}
                  {r.voice && badge('status-warning', t('voice_tag_prefix', { voice: r.voice }))}
                </div>
                <div className="mt-auto border border-[var(--color-line)] bg-[var(--color-surface-muted)] rounded-lg px-2.5 py-2">
                  <p className="mb-1 text-[11px] font-bold uppercase tracking-wide text-[var(--color-text-tertiary)]">{t('prompt')}</p>
                  <p className="text-xs text-[var(--color-text)] line-clamp-3">{r.prompt || t('prompt_not_set')}</p>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={showDialog} onOpenChange={v => { if (!v) setShowDialog(false) }}>
        <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto" aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{editing ? t('edit_global_role') : t('create_global_role')}</DialogTitle></DialogHeader>
          <div className="grid gap-5 py-2">
            <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('basic_info')}</p>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('role_name')}</label><Input value={form.name} onChange={e => setF({ name: e.target.value })} placeholder={t('enter_role_name')} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('description')}</label><Textarea value={form.description} onChange={e => setF({ description: e.target.value })} rows={3} placeholder={t('enter_role_description')} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('sort_order')}</label><Input type="number" value={form.sort_order} onChange={e => setF({ sort_order: Number(e.target.value) })} min={0} className="w-32" /></div>
            <div className="flex items-center justify-between gap-4"><label className="text-sm font-semibold">{t('default_role_tag')}</label><Switch checked={form.is_default} onCheckedChange={v => setF({ is_default: v })} /></div>
            <div className="border-t border-[var(--color-line)]" />
            <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('prompt_config_section')}</p>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('system_prompt_label')}</label><Textarea value={form.prompt} onChange={e => setF({ prompt: e.target.value })} rows={6} placeholder={t('enter_system_prompt')} /></div>
            <div className="border-t border-[var(--color-line)]" />
            <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('model_config_section')}</p>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('llm_config_label')}</label>
              <Select value={form.llm_config_id || '__none__'} onValueChange={v => setF({ llm_config_id: v === '__none__' ? '' : v })}>
                <SelectTrigger><SelectValue placeholder={t('select_llm_config_opt')} /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">{t('default')}</SelectItem>
                  {llmConfigs.map(c => <SelectItem key={c.id} value={c.config_id} disabled={!c.enabled}>{c.name} ({c.config_id}){c.is_default ? ` · ${t('default')}` : ''}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('tts_config_label')}</label>
              <Select value={form.tts_config_id || '__none__'} onValueChange={v => setF({ tts_config_id: v === '__none__' ? '' : v, voice: '' })}>
                <SelectTrigger><SelectValue placeholder={t('select_tts_config_opt')} /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">{t('default')}</SelectItem>
                  {ttsConfigs.map(c => <SelectItem key={c.id} value={c.config_id} disabled={!c.enabled}>{c.name} ({c.config_id}){c.is_default ? ` · ${t('default')}` : ''}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            {form.tts_config_id && (
              <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('voice_timbre')}</label><Input value={form.voice} onChange={e => setF({ voice: e.target.value })} placeholder={t('select_or_enter_voice_custom')} /></div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDialog(false)}>{t('cancel')}</Button>
            <Button disabled={saving} onClick={handleSave}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_global_role')} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/global-roles')({
  component: GlobalRolesPage,
})
