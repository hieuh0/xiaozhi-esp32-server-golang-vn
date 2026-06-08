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
import { UserQuotaDialog } from '@/components/admin/user-quota-dialog'

interface User { id: number; username: string; email: string; role: string; created_at: string }

const defaultForm = () => ({ username: '', email: '', password: '', role: '' })
const defaultPwdForm = () => ({ newPassword: '', confirmPassword: '' })

function UsersPage() {
  const { t } = useLocale()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')
  const pageSize = 20

  const [showUser, setShowUser] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [userForm, setUserForm] = useState(defaultForm())
  const [showPwd, setShowPwd] = useState(false)
  const [userSaving, setUserSaving] = useState(false)

  const [showReset, setShowReset] = useState(false)
  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [pwdForm, setPwdForm] = useState(defaultPwdForm())
  const [resetSaving, setResetSaving] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [quotaUser, setQuotaUser] = useState<User | null>(null)

  const setF = (p: Partial<typeof userForm>) => setUserForm(f => ({ ...f, ...p }))
  const setP = (p: Partial<typeof pwdForm>) => setPwdForm(f => ({ ...f, ...p }))

  const load = async (p = page) => {
    setLoading(true)
    try {
      const res = await api.get('/admin/users', { params: { page: p, page_size: pageSize } })
      setUsers(res.data.data || [])
      setTotal(res.data.total || 0)
    } catch { toast.error(t('load_user_list_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load(1) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = search ? users.filter(u => u.username.toLowerCase().includes(search.toLowerCase()) || (u.email || '').toLowerCase().includes(search.toLowerCase())) : users
  const totalPages = Math.ceil(total / pageSize) || 1

  const openAdd = () => { setEditingUser(null); setUserForm(defaultForm()); setShowPwd(false); setShowUser(true) }
  const openEdit = (u: User) => { setEditingUser(u); setUserForm({ username: u.username, email: u.email, password: '', role: u.role }); setShowUser(true) }

  const handleUserSave = async () => {
    if (!editingUser && !userForm.username.trim()) { toast.error(t('enter_username')); return }
    if (!userForm.email.trim() || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(userForm.email)) { toast.error(t('enter_valid_email')); return }
    if (!editingUser && userForm.password.length < 6) { toast.error(t('password_min_length')); return }
    if (!userForm.role) { toast.error(t('select_role')); return }
    setUserSaving(true)
    try {
      if (editingUser) { await api.put(`/admin/users/${editingUser.id}`, { email: userForm.email, role: userForm.role }); toast.success(t('user_update_success')) }
      else { await api.post('/admin/users', { username: userForm.username, email: userForm.email, password: userForm.password, role: userForm.role }); toast.success(t('user_add_success')) }
      setShowUser(false); await load(page)
    } catch { toast.error(editingUser ? t('update_user_failed') : t('add_user_failed')) }
    finally { setUserSaving(false) }
  }

  const handleReset = async () => {
    if (pwdForm.newPassword.length < 6) { toast.error(t('password_min_length')); return }
    if (pwdForm.newPassword !== pwdForm.confirmPassword) { toast.error(t('password_mismatch')); return }
    if (!resetTarget) return
    setResetSaving(true)
    try { await api.post(`/admin/users/${resetTarget.id}/reset-password`, { new_password: pwdForm.newPassword }); toast.success(t('password_reset_success')); setShowReset(false) }
    catch { toast.error(t('reset_password_failed')) }
    finally { setResetSaving(false) }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try { await api.delete(`/admin/users/${deleteTarget.id}`); toast.success(t('user_delete_success')); setDeleteTarget(null); await load(page) }
    catch { toast.error(t('delete_user_failed')) }
    finally { setDeleting(false) }
  }

  const roleBadge = (role: string) => role === 'admin'
    ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
    : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800'

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('user_management')} />
      <div className="flex items-center justify-between gap-4">
        <Input value={search} onChange={e => setSearch(e.target.value)} placeholder={t('search_user')} className="max-w-xs" />
        <Button onClick={openAdd}><Plus className="w-4 h-4 mr-1.5" />{t('add_user')}</Button>
      </div>
      <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">ID</TableHead>
              <TableHead>{t('username')}</TableHead>
              <TableHead>{t('email')}</TableHead>
              <TableHead className="w-28">{t('role')}</TableHead>
              <TableHead className="w-44">{t('created_at')}</TableHead>
              <TableHead className="w-16 text-center">{t('actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? <TableRow><TableCell colSpan={6} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</TableCell></TableRow>
              : filtered.length === 0 ? <TableRow><TableCell colSpan={6} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
              : filtered.map(u => (
                <TableRow key={u.id}>
                  <TableCell className="text-[var(--color-text-secondary)] text-xs font-mono">{u.id}</TableCell>
                  <TableCell className="font-semibold">{u.username}</TableCell>
                  <TableCell className="text-[var(--color-text-secondary)]">{u.email}</TableCell>
                  <TableCell><span className={roleBadge(u.role)}>{u.role === 'admin' ? t('admin') : t('normal_user')}</span></TableCell>
                  <TableCell className="text-[var(--color-text-secondary)] text-sm">{new Date(u.created_at).toLocaleString()}</TableCell>
                  <TableCell className="text-center">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild><Button variant="ghost" size="icon" className="h-8 w-8"><MoreHorizontal className="h-4 w-4" /></Button></DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => openEdit(u)}>{t('edit')}</DropdownMenuItem>
                        <DropdownMenuItem disabled={u.role === 'admin'} onClick={() => u.role !== 'admin' && setQuotaUser(u)}>{t('voice_clone_quota')}</DropdownMenuItem>
                        <DropdownMenuItem onClick={() => { setResetTarget(u); setPwdForm(defaultPwdForm()); setShowReset(true) }}>{t('reset_password_title')}</DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem disabled={u.role === 'admin'} className="text-destructive" onClick={() => u.role !== 'admin' && setDeleteTarget(u)}>{t('delete')}</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
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

      {/* Add/Edit user dialog */}
      <Dialog open={showUser} onOpenChange={v => { if (!v) setShowUser(false) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{editingUser ? t('edit_user') : t('add_user')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('username')}</label>
              <Input value={userForm.username} disabled={!!editingUser} onChange={e => setF({ username: e.target.value })} placeholder={t('enter_username')} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('email')}</label>
              <Input type="email" value={userForm.email} onChange={e => setF({ email: e.target.value })} placeholder={t('enter_email')} />
            </div>
            {!editingUser && (
              <div className="grid gap-1.5">
                <label className="text-sm font-semibold">{t('password')}</label>
                <div className="relative">
                  <Input type={showPwd ? 'text' : 'password'} value={userForm.password} onChange={e => setF({ password: e.target.value })} placeholder={t('enter_password_min6')} className="pr-16" />
                  <button type="button" className="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)]" onClick={() => setShowPwd(v => !v)}>
                    {showPwd ? t('hide') : t('show')}
                  </button>
                </div>
              </div>
            )}
            <div className="grid gap-1.5">
              <label className="text-sm font-semibold">{t('role')}</label>
              <Select value={userForm.role} onValueChange={v => setF({ role: v })}>
                <SelectTrigger><SelectValue placeholder={t('select_role')} /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="user">{t('normal_user')}</SelectItem>
                  <SelectItem value="admin">{t('admin')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowUser(false)}>{t('cancel')}</Button>
            <Button disabled={userSaving} onClick={handleUserSave}>{editingUser ? t('save') : t('add')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reset password dialog */}
      <Dialog open={showReset} onOpenChange={v => { if (!v) setShowReset(false) }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>{t('reset_password_title')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('user')}</label><Input value={resetTarget?.username || ''} disabled /></div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('new_password')}</label><Input type="password" value={pwdForm.newPassword} onChange={e => setP({ newPassword: e.target.value })} placeholder={t('new_password_ph')} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-semibold">{t('confirm_password')}</label><Input type="password" value={pwdForm.confirmPassword} onChange={e => setP({ confirmPassword: e.target.value })} placeholder={t('confirm_password_ph')} /></div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowReset(false)}>{t('cancel')}</Button>
            <Button disabled={resetSaving} onClick={handleReset}>{t('confirm_reset')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_user', { name: deleteTarget?.username || '' })} />
      <UserQuotaDialog open={!!quotaUser} user={quotaUser} onClose={() => setQuotaUser(null)} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/users')({
  component: UsersPage,
})
