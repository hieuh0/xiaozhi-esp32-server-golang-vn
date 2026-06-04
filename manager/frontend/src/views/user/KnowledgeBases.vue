<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button type="primary" @click="openDialog()">{{ t('add_knowledge_base') }}新增知识库</el-button>
    </div>

    <el-table :data="items" v-loading="loading" stripe table-layout="fixed" style="width: 100%">
      <el-table-column prop="id" label="ID" width="56" />
      <el-table-column prop="name" :label="t('name')" width="124" show-overflow-tooltip />
      <el-table-column :label="t('description')" min-width="180" show-overflow-tooltip>
        <template #default="scope">
          <span class="kb-desc-text" :class="{ 'is-empty': !(scope.row.description || '').trim() }">
            {{ (scope.row.description || '').trim() || '-' }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="提供商" width="88" show-overflow-tooltip>
        <template #default="scope">
          <el-tag size="small" effect="plain">{{ formatProviderText(scope.row.sync_provider) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="文档数" width="72" align="center">
        <template #default="scope">
          <el-tag size="small" type="info">{{ formatDocCount(scope.row.doc_count) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="同步状态" width="132">
        <template #default="scope">
          <div class="kb-sync-status-cell">
            <el-tag :type="getSyncStatusTagType(scope.row.sync_status)" size="small">{{ getSyncStatusText(scope.row.sync_status) }}</el-tag>
            <el-tooltip v-if="shouldShowSyncErrorTip(scope.row)" placement="top">
              <template #content>
                <div class="kb-sync-error-tooltip">{{ scope.row.sync_error }}</div>
              </template>
              <el-icon class="kb-sync-error-icon"><WarningFilled /></el-icon>
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="最近同步" width="168" show-overflow-tooltip>
        <template #default="scope">
          <span>{{ formatDateTimeCell(scope.row.last_synced_at) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="92" align="center">
        <template #default="scope">
          <el-switch
            :model-value="String(scope.row.status || '').trim() === 'active'"
            inline-prompt
            active-text="开"
            inactive-text="关"
            :loading="isStatusSwitchLoading(scope.row.id)"
            @change="(checked) => toggleKnowledgeBaseStatus(scope.row, checked)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="176">
        <template #default="scope">
          <div class="action-buttons">
            <el-button size="small" type="primary" plain @click="openDocuments(scope.row)">文档</el-button>
            <el-button size="small" type="success" plain @click="openSearchTestDialog(scope.row)">测试</el-button>
            <el-dropdown trigger="click" @command="(cmd) => handleKnowledgeBaseAction(cmd, scope.row)">
              <el-button size="small">更多</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">编辑</el-dropdown-item>
                  <el-dropdown-item command="sync">重试同步</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑知识库' : '新增知识库'" width="680px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="名称">
          <el-input v-model="form.name" maxlength="100" show-word-limit />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" />
        </el-form-item>
        <el-form-item label="同步说明">
          <div class="kb-helper-text">保存后会自动异步同步到管理员配置的知识库提供商（如 Dify / RAGFlow / WeKnora）。文档请在“文档管理”中新增。</div>
        </el-form-item>
        <el-form-item label="检索阈值">
          <el-input
            v-model="form.retrieval_threshold_text"
            placeholder="请输入 0~1 之间的小数，如 0.2"
            clearable
          />
          <div class="kb-helper-text is-spaced">
            默认填充提供商全局阈值。当前提供商：{{ form.threshold_provider || '-' }}，全局阈值：{{ formatKnowledgeThreshold(form.global_threshold) }}。
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width: 100%">
            <el-option value="active" label="active" />
            <el-option value="inactive" label="inactive" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="documentsVisible" title="文档管理" width="900px">
      <div class="dialog-toolbar">
        <div>
          当前知识库: <strong>{{ currentKb?.name || '-' }}</strong>
        </div>
        <div class="dialog-toolbar-actions">
          <el-upload
            :show-file-list="false"
            :http-request="uploadDocumentFile"
            :accept="uploadAcceptByProvider"
            :disabled="!isUploadProviderSupported"
          >
            <el-button type="success" plain>上传文件</el-button>
          </el-upload>
          <el-button type="primary" @click="openDocumentDialog()">新增文档</el-button>
        </div>
      </div>
      <div class="kb-helper-text is-bottom">
        {{ uploadTipText }}
      </div>
      <el-table :data="documentItems" v-loading="documentsLoading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="文档名" width="180" />
        <el-table-column prop="external_doc_id" label="Document ID" width="220" />
        <el-table-column label="内容预览">
          <template #default="scope">
            {{ getDocumentPreview(scope.row) }}
          </template>
        </el-table-column>
        <el-table-column label="同步状态" width="110">
          <template #default="scope">
            <el-tag :type="getSyncStatusTagType(scope.row.sync_status)">{{ getSyncStatusText(scope.row.sync_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_synced_at" label="最近同步时间" width="170" />
        <el-table-column label="操作" width="250">
          <template #default="scope">
            <div class="action-buttons">
              <el-button size="small" :disabled="isUploadedFileDocument(scope.row)" @click="openDocumentDialog(scope.row)">编辑</el-button>
              <el-button size="small" type="primary" plain @click="syncDocument(scope.row.id)">重试同步</el-button>
              <el-button size="small" type="danger" @click="removeDocument(scope.row.id)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="documentDialogVisible" :title="documentEditing ? '编辑文档' : '新增文档'" width="700px">
      <el-form :model="documentForm" label-width="90px">
        <el-form-item label="文档名">
          <el-input v-model="documentForm.name" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="documentForm.content" type="textarea" :rows="12" placeholder="请输入文档内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="documentDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitDocument">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="searchTestVisible" title="召回测试" width="960px">
      <div style="display: flex; justify-content: space-between; gap: 12px; margin-bottom: 12px; flex-wrap: wrap;">
        <div>
          当前知识库: <strong>{{ searchTestKb?.name || '-' }}</strong>
          <el-tag size="small" style="margin-left: 8px;">{{ searchTestKb?.sync_provider || '-' }}</el-tag>
        </div>
        <div style="display: flex; gap: 8px; flex: 1; min-width: 420px; justify-content: flex-end;">
          <el-input
            v-model="searchTestForm.query"
            placeholder="输入测试关键词或问题，如：退款流程/接口鉴权"
            clearable
            @keyup.enter="runSearchTest"
          />
          <el-tooltip content="TopK：返回前 K 条召回结果" placement="top">
            <span style="display:inline-flex;align-items:center;color:#909399;font-size:12px;white-space:nowrap;">TopK</span>
          </el-tooltip>
          <el-select v-model="searchTestForm.top_k" style="width: 110px;">
            <el-option v-for="k in topKOptions" :key="k" :value="k" :label="String(k)" />
          </el-select>
          <el-tooltip content="仅本次召回测试生效；为空则使用知识库当前阈值（或全局阈值）" placement="top">
            <span style="display:inline-flex;align-items:center;color:#909399;font-size:12px;white-space:nowrap;">阈值</span>
          </el-tooltip>
          <el-input
            v-model="searchTestForm.threshold_text"
            placeholder="如 0.2"
            clearable
            style="width: 120px;"
          />
          <el-button type="primary" :loading="searchTestLoading" @click="runSearchTest">开始测试</el-button>
        </div>
      </div>
      <div class="kb-helper-text is-bottom">
        召回测试会直接调用当前知识库对应 provider 的检索接口（Dify / RAGFlow / WeKnora），用于验证关键词召回效果。
      </div>
      <div v-if="searchTestElapsedMs !== null" class="kb-helper-text is-bottom is-regular">
        响应耗时：{{ searchTestElapsedMs }} ms
      </div>
      <el-table :data="searchTestResult.hits" v-loading="searchTestLoading" style="width: 100%" max-height="420">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="title" label="来源" width="200" />
        <el-table-column label="分数" width="110">
          <template #default="scope">
            {{ formatHitScore(scope.row.score) }}
          </template>
        </el-table-column>
        <el-table-column prop="content" label="命中内容" min-width="480">
          <template #default="scope">
            <div class="search-hit-content">
              {{ scope.row.content }}
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!searchTestLoading && hasRunSearchTest && searchTestResult.hits.length === 0" class="kb-helper-text is-empty">
        未命中内容，可尝试更换关键词或检查该知识库是否已同步完成。
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { WarningFilled } from '@element-plus/icons-vue'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const items = ref([])
const statusSwitchLoadingMap = ref({})
const dialogVisible = ref(false)
const editing = ref(false)
const currentId = ref(null)

const documentsVisible = ref(false)
const documentsLoading = ref(false)
const documentItems = ref([])
const currentKb = ref(null)

const documentDialogVisible = ref(false)
const documentEditing = ref(false)
const currentDocumentId = ref(null)
const searchTestVisible = ref(false)
const searchTestLoading = ref(false)
const searchTestKb = ref(null)
const hasRunSearchTest = ref(false)
const searchTestElapsedMs = ref(null)
const searchTestResult = reactive({
  query: '',
  count: 0,
  hits: []
})

const form = reactive({
  name: '',
  description: '',
  status: 'active',
  inherit_global_threshold: true,
  retrieval_threshold_text: '0.2',
  threshold_provider: 'dify',
  global_threshold: 0.2
})

const documentForm = reactive({
  name: '',
  content: ''
})
const searchTestForm = reactive({
  query: '',
  top_k: 5,
  threshold_text: ''
})
const topKOptions = Array.from({ length: 20 }, (_, i) => i + 1)

const FILE_UPLOAD_CONTENT_PREFIX = '__KB_FILE_UPLOAD_V1__:'
const DIFY_UPLOAD_ACCEPT = '.txt,.md,.markdown,.pdf,.html,.htm,.xlsx,.xls,.docx,.csv,.eml,.msg,.pptx,.ppt,.xml,.epub'
const RAGFLOW_UPLOAD_ACCEPT = '.txt,.text,.md,.markdown,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.wps,.json,.csv,.log,.xml,.html,.htm,.yml,.yaml,.rtf,.sql,.ini,.jpg,.jpeg,.png,.gif,.bmp,.webp,.tif,.tiff,.eml,.msg'
const WEKNORA_UPLOAD_ACCEPT = '.txt,.text,.md,.markdown,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.wps,.json,.csv,.log,.xml,.html,.htm,.yml,.yaml,.rtf,.sql,.ini,.jpg,.jpeg,.png,.gif,.bmp,.webp,.tif,.tiff,.eml,.msg'
const DEFAULT_DIFY_THRESHOLD = 0.2
const DEFAULT_RAGFLOW_THRESHOLD = 0.2
const DEFAULT_WEKNORA_THRESHOLD = 0.2

const knowledgeGlobalConfig = reactive({
  default_provider: 'dify',
  providers: {}
})

const currentKBProvider = computed(() => (currentKb.value?.sync_provider || 'dify').toLowerCase())
const uploadAcceptByProvider = computed(() => {
  if (currentKBProvider.value === 'dify') return DIFY_UPLOAD_ACCEPT
  if (currentKBProvider.value === 'ragflow') return RAGFLOW_UPLOAD_ACCEPT
  if (currentKBProvider.value === 'weknora') return WEKNORA_UPLOAD_ACCEPT
  return ''
})
const isUploadProviderSupported = computed(() => currentKBProvider.value === 'dify' || currentKBProvider.value === 'ragflow' || currentKBProvider.value === 'weknora')
const uploadTipText = computed(() => {
  if (currentKBProvider.value === 'dify') {
    return '按 Dify 支持格式限制上传（txt/md/pdf/html/xlsx/docx/csv/eml/msg/pptx/xml/epub），上传后自动创建文档并异步同步。'
  }
  if (currentKBProvider.value === 'ragflow') {
    return '按 RAGFlow 支持格式限制上传（如 txt/md/pdf/docx/xlsx/pptx/jpg/png/eml 等），上传后自动创建文档并异步同步。'
  }
  if (currentKBProvider.value === 'weknora') {
    return '按 WeKnora 支持格式限制上传（如 txt/md/pdf/docx/xlsx/pptx/jpg/png/eml 等），上传后自动创建文档并异步同步。'
  }
  return `当前提供商 ${currentKBProvider.value} 暂不支持上传建文档。`
})

const applyKnowledgeGlobalConfig = (knowledge) => {
  const payload = knowledge && typeof knowledge === 'object' ? knowledge : {}
  knowledgeGlobalConfig.default_provider = normalizeProvider(payload.default_provider || 'dify')
  knowledgeGlobalConfig.providers = payload.providers && typeof payload.providers === 'object' ? payload.providers : {}
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await api.get('/user/knowledge-bases')
    items.value = res.data.data || []
    applyKnowledgeGlobalConfig(res.data.knowledge)
  } finally {
    loading.value = false
  }
}

const normalizeProvider = (provider) => {
  const p = String(provider || '').trim().toLowerCase()
  if (p === 'dify' || p === 'ragflow' || p === 'weknora') return p
  return 'dify'
}

const getGlobalThresholdByProvider = (provider) => {
  const p = normalizeProvider(provider)
  const cfg = knowledgeGlobalConfig.providers?.[p] || {}
  if (p === 'dify') {
    const v = Number(cfg.score_threshold)
    if (!Number.isNaN(v) && v >= 0 && v <= 1) return v
    return DEFAULT_DIFY_THRESHOLD
  }
  if (p === 'ragflow') {
    const v = Number(cfg.similarity_threshold)
    if (!Number.isNaN(v) && v >= 0 && v <= 1) return v
    return DEFAULT_RAGFLOW_THRESHOLD
  }
  if (p === 'weknora') {
    const v = Number(cfg.score_threshold)
    if (!Number.isNaN(v) && v >= 0 && v <= 1) return v
    return DEFAULT_WEKNORA_THRESHOLD
  }
  return DEFAULT_DIFY_THRESHOLD
}

const openDialog = (row = null) => {
  editing.value = !!row
  currentId.value = row?.id || null
  form.name = row?.name || ''
  form.description = row?.description || ''
  form.status = row?.status || 'active'
  const provider = normalizeProvider(row?.sync_provider || knowledgeGlobalConfig.default_provider || 'dify')
  const globalThreshold = getGlobalThresholdByProvider(provider)
  form.threshold_provider = provider
  form.global_threshold = globalThreshold
  if (row && row.retrieval_threshold !== null && row.retrieval_threshold !== undefined) {
    form.inherit_global_threshold = false
    form.retrieval_threshold_text = String(row.retrieval_threshold)
  } else {
    form.inherit_global_threshold = true
    form.retrieval_threshold_text = String(globalThreshold)
  }
  dialogVisible.value = true
}

const submit = async () => {
  if (!form.name.trim()) {
    ElMessage.error(t('name_required'))
    return
  }
  const rawThreshold = String(form.retrieval_threshold_text || '').trim()
  const threshold = Number(rawThreshold)
  if (!rawThreshold || Number.isNaN(threshold) || threshold < 0 || threshold > 1) {
    ElMessage.error(t('retrieval_threshold_range'))
    return
  }
  const globalThreshold = Number(form.global_threshold)
  const sameAsGlobal = !Number.isNaN(globalThreshold) && Math.abs(threshold - globalThreshold) < 0.000001
  if (form.inherit_global_threshold && !sameAsGlobal) {
    form.inherit_global_threshold = false
  }
  try {
    const useInheritGlobal = form.inherit_global_threshold && sameAsGlobal
    const payload = {
      name: form.name,
      description: form.description,
      status: form.status,
      inherit_global_threshold: useInheritGlobal,
      retrieval_threshold: useInheritGlobal ? null : threshold
    }
    let res = null
    if (editing.value) {
      res = await api.put(`/user/knowledge-bases/${currentId.value}`, payload)
    } else {
      res = await api.post('/user/knowledge-bases', payload)
    }
    ElMessage.success(t('save_success'))
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    dialogVisible.value = false
    await loadData()
  } catch (e) {
    ElMessage.error(t('save_failed'))
  }
}

const removeItem = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_knowledge_base'), t('hint'), { type: 'warning' })
    const res = await api.delete(`/user/knowledge-bases/${id}`)
    ElMessage.success(t('delete_success'))
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    await loadData()
  } catch {}
}

const isStatusSwitchLoading = (id) => !!statusSwitchLoadingMap.value?.[id]

const toggleKnowledgeBaseStatus = async (row, checked) => {
  if (!row?.id) return
  const id = row.id
  const prevStatus = String(row.status || 'inactive').trim() === 'active' ? 'active' : 'inactive'
  const nextStatus = checked ? 'active' : 'inactive'
  if (prevStatus === nextStatus) return
  if (isStatusSwitchLoading(id)) return

  statusSwitchLoadingMap.value = {
    ...statusSwitchLoadingMap.value,
    [id]: true
  }
  row.status = nextStatus

  try {
    const res = await api.put(`/user/knowledge-bases/${id}`, {
      name: row.name || '',
      description: row.description || '',
      content: row.content || '',
      status: nextStatus
    })
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    } else {
      ElMessage.success(`已${nextStatus === 'active' ? '启用' : '停用'}`)
    }
    await loadData()
  } catch (e) {
    row.status = prevStatus
    const msg = e?.response?.data?.error || '状态更新失败'
    ElMessage.error(msg)
  } finally {
    statusSwitchLoadingMap.value = {
      ...statusSwitchLoadingMap.value,
      [id]: false
    }
  }
}

const handleKnowledgeBaseAction = async (command, row) => {
  if (!row?.id) return
  if (command === 'edit') {
    openDialog(row)
    return
  }
  if (command === 'sync') {
    await syncItem(row.id)
    return
  }
  if (command === 'delete') {
    await removeItem(row.id)
  }
}

const syncItem = async (id) => {
  try {
    const res = await api.post(`/user/knowledge-bases/${id}/sync`)
    ElMessage.success(res?.data?.message || '同步任务已提交')
    await loadData()
  } catch (e) {
    const msg = e?.response?.data?.error || '同步失败'
    ElMessage.error(msg)
    await loadData()
  }
}

const openSearchTestDialog = (row) => {
  searchTestKb.value = row || null
  searchTestForm.query = ''
  searchTestForm.top_k = 5
  const provider = normalizeProvider(row?.sync_provider || knowledgeGlobalConfig.default_provider || 'dify')
  const globalThreshold = getGlobalThresholdByProvider(provider)
  const kbThreshold = row?.retrieval_threshold
  const effectiveThreshold = (kbThreshold !== null && kbThreshold !== undefined) ? Number(kbThreshold) : Number(globalThreshold)
  searchTestForm.threshold_text = Number.isNaN(effectiveThreshold) ? '' : String(effectiveThreshold)
  searchTestResult.query = ''
  searchTestResult.count = 0
  searchTestResult.hits = []
  searchTestElapsedMs.value = null
  hasRunSearchTest.value = false
  searchTestVisible.value = true
}

const runSearchTest = async () => {
  if (!searchTestKb.value?.id) {
    ElMessage.error(t('select_knowledge_base'))
    return
  }
  const query = (searchTestForm.query || '').trim()
  if (!query) {
    ElMessage.error(t('enter_test_keyword'))
    return
  }
  searchTestLoading.value = true
  const startedAt = Date.now()
  try {
    const rawThreshold = String(searchTestForm.threshold_text || '').trim()
    let threshold = null
    if (rawThreshold !== '') {
      const parsed = Number(rawThreshold)
      if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) {
        ElMessage.error(t('threshold_0_to_1'))
        return
      }
      threshold = parsed
    }
    const payload = {
      query,
      top_k: Number(searchTestForm.top_k) || 5,
      threshold
    }
    const res = await api.post(`/user/knowledge-bases/${searchTestKb.value.id}/test-search`, payload)
    const data = res?.data?.data || {}
    searchTestResult.query = data.query || query
    searchTestResult.count = Number(data.count || 0)
    searchTestResult.hits = Array.isArray(data.hits) ? data.hits : []
    const elapsed = Number(data.elapsed_ms)
    searchTestElapsedMs.value = Number.isNaN(elapsed) ? Date.now() - startedAt : elapsed
    hasRunSearchTest.value = true
    ElMessage.success(`召回完成，共返回 ${searchTestResult.count} 条`)
  } catch (e) {
    const msg = e?.response?.data?.error || '测试失败'
    ElMessage.error(msg)
  } finally {
    searchTestLoading.value = false
  }
}

const openDocuments = async (row) => {
  currentKb.value = row
  documentsVisible.value = true
  await loadDocuments()
}

const loadDocuments = async () => {
  if (!currentKb.value?.id) return
  documentsLoading.value = true
  try {
    const res = await api.get(`/user/knowledge-bases/${currentKb.value.id}/documents`)
    documentItems.value = res.data.data || []
  } finally {
    documentsLoading.value = false
  }
}

const openDocumentDialog = (row = null) => {
  if (row && isUploadedFileDocument(row)) {
    ElMessage.warning(t('file_doc_no_online_edit'))
    return
  }
  documentEditing.value = !!row
  currentDocumentId.value = row?.id || null
  documentForm.name = row?.name || ''
  documentForm.content = row?.content || ''
  documentDialogVisible.value = true
}

const submitDocument = async () => {
  if (!currentKb.value?.id) return
  if (!documentForm.name.trim()) {
    ElMessage.error(t('doc_name_required'))
    return
  }
  if (!documentForm.content.trim()) {
    ElMessage.error(t('doc_content_required'))
    return
  }
  try {
    let res = null
    if (documentEditing.value) {
      res = await api.put(`/user/knowledge-bases/${currentKb.value.id}/documents/${currentDocumentId.value}`, documentForm)
    } else {
      res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents`, documentForm)
    }
    ElMessage.success(t('doc_save_success'))
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    documentDialogVisible.value = false
    await loadDocuments()
    await loadData()
  } catch (e) {
    const msg = e?.response?.data?.error || '文档保存失败'
    ElMessage.error(msg)
  }
}

const removeDocument = async (docId) => {
  if (!currentKb.value?.id) return
  try {
    await ElMessageBox.confirm(t('confirm_delete_document'), t('hint'), { type: 'warning' })
    const res = await api.delete(`/user/knowledge-bases/${currentKb.value.id}/documents/${docId}`)
    ElMessage.success(t('delete_success'))
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    await loadDocuments()
    await loadData()
  } catch {}
}

const syncDocument = async (docId) => {
  if (!currentKb.value?.id) return
  try {
    const res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents/${docId}/sync`)
    ElMessage.success(res?.data?.message || '同步任务已提交')
    await loadDocuments()
    await loadData()
  } catch (e) {
    const msg = e?.response?.data?.error || '同步失败'
    ElMessage.error(msg)
  }
}

const uploadDocumentFile = async (options) => {
  if (!currentKb.value?.id) {
    ElMessage.error(t('select_knowledge_base'))
    options?.onError?.(new Error('missing knowledge base'))
    return
  }
  if (!isUploadProviderSupported.value) {
    ElMessage.error(`当前知识库提供商为 ${currentKBProvider.value}，暂不支持文件上传创建文档`)
    options?.onError?.(new Error('provider not supported'))
    return
  }
  const file = options?.file
  if (!file) {
    ElMessage.error(t('select_upload_file'))
    options?.onError?.(new Error('missing file'))
    return
  }

  const formData = new FormData()
  formData.append('file', file)
  const fileName = (file.name || '').replace(/\.[^/.]+$/, '')
  if (fileName) {
    formData.append('name', fileName)
  }

  try {
    const res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents/upload`, formData)
    ElMessage.success(res?.data?.message || '文件上传成功')
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    await loadDocuments()
    await loadData()
    options?.onSuccess?.(res?.data)
  } catch (e) {
    const msg = e?.response?.data?.error || '文件上传失败'
    ElMessage.error(msg)
    options?.onError?.(e)
  }
}

const isUploadedFileDocument = (doc) => {
  const content = doc?.content
  return typeof content === 'string' && content.startsWith(FILE_UPLOAD_CONTENT_PREFIX)
}

const getDocumentPreview = (doc) => {
  const content = doc?.content || ''
  if (isUploadedFileDocument(doc)) {
    try {
      const payload = JSON.parse(content.slice(FILE_UPLOAD_CONTENT_PREFIX.length))
      const fileName = payload?.file_name || doc?.name || '上传文件'
      return `[文件] ${fileName}`
    } catch {
      return `[文件] ${doc?.name || '上传文件'}`
    }
  }
  const text = String(content)
  return `${text.slice(0, 120)}${text.length > 120 ? '...' : ''}`
}

const getSyncStatusText = (status) => {
  if (status === 'uploading') return '上传中'
  if (status === 'uploaded') return '已上传'
  if (status === 'parsing') return '解析中'
  if (status === 'upload_failed') return '上传失败'
  if (status === 'parse_failed') return '解析失败'
  if (status === 'synced') return '已同步'
  if (status === 'failed') return '失败'
  return '待同步'
}

const getSyncStatusTagType = (status) => {
  if (status === 'upload_failed' || status === 'parse_failed') return 'danger'
  if (status === 'uploading' || status === 'parsing') return 'warning'
  if (status === 'uploaded') return 'info'
  if (status === 'synced') return 'success'
  if (status === 'failed') return 'danger'
  return 'warning'
}

const getKnowledgeStatusText = (status) => {
  return String(status || '').trim() === 'active' ? '启用' : '停用'
}

const formatProviderText = (provider) => {
  const p = String(provider || '').trim().toLowerCase()
  if (p === 'ragflow') return 'RAGFlow'
  if (p === 'weknora') return 'WeKnora'
  if (p === 'dify') return 'Dify'
  return provider || '-'
}

const shouldShowSyncErrorTip = (row) => {
  const status = String(row?.sync_status || '').trim()
  const syncError = String(row?.sync_error || '').trim()
  if (!syncError) return false
  return status === 'failed' || status === 'upload_failed' || status === 'parse_failed'
}

const formatHitScore = (score) => {
  const n = Number(score)
  if (Number.isNaN(n)) return '-'
  return n.toFixed(4)
}

const formatDocCount = (value) => {
  const n = Number(value)
  if (Number.isNaN(n) || n < 0) return 0
  return n
}

const formatDateTimeCell = (value) => {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString()
}

const formatKnowledgeThreshold = (value) => {
  if (value === null || value === undefined || value === '') return '全局'
  const n = Number(value)
  if (Number.isNaN(n)) return '全局'
  return n.toFixed(2)
}

onMounted(async () => {
  await loadData()
})
</script>

<style scoped>
.page-actions {
  display: flex;
  justify-content: flex-end;
  margin: 10px 0 14px;
}

.kb-sync-status-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.kb-sync-error-tooltip {
  max-width: 320px;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.5;
}

.kb-sync-error-icon {
  color: var(--el-color-danger);
  cursor: pointer;
  font-size: 14px;
}

.kb-desc-text {
  color: var(--el-text-color-regular);
}

.kb-desc-text.is-empty {
  color: var(--el-text-color-placeholder);
}

.action-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.action-buttons :deep(.el-button) {
  margin: 0;
  white-space: nowrap;
}

.action-buttons :deep(.el-dropdown) {
  display: inline-flex;
}

.dialog-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.dialog-toolbar-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.kb-helper-text {
  color: var(--apple-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.kb-helper-text.is-spaced {
  margin-top: 6px;
}

.kb-helper-text.is-bottom {
  margin-bottom: 8px;
}

.kb-helper-text.is-empty {
  margin-top: 10px;
}

.kb-helper-text.is-regular {
  color: var(--el-text-color-regular);
}

.search-hit-content {
  white-space: pre-wrap;
  line-height: 1.4;
}
</style>
