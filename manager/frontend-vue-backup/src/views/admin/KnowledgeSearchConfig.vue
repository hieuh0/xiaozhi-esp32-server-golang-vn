<script setup>
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '@/utils/api'
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

const items = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const editing = ref(false)
const currentId = ref(null)
const weknoraModelLoading = ref(false)
const weknoraModelLoadError = ref('')
const weknoraEmbeddingModels = ref([])
const weknoraLLMModels = ref([])
const weknoraRerankModels = ref([])
const lastWeknoraFetchKey = ref('')
const rowSwitchLoading = ref({})
let weknoraModelFetchTimer = null
let weknoraFetchSeq = 0

const DEFAULT_DIFY_BASE_URL = 'https://api.dify.ai/v1'
const DEFAULT_RAGFLOW_BASE_URL = 'http://127.0.0.1'
const DEFAULT_WEKNORA_BASE_URL = 'http://127.0.0.1:8080/api/v1'
const DEFAULT_DIFY_SCORE_THRESHOLD = 0.2
const DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD = 0.2
const DEFAULT_WEKNORA_SCORE_THRESHOLD = 0.2
const DEFAULT_WEKNORA_CHUNK_SIZE = 1000
const DEFAULT_WEKNORA_CHUNK_OVERLAP = 200
const DEFAULT_WEKNORA_SEPARATORS = ['\\n\\n', '\\n', '。', '！', '？', ';', '；']
const DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS = 1000
const DEFAULT_WEKNORA_PARSE_TIMEOUT_MS = 120000

const form = reactive({
  name: '', config_id: '', provider: 'dify', base_url: DEFAULT_DIFY_BASE_URL, api_key: '',
  score_threshold: DEFAULT_DIFY_SCORE_THRESHOLD, dataset_permission: '', dataset_provider: '',
  dataset_indexing_technique: '', similarity_threshold: DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD,
  vector_similarity_weight: 0.3, keyword: false, highlight: false, dataset_chunk_method: '',
  embedding_model_id: '', chunk_size: DEFAULT_WEKNORA_CHUNK_SIZE, chunk_overlap: DEFAULT_WEKNORA_CHUNK_OVERLAP,
  separators_raw: DEFAULT_WEKNORA_SEPARATORS.join(','), enable_multimodal: true, summary_model_id: '',
  rerank_model_id: '', vlm_model_id: '', parse_poll_interval_ms: DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS,
  parse_timeout_ms: DEFAULT_WEKNORA_PARSE_TIMEOUT_MS, enabled: true, is_default: false
})

const normalizeProvider = (provider) => {
  const p = String(provider || '').trim().toLowerCase()
  return ['dify', 'ragflow', 'weknora'].includes(p) ? p : 'dify'
}

const PROVIDER_WEBSITE = {
  dify: 'https://dify.ai/',
  ragflow: 'https://github.com/infiniflow/ragflow',
  weknora: 'https://github.com/Tencent/WeKnora'
}

const getProviderWebsite = (provider) => PROVIDER_WEBSITE[normalizeProvider(provider)] || PROVIDER_WEBSITE.dify

const parseSeparators = (raw) => {
  if (Array.isArray(raw)) {
    const values = raw.map(item => String(item || '').trim()).filter(Boolean)
    return values.length > 0 ? values : [...DEFAULT_WEKNORA_SEPARATORS]
  }
  const text = String(raw || '').trim()
  if (!text) return [...DEFAULT_WEKNORA_SEPARATORS]
  if (text.startsWith('[') && text.endsWith(']')) {
    try {
      const arr = JSON.parse(text)
      if (Array.isArray(arr)) {
        const values = arr.map(item => String(item || '').trim()).filter(Boolean)
        if (values.length > 0) return values
      }
    } catch {}
  }
  const values = text.split(',').map(item => item.trim()).filter(Boolean)
  return values.length > 0 ? values : [...DEFAULT_WEKNORA_SEPARATORS]
}

const normalizeModelOptions = (list) => {
  if (!Array.isArray(list)) return []
  const ret = []
  const seen = new Set()
  list.forEach((item) => {
    const id = String(item?.id || '').trim()
    if (!id || seen.has(id)) return
    seen.add(id)
    ret.push({ id, name: String(item?.name || id).trim() || id })
  })
  return ret
}

const clearWeknoraModels = () => {
  weknoraEmbeddingModels.value = []
  weknoraLLMModels.value = []
  weknoraRerankModels.value = []
  weknoraModelLoadError.value = ''
  lastWeknoraFetchKey.value = ''
}

const fetchWeknoraModels = async (force = false, silent = true) => {
  if (normalizeProvider(form.provider) !== 'weknora') return
  const baseURL = String(form.base_url || '').trim()
  const apiKey = String(form.api_key || '').trim()
  if (!baseURL || !apiKey) {
    if (!silent) ElMessage.warning(t('fill_weknora_url_apikey'))
    return
  }
  const fetchKey = `${baseURL}::${apiKey}`
  if (!force && fetchKey === lastWeknoraFetchKey.value &&
    weknoraEmbeddingModels.value.length + weknoraLLMModels.value.length + weknoraRerankModels.value.length > 0) return

  const seq = ++weknoraFetchSeq
  weknoraModelLoading.value = true
  weknoraModelLoadError.value = ''
  try {
    const res = await api.post('/admin/knowledge-search-configs/weknora/models', { base_url: baseURL, api_key: apiKey })
    if (seq !== weknoraFetchSeq) return
    const data = res?.data?.data || {}
    const embedding = normalizeModelOptions(data.embedding_models)
    const llm = normalizeModelOptions(data.llm_models)
    const rerank = normalizeModelOptions(data.rerank_models)
    weknoraEmbeddingModels.value = embedding
    weknoraLLMModels.value = llm
    weknoraRerankModels.value = rerank
    lastWeknoraFetchKey.value = fetchKey
    if (!String(form.embedding_model_id || '').trim() && embedding.length > 0) form.embedding_model_id = embedding[0].id
    if (!String(form.summary_model_id || '').trim() && llm.length > 0) form.summary_model_id = llm[0].id
  } catch (e) {
    if (seq !== weknoraFetchSeq) return
    const msg = e?.response?.data?.error || t('fetch_weknora_failed')
    weknoraModelLoadError.value = msg
    if (!silent) ElMessage.error(msg)
  } finally {
    if (seq === weknoraFetchSeq) weknoraModelLoading.value = false
  }
}

const scheduleWeknoraModelFetch = (force = false, silent = true) => {
  if (weknoraModelFetchTimer) { clearTimeout(weknoraModelFetchTimer); weknoraModelFetchTimer = null }
  if (normalizeProvider(form.provider) !== 'weknora') return
  const baseURL = String(form.base_url || '').trim()
  const apiKey = String(form.api_key || '').trim()
  if (!baseURL || !apiKey) return
  weknoraModelFetchTimer = setTimeout(() => { fetchWeknoraModels(force, silent) }, 450)
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await api.get('/admin/knowledge-search-configs', { params: { page: page.value, page_size: pageSize.value } })
    items.value = res.data.data || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const applyProviderDefaults = (provider, force = false) => {
  provider = normalizeProvider(provider)
  if (provider === 'dify') {
    if (force || !form.base_url || form.base_url === DEFAULT_RAGFLOW_BASE_URL || form.base_url === DEFAULT_WEKNORA_BASE_URL) form.base_url = DEFAULT_DIFY_BASE_URL
    if (force || Number.isNaN(Number(form.score_threshold))) form.score_threshold = DEFAULT_DIFY_SCORE_THRESHOLD
    return
  }
  if (provider === 'ragflow') {
    if (force || !form.base_url || form.base_url === DEFAULT_DIFY_BASE_URL || form.base_url === DEFAULT_WEKNORA_BASE_URL) form.base_url = DEFAULT_RAGFLOW_BASE_URL
    if (force || Number.isNaN(Number(form.similarity_threshold))) form.similarity_threshold = DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD
    return
  }
  if (provider === 'weknora') {
    if (force || !form.base_url || form.base_url === DEFAULT_DIFY_BASE_URL || form.base_url === DEFAULT_RAGFLOW_BASE_URL) form.base_url = DEFAULT_WEKNORA_BASE_URL
    if (force || Number.isNaN(Number(form.score_threshold))) form.score_threshold = DEFAULT_WEKNORA_SCORE_THRESHOLD
    if (force || Number.isNaN(Number(form.chunk_size)) || Number(form.chunk_size) <= 0) form.chunk_size = DEFAULT_WEKNORA_CHUNK_SIZE
    if (force || Number.isNaN(Number(form.chunk_overlap)) || Number(form.chunk_overlap) < 0) form.chunk_overlap = DEFAULT_WEKNORA_CHUNK_OVERLAP
    if (force || !String(form.separators_raw || '').trim()) form.separators_raw = DEFAULT_WEKNORA_SEPARATORS.join(',')
    if (force || Number.isNaN(Number(form.parse_poll_interval_ms)) || Number(form.parse_poll_interval_ms) <= 0) form.parse_poll_interval_ms = DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS
    if (force || Number.isNaN(Number(form.parse_timeout_ms)) || Number(form.parse_timeout_ms) <= 0) form.parse_timeout_ms = DEFAULT_WEKNORA_PARSE_TIMEOUT_MS
  }
}

const onProviderChange = (provider) => {
  form.provider = provider
  applyProviderDefaults(provider, true)
  if (normalizeProvider(provider) === 'weknora') scheduleWeknoraModelFetch(true, true)
  else clearWeknoraModels()
}

const openDialog = (row = null) => {
  editing.value = !!row
  currentId.value = row?.id || null
  const data = row?.json_data ? JSON.parse(row.json_data || '{}') : {}
  const provider = normalizeProvider(row?.provider || 'dify')
  const separators = parseSeparators(data.separators)
  form.name = row?.name || ''
  form.config_id = row?.config_id || ''
  form.provider = provider
  form.base_url = data.base_url || (provider === 'ragflow' ? DEFAULT_RAGFLOW_BASE_URL : provider === 'weknora' ? DEFAULT_WEKNORA_BASE_URL : DEFAULT_DIFY_BASE_URL)
  form.api_key = data.api_key || ''
  form.score_threshold = Number(data.score_threshold ?? (provider === 'weknora' ? DEFAULT_WEKNORA_SCORE_THRESHOLD : DEFAULT_DIFY_SCORE_THRESHOLD))
  form.dataset_permission = data.dataset_permission || ''
  form.dataset_provider = data.dataset_provider || ''
  form.dataset_indexing_technique = data.dataset_indexing_technique || ''
  form.similarity_threshold = Number(data.similarity_threshold ?? DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD)
  form.vector_similarity_weight = Number(data.vector_similarity_weight ?? 0.3)
  form.keyword = !!data.keyword
  form.highlight = !!data.highlight
  form.dataset_chunk_method = data.dataset_chunk_method || ''
  form.embedding_model_id = data.embedding_model_id || ''
  form.chunk_size = Number(data.chunk_size ?? DEFAULT_WEKNORA_CHUNK_SIZE)
  form.chunk_overlap = Number(data.chunk_overlap ?? DEFAULT_WEKNORA_CHUNK_OVERLAP)
  form.separators_raw = separators.join(',')
  form.enable_multimodal = data.enable_multimodal !== undefined ? !!data.enable_multimodal : true
  form.summary_model_id = data.summary_model_id || ''
  form.rerank_model_id = data.rerank_model_id || ''
  form.vlm_model_id = data.vlm_model_id || ''
  form.parse_poll_interval_ms = Number(data.parse_poll_interval_ms ?? DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS)
  form.parse_timeout_ms = Number(data.parse_timeout_ms ?? DEFAULT_WEKNORA_PARSE_TIMEOUT_MS)
  form.enabled = row?.enabled ?? true
  form.is_default = row?.is_default ?? false
  if (!row) { applyProviderDefaults(provider, true); clearWeknoraModels() }
  else if (provider !== 'weknora') clearWeknoraModels()
  if (provider === 'weknora') scheduleWeknoraModelFetch(false, true)
  dialogVisible.value = true
}

const submit = async () => {
  if (form.provider === 'weknora' && !String(form.embedding_model_id || '').trim()) {
    ElMessage.error(t('embedding_model_id_required')); return
  }
  const weknoraSeparators = parseSeparators(form.separators_raw)
  const payload = {
    type: 'knowledge_search', name: form.name, config_id: form.config_id, provider: form.provider,
    enabled: form.enabled, is_default: form.is_default,
    json_data: JSON.stringify(form.provider === 'dify'
      ? { base_url: form.base_url, api_key: form.api_key, score_threshold: form.score_threshold,
          dataset_permission: form.dataset_permission, dataset_provider: form.dataset_provider,
          dataset_indexing_technique: form.dataset_indexing_technique }
      : form.provider === 'ragflow'
        ? { base_url: form.base_url, api_key: form.api_key, similarity_threshold: form.similarity_threshold,
            vector_similarity_weight: form.vector_similarity_weight, keyword: form.keyword,
            highlight: form.highlight, dataset_permission: form.dataset_permission,
            dataset_chunk_method: form.dataset_chunk_method }
        : { base_url: form.base_url, api_key: form.api_key, score_threshold: form.score_threshold,
            embedding_model_id: String(form.embedding_model_id || '').trim(),
            chunk_size: Number(form.chunk_size) || DEFAULT_WEKNORA_CHUNK_SIZE,
            chunk_overlap: Number(form.chunk_overlap) || DEFAULT_WEKNORA_CHUNK_OVERLAP,
            separators: weknoraSeparators, enable_multimodal: !!form.enable_multimodal,
            summary_model_id: String(form.summary_model_id || '').trim(),
            rerank_model_id: String(form.rerank_model_id || '').trim(),
            vlm_model_id: String(form.vlm_model_id || '').trim(),
            parse_poll_interval_ms: Number(form.parse_poll_interval_ms) || DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS,
            parse_timeout_ms: Number(form.parse_timeout_ms) || DEFAULT_WEKNORA_PARSE_TIMEOUT_MS })
  }
  try {
    if (editing.value) await api.put(`/admin/knowledge-search-configs/${currentId.value}`, payload)
    else await api.post('/admin/knowledge-search-configs', payload)
    ElMessage.success(t('save_success'))
    dialogVisible.value = false
    await loadData()
  } catch {
    ElMessage.error(t('save_failed'))
  }
}

const isRowSwitchLoading = (id, field) => !!rowSwitchLoading.value[`${field}_${id}`]

const setRowSwitchLoading = (id, field, val) => {
  rowSwitchLoading.value = { ...rowSwitchLoading.value, [`${field}_${id}`]: val }
}

const onRowSwitchChange = async (row, field, value) => {
  const id = row?.id
  if (!id || !['enabled', 'is_default'].includes(field)) return
  if (isRowSwitchLoading(id, field)) return
  setRowSwitchLoading(id, field, true)
  try {
    const provider = normalizeProvider(row.provider)
    const rawData = row?.json_data ? JSON.parse(row.json_data || '{}') : {}
    let enabled = field === 'enabled' ? !!value : !!row.enabled
    let isDefault = field === 'is_default' ? !!value : !!row.is_default
    if (!enabled && isDefault) isDefault = false
    if (isDefault && !enabled) enabled = true
    const payload = { type: 'knowledge_search', name: row.name, config_id: row.config_id, provider, enabled, is_default: isDefault, json_data: JSON.stringify(rawData) }
    await api.put(`/admin/knowledge-search-configs/${id}`, payload)
    if (isDefault) items.value.forEach((item) => { if (item.id !== id) item.is_default = false })
    row.enabled = enabled
    row.is_default = isDefault
    ElMessage.success(t('update_success'))
    await loadData()
  } catch {
    await loadData()
    ElMessage.error(t('update_failed'))
  } finally {
    setRowSwitchLoading(id, field, false)
  }
}

const remove = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_this_config'), t('hint'), { type: 'warning' })
    await api.delete(`/admin/knowledge-search-configs/${id}`)
    ElMessage.success(t('delete_success'))
    await loadData()
  } catch {}
}

const getConfigSummary = (row) => {
  const data = row?.json_data ? JSON.parse(row.json_data || '{}') : {}
  const provider = normalizeProvider(row?.provider)
  if (provider === 'dify') return `base_url: ${data.base_url || DEFAULT_DIFY_BASE_URL}; score_threshold: ${data.score_threshold ?? DEFAULT_DIFY_SCORE_THRESHOLD}`
  if (provider === 'ragflow') return `base_url: ${data.base_url || DEFAULT_RAGFLOW_BASE_URL}; similarity_threshold: ${data.similarity_threshold ?? DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD}`
  if (provider === 'weknora') return `base_url: ${data.base_url || DEFAULT_WEKNORA_BASE_URL}; score_threshold: ${data.score_threshold ?? DEFAULT_WEKNORA_SCORE_THRESHOLD}`
  return '-'
}

const handleAction = (command, row) => {
  if (command === 'edit') openDialog(row)
  else if (command === 'delete') remove(row.id)
}

onMounted(loadData)

watch(
  () => [normalizeProvider(form.provider), String(form.base_url || '').trim(), String(form.api_key || '').trim()],
  ([provider]) => {
    if (provider !== 'weknora') { clearWeknoraModels(); return }
    scheduleWeknoraModelFetch(false, true)
  }
)

onBeforeUnmount(() => {
  if (weknoraModelFetchTimer) { clearTimeout(weknoraModelFetchTimer); weknoraModelFetchTimer = null }
})
</script>

<template>
  <div class="grid gap-4">
    <!-- Actions -->
    <div class="flex justify-end">
      <Button @click="openDialog()">
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
            <TableHead class="w-28">{{ t('provider') }}</TableHead>
            <TableHead class="w-36">{{ t('name') }}</TableHead>
            <TableHead class="w-40">{{ t('config_id') }}</TableHead>
            <TableHead>{{ t('config_summary') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('enabled') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('default') }}</TableHead>
            <TableHead class="w-14 text-center">{{ t('actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableEmpty v-if="!items.length" :colspan="8" />
          <TableRow v-for="row in items" :key="row.id">
            <TableCell class="text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
            <TableCell>{{ row.provider }}</TableCell>
            <TableCell class="font-medium">{{ row.name }}</TableCell>
            <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.config_id }}</TableCell>
            <TableCell class="text-sm text-[var(--color-text-secondary)] max-w-xs truncate">{{ getConfigSummary(row) }}</TableCell>
            <TableCell class="text-center">
              <Switch
                :checked="row.enabled"
                :disabled="isRowSwitchLoading(row.id, 'enabled')"
                @update:checked="v => onRowSwitchChange(row, 'enabled', v)"
              />
            </TableCell>
            <TableCell class="text-center">
              <Switch
                :checked="row.is_default"
                :disabled="isRowSwitchLoading(row.id, 'is_default') || (!row.enabled && !row.is_default)"
                @update:checked="v => onRowSwitchChange(row, 'is_default', v)"
              />
            </TableCell>
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
      <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadData()">Prev</Button>
      <span class="text-[var(--color-text-secondary)]">{{ page }}</span>
      <Button variant="outline" size="sm" :disabled="page * pageSize >= total" @click="page++; loadData()">Next</Button>
    </div>

    <!-- Add/Edit dialog -->
    <Dialog v-model:open="dialogVisible">
      <DialogContent class="max-w-[700px]">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('edit_config') : t('add_config_new') }}</DialogTitle>
        </DialogHeader>
        <div class="max-h-[65vh] overflow-y-auto pr-1 grid gap-3 py-2">

          <!-- Provider -->
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('provider') }}</label>
            <Select :model-value="form.provider" @update:model-value="onProviderChange">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="dify">dify</SelectItem>
                <SelectItem value="ragflow">ragflow</SelectItem>
                <SelectItem value="weknora">weknora</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <!-- Provider website link -->
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('provider_website') }}</label>
            <a :href="getProviderWebsite(form.provider)" target="_blank" rel="noopener noreferrer"
              class="text-sm text-[var(--color-primary)] hover:underline truncate">
              {{ getProviderWebsite(form.provider) }}
            </a>
          </div>

          <!-- Common fields -->
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('name') }}</label>
            <Input v-model="form.name" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_id') }}</label>
            <Input v-model="form.config_id" />
          </div>

          <!-- Dify fields -->
          <template v-if="form.provider === 'dify'">
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Base URL</label>
              <Input v-model="form.base_url" :placeholder="DEFAULT_DIFY_BASE_URL" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
              <Input v-model="form.api_key" type="password" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('threshold') }}</label>
              <NumberInput v-model="form.score_threshold" :min="0" :max="1" :step="0.01" :precision="2" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-start">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('dataset_permission') }}</label>
              <div class="grid gap-1.5">
                <Select :model-value="form.dataset_permission" @update:model-value="v => form.dataset_permission = v">
                  <SelectTrigger><SelectValue :placeholder="t('select_placeholder')" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="only_me">{{ t('only_me_option') }}</SelectItem>
                    <SelectItem value="all_team_members">{{ t('all_team_members_option') }}</SelectItem>
                    <SelectItem value="partial_members">{{ t('partial_members_option') }}</SelectItem>
                  </SelectContent>
                </Select>
                <p class="text-xs text-[var(--color-text-secondary)]">{{ t('dataset_permission_hint') }}</p>
              </div>
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('dataset_provider_label') }}</label>
              <Input v-model="form.dataset_provider" placeholder="vendor" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('index_strategy') }}</label>
              <Select :model-value="form.dataset_indexing_technique" @update:model-value="v => form.dataset_indexing_technique = v">
                <SelectTrigger><SelectValue :placeholder="t('select_placeholder')" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="high_quality">{{ t('high_quality_option') }}</SelectItem>
                  <SelectItem value="economy">{{ t('economy_option') }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </template>

          <!-- Ragflow fields -->
          <template v-else-if="form.provider === 'ragflow'">
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Base URL</label>
              <Input v-model="form.base_url" :placeholder="DEFAULT_RAGFLOW_BASE_URL" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
              <Input v-model="form.api_key" type="password" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('similarity_threshold_label') }}</label>
              <NumberInput v-model="form.similarity_threshold" :min="0" :max="1" :step="0.01" :precision="2" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('vector_weight_label') }}</label>
              <NumberInput v-model="form.vector_similarity_weight" :min="0" :max="1" :step="0.01" :precision="2" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('enable_keyword_label') }}</label>
              <Switch :checked="form.keyword" @update:checked="v => form.keyword = v" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('enable_highlight_label') }}</label>
              <Switch :checked="form.highlight" @update:checked="v => form.highlight = v" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-start">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('dataset_permission') }}</label>
              <div class="grid gap-1.5">
                <Select :model-value="form.dataset_permission" @update:model-value="v => form.dataset_permission = v">
                  <SelectTrigger><SelectValue :placeholder="t('select_placeholder')" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="me">{{ t('me_option') }}</SelectItem>
                    <SelectItem value="team">{{ t('team_option') }}</SelectItem>
                  </SelectContent>
                </Select>
                <p class="text-xs text-[var(--color-text-secondary)]">{{ t('dataset_permission_hint') }}</p>
              </div>
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('chunk_strategy') }}</label>
              <Select :model-value="form.dataset_chunk_method" @update:model-value="v => form.dataset_chunk_method = v">
                <SelectTrigger><SelectValue :placeholder="t('select_placeholder')" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="naive">naive</SelectItem>
                  <SelectItem value="qa">qa</SelectItem>
                  <SelectItem value="table">table</SelectItem>
                  <SelectItem value="paper">paper</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </template>

          <!-- Weknora fields -->
          <template v-else-if="form.provider === 'weknora'">
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Base URL</label>
              <Input v-model="form.base_url" :placeholder="DEFAULT_WEKNORA_BASE_URL" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
              <Input v-model="form.api_key" type="password" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('threshold') }}</label>
              <NumberInput v-model="form.score_threshold" :min="0" :max="1" :step="0.01" :precision="2" />
            </div>
            <!-- Model list refresh -->
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model_list_label') }}</label>
              <div class="flex items-center gap-2 flex-wrap">
                <Button size="sm" variant="outline" :disabled="weknoraModelLoading" @click="fetchWeknoraModels(true, false)">
                  {{ t('refresh_models') }}
                </Button>
                <span v-if="weknoraModelLoading" class="text-xs text-[var(--color-text-secondary)]">{{ t('models_loading') }}</span>
                <span v-else-if="weknoraModelLoadError" class="text-xs text-red-500">{{ weknoraModelLoadError }}</span>
                <span v-else class="text-xs text-[var(--color-text-secondary)]">{{ t('auto_pull_models_hint') }}</span>
              </div>
            </div>
            <!-- Embedding model -->
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('embedding_model_label') }}</label>
              <div>
                <Input v-model="form.embedding_model_id" list="weknora-embedding-list" :placeholder="t('kb_required_ph')" />
                <datalist id="weknora-embedding-list">
                  <option v-for="item in weknoraEmbeddingModels" :key="item.id" :value="item.id">{{ item.name }}</option>
                </datalist>
              </div>
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('chunk_size_label') }}</label>
              <NumberInput v-model="form.chunk_size" :min="1" :step="100" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('chunk_overlap_label') }}</label>
              <NumberInput v-model="form.chunk_overlap" :min="0" :step="50" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-start">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('separator_label') }}</label>
              <div class="grid gap-1.5">
                <Input v-model="form.separators_raw" :placeholder="t('separator_ph')" />
                <p class="text-xs text-[var(--color-text-secondary)]">{{ t('save_as_separators') }}</p>
              </div>
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('multimodal_label') }}</label>
              <Switch :checked="form.enable_multimodal" @update:checked="v => form.enable_multimodal = v" />
            </div>
            <!-- Summary model -->
            <div class="grid grid-cols-[140px_1fr] gap-3 items-start">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('summary_model_label') }}</label>
              <div class="grid gap-1.5">
                <Input v-model="form.summary_model_id" list="weknora-llm-list" :placeholder="t('kb_optional_ph')" />
                <datalist id="weknora-llm-list">
                  <option v-for="item in weknoraLLMModels" :key="item.id" :value="item.id">{{ item.name }}</option>
                </datalist>
                <p class="text-xs text-[var(--color-text-secondary)]">{{ t('summary_model_desc') }}</p>
              </div>
            </div>
            <!-- Rerank model -->
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('rerank_model_label') }}</label>
              <div>
                <Input v-model="form.rerank_model_id" list="weknora-rerank-list" :placeholder="t('kb_optional_ph')" />
                <datalist id="weknora-rerank-list">
                  <option v-for="item in weknoraRerankModels" :key="item.id" :value="item.id">{{ item.name }}</option>
                </datalist>
              </div>
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('vlm_model_label') }}</label>
              <Input v-model="form.vlm_model_id" :placeholder="t('optional_label')" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('parse_poll_interval_label') }}</label>
              <NumberInput v-model="form.parse_poll_interval_ms" :min="100" :step="100" />
            </div>
            <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
              <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('parse_timeout_label') }}</label>
              <NumberInput v-model="form.parse_timeout_ms" :min="1000" :step="1000" />
            </div>
          </template>

          <!-- Common bottom switches -->
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('enabled') }}</label>
            <Switch :checked="form.enabled" @update:checked="v => form.enabled = v" />
          </div>
          <div class="grid grid-cols-[140px_1fr] gap-3 items-center">
            <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('default') }}</label>
            <Switch :checked="form.is_default" @update:checked="v => form.is_default = v" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="dialogVisible = false">{{ t('cancel') }}</Button>
          <Button @click="submit">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
