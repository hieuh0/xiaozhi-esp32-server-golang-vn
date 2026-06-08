// Composable: all state + API calls for AgentDevices page
import { ref, computed, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../utils/api'
import { createDefaultDeviceForm } from './useAgentFormOptions'
import { useLocale } from './useLocale'

export function useAgentDevices() {
  const { t } = useLocale()
  const router = useRouter()
  const route = useRoute()

  // --- Route-derived state ---
  const routeAgentId = computed(() => route.params.id ? String(route.params.id) : '')
  const showBackButton = computed(() => !!routeAgentId.value)
  const filterAgentId = ref(String(route.params.id || route.query.agent_id || ''))
  const bindingAgentId = computed(() => filterAgentId.value || null)

  // --- Core state ---
  const agents = ref([])
  const devices = ref([])
  const showAddDeviceDialog = ref(false)
  const addingDevice = ref(false)
  const deviceFormRef = ref(null)
  const deviceForm = ref(createDefaultDeviceForm({ mode: 'bind' }))

  // --- Voice push ---
  const showVoicePushDialog = ref(false)
  const voicePushDeviceId = ref('')

  // --- Inline rename ---
  const editingDeviceId = ref(null)
  const editingDeviceName = ref('')
  const renamingDeviceId = ref(null)
  const deviceNameInputRef = ref(null)

  // --- MCP dialog ---
  const showMcpDialog = ref(false)
  const mcpLoading = ref(false)
  const toolsLoading = ref(false)
  const callingTool = ref(false)
  const currentDeviceId = ref(null)
  const mcpTools = ref([])
  const mcpCallResult = ref('')
  const mcpCallForm = ref({ tool_name: '', argumentsText: '{}' })

  // --- Role config dialog ---
  const showRoleConfigDialog = ref(false)
  const roleConfigLoading = ref(false)
  const currentDevice = ref({})
  const selectedRoleId = ref(null)
  const selectedRole = ref(null)
  const availableRoles = ref([])

  const isRoleActive = (role) => role?.status === 'active' || !role?.status

  // --- Computed ---
  const agentNameMap = computed(() => {
    const map = new Map()
    for (const agent of agents.value) {
      map.set(String(agent.id), agent.name || t('agent_id_param', { id: agent.id }))
    }
    return map
  })

  const selectedAgentName = computed(() => {
    if (!filterAgentId.value) return ''
    return agentNameMap.value.get(String(filterAgentId.value)) || t('agent_id_param', { id: filterAgentId.value })
  })

  const filteredDevices = computed(() => {
    if (!filterAgentId.value) return devices.value
    return devices.value.filter(device => String(device.agent_id || '') === String(filterAgentId.value))
  })

  const emptyDescription = computed(() => {
    return selectedAgentName.value ? t('agent_no_device') : t('no_device_bound')
  })

  // --- API calls ---
  const loadAgents = async () => {
    try {
      const response = await api.get('/user/agents')
      agents.value = response.data.data || []
    } catch (error) {
      agents.value = []
      ElMessage.error(t('load_agent_list_failed'))
    }
  }

  const loadDevices = async () => {
    try {
      const response = await api.get('/user/devices')
      devices.value = response.data.data || []
    } catch (error) {
      ElMessage.error(t('load_device_list_failed'))
    }
  }

  const loadRoles = async () => {
    try {
      const response = await api.get('/user/roles')
      const globalRoles = response.data.data?.global_roles || []
      const userRoles = response.data.data?.user_roles || []
      availableRoles.value = [...globalRoles, ...userRoles].filter(isRoleActive)
    } catch (error) {
      console.error(t('load_role_list_failed'), error)
    }
  }

  // --- Device form ---
  const resetAddDeviceForm = () => {
    deviceForm.value = createDefaultDeviceForm({ mode: 'bind', fixedAgentId: bindingAgentId.value || null })
    deviceFormRef.value?.clearValidate?.()
  }

  const openAddDeviceDialog = () => {
    if (!agents.value.length) { ElMessage.warning(t('create_agent_before_bind')); return }
    resetAddDeviceForm()
    showAddDeviceDialog.value = true
  }

  const handleAddDevice = async () => {
    if (!deviceFormRef.value) return
    try { await deviceFormRef.value.validate() } catch { return }
    const agentId = bindingAgentId.value || deviceForm.value.agent_id
    if (!agentId) { ElMessage.warning(t('select_target_agent')); return }
    addingDevice.value = true
    try {
      const response = await api.post(`/user/agents/${agentId}/devices`, deviceFormRef.value.buildPayload())
      if (response.data?.success) {
        ElMessage.success(t('device_bind_success'))
        showAddDeviceDialog.value = false
        await loadDevices()
      }
    } catch (error) {
      ElMessage.error(error.response?.data?.error || t('device_bind_failed'))
    } finally { addingDevice.value = false }
  }

  // --- Voice push ---
  const handleVoicePush = (device) => {
    if (!device?.device_name) { ElMessage.warning(t('device_no_id_notify')); return }
    voicePushDeviceId.value = device.device_name
    showVoicePushDialog.value = true
  }

  const handleVoicePushSuccess = () => { voicePushDeviceId.value = '' }

  // --- Filter ---
  const handleAgentFilterChange = (value) => {
    const query = value ? { agent_id: value } : {}
    router.replace({ path: '/user/devices', query })
  }

  // --- Display helpers ---
  const getDeviceAgentName = (device) => {
    if (!device?.agent_id) return t('not_bound')
    return device.agent_name || agentNameMap.value.get(String(device.agent_id)) || t('agent_id_param', { id: device.agent_id })
  }

  const getDeviceDisplayName = (device) => {
    const nickName = String(device?.nick_name || '').trim()
    if (nickName) return nickName
    return String(device?.device_name || '').trim() || t('unnamed_device')
  }

  const getDeviceIdentityText = (device) => {
    const deviceId = String(device?.device_name || '').trim() || '-'
    return t('device_id_display', { id: deviceId })
  }

  // --- Inline rename ---
  const startDeviceNameEdit = (device) => {
    editingDeviceId.value = device.id
    editingDeviceName.value = String(device.nick_name || '').trim() || getDeviceDisplayName(device)
    nextTick(() => deviceNameInputRef.value?.focus?.())
  }

  const cancelDeviceNameEdit = () => { editingDeviceId.value = null; editingDeviceName.value = '' }

  const saveDeviceName = async (device) => {
    const name = editingDeviceName.value.trim()
    if (!name) { ElMessage.warning(t('device_nickname_required')); return }
    if (name === String(device.nick_name || '').trim()) { cancelDeviceNameEdit(); return }
    renamingDeviceId.value = device.id
    try {
      const response = await api.put(`/user/devices/${device.id}`, { nick_name: name })
      const updatedDevice = response.data?.data || {}
      const target = devices.value.find(item => item.id === device.id)
      if (target) target.nick_name = updatedDevice.nick_name || name
      ElMessage.success(t('device_nickname_updated'))
      cancelDeviceNameEdit()
    } catch (error) {
      ElMessage.error(error.response?.data?.error || t('update_device_nickname_failed'))
    } finally { renamingDeviceId.value = null }
  }

  // --- MCP ---
  const handleDeviceMcp = async (device) => {
    currentDeviceId.value = device.id
    showMcpDialog.value = true
    mcpLoading.value = true
    mcpCallResult.value = ''
    mcpCallForm.value = { tool_name: '', argumentsText: '{}' }
    try { await refreshDeviceMcpTools() } finally { mcpLoading.value = false }
  }

  const refreshDeviceMcpTools = async () => {
    if (!currentDeviceId.value) return
    toolsLoading.value = true
    try {
      const response = await api.get(`/user/devices/${currentDeviceId.value}/mcp-tools`)
      mcpTools.value = response.data.data?.tools || []
      if (!mcpCallForm.value.tool_name && mcpTools.value.length > 0) mcpCallForm.value.tool_name = mcpTools.value[0].name
    } catch (error) { ElMessage.error(t('fetch_device_mcp_failed')); mcpTools.value = [] }
    finally { toolsLoading.value = false }
  }

  const buildExampleFromSchema = (schema = {}) => {
    if (!schema || typeof schema !== 'object') return {}
    if (Array.isArray(schema.enum) && schema.enum.length > 0) return schema.enum[0]
    const type = schema.type || 'object'
    if (type === 'object') { const props = schema.properties || {}; const result = {}; Object.keys(props).sort().forEach(key => { result[key] = buildExampleFromSchema(props[key]) }); return result }
    if (type === 'array') return [buildExampleFromSchema(schema.items || {})]
    if (type === 'number') return 0.1
    if (type === 'integer') return 0
    if (type === 'boolean') return false
    return ''
  }

  const handleMcpToolChange = (toolName) => {
    const selectedTool = mcpTools.value.find(item => item.name === toolName)
    if (selectedTool) mcpCallForm.value.argumentsText = JSON.stringify(buildExampleFromSchema(selectedTool.input_schema || {}), null, 2)
  }

  const formatMcpCallResult = (payload) => {
    const MAX_PARSE_DEPTH = 8
    const tryParseJSONString = (value) => {
      if (typeof value !== 'string') return { parsed: false, value }
      let text = value.trim()
      if (!text) return { parsed: false, value }
      const fenced = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i)
      if (fenced) text = fenced[1].trim()
      const looksLikeJSON = (text.startsWith('{') && text.endsWith('}')) || (text.startsWith('[') && text.endsWith(']'))
      if (!looksLikeJSON) return { parsed: false, value }
      try { return { parsed: true, value: JSON.parse(text) } } catch (_) { return { parsed: false, value } }
    }
    const deepParseJSONStrings = (value, depth = 0) => {
      if (depth >= MAX_PARSE_DEPTH || value == null) return value
      if (typeof value === 'string') { const parsed = tryParseJSONString(value); if (!parsed.parsed) return value; return deepParseJSONStrings(parsed.value, depth + 1) }
      if (Array.isArray(value)) return value.map(item => deepParseJSONStrings(item, depth + 1))
      if (typeof value === 'object') {
        const out = {}
        Object.keys(value).forEach(key => { out[key] = deepParseJSONStrings(value[key], depth + 1) })
        if (Array.isArray(out.content) && out.content.length === 1) {
          const first = out.content[0]
          if (first && typeof first === 'object' && !Array.isArray(first) && first.type === 'text' && Object.prototype.hasOwnProperty.call(first, 'text')) {
            const textValue = first.text
            if (textValue && typeof textValue === 'object') return textValue
          }
        }
        return out
      }
      return value
    }
    const data = payload ?? {}
    const raw = (data && typeof data === 'object' && !Array.isArray(data) && Object.prototype.hasOwnProperty.call(data, 'result')) ? data.result : data
    return JSON.stringify(deepParseJSONStrings(raw), null, 2)
  }

  const callDeviceMcpTool = async () => {
    if (!currentDeviceId.value || !mcpCallForm.value.tool_name) { ElMessage.warning(t('select_tool')); return }
    let argumentsObj = {}
    try { argumentsObj = mcpCallForm.value.argumentsText ? JSON.parse(mcpCallForm.value.argumentsText) : {} }
    catch (e) { ElMessage.error(t('params_json_format_error')); return }
    callingTool.value = true
    try {
      const response = await api.post(`/user/devices/${currentDeviceId.value}/mcp-call`, { tool_name: mcpCallForm.value.tool_name, arguments: argumentsObj })
      mcpCallResult.value = formatMcpCallResult(response.data.data || {})
      ElMessage.success(t('mcp_tool_call_success'))
    } catch (error) {
      mcpCallResult.value = JSON.stringify(error.response?.data || { error: error.message }, null, 2)
      ElMessage.error(t('mcp_tool_call_failed'))
    } finally { callingTool.value = false }
  }

  // --- Role config ---
  const handleDeviceRole = async (deviceId) => {
    const device = devices.value.find(d => d.id === deviceId)
    if (!device) return
    currentDevice.value = { ...device }
    selectedRoleId.value = device.role_id || null
    selectedRole.value = null
    if (availableRoles.value.length === 0) await loadRoles()
    if (device.role_id) { const role = availableRoles.value.find(r => r.id === device.role_id); if (role) selectedRole.value = role }
    showRoleConfigDialog.value = true
  }

  const handleRoleSelect = (roleId) => {
    if (!roleId) { selectedRole.value = null; return }
    const role = availableRoles.value.find(r => r.id === roleId)
    if (role) selectedRole.value = role
  }

  const handleApplyRole = async () => {
    if (!currentDevice.value.id) return
    roleConfigLoading.value = true
    try {
      await api.post(`/devices/${currentDevice.value.id}/apply-role`, { role_id: selectedRoleId.value || null })
      ElMessage.success(selectedRoleId.value ? t('role_applied_to_device') : t('device_role_cancelled'))
      showRoleConfigDialog.value = false
      await loadDevices()
    } catch (error) {
      ElMessage.error(t('operation_failed_colon') + ' ' + (error.response?.data?.error || error.message))
    } finally { roleConfigLoading.value = false }
  }

  const handleCloseRoleConfig = () => { showRoleConfigDialog.value = false; currentDevice.value = {}; selectedRoleId.value = null; selectedRole.value = null }

  // --- Delete device ---
  const handleDeleteDevice = async (device) => {
    if (!device?.id) return
    try {
      await ElMessageBox.confirm(
        t('confirm_delete_device_msg', { name: getDeviceDisplayName(device) }),
        t('confirm_delete_device'),
        { confirmButtonText: t('delete'), cancelButtonText: t('cancel'), type: 'warning' }
      )
      const response = await api.delete(`/user/devices/${device.id}`)
      if (response.data.success) { ElMessage.success(response.data.message || t('device_deleted')); await loadDevices() }
    } catch (error) {
      if (error !== 'cancel') ElMessage.error(error.response?.data?.error || t('delete_device_failed'))
    }
  }

  const goBack = () => router.push('/agents')

  const formatDate = (dateString) => { if (!dateString) return t('never'); return new Date(dateString).toLocaleString('zh-CN') }
  const isDeviceOnline = (lastActiveAt) => { if (!lastActiveAt) return false; return (new Date() - new Date(lastActiveAt)) < 5 * 60 * 1000 }

  return {
    // State
    routeAgentId, showBackButton, filterAgentId, bindingAgentId,
    agents, devices, showAddDeviceDialog, addingDevice, deviceFormRef, deviceForm,
    showVoicePushDialog, voicePushDeviceId,
    editingDeviceId, editingDeviceName, renamingDeviceId, deviceNameInputRef,
    showMcpDialog, mcpLoading, toolsLoading, callingTool, currentDeviceId, mcpTools, mcpCallResult, mcpCallForm,
    showRoleConfigDialog, roleConfigLoading, currentDevice, selectedRoleId, selectedRole, availableRoles,
    // Computed
    agentNameMap, selectedAgentName, filteredDevices, emptyDescription,
    // Methods
    loadAgents, loadDevices, loadRoles,
    resetAddDeviceForm, openAddDeviceDialog, handleAddDevice,
    handleVoicePush, handleVoicePushSuccess, handleAgentFilterChange,
    getDeviceAgentName, getDeviceDisplayName, getDeviceIdentityText,
    startDeviceNameEdit, cancelDeviceNameEdit, saveDeviceName,
    handleDeviceMcp, refreshDeviceMcpTools, handleMcpToolChange, callDeviceMcpTool,
    handleDeviceRole, handleRoleSelect, handleApplyRole, handleCloseRoleConfig,
    handleDeleteDevice, goBack, formatDate, isDeviceOnline
  }
}
