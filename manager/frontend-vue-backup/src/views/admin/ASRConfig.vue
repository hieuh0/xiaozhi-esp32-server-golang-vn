<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { testSingleConfig, testWithData, parseJsonData } from '../../utils/configTest'
import ASRConfigForm from './forms/ASRConfigForm.vue'
import { resolveASRProvider } from './forms/configProviderUtils'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
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
  funasr: {
    host: 'localhost',
    port: 10095,
    mode: 'offline',
    sample_rate: 16000,
    chunk_size: [5, 10, 5],
    chunk_interval: 10,
    max_connections: 100,
    timeout: 30,
    auto_end: false
  },
  aliyun_funasr: {
    api_key: '',
    ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/inference/',
    model: 'fun-asr-realtime',
    format: 'pcm',
    sample_rate: 16000,
    language_hints: ['zh'],
    vocabulary_id: '',
    disfluency_removal_enabled: false,
    timeout: 30
  },
  doubao: {
    appid: '',
    access_token: '',
    ws_url: 'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async',
    resource_id: 'volc.bigasr.sauc.duration',
    model_name: 'bigmodel',
    end_window_size: 800,
    enable_punc: true,
    enable_itn: true,
    enable_ddc: false,
    chunk_duration: 200,
    timeout: 30
  },
  aliyun_qwen3: {
    api_key: '',
    ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/realtime',
    model: 'qwen3-asr-flash-realtime',
    format: 'pcm',
    sample_rate: 16000,
    language: 'zh',
    auto_end: false,
    vad_threshold: 0.0,
    vad_silence_ms: 400,
    timeout: 30
  },
  xunfei: {
    appid: '',
    api_key: '',
    api_secret: '',
    host: 'iat-api.xfyun.cn',
    path: '/v2/iat',
    domain: 'iat',
    language: 'zh_cn',
    accent: 'mandarin',
    sample_rate: 16000,
    timeout: 30
  }
})

watch(showDialog, open => {
  if (!open) resetForm()
})

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/asr-configs', { params: { page: page.value, page_size: pageSize.value } })
    configs.value = (response.data.data || []).map(normalizeASRConfigRow)
    total.value = response.data.total || 0
  } catch (error) {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

function normalizeASRConfigRow(row) {
  const data = parseJsonData(row?.json_data)
  return { ...row, provider: resolveASRProvider(row?.provider, row?.config_id, data) }
}

const editConfig = (config) => {
  config = normalizeASRConfigRow(config)
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.is_default = config.is_default
  form.enabled = config.enabled

  try {
    const configObj = JSON.parse(config.json_data || '{}')
    if (configObj.funasr) {
      const funasrConfig = { ...form.funasr, ...configObj.funasr }
      if (typeof funasrConfig.chunk_size === 'number' || !Array.isArray(funasrConfig.chunk_size) || funasrConfig.chunk_size.length !== 3) {
        funasrConfig.chunk_size = [5, 10, 5]
      }
      form.funasr = funasrConfig
    } else if (configObj.aliyun_funasr) {
      form.aliyun_funasr = { ...form.aliyun_funasr, ...configObj.aliyun_funasr }
    } else if (configObj.doubao) {
      form.doubao = { ...form.doubao, ...configObj.doubao }
    } else if (config.provider === 'funasr' && configObj.host) {
      const funasrConfig = { ...form.funasr, ...configObj }
      if (typeof funasrConfig.chunk_size === 'number' || !Array.isArray(funasrConfig.chunk_size) || funasrConfig.chunk_size.length !== 3) {
        funasrConfig.chunk_size = [5, 10, 5]
      }
      form.funasr = funasrConfig
    } else if (config.provider === 'aliyun_funasr' && (configObj.ws_url || configObj.model || configObj.api_key)) {
      form.aliyun_funasr = { ...form.aliyun_funasr, ...configObj }
    } else if (config.provider === 'doubao' && (configObj.appid || configObj.access_token)) {
      form.doubao = { ...form.doubao, ...configObj }
    } else if (configObj.aliyun_qwen3) {
      form.aliyun_qwen3 = { ...form.aliyun_qwen3, ...configObj.aliyun_qwen3 }
    } else if (config.provider === 'aliyun_qwen3' && (configObj.ws_url || configObj.model || configObj.api_key)) {
      form.aliyun_qwen3 = { ...form.aliyun_qwen3, ...configObj }
    } else if (configObj.xunfei) {
      form.xunfei = { ...form.xunfei, ...configObj.xunfei }
    } else if (config.provider === 'xunfei' && (configObj.appid || configObj.api_key || configObj.api_secret)) {
      form.xunfei = { ...form.xunfei, ...configObj }
    }
  } catch (error) {
    console.error(t('parse_config_json_failed'), error)
  }

  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) {
    ElMessage.warning(t('form_not_ready'))
    return
  }
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
      await api.put(`/admin/asr-configs/${editingConfig.value.id}`, configData)
      ElMessage.success(t('config_update_success'))
    } else {
      await api.post('/admin/asr-configs', configData)
      ElMessage.success(t('config_create_success'))
    }
    showDialog.value = false
    loadConfigs()
  } catch (error) {
    const msg = error.response?.data?.error || error.response?.data?.message || error.message
    ElMessage.error(t('save_failed_colon') + msg)
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
    await api.put(`/admin/asr-configs/${config.id}`, configData)
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
  return r.first_packet_ms != null ? t('correct_result', { ms: r.first_packet_ms }) : t('correct')
}
function formatTestResultTip(r) {
  if (!r?.ok) return ''
  return r.first_packet_ms != null ? t('passed_result', { ms: r.first_packet_ms }) : t('passed')
}
function formatTestMessage(result) {
  const base = result.message || ''
  return result.first_packet_ms != null ? `${base} ${result.first_packet_ms}ms` : base
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
        const result = await testSingleConfig('asr', row.config_id)
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
    const result = await testWithData('asr', { [configId]: payload })
    if (result.ok) {
      ElMessage.success(formatTestMessage(result) || t('test_passed'))
    } else {
      ElMessage.warning(result.message || t('test_not_passed'))
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
    await api.delete(`/admin/asr-configs/${id}`)
    ElMessage.success(t('delete_success'))
    loadConfigs()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('delete_failed'))
  }
}

const resetForm = () => {
  editingConfig.value = null
  form.name = ''
  form.config_id = ''
  form.provider = ''
  form.is_default = false
  form.enabled = true
  form.funasr = { host: 'localhost', port: 10095, mode: 'offline', sample_rate: 16000, chunk_size: [5, 10, 5], chunk_interval: 10, max_connections: 100, timeout: 30, auto_end: false }
  form.aliyun_funasr = { api_key: '', ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/inference/', model: 'fun-asr-realtime', format: 'pcm', sample_rate: 16000, language_hints: ['zh'], vocabulary_id: '', disfluency_removal_enabled: false, timeout: 30 }
  form.doubao = { appid: '', access_token: '', ws_url: 'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async', resource_id: 'volc.bigasr.sauc.duration', model_name: 'bigmodel', end_window_size: 800, enable_punc: true, enable_itn: true, enable_ddc: false, chunk_duration: 200, timeout: 30 }
  form.aliyun_qwen3 = { api_key: '', ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/realtime', model: 'qwen3-asr-flash-realtime', format: 'pcm', sample_rate: 16000, language: 'zh', auto_end: false, vad_threshold: 0.0, vad_silence_ms: 400, timeout: 30 }
  form.xunfei = { appid: '', api_key: '', api_secret: '', host: 'iat-api.xfyun.cn', path: '/v2/iat', domain: 'iat', language: 'zh_cn', accent: 'mandarin', sample_rate: 16000, timeout: 30 }
}

const handleAction = (command, row) => {
  switch (command) {
    case 'edit': editConfig(row); break
    case 'test': testConfig(row, 'asr'); break
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
      <Button @click="showDialog = true">
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
            <TableHead>{{ t('provider') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('enabled_status') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('default_config') }}</TableHead>
            <TableHead class="w-28 text-center">{{ t('test_result') }}</TableHead>
            <TableHead class="w-44">{{ t('created_at') }}</TableHead>
            <TableHead class="w-14 text-center">{{ t('actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableEmpty v-if="!configs.length" :colspan="9" />
          <TableRow v-for="row in configs" :key="row.id">
            <TableCell class="text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
            <TableCell class="font-medium">{{ row.name }}</TableCell>
            <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.config_id }}</TableCell>
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
      <DialogContent class="max-w-[720px]">
        <DialogHeader>
          <DialogTitle>{{ editingConfig ? t('edit_asr_config') : t('add_asr_config') }}</DialogTitle>
        </DialogHeader>
        <div class="max-h-[65vh] overflow-y-auto pr-1">
          <ASRConfigForm ref="formRef" :model="form" />
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
