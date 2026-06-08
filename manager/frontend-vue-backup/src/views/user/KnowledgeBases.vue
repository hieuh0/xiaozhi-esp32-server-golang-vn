<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { AlertTriangle } from '@lucide/vue'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

const { t } = useLocale()
const { formatDate: formatDateTimeCell } = useFormatDate()

const loading = ref(false)
const items = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
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
const searchTestResult = reactive({ query: '', count: 0, hits: [] })

const docUploadInput = ref()

const form = reactive({
  name: '', description: '', status: 'active',
  inherit_global_threshold: true, retrieval_threshold_text: '0.2',
  threshold_provider: 'dify', global_threshold: 0.2
})

const documentForm = reactive({ name: '', content: '' })
const searchTestForm = reactive({ query: '', top_k: 5, threshold_text: '' })
const topKOptions = Array.from({ length: 20 }, (_, i) => i + 1)

const FILE_UPLOAD_CONTENT_PREFIX = '__KB_FILE_UPLOAD_V1__:'
const DIFY_UPLOAD_ACCEPT = '.txt,.md,.markdown,.pdf,.html,.htm,.xlsx,.xls,.docx,.csv,.eml,.msg,.pptx,.ppt,.xml,.epub'
const RAGFLOW_UPLOAD_ACCEPT = '.txt,.text,.md,.markdown,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.wps,.json,.csv,.log,.xml,.html,.htm,.yml,.yaml,.rtf,.sql,.ini,.jpg,.jpeg,.png,.gif,.bmp,.webp,.tif,.tiff,.eml,.msg'
const WEKNORA_UPLOAD_ACCEPT = '.txt,.text,.md,.markdown,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.wps,.json,.csv,.log,.xml,.html,.htm,.yml,.yaml,.rtf,.sql,.ini,.jpg,.jpeg,.png,.gif,.bmp,.webp,.tif,.tiff,.eml,.msg'
const DEFAULT_THRESHOLD = 0.2

const knowledgeGlobalConfig = reactive({ default_provider: 'dify', providers: {} })

const currentKBProvider = computed(() => (currentKb.value?.sync_provider || 'dify').toLowerCase())
const uploadAcceptByProvider = computed(() => {
  if (currentKBProvider.value === 'ragflow') return RAGFLOW_UPLOAD_ACCEPT
  if (currentKBProvider.value === 'weknora') return WEKNORA_UPLOAD_ACCEPT
  return DIFY_UPLOAD_ACCEPT
})
const isUploadProviderSupported = computed(() => ['dify', 'ragflow', 'weknora'].includes(currentKBProvider.value))
const uploadTipText = computed(() => {
  if (currentKBProvider.value === 'dify') return t('dify_upload_hint')
  if (currentKBProvider.value === 'ragflow') return t('ragflow_upload_hint')
  if (currentKBProvider.value === 'weknora') return t('weknora_upload_hint')
  return t('provider_upload_unsupported', { provider: currentKBProvider.value })
})

const totalPages = () => Math.ceil(total.value / pageSize.value) || 1

const normalizeProvider = (provider) => {
  const p = String(provider || '').trim().toLowerCase()
  return ['dify', 'ragflow', 'weknora'].includes(p) ? p : 'dify'
}

const getGlobalThresholdByProvider = (provider) => {
  const p = normalizeProvider(provider)
  const cfg = knowledgeGlobalConfig.providers?.[p] || {}
  const key = p === 'dify' ? 'score_threshold' : p === 'ragflow' ? 'similarity_threshold' : 'score_threshold'
  const v = Number(cfg[key])
  return (!Number.isNaN(v) && v >= 0 && v <= 1) ? v : DEFAULT_THRESHOLD
}

const applyKnowledgeGlobalConfig = (knowledge) => {
  const payload = knowledge && typeof knowledge === 'object' ? knowledge : {}
  knowledgeGlobalConfig.default_provider = normalizeProvider(payload.default_provider || 'dify')
  knowledgeGlobalConfig.providers = payload.providers && typeof payload.providers === 'object' ? payload.providers : {}
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await api.get('/user/knowledge-bases', { params: { page: page.value, page_size: pageSize.value } })
    items.value = res.data.data || []
    total.value = res.data.total || 0
    applyKnowledgeGlobalConfig(res.data.knowledge)
  } finally {
    loading.value = false
  }
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
  if (!form.name.trim()) { ElMessage.error(t('name_required')); return }
  const rawThreshold = String(form.retrieval_threshold_text || '').trim()
  const threshold = Number(rawThreshold)
  if (!rawThreshold || Number.isNaN(threshold) || threshold < 0 || threshold > 1) { ElMessage.error(t('retrieval_threshold_range')); return }
  const globalThreshold = Number(form.global_threshold)
  const sameAsGlobal = !Number.isNaN(globalThreshold) && Math.abs(threshold - globalThreshold) < 0.000001
  if (form.inherit_global_threshold && !sameAsGlobal) form.inherit_global_threshold = false
  try {
    const useInheritGlobal = form.inherit_global_threshold && sameAsGlobal
    const payload = { name: form.name, description: form.description, status: form.status, inherit_global_threshold: useInheritGlobal, retrieval_threshold: useInheritGlobal ? null : threshold }
    const res = editing.value ? await api.put(`/user/knowledge-bases/${currentId.value}`, payload) : await api.post('/user/knowledge-bases', payload)
    ElMessage.success(t('save_success'))
    if (res?.data?.warning) ElMessage.warning(res.data.warning)
    dialogVisible.value = false
    await loadData()
  } catch { ElMessage.error(t('save_failed')) }
}

const removeItem = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_knowledge_base'), t('hint'), { type: 'warning' })
    const res = await api.delete(`/user/knowledge-bases/${id}`)
    ElMessage.success(t('delete_success'))
    if (res?.data?.warning) ElMessage.warning(res.data.warning)
    await loadData()
  } catch {}
}

const isStatusSwitchLoading = (id) => !!statusSwitchLoadingMap.value?.[id]

const toggleKnowledgeBaseStatus = async (row, checked) => {
  if (!row?.id) return
  const id = row.id
  const prevStatus = String(row.status || 'inactive').trim() === 'active' ? 'active' : 'inactive'
  const nextStatus = checked ? 'active' : 'inactive'
  if (prevStatus === nextStatus || isStatusSwitchLoading(id)) return
  statusSwitchLoadingMap.value = { ...statusSwitchLoadingMap.value, [id]: true }
  row.status = nextStatus
  try {
    const res = await api.put(`/user/knowledge-bases/${id}`, { name: row.name || '', description: row.description || '', content: row.content || '', status: nextStatus })
    if (res?.data?.warning) ElMessage.warning(res.data.warning)
    else ElMessage.success(nextStatus === 'active' ? t('enabled') : t('deactivate'))
    await loadData()
  } catch (e) {
    row.status = prevStatus
    ElMessage.error(e?.response?.data?.error || t('status_update_failed'))
  } finally {
    statusSwitchLoadingMap.value = { ...statusSwitchLoadingMap.value, [id]: false }
  }
}

const handleKnowledgeBaseAction = async (command, row) => {
  if (!row?.id) return
  if (command === 'edit') { openDialog(row); return }
  if (command === 'sync') { await syncItem(row.id); return }
  if (command === 'delete') await removeItem(row.id)
}

const syncItem = async (id) => {
  try {
    const res = await api.post(`/user/knowledge-bases/${id}/sync`)
    ElMessage.success(res?.data?.message || t('sync_submitted'))
    await loadData()
  } catch (e) {
    ElMessage.error(e?.response?.data?.error || t('sync_failed'))
    await loadData()
  }
}

const openSearchTestDialog = (row) => {
  searchTestKb.value = row || null
  searchTestForm.query = ''
  searchTestForm.top_k = 5
  const provider = normalizeProvider(row?.sync_provider || knowledgeGlobalConfig.default_provider || 'dify')
  const globalThreshold = getGlobalThresholdByProvider(provider)
  const effectiveThreshold = (row?.retrieval_threshold !== null && row?.retrieval_threshold !== undefined) ? Number(row.retrieval_threshold) : Number(globalThreshold)
  searchTestForm.threshold_text = Number.isNaN(effectiveThreshold) ? '' : String(effectiveThreshold)
  searchTestResult.query = ''; searchTestResult.count = 0; searchTestResult.hits = []
  searchTestElapsedMs.value = null; hasRunSearchTest.value = false
  searchTestVisible.value = true
}

const runSearchTest = async () => {
  if (!searchTestKb.value?.id) { ElMessage.error(t('select_knowledge_base')); return }
  const query = (searchTestForm.query || '').trim()
  if (!query) { ElMessage.error(t('enter_test_keyword')); return }
  searchTestLoading.value = true
  const startedAt = Date.now()
  try {
    const rawThreshold = String(searchTestForm.threshold_text || '').trim()
    let threshold = null
    if (rawThreshold !== '') {
      const parsed = Number(rawThreshold)
      if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) { ElMessage.error(t('threshold_0_to_1')); return }
      threshold = parsed
    }
    const res = await api.post(`/user/knowledge-bases/${searchTestKb.value.id}/test-search`, { query, top_k: Number(searchTestForm.top_k) || 5, threshold })
    const data = res?.data?.data || {}
    searchTestResult.query = data.query || query
    searchTestResult.count = Number(data.count || 0)
    searchTestResult.hits = Array.isArray(data.hits) ? data.hits : []
    const elapsed = Number(data.elapsed_ms)
    searchTestElapsedMs.value = Number.isNaN(elapsed) ? Date.now() - startedAt : elapsed
    hasRunSearchTest.value = true
    ElMessage.success(t('recall_complete', { count: searchTestResult.count }))
  } catch (e) {
    ElMessage.error(e?.response?.data?.error || t('test_failed'))
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
  if (row && isUploadedFileDocument(row)) { ElMessage.warning(t('file_doc_no_online_edit')); return }
  documentEditing.value = !!row
  currentDocumentId.value = row?.id || null
  documentForm.name = row?.name || ''
  documentForm.content = row?.content || ''
  documentDialogVisible.value = true
}

const submitDocument = async () => {
  if (!currentKb.value?.id) return
  if (!documentForm.name.trim()) { ElMessage.error(t('doc_name_required')); return }
  if (!documentForm.content.trim()) { ElMessage.error(t('doc_content_required')); return }
  try {
    const res = documentEditing.value
      ? await api.put(`/user/knowledge-bases/${currentKb.value.id}/documents/${currentDocumentId.value}`, documentForm)
      : await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents`, documentForm)
    ElMessage.success(t('doc_save_success'))
    if (res?.data?.warning) ElMessage.warning(res.data.warning)
    documentDialogVisible.value = false
    await loadDocuments(); await loadData()
  } catch (e) { ElMessage.error(e?.response?.data?.error || t('doc_save_failed')) }
}

const removeDocument = async (docId) => {
  if (!currentKb.value?.id) return
  try {
    await ElMessageBox.confirm(t('confirm_delete_document'), t('hint'), { type: 'warning' })
    const res = await api.delete(`/user/knowledge-bases/${currentKb.value.id}/documents/${docId}`)
    ElMessage.success(t('delete_success'))
    if (res?.data?.warning) ElMessage.warning(res.data.warning)
    await loadDocuments(); await loadData()
  } catch {}
}

const syncDocument = async (docId) => {
  if (!currentKb.value?.id) return
  try {
    const res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents/${docId}/sync`)
    ElMessage.success(res?.data?.message || t('sync_submitted'))
    await loadDocuments(); await loadData()
  } catch (e) { ElMessage.error(e?.response?.data?.error || t('sync_failed')) }
}

const triggerDocUpload = () => { if (isUploadProviderSupported.value) docUploadInput.value?.click() }

const handleDocFileSelect = async (e) => {
  const file = e.target?.files?.[0]
  if (!file || !currentKb.value?.id) return
  if (!isUploadProviderSupported.value) { ElMessage.error(t('provider_upload_unsupported', { provider: currentKBProvider.value })); return }
  const formData = new FormData()
  formData.append('file', file)
  const fileName = (file.name || '').replace(/\.[^/.]+$/, '')
  if (fileName) formData.append('name', fileName)
  try {
    const res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents/upload`, formData)
    ElMessage.success(res?.data?.message || t('file_upload_success'))
    if (res?.data?.warning) ElMessage.warning(res.data.warning)
    await loadDocuments(); await loadData()
  } catch (e) {
    ElMessage.error(e?.response?.data?.error || t('file_upload_failed'))
  } finally {
    e.target.value = ''
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
      return t('file_document_name', { name: payload?.file_name || doc?.name || t('upload_file') })
    } catch { return t('file_document_name', { name: doc?.name || t('upload_file') }) }
  }
  const text = String(content)
  return `${text.slice(0, 120)}${text.length > 120 ? '...' : ''}`
}

const getSyncStatusText = (status) => {
  const map = { uploading: t('uploading'), uploaded: t('uploaded'), parsing: t('parsing'), upload_failed: t('upload_failed'), parse_failed: t('parse_failed'), synced: t('synced'), failed: t('failed') }
  return map[status] || t('pending_sync')
}

const getSyncStatusBadge = (status) => {
  if (status === 'upload_failed' || status === 'parse_failed' || status === 'failed') return 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
  if (status === 'uploading' || status === 'parsing') return 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800'
  if (status === 'synced') return 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
  return 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'
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
  return !!syncError && ['failed', 'upload_failed', 'parse_failed'].includes(status)
}

const formatHitScore = (score) => { const n = Number(score); return Number.isNaN(n) ? '-' : n.toFixed(4) }
const formatDocCount = (value) => { const n = Number(value); return Number.isNaN(n) || n < 0 ? 0 : n }
const formatKnowledgeThreshold = (value) => {
  if (value === null || value === undefined || value === '') return t('global')
  const n = Number(value)
  return Number.isNaN(n) ? t('global') : n.toFixed(2)
}

onMounted(loadData)
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex justify-end">
      <Button @click="openDialog()">{{ t('add_knowledge_base') }}</Button>
    </div>

    <!-- Main table -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
      <div class="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead class="w-14">ID</TableHead>
              <TableHead class="w-32">{{ t('name') }}</TableHead>
              <TableHead>{{ t('description') }}</TableHead>
              <TableHead class="w-24">{{ t('provider') }}</TableHead>
              <TableHead class="w-20 text-center">{{ t('doc_count_col') }}</TableHead>
              <TableHead class="w-36">{{ t('sync_status') }}</TableHead>
              <TableHead class="w-44">{{ t('last_sync_col') }}</TableHead>
              <TableHead class="w-24 text-center">{{ t('status') }}</TableHead>
              <TableHead class="w-48">{{ t('actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="9" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableEmpty v-if="!items.length" />
              <TableRow v-for="row in items" :key="row.id">
                <TableCell class="text-xs font-mono text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
                <TableCell class="font-medium text-sm" :title="row.name">{{ row.name }}</TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)] max-w-[180px] truncate" :title="(row.description || '').trim() || '-'">
                  {{ (row.description || '').trim() || '-' }}
                </TableCell>
                <TableCell>
                  <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700">
                    {{ formatProviderText(row.sync_provider) }}
                  </span>
                </TableCell>
                <TableCell class="text-center text-sm">{{ formatDocCount(row.doc_count) }}</TableCell>
                <TableCell>
                  <div class="flex items-center gap-1.5">
                    <span :class="getSyncStatusBadge(row.sync_status)">{{ getSyncStatusText(row.sync_status) }}</span>
                    <AlertTriangle
                      v-if="shouldShowSyncErrorTip(row)"
                      class="w-3.5 h-3.5 text-red-500 shrink-0 cursor-help"
                      :title="row.sync_error"
                    />
                  </div>
                </TableCell>
                <TableCell class="text-xs text-[var(--color-text-secondary)]">{{ formatDateTimeCell(row.last_synced_at) }}</TableCell>
                <TableCell class="text-center">
                  <Switch
                    :model-value="String(row.status || '').trim() === 'active'"
                    :disabled="isStatusSwitchLoading(row.id)"
                    @update:model-value="(v) => toggleKnowledgeBaseStatus(row, v)"
                  />
                </TableCell>
                <TableCell>
                  <div class="flex items-center gap-1 flex-wrap">
                    <Button variant="outline" size="sm" @click="openDocuments(row)">{{ t('document_btn') }}</Button>
                    <Button variant="outline" size="sm" @click="openSearchTestDialog(row)">{{ t('test') }}</Button>
                    <DropdownMenu>
                      <DropdownMenuTrigger as-child>
                        <Button variant="outline" size="sm">{{ t('more') }}</Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem @click="handleKnowledgeBaseAction('edit', row)">{{ t('edit') }}</DropdownMenuItem>
                        <DropdownMenuItem @click="handleKnowledgeBaseAction('sync', row)">{{ t('retry_sync') }}</DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem class="text-destructive" @click="handleKnowledgeBaseAction('delete', row)">{{ t('delete') }}</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
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
        <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadData()">{{ t('prev') }}</Button>
        <span>{{ page }} / {{ totalPages() }}</span>
        <Button variant="outline" size="sm" :disabled="page >= totalPages()" @click="page++; loadData()">{{ t('next') }}</Button>
      </div>
    </div>

    <!-- KB form dialog -->
    <Dialog v-model:open="dialogVisible">
      <DialogContent class="max-w-[680px]">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('edit_knowledge_base') : t('add_knowledge_base') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('name') }}</label>
            <Input v-model="form.name" maxlength="100" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('description') }}</label>
            <Input v-model="form.description" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('sync_note_label') }}</label>
            <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('kb_helper_text') }}</p>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('retrieve_threshold') }}</label>
            <Input v-model="form.retrieval_threshold_text" :placeholder="t('threshold_test_ph')" />
            <p class="text-xs text-[var(--color-text-tertiary)]">
              {{ t('threshold_hint', { provider: form.threshold_provider || '-', threshold: formatKnowledgeThreshold(form.global_threshold) }) }}
            </p>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('status') }}</label>
            <Select v-model="form.status">
              <SelectTrigger class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">active</SelectItem>
                <SelectItem value="inactive">inactive</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="dialogVisible = false">{{ t('cancel') }}</Button>
          <Button @click="submit">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Documents management dialog -->
    <Dialog v-model:open="documentsVisible">
      <DialogContent class="max-w-[900px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('document_management') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-3 py-2">
          <div class="flex items-center justify-between gap-2 flex-wrap">
            <span class="text-sm text-[var(--color-text-secondary)]">{{ t('current_kb_info', { name: currentKb?.name || '-' }) }}</span>
            <div class="flex gap-2">
              <Button variant="outline" :disabled="!isUploadProviderSupported" @click="triggerDocUpload">{{ t('upload_file') }}</Button>
              <input ref="docUploadInput" type="file" :accept="uploadAcceptByProvider" class="hidden" @change="handleDocFileSelect" />
              <Button @click="openDocumentDialog()">{{ t('add_document') }}</Button>
            </div>
          </div>
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ uploadTipText }}</p>

          <div class="rounded-xl border border-[var(--color-line)] overflow-hidden">
            <div class="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead class="w-20">ID</TableHead>
                    <TableHead class="w-44">{{ t('document_name') }}</TableHead>
                    <TableHead class="w-52">Document ID</TableHead>
                    <TableHead>{{ t('content_preview') }}</TableHead>
                    <TableHead class="w-28">{{ t('sync_status') }}</TableHead>
                    <TableHead class="w-44">{{ t('latest_sync_time') }}</TableHead>
                    <TableHead class="w-48">{{ t('actions') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-if="documentsLoading">
                    <TableCell colspan="7" class="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
                  </TableRow>
                  <template v-else>
                    <TableEmpty v-if="!documentItems.length" />
                    <TableRow v-for="doc in documentItems" :key="doc.id">
                      <TableCell class="text-xs font-mono text-[var(--color-text-secondary)]">{{ doc.id }}</TableCell>
                      <TableCell class="text-sm font-medium">{{ doc.name }}</TableCell>
                      <TableCell class="text-xs font-mono text-[var(--color-text-secondary)]">{{ doc.external_doc_id }}</TableCell>
                      <TableCell class="text-xs text-[var(--color-text-secondary)] max-w-[180px] truncate">{{ getDocumentPreview(doc) }}</TableCell>
                      <TableCell><span :class="getSyncStatusBadge(doc.sync_status)">{{ getSyncStatusText(doc.sync_status) }}</span></TableCell>
                      <TableCell class="text-xs text-[var(--color-text-secondary)]">{{ doc.last_synced_at }}</TableCell>
                      <TableCell>
                        <div class="flex gap-1 flex-wrap">
                          <Button variant="outline" size="sm" :disabled="isUploadedFileDocument(doc)" @click="openDocumentDialog(doc)">{{ t('edit') }}</Button>
                          <Button variant="outline" size="sm" @click="syncDocument(doc.id)">{{ t('retry_sync') }}</Button>
                          <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="removeDocument(doc.id)">{{ t('delete') }}</Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  </template>
                </TableBody>
              </Table>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Document form dialog -->
    <Dialog v-model:open="documentDialogVisible">
      <DialogContent class="max-w-[700px]">
        <DialogHeader>
          <DialogTitle>{{ documentEditing ? t('edit_document') : t('add_document') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('document_name') }}</label>
            <Input v-model="documentForm.name" maxlength="200" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('content_label') }}</label>
            <textarea
              v-model="documentForm.content"
              rows="12"
              :placeholder="t('content_ph')"
              class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none resize-none"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="documentDialogVisible = false">{{ t('cancel') }}</Button>
          <Button @click="submitDocument">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Search test dialog -->
    <Dialog v-model:open="searchTestVisible">
      <DialogContent class="max-w-[960px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('search_test_title') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-3 py-2">
          <div class="flex items-center justify-between gap-2 flex-wrap">
            <div class="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
              <span>{{ t('current_kb_info', { name: searchTestKb?.name || '-' }) }}</span>
              <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700">
                {{ searchTestKb?.sync_provider || '-' }}
              </span>
            </div>
            <div class="flex items-center gap-2 flex-wrap">
              <Input v-model="searchTestForm.query" :placeholder="t('search_test_ph')" class="w-[240px]" @keyup.enter="runSearchTest" />
              <span class="text-xs text-[var(--color-text-tertiary)] whitespace-nowrap" :title="t('topk_tooltip')">TopK</span>
              <Select v-model="searchTestForm.top_k">
                <SelectTrigger class="w-[90px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="k in topKOptions" :key="k" :value="k">{{ k }}</SelectItem>
                </SelectContent>
              </Select>
              <span class="text-xs text-[var(--color-text-tertiary)] whitespace-nowrap" :title="t('threshold_test_hint')">{{ t('threshold') }}</span>
              <Input v-model="searchTestForm.threshold_text" :placeholder="t('threshold_test_ph')" class="w-[100px]" />
              <Button :disabled="searchTestLoading" @click="runSearchTest">{{ t('run_search_test') }}</Button>
            </div>
          </div>
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('search_test_hint') }}</p>
          <p v-if="searchTestElapsedMs !== null" class="text-xs text-[var(--color-text-secondary)]">{{ t('response_time_ms', { ms: searchTestElapsedMs }) }}</p>

          <div class="rounded-xl border border-[var(--color-line)] overflow-hidden max-h-[420px] overflow-y-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead class="w-12">#</TableHead>
                  <TableHead class="w-48">{{ t('source_label') }}</TableHead>
                  <TableHead class="w-28">{{ t('score_col') }}</TableHead>
                  <TableHead>{{ t('hit_content_col') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="searchTestLoading">
                  <TableCell colspan="4" class="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
                </TableRow>
                <template v-else>
                  <TableEmpty v-if="!searchTestResult.hits.length" />
                  <TableRow v-for="(hit, idx) in searchTestResult.hits" :key="idx">
                    <TableCell class="text-[var(--color-text-secondary)]">{{ idx + 1 }}</TableCell>
                    <TableCell class="text-sm">{{ hit.title }}</TableCell>
                    <TableCell class="text-sm font-mono">{{ formatHitScore(hit.score) }}</TableCell>
                    <TableCell class="text-sm whitespace-pre-wrap leading-relaxed">{{ hit.content }}</TableCell>
                  </TableRow>
                </template>
              </TableBody>
            </Table>
          </div>
          <p v-if="!searchTestLoading && hasRunSearchTest && !searchTestResult.hits.length" class="text-xs text-[var(--color-text-tertiary)]">{{ t('search_no_hit') }}</p>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
