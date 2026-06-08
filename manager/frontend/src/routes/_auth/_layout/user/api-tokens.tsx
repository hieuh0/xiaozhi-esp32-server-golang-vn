import { useEffect, useState } from 'react'
import { createFileRoute, Link } from '@tanstack/react-router'
import { Copy, Plus } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'

interface Token { id: number; name: string; token_prefix: string; is_active: boolean; last_used_at: string; expires_at: string; created_at: string }

const fmt = (v: string) => v ? new Date(v).toLocaleString() : '-'
const activeBadge = 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
const inactiveBadge = 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'

function ApiTokensPage() {
  const { t } = useLocale()
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const pageSize = 20

  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState({ name: '', expires_in_days: 0 })

  const [showToken, setShowToken] = useState(false)
  const [latestToken, setLatestToken] = useState('')
  const [revokeTarget, setRevokeTarget] = useState<Token | null>(null)
  const [revoking, setRevoking] = useState(false)

  const totalPages = Math.ceil(total / pageSize) || 1

  const load = async (p = page) => {
    setLoading(true)
    try {
      const res = await api.get('/user/api-tokens', { params: { page: p, page_size: pageSize } })
      setTokens(res.data.data || [])
      setTotal(res.data.total || 0)
    } catch { toast.error(t('load_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load(1) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleCreate = async () => {
    if (!form.name.trim()) { toast.error(t('enter_token_name')); return }
    setCreating(true)
    try {
      const res = await api.post('/user/api-tokens', form)
      setLatestToken(res.data?.data?.token || '')
      setShowCreate(false)
      setShowToken(true)
      toast.success(t('token_created'))
      await load(page)
    } catch { toast.error(t('create_failed')) }
    finally { setCreating(false) }
  }

  const handleRevoke = async () => {
    if (!revokeTarget) return
    setRevoking(true)
    try {
      await api.delete(`/user/api-tokens/${revokeTarget.id}`)
      toast.success(t('token_revoked'))
      setRevokeTarget(null)
      await load(page)
    } catch { toast.error(t('revoke_failed')) }
    finally { setRevoking(false) }
  }

  const copyToken = async () => {
    await navigator.clipboard.writeText(latestToken)
    toast.success(t('token_copied'))
  }

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('api_token_management')} />
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <Link to="/openapi-docs" className="text-sm text-[var(--color-primary)] hover:underline">{t('view_public_openapi')}</Link>
        <Button onClick={() => { setForm({ name: '', expires_in_days: 0 }); setShowCreate(true) }}>
          <Plus className="w-4 h-4 mr-1.5" />{t('create_token')}
        </Button>
      </div>
      <div className="rounded-lg border border-blue-200 bg-blue-50 text-blue-800 p-3 text-sm dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-300">
        {t('call_method_hint')}
      </div>
      <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('name')}</TableHead>
              <TableHead>{t('prefix_col')}</TableHead>
              <TableHead className="w-24">{t('status')}</TableHead>
              <TableHead>{t('last_used_col')}</TableHead>
              <TableHead>{t('expire_time_col')}</TableHead>
              <TableHead>{t('created_at')}</TableHead>
              <TableHead className="w-20 text-center">{t('actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow><TableCell colSpan={7} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</TableCell></TableRow>
            ) : tokens.length === 0 ? (
              <TableRow><TableCell colSpan={7} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
            ) : tokens.map(row => (
              <TableRow key={row.id}>
                <TableCell className="font-medium">{row.name}</TableCell>
                <TableCell className="font-mono text-xs">{row.token_prefix}</TableCell>
                <TableCell><span className={row.is_active ? activeBadge : inactiveBadge}>{row.is_active ? t('available') : t('revoked')}</span></TableCell>
                <TableCell className="text-sm text-[var(--color-text-secondary)]">{fmt(row.last_used_at)}</TableCell>
                <TableCell className="text-sm text-[var(--color-text-secondary)]">{fmt(row.expires_at)}</TableCell>
                <TableCell className="text-sm text-[var(--color-text-secondary)]">{fmt(row.created_at)}</TableCell>
                <TableCell className="text-center">
                  <Button variant="ghost" size="sm" disabled={!row.is_active} className="text-destructive hover:text-destructive" onClick={() => row.is_active && setRevokeTarget(row)}>
                    {t('revoke')}
                  </Button>
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

      <Dialog open={showCreate} onOpenChange={v => { if (!v) setShowCreate(false) }}>
        <DialogContent className="max-w-md">
          <DialogHeader><DialogTitle>{t('create_api_token_title')}</DialogTitle></DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('token_name_label')}</label>
              <Input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} maxLength={100} placeholder={t('token_name_ph')} />
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium">{t('valid_days_label')}</label>
              <Input type="number" value={form.expires_in_days} onChange={e => setForm(f => ({ ...f, expires_in_days: Number(e.target.value) }))} min={0} max={3650} />
              <p className="text-xs text-[var(--color-text-tertiary)]">{t('valid_forever_hint')}</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>{t('cancel')}</Button>
            <Button disabled={creating} onClick={handleCreate}>{t('create')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showToken} onOpenChange={v => { if (!v) setShowToken(false) }}>
        <DialogContent className="max-w-xl">
          <DialogHeader><DialogTitle>{t('save_token_now')}</DialogTitle></DialogHeader>
          <div className="grid gap-3 py-2">
            <div className="rounded-lg border border-yellow-200 bg-yellow-50 text-yellow-800 p-3 text-sm dark:bg-yellow-900/20 dark:border-yellow-800 dark:text-yellow-300">
              {t('plain_token_hint')}
            </div>
            <textarea value={latestToken} readOnly rows={3} className="dark:bg-input/30 border-input rounded-md border bg-transparent px-2.5 py-2 text-sm font-mono w-full resize-none focus-visible:outline-none" />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowToken(false)}>{t('close')}</Button>
            <Button onClick={copyToken}><Copy className="w-4 h-4 mr-1.5" />{t('copy_token')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!revokeTarget} onClose={() => setRevokeTarget(null)} onConfirm={handleRevoke} isLoading={revoking} title={t('hint')} description={t('confirm_revoke_token', { name: revokeTarget?.name || '' })} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/user/api-tokens')({
  component: ApiTokensPage,
})
