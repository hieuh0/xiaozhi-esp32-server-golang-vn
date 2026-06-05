<template>
  <div class="admin-devices">
    <div class="toolbar">
      <el-button type="primary" @click="openAddDialog">
        <el-icon><Plus /></el-icon>
        {{ t('add_agent') }}
      </el-button>
      <el-button @click="loadDevices">
        <el-icon><Refresh /></el-icon>
        {{ t('refresh') }}
      </el-button>
    </div>

    <el-table :data="devices" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column :label="t('device_nickname')" min-width="170">
        <template #default="{ row }">
          <span class="device-nick-name">{{ getDeviceDisplayName(row) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="device_code" :label="t('activation_code')" width="150" />
      <el-table-column :label="t('device_id')" width="190">
        <template #default="{ row }">
          <span class="device-id-text">{{ row.device_name || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="user_id" :label="t('user_prefix') + 'ID'" width="100" />
      <el-table-column :label="t('link_agent')" width="150">
        <template #default="{ row }">
          <span v-if="row.agent_id > 0">
            {{ row.agent_name || t('agent_id_fallback', { id: row.agent_id }) }}
          </span>
          <el-tag v-else type="info" size="small">{{ t('no_agent_linked') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('activation_status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.activated ? 'success' : 'warning'">
            {{ row.activated ? t('active') : t('no_agent_linked') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('online_devices')" width="100">
        <template #default="{ row }">
          <el-tag :type="isDeviceOnline(row.last_active_at) ? 'success' : 'danger'">
            {{ isDeviceOnline(row.last_active_at) ? t('online') : t('close') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="last_active_at" :label="t('latest_data')" width="180">
        <template #default="{ row }">
          {{ row.last_active_at ? new Date(row.last_active_at).toLocaleString() : t('never_active') }}
        </template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('start_date')" width="180">
        <template #default="{ row }">
          {{ new Date(row.created_at).toLocaleString() }}
        </template>
      </el-table-column>
      <el-table-column :label="t('operation_failed').replace(t('failed'), '')" width="300">
        <template #default="{ row }">
          <el-button size="small" @click="editDevice(row)">
            {{ t('cancel_edit').replace(t('cancel'), '') }}
          </el-button>
          <el-button size="small" type="primary" @click="showDeviceMcp(row)">
            MCP
          </el-button>
          <el-button size="small" type="danger" @click="deleteDevice(row)">
            {{ t('delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑设备对话框 -->

    <el-dialog v-model="showMcpDialog" :title="t('device_mcp_tools')" width="760px">
      <div v-loading="mcpLoading">
        <div class="mcp-tools-header">
          <el-button size="small" type="primary" @click="refreshDeviceMcpTools" :loading="toolsLoading">{{ t('refresh') }}</el-button>
        </div>

        <div v-if="mcpTools.length === 0" class="tools-empty">{{ t('no_mcp_endpoint_to_copy') }}</div>
        <div v-else class="tools-tags">
          <el-tag v-for="tool in mcpTools" :key="tool.name" class="tool-tag">{{ tool.name }}</el-tag>
        </div>

        <el-divider />

        <el-form :model="mcpCallForm" label-width="90px">
          <el-form-item :label="t('tool_name')">
            <el-select v-model="mcpCallForm.tool_name" :placeholder="t('select_tool')" style="width:100%" @change="handleMcpToolChange">
              <el-option v-for="tool in mcpTools" :key="tool.name" :label="tool.name" :value="tool.name" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('params_json_format_error').replace(t('format_error'), '')">
            <el-input v-model="mcpCallForm.argumentsText" type="textarea" :rows="6" :placeholder="t('mcp_args_placeholder')" />
          </el-form-item>
        </el-form>

        <el-button type="primary" @click="callDeviceMcpTool" :loading="callingTool">{{ t('call_device_tool') }}</el-button>

        <el-divider />
        <div class="endpoint-content">{{ mcpCallResult || t('no_mcp_endpoint_to_copy') }}</div>
      </div>
    </el-dialog>

    <el-dialog
      v-model="showAddDialog"
      :title="editingDevice ? t('device_update_success').replace(t('update_success'), t('edit')) : t('add_agent')"
      width="500px"
    >
      <DeviceForm
        ref="deviceFormRef"
        v-model="deviceForm"
        is-admin
        :mode="editingDevice ? 'edit' : 'create'"
      />
      <template #footer>
        <el-button @click="showAddDialog = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" @click="saveDevice" :loading="saving">
          {{ editingDevice ? t('update_success').replace(t('success'), '') : t('add_agent').replace(t('agent'), '') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import api from '../../utils/api'
import DeviceForm from '../../components/common/DeviceForm.vue'
import { createDefaultDeviceForm, deviceToForm } from '../../composables/useAgentFormOptions'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

const devices = ref([])
const loading = ref(false)
const showAddDialog = ref(false)
const editingDevice = ref(null)
const saving = ref(false)
const deviceFormRef = ref()

const showMcpDialog = ref(false)
const mcpLoading = ref(false)
const toolsLoading = ref(false)
const callingTool = ref(false)
const currentDeviceId = ref(null)
const mcpTools = ref([])
const mcpCallResult = ref('')
const mcpCallForm = ref({ tool_name: '', argumentsText: '{}' })
const deviceForm = ref(createDefaultDeviceForm({ isAdmin: true }))

const loadDevices = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/devices')
    devices.value = response.data.data || []
  } catch (error) {
    ElMessage.error(t('load_device_list_failed'))
    console.error('Error loading devices:', error)
  } finally {
    loading.value = false
  }
}

const getDeviceDisplayName = (device) => {
  const nickName = String(device?.nick_name || '').trim()
  if (nickName) return nickName
  return String(device?.device_name || '').trim() || t('unnamed_device')
}

const openAddDialog = () => {
  editingDevice.value = null
  deviceForm.value = createDefaultDeviceForm({ isAdmin: true })
  showAddDialog.value = true
}

const editDevice = (device) => {
  editingDevice.value = device
  deviceForm.value = deviceToForm(device, { isAdmin: true })
  showAddDialog.value = true
}

const saveDevice = async () => {
  if (!deviceFormRef.value) return

  const valid = await deviceFormRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const payload = deviceFormRef.value.buildPayload()
    if (editingDevice.value) {
      await api.put(`/admin/devices/${editingDevice.value.id}`, payload)
      ElMessage.success(t('device_update_success'))
    } else {
      const response = await api.post('/admin/devices', payload)
      // 根据后端返回的消息显示不同的提示
      const message = response.data.message || t('device_bind_success')
      ElMessage.success(message)
    }
    showAddDialog.value = false
    resetForm()
    loadDevices()
  } catch (error) {
    const errorMessage = error.response?.data?.error || (editingDevice.value ? t('update_failed') : t('delete_failed'))
    ElMessage.error(errorMessage)
    console.error('Error saving device:', error)
  } finally {
    saving.value = false
  }
}

const deleteDevice = async (device) => {
  try {
    await ElMessageBox.confirm(
      t('confirm_delete_device_msg', { name: getDeviceDisplayName(device) }),
      t('confirm'),
      {
        confirmButtonText: t('confirm'),
        cancelButtonText: t('cancel'),
        type: 'warning'
      }
    )

    await api.delete(`/admin/devices/${device.id}`)
    ElMessage.success(t('device_delete_success'))
    loadDevices()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('device_delete_failed'))
      console.error('Error deleting device:', error)
    }
  }
}



const showDeviceMcp = async (device) => {
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
    const response = await api.get(`/admin/devices/${currentDeviceId.value}/mcp-tools`)
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
    const response = await api.post(`/admin/devices/${currentDeviceId.value}/mcp-call`, {
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

const resetForm = () => {
  editingDevice.value = null
  deviceForm.value = createDefaultDeviceForm({ isAdmin: true })
  if (deviceFormRef.value) {
    deviceFormRef.value.resetFields()
  }
}

// 判断设备是否在线（基于最后活跃时间）
const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  const now = new Date()
  const lastActive = new Date(lastActiveAt)
  // 5分钟内有活动认为在线
  return (now - lastActive) < 5 * 60 * 1000
}

onMounted(() => {
  loadDevices()
})
</script>

<style scoped>
.admin-devices {
  padding: 20px;
}

.toolbar {
  margin-bottom: 20px;
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.device-nick-name {
  font-weight: 700;
  color: var(--apple-text, #1d1d1f);
}

.device-id-text {
  color: rgba(107, 114, 128, 0.72);
  font-family: monospace;
  font-size: 12px;
}

.form-tip {
  margin-top: 6px;
  color: rgba(107, 114, 128, 0.78);
  font-size: 12px;
  line-height: 1.5;
}

.tools-tags { display:flex; flex-wrap:wrap; gap:8px; margin-bottom:12px; }
.tools-empty { color:#909399; margin: 8px 0 16px; }
.endpoint-content {
  white-space: pre-wrap;
  font-family: monospace;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 10px;
  min-height: 80px;
}
.mcp-tools-header { margin-bottom: 12px; }
</style>
