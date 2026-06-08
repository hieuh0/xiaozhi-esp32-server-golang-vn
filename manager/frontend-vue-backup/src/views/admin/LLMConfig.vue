<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { testSingleConfig, testWithData, parseJsonData } from '../../utils/configTest'
import LLMConfigForm from './forms/LLMConfigForm.vue'
import { getProviderFixedType, getProviderThinkingConfig, resolveLLMProvider } from './forms/llmCatalog'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

const { t } = useLocale()
const { formatDate } = useFormatDate()

const configs = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const testingId = ref(null)
const testingAll = ref(false)
const testingCurrent = ref(false)
const testResults = ref({})
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)
const formRef = ref()

const form = reactive({
  name: '',
  config_id: '',
  provider: '',
  is_default: false,
  enabled: true,
  type: 'openai',
  model_name: 'gpt-3.5-turbo',
  api_key: '',
  base_url: 'https://api.openai.com/v1',
  max_tokens: 4000,
  temperature: 0.7,
  top_p: 0.9,
  thinking_mode: 'default',
  thinking_budget_tokens: null,
  thinking_effort: 'medium',
  thinking_clear_thinking: 'default',
  bot_id: '',
  user_prefix: '',
  connector_id: '1024'
})

watch(showDialog, open => {
  if (!open) resetForm()
})

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/llm-configs', { params: { page: page.value, page_size: pageSize.value } })
    configs.value = (response.data.data || []).map((row) => {
      const configObj = parseJsonData(row?.json_data)
      return { ...row, provider: resolveLLMProvider(row?.provider, configObj?.type) }
    })
    total.value = response.data.total || 0
  } catch (error) {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

const editConfig = (config) => {
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.is_default = config.is_default
  form.enabled = config.enabled

  try {
    const configObj = JSON.parse(config.json_data || '{}')
    const detectedProvider = resolveLLMProvider(config.provider, configObj.type)
    const detectedType = getProviderFixedType(detectedProvider)
    form.provider = detectedProvider
    form.type = detectedType
    form.model_name = configObj.model_name || (detectedType === 'coze' ? 'coze' : (detectedType === 'dify' ? 'dify' : ''))
    form.api_key = configObj.api_key || ''
    form.base_url = configObj.base_url || (detectedType === 'coze' ? 'https://api.coze.com' : (detectedType === 'dify' ? 'https://api.dify.ai/v1' : ''))
    form.max_tokens = configObj.max_tokens || 4000
    form.temperature = configObj.temperature || 0.7
    form.top_p = configObj.top_p || 0.9
    form.thinking_mode = configObj.thinking?.mode || 'default'
    form.thinking_budget_tokens = configObj.thinking?.budget_tokens !== undefined ? Number(configObj.thinking.budget_tokens) || null : null
    form.thinking_effort = configObj.thinking?.effort || 'medium'
    form.thinking_clear_thinking = configObj.thinking?.clear_thinking !== undefined ? configObj.thinking.clear_thinking : 'default'
    form.bot_id = configObj.bot_id || ''
    form.user_prefix = configObj.user_prefix || ''
    form.connector_id = configObj.connector_id || '1024'
  } catch (error) {
    console.error(t('parse_config_json_failed'), error)
  }

  showDialog.value = true
}

const openCreateDialog = () => {
  resetForm()
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch (_) {
    return
  }
  saving.value = true
  try {
    const isFirstConfig = !editingConfig.value && configs.value.length === 0
    const configData = {
      name: form.name,
      config_id: form.config_id,
      provider: form.provider,
      is_default: isFirstConfig || form.is_default,
      enabled: form.enabled !== undefined ? form.enabled : true,
      json_data: formRef.value.getJsonData()
    }
    if (editingConfig.value) {
      await api.put(`/admin/llm-configs/${editingConfig.value.id}`, configData)
      ElMessage.success(t('config_update_success'))
    } else {
      await api.post('/admin/llm-configs', configData)
      ElMessage.success(t('config_create_success'))
    }
    showDialog.value = false
    loadConfigs()
  } catch (error) {
    ElMessage.error(t('save_failed_colon') + (error.response?.data?.message || error.message))
  } finally {
    saving.value = false
  }
}

const toggleEnable = async (config) => {
  try {
    await api.post(`/admin/configs/${config.id}/toggle`)
    ElMessage.success(config.enabled ? t('enabled_success') : t('disable_success'))
  } catch (error) {
    config.enabled = !config.enabled
    ElMessage.error(t('operation_failed'))
  }
}

const toggleDefault = async (config) => {
  try {
    if (!config.enabled) {
      ElMessage.warning(t('enable_config_before_default'))
      config.is_default = false
      return
    }
    const configData = {
      name: config.name,
      config_id: config.config_id,
      provider: config.provider,
      is_default: config.is_default,
      enabled: config.enabled,
      json_data: config.json_data
    }
    await api.put(`/admin/llm-configs/${config.id}`, configData)
    ElMessage.success(config.is_default ? t('set_default_success') : t('cancel_default_success'))
    loadConfigs()
  } catch (error) {
    config.is_default = !config.is_default
    ElMessage.error(t('operation_failed'))
  }
}

const getEnabledConfigs = () => configs.value.filter(config => config.enabled)

function formatTestResultLabel(r) {
  if (!r?.ok) return t('error')
  return r.first_packet_ms != null ? `${r.first_packet_ms}ms` : t('passed')
}
function formatTestResultTip(r) {
  if (!r?.ok) return ''
  const parts = []
  if (r.first_packet_ms != null) parts.push(t('first_packet_result', { ms: r.first_packet_ms }))
  if (r.reasoning_content_returned) parts.push(t('upstream_thinking_detected'))
  return parts.length ? parts.join(t('list_join_sep')) : t('passed')
}
function formatTestMessage(result) {
  const base = result.message || ''
  const suffix = []
  if (result.first_packet_ms != null) suffix.push(`${result.first_packet_ms}ms`)
  if (result.reasoning_content_returned) suffix.push(t('upstream_thinking_detected'))
  return suffix.length ? `${base} ${suffix.join(' · ')}` : base
}
function formatDraftTestLabel(name, configId) {
  return name?.trim() || configId?.trim() || t('current_config')
}
function getThinkingLabel(row) {
  const config = parseJsonData(row?.json_data)
  const provider = resolveLLMProvider(row?.provider, config?.type)
  const mode = String(config?.thinking?.mode || '').trim()
  if (!mode || mode === 'default') return ''
  const thinkingConfig = getProviderThinkingConfig(provider, config?.model_name)
  const option = (thinkingConfig?.options || []).find(item => item.value === mode)
  return option?.label || mode
}
function getProviderLabel(provider) {
  const labels = {
    azure: 'Azure OpenAI',
    anthropic: 'Anthropic',
    zhipu: t('zhipu_ai'),
    aliyun: t('aliyun'),
    doubao: t('doubao'),
    siliconflow: t('silicon_flow'),
    deepseek: t('deepseek_label'),
    openai: 'OpenAI',
    ollama: 'Ollama',
    dify: 'Dify',
    coze: 'Coze'
  }
  return labels[provider] || provider
}

const testConfig = async (row, type) => {
  testingId.value = row.config_id
  try {
    const result = await testSingleConfig(type, row.config_id)
    testResults.value = { ...testResults.value, [row.config_id]: result }
    if (result.ok) {
      ElMessage.success(`${row.name || row.config_id}${t('label_colon')}${formatTestMessage(result)}`)
    } else {
      ElMessage.warning(`${row.name || row.config_id}${t('label_colon')}${result.message}`)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || t('test_request_failed_v2'))
  } finally {
    testingId.value = null
  }
}

const testAllConfigs = async () => {
  const list = getEnabledConfigs()
  if (!list.length) {
    ElMessage.warning(t('no_enabled_config'))
    return
  }
  testingAll.value = true
  testResults.value = {}
  let okCount = 0
  try {
    for (const row of list) {
      try {
        const result = await testSingleConfig('llm', row.config_id)
        testResults.value = { ...testResults.value, [row.config_id]: result }
        if (result.ok) okCount++
      } catch (_) {
        testResults.value = { ...testResults.value, [row.config_id]: { ok: false, message: t('request_failed') } }
      }
    }
    ElMessage.success(t('all_tests_complete_msg', { ok: okCount, total: list.length }))
  } catch (err) {
    ElMessage.error(err.response?.data?.error || t('test_request_failed_v2'))
  } finally {
    testingAll.value = false
  }
}

const testCurrentConfig = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch (_) {
    return
  }
  const configId = form.config_id?.trim()
  if (!configId) {
    ElMessage.warning(t('fill_config_id'))
    return
  }
  const payload = {
    name: form.name,
    config_id: configId,
    provider: form.provider,
    is_default: form.is_default,
    ...parseJsonData(formRef.value.getJsonData())
  }
  testingCurrent.value = true
  try {
    const result = await testWithData('llm', { provider: configId, [configId]: payload })
    const label = formatDraftTestLabel(form.name, configId)
    if (result.ok) {
      ElMessage.success(`${label}${t('label_colon')}${formatTestMessage(result) || t('test_passed')}`)
    } else {
      ElMessage.warning(`${label}${t('label_colon')}${result.message || t('test_not_passed')}`)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || t('test_request_failed_v2'))
  } finally {
    testingCurrent.value = false
  }
}

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_config'), t('hint'), {
      confirmButtonText: t('confirm'),
      cancelButtonText: t('cancel'),
      type: 'warning'
    })
    await api.delete(`/admin/llm-configs/${id}`)
    ElMessage.success(t('delete_success'))
    loadConfigs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('delete_failed'))
    }
  }
}

const resetForm = () => {
  editingConfig.value = null
  form.name = ''
  form.config_id = ''
  form.provider = ''
  form.is_default = false
  form.enabled = true
  form.type = 'openai'
  form.model_name = ''
  form.api_key = ''
  form.base_url = ''
  form.max_tokens = 4000
  form.temperature = 0.7
  form.top_p = 0.9
  form.thinking_mode = 'default'
  form.thinking_budget_tokens = null
  form.thinking_effort = 'medium'
  form.thinking_clear_thinking = 'default'
  form.bot_id = ''
  form.user_prefix = ''
  form.connector_id = '1024'
}

const handleAction = (command, row) => {
  switch (command) {
    case 'edit': editConfig(row); break
    case 'test': testConfig(row, 'llm'); break
    case 'delete': deleteConfig(row.id); break
  }
}

onMounted(loadConfigs)
</script>

<template>
  <div class="grid gap-4">
    <!-- Actions -->
    <div class="flex justify-end items-center gap-2 flex-wrap">
      <Button variant="outline" :disabled="testingAll || !getEnabledConfigs().length" @click="testAllConfigs">
        {{ t('test_all') }}
      </Button>
      <Button @click="openCreateDialog">
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
            <TableHead>{{ t('type') }}</TableHead>
            <TableHead class="w-24 text-center">{{ t('thinking_col') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('enabled_status') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('default_config') }}</TableHead>
            <TableHead class="w-24 text-center">{{ t('time_cost') }}</TableHead>
            <TableHead class="w-44">{{ t('created_at') }}</TableHead>
            <TableHead class="w-14 text-center">{{ t('actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableEmpty v-if="!configs.length" :colspan="10" />
          <TableRow v-for="row in configs" :key="row.id">
            <TableCell class="text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
            <TableCell class="font-medium">{{ row.name }}</TableCell>
            <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.config_id }}</TableCell>
            <TableCell>{{ getProviderLabel(row.provider) }}</TableCell>
            <TableCell class="text-center">
              <Badge v-if="getThinkingLabel(row)" variant="secondary" class="max-w-[84px] truncate text-xs">
                {{ getThinkingLabel(row) }}
              </Badge>
              <span v-else class="text-xs text-[var(--color-text-tertiary)]">–</span>
            </TableCell>
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
            <TableCell class="text-center">
              <template v-if="testResults[row.config_id]">
                <TooltipProvider v-if="testResults[row.config_id].ok">
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <span class="text-xs font-medium text-green-600 dark:text-green-400 cursor-default">
                        {{ formatTestResultLabel(testResults[row.config_id]) }}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>{{ formatTestResultTip(testResults[row.config_id]) }}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <TooltipProvider v-else>
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <span class="text-xs font-medium text-red-600 dark:text-red-400 cursor-help">{{ t('error') }}</span>
                    </TooltipTrigger>
                    <TooltipContent>{{ testResults[row.config_id].message }}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </template>
              <span v-else class="text-xs text-[var(--color-text-tertiary)]">–</span>
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
                  <DropdownMenuItem @click="handleAction('test', row)">{{ t('test') }}</DropdownMenuItem>
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

    <!-- Add / Edit dialog -->
    <Dialog v-model:open="showDialog">
      <DialogContent class="max-w-[620px]">
        <DialogHeader>
          <DialogTitle>{{ editingConfig ? t('edit_llm_config') : t('add_llm_config') }}</DialogTitle>
        </DialogHeader>
        <div class="max-h-[65vh] overflow-y-auto pr-1">
          <LLMConfigForm ref="formRef" :model="form" />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showDialog = false">{{ t('cancel') }}</Button>
          <Button variant="outline" :disabled="testingCurrent" @click="testCurrentConfig">{{ t('test') }}</Button>
          <Button :disabled="saving" @click="handleSave">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
