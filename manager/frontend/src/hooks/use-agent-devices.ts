import { useCallback, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'
import { devicesApi } from '@/features/devices/api/devices-api'
import type { Device, DeviceFormData, McpTool, Role } from '@/features/devices/types'
import { createDefaultDeviceForm, getDeviceDisplayName } from '@/features/devices/types'
import type { Agent } from '@/features/agents/types'
import { agentsApi } from '@/features/agents/api/agents-api'

export interface AgentDevicesState {
  agents: Agent[]
  devices: Device[]
  filterAgentId: string
  // add device
  showAddDialog: boolean
  addingDevice: boolean
  deviceForm: DeviceFormData
  // voice push
  showVoicePushDialog: boolean
  voicePushDeviceId: string
  // inline rename
  editingDeviceId: number | null
  editingDeviceName: string
  renamingDeviceId: number | null
  // mcp dialog
  showMcpDialog: boolean
  mcpLoading: boolean
  toolsLoading: boolean
  callingTool: boolean
  currentDeviceId: number | null
  mcpTools: McpTool[]
  mcpCallResult: string
  mcpToolName: string
  mcpToolArgs: string
  // role config dialog
  showRoleDialog: boolean
  roleLoading: boolean
  currentDevice: Device | null
  selectedRoleId: number | null
  availableRoles: Role[]
}

export function useAgentDevices(initialFilterAgentId = '') {
  const [agents, setAgents] = useState<Agent[]>([])
  const [devices, setDevices] = useState<Device[]>([])
  const [filterAgentId, setFilterAgentId] = useState(initialFilterAgentId)

  const [showAddDialog, setShowAddDialog] = useState(false)
  const [addingDevice, setAddingDevice] = useState(false)
  const [deviceForm, setDeviceForm] = useState<DeviceFormData>(createDefaultDeviceForm())

  const [showVoicePushDialog, setShowVoicePushDialog] = useState(false)
  const [voicePushDeviceId, setVoicePushDeviceId] = useState('')

  const [editingDeviceId, setEditingDeviceId] = useState<number | null>(null)
  const [editingDeviceName, setEditingDeviceName] = useState('')
  const [renamingDeviceId, setRenamingDeviceId] = useState<number | null>(null)

  const [showMcpDialog, setShowMcpDialog] = useState(false)
  const [mcpLoading, setMcpLoading] = useState(false)
  const [toolsLoading] = useState(false)
  const [callingTool, setCallingTool] = useState(false)
  const [currentDeviceId, setCurrentDeviceId] = useState<number | null>(null)
  const [mcpTools, setMcpTools] = useState<McpTool[]>([])
  const [mcpCallResult, setMcpCallResult] = useState('')
  const [mcpToolName, setMcpToolName] = useState('')
  const [mcpToolArgs, setMcpToolArgs] = useState('{}')

  const [showRoleDialog, setShowRoleDialog] = useState(false)
  const [roleLoading, setRoleLoading] = useState(false)
  const [currentDevice, setCurrentDevice] = useState<Device | null>(null)
  const [selectedRoleId, setSelectedRoleId] = useState<number | null>(null)
  const [availableRoles, setAvailableRoles] = useState<Role[]>([])

  const nameInputRef = useRef<HTMLInputElement>(null)

  const filteredDevices = useMemo(() =>
    filterAgentId ? devices.filter((d) => String(d.agent_id || '') === filterAgentId) : devices,
    [devices, filterAgentId])

  const agentNameMap = useMemo(() => new Map(agents.map((a) => [String(a.id), a.name])), [agents])

  const getDeviceAgentName = useCallback((device: Device) => {
    if (!device.agent_id) return 'Not bound'
    return device.agent_name || agentNameMap.get(String(device.agent_id)) || `Agent #${device.agent_id}`
  }, [agentNameMap])

  const loadAll = useCallback(async () => {
    const [agentList, deviceList] = await Promise.all([
      agentsApi.getUserAgents().catch(() => []),
      devicesApi.getUserDevices().catch(() => []),
    ])
    setAgents(agentList)
    setDevices(deviceList)
  }, [])

  const openAddDialog = useCallback((fixedAgentId?: string) => {
    setDeviceForm(createDefaultDeviceForm({ fixedAgentId: fixedAgentId ? Number(fixedAgentId) : null }))
    setShowAddDialog(true)
  }, [])

  const handleAddDevice = useCallback(async () => {
    const agentId = deviceForm.agent_id
    if (!agentId) { toast.error('Select target agent'); return }
    const identifier = deviceForm.identifier.trim()
    if (!identifier) { toast.error('Enter device ID or MAC address'); return }
    setAddingDevice(true)
    try {
      await devicesApi.addDevice(agentId, { identifier, nick_name: (deviceForm.nick_name ?? '').trim() || undefined })
      toast.success('Device bound successfully')
      setShowAddDialog(false)
      setDevices(await devicesApi.getUserDevices())
    } catch (e) { toast.error((e as Error).message || 'Failed to bind device') }
    finally { setAddingDevice(false) }
  }, [deviceForm])

  const startNameEdit = useCallback((device: Device) => {
    setEditingDeviceId(device.id)
    setEditingDeviceName(String(device.nick_name || '').trim() || getDeviceDisplayName(device))
    setTimeout(() => nameInputRef.current?.focus(), 50)
  }, [])

  const cancelNameEdit = useCallback(() => { setEditingDeviceId(null); setEditingDeviceName('') }, [])

  const saveDeviceName = useCallback(async (device: Device) => {
    const name = editingDeviceName.trim()
    if (!name) { toast.error('Nickname required'); return }
    if (name === String(device.nick_name || '').trim()) { cancelNameEdit(); return }
    setRenamingDeviceId(device.id)
    try {
      await devicesApi.updateDevice(device.id, { nick_name: name })
      setDevices((prev) => prev.map((d) => d.id === device.id ? { ...d, nick_name: name } : d))
      toast.success('Nickname updated')
      cancelNameEdit()
    } catch (e) { toast.error((e as Error).message || 'Failed to update nickname') }
    finally { setRenamingDeviceId(null) }
  }, [editingDeviceName, cancelNameEdit])

  const handleDeleteDevice = useCallback(async (device: Device) => {
    try {
      await devicesApi.deleteDevice(device.id)
      toast.success('Device deleted')
      setDevices((prev) => prev.filter((d) => d.id !== device.id))
    } catch (e) { toast.error((e as Error).message || 'Failed to delete device') }
  }, [])

  const openMcpDialog = useCallback(async (device: Device) => {
    setCurrentDeviceId(device.id)
    setMcpCallResult(''); setMcpToolName(''); setMcpToolArgs('{}')
    setShowMcpDialog(true); setMcpLoading(true)
    try {
      const tools = await devicesApi.getDeviceMcpTools(device.id)
      setMcpTools(tools)
      if (tools.length) setMcpToolName(tools[0].name)
    } catch { toast.error('Failed to load MCP tools') }
    finally { setMcpLoading(false) }
  }, [])

  const callMcpTool = useCallback(async () => {
    if (!currentDeviceId || !mcpToolName) { toast.error('Select a tool'); return }
    let args: Record<string, unknown> = {}
    try { args = JSON.parse(mcpToolArgs) } catch { toast.error('Invalid JSON arguments'); return }
    setCallingTool(true)
    try {
      const result = await devicesApi.callDeviceMcpTool(currentDeviceId, mcpToolName, args)
      setMcpCallResult(typeof result === 'string' ? result : JSON.stringify(result, null, 2))
    } catch (e) { setMcpCallResult((e as Error).message) }
    finally { setCallingTool(false) }
  }, [currentDeviceId, mcpToolName, mcpToolArgs])

  const openRoleDialog = useCallback(async (device: Device) => {
    setCurrentDevice(device)
    setSelectedRoleId(device.role_id || null)
    setShowRoleDialog(true)
    if (!availableRoles.length) {
      try { setAvailableRoles(await devicesApi.getRoles()) } catch { /* ignore */ }
    }
  }, [availableRoles.length])

  const applyRole = useCallback(async () => {
    if (!currentDevice) return
    setRoleLoading(true)
    try {
      await devicesApi.applyRole(currentDevice.id, selectedRoleId)
      toast.success(selectedRoleId ? 'Role applied' : 'Role removed')
      setShowRoleDialog(false)
      setDevices(await devicesApi.getUserDevices())
    } catch (e) { toast.error((e as Error).message || 'Operation failed') }
    finally { setRoleLoading(false) }
  }, [currentDevice, selectedRoleId])

  const openVoicePush = useCallback((device: Device) => {
    if (!device.device_name) { toast.error('Device has no ID'); return }
    setVoicePushDeviceId(device.device_name)
    setShowVoicePushDialog(true)
  }, [])

  return {
    agents, devices, filteredDevices, filterAgentId, setFilterAgentId,
    showAddDialog, setShowAddDialog, addingDevice, deviceForm, setDeviceForm,
    showVoicePushDialog, setShowVoicePushDialog, voicePushDeviceId,
    editingDeviceId, editingDeviceName, setEditingDeviceName, renamingDeviceId, nameInputRef,
    showMcpDialog, setShowMcpDialog, mcpLoading, toolsLoading, callingTool,
    mcpTools, mcpCallResult, mcpToolName, setMcpToolName, mcpToolArgs, setMcpToolArgs,
    showRoleDialog, setShowRoleDialog, roleLoading, currentDevice, selectedRoleId, setSelectedRoleId, availableRoles,
    getDeviceAgentName, agentNameMap,
    loadAll, openAddDialog, handleAddDevice,
    startNameEdit, cancelNameEdit, saveDeviceName, handleDeleteDevice,
    openMcpDialog, callMcpTool,
    openRoleDialog, applyRole,
    openVoicePush,
  }
}
