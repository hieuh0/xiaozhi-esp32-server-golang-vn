import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { useAgentDevices } from '@/hooks/use-agent-devices'
import { AgentDeviceGrid } from '@/components/devices/agent-device-grid'
import { MessageInjectDialog } from '@/components/devices/message-inject-dialog'
import { DeviceForm } from '@/components/devices/device-form'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import type { Device } from '@/features/devices/types'

function UserDevicesPage() {
  const { t } = useLocale()
  const {
    agents, devices, filteredDevices, filterAgentId, setFilterAgentId,
    showAddDialog, setShowAddDialog, addingDevice, deviceForm, setDeviceForm,
    showVoicePushDialog, setShowVoicePushDialog, voicePushDeviceId,
    editingDeviceId, editingDeviceName, setEditingDeviceName, renamingDeviceId, nameInputRef,
    showMcpDialog, setShowMcpDialog, mcpLoading, callingTool,
    mcpTools, mcpCallResult, mcpToolName, setMcpToolName, mcpToolArgs, setMcpToolArgs,
    showRoleDialog, setShowRoleDialog, roleLoading, currentDevice, selectedRoleId, setSelectedRoleId, availableRoles,
    getDeviceAgentName,
    loadAll, openAddDialog, handleAddDevice,
    startNameEdit, cancelNameEdit, saveDeviceName, handleDeleteDevice,
    openMcpDialog, callMcpTool,
    openRoleDialog, applyRole,
    openVoicePush,
  } = useAgentDevices()

  const [deleteTarget, setDeleteTarget] = useState<Device | null>(null)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => { loadAll() }, [loadAll])

  const confirmDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await handleDeleteDevice(deleteTarget)
      setDeleteTarget(null)
    } finally { setDeleting(false) }
  }

  const selectedRole = availableRoles.find((r) => r.id === selectedRoleId) || null

  return (
    <div className="grid gap-5 p-6">
      <AgentDeviceGrid
        agents={agents}
        filteredDevices={filteredDevices}
        filterAgentId={filterAgentId}
        onFilterAgentIdChange={setFilterAgentId}
        editingDeviceId={editingDeviceId}
        editingDeviceName={editingDeviceName}
        onEditingDeviceNameChange={setEditingDeviceName}
        renamingDeviceId={renamingDeviceId}
        nameInputRef={nameInputRef}
        getDeviceAgentName={getDeviceAgentName}
        onAddDevice={() => openAddDialog()}
        onStartNameEdit={startNameEdit}
        onCancelNameEdit={cancelNameEdit}
        onSaveName={saveDeviceName}
        onDeviceRole={(id) => {
          const d = devices.find((x) => x.id === id)
          if (d) openRoleDialog(d)
        }}
        onDeviceMcp={openMcpDialog}
        onVoicePush={openVoicePush}
        onDeleteDevice={(d) => setDeleteTarget(d)}
      />

      {/* Bind device dialog */}
      <Dialog open={showAddDialog} onOpenChange={(v) => { if (!v) setShowAddDialog(false) }}>
        <DialogContent className="max-w-[520px] max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('bind_device')}</DialogTitle></DialogHeader>
          <DeviceForm value={deviceForm} onChange={setDeviceForm} agents={agents} />
          <DialogFooter className="mt-4">
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>{t('cancel')}</Button>
            <Button disabled={addingDevice} onClick={handleAddDevice}>
              {addingDevice ? t('binding') : t('bind_device')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* MCP tools dialog */}
      <Dialog open={showMcpDialog} onOpenChange={setShowMcpDialog}>
        <DialogContent className="max-w-[720px] max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('device_mcp_tools')}</DialogTitle></DialogHeader>
          <div className={mcpLoading ? 'opacity-60 pointer-events-none grid gap-4' : 'grid gap-4'}>
            <p className="text-xs text-[var(--color-text-tertiary)]">{t('device_mcp_tools_hint1')}</p>
            {mcpTools.length === 0 ? (
              <p className="text-sm text-[var(--color-text-secondary)] py-2">{t('no_tool_data')}</p>
            ) : (
              <div className="flex flex-wrap gap-1.5">
                {mcpTools.map((tool) => (
                  <span key={tool.name} className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-2)] border border-[var(--color-line)] text-[var(--color-text)]">{tool.name}</span>
                ))}
              </div>
            )}
            <hr className="border-[var(--color-line)]" />
            <div className="grid gap-3">
              <div className="grid gap-1.5">
                <label className="text-sm font-medium text-[var(--color-text)]">{t('tool')}</label>
                <Select value={mcpToolName} onValueChange={setMcpToolName}>
                  <SelectTrigger className="w-full"><SelectValue placeholder={t('select_tool')} /></SelectTrigger>
                  <SelectContent>
                    {mcpTools.map((tool) => <SelectItem key={tool.name} value={tool.name}>{tool.name}</SelectItem>)}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium text-[var(--color-text)]">{t('args_json_label')}</label>
                <Textarea value={mcpToolArgs} onChange={(e) => setMcpToolArgs(e.target.value)} placeholder={t('mcp_args_placeholder')} rows={6} className="font-mono text-sm" />
              </div>
              <Button disabled={callingTool} onClick={callMcpTool}>{t('call_tool')}</Button>
            </div>
            <hr className="border-[var(--color-line)]" />
            <pre className="min-h-[60px] p-3 rounded-lg bg-[var(--color-surface-2)] border border-[var(--color-line)] text-xs font-mono whitespace-pre-wrap break-all">
              {mcpCallResult || t('no_call_results')}
            </pre>
          </div>
        </DialogContent>
      </Dialog>

      {/* Role config dialog */}
      <Dialog open={showRoleDialog} onOpenChange={(v) => { if (!v) setShowRoleDialog(false) }}>
        <DialogContent className="max-w-[660px] max-h-[90vh] overflow-y-auto">
          <DialogHeader><DialogTitle>{t('device_role_config_title')}</DialogTitle></DialogHeader>
          <div className={roleLoading ? 'opacity-60 pointer-events-none grid gap-4 py-2' : 'grid gap-4 py-2'}>
            <div className="flex gap-2 p-3 rounded-lg border text-sm status-primary">
              <span>{t('role_config_note')}: {t('role_override_desc')}</span>
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">{t('current_role_label')}</label>
              {currentDevice?.role_id ? (
                <div className="text-sm">
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs border status-success">{t('role_linked')}</span>
                  <p className="mt-1 text-[var(--color-text-secondary)]"><strong>{t('role_id_label')}</strong> {currentDevice.role_id}</p>
                </div>
              ) : (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-2)] border border-[var(--color-line)] text-[var(--color-text-secondary)]">{t('role_not_linked')}</span>
              )}
            </div>
            <div className="grid gap-1.5">
              <label className="text-sm font-medium text-[var(--color-text)]">{t('select_role_label')}</label>
              <Select value={selectedRoleId ? String(selectedRoleId) : '__none__'} onValueChange={(v) => setSelectedRoleId(v === '__none__' ? null : Number(v))}>
                <SelectTrigger className="w-full"><SelectValue placeholder={t('select_role_opt_ph')} /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">{t('no_agent_linked')}</SelectItem>
                  {availableRoles.map((r) => <SelectItem key={r.id} value={String(r.id)}>{r.name}</SelectItem>)}
                </SelectContent>
              </Select>
              <p className="text-xs text-[var(--color-text-tertiary)]">{t('role_override_help')}</p>
            </div>
            {selectedRole && (
              <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-1)] p-4 grid gap-2 text-sm">
                <p><strong className="text-[var(--color-text)]">{t('name_colon')}</strong> {selectedRole.name}</p>
                <hr className="border-[var(--color-line)]" />
                <p className="font-semibold">{t('prompt')}:</p>
                <p className="text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-2)] p-3 rounded-lg whitespace-pre-wrap">
                  {selectedRole.prompt ? selectedRole.prompt.substring(0, 200) + (selectedRole.prompt.length > 200 ? '...' : '') : '—'}
                </p>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRoleDialog(false)}>{t('cancel')}</Button>
            <Button disabled={roleLoading || (!selectedRoleId && !currentDevice?.role_id)} onClick={applyRole}>
              {selectedRoleId ? t('apply_role') : t('cancel_role')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Voice push dialog */}
      <MessageInjectDialog
        open={showVoicePushDialog}
        onOpenChange={setShowVoicePushDialog}
        devices={devices}
        defaultDeviceId={voicePushDeviceId}
        lockDevice={!!voicePushDeviceId}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={confirmDelete}
        isLoading={deleting}
        title={t('confirm_delete')}
        description={t('confirm_delete_device_msg', { name: deleteTarget?.nick_name || deleteTarget?.device_name || '' })}
      />
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/user/devices')({
  component: UserDevicesPage,
})
