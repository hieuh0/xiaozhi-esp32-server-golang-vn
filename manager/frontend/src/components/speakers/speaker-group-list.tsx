import { Search, Plus, Play, Eye, Edit, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { useLocale } from '@/hooks/use-locale'

interface Agent { id: number; name: string }
interface SpeakerGroup {
  id: number
  name: string
  agent_name?: string
  prompt?: string
  sample_count: number
  created_at: string
}

interface Props {
  agents: Agent[]
  groups: SpeakerGroup[]
  loading: boolean
  filterAgentId: string
  searchKeyword: string
  onFilterAgentChange: (v: string) => void
  onSearchChange: (v: string) => void
  onAdd: () => void
  onEdit: (row: SpeakerGroup) => void
  onDelete: (row: SpeakerGroup) => void
  onViewSamples: (row: SpeakerGroup) => void
  onVerify: (row: SpeakerGroup) => void
  formatDate: (v: string) => string
  truncateText: (v: string, n: number) => string
}

export function SpeakerGroupList({
  agents, groups, loading,
  filterAgentId, searchKeyword,
  onFilterAgentChange, onSearchChange,
  onAdd, onEdit, onDelete, onViewSamples, onVerify,
  formatDate, truncateText,
}: Props) {
  const { t } = useLocale()

  return (
    <div className="grid gap-4 px-6 pb-8">
      {/* Filter bar */}
      <div className="flex items-center gap-2 flex-wrap">
        <Select value={filterAgentId || '__all__'} onValueChange={(v) => onFilterAgentChange(v === '__all__' ? '' : v)}>
          <SelectTrigger className="w-[200px]"><SelectValue placeholder={t('filter_by_agent')} /></SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">{t('all_agents')}</SelectItem>
            {agents.map((a) => <SelectItem key={a.id} value={String(a.id)}>{a.name}</SelectItem>)}
          </SelectContent>
        </Select>
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-tertiary)]" />
          <Input value={searchKeyword} onChange={(e) => onSearchChange(e.target.value)}
            placeholder={t('search_voiceprint_group')} className="pl-8 w-[250px]" />
        </div>
        <Button className="ml-auto" onClick={onAdd}>
          <Plus className="w-4 h-4 mr-1.5" />{t('create_voiceprint_group')}
        </Button>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] overflow-hidden">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('voiceprint_group_name')}</TableHead>
                <TableHead>{t('link_agent')}</TableHead>
                <TableHead>Prompt</TableHead>
                <TableHead className="w-24 text-center">{t('sample_count')}</TableHead>
                <TableHead>{t('created_at')}</TableHead>
                <TableHead className="text-center">{t('actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 4 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 6 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : groups.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-12 text-center text-sm text-[var(--color-text-tertiary)]">
                    {t('no_data')}
                  </TableCell>
                </TableRow>
              ) : (
                groups.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-semibold">{row.name}</TableCell>
                    <TableCell className="text-[var(--color-text-secondary)]">{row.agent_name}</TableCell>
                    <TableCell className="text-[var(--color-text-secondary)] text-sm">
                      {row.prompt
                        ? <span title={row.prompt} className="cursor-help">{truncateText(row.prompt, 30)}</span>
                        : <span className="text-[var(--color-text-tertiary)]">-</span>}
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge variant="secondary">{row.sample_count}</Badge>
                    </TableCell>
                    <TableCell className="text-sm text-[var(--color-text-secondary)]">{formatDate(row.created_at)}</TableCell>
                    <TableCell>
                      <div className="flex items-center justify-center gap-1 flex-wrap">
                        <Button variant="outline" size="sm" onClick={() => onVerify(row)}>
                          <Play className="w-3.5 h-3.5 mr-1" />{t('verify')}
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => onViewSamples(row)}>
                          <Eye className="w-3.5 h-3.5 mr-1" />{t('manage_voiceprints')}
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => onEdit(row)}>
                          <Edit className="w-3.5 h-3.5 mr-1" />{t('edit')}
                        </Button>
                        <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive"
                          onClick={() => onDelete(row)}>
                          <Trash2 className="w-3.5 h-3.5 mr-1" />{t('delete')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  )
}
