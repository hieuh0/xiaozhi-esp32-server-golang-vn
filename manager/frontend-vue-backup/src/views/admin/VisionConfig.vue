<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { resolveVisionProvider } from './forms/configProviderUtils'
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
const baseSaving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)

const baseForm = reactive({ enable_auth: false, vision_url: '' })

const form = reactive({
  name: '',
  provider: 'aliyun_vision',
  is_default: false,
  enabled: true,
  type: 'openai',
  model_name: 'qwen-vl-max',
  api_key: '',
  base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  max_tokens: 1000,
  temperature: 0.1,
  top_p: 0.1,
  timeout: 30
})

watch(showDialog, open => { if (!open) resetForm() })

const parseJsonData = (jsonData) => {
  try { return JSON.parse(jsonData || '{}') } catch { return {} }
}

const normalizeVisionConfigRow = (config) => {
  const data = parseJsonData(config.json_data)
  return { ...config, provider: resolveVisionProvider(config.provider, config.config_id, data) }
}

const generateConfig = () => JSON.stringify({
  provider: form.provider, type: form.type, model_name: form.model_name, api_key: form.api_key,
  base_url: form.base_url, max_tokens: form.max_tokens, temperature: form.temperature,
  top_p: form.top_p, timeout: form.timeout
})

const validateBase = () => {
  if (!String(baseForm.vision_url || '').trim()) { ElMessage.error(t('enter_vision_url')); return false }
  return true
}

const validateForm = () => {
  if (!form.name.trim()) { ElMessage.error(t('enter_config_name')); return false }
  if (!form.provider) { ElMessage.error(t('select_provider')); return false }
  if (!form.type.trim()) { ElMessage.error(t('enter_type')); return false }
  if (!form.model_name.trim()) { ElMessage.error(t('enter_model_name')); return false }
  if (!form.api_key.trim()) { ElMessage.error(t('enter_api_password')); return false }
  if (!form.base_url.trim()) { ElMessage.error(t('enter_base_url')); return false }
  return true
}

const loadBaseConfig = async () => {
  try {
    const response = await api.get('/admin/vision-base-config')
    const data = response.data.data || {}
    baseForm.enable_auth = data.enable_auth || false
    baseForm.vision_url = data.vision_url || ''
  } catch {}
}

const saveBaseConfig = async () => {
  if (!validateBase()) return
  baseSaving.value = true
  try {
    await api.put('/admin/vision-base-config', { enable_auth: baseForm.enable_auth, vision_url: baseForm.vision_url })
    ElMessage.success(t('basic_config_save_success'))
  } catch {
    ElMessage.error(t('save_failed_check_network'))
  } finally {
    baseSaving.value = false
  }
}

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/vision-configs', { params: { page: page.value, page_size: pageSize.value } })
    const allConfigs = response.data.data || []
    configs.value = allConfigs.filter(c => c.config_id !== 'vision_base').map(normalizeVisionConfigRow)
    total.value = response.data.total || 0
  } catch {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

const editConfig = (config) => {
  const normalized = normalizeVisionConfigRow(config)
  editingConfig.value = normalized
  form.name = normalized.name
  form.provider = normalized.provider
  form.is_default = normalized.is_default
  form.enabled = normalized.enabled
  try {
    const d = parseJsonData(normalized.json_data)
    form.type = d.type || ''
    form.model_name = d.model_name || ''
    form.api_key = d.api_key || ''
    form.base_url = d.base_url || ''
    form.max_tokens = d.max_tokens || 4096
    form.temperature = d.temperature !== undefined ? d.temperature : 0.7
    form.top_p = d.top_p !== undefined ? d.top_p : 0.9
    form.timeout = d.timeout || 30
  } catch {
    ElMessage.warning(t('config_format_error_reset'))
  }
  showDialog.value = true
}

const handleSave = async () => {
  if (!validateForm()) return
  saving.value = true
  try {
    const isFirst = !editingConfig.value && configs.value.length === 0
    const payload = {
      name: form.name, provider: form.provider,
      is_default: isFirst || form.is_default,
      enabled: form.enabled !== undefined ? form.enabled : true,
      json_data: generateConfig()
    }
    if (editingConfig.value) {
      await api.put(`/admin/vision-configs/${editingConfig.value.id}`, payload)
      ElMessage.success(t('update_success'))
    } else {
      await api.post('/admin/vision-configs', payload)
      ElMessage.success(t('add_success'))
    }
    showDialog.value = false
    loadConfigs()
  } catch {
    ElMessage.error(t('save_failed_check_network'))
  } finally {
    saving.value = false
  }
}

const toggleEnable = async (config) => {
  try {
    await api.post(`/admin/configs/${config.id}/toggle`)
    ElMessage.success(config.enabled ? t('enabled_success') : t('disable_success'))
  } catch {
    config.enabled = !config.enabled
    ElMessage.error(t('operation_failed'))
  }
}

const toggleDefault = async (config) => {
  try {
    if (!config.enabled) { ElMessage.warning(t('enable_config_before_default')); config.is_default = false; return }
    await api.put(`/admin/vision-configs/${config.id}`, {
      name: config.name, provider: config.provider,
      is_default: config.is_default, enabled: config.enabled, json_data: config.json_data
    })
    ElMessage.success(config.is_default ? t('set_default_success') : t('cancel_default_success'))
    loadConfigs()
  } catch {
    config.is_default = !config.is_default
    ElMessage.error(t('operation_failed'))
  }
}

const getEnabledConfigs = () => configs.value.filter(c => c.enabled)

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_config'), t('hint'), {
      confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning'
    })
    await api.delete(`/admin/vision-configs/${id}`)
    ElMessage.success(t('delete_success'))
    loadConfigs()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('delete_failed'))
  }
}

const resetForm = () => {
  editingConfig.value = null
  Object.assign(form, {
    name: '', provider: 'aliyun_vision', is_default: false, enabled: true,
    type: 'openai', model_name: 'qwen-vl-max', api_key: '',
    base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    max_tokens: 1000, temperature: 0.1, top_p: 0.1, timeout: 30
  })
}

const handleAction = (command, row) => {
  if (command === 'edit') editConfig(row)
  else if (command === 'delete') deleteConfig(row.id)
}

onMounted(() => { loadBaseConfig(); loadConfigs() })
</script>

<template>
  <div class="grid gap-4">
    <!-- Base config card -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
      <div class="flex items-center justify-between px-6 py-4 border-b border-[var(--color-line)]">
        <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('basic_config') }}</p>
      </div>
      <div class="p-6 grid gap-5 max-w-[600px]">
        <div class="flex items-center justify-between gap-4">
          <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('enable_auth') }}</label>
          <Switch :checked="baseForm.enable_auth" @update:checked="v => baseForm.enable_auth = v" />
        </div>
        <div class="grid gap-1.5">
          <label class="text-sm font-semibold text-[var(--color-text)]">Vision URL</label>
          <Input v-model="baseForm.vision_url" :placeholder="t('vision_url_ph')" />
          <p class="text-xs text-[var(--color-text-secondary)]">{{ t('vision_url_hint') }}</p>
        </div>
        <div>
          <Button :disabled="baseSaving" @click="saveBaseConfig">{{ t('save_basic_config') }}</Button>
        </div>
      </div>
    </div>

    <!-- Config list -->
    <div class="flex justify-between items-center gap-2">
      <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('model_config_list') }}</p>
      <Button @click="showDialog = true">
        <Plus class="w-4 h-4 mr-1.5" />
        {{ t('add_config') }}
      </Button>
    </div>

    <div class="rounded-xl border border-[var(--color-line)] overflow-hidden">
      <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
      <Table v-else>
        <TableHeader>
          <TableRow>
            <TableHead class="w-14">ID</TableHead>
            <TableHead>{{ t('config_name') }}</TableHead>
            <TableHead>{{ t('provider') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('enabled_status') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('default_config') }}</TableHead>
            <TableHead class="w-44">{{ t('created_at') }}</TableHead>
            <TableHead class="w-14 text-center">{{ t('actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableEmpty v-if="!configs.length" :colspan="7" />
          <TableRow v-for="row in configs" :key="row.id">
            <TableCell class="text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
            <TableCell class="font-medium">{{ row.name }}</TableCell>
            <TableCell>{{ row.provider }}</TableCell>
            <TableCell class="text-center">
              <Switch :checked="row.enabled" @update:checked="v => { row.enabled = v; toggleEnable(row) }" />
            </TableCell>
            <TableCell class="text-center">
              <Switch
                :checked="row.is_default"
                :disabled="row.is_default && getEnabledConfigs().length === 1"
                @update:checked="v => { row.is_default = v; toggleDefault(row) }"
              />
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
      <DialogContent class="max-w-[700px]">
        <DialogHeader>
          <DialogTitle>{{ editingConfig ? t('edit_vision_config') : t('add_vision_config') }}</DialogTitle>
        </DialogHeader>
        <div class="max-h-[65vh] overflow-y-auto pr-1 grid gap-3 py-2">
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('provider') }}</label>
            <Select :model-value="form.provider" @update:model-value="v => form.provider = v">
              <SelectTrigger><SelectValue :placeholder="t('select_provider')" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="aliyun_vision">{{ t('aliyun_vision') }}</SelectItem>
                <SelectItem value="doubao_vision">{{ t('doubao_vision') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_name') }}</label>
            <Input v-model="form.name" :placeholder="t('enter_config_name')" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('type') }}</label>
            <Input v-model="form.type" :placeholder="t('enter_type')" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model_name_label') }}</label>
            <Input v-model="form.model_name" :placeholder="t('enter_model_name')" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('api_key') }}</label>
            <Input v-model="form.api_key" type="password" :placeholder="t('enter_api_password')" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('base_url') }}</label>
            <Input v-model="form.base_url" :placeholder="t('enter_base_url')" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('max_tokens_label') }}</label>
            <NumberInput v-model="form.max_tokens" :min="1" :max="100000" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('temperature') }}</label>
            <NumberInput v-model="form.temperature" :min="0" :max="2" :step="0.1" :precision="1" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Top P</label>
            <NumberInput v-model="form.top_p" :min="0" :max="1" :step="0.1" :precision="1" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('timeout_seconds') }}</label>
            <NumberInput v-model="form.timeout" :min="1" :max="300" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showDialog = false">{{ t('cancel') }}</Button>
          <Button :disabled="saving" @click="handleSave">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
