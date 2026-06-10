import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { MoreHorizontal, Plus, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { PageHeader } from '@/components/ui/page-header'

interface MarketRow { id: number; name: string; provider_id: string; catalog_url: string; enabled: boolean; created_at: string }
interface ImportedRow { id: number; name: string; type: string; url: string; enabled: boolean; created_at: string }

interface MarketForm { name: string; provider_id: string; catalog_url: string; detail_url_template: string; enabled: boolean; auth_token: string }
interface ImportedForm { name: string; type: string; url: string; description: string; enabled: boolean }

const defaultMarket = (): MarketForm => ({ name: '', provider_id: 'modelscope', catalog_url: '', detail_url_template: '', enabled: true, auth_token: '' })
const defaultImported = (): ImportedForm => ({ name: '', type: 'streamablehttp', url: '', description: '', enabled: true })

function McpMarketPage() {
  const { t } = useLocale()

  const [markets, setMarkets] = useState<MarketRow[]>([])
  const [marketsLoading, setMarketsLoading] = useState(false)
  const [showMarketDialog, setShowMarketDialog] = useState(false)
  const [editingMarket, setEditingMarket] = useState<MarketRow | null>(null)
  const [marketForm, setMarketForm] = useState<MarketForm>(defaultMarket())
  const [marketSaving, setMarketSaving] = useState(false)
  const [deleteMarket, setDeleteMarket] = useState<MarketRow | null>(null)

  const [imported, setImported] = useState<ImportedRow[]>([])
  const [importedLoading, setImportedLoading] = useState(false)
  const [showImportedDialog, setShowImportedDialog] = useState(false)
  const [editingImported, setEditingImported] = useState<ImportedRow | null>(null)
  const [importedForm, setImportedForm] = useState<ImportedForm>(defaultImported())
  const [importedSaving, setImportedSaving] = useState(false)
  const [deleteImported, setDeleteImported] = useState<ImportedRow | null>(null)
  const [deleting, setDeleting] = useState(false)

  const setMF = (patch: Partial<MarketForm>) => setMarketForm((f) => ({ ...f, ...patch }))
  const setIF = (patch: Partial<ImportedForm>) => setImportedForm((f) => ({ ...f, ...patch }))

  const loadMarkets = async () => {
    setMarketsLoading(true)
    try { const r = await api.get('/admin/mcp-markets'); setMarkets(Array.isArray(r.data?.data) ? r.data.data : []) }
    catch { toast.error(t('load_config_failed')) }
    finally { setMarketsLoading(false) }
  }

  const loadImported = async () => {
    setImportedLoading(true)
    try { const r = await api.get('/admin/mcp-market/imported-services'); setImported(Array.isArray(r.data?.data) ? r.data.data : []) }
    catch { toast.error(t('load_config_failed')) }
    finally { setImportedLoading(false) }
  }

  useEffect(() => { loadMarkets(); loadImported() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openAddMarket = () => { setEditingMarket(null); setMarketForm(defaultMarket()); setShowMarketDialog(true) }
  const openEditMarket = (row: MarketRow) => { setEditingMarket(row); setMarketForm({ name: row.name, provider_id: row.provider_id, catalog_url: row.catalog_url, detail_url_template: '', enabled: row.enabled, auth_token: '' }); setShowMarketDialog(true) }

  const saveMarket = async () => {
    if (!marketForm.name.trim()) { toast.error(t('enter_config_name')); return }
    setMarketSaving(true)
    try {
      const payload = { name: marketForm.name, provider_id: marketForm.provider_id, catalog_url: marketForm.catalog_url, detail_url_template: marketForm.detail_url_template, enabled: marketForm.enabled, auth: marketForm.auth_token ? { type: 'bearer', token: marketForm.auth_token, header_name: 'Authorization' } : undefined }
      if (editingMarket) await api.put(`/admin/mcp-markets/${editingMarket.id}`, payload)
      else await api.post('/admin/mcp-markets', payload)
      toast.success(t('save_success')); setShowMarketDialog(false); await loadMarkets()
    } catch (e) { toast.error((e as Error).message || t('save_failed')) }
    finally { setMarketSaving(false) }
  }

  const openAddImported = () => { setEditingImported(null); setImportedForm(defaultImported()); setShowImportedDialog(true) }
  const openEditImported = (row: ImportedRow) => { setEditingImported(row); setImportedForm({ name: row.name, type: row.type, url: row.url, description: '', enabled: row.enabled }); setShowImportedDialog(true) }

  const saveImported = async () => {
    if (!importedForm.name.trim()) { toast.error(t('enter_config_name')); return }
    setImportedSaving(true)
    try {
      if (editingImported) await api.put(`/admin/mcp-market/imported-services/${editingImported.id}`, importedForm)
      else await api.post('/admin/mcp-market/imported-services', importedForm)
      toast.success(t('save_success')); setShowImportedDialog(false); await loadImported()
    } catch (e) { toast.error((e as Error).message || t('save_failed')) }
    finally { setImportedSaving(false) }
  }

  const handleDelete = async () => {
    const target = deleteMarket || deleteImported
    if (!target) return
    setDeleting(true)
    try {
      if (deleteMarket) { await api.delete(`/admin/mcp-markets/${target.id}`); setDeleteMarket(null); await loadMarkets() }
      else { await api.delete(`/admin/mcp-market/imported-services/${target.id}`); setDeleteImported(null); await loadImported() }
      toast.success(t('delete_success'))
    } catch { toast.error(t('delete_failed')) }
    finally { setDeleting(false) }
  }

  return (
    <div className="grid gap-4 px-6 pb-8">
      <PageHeader title={t('mcp_market')} />
      <Tabs defaultValue="markets">
        <TabsList>
          <TabsTrigger value="markets">{t('mcp_market_sources')}</TabsTrigger>
          <TabsTrigger value="imported">{t('mcp_imported')}</TabsTrigger>
        </TabsList>

        <TabsContent value="markets" className="grid gap-4 mt-4">
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={loadMarkets}><RefreshCw className="w-4 h-4 mr-1.5" />{t('refresh')}</Button>
            <Button onClick={openAddMarket}><Plus className="w-4 h-4 mr-1.5" />{t('add_market')}</Button>
          </div>
          <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
            {marketsLoading ? <div className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div> : (
              <Table>
                <TableHeader><TableRow>
                  <TableHead className="w-14">ID</TableHead>
                  <TableHead>{t('name')}</TableHead>
                  <TableHead className="w-36">{t('provider')}</TableHead>
                  <TableHead>{t('catalog_url')}</TableHead>
                  <TableHead className="w-20 text-center">{t('enabled_status')}</TableHead>
                  <TableHead className="w-40">{t('created_at')}</TableHead>
                  <TableHead className="w-14 text-center">{t('actions')}</TableHead>
                </TableRow></TableHeader>
                <TableBody>
                  {markets.length === 0 ? <TableRow><TableCell colSpan={7} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
                    : markets.map((row) => (
                      <TableRow key={row.id}>
                        <TableCell className="text-xs text-[var(--color-text-secondary)]">{row.id}</TableCell>
                        <TableCell className="font-medium">{row.name}</TableCell>
                        <TableCell className="text-sm">{row.provider_id}</TableCell>
                        <TableCell className="text-xs text-[var(--color-text-secondary)] font-mono truncate max-w-xs">{row.catalog_url}</TableCell>
                        <TableCell className="text-center"><Switch checked={row.enabled} onCheckedChange={async (v) => { setMarkets((prev) => prev.map((r) => r.id === row.id ? { ...r, enabled: v } : r)); try { await api.put(`/admin/mcp-markets/${row.id}`, { ...row, enabled: v }) } catch { await loadMarkets() } }} /></TableCell>
                        <TableCell className="text-sm text-[var(--color-text-secondary)]">{new Date(row.created_at).toLocaleString()}</TableCell>
                        <TableCell className="text-center"><DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" size="icon" className="h-7 w-7"><MoreHorizontal className="w-4 h-4" /></Button></DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuItem onClick={() => openEditMarket(row)}>{t('edit')}</DropdownMenuItem><DropdownMenuSeparator /><DropdownMenuItem className="text-destructive" onClick={() => setDeleteMarket(row)}>{t('delete')}</DropdownMenuItem></DropdownMenuContent></DropdownMenu></TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            )}
          </div>
        </TabsContent>

        <TabsContent value="imported" className="grid gap-4 mt-4">
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={loadImported}><RefreshCw className="w-4 h-4 mr-1.5" />{t('refresh')}</Button>
            <Button onClick={openAddImported}><Plus className="w-4 h-4 mr-1.5" />{t('add_mcp_service')}</Button>
          </div>
          <div className="rounded-xl border border-[var(--color-line)] overflow-hidden">
            {importedLoading ? <div className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div> : (
              <Table>
                <TableHeader><TableRow>
                  <TableHead className="w-14">ID</TableHead>
                  <TableHead>{t('name')}</TableHead>
                  <TableHead className="w-36">{t('type')}</TableHead>
                  <TableHead>URL</TableHead>
                  <TableHead className="w-20 text-center">{t('enabled_status')}</TableHead>
                  <TableHead className="w-40">{t('created_at')}</TableHead>
                  <TableHead className="w-14 text-center">{t('actions')}</TableHead>
                </TableRow></TableHeader>
                <TableBody>
                  {imported.length === 0 ? <TableRow><TableCell colSpan={7} className="py-10 text-center text-sm text-[var(--color-text-secondary)]">{t('no_data')}</TableCell></TableRow>
                    : imported.map((row) => (
                      <TableRow key={row.id}>
                        <TableCell className="text-xs text-[var(--color-text-secondary)]">{row.id}</TableCell>
                        <TableCell className="font-medium">{row.name}</TableCell>
                        <TableCell className="text-sm">{row.type}</TableCell>
                        <TableCell className="text-xs text-[var(--color-text-secondary)] font-mono truncate max-w-xs">{row.url}</TableCell>
                        <TableCell className="text-center"><Switch checked={row.enabled} onCheckedChange={async (v) => { setImported((prev) => prev.map((r) => r.id === row.id ? { ...r, enabled: v } : r)); try { await api.put(`/admin/mcp-market/imported-services/${row.id}`, { ...row, enabled: v }) } catch { await loadImported() } }} /></TableCell>
                        <TableCell className="text-sm text-[var(--color-text-secondary)]">{new Date(row.created_at).toLocaleString()}</TableCell>
                        <TableCell className="text-center"><DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" size="icon" className="h-7 w-7"><MoreHorizontal className="w-4 h-4" /></Button></DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuItem onClick={() => openEditImported(row)}>{t('edit')}</DropdownMenuItem><DropdownMenuSeparator /><DropdownMenuItem className="text-destructive" onClick={() => setDeleteImported(row)}>{t('delete')}</DropdownMenuItem></DropdownMenuContent></DropdownMenu></TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            )}
          </div>
        </TabsContent>
      </Tabs>

      {/* Market dialog */}
      <Dialog open={showMarketDialog} onOpenChange={(v) => { if (!v) setShowMarketDialog(false) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{editingMarket ? t('edit_market') : t('add_market')}</DialogTitle></DialogHeader>
          <div className="grid gap-3 py-2">
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('name')}</label><Input value={marketForm.name} onChange={(e) => setMF({ name: e.target.value })} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('provider')}</label>
              <Select value={marketForm.provider_id} onValueChange={(v) => setMF({ provider_id: v })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="modelscope">ModelScope</SelectItem><SelectItem value="huggingface">HuggingFace</SelectItem><SelectItem value="generic">Generic</SelectItem></SelectContent></Select>
            </div>
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('catalog_url')}</label><Input value={marketForm.catalog_url} onChange={(e) => setMF({ catalog_url: e.target.value })} placeholder="https://..." /></div>
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('auth_token')}</label><Input type="password" value={marketForm.auth_token} onChange={(e) => setMF({ auth_token: e.target.value })} placeholder={t('optional_label')} /></div>
            <div className="flex items-center gap-2"><Switch checked={marketForm.enabled} onCheckedChange={(v) => setMF({ enabled: v })} /><span className="text-sm">{t('enabled_status')}</span></div>
          </div>
          <DialogFooter><Button variant="outline" onClick={() => setShowMarketDialog(false)}>{t('cancel')}</Button><Button disabled={marketSaving} onClick={saveMarket}>{t('save')}</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Imported service dialog */}
      <Dialog open={showImportedDialog} onOpenChange={(v) => { if (!v) setShowImportedDialog(false) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader><DialogTitle>{editingImported ? t('edit_mcp_service') : t('add_mcp_service')}</DialogTitle></DialogHeader>
          <div className="grid gap-3 py-2">
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('name')}</label><Input value={importedForm.name} onChange={(e) => setIF({ name: e.target.value })} /></div>
            <div className="grid gap-1.5"><label className="text-sm font-medium">{t('type')}</label>
              <Select value={importedForm.type} onValueChange={(v) => setIF({ type: v })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="streamablehttp">Streamable HTTP</SelectItem><SelectItem value="sse">SSE</SelectItem><SelectItem value="stdio">Stdio</SelectItem></SelectContent></Select>
            </div>
            <div className="grid gap-1.5"><label className="text-sm font-medium">URL</label><Input value={importedForm.url} onChange={(e) => setIF({ url: e.target.value })} placeholder="https://..." /></div>
            <div className="flex items-center gap-2"><Switch checked={importedForm.enabled} onCheckedChange={(v) => setIF({ enabled: v })} /><span className="text-sm">{t('enabled_status')}</span></div>
          </div>
          <DialogFooter><Button variant="outline" onClick={() => setShowImportedDialog(false)}>{t('cancel')}</Button><Button disabled={importedSaving} onClick={saveImported}>{t('save')}</Button></DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog open={!!(deleteMarket || deleteImported)} onClose={() => { setDeleteMarket(null); setDeleteImported(null) }} onConfirm={handleDelete} isLoading={deleting} title={t('confirm_delete')} description={t('confirm_delete_config')} />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/mcp-market')({
  component: McpMarketPage,
})
