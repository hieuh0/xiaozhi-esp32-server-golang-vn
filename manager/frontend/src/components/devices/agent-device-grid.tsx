import type React from 'react'
import { ArrowLeft, Check, MessageCircle, Monitor, PenLine, Plus, Settings, Trash2, User, X } from 'lucide-react'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'
import type { Device } from '@/features/devices/types'
import { getDeviceDisplayName, isDeviceOnline } from '@/features/devices/types'
import type { Agent } from '@/features/agents/types'

function formatDate(val?: string | null): string {
  if (!val) return '—'
  return new Date(val).toLocaleString()
}

function getDeviceIdentityText(device: Device): string {
  const parts = [device.device_name, device.device_code].filter(Boolean)
  return parts.join(' · ') || '—'
}

interface AgentDeviceGridProps {
  agents: Agent[]
  filteredDevices: Device[]
  emptyDescription?: string
  showBackButton?: boolean
  filterAgentId: string
  onFilterAgentIdChange: (v: string) => void
  editingDeviceId: number | null
  editingDeviceName: string
  onEditingDeviceNameChange: (v: string) => void
  renamingDeviceId: number | null
  nameInputRef: React.RefObject<HTMLInputElement | null>
  getDeviceAgentName: (device: Device) => string
  onGoBack?: () => void
  onAddDevice: () => void
  onStartNameEdit: (device: Device) => void
  onCancelNameEdit: () => void
  onSaveName: (device: Device) => void
  onDeviceRole: (deviceId: number) => void
  onDeviceMcp: (device: Device) => void
  onVoicePush: (device: Device) => void
  onDeleteDevice: (device: Device) => void
}

export function AgentDeviceGrid({
  agents, filteredDevices, emptyDescription, showBackButton, filterAgentId,
  onFilterAgentIdChange, editingDeviceId, editingDeviceName, onEditingDeviceNameChange,
  renamingDeviceId, nameInputRef, getDeviceAgentName, onGoBack, onAddDevice,
  onStartNameEdit, onCancelNameEdit, onSaveName, onDeviceRole, onDeviceMcp,
  onVoicePush, onDeleteDevice,
}: AgentDeviceGridProps) {
  const { t } = useLocale()

  return (
    <div>
      <div className="flex items-center justify-between gap-3 mb-4 px-4 py-3 rounded-xl bg-[var(--color-surface-1)] border border-[var(--color-line)]">
        <div className="flex items-center gap-3 flex-wrap min-w-0">
          {showBackButton && (
            <Button variant="ghost" size="sm" onClick={onGoBack}>
              <ArrowLeft className="w-4 h-4 mr-1" />{t('back')}
            </Button>
          )}
          <Select value={filterAgentId} onValueChange={onFilterAgentIdChange}>
            <SelectTrigger className="w-[220px]">
              <SelectValue placeholder={t('filter_by_agent')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">{t('all_devices')}</SelectItem>
              {agents.map((a) => (
                <SelectItem key={a.id} value={String(a.id)}>{a.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <span className="text-sm font-semibold text-[var(--color-text-secondary)]">
            {t('device_count_n', { count: filteredDevices.length })}
          </span>
        </div>
        <Button onClick={onAddDevice}>
          <Plus className="w-4 h-4 mr-1.5" />{t('add_device')}
        </Button>
      </div>

      {filteredDevices.length === 0 ? (
        <div className="mt-10 text-center py-16 px-6 bg-[var(--color-surface-1)] border border-[var(--color-line)] rounded-xl">
          <Monitor className="w-16 h-16 mx-auto mb-4 text-[var(--color-text-tertiary)]" />
          <h3 className="text-lg font-semibold text-[var(--color-text)] mb-2">{t('no_devices')}</h3>
          <p className="text-sm text-[var(--color-text-secondary)] mb-6">{emptyDescription}</p>
          <Button onClick={onAddDevice}>
            <Plus className="w-4 h-4 mr-1.5" />{t('add_first_device')}
          </Button>
        </div>
      ) : (
        <div className="mt-5 grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(280px,340px))]">
          {filteredDevices.map((device) => {
            const online = isDeviceOnline(device.last_active_at)
            const isEditing = editingDeviceId === device.id
            return (
              <div key={device.id} className="bg-[var(--color-surface-1)] rounded-xl p-4 border border-[var(--color-line)] flex flex-col h-full shadow-[var(--shadow-card)] transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[var(--shadow-card-hover)] hover:border-[var(--color-primary)]/30">
                <div className="flex items-start gap-3 mb-4">
                  <div className="w-10 h-10 rounded-xl bg-gradient-to-b from-indigo-400 to-indigo-600 flex items-center justify-center text-white shadow-[0_6px_16px_rgba(99,102,241,0.2)] shrink-0">
                    <Monitor className="w-5 h-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      {isEditing ? (
                        <Input
                          ref={nameInputRef}
                          value={editingDeviceName}
                          onChange={(e) => onEditingDeviceNameChange(e.target.value)}
                          className="h-7 text-sm flex-1 min-w-0"
                          maxLength={50}
                          placeholder={t('enter_device_nickname')}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') { e.preventDefault(); onSaveName(device) }
                            if (e.key === 'Escape') { e.preventDefault(); onCancelNameEdit() }
                          }}
                        />
                      ) : (
                        <button
                          type="button"
                          className="flex-1 min-w-0 text-left font-bold text-[var(--color-text)] text-sm truncate hover:text-[var(--color-primary)] cursor-text"
                          title={t('click_edit_nickname', { name: getDeviceDisplayName(device) })}
                          onClick={() => onStartNameEdit(device)}
                        >
                          {getDeviceDisplayName(device)}
                        </button>
                      )}
                      <div className="flex items-center shrink-0 gap-0.5">
                        {isEditing ? (
                          <>
                            <Button variant="ghost" size="icon" className="h-6 w-6" disabled={renamingDeviceId === device.id} title={t('save_nickname')} onClick={() => onSaveName(device)}>
                              <Check className="h-3 w-3 text-[var(--color-primary)]" />
                            </Button>
                            <Button variant="ghost" size="icon" className="h-6 w-6" title={t('cancel_edit')} onClick={onCancelNameEdit}>
                              <X className="h-3 w-3" />
                            </Button>
                          </>
                        ) : (
                          <Button variant="ghost" size="icon" className="h-6 w-6 opacity-20 hover:opacity-100 transition-opacity" title={t('edit_nickname_title')} onClick={() => onStartNameEdit(device)}>
                            <PenLine className="h-3 w-3" />
                          </Button>
                        )}
                      </div>
                    </div>
                    <p className="text-[11px] text-[var(--color-text-tertiary)] font-mono truncate">{getDeviceIdentityText(device)}</p>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <span className={cn('w-2 h-2 rounded-full', online ? 'bg-[var(--color-success)]' : 'bg-[var(--color-danger)]')} />
                    <span className="text-xs text-[var(--color-text-secondary)]">{online ? t('online') : t('offline')}</span>
                  </div>
                </div>

                <div className="flex-1 grid gap-2 mb-4 text-xs">
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">{t('link_agent')}</span>
                    <span className="font-medium text-[var(--color-text)]">{getDeviceAgentName(device)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">{t('activation_status')}</span>
                    <span className={cn('inline-flex items-center px-1.5 py-0.5 rounded-full border text-[10px] font-medium', device.activated ? 'status-success' : 'status-warning')}>
                      {device.activated ? t('activated') : t('not_activated')}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">{t('last_active')}</span>
                    <span className="text-[var(--color-text)]">{device.last_active_at ? formatDate(device.last_active_at) : t('never_active')}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">{t('created_at')}</span>
                    <span className="text-[var(--color-text)]">{formatDate(device.created_at)}</span>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-2">
                  <Button variant="outline" size="sm" className="status-primary border hover:opacity-90" onClick={() => onDeviceRole(device.id)}>
                    <User className="w-3.5 h-3.5 mr-1" />{t('role')}
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => onDeviceMcp(device)}>
                    <Settings className="w-3.5 h-3.5 mr-1" />MCP
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => onVoicePush(device)}>
                    <MessageCircle className="w-3.5 h-3.5 mr-1" />{t('voice_notify')}
                  </Button>
                  <Button variant="outline" size="sm" className="text-destructive border-destructive/30 hover:bg-destructive/10" onClick={() => onDeleteDevice(device)}>
                    <Trash2 className="w-3.5 h-3.5 mr-1" />{t('delete')}
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
