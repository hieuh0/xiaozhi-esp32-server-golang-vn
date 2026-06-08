<script setup>
import { ref, reactive, onMounted, computed, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { resolveMemoryProvider } from './forms/configProviderUtils'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'
import { Switch } from '@/components/ui/switch'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

const { t } = useLocale()
const { formatDate } = useFormatDate()

const configs = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)

const safeConfigs = computed(() => Array.isArray(configs.value) ? configs.value : [])

const defaultUrls = {
  memobase: 'https://api.memobase.dev',
  mem0: 'https://api.mem0.ai',
  memos: 'https://memos.memtensor.cn/api/openmem/v1'
}

const form = reactive({
  name: '', config_id: '', provider: 'memobase', is_default: false, enabled: true,
  api_key: '', base_url: '', enable_search: true, search_threshold: 0.5, search_top_k: 3, timeout_ms: 10000
})

watch(showDialog, open => { if (!open) handleDialogClose() })

const getProviderBadgeClass = (provider) => {
  if (provider === 'memobase') return 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800'
  if (provider === 'memos') return 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800'
  return 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800'
}

const handleProviderChange = (value) => {
  form.api_key = ''
  form.base_url = defaultUrls[value] || ''
  form.enable_search = true
  form.search_threshold = 0.5
  form.search_top_k = 3
  form.timeout_ms = 10000
}

const generateConfig = () => {
  const config = {
    api_key: form.api_key, base_url: form.base_url,
    enable_search: form.enable_search, search_threshold: form.search_threshold, search_top_k: form.search_top_k
  }
  if (form.provider === 'memos') config.timeout_ms = form.timeout_ms
  return JSON.stringify(config)
}

const parseConfig = (jsonData) => {
  try {
    const config = JSON.parse(jsonData)
    form.api_key = config.api_key || ''
    form.base_url = config.base_url || defaultUrls[form.provider] || ''
    form.enable_search = config.enable_search !== undefined ? config.enable_search : true
    form.search_threshold = config.search_threshold !== undefined ? config.search_threshold : 0.5
    form.search_top_k = config.search_top_k !== undefined ? config.search_top_k : 3
    form.timeout_ms = config.timeout_ms !== undefined ? config.timeout_ms : 10000
  } catch {}
}

const validate = () => {
  if (!form.name.trim()) { ElMessage.error(t('enter_config_name')); return false }
  if (!form.config_id.trim()) { ElMessage.error(t('enter_config_id')); return false }
  if (!form.provider) { ElMessage.error(t('select_provider')); return false }
  if (!form.api_key.trim()) { ElMessage.error(t('enter_api_password')); return false }
  if (!form.base_url.trim()) { ElMessage.error(t('enter_base_url')); return false }
  return true
}

function parseJsonData(jsonData) {
  if (!jsonData || typeof jsonData !== 'string') return {}
  try { return JSON.parse(jsonData) || {} } catch { return {} }
}

function normalizeMemoryConfigRow(row) {
  const data = parseJsonData(row?.json_data)
  return { ...row, provider: resolveMemoryProvider(row?.provider, row?.config_id, data) }
}

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/memory-configs', { params: { page: page.value, page_size: pageSize.value } })
    await nextTick()
    if (response?.data?.data && Array.isArray(response.data.data)) {
      configs.value = response.data.data.map(normalizeMemoryConfigRow)
    } else if (response?.data?.data) {
      configs.value = [normalizeMemoryConfigRow(response.data.data)]
    } else {
      configs.value = []
    }
    total.value = response.data.total || 0
  } catch (error) {
    ElMessage.error(t('load_config_failed_prefix') + (error.message || t('unknown_error')))
    configs.value = []
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!validate()) return
  saving.value = true
  try {
    const payload = {
      name: form.name, config_id: form.config_id, provider: form.provider,
      enabled: form.enabled, is_default: form.is_default, json_data: generateConfig()
    }
    if (editingConfig.value) {
      await api.put(`/admin/memory-configs/${editingConfig.value.id}`, payload)
      ElMessage.success(t('config_update_success'))
    } else {
      await api.post('/admin/memory-configs', payload)
      ElMessage.success(t('config_create_success'))
    }
    showDialog.value = false
    await loadConfigs()
  } catch (error) {
    ElMessage.error(t('save_failed_colon') + error.message)
  } finally {
    saving.value = false
  }
}

const editConfig = (config) => {
  config = normalizeMemoryConfigRow(config)
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.enabled = config.enabled
  form.is_default = config.is_default
  if (config.json_data) parseConfig(config.json_data)
  showDialog.value = true
}

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_config'), t('confirm_delete'), { type: 'warning' })
    await api.delete(`/admin/memory-configs/${id}`)
    ElMessage.success(t('delete_success'))
    await loadConfigs()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('delete_failed_prefix') + error.message)
  }
}

const toggleEnable = async (config) => {
  try {
    await api.put(`/admin/memory-configs/${config.id}`, { ...config, enabled: config.enabled })
    ElMessage.success(config.enabled ? t('enabled') : t('disabled_done'))
  } catch (error) {
    config.enabled = !config.enabled
    ElMessage.error(t('op_failed_prefix') + error.message)
  }
}

const toggleDefault = async (config) => {
  try {
    if (config.is_default) {
      await api.post(`/admin/memory-configs/${config.id}/set-default`)
      ElMessage.success(t('set_as_default_config'))
      await loadConfigs()
    } else {
      await api.put(`/admin/memory-configs/${config.id}`, {
        name: config.name, config_id: config.config_id, provider: config.provider,
        enabled: config.enabled, is_default: false, json_data: config.json_data || ''
      })
      ElMessage.success(t('long_memory_disabled'))
      await loadConfigs()
    }
  } catch (error) {
    config.is_default = !config.is_default
    ElMessage.error(t('op_failed_prefix') + error.message)
  }
}

const handleAddConfig = () => {
  Object.assign(form, {
    name: '', config_id: '', provider: 'memobase', is_default: false, enabled: true,
    api_key: '', base_url: defaultUrls['memobase'], enable_search: true,
    search_threshold: 0.5, search_top_k: 3, timeout_ms: 10000
  })
  editingConfig.value = null
  showDialog.value = true
}

const handleDialogClose = () => {
  editingConfig.value = null
  Object.assign(form, {
    name: '', config_id: '', provider: 'memobase', is_default: false, enabled: true,
    api_key: '', base_url: '', enable_search: true, search_threshold: 0.5, search_top_k: 3, timeout_ms: 10000
  })
}

const handleAction = (command, row) => {
  if (command === 'edit') editConfig(row)
  else if (command === 'delete') deleteConfig(row.id)
}

onMounted(() => { loadConfigs() })
</script>

<template>
  <div class="grid gap-4">
    <!-- Actions -->
    <div class="flex justify-end">
      <Button @click="handleAddConfig">
        <Plus class="w-4 h-4 mr-1.5" />
        {{ t('add_config') }}
      </Button>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-[var(--color-line)] overflow-hidden">
      <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
      <Table v-else>
        <TableHeader>
          <TableRow>
            <TableHead class="w-14">ID</TableHead>
            <TableHead>{{ t('config_name') }}</TableHead>
            <TableHead class="w-36">{{ t('config_id') }}</TableHead>
            <TableHead class="w-28">{{ t('provider') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('enabled_status') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('default_config') }}</TableHead>
            <TableHead class="w-44">{{ t('created_at') }}</TableHead>
            <TableHead class="w-14 text-center">{{ t('actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableEmpty v-if="!safeConfigs.length" :colspan="8" />
          <TableRow v-for="row in safeConfigs" :key="row.id">
            <TableCell class="text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
            <TableCell class="font-medium">{{ row.name }}</TableCell>
            <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.config_id }}</TableCell>
            <TableCell>
              <span :class="getProviderBadgeClass(row.provider)"
                class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-semibold">
                {{ row.provider }}
              </span>
            </TableCell>
            <TableCell class="text-center">
              <Switch :checked="row.enabled" @update:checked="v => { row.enabled = v; toggleEnable(row) }" />
            </TableCell>
            <TableCell class="text-center">
              <Switch :checked="row.is_default" @update:checked="v => { row.is_default = v; toggleDefault(row) }" />
            </TableCell>
            <TableCell class="text-sm">{{ formatDate(row.created_at) }}</TableCell>
            <TableCell class="text-center">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="icon" class="h-7 w-7" :aria-label="t('more_actions')">
                    <MoreHorizontal class="w-4 h-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem @click="handleAction('edit', row)">{{ t('edit') }}</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem class="text-destructive focus:text-destructive" @click="handleAction('delete', row)">
                    {{ t('delete') }}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <!-- Pagination -->
    <div v-if="total > pageSize" class="flex items-center justify-end gap-2 text-sm">
      <span class="text-[var(--color-text-secondary)]">{{ total }} items</span>
      <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadConfigs()">Prev</Button>
      <span class="text-[var(--color-text-secondary)]">{{ page }}</span>
      <Button variant="outline" size="sm" :disabled="page * pageSize >= total" @click="page++; loadConfigs()">Next</Button>
    </div>

    <!-- Add/Edit dialog -->
    <Dialog v-model:open="showDialog">
      <DialogContent class="max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{{ editingConfig ? t('edit_memory_config') : t('add_memory_config') }}</DialogTitle>
        </DialogHeader>
        <div class="max-h-[65vh] overflow-y-auto pr-1 grid gap-3 py-2">
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('provider') }}</label>
            <Select :model-value="form.provider" @update:model-value="v => { form.provider = v; handleProviderChange(v) }">
              <SelectTrigger><SelectValue :placeholder="t('select_provider')" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="memobase">Memobase</SelectItem>
                <SelectItem value="mem0">Mem0</SelectItem>
                <SelectItem value="memos">MemOS</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_name') }}</label>
            <Input v-model="form.name" :placeholder="t('enter_config_name')" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_id') }}</label>
            <Input v-model="form.config_id" :placeholder="t('enter_unique_config_id')" />
          </div>

          <!-- Memobase / Mem0 / Memos shared fields -->
          <template v-if="['memobase', 'mem0', 'memos'].includes(form.provider)">
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('api_key') }}</label>
              <Input v-model="form.api_key" type="password"
                :placeholder="form.provider === 'memos' ? t('enter_memos_api_key') : form.provider === 'mem0' ? t('enter_mem0_api_key') : t('memobase_api_key_ph')" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('base_url') }}</label>
              <Input v-model="form.base_url"
                :placeholder="form.provider === 'memos' ? t('enter_memos_base_url') : form.provider === 'mem0' ? t('enter_mem0_base_url') : t('memobase_base_url_ph')" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('enable_search') }}</label>
              <Switch :checked="form.enable_search" @update:checked="v => form.enable_search = v" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('search_threshold') }}</label>
              <NumberInput v-model="form.search_threshold" :min="0" :max="1" :step="0.1" :precision="1" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('search_top_k') }}</label>
              <NumberInput v-model="form.search_top_k" :min="1" :step="1" />
            </div>
          </template>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showDialog = false">{{ t('cancel') }}</Button>
          <Button :disabled="saving" @click="handleSave">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
