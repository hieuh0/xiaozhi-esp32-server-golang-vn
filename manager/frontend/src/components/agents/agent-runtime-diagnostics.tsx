import { useState } from 'react'
import { ChevronDown, Copy, RefreshCw, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { buildOpenClawCommands } from '@/utils/openclaw'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

interface Props { agentId: number | string; scope?: 'user' | 'admin' }

interface EndpointData { endpoint: string; connected: boolean; status: string; status_message: string; client_count: number }
interface McpTool { name: string; description?: string; input_schema?: Record<string, unknown> }

const statusClass = (connected: boolean, status: string) => {
  const s = status.toLowerCase()
  if (connected || s === 'online') return 'inline-flex items-center px-2 py-0.5 rounded text-xs bg-green-100 text-green-700 border border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800'
  if (s === 'offline') return 'inline-flex items-center px-2 py-0.5 rounded text-xs bg-red-100 text-red-600 border border-red-200 dark:bg-red-900/30 dark:text-red-400 dark:border-red-800'
  return 'inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-2)] border border-[var(--color-line)] text-[var(--color-text-secondary)]'
}

const copy = (text: string, label: string) => navigator.clipboard.writeText(text).then(() => toast.success(`${label} copied`)).catch(() => toast.error('Copy failed'))

function PanelHeader({ title, summary, open, onToggle }: { title: string; summary: string; open: boolean; onToggle: () => void }) {
  return (
    <button type="button" onClick={onToggle} className="flex items-center justify-between w-full gap-3 text-left">
      <span className="text-sm font-semibold text-[var(--color-text)]">{title}</span>
      <span className="flex items-center gap-2">
        <span className="text-xs text-[var(--color-text-secondary)]">{summary}</span>
        <ChevronDown className={cn('w-4 h-4 text-[var(--color-text-secondary)] transition-transform', open && 'rotate-180')} />
      </span>
    </button>
  )
}

export function AgentRuntimeDiagnostics({ agentId, scope = 'user' }: Props) {
  const { t } = useLocale()
  const base = `/${scope}/agents/${agentId}`

  // MCP panel
  const [mcpOpen, setMcpOpen] = useState(false)
  const [mcpLoading, setMcpLoading] = useState(false)
  const [mcpData, setMcpData] = useState<EndpointData | null>(null)
  const [mcpTools, setMcpTools] = useState<McpTool[]>([])
  const [toolsLoading, setToolsLoading] = useState(false)
  const [selectedTool, setSelectedTool] = useState('')
  const [toolArgs, setToolArgs] = useState('{}')
  const [toolResult, setToolResult] = useState('')
  const [callingTool, setCallingTool] = useState(false)

  // OpenClaw panel
  const [clawOpen, setClawOpen] = useState(false)
  const [clawLoading, setClawLoading] = useState(false)
  const [clawData, setClawData] = useState<Omit<EndpointData, 'client_count'> | null>(null)
  const [clawTestResult, setClawTestResult] = useState('')
  const [clawTestLoading, setClawTestLoading] = useState(false)
  const [clawMessage, setClawMessage] = useState('')

  const loadMcp = async () => {
    setMcpLoading(true)
    try {
      const { data } = await api.get<{ data: EndpointData }>(`${base}/mcp-endpoint`)
      setMcpData({ ...data.data, status: (data.data.status || '').toLowerCase() })
    } catch { setMcpData({ endpoint: '', connected: false, status: 'unknown', status_message: '', client_count: 0 }) }
    finally { setMcpLoading(false) }
  }

  const loadMcpTools = async () => {
    setToolsLoading(true)
    try {
      const { data } = await api.get<{ data: { tools: McpTool[] } }>(`${base}/mcp-tools`)
      const tools = data.data?.tools || []
      setMcpTools(tools)
      if (tools.length) { setSelectedTool(tools[0].name); setToolArgs('{}') }
    } catch { toast.error(t('load_tools_failed')) }
    finally { setToolsLoading(false) }
  }

  const callTool = async () => {
    setCallingTool(true); setToolResult('')
    try {
      let args: Record<string, unknown> = {}
      try { args = JSON.parse(toolArgs) } catch { toast.error('Invalid JSON'); setCallingTool(false); return }
      const { data } = await api.post<{ data: unknown }>(`${base}/mcp-call`, { tool_name: selectedTool, arguments: args })
      setToolResult(typeof data.data === 'string' ? data.data : JSON.stringify(data.data, null, 2))
    } catch (e) { setToolResult((e as Error).message || 'Error') }
    finally { setCallingTool(false) }
  }

  const toggleMcp = async () => {
    const next = !mcpOpen; setMcpOpen(next)
    if (next && !mcpData) await loadMcp()
  }

  const loadClaw = async () => {
    setClawLoading(true)
    try {
      const { data } = await api.get<{ data: Omit<EndpointData, 'client_count'> }>(`${base}/openclaw-endpoint`)
      setClawData({ ...data.data, status: (data.data.status || '').toLowerCase() })
    } catch { setClawData({ endpoint: '', connected: false, status: 'unknown', status_message: '' }) }
    finally { setClawLoading(false) }
  }

  const toggleClaw = async () => {
    const next = !clawOpen; setClawOpen(next)
    if (next && !clawData) await loadClaw()
  }

  const testClawChat = async () => {
    if (!clawMessage.trim()) return
    setClawTestLoading(true); setClawTestResult('')
    try {
      const { data } = await api.post<{ data: string }>(`${base}/openclaw-chat-test`, { message: clawMessage })
      setClawTestResult(data.data || t('detail_not_available'))
    } catch (e) { setClawTestResult((e as Error).message) }
    finally { setClawTestLoading(false) }
  }

  const clawCmds = buildOpenClawCommands(clawData?.endpoint || '')
  const statusText = (d: { connected: boolean; status: string } | null) => {
    if (!d) return t('expand_load_endpoints')
    const s = d.status; if (d.connected || s === 'online') return t('connected')
    if (s === 'offline') return t('not_connected'); return t('status_unknown')
  }

  return (
    <div className="grid gap-3">
      {/* MCP Panel */}
      <div className="border border-[var(--color-line)] rounded-xl p-4 bg-[var(--color-surface-1)] grid gap-3">
        <PanelHeader title="MCP" summary={!mcpData ? t('expand_load_endpoints') : `${statusText(mcpData)} · ${mcpTools.length} ${t('tools_count_short', { status: '', count: mcpTools.length })}`} open={mcpOpen} onToggle={toggleMcp} />
        {mcpOpen && (
          <div className="grid gap-3 pt-1">
            {mcpLoading ? <div className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]"><Loader2 className="w-4 h-4 animate-spin" />{t('loading')}</div> : mcpData && (
              <>
                <div className="flex items-center gap-2 flex-wrap">
                  <span className={statusClass(mcpData.connected, mcpData.status)}>{statusText(mcpData)}</span>
                  {mcpData.client_count > 0 && <span className="text-xs text-[var(--color-text-secondary)]">{t('mcp_client_count_text', { count: mcpData.client_count })}</span>}
                  <Button variant="ghost" size="icon" className="w-6 h-6 ml-auto" onClick={loadMcp}><RefreshCw className="w-3.5 h-3.5" /></Button>
                </div>
                {mcpData.endpoint && (
                  <div className="flex items-center gap-2 p-2 rounded-lg bg-[var(--color-surface-2)] border border-[var(--color-line)]">
                    <code className="text-xs flex-1 truncate">{mcpData.endpoint}</code>
                    <Button variant="ghost" size="icon" className="w-6 h-6 shrink-0" onClick={() => copy(mcpData.endpoint, 'Endpoint')}><Copy className="w-3 h-3" /></Button>
                  </div>
                )}
                <Button variant="outline" size="sm" disabled={toolsLoading} onClick={loadMcpTools}>
                  {toolsLoading && <Loader2 className="w-3.5 h-3.5 mr-1 animate-spin" />}{t('load_tools')}
                </Button>
                {mcpTools.length > 0 && (
                  <div className="grid gap-2">
                    <Select value={selectedTool} onValueChange={setSelectedTool}>
                      <SelectTrigger><SelectValue placeholder={t('select_tool')} /></SelectTrigger>
                      <SelectContent>{mcpTools.map((m) => <SelectItem key={m.name} value={m.name}>{m.name}</SelectItem>)}</SelectContent>
                    </Select>
                    <Textarea value={toolArgs} onChange={(e) => setToolArgs(e.target.value)} rows={3} className="font-mono text-xs" />
                    <Button size="sm" disabled={callingTool} onClick={callTool}>{callingTool && <Loader2 className="w-3.5 h-3.5 mr-1 animate-spin" />}{t('call_tool')}</Button>
                    {toolResult && <pre className="text-xs p-2 rounded bg-[var(--color-surface-2)] border border-[var(--color-line)] max-h-40 overflow-auto whitespace-pre-wrap">{toolResult}</pre>}
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>

      {/* OpenClaw Panel */}
      <div className="border border-[var(--color-line)] rounded-xl p-4 bg-[var(--color-surface-1)] grid gap-3">
        <PanelHeader title="OpenClaw" summary={statusText(clawData)} open={clawOpen} onToggle={toggleClaw} />
        {clawOpen && (
          <div className="grid gap-3 pt-1">
            {clawLoading ? <div className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]"><Loader2 className="w-4 h-4 animate-spin" />{t('loading')}</div> : clawData && (
              <>
                <div className="flex items-center gap-2">
                  <span className={statusClass(clawData.connected, clawData.status)}>{statusText(clawData)}</span>
                  <Button variant="ghost" size="icon" className="w-6 h-6 ml-auto" onClick={loadClaw}><RefreshCw className="w-3.5 h-3.5" /></Button>
                </div>
                {clawCmds.ready ? (
                  <div className="grid gap-2">
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-xs font-medium text-[var(--color-text-secondary)]">{t('install_commands')}</span>
                      <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={() => copy(clawCmds.copyText, 'Commands')}><Copy className="w-3 h-3 mr-1" />{t('copy')}</Button>
                    </div>
                    <pre className="text-xs p-2 rounded bg-[var(--color-surface-2)] border border-[var(--color-line)] whitespace-pre-wrap overflow-auto max-h-32">{clawCmds.copyText}</pre>
                  </div>
                ) : (
                  <p className="text-xs text-[var(--color-text-secondary)]">{t('no_install_command_refresh')}</p>
                )}
                <div className="flex gap-2">
                  <input value={clawMessage} onChange={(e) => setClawMessage(e.target.value)} placeholder={t('test_message')} className="flex-1 h-8 text-sm border border-[var(--color-line)] rounded-md px-3 bg-[var(--color-surface-1)] text-[var(--color-text)] outline-none focus:ring-2 focus:ring-[var(--color-primary)]/30" />
                  <Button size="sm" disabled={clawTestLoading} onClick={testClawChat}>{clawTestLoading && <Loader2 className="w-3.5 h-3.5 mr-1 animate-spin" />}{t('send')}</Button>
                </div>
                {clawTestResult && <pre className="text-xs p-2 rounded bg-[var(--color-surface-2)] border border-[var(--color-line)] max-h-40 overflow-auto whitespace-pre-wrap">{clawTestResult}</pre>}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
