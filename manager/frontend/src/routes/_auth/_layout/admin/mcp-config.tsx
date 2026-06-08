import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Plus, Trash2, Search } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PageHeader } from '@/components/ui/page-header'
import { cn } from '@/lib/utils'

interface McpServer {
  name: string; type: string; url: string; enabled: boolean
  allowed_tools: string[]; _tool_options: string[]; _tools_loading: boolean
}

interface McpForm {
  mcp: { global: { enabled: boolean; reconnect_interval: number; max_reconnect_attempts: number; servers: McpServer[] } }
  local_mcp: { exit_conversation: boolean; clear_conversation_history: boolean; play_music: boolean }
}

const newServer = (): McpServer => ({ name: '', type: 'streamablehttp', url: '', enabled: true, allowed_tools: [], _tool_options: [], _tools_loading: false })

const defaults: McpForm = { mcp: { global: { enabled: true, servers: [], reconnect_interval: 300, max_reconnect_attempts: 10 } }, local_mcp: { exit_conversation: true, clear_conversation_history: true, play_music: false } }

function McpConfigPage() {
  const { t } = useLocale()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [configId, setConfigId] = useState<number | null>(null)
  const [form, setForm] = useState<McpForm>(defaults)

  const setGlobal = (patch: Partial<typeof defaults.mcp.global>) => setForm((f) => ({ ...f, mcp: { ...f.mcp, global: { ...f.mcp.global, ...patch } } }))
  const setLocal = (patch: Partial<typeof defaults.local_mcp>) => setForm((f) => ({ ...f, local_mcp: { ...f.local_mcp, ...patch } }))
  const setServer = (i: number, patch: Partial<McpServer>) => setForm((f) => { const servers = [...f.mcp.global.servers]; servers[i] = { ...servers[i], ...patch }; return { ...f, mcp: { ...f.mcp, global: { ...f.mcp.global, servers } } } })

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.get('/admin/mcp-configs')
      const configs = res.data?.data || []
      if (configs.length > 0) {
        const c = configs.find((x: { is_default: boolean }) => x.is_default) || configs[0]
        setConfigId(c.id)
        const d = JSON.parse(c.json_data || '{}')
        const globalData = d.mcp?.global || d.global || {}
        setForm({
          mcp: { global: { enabled: globalData.enabled !== false, reconnect_interval: Number(globalData.reconnect_interval) || 300, max_reconnect_attempts: Number(globalData.max_reconnect_attempts) || 10, servers: (globalData.servers || []).map((s: McpServer) => ({ ...s, allowed_tools: s.allowed_tools || [], _tool_options: [], _tools_loading: false })) } },
          local_mcp: { exit_conversation: d.local_mcp?.exit_conversation !== false, clear_conversation_history: d.local_mcp?.clear_conversation_history !== false, play_music: !!d.local_mcp?.play_music }
        })
      } else { setConfigId(null); setForm(defaults) }
    } catch { toast.error(t('load_mcp_config_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSave = async () => {
    setSaving(true)
    try {
      const cleanServers = form.mcp.global.servers.map(({ _tool_options: _to, _tools_loading: _tl, ...s }) => s)
      const json_data = JSON.stringify({ mcp: { global: { ...form.mcp.global, servers: cleanServers } }, local_mcp: form.local_mcp })
      const payload = { name: t('mcp_global_config'), config_id: 'mcp_global_config', is_default: true, json_data }
      if (configId) { await api.put(`/admin/mcp-configs/${configId}`, payload); toast.success(t('mcp_config_updated')) }
      else { const r = await api.post('/admin/mcp-configs', payload); setConfigId(r.data?.data?.id || null); toast.success(t('mcp_config_saved')) }
      await load()
    } catch (e) { toast.error((e as Error).message || t('save_mcp_failed')) }
    finally { setSaving(false) }
  }

  const discoverTools = async (i: number) => {
    const s = form.mcp.global.servers[i]
    if (!s.url) { toast.warning(t('fill_server_url')); return }
    setServer(i, { _tools_loading: true })
    try {
      const res = await api.post('/admin/mcp-configs/discover-tools', { transport: s.type, url: s.url })
      const tools = (res.data?.data?.tools || []).map((t: { name: string }) => t.name)
      setServer(i, { _tool_options: tools, _tools_loading: false })
      toast.success(t('probing_tools_count', { count: tools.length }))
    } catch (e) { setServer(i, { _tools_loading: false }); toast.error((e as Error).message || t('probe_tools_failed')) }
  }

  const toggleTool = (i: number, tool: string) => {
    const s = form.mcp.global.servers[i]
    const tools = s.allowed_tools.includes(tool) ? s.allowed_tools.filter((t) => t !== tool) : [...s.allowed_tools, tool]
    setServer(i, { allowed_tools: tools })
  }

  const addServer = () => setForm((f) => ({ ...f, mcp: { ...f.mcp, global: { ...f.mcp.global, servers: [...f.mcp.global.servers, newServer()] } } }))
  const removeServer = (i: number) => setForm((f) => { const servers = f.mcp.global.servers.filter((_, idx) => idx !== i); return { ...f, mcp: { ...f.mcp, global: { ...f.mcp.global, servers } } } })

  if (loading) return <div className="p-6 text-center text-sm text-[var(--color-text-secondary)]">{t('loading')}</div>

  return (
    <div className="grid gap-6 px-6 pb-8">
      <PageHeader title="MCP Config" />

      {/* Global settings */}
      <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] p-6 grid gap-5 max-w-[640px]">
        <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('mcp_global_config')}</p>
        <div className="flex items-center justify-between gap-4">
          <p className="text-sm font-semibold text-[var(--color-text)]">{t('enabled_status')}</p>
          <Switch checked={form.mcp.global.enabled} onCheckedChange={(v) => setGlobal({ enabled: v })} />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('reconnect_interval_s')}</label>
            <Input type="number" value={form.mcp.global.reconnect_interval} onChange={(e) => setGlobal({ reconnect_interval: Number(e.target.value) })} min={1} max={3600} />
          </div>
          <div className="grid gap-1.5">
            <label className="text-sm font-semibold text-[var(--color-text)]">{t('max_reconnect_attempts')}</label>
            <Input type="number" value={form.mcp.global.max_reconnect_attempts} onChange={(e) => setGlobal({ max_reconnect_attempts: Number(e.target.value) })} min={1} max={100} />
          </div>
        </div>
      </div>

      {/* Server list */}
      <div className="grid gap-3">
        <div className="flex items-center justify-between">
          <p className="text-sm font-semibold text-[var(--color-text)]">{t('mcp_servers')} ({form.mcp.global.servers.length})</p>
          <Button size="sm" onClick={addServer}><Plus className="w-4 h-4 mr-1.5" />{t('add_server')}</Button>
        </div>
        {form.mcp.global.servers.map((server, i) => (
          <div key={i} className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] p-5 grid gap-4">
            <div className="flex items-center gap-3 flex-wrap">
              <Input className="flex-1 min-w-[160px]" placeholder={t('server_name')} value={server.name} onChange={(e) => setServer(i, { name: e.target.value })} />
              <Select value={server.type} onValueChange={(v) => setServer(i, { type: v })}>
                <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="streamablehttp">Streamable HTTP</SelectItem>
                  <SelectItem value="sse">SSE</SelectItem>
                  <SelectItem value="stdio">Stdio</SelectItem>
                </SelectContent>
              </Select>
              <div className="flex items-center gap-1.5">
                <Switch checked={server.enabled} onCheckedChange={(v) => setServer(i, { enabled: v })} />
                <span className="text-xs text-[var(--color-text-secondary)]">{t('enabled_status')}</span>
              </div>
              <Button variant="ghost" size="icon" className="text-destructive h-8 w-8" onClick={() => removeServer(i)}><Trash2 className="w-4 h-4" /></Button>
            </div>
            <div className="flex gap-2">
              <Input className="flex-1" placeholder="URL" value={server.url} onChange={(e) => setServer(i, { url: e.target.value })} />
              <Button size="sm" variant="outline" disabled={server._tools_loading} onClick={() => discoverTools(i)}>
                <Search className="w-4 h-4 mr-1.5" />{server._tools_loading ? '...' : t('discover_tools')}
              </Button>
            </div>
            {server._tool_options.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {server._tool_options.map((tool) => (
                  <button key={tool} type="button" onClick={() => toggleTool(i, tool)} className={cn('text-xs px-2 py-1 rounded-full border transition-colors', server.allowed_tools.includes(tool) ? 'bg-[var(--color-primary)] text-white border-[var(--color-primary)]' : 'border-[var(--color-line)] text-[var(--color-text-secondary)] hover:border-[var(--color-primary)]')}>
                    {tool}
                  </button>
                ))}
              </div>
            )}
            {server.allowed_tools.length > 0 && (
              <p className="text-xs text-[var(--color-text-secondary)]">{t('allowed_tools')}: {server.allowed_tools.join(', ')}</p>
            )}
          </div>
        ))}
      </div>

      {/* Local MCP */}
      <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] p-6 grid gap-4 max-w-[640px]">
        <p className="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">{t('local_mcp')}</p>
        {[
          { key: 'exit_conversation' as const, label: t('mcp_exit_conversation') },
          { key: 'clear_conversation_history' as const, label: t('mcp_clear_history') },
          { key: 'play_music' as const, label: t('mcp_play_music') },
        ].map(({ key, label }) => (
          <div key={key} className="flex items-center justify-between gap-4">
            <p className="text-sm font-semibold text-[var(--color-text)]">{label}</p>
            <Switch checked={form.local_mcp[key]} onCheckedChange={(v) => setLocal({ [key]: v })} />
          </div>
        ))}
      </div>

      <div className="flex items-center justify-end gap-3">
        <Button variant="outline" disabled={loading} onClick={load}>{t('reset_to_current')}</Button>
        <Button disabled={saving} onClick={handleSave}>{t('save_config')}</Button>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/mcp-config')({
  component: McpConfigPage,
})
