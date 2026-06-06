<template>
  <div class="agent-devices-page">
    <div class="devices-filter-bar">
      <div class="filter-controls">
        <el-button v-if="showBackButton" @click="goBack" type="text" class="back-btn">
          <el-icon><ArrowLeft /></el-icon>
          {{ t('back') }}</el-button>
        <el-select
          v-model="filterAgentId"
          :placeholder="t('filter_by_agent')"
          clearable
          filterable
          class="agent-filter-select"
          @change="handleAgentFilterChange"
        >
          <el-option :label="t('all_devices')" value="" />
          <el-option
            v-for="agent in agents"
            :key="agent.id"
            :label="agent.name"
            :value="String(agent.id)"
          />
        </el-select>
        <span class="devices-count">{{ t('device_count_n', { count: filteredDevices.length }) }}</span>
      </div>
      <el-button class="add-device-button" type="primary" @click="openAddDeviceDialog">
        <el-icon><Plus /></el-icon>
        {{ t('add_device') }}</el-button>
    </div>

    <div v-if="filteredDevices.length === 0" class="empty-section">
      <el-card class="empty-card">
        <div class="empty-content">
          <el-icon size="64" color="var(--apple-text-tertiary)"><Monitor /></el-icon>
          <h3>{{ t('no_devices') }}</h3>
          <p>{{ emptyDescription }}</p>
          <div class="empty-actions">
            <el-button type="primary" size="large" @click="openAddDeviceDialog">
              <el-icon><Plus /></el-icon>
              {{ t('add_first_device') }}</el-button>
          </div>
        </div>
      </el-card>
    </div>

    <div v-else class="devices-grid">
      <div v-for="device in filteredDevices" :key="device.id" class="device-item">
        <div class="device-card">
          <div class="device-header">
            <div class="device-icon">
              <el-icon size="28"><Monitor /></el-icon>
            </div>
            <div class="device-info">
              <div class="device-name-row">
	                <el-input
	                  v-if="editingDeviceId === device.id"
	                  ref="deviceNameInputRef"
	                  v-model="editingDeviceName"
	                  class="device-name-input"
	                  size="small"
                  maxlength="50"
                  show-word-limit
                  :placeholder="t('enter_device_nickname')"
	                  @keydown.enter.prevent="saveDeviceName(device)"
	                  @keydown.esc.prevent="cancelDeviceNameEdit"
	                />
	                <button
	                  v-else
	                  type="button"
	                  class="device-name-button"
	                  :title="t('click_edit_nickname', { name: getDeviceDisplayName(device) })"
	                  @click="startDeviceNameEdit(device)"
	                >
	                  <span class="device-name">{{ getDeviceDisplayName(device) }}</span>
	                </button>
	                <div class="device-name-actions">
	                  <template v-if="editingDeviceId === device.id">
	                    <el-button
	                      class="name-action-button"
	                      type="primary"
	                      :icon="Check"
	                      circle
	                      :loading="renamingDeviceId === device.id"
	                      :title="t('save_nickname')"
	                      @click="saveDeviceName(device)"
	                    />
	                    <el-button
	                      class="name-action-button"
	                      :icon="Close"
	                      circle
	                      :title="t('cancel_edit')"
	                      @click="cancelDeviceNameEdit"
	                    />
	                  </template>
	                  <el-button
	                    v-else
	                    class="rename-icon-button"
	                    :icon="EditPen"
	                    circle
	                    :title="t('edit_nickname_title')"
	                    @click="startDeviceNameEdit(device)"
	                  />
	                </div>
              </div>
              <p class="device-identity" :title="getDeviceIdentityText(device)">{{ getDeviceIdentityText(device) }}</p>
            </div>
            <div class="device-status">
              <span :class="['status-dot', isDeviceOnline(device.last_active_at) ? 'online' : 'offline']"></span>
              <span class="status-text">{{ isDeviceOnline(device.last_active_at) ? t('online') : t('offline') }}</span>
            </div>
          </div>
          
          <div class="device-meta">
            <div class="meta-row">
              <span class="meta-label">{{ t('link_agent') }}</span>
              <span class="meta-value">{{ getDeviceAgentName(device) }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('device_type_label') }}</span>
              <span class="meta-value">{{ t('esp32_device') }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('activation_status') }}</span>
              <span class="meta-value">
                <el-tag :type="device.activated ? 'success' : 'warning'" size="small">
                  {{ device.activated ? t('activated') : t('not_activated') }}
                </el-tag>
              </span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('last_active') }}</span>
              <span class="meta-value">{{ formatDate(device.last_active_at) }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('created_at') }}</span>
              <span class="meta-value">{{ formatDate(device.created_at) }}</span>
            </div>
          </div>
          
          <div class="device-actions">
            <el-button class="device-action-button device-action-button-feature" size="small" @click="handleDeviceRole(device.id)">
              <el-icon><User /></el-icon>
              {{ t('role') }}
            </el-button>
            <el-button class="device-action-button" size="small" @click="handleDeviceMcp(device)">
              <el-icon><Setting /></el-icon>
              MCP
            </el-button>
            <el-button class="device-action-button device-action-button-voice" size="small" @click="handleVoicePush(device)">
              <el-icon><ChatDotRound /></el-icon>{{ t('voice_notify') }}</el-button>
            <el-button
              class="device-action-button device-action-button-danger"
              size="small"
              @click="handleDeleteDevice(device)"
            >
              <el-icon><Delete /></el-icon>
              {{ t('delete') }}
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="showAddDeviceDialog"
      :title="t('bind_device')"
      width="520px"
      :close-on-click-modal="false"
      @closed="resetAddDeviceForm"
    >
      <DeviceForm
        ref="deviceFormRef"
        v-model="deviceForm"
        mode="bind"
        :agents="agents"
        :fixed-agent-id="bindingAgentId"
      />
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDeviceDialog = false">{{ t('cancel') }}</el-button>
          <el-button type="primary" :loading="addingDevice" @click="handleAddDevice">
            {{ addingDevice ? t('binding') : t('bind_device') }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <MessageInjectDialog
      v-model="showVoicePushDialog"
      :devices="devices"
      :default-device-id="voicePushDeviceId"
      :lock-device="Boolean(voicePushDeviceId)"
      @success="handleVoicePushSuccess"
    />

    <!-- Device MCP dialog -->

    <el-dialog
      v-model="showMcpDialog"
      :title="t('device_mcp_tools')"
      width="760px"
    >
      <div v-loading="mcpLoading">
        <div class="mcp-tools-header">
          <el-button size="small" type="primary" @click="refreshDeviceMcpTools" :loading="toolsLoading">{{ t('refresh_tool_list') }}</el-button>
        </div>

        <div v-if="mcpTools.length === 0" class="tools-empty">{{ t('no_tool_data') }}</div>
        <div v-else class="tools-tags">
          <el-tag v-for="tool in mcpTools" :key="tool.name" class="tool-tag">{{ tool.name }}</el-tag>
        </div>

        <el-divider />
        <el-form :model="mcpCallForm" label-width="90px">
          <el-form-item :label="t('tool')">
            <el-select v-model="mcpCallForm.tool_name" :placeholder="t('select_tool')" style="width:100%" @change="handleMcpToolChange">
              <el-option v-for="tool in mcpTools" :key="tool.name" :label="tool.name" :value="tool.name" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('args_json_label')">
            <el-input v-model="mcpCallForm.argumentsText" type="textarea" :rows="6" :placeholder="t('mcp_args_placeholder')" />
          </el-form-item>
        </el-form>

        <el-button type="primary" @click="callDeviceMcpTool" :loading="callingTool">{{ t('call_tool') }}</el-button>

        <el-divider />
        <div class="mcp-result-box">{{ mcpCallResult || t('no_call_results') }}</div>
      </div>
    </el-dialog>


    <!-- Device role configuration dialog -->
    <el-dialog
      v-model="showRoleConfigDialog"
      :title="t('device_role_config_title')"
      width="700px"
      @close="handleCloseRoleConfig"
    >
      <div v-loading="roleConfigLoading">
        <div class="role-config-content">
          <el-alert
            :title="t('role_config_note')"
            type="info"
            :closable="false"
            style="margin-bottom: 16px"
          >
            {{ t('role_override_desc') }}
          </el-alert>

          <el-form label-width="120px">
            <el-form-item :label="t('current_role_label')">
              <div v-if="currentDevice.role_id">
                <el-tag type="success" size="large">{{ t('role_linked') }}</el-tag>
                <div class="current-role-info">
                  <p><strong>{{ t('role_id_label') }}</strong> {{ currentDevice.role_id }}</p>
                </div>
              </div>
              <el-tag v-else type="info" size="large">{{ t('role_not_linked') }}</el-tag>
            </el-form-item>

            <el-form-item :label="t('select_role_label')">
              <el-select
                v-model="selectedRoleId"
                :placeholder="t('select_role_opt_ph')"
                style="width: 100%"
                clearable
                filterable
                @change="handleRoleSelect"
              >
                <el-option
                  v-for="role in availableRoles"
                  :key="role.id"
                  :label="role.name"
                  :value="role.id"
                >
                  <div class="role-option-item">
                    <div class="role-option-main">
                      <span>{{ role.name }}</span>
                      <el-tag v-if="role.role_type === 'global'" size="small" type="success">{{ t('global') }}</el-tag>
                    </div>
                    <el-tag size="small" type="info">LLM: {{ role.llm_config_id || t('default') }}</el-tag>
                  </div>
                </el-option>
              </el-select>
              <div class="form-help">{{ t('role_override_help') }}</div>
            </el-form-item>

            <el-form-item :label="t('role_detail_label')" v-if="selectedRole">
              <el-card class="role-preview-card">
                <div class="role-preview-content">
                  <p><strong>{{ t('name_colon') }}</strong> {{ selectedRole.name }}</p>
                  <p v-if="selectedRole.description"><strong>{{ t('desc_colon') }}</strong> {{ selectedRole.description }}</p>
                  <el-divider />
                  <p><strong>Prompt:</strong></p>
                  <p class="prompt-preview">{{ selectedRole.prompt.substring(0, 200) }}{{ selectedRole.prompt.length > 200 ? '...' : '' }}</p>
                  <div class="role-configs-preview">
                    <el-tag size="small">LLM: {{ selectedRole.llm_config_id || t('default') }}</el-tag>
                    <el-tag size="small">TTS: {{ selectedRole.tts_config_id || t('default') }}</el-tag>
                    <el-tag v-if="selectedRole.voice" size="small">{{ t('voice_colon') }}{{ selectedRole.voice }}</el-tag>
                  </div>
                </div>
              </el-card>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <template #footer>
        <el-button @click="handleCloseRoleConfig">{{ t('cancel') }}</el-button>
        <el-button
          type="primary"
          @click="handleApplyRole"
          :loading="roleConfigLoading"
          :disabled="!selectedRoleId && !currentDevice.role_id"
        >
          {{ selectedRoleId ? t('apply_role') : t('cancel_role') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, nextTick, ref, reactive, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Plus, Monitor, Setting, Delete, User, ChatDotRound, EditPen, Check, Close } from '@element-plus/icons-vue'
import api from '../../utils/api'
import DeviceForm from '../../components/common/DeviceForm.vue'
import MessageInjectDialog from '../../components/user/MessageInjectDialog.vue'
import { createDefaultDeviceForm } from '../../composables/useAgentFormOptions'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const router = useRouter()
const route = useRoute()

const routeAgentId = computed(() => route.params.id ? String(route.params.id) : '')
const showBackButton = computed(() => !!routeAgentId.value)
const filterAgentId = ref(String(route.params.id || route.query.agent_id || ''))
const bindingAgentId = computed(() => filterAgentId.value || null)
const agents = ref([])
const devices = ref([])
const showAddDeviceDialog = ref(false)
const addingDevice = ref(false)
const deviceFormRef = ref(null)
const deviceForm = ref(createDefaultDeviceForm({ mode: 'bind' }))
const showVoicePushDialog = ref(false)
const voicePushDeviceId = ref('')
const editingDeviceId = ref(null)
const editingDeviceName = ref('')
const renamingDeviceId = ref(null)
const deviceNameInputRef = ref(null)

const showMcpDialog = ref(false)
const mcpLoading = ref(false)
const toolsLoading = ref(false)
const callingTool = ref(false)
const currentDeviceId = ref(null)
const mcpTools = ref([])
const mcpCallResult = ref('')
const mcpCallForm = ref({ tool_name: '', argumentsText: '{}' })

// Device role configuration state
const showRoleConfigDialog = ref(false)
const roleConfigLoading = ref(false)
const currentDevice = ref({})
const selectedRoleId = ref(null)
const selectedRole = ref(null)
const availableRoles = ref([])
const isRoleActive = (role) => role?.status === 'active' || !role?.status
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
  if (selectedAgentName.value) return t('agent_no_device')
  return t('no_device_bound')
})

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

const handleDeviceBound = async () => {
  await loadDevices()
}

const resetAddDeviceForm = () => {
  deviceForm.value = createDefaultDeviceForm({ mode: 'bind', fixedAgentId: bindingAgentId.value || null })
  deviceFormRef.value?.clearValidate?.()
}

const openAddDeviceDialog = () => {
  if (!agents.value.length) {
    ElMessage.warning(t('create_agent_before_bind'))
    return
  }
  resetAddDeviceForm()
  showAddDeviceDialog.value = true
}

const handleAddDevice = async () => {
  if (!deviceFormRef.value) return
  try {
    await deviceFormRef.value.validate()
  } catch {
    return
  }
  const agentId = bindingAgentId.value || deviceForm.value.agent_id
  if (!agentId) {
    ElMessage.warning(t('select_target_agent'))
    return
  }

  addingDevice.value = true
  try {
    const response = await api.post(`/user/agents/${agentId}/devices`, deviceFormRef.value.buildPayload())
    if (response.data?.success) {
      ElMessage.success(t('device_bind_success'))
      showAddDeviceDialog.value = false
      await handleDeviceBound()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('device_bind_failed'))
  } finally {
    addingDevice.value = false
  }
}

const handleVoicePush = (device) => {
  if (!device?.device_name) {
    ElMessage.warning(t('device_no_id_notify'))
    return
  }
  voicePushDeviceId.value = device.device_name
  showVoicePushDialog.value = true
}

const handleVoicePushSuccess = () => {
  voicePushDeviceId.value = ''
}

const handleAgentFilterChange = (value) => {
  const query = value ? { agent_id: value } : {}
  router.replace({ path: '/user/devices', query })
}

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

const startDeviceNameEdit = (device) => {
  editingDeviceId.value = device.id
  editingDeviceName.value = String(device.nick_name || '').trim() || getDeviceDisplayName(device)
  nextTick(() => {
    deviceNameInputRef.value?.focus?.()
  })
}

const cancelDeviceNameEdit = () => {
  editingDeviceId.value = null
  editingDeviceName.value = ''
}

const saveDeviceName = async (device) => {
  const name = editingDeviceName.value.trim()
  if (!name) {
    ElMessage.warning(t('device_nickname_required'))
    return
  }
  if (name === String(device.nick_name || '').trim()) {
    cancelDeviceNameEdit()
    return
  }

  renamingDeviceId.value = device.id
  try {
    const response = await api.put(`/user/devices/${device.id}`, { nick_name: name })
    const updatedDevice = response.data?.data || {}
    const target = devices.value.find(item => item.id === device.id)
    if (target) {
      target.nick_name = updatedDevice.nick_name || name
    }
    ElMessage.success(t('device_nickname_updated'))
    cancelDeviceNameEdit()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('update_device_nickname_failed'))
  } finally {
    renamingDeviceId.value = null
  }
}

const handleDeviceMcp = async (device) => {
  currentDeviceId.value = device.id
  showMcpDialog.value = true
  mcpLoading.value = true
  mcpCallResult.value = ''
  mcpCallForm.value = { tool_name: '', argumentsText: '{}' }
  try {
    await refreshDeviceMcpTools()
  } finally {
    mcpLoading.value = false
  }
}

const refreshDeviceMcpTools = async () => {
  if (!currentDeviceId.value) return
  toolsLoading.value = true
  try {
    const response = await api.get(`/user/devices/${currentDeviceId.value}/mcp-tools`)
    mcpTools.value = response.data.data?.tools || []
    if (!mcpCallForm.value.tool_name && mcpTools.value.length > 0) {
      mcpCallForm.value.tool_name = mcpTools.value[0].name
    }
  } catch (error) {
    ElMessage.error(t('fetch_device_mcp_failed'))
    mcpTools.value = []
  } finally {
    toolsLoading.value = false
  }
}



const buildExampleFromSchema = (schema = {}) => {
  if (!schema || typeof schema !== 'object') return {}
  if (Array.isArray(schema.enum) && schema.enum.length > 0) return schema.enum[0]

  const type = schema.type || 'object'
  if (type === 'object') {
    const props = schema.properties || {}
    const result = {}
    Object.keys(props).sort().forEach((key) => {
      result[key] = buildExampleFromSchema(props[key])
    })
    return result
  }
  if (type === 'array') {
    return [buildExampleFromSchema(schema.items || {})]
  }
  if (type === 'number') return 0.1
  if (type === 'integer') return 0
  if (type === 'boolean') return false
  return ''
}

const updateMcpExampleByTool = (toolName) => {
  const selectedTool = mcpTools.value.find(item => item.name === toolName)
  if (!selectedTool) return

  const example = buildExampleFromSchema(selectedTool.input_schema || {})
  mcpCallForm.value.argumentsText = JSON.stringify(example ?? {}, null, 2)
}

const handleMcpToolChange = (toolName) => {
  updateMcpExampleByTool(toolName)
}

const formatMcpCallResult = (payload) => {
  const MAX_PARSE_DEPTH = 8

  const tryParseJSONString = (value) => {
    if (typeof value !== 'string') return { parsed: false, value }
    let text = value.trim()
    if (!text) return { parsed: false, value }

    const fenced = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i)
    if (fenced) {
      text = fenced[1].trim()
    }

    const looksLikeJSON =
      (text.startsWith('{') && text.endsWith('}')) ||
      (text.startsWith('[') && text.endsWith(']'))
    if (!looksLikeJSON) return { parsed: false, value }

    try {
      return { parsed: true, value: JSON.parse(text) }
    } catch (_) {
      return { parsed: false, value }
    }
  }

  const deepParseJSONStrings = (value, depth = 0) => {
    if (depth >= MAX_PARSE_DEPTH || value == null) return value

    if (typeof value === 'string') {
      const parsed = tryParseJSONString(value)
      if (!parsed.parsed) return value
      return deepParseJSONStrings(parsed.value, depth + 1)
    }

    if (Array.isArray(value)) {
      return value.map((item) => deepParseJSONStrings(item, depth + 1))
    }

    if (typeof value === 'object') {
      const out = {}
      Object.keys(value).forEach((key) => {
        out[key] = deepParseJSONStrings(value[key], depth + 1)
      })

      if (Array.isArray(out.content) && out.content.length === 1) {
        const first = out.content[0]
        if (first && typeof first === 'object' && !Array.isArray(first) && first.type === 'text' && Object.prototype.hasOwnProperty.call(first, 'text')) {
          const textValue = first.text
          if (textValue && typeof textValue === 'object') {
            return textValue
          }
        }
      }

      return out
    }

    return value
  }

  const data = payload ?? {}
  const raw = (data && typeof data === 'object' && !Array.isArray(data) && Object.prototype.hasOwnProperty.call(data, 'result'))
    ? data.result
    : data

  return JSON.stringify(deepParseJSONStrings(raw), null, 2)
}

const callDeviceMcpTool = async () => {
  if (!currentDeviceId.value || !mcpCallForm.value.tool_name) {
    ElMessage.warning(t('select_tool'))
    return
  }

  let argumentsObj = {}
  try {
    argumentsObj = mcpCallForm.value.argumentsText ? JSON.parse(mcpCallForm.value.argumentsText) : {}
  } catch (e) {
    ElMessage.error(t('params_json_format_error'))
    return
  }

  callingTool.value = true
  try {
    const response = await api.post(`/user/devices/${currentDeviceId.value}/mcp-call`, {
      tool_name: mcpCallForm.value.tool_name,
      arguments: argumentsObj
    })
    mcpCallResult.value = formatMcpCallResult(response.data.data || {})
    ElMessage.success(t('mcp_tool_call_success'))
  } catch (error) {
    mcpCallResult.value = JSON.stringify(error.response?.data || { error: error.message }, null, 2)
    ElMessage.error(t('mcp_tool_call_failed'))
  } finally {
    callingTool.value = false
  }
}

// Load role list
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

// Open device role configuration dialog
const handleDeviceRole = async (deviceId) => {
  const device = devices.value.find(d => d.id === deviceId)
  if (!device) return

  currentDevice.value = { ...device }
  selectedRoleId.value = device.role_id || null
  selectedRole.value = null

  // Load roles if not yet loaded
  if (availableRoles.value.length === 0) {
    await loadRoles()
  }

  // Find the currently associated role
  if (device.role_id) {
    const role = availableRoles.value.find(r => r.id === device.role_id)
    if (role) {
      selectedRole.value = role
    }
  }

  showRoleConfigDialog.value = true
}

// Handle role selection change
const handleRoleSelect = (roleId) => {
  if (!roleId) {
    selectedRole.value = null
    return
  }
  const role = availableRoles.value.find(r => r.id === roleId)
  if (role) {
    selectedRole.value = role
  }
}

// Apply selected role to device
const handleApplyRole = async () => {
  if (!currentDevice.value.id) return

  roleConfigLoading.value = true
  try {
    const data = {
      role_id: selectedRoleId.value || null
    }

    await api.post(`/devices/${currentDevice.value.id}/apply-role`, data)
    ElMessage.success(selectedRoleId.value ? t('role_applied_to_device') : t('device_role_cancelled'))
    showRoleConfigDialog.value = false
    await loadDevices()
  } catch (error) {
    ElMessage.error(t('operation_failed_colon') + ' ' + (error.response?.data?.error || error.message))
  } finally {
    roleConfigLoading.value = false
  }
}

// Close role configuration dialog
const handleCloseRoleConfig = () => {
  showRoleConfigDialog.value = false
  currentDevice.value = {}
  selectedRoleId.value = null
  selectedRole.value = null
}

const handleDeleteDevice = async (device) => {
  if (!device?.id) {
    return
  }

	  try {
	    await ElMessageBox.confirm(
	      t('confirm_delete_device_msg', { name: getDeviceDisplayName(device) }),
	      t('confirm_delete_device'),
      {
        confirmButtonText: t('delete'),
        cancelButtonText: t('cancel'),
        type: 'warning',
      }
    )

    const response = await api.delete(`/user/devices/${device.id}`)
    if (response.data.success) {
      ElMessage.success(response.data.message || t('device_deleted'))
      await loadDevices()
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || t('delete_device_failed'))
    }
  }
}

const goBack = () => {
  router.push('/agents')
}

const formatDate = (dateString) => {
  if (!dateString) return t('never')
  return new Date(dateString).toLocaleString('zh-CN')
}

// Check if device is online based on last active timestamp
const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  const now = new Date()
  const lastActive = new Date(lastActiveAt)
  // Online if active within the last 5 minutes
  return (now - lastActive) < 5 * 60 * 1000
}

onMounted(() => {
  loadAgents()
  loadDevices()
  loadRoles()
})

watch(
  () => [route.params.id, route.query.agent_id],
  () => {
    filterAgentId.value = String(route.params.id || route.query.agent_id || '')
  }
)
</script>

<style scoped>
.agent-devices-page {
  padding: 0;
}

.back-btn {
  padding: 8px;
  color: var(--apple-primary);
}

.empty-section {
  margin-top: 40px;
}

.empty-card {
  text-align: center;
  padding: 40px 20px;
}

.empty-content h3 {
  margin: 20px 0 10px 0;
  color: var(--apple-text);
}

.empty-content p {
  color: var(--apple-text-secondary);
  margin-bottom: 30px;
}

.devices-filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
  padding: 14px 16px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  box-shadow: var(--apple-shadow-sm);
}

.filter-controls {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.agent-filter-select {
  width: 240px;
}

.devices-count {
  color: var(--apple-text-secondary);
  font-size: 13px;
  font-weight: 600;
}

.add-device-button {
  flex: none;
  margin-left: auto;
}

.devices-grid {
  margin-top: 20px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 340px));
  gap: 16px;
  justify-content: flex-start;
}

.device-item {
  min-width: 0;
}

.device-card {
  background: rgba(255, 255, 255, 0.88);
  border-radius: 20px;
  padding: 16px;
  border: 1px solid rgba(229, 229, 234, 0.72);
  box-shadow: var(--apple-shadow-md);
  transition: all 0.3s ease;
  height: 100%;
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  max-width: 340px;
}

.device-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--apple-shadow-sm);
  border-color: rgba(0, 122, 255, 0.18);
}

.device-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

.device-icon {
  width: 42px;
  height: 42px;
  background: linear-gradient(180deg, #2e90ff 0%, #007aff 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.device-info {
  flex: 1;
  min-width: 0;
}

.device-name-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
  min-width: 0;
}

.device-name {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--apple-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.device-name-button {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 100%;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: text;
}

.device-name-button:hover .device-name,
.device-name-button:focus-visible .device-name {
  color: var(--apple-primary);
}

.device-name-button:focus-visible {
  outline: 2px solid rgba(0, 122, 255, 0.22);
  outline-offset: 3px;
  border-radius: 8px;
}

.device-name-input {
  min-width: 0;
}

.device-name-actions {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  gap: 2px;
}

.device-name-actions :deep(.el-button) {
  min-height: auto;
  width: 26px;
  height: 26px;
  padding: 0;
  margin: 0;
  font-size: 12px;
  border-radius: 9px;
}

.rename-icon-button {
  opacity: 0.28;
  color: var(--apple-text-tertiary);
  border-color: rgba(229, 229, 234, 0.78);
  background: rgba(255, 255, 255, 0.7);
  transition: opacity 0.2s ease, color 0.2s ease, border-color 0.2s ease, transform 0.2s ease;
}

.device-name-row:hover .rename-icon-button,
.device-card:hover .rename-icon-button,
.rename-icon-button:focus-visible {
  opacity: 1;
  color: var(--apple-primary);
  border-color: rgba(0, 122, 255, 0.28);
  transform: translateY(-1px);
}

.name-action-button {
  box-shadow: none;
}

.device-identity {
  margin: 0;
  font-size: 11px;
  color: rgba(107, 114, 128, 0.74);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.device-status {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--apple-line-strong);
}

.status-dot.online {
  background: var(--apple-success);
}

.status-dot.offline {
  background: var(--apple-danger);
}

.status-text {
  font-size: 12px;
  color: var(--apple-text-secondary);
}

.device-meta {
  flex: 1;
  margin-bottom: 16px;
}

.meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.meta-row:last-child {
  margin-bottom: 0;
}

.meta-label {
  font-size: 12px;
  color: var(--apple-text-secondary);
}

.meta-value {
  font-size: 12px;
  color: var(--apple-text);
  font-weight: 500;
}

.mcp-tools-header { margin-bottom: 12px; }
.tools-tags { display:flex; flex-wrap:wrap; gap:8px; margin-bottom:12px; }
.tools-empty { color: var(--apple-text-secondary); margin: 8px 0 16px; }
.mcp-result-box {
  white-space: pre-wrap;
  font-family: monospace;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px;
  min-height: 80px;
}

.device-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: auto;
}

.device-actions .el-button {
  min-width: 0;
  width: 100%;
  min-height: 34px;
  margin-left: 0;
  padding: 0 10px;
  justify-content: center;
  border-radius: 12px;
  border: 1px solid rgba(214, 219, 228, 0.9);
  background: rgba(248, 250, 252, 0.92);
  color: #4b5563;
  box-shadow: none;
  font-size: 12px;
  font-weight: 600;
}

.device-actions .el-button + .el-button {
  margin-left: 0;
}

.device-actions :deep(.el-button > span) {
  min-width: 0;
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.device-actions :deep(.el-icon) {
  flex: none;
  font-size: 13px;
}

.device-action-button-feature {
  border-color: rgba(147, 197, 253, 0.85);
  background: linear-gradient(180deg, rgba(239, 246, 255, 0.98) 0%, rgba(219, 234, 254, 0.9) 100%);
  color: #1d4ed8;
}

.device-action-button-voice {
  border-color: rgba(153, 246, 228, 0.95);
  background: rgba(240, 253, 250, 0.96);
  color: #0f766e;
}

.device-action-button-danger {
  border-color: rgba(244, 191, 191, 0.95);
  background: rgba(255, 245, 245, 0.96);
  color: #b42318;
}

.device-actions .el-button:hover,
.device-actions .el-button:focus {
  border-color: rgba(148, 163, 184, 0.82);
  background: rgba(241, 245, 249, 0.98);
  color: #334155;
}

.device-action-button-feature:hover,
.device-action-button-feature:focus {
  border-color: rgba(96, 165, 250, 0.95);
  background: linear-gradient(180deg, rgba(219, 234, 254, 0.98) 0%, rgba(191, 219, 254, 0.92) 100%);
  color: #1e40af;
}

.device-action-button-voice:hover,
.device-action-button-voice:focus {
  color: #075985;
  background: rgba(236, 254, 255, 0.98);
  border-color: rgba(34, 211, 238, 0.58);
}

.device-action-button-danger:hover,
.device-action-button-danger:focus {
  border-color: rgba(248, 113, 113, 0.78);
  background: rgba(254, 242, 242, 0.98);
  color: #991b1b;
}

.dialog-footer {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.dialog-footer .el-button {
  min-width: 80px;
}

/* Device role configuration styles */
.role-config-content {
  padding: 20px 0;
}

.current-role-info {
  margin-bottom: 16px;
}

.current-role-info p {
  margin: 4px 0;
  color: var(--apple-text-secondary);
}

.role-option-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-direction: column;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 8px;
}

.role-option-main {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.role-preview-card {
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
}

.role-preview-content {
  font-size: 14px;
}

.role-preview-content p {
  margin: 8px 0;
}

.role-preview-content strong {
  color: var(--apple-text);
  margin-right: 8px;
}

.role-configs-preview {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.prompt-preview {
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  padding: 12px;
  border-radius: 14px;
  font-size: 13px;
  color: var(--apple-text-secondary);
  line-height: 1.6;
}

@media (max-width: 768px) {
  .devices-filter-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .filter-controls {
    align-items: stretch;
    flex-direction: column;
  }

  .agent-filter-select {
    width: 100%;
  }

  .devices-count {
    align-self: flex-start;
  }

  .add-device-button {
    width: 100%;
    margin-left: 0;
  }

  .devices-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .device-actions {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .rename-icon-button {
    opacity: 1;
  }
}

@media (max-width: 560px) {
  .devices-grid {
    gap: 10px;
  }

  .device-actions {
    grid-template-columns: 1fr;
  }
}

@media (min-width: 769px) and (max-width: 1180px) {
  .devices-grid {
    grid-template-columns: repeat(auto-fill, minmax(280px, 340px));
  }
}
</style>
