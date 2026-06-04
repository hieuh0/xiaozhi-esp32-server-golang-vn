<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button type="primary" @click="openDialog()">{{ t('add_config') }}添加配置</el-button>
    </div>

    <el-table :data="items" v-loading="loading" style="width: 100%">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="provider" :label="t('provider')" width="120" />
      <el-table-column prop="name" :label="t('name')" width="160" />
      <el-table-column prop="config_id" :label="t('config_id')" width="170" />
      <el-table-column :label="t('config_summary')">
        <template #default="scope">{{ getConfigSummary(scope.row) }}</template>
      </el-table-column>
      <el-table-column label="启用" width="80">
        <template #default="scope">
          <el-switch
            v-model="scope.row.enabled"
            :loading="isRowSwitchLoading(scope.row.id, 'enabled')"
            @change="(val) => onRowSwitchChange(scope.row, 'enabled', val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="默认" width="80">
        <template #default="scope">
          <el-switch
            v-model="scope.row.is_default"
            :loading="isRowSwitchLoading(scope.row.id, 'is_default')"
            :disabled="!scope.row.enabled && !scope.row.is_default"
            @change="(val) => onRowSwitchChange(scope.row, 'is_default', val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="170">
        <template #default="scope">
          <el-button size="small" @click="openDialog(scope.row)">编辑</el-button>
          <el-button size="small" type="danger" @click="remove(scope.row.id)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑配置' : '新增配置'" width="700px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="提供商">
          <el-select v-model="form.provider" style="width: 100%" @change="onProviderChange">
            <el-option value="dify" label="dify" />
            <el-option value="ragflow" label="ragflow" />
            <el-option value="weknora" label="weknora" />
          </el-select>
        </el-form-item>
        <el-form-item label="提供商官网">
          <a
            :href="getProviderWebsite(form.provider)"
            target="_blank"
            rel="noopener noreferrer"
            style="color: var(--apple-primary); text-decoration:none;"
          >
            {{ getProviderWebsite(form.provider) }}
          </a>
        </el-form-item>
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="配置ID"><el-input v-model="form.config_id" /></el-form-item>
        <template v-if="form.provider === 'dify'">
          <el-form-item label="Base URL"><el-input v-model="form.base_url" :placeholder="DEFAULT_DIFY_BASE_URL" /></el-form-item>
          <el-form-item label="API Key"><el-input v-model="form.api_key" type="password" show-password /></el-form-item>
          <el-form-item label="阈值"><el-input-number v-model="form.score_threshold" :min="0" :max="1" :step="0.01" :precision="2" style="width:100%" /></el-form-item>
          <el-form-item label="Dataset权限">
            <el-select v-model="form.dataset_permission" style="width: 100%" placeholder="请选择">
              <el-option value="only_me" label="only_me（仅自己可见）" />
              <el-option value="all_team_members" label="all_team_members（团队可见）" />
              <el-option value="partial_members" label="partial_members（部分成员可见）" />
            </el-select>
            <div class="form-helper-text">
              控制外部知识库平台中该 dataset 的可见范围，不影响本系统用户权限。
            </div>
          </el-form-item>
          <el-form-item label="Dataset提供方"><el-input v-model="form.dataset_provider" placeholder="vendor" /></el-form-item>
          <el-form-item label="索引策略">
            <el-select v-model="form.dataset_indexing_technique" style="width: 100%" placeholder="请选择">
              <el-option value="high_quality" label="high_quality（高质量）" />
              <el-option value="economy" label="economy（经济）" />
            </el-select>
          </el-form-item>
        </template>
        <template v-else-if="form.provider === 'ragflow'">
          <el-form-item label="Base URL"><el-input v-model="form.base_url" :placeholder="DEFAULT_RAGFLOW_BASE_URL" /></el-form-item>
          <el-form-item label="API Key"><el-input v-model="form.api_key" type="password" show-password /></el-form-item>
          <el-form-item label="相似度阈值"><el-input-number v-model="form.similarity_threshold" :min="0" :max="1" :step="0.01" :precision="2" style="width:100%" /></el-form-item>
          <el-form-item label="向量权重"><el-input-number v-model="form.vector_similarity_weight" :min="0" :max="1" :step="0.01" :precision="2" style="width:100%" /></el-form-item>
          <el-form-item label="启用关键词"><el-switch v-model="form.keyword" /></el-form-item>
          <el-form-item label="启用高亮"><el-switch v-model="form.highlight" /></el-form-item>
          <el-form-item label="Dataset权限">
            <el-select v-model="form.dataset_permission" style="width: 100%" placeholder="请选择">
              <el-option value="me" label="me（仅自己可见）" />
              <el-option value="team" label="team（团队可见）" />
            </el-select>
            <div class="form-helper-text">
              控制外部知识库平台中该 dataset 的可见范围，不影响本系统用户权限。
            </div>
          </el-form-item>
          <el-form-item label="分块策略">
            <el-select v-model="form.dataset_chunk_method" style="width: 100%" placeholder="请选择">
              <el-option value="naive" label="naive" />
              <el-option value="qa" label="qa" />
              <el-option value="table" label="table" />
              <el-option value="paper" label="paper" />
            </el-select>
          </el-form-item>
        </template>
        <template v-else-if="form.provider === 'weknora'">
          <el-form-item label="Base URL"><el-input v-model="form.base_url" :placeholder="DEFAULT_WEKNORA_BASE_URL" /></el-form-item>
          <el-form-item label="API Key"><el-input v-model="form.api_key" type="password" show-password /></el-form-item>
          <el-form-item label="阈值"><el-input-number v-model="form.score_threshold" :min="0" :max="1" :step="0.01" :precision="2" style="width:100%" /></el-form-item>
          <el-form-item label="模型列表">
            <div class="inline-helper-row">
              <el-button size="small" :loading="weknoraModelLoading" @click="fetchWeknoraModels(true, false)">刷新模型</el-button>
              <span v-if="weknoraModelLoading" class="inline-helper-text">正在拉取模型列表...</span>
              <span v-else-if="weknoraModelLoadError" class="inline-helper-text is-error">{{ weknoraModelLoadError }}</span>
              <span v-else class="inline-helper-text">自动拉取 embedding/llm/rerank 模型；也可手动输入。</span>
            </div>
          </el-form-item>
          <el-form-item label="Embedding模型">
            <el-select
              v-model="form.embedding_model_id"
              filterable
              allow-create
              default-first-option
              clearable
              style="width:100%;"
              placeholder="必填：请选择或手动输入"
            >
              <el-option
                v-for="item in weknoraEmbeddingModels"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="分块大小"><el-input-number v-model="form.chunk_size" :min="1" :step="100" style="width:100%" /></el-form-item>
          <el-form-item label="重叠大小"><el-input-number v-model="form.chunk_overlap" :min="0" :step="50" style="width:100%" /></el-form-item>
          <el-form-item label="分隔符">
            <el-input v-model="form.separators_raw" placeholder="逗号分隔，如 \n\n,\n,。,！,？,;,；" />
            <div class="form-helper-text">
              保存时将按逗号拆分为 separators 数组。
            </div>
          </el-form-item>
          <el-form-item label="多模态"><el-switch v-model="form.enable_multimodal" /></el-form-item>
          <el-form-item label="Summary模型">
            <el-select
              v-model="form.summary_model_id"
              filterable
              allow-create
              default-first-option
              clearable
              style="width:100%;"
              placeholder="可选：请选择或手动输入"
            >
              <el-option
                v-for="item in weknoraLLMModels"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </el-select>
            <div class="form-helper-text">
              用于知识摘要生成；不配置则跳过摘要步骤。
            </div>
          </el-form-item>
          <el-form-item label="Rerank模型">
            <el-select
              v-model="form.rerank_model_id"
              filterable
              allow-create
              default-first-option
              clearable
              style="width:100%;"
              placeholder="可选：请选择或手动输入"
            >
              <el-option
                v-for="item in weknoraRerankModels"
                :key="item.id"
                :label="item.name"
                :value="item.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="VLM模型ID"><el-input v-model="form.vlm_model_id" placeholder="可选" /></el-form-item>
          <el-form-item label="轮询间隔ms"><el-input-number v-model="form.parse_poll_interval_ms" :min="100" :step="100" style="width:100%" /></el-form-item>
          <el-form-item label="解析超时ms"><el-input-number v-model="form.parse_timeout_ms" :min="1000" :step="1000" style="width:100%" /></el-form-item>
        </template>
        <el-form-item label="启用"><el-switch v-model="form.enabled" /></el-form-item>
        <el-form-item label="默认"><el-switch v-model="form.is_default" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const items = ref([])
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
  name: '',
  config_id: '',
  provider: 'dify',
  base_url: DEFAULT_DIFY_BASE_URL,
  api_key: '',
  score_threshold: DEFAULT_DIFY_SCORE_THRESHOLD,
  dataset_permission: '',
  dataset_provider: '',
  dataset_indexing_technique: '',
  similarity_threshold: DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD,
  vector_similarity_weight: 0.3,
  keyword: false,
  highlight: false,
  dataset_chunk_method: '',
  embedding_model_id: '',
  chunk_size: DEFAULT_WEKNORA_CHUNK_SIZE,
  chunk_overlap: DEFAULT_WEKNORA_CHUNK_OVERLAP,
  separators_raw: DEFAULT_WEKNORA_SEPARATORS.join(','),
  enable_multimodal: true,
  summary_model_id: '',
  rerank_model_id: '',
  vlm_model_id: '',
  parse_poll_interval_ms: DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS,
  parse_timeout_ms: DEFAULT_WEKNORA_PARSE_TIMEOUT_MS,
  enabled: true,
  is_default: false
})

const normalizeProvider = (provider) => {
  const p = String(provider || '').trim().toLowerCase()
  if (p === 'dify' || p === 'ragflow' || p === 'weknora') {
    return p
  }
  return 'dify'
}

const PROVIDER_WEBSITE = {
  dify: 'https://dify.ai/',
  ragflow: 'https://github.com/infiniflow/ragflow',
  weknora: 'https://github.com/Tencent/WeKnora'
}

const getProviderWebsite = (provider) => {
  const key = normalizeProvider(provider)
  return PROVIDER_WEBSITE[key] || PROVIDER_WEBSITE.dify
}

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
    ret.push({
      id,
      name: String(item?.name || id).trim() || id,
      type: String(item?.type || '').trim(),
      provider: String(item?.provider || '').trim()
    })
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
    if (!silent) {
      ElMessage.warning(t('fill_weknora_url_apikey'))
    }
    return
  }

  const fetchKey = `${baseURL}::${apiKey}`
  if (
    !force &&
    fetchKey === lastWeknoraFetchKey.value &&
    weknoraEmbeddingModels.value.length + weknoraLLMModels.value.length + weknoraRerankModels.value.length > 0
  ) {
    return
  }

  const seq = ++weknoraFetchSeq
  weknoraModelLoading.value = true
  weknoraModelLoadError.value = ''
  try {
    const res = await api.post('/admin/knowledge-search-configs/weknora/models', {
      base_url: baseURL,
      api_key: apiKey
    })
    if (seq !== weknoraFetchSeq) return

    const data = res?.data?.data || {}
    const embedding = normalizeModelOptions(data.embedding_models)
    const llm = normalizeModelOptions(data.llm_models)
    const rerank = normalizeModelOptions(data.rerank_models)
    weknoraEmbeddingModels.value = embedding
    weknoraLLMModels.value = llm
    weknoraRerankModels.value = rerank
    lastWeknoraFetchKey.value = fetchKey

    if (!String(form.embedding_model_id || '').trim() && embedding.length > 0) {
      form.embedding_model_id = embedding[0].id
    }
    if (!String(form.summary_model_id || '').trim() && llm.length > 0) {
      form.summary_model_id = llm[0].id
    }
  } catch (e) {
    if (seq !== weknoraFetchSeq) return
    const msg = e?.response?.data?.error || '拉取 WeKnora 模型列表失败'
    weknoraModelLoadError.value = msg
    if (!silent) {
      ElMessage.error(msg)
    }
  } finally {
    if (seq === weknoraFetchSeq) {
      weknoraModelLoading.value = false
    }
  }
}

const scheduleWeknoraModelFetch = (force = false, silent = true) => {
  if (weknoraModelFetchTimer) {
    clearTimeout(weknoraModelFetchTimer)
    weknoraModelFetchTimer = null
  }
  if (normalizeProvider(form.provider) !== 'weknora') return
  const baseURL = String(form.base_url || '').trim()
  const apiKey = String(form.api_key || '').trim()
  if (!baseURL || !apiKey) return

  weknoraModelFetchTimer = setTimeout(() => {
    fetchWeknoraModels(force, silent)
  }, 450)
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await api.get('/admin/knowledge-search-configs')
    items.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

const applyProviderDefaults = (provider, force = false) => {
  provider = normalizeProvider(provider)
  if (provider === 'dify') {
    if (force || !form.base_url || form.base_url === DEFAULT_RAGFLOW_BASE_URL || form.base_url === DEFAULT_WEKNORA_BASE_URL) {
      form.base_url = DEFAULT_DIFY_BASE_URL
    }
    if (force || Number.isNaN(Number(form.score_threshold))) {
      form.score_threshold = DEFAULT_DIFY_SCORE_THRESHOLD
    }
    return
  }
  if (provider === 'ragflow') {
    if (force || !form.base_url || form.base_url === DEFAULT_DIFY_BASE_URL || form.base_url === DEFAULT_WEKNORA_BASE_URL) {
      form.base_url = DEFAULT_RAGFLOW_BASE_URL
    }
    if (force || Number.isNaN(Number(form.similarity_threshold))) {
      form.similarity_threshold = DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD
    }
    return
  }
  if (provider === 'weknora') {
    if (force || !form.base_url || form.base_url === DEFAULT_DIFY_BASE_URL || form.base_url === DEFAULT_RAGFLOW_BASE_URL) {
      form.base_url = DEFAULT_WEKNORA_BASE_URL
    }
    if (force || Number.isNaN(Number(form.score_threshold))) {
      form.score_threshold = DEFAULT_WEKNORA_SCORE_THRESHOLD
    }
    if (force || Number.isNaN(Number(form.chunk_size)) || Number(form.chunk_size) <= 0) {
      form.chunk_size = DEFAULT_WEKNORA_CHUNK_SIZE
    }
    if (force || Number.isNaN(Number(form.chunk_overlap)) || Number(form.chunk_overlap) < 0) {
      form.chunk_overlap = DEFAULT_WEKNORA_CHUNK_OVERLAP
    }
    if (force || !String(form.separators_raw || '').trim()) {
      form.separators_raw = DEFAULT_WEKNORA_SEPARATORS.join(',')
    }
    if (force || Number.isNaN(Number(form.parse_poll_interval_ms)) || Number(form.parse_poll_interval_ms) <= 0) {
      form.parse_poll_interval_ms = DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS
    }
    if (force || Number.isNaN(Number(form.parse_timeout_ms)) || Number(form.parse_timeout_ms) <= 0) {
      form.parse_timeout_ms = DEFAULT_WEKNORA_PARSE_TIMEOUT_MS
    }
  }
}

const onProviderChange = (provider) => {
  applyProviderDefaults(provider, true)
  if (normalizeProvider(provider) === 'weknora') {
    scheduleWeknoraModelFetch(true, true)
  } else {
    clearWeknoraModels()
  }
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
  if (!row) {
    applyProviderDefaults(provider, true)
    clearWeknoraModels()
  } else if (provider !== 'weknora') {
    clearWeknoraModels()
  }
  if (provider === 'weknora') {
    scheduleWeknoraModelFetch(false, true)
  }
  dialogVisible.value = true
}

const submit = async () => {
  if (form.provider === 'weknora' && !String(form.embedding_model_id || '').trim()) {
    ElMessage.error(t('embedding_model_id_required'))
    return
  }
  const weknoraSeparators = parseSeparators(form.separators_raw)
  const payload = {
    type: 'knowledge_search',
    name: form.name,
    config_id: form.config_id,
    provider: form.provider,
    enabled: form.enabled,
    is_default: form.is_default,
    json_data: JSON.stringify(form.provider === 'dify'
      ? {
          base_url: form.base_url,
          api_key: form.api_key,
          score_threshold: form.score_threshold,
          dataset_permission: form.dataset_permission,
          dataset_provider: form.dataset_provider,
          dataset_indexing_technique: form.dataset_indexing_technique
        }
      : form.provider === 'ragflow'
        ? {
            base_url: form.base_url,
            api_key: form.api_key,
            similarity_threshold: form.similarity_threshold,
            vector_similarity_weight: form.vector_similarity_weight,
            keyword: form.keyword,
            highlight: form.highlight,
            dataset_permission: form.dataset_permission,
            dataset_chunk_method: form.dataset_chunk_method
          }
        : {
            base_url: form.base_url,
            api_key: form.api_key,
            score_threshold: form.score_threshold,
            embedding_model_id: String(form.embedding_model_id || '').trim(),
            chunk_size: Number(form.chunk_size) || DEFAULT_WEKNORA_CHUNK_SIZE,
            chunk_overlap: Number(form.chunk_overlap) || DEFAULT_WEKNORA_CHUNK_OVERLAP,
            separators: weknoraSeparators,
            enable_multimodal: !!form.enable_multimodal,
            summary_model_id: String(form.summary_model_id || '').trim(),
            rerank_model_id: String(form.rerank_model_id || '').trim(),
            vlm_model_id: String(form.vlm_model_id || '').trim(),
            parse_poll_interval_ms: Number(form.parse_poll_interval_ms) || DEFAULT_WEKNORA_PARSE_POLL_INTERVAL_MS,
            parse_timeout_ms: Number(form.parse_timeout_ms) || DEFAULT_WEKNORA_PARSE_TIMEOUT_MS
          })
  }
  try {
    if (editing.value) {
      await api.put(`/admin/knowledge-search-configs/${currentId.value}`, payload)
    } else {
      await api.post('/admin/knowledge-search-configs', payload)
    }
    ElMessage.success(t('save_success'))
    dialogVisible.value = false
    await loadData()
  } catch (e) {
    ElMessage.error(t('save_failed'))
  }
}

const isRowSwitchLoading = (id, field) => {
  return !!rowSwitchLoading.value[`${field}_${id}`]
}

const setRowSwitchLoading = (id, field, loading) => {
  rowSwitchLoading.value = {
    ...rowSwitchLoading.value,
    [`${field}_${id}`]: loading
  }
}

const onRowSwitchChange = async (row, field, value) => {
  const id = row?.id
  if (!id || (field !== 'enabled' && field !== 'is_default')) return
  if (isRowSwitchLoading(id, field)) return

  setRowSwitchLoading(id, field, true)
  try {
    const provider = normalizeProvider(row.provider)
    const rawData = row?.json_data ? JSON.parse(row.json_data || '{}') : {}
    let enabled = field === 'enabled' ? !!value : !!row.enabled
    let isDefault = field === 'is_default' ? !!value : !!row.is_default

    if (!enabled && isDefault) {
      isDefault = false
    }
    if (isDefault && !enabled) {
      enabled = true
    }

    const payload = {
      type: 'knowledge_search',
      name: row.name,
      config_id: row.config_id,
      provider,
      enabled,
      is_default: isDefault,
      json_data: JSON.stringify(rawData)
    }
    await api.put(`/admin/knowledge-search-configs/${id}`, payload)

    if (isDefault) {
      items.value.forEach((item) => {
        if (item.id !== id) item.is_default = false
      })
    }
    row.enabled = enabled
    row.is_default = isDefault
    ElMessage.success(t('update_success'))
    await loadData()
  } catch (e) {
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
  if (provider === 'dify') {
    return `base_url: ${data.base_url || DEFAULT_DIFY_BASE_URL}; score_threshold: ${data.score_threshold ?? DEFAULT_DIFY_SCORE_THRESHOLD}`
  }
  if (provider === 'ragflow') {
    return `base_url: ${data.base_url || DEFAULT_RAGFLOW_BASE_URL}; similarity_threshold: ${data.similarity_threshold ?? DEFAULT_RAGFLOW_SIMILARITY_THRESHOLD}`
  }
  if (provider === 'weknora') {
    return `base_url: ${data.base_url || DEFAULT_WEKNORA_BASE_URL}; score_threshold: ${data.score_threshold ?? DEFAULT_WEKNORA_SCORE_THRESHOLD}`
  }
  return '-'
}

onMounted(loadData)

watch(
  () => [normalizeProvider(form.provider), String(form.base_url || '').trim(), String(form.api_key || '').trim()],
  ([provider]) => {
    if (provider !== 'weknora') {
      clearWeknoraModels()
      return
    }
    scheduleWeknoraModelFetch(false, true)
  }
)

onBeforeUnmount(() => {
  if (weknoraModelFetchTimer) {
    clearTimeout(weknoraModelFetchTimer)
    weknoraModelFetchTimer = null
  }
})
</script>

<style scoped>
.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.form-helper-text {
  margin-top: 6px;
  color: var(--apple-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.inline-helper-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  flex-wrap: wrap;
}

.inline-helper-text {
  color: var(--apple-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.inline-helper-text.is-error {
  color: var(--apple-danger);
}
</style>
