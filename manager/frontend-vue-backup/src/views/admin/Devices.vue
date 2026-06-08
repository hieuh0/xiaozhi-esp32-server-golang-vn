<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal, RefreshCw } from '@lucide/vue'
import api from '../../utils/api'
import DeviceForm from '../../components/common/DeviceForm.vue'
import { createDefaultDeviceForm, deviceToForm } from '../../composables/useAgentFormOptions'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const { t } = useLocale()
const { formatDate } = useFormatDate()

const devices = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const showAddDialog = ref(false)
const editingDevice = ref(null)
const saving = ref(false)
const deviceFormRef = ref()
const deviceForm = ref(createDefaultDeviceForm({ isAdmin: true }))

const showMcpDialog = ref(false)
const mcpLoading = ref(false)
const toolsLoading = ref(false)
const callingTool = ref(false)
const currentDeviceId = ref(null)
const mcpTools = ref([])
const mcpCallResult = ref('')
const mcpCallForm = ref({ tool_name: '', argumentsText: '{}' })

const loadDevices = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/devices', { params: { page: page.value, page_size: pageSize.value } })
    devices.value = response.data.data || []
    total.value = response.data.total || 0
  } catch {
    ElMessage.error(t('load_device_list_failed'))
  } finally {
    loading.value = false
  }
}

const handleDeviceAction = (command, row) => {
  switch (command) {
    case 'edit': editDevice(row); break
    case 'mcp': showDeviceMcp(row); break
    case 'delete': deleteDevice(row); break
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
      ElMessage.success(response.data.message || t('device_bind_success'))
    }
    showAddDialog.value = false
    resetForm()
    loadDevices()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || (editingDevice.value ? t('update_failed') : t('delete_failed')))
  } finally {
    saving.value = false
  }
}

const deleteDevice = async (device) => {
  try {
    await ElMessageBox.confirm(
      t('confirm_delete_device_msg', { name: getDeviceDisplayName(device) }),
      t('confirm'),
      { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' }
    )
    await api.delete(`/admin/devices/${device.id}`)
    ElMessage.success(t('device_delete_success'))
    loadDevices()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('device_delete_failed'))
  }
}

const resetForm = () => {
  editingDevice.value = null
  deviceForm.value = createDefaultDeviceForm({ isAdmin: true })
}

const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  return (new Date() - new Date(lastActiveAt)) < 5 * 60 * 1000
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
  } catch {
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
    Object.keys(props).sort().forEach((key) => { result[key] = buildExampleFromSchema(props[key]) })
    return result
  }
  if (type === 'array') return [buildExampleFromSchema(schema.items || {})]
  if (type === 'number') return 0.1
  if (type === 'integer') return 0
  if (type === 'boolean') return false
  return ''
}

const handleMcpToolChange = (toolName) => {
  const selectedTool = mcpTools.value.find(item => item.name === toolName)
  if (!selectedTool) return
  const example = buildExampleFromSchema(selectedTool.input_schema || {})
  mcpCallForm.value.argumentsText = JSON.stringify(example ?? {}, null, 2)
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
    try { return { parsed: true, value: JSON.parse(text) } } catch { return { parsed: false, value } }
  }
  const deepParseJSONStrings = (value, depth = 0) => {
    if (depth >= MAX_PARSE_DEPTH || value == null) return value
    if (typeof value === 'string') {
      const parsed = tryParseJSONString(value)
      if (!parsed.parsed) return value
      return deepParseJSONStrings(parsed.value, depth + 1)
    }
    if (Array.isArray(value)) return value.map((item) => deepParseJSONStrings(item, depth + 1))
    if (typeof value === 'object') {
      const out = {}
      Object.keys(value).forEach((key) => { out[key] = deepParseJSONStrings(value[key], depth + 1) })
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
  const raw = (data && typeof data === 'object' && !Array.isArray(data) && Object.prototype.hasOwnProperty.call(data, 'result'))
    ? data.result : data
  return JSON.stringify(deepParseJSONStrings(raw), null, 2)
}

const callDeviceMcpTool = async () => {
  if (!currentDeviceId.value || !mcpCallForm.value.tool_name) {
    ElMessage.warning(t('select_tool')); return
  }
  let argumentsObj = {}
  try {
    argumentsObj = mcpCallForm.value.argumentsText ? JSON.parse(mcpCallForm.value.argumentsText) : {}
  } catch {
    ElMessage.error(t('params_json_format_error')); return
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

const totalPages = () => Math.ceil(total.value / pageSize.value) || 1

onMounted(() => { loadDevices() })
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex items-center justify-end gap-2">
      <Button variant="outline" @click="loadDevices">
        <RefreshCw class="w-4 h-4 mr-1.5" />{{ t('refresh') }}
      </Button>
      <Button @click="openAddDialog">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('add_agent') }}
      </Button>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
      <div class="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead class="w-16">ID</TableHead>
              <TableHead>{{ t('device_nickname') }}</TableHead>
              <TableHead class="w-32">{{ t('activation_code') }}</TableHead>
              <TableHead class="w-44">{{ t('device_id') }}</TableHead>
              <TableHead class="w-24">{{ t('user_prefix') }}ID</TableHead>
              <TableHead class="w-36">{{ t('link_agent') }}</TableHead>
              <TableHead class="w-28">{{ t('activation_status') }}</TableHead>
              <TableHead class="w-24">{{ t('online_devices') }}</TableHead>
              <TableHead class="w-44">{{ t('latest_data') }}</TableHead>
              <TableHead class="w-44">{{ t('start_date') }}</TableHead>
              <TableHead class="w-16 text-center">{{ t('actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="11" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableEmpty v-if="!devices.length" />
              <TableRow v-for="row in devices" :key="row.id">
                <TableCell class="text-xs font-mono text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
                <TableCell class="font-bold">{{ getDeviceDisplayName(row) }}</TableCell>
                <TableCell>{{ row.device_code }}</TableCell>
                <TableCell class="font-mono text-xs text-[var(--color-text-secondary)]">{{ row.device_name || '-' }}</TableCell>
                <TableCell>{{ row.user_id }}</TableCell>
                <TableCell>
                  <span v-if="row.agent_id > 0" class="text-sm">{{ row.agent_name || t('agent_id_fallback', { id: row.agent_id }) }}</span>
                  <span v-else class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-500 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700">
                    {{ t('no_agent_linked') }}
                  </span>
                </TableCell>
                <TableCell>
                  <span :class="row.activated
                    ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
                    : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800'">
                    {{ row.activated ? t('active') : t('no_agent_linked') }}
                  </span>
                </TableCell>
                <TableCell>
                  <span :class="isDeviceOnline(row.last_active_at)
                    ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
                    : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'">
                    {{ isDeviceOnline(row.last_active_at) ? t('online') : t('close') }}
                  </span>
                </TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.last_active_at ? formatDate(row.last_active_at) : t('never_active') }}</TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ formatDate(row.created_at) }}</TableCell>
                <TableCell class="text-center">
                  <DropdownMenu>
                    <DropdownMenuTrigger as-child>
                      <Button variant="ghost" size="icon" class="h-8 w-8" :aria-label="t('more_actions')">
                        <MoreHorizontal class="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem @click="handleDeviceAction('edit', row)">{{ t('edit') }}</DropdownMenuItem>
                      <DropdownMenuItem @click="handleDeviceAction('mcp', row)">MCP</DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem class="text-destructive" @click="handleDeviceAction('delete', row)">{{ t('delete') }}</DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
      <span>{{ total }} {{ t('total_items') }}</span>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadDevices()">{{ t('prev') }}</Button>
        <span>{{ page }} / {{ totalPages() }}</span>
        <Button variant="outline" size="sm" :disabled="page >= totalPages()" @click="page++; loadDevices()">{{ t('next') }}</Button>
      </div>
    </div>

    <!-- Add/Edit device dialog -->
    <Dialog v-model:open="showAddDialog" @update:open="v => { if (!v) resetForm() }">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editingDevice ? t('edit') : t('add_agent') }}</DialogTitle>
        </DialogHeader>
        <DeviceForm ref="deviceFormRef" v-model="deviceForm" is-admin :mode="editingDevice ? 'edit' : 'create'" />
        <DialogFooter>
          <Button variant="outline" @click="showAddDialog = false">{{ t('cancel') }}</Button>
          <Button :disabled="saving" @click="saveDevice">{{ editingDevice ? t('save') : t('add') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- MCP Tools dialog -->
    <Dialog v-model:open="showMcpDialog">
      <DialogContent class="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{{ t('device_mcp_tools') }}</DialogTitle>
        </DialogHeader>

        <div v-if="mcpLoading" class="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
        <div v-else class="grid gap-4">
          <!-- Header -->
          <div class="flex items-center justify-between">
            <div class="flex flex-wrap gap-1.5">
              <span v-if="!mcpTools.length" class="text-sm text-[var(--color-text-secondary)]">{{ t('no_mcp_endpoint_to_copy') }}</span>
              <span v-else v-for="tool in mcpTools" :key="tool.name"
                class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)] border-[var(--color-line)]">
                {{ tool.name }}
              </span>
            </div>
            <Button size="sm" variant="outline" :disabled="toolsLoading" @click="refreshDeviceMcpTools">{{ t('refresh') }}</Button>
          </div>

          <div class="border-t border-[var(--color-line)]" />

          <!-- Tool call form -->
          <div class="grid gap-3">
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('tool_name') }}</label>
              <Select :model-value="mcpCallForm.tool_name" @update:model-value="v => { mcpCallForm.tool_name = v; handleMcpToolChange(v) }">
                <SelectTrigger><SelectValue :placeholder="t('select_tool')" /></SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="tool in mcpTools" :key="tool.name" :value="tool.name">{{ tool.name }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('params_json_format_error').replace(t('format_error'), '') }}</label>
              <textarea v-model="mcpCallForm.argumentsText" rows="6" :placeholder="t('mcp_args_placeholder')"
                class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm font-mono shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none placeholder:text-muted-foreground w-full resize-y" />
            </div>
            <div>
              <Button :disabled="callingTool" @click="callDeviceMcpTool">{{ t('call_device_tool') }}</Button>
            </div>
          </div>

          <!-- Result -->
          <div class="border-t border-[var(--color-line)]" />
          <pre class="whitespace-pre-wrap font-mono text-xs bg-[var(--color-surface-muted)] border border-[var(--color-line)] rounded-lg p-3 min-h-[80px] max-h-[200px] overflow-auto">{{ mcpCallResult || t('no_mcp_endpoint_to_copy') }}</pre>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
