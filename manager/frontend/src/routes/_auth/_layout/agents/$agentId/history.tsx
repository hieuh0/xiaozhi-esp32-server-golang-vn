import { useEffect, useState } from 'react'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { ArrowLeft, Download } from 'lucide-react'
import { toast } from 'sonner'
import { agentsApi } from '@/features/agents/api/agents-api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'

interface Message { id: number; role: string; content: string; device_id?: string; created_at: string }
interface Device { id: number; mac: string; name?: string }

function AgentHistoryPage() {
  const { t } = useLocale()
  const router = useRouter()
  const { agentId } = Route.useParams()

  const [messages, setMessages] = useState<Message[]>([])
  const [total, setTotal] = useState(0)
  const [devices, setDevices] = useState<Device[]>([])
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState(false)
  const [agentName, setAgentName] = useState('')

  const [filters, setFilters] = useState({ role: '', device_id: '', start_date: '', end_date: '' })
  const [page, setPage] = useState(1)
  const pageSize = 50
  const totalPages = Math.ceil(total / pageSize)

  const load = async (p = page) => {
    setLoading(true)
    try {
      const params = { page: p, page_size: pageSize, ...Object.fromEntries(Object.entries(filters).filter(([, v]) => v)) }
      const res = await agentsApi.getAgentHistory(agentId, params)
      setMessages(res.messages); setTotal(res.total)
    } catch { toast.error(t('load_history_failed')) }
    finally { setLoading(false) }
  }

  useEffect(() => {
    agentsApi.getAgent(agentId).then((a) => setAgentName(a.name)).catch(() => {})
    agentsApi.getAgentDevices(agentId).then(setDevices).catch(() => {})
    load(1)
  }, [agentId]) // eslint-disable-line react-hooks/exhaustive-deps

  const applyFilters = () => { setPage(1); load(1) }

  const handleExport = async () => {
    setExporting(true)
    try {
      const blob = await agentsApi.exportHistory(agentId, Object.fromEntries(Object.entries(filters).filter(([, v]) => v)))
      const url = URL.createObjectURL(blob)
      const a = Object.assign(document.createElement('a'), { href: url, download: `history-${agentId}.csv` })
      document.body.appendChild(a); a.click(); URL.revokeObjectURL(url); document.body.removeChild(a)
    } catch { toast.error(t('export_failed')) }
    finally { setExporting(false) }
  }

  const roleColor: Record<string, string> = { user: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400', assistant: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' }

  return (
    <div className="min-h-full py-2 pb-6">
      <div className="max-w-[1120px] mx-auto mb-4 flex items-center justify-between gap-4 px-2">
        <div className="flex items-center gap-2.5 min-w-0">
          <Button variant="ghost" size="sm" onClick={() => router.navigate({ to: '/agents' as never })}>
            <ArrowLeft className="w-4 h-4 mr-1" />{t('back')}
          </Button>
          <h2 className="text-xl font-bold text-[var(--color-text)] truncate">{agentName || t('agent')} — {t('chat')}</h2>
        </div>
        <Button variant="outline" size="sm" disabled={exporting} onClick={handleExport}>
          <Download className="w-4 h-4 mr-1.5" />{t('export')}
        </Button>
      </div>

      <div className="max-w-[1120px] mx-auto mb-4 flex flex-wrap gap-3 items-end px-2">
        <Select value={filters.role} onValueChange={(v) => setFilters((f) => ({ ...f, role: v === 'all' ? '' : v }))}>
          <SelectTrigger className="w-32"><SelectValue placeholder={t('all_roles')} /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('all_roles')}</SelectItem>
            <SelectItem value="user">{t('user')}</SelectItem>
            <SelectItem value="assistant">{t('assistant')}</SelectItem>
          </SelectContent>
        </Select>
        <Select value={filters.device_id} onValueChange={(v) => setFilters((f) => ({ ...f, device_id: v === 'all' ? '' : v }))}>
          <SelectTrigger className="w-40"><SelectValue placeholder={t('all_devices')} /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('all_devices')}</SelectItem>
            {devices.map((d) => <SelectItem key={d.id} value={String(d.id)}>{d.name || d.mac}</SelectItem>)}
          </SelectContent>
        </Select>
        <input type="date" value={filters.start_date} onChange={(e) => setFilters((f) => ({ ...f, start_date: e.target.value }))} className="h-9 px-3 text-sm border border-[var(--color-line)] rounded-md bg-[var(--color-surface-1)] text-[var(--color-text)]" />
        <input type="date" value={filters.end_date} onChange={(e) => setFilters((f) => ({ ...f, end_date: e.target.value }))} className="h-9 px-3 text-sm border border-[var(--color-line)] rounded-md bg-[var(--color-surface-1)] text-[var(--color-text)]" />
        <Button size="sm" onClick={applyFilters}>{t('filter')}</Button>
      </div>

      <div className="max-w-[1120px] mx-auto px-2 grid gap-2">
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-16 rounded-xl" />)
        ) : messages.length === 0 ? (
          <p className="text-sm text-[var(--color-text-secondary)] text-center py-12">{t('no_history')}</p>
        ) : messages.map((msg) => (
          <div key={msg.id} className="p-4 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] grid gap-1.5">
            <div className="flex items-center gap-2">
              <span className={`px-2 py-0.5 rounded text-xs font-medium ${roleColor[msg.role] || 'bg-[var(--color-surface-2)] text-[var(--color-text-secondary)]'}`}>{msg.role}</span>
              {msg.device_id && <span className="text-xs text-[var(--color-text-tertiary)]">device: {msg.device_id}</span>}
              <span className="text-xs text-[var(--color-text-tertiary)] ml-auto">{new Date(msg.created_at).toLocaleString()}</span>
            </div>
            <p className="text-sm text-[var(--color-text)] whitespace-pre-wrap line-clamp-3">{msg.content}</p>
          </div>
        ))}
        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-3 pt-4">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => { const p = page - 1; setPage(p); load(p) }}>{t('prev')}</Button>
            <span className="text-sm text-[var(--color-text-secondary)]">{page} / {totalPages}</span>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => { const p = page + 1; setPage(p); load(p) }}>{t('next')}</Button>
          </div>
        )}
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/agents/$agentId/history')({
  component: AgentHistoryPage,
})
