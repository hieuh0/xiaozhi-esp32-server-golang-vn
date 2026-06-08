<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal, RefreshCw, Search } from '@lucide/vue'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const { t } = useLocale()

const activeTab = ref('discover')

const markets = ref([])
const marketsLoading = ref(false)
const providerOptions = ref([])
const marketDialogVisible = ref(false)
const marketSaving = ref(false)
const editingMarket = ref(null)

const marketForm = reactive({
  name: '', provider_id: 'modelscope', catalog_url: '', detail_url_template: '', enabled: true,
  auth: { type: 'bearer', token: '', header_name: 'Authorization' }
})

const selectableProviderOptions = computed(() => providerOptions.value.filter(item => item.id !== 'generic'))
const currentProvider = computed(() => selectableProviderOptions.value.find(item => item.id === marketForm.provider_id) || null)

const services = ref([])
const servicesLoading = ref(false)
const serviceWarnings = ref([])
const servicePage = ref(1)
const servicePageSize = ref(20)
const serviceTotal = ref(0)
const serviceQuery = ref('')
const detailDialogVisible = ref(false)
const detailLoading = ref(false)
const detailImporting = ref(false)
const serviceDetail = ref(null)

const importedLoading = ref(false)
const importedSaving = ref(false)
const importedDialogVisible = ref(false)
const importedToolsDialogVisible = ref(false)
const importedToolsLoading = ref(false)
const editingImported = ref(null)
const importedToolTarget = ref(null)
const importedItems = ref([])
const importedToolOptions = ref([])
const importedPage = ref(1)
const importedPageSize = ref(20)
const importedTotal = ref(0)
const importedQuery = ref('')
const importedHeadersText = ref('')
const importedToolMode = ref('all')
const importedToolQuery = ref('')
const importedSelectedTools = ref([])

const importedForm = reactive({
  name: '', enabled: true, transport: 'streamablehttp', url: '',
  allowed_tools: [], market_id: null, provider_id: '', service_id: '', service_name: ''
})

const toolDialogTitle = computed(() =>
  importedToolTarget.value ? t('tool_selection_title', { name: importedToolTarget.value.name }) : t('tool_selection')
)

const filteredImportedToolOptions = computed(() => {
  const query = importedToolQuery.value.trim().toLowerCase()
  if (!query) return importedToolOptions.value
  return importedToolOptions.value.filter(tool =>
    tool.name.toLowerCase().includes(query) || (tool.description || '').toLowerCase().includes(query)
  )
})

const servicePageCount = computed(() => Math.ceil(serviceTotal.value / servicePageSize.value) || 1)
const importedPageCount = computed(() => Math.ceil(importedTotal.value / importedPageSize.value) || 1)

const badgeClass = (color) => {
  const map = {
    gray: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700',
    blue: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800',
    green: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800',
    yellow: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800',
  }
  return map[color] || map.gray
}

const validateMarketForm = () => {
  if (!marketForm.name.trim()) { ElMessage.error(t('enter_name')); return false }
  if (!marketForm.catalog_url.trim()) { ElMessage.error(t('enter_directory_url')); return false }
  return true
}

const validateImportedForm = () => {
  if (!importedForm.name.trim()) { ElMessage.error(t('enter_name')); return false }
  if (!importedForm.transport) { ElMessage.error(t('select_transport_type')); return false }
  if (!importedForm.url.trim()) { ElMessage.error(t('enter_url')); return false }
  return true
}

const getDefaultProviderId = () => {
  if (selectableProviderOptions.value.length === 0) return 'modelscope'
  if (selectableProviderOptions.value.some(item => item.id === 'modelscope')) return 'modelscope'
  return selectableProviderOptions.value[0].id
}

const loadProviders = async () => {
  try {
    const resp = await api.get('/admin/mcp-market/providers')
    providerOptions.value = resp.data.data || []
    if (!marketForm.provider_id) marketForm.provider_id = getDefaultProviderId()
    if (!selectableProviderOptions.value.some(item => item.id === marketForm.provider_id))
      marketForm.provider_id = getDefaultProviderId()
  } catch (error) {
    providerOptions.value = [{ id: 'modelscope', name: t('modelscope') }]
    marketForm.provider_id = marketForm.provider_id || 'modelscope'
    ElMessage.error(error.response?.data?.error || t('load_provider_failed'))
  }
}

const applyProviderPreset = (providerId, force = false) => {
  const provider = selectableProviderOptions.value.find(item => item.id === providerId)
  if (!provider) return
  if (force || !marketForm.catalog_url) marketForm.catalog_url = provider.catalog_url || ''
  if (force || !marketForm.detail_url_template) marketForm.detail_url_template = provider.detail_url_template || ''
  if (force || !marketForm.auth.type) marketForm.auth.type = 'bearer'
  marketForm.auth.header_name = 'Authorization'
  if (force) marketForm.auth.token = ''
  if (!editingMarket.value && (force || !marketForm.name) && provider.id === 'modelscope')
    marketForm.name = t('modao_mcp_market')
}

const handleProviderChange = (providerId) => { applyProviderPreset(providerId, true) }

const handleMarketAction = async (command, row) => {
  if (command === 'edit') { openEditDialog(row); return }
  if (command === 'test') { await testMarket(row); return }
  if (command === 'delete') await deleteMarket(row)
}

const loadMarkets = async () => {
  marketsLoading.value = true
  try {
    const resp = await api.get('/admin/mcp-markets')
    markets.value = resp.data.data || []
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('load_mcp_market_failed'))
  } finally {
    marketsLoading.value = false
  }
}

const resetMarketForm = () => {
  marketForm.name = ''; marketForm.provider_id = getDefaultProviderId()
  marketForm.catalog_url = ''; marketForm.detail_url_template = ''; marketForm.enabled = true
  marketForm.auth.type = 'bearer'; marketForm.auth.token = ''; marketForm.auth.header_name = 'Authorization'
}

const openCreateDialog = () => {
  editingMarket.value = null; resetMarketForm()
  applyProviderPreset(marketForm.provider_id, true); marketDialogVisible.value = true
}

const openEditDialog = (row) => {
  editingMarket.value = row; marketForm.name = row.name
  const rowProviderId = row.provider_id || getDefaultProviderId()
  marketForm.provider_id = selectableProviderOptions.value.some(item => item.id === rowProviderId) ? rowProviderId : getDefaultProviderId()
  marketForm.catalog_url = row.catalog_url; marketForm.detail_url_template = row.detail_url_template || ''
  marketForm.enabled = !!row.enabled; marketForm.auth.type = 'bearer'
  marketForm.auth.header_name = 'Authorization'; marketForm.auth.token = ''
  marketDialogVisible.value = true
}

const saveMarket = async () => {
  if (!validateMarketForm()) return
  const payload = {
    name: marketForm.name, provider_id: marketForm.provider_id, catalog_url: marketForm.catalog_url,
    detail_url_template: marketForm.detail_url_template, enabled: marketForm.enabled,
    auth: { type: 'bearer', token: marketForm.auth.token, header_name: 'Authorization' }
  }
  marketSaving.value = true
  try {
    if (editingMarket.value) {
      await api.put(`/admin/mcp-markets/${editingMarket.value.id}`, payload)
      ElMessage.success(t('update_success'))
    } else {
      await api.post('/admin/mcp-markets', payload)
      ElMessage.success(t('create_success'))
    }
    marketDialogVisible.value = false; await loadMarkets(); await loadServices(1)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('save_failed'))
  } finally {
    marketSaving.value = false
  }
}

const deleteMarket = async (row) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_mcp_market', { name: row.name }), t('hint'), {
      type: 'warning', confirmButtonText: t('delete'), cancelButtonText: t('cancel')
    })
    await api.delete(`/admin/mcp-markets/${row.id}`)
    ElMessage.success(t('delete_success')); await loadMarkets(); await loadServices(1)
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.error || t('delete_failed'))
  }
}

const testMarket = async (row) => {
  try {
    const resp = await api.post(`/admin/mcp-markets/${row.id}/test`)
    ElMessage.success(t('connection_success_count', { count: resp.data?.data?.service_count ?? 0 }))
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('connection_test_failed'))
  }
}

const loadServices = async (page = 1) => {
  servicePage.value = page; servicesLoading.value = true
  try {
    const resp = await api.get('/admin/mcp-market/services', {
      params: { q: serviceQuery.value, page: servicePage.value, page_size: servicePageSize.value }
    })
    const data = resp.data.data || {}
    services.value = data.items || []; serviceTotal.value = data.total || 0; serviceWarnings.value = data.warnings || []
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('load_agg_service_failed'))
  } finally {
    servicesLoading.value = false
  }
}

const loadServiceDetail = async (row) => {
  detailDialogVisible.value = true; detailLoading.value = true; serviceDetail.value = null
  try {
    const resp = await api.get(`/admin/mcp-market/services/${row.market_id}/${encodeURIComponent(row.service_id)}`)
    serviceDetail.value = resp.data?.data || null
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('load_service_detail_failed'))
  } finally {
    detailLoading.value = false
  }
}

const importFromDetail = async () => {
  const row = serviceDetail.value
  if (!row?.market_id || !row?.service_id) { ElMessage.error(t('service_id_missing')); return }
  detailImporting.value = true
  try {
    const resp = await api.post('/admin/mcp-market/import', { market_id: row.market_id, service_id: row.service_id, name_override: '' })
    ElMessage.success(t('import_success_count', { count: resp.data.data?.imported_count || 0 }))
    await loadServices(servicePage.value); await loadImportedItems(1)
    detailDialogVisible.value = false; activeTab.value = 'imported'
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('import_failed'))
  } finally {
    detailImporting.value = false
  }
}

const loadImportedItems = async (page = 1) => {
  importedPage.value = page; importedLoading.value = true
  try {
    const resp = await api.get('/admin/mcp-market/imported-services', {
      params: { q: importedQuery.value, page: importedPage.value, page_size: importedPageSize.value }
    })
    const data = resp.data.data || {}
    importedItems.value = data.items || []; importedTotal.value = data.total || 0
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('load_import_service_failed'))
  } finally {
    importedLoading.value = false
  }
}

const parseImportedHeaders = () => {
  const txt = importedHeadersText.value.trim()
  if (!txt) return null
  try {
    const parsed = JSON.parse(txt)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error(t('headers_must_json'))
    return parsed
  } catch {
    throw new Error(t('headers_invalid_json'))
  }
}

const resetImportedForm = () => {
  importedForm.name = ''; importedForm.enabled = true; importedForm.transport = 'streamablehttp'
  importedForm.url = ''; importedForm.market_id = null; importedForm.provider_id = ''
  importedForm.service_id = ''; importedForm.service_name = ''; importedHeadersText.value = ''
}

const mergeImportedToolOptions = (tools = [], selected = []) => {
  const merged = new Map()
  ;(tools || []).forEach(tool => { if (!tool?.name) return; merged.set(tool.name, { name: tool.name, description: tool.description || '' }) })
  ;(selected || []).forEach(name => { if (!name || merged.has(name)) return; merged.set(name, { name, description: t('current_config_selected') }) })
  importedToolOptions.value = Array.from(merged.values()).sort((a, b) => a.name.localeCompare(b.name))
}

const loadImportedToolOptions = async (serviceId) => {
  if (!serviceId) { mergeImportedToolOptions([], importedSelectedTools.value); return }
  importedToolsLoading.value = true
  try {
    const resp = await api.get(`/admin/mcp-market/imported-services/${serviceId}/tools`)
    mergeImportedToolOptions(resp.data?.data?.tools || [], importedSelectedTools.value)
  } catch (error) {
    mergeImportedToolOptions([], importedSelectedTools.value)
    ElMessage.error(error.response?.data?.error || t('load_tool_list_failed'))
  } finally {
    importedToolsLoading.value = false
  }
}

const openCreateImportedDialog = () => { editingImported.value = null; resetImportedForm(); importedDialogVisible.value = true }

const openEditImportedDialog = (row) => {
  editingImported.value = row; importedForm.name = row.name || ''; importedForm.enabled = !!row.enabled
  importedForm.transport = row.transport || 'streamablehttp'; importedForm.url = row.url || ''
  importedForm.market_id = row.market_id || null; importedForm.provider_id = row.provider_id || ''
  importedForm.service_id = row.service_id || ''; importedForm.service_name = row.service_name || ''
  importedHeadersText.value = row.headers ? JSON.stringify(row.headers, null, 2) : ''
  importedDialogVisible.value = true
}

const syncImportedToolMode = (selected = importedSelectedTools.value) => {
  importedToolMode.value = selected.length > 0 ? 'selected' : 'all'
}

const handleImportedToolModeChange = (mode) => {
  importedToolMode.value = mode; importedToolQuery.value = ''
  if (mode === 'all') importedSelectedTools.value = []
}

const toggleImportedTool = (name) => {
  const idx = importedSelectedTools.value.indexOf(name)
  if (idx === -1) importedSelectedTools.value.push(name)
  else importedSelectedTools.value.splice(idx, 1)
}

const openImportedToolsDialog = async (row) => {
  importedToolTarget.value = row
  importedSelectedTools.value = Array.isArray(row.allowed_tools) ? [...row.allowed_tools] : []
  importedToolQuery.value = ''; syncImportedToolMode(importedSelectedTools.value)
  mergeImportedToolOptions([], importedSelectedTools.value); importedToolsDialogVisible.value = true
  await loadImportedToolOptions(row.id)
}

const refreshImportedTools = async () => {
  if (!importedToolTarget.value?.id) { ElMessage.warning(t('select_imported_service')); return }
  await loadImportedToolOptions(importedToolTarget.value.id)
}

const saveImportedItem = async () => {
  if (!validateImportedForm()) return
  let headers = null
  try { headers = parseImportedHeaders() } catch (error) { ElMessage.error(error.message); return }
  const payload = {
    name: importedForm.name, enabled: importedForm.enabled, transport: importedForm.transport,
    url: importedForm.url, headers, allowed_tools: editingImported.value?.allowed_tools || [],
    market_id: importedForm.market_id || null, provider_id: importedForm.provider_id,
    service_id: importedForm.service_id, service_name: importedForm.service_name
  }
  importedSaving.value = true
  try {
    if (editingImported.value) {
      await api.put(`/admin/mcp-market/imported-services/${editingImported.value.id}`, payload)
      ElMessage.success(t('update_success'))
    } else {
      await api.post('/admin/mcp-market/imported-services', payload)
      ElMessage.success(t('create_success'))
    }
    importedDialogVisible.value = false; await loadImportedItems(importedPage.value)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('save_failed'))
  } finally {
    importedSaving.value = false
  }
}

const saveImportedToolSelection = async () => {
  if (!importedToolTarget.value) return
  if (importedToolMode.value === 'selected' && importedSelectedTools.value.length === 0) {
    ElMessage.warning(t('select_at_least_one_tool')); return
  }
  const row = importedToolTarget.value
  const payload = {
    name: row.name, enabled: row.enabled, transport: row.transport, url: row.url,
    headers: row.headers || null, allowed_tools: importedToolMode.value === 'all' ? [] : importedSelectedTools.value,
    market_id: row.market_id || null, provider_id: row.provider_id || '',
    service_id: row.service_id || '', service_name: row.service_name || ''
  }
  importedSaving.value = true
  try {
    await api.put(`/admin/mcp-market/imported-services/${row.id}`, payload)
    ElMessage.success(t('tool_strategy_updated'))
    importedToolsDialogVisible.value = false; importedToolTarget.value = null; importedToolQuery.value = ''
    await loadImportedItems(importedPage.value)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('save_failed'))
  } finally {
    importedSaving.value = false
  }
}

const toggleImportedEnabled = async (row) => {
  const payload = {
    name: row.name, enabled: !row.enabled, transport: row.transport, url: row.url,
    headers: row.headers || null, allowed_tools: row.allowed_tools || [],
    market_id: row.market_id || null, provider_id: row.provider_id || '',
    service_id: row.service_id || '', service_name: row.service_name || ''
  }
  try {
    await api.put(`/admin/mcp-market/imported-services/${row.id}`, payload)
    ElMessage.success(row.enabled ? t('disabled_done') : t('enabled'))
    await loadImportedItems(importedPage.value)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('update_status_failed'))
  }
}

const deleteImportedItem = async (row) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_imported_service', { name: row.name }), t('hint'), {
      type: 'warning', confirmButtonText: t('delete'), cancelButtonText: t('cancel')
    })
    await api.delete(`/admin/mcp-market/imported-services/${row.id}`)
    ElMessage.success(t('delete_success')); await loadImportedItems(importedPage.value)
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.error || t('delete_failed'))
  }
}

onMounted(async () => {
  await loadProviders(); await loadMarkets(); await loadServices(1); await loadImportedItems(1)
})
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <Tabs v-model="activeTab">
      <TabsList>
        <TabsTrigger value="discover">{{ t('market_discovery') }}</TabsTrigger>
        <TabsTrigger value="imported">
          {{ t('mcp_market_imported') }}
          <span v-if="importedTotal > 0" class="ml-1.5 inline-flex items-center justify-center rounded-full bg-[var(--color-primary)] text-white text-[10px] font-semibold leading-none px-1.5 py-0.5 min-w-[18px]">{{ importedTotal > 999 ? '999+' : importedTotal }}</span>
        </TabsTrigger>
      </TabsList>

      <!-- Discover Tab -->
      <TabsContent value="discover" class="mt-4">
        <div class="grid grid-cols-1 lg:grid-cols-[45fr_55fr] gap-4">
          <!-- Markets panel -->
          <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
            <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--color-line)]">
              <span class="font-medium text-sm text-[var(--color-text)]">{{ t('mcp_market_title') }}</span>
              <div class="flex gap-2">
                <Button size="sm" @click="openCreateDialog"><Plus class="w-3.5 h-3.5 mr-1" />{{ t('add_connection') }}</Button>
                <Button variant="outline" size="sm" @click="loadMarkets"><RefreshCw class="w-3.5 h-3.5" /></Button>
              </div>
            </div>
            <div class="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('name') }}</TableHead>
                    <TableHead class="w-28">{{ t('provider') }}</TableHead>
                    <TableHead>{{ t('catalog_url') }}</TableHead>
                    <TableHead class="w-24">{{ t('auth') }}</TableHead>
                    <TableHead class="w-20">{{ t('status') }}</TableHead>
                    <TableHead class="w-16 text-center">{{ t('actions') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-if="marketsLoading">
                    <TableCell colspan="6" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
                  </TableRow>
                  <template v-else>
                    <TableEmpty v-if="!markets.length" />
                    <TableRow v-for="row in markets" :key="row.id">
                      <TableCell class="font-medium text-sm">{{ row.name }}</TableCell>
                      <TableCell><span :class="badgeClass('blue')">{{ row.provider_id || 'generic' }}</span></TableCell>
                      <TableCell class="text-xs text-[var(--color-text-secondary)] truncate max-w-[160px]" :title="row.catalog_url">{{ row.catalog_url }}</TableCell>
                      <TableCell><span :class="badgeClass(row.has_token ? 'green' : 'gray')">{{ row.auth_type || 'none' }}</span></TableCell>
                      <TableCell><span :class="badgeClass(row.enabled ? 'green' : 'gray')">{{ row.enabled ? t('enabled') : t('disable') }}</span></TableCell>
                      <TableCell class="text-center">
                        <DropdownMenu>
                          <DropdownMenuTrigger as-child>
                            <Button variant="ghost" size="icon" class="h-8 w-8" :aria-label="t('more_actions')"><MoreHorizontal class="h-4 w-4" /></Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem @click="handleMarketAction('edit', row)">{{ t('edit') }}</DropdownMenuItem>
                            <DropdownMenuItem @click="handleMarketAction('test', row)">{{ t('test') }}</DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem class="text-destructive" @click="handleMarketAction('delete', row)">{{ t('delete') }}</DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  </template>
                </TableBody>
              </Table>
            </div>
          </div>

          <!-- Services panel -->
          <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
            <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--color-line)] flex-wrap gap-2">
              <span class="font-medium text-sm text-[var(--color-text)]">{{ t('aggregate_service_list') }}</span>
              <div class="flex gap-2 items-center">
                <div class="relative">
                  <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--color-text-tertiary)]" />
                  <Input v-model="serviceQuery" :placeholder="t('search_placeholder_mcp')" class="pl-8 h-8 w-52" @keyup.enter="loadServices(1)" />
                </div>
                <Button variant="outline" size="sm" @click="loadServices(servicePage)"><RefreshCw class="w-3.5 h-3.5" /></Button>
              </div>
            </div>
            <div class="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{{ t('service') }}</TableHead>
                    <TableHead class="w-32">{{ t('source_market') }}</TableHead>
                    <TableHead>Service ID</TableHead>
                    <TableHead class="w-20 text-center">{{ t('actions') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-if="servicesLoading">
                    <TableCell colspan="4" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
                  </TableRow>
                  <template v-else>
                    <TableEmpty v-if="!services.length" />
                    <TableRow v-for="row in services" :key="row.service_id">
                      <TableCell class="font-medium text-sm truncate max-w-[200px]" :title="row.name">{{ row.name }}</TableCell>
                      <TableCell class="text-xs text-[var(--color-text-secondary)] truncate max-w-[120px]" :title="row.market_name">{{ row.market_name }}</TableCell>
                      <TableCell class="text-xs font-mono text-[var(--color-text-secondary)] truncate max-w-[180px]" :title="row.service_id">{{ row.service_id }}</TableCell>
                      <TableCell class="text-center">
                        <Button variant="ghost" size="sm" class="h-7 text-xs" @click="loadServiceDetail(row)">{{ t('detail_btn') }}</Button>
                      </TableCell>
                    </TableRow>
                  </template>
                </TableBody>
              </Table>
            </div>
            <div class="flex items-center justify-between px-4 py-2 border-t border-[var(--color-line)] text-sm text-[var(--color-text-secondary)]">
              <span>{{ serviceTotal }} {{ t('total_items') }}</span>
              <div class="flex items-center gap-2">
                <Button variant="outline" size="sm" :disabled="servicePage <= 1" @click="loadServices(servicePage - 1)">{{ t('prev') }}</Button>
                <span>{{ servicePage }} / {{ servicePageCount }}</span>
                <Button variant="outline" size="sm" :disabled="servicePage >= servicePageCount" @click="loadServices(servicePage + 1)">{{ t('next') }}</Button>
              </div>
            </div>
            <div v-if="serviceWarnings.length > 0" class="mx-4 mb-4 rounded-lg border border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-900/20 px-3.5 py-2.5 text-xs text-yellow-800 dark:text-yellow-300">
              <div class="font-medium mb-1">{{ t('partial_market_failed_title') }}</div>
              <div v-for="(warn, idx) in serviceWarnings" :key="idx">{{ warn }}</div>
            </div>
          </div>
        </div>
      </TabsContent>

      <!-- Imported Tab -->
      <TabsContent value="imported" class="mt-4">
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
          <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--color-line)] flex-wrap gap-2">
            <span class="font-medium text-sm text-[var(--color-text)]">{{ t('mcp_market_imported') }}</span>
            <div class="flex gap-2 items-center">
              <div class="relative">
                <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--color-text-tertiary)]" />
                <Input v-model="importedQuery" :placeholder="t('search_name_id_url')" class="pl-8 h-8 w-64" @keyup.enter="loadImportedItems(1)" />
              </div>
              <Button variant="outline" size="sm" @click="loadImportedItems(importedPage)"><RefreshCw class="w-3.5 h-3.5" /></Button>
              <Button size="sm" @click="openCreateImportedDialog"><Plus class="w-3.5 h-3.5 mr-1" />{{ t('new_service') }}</Button>
            </div>
          </div>
          <div class="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('name') }}</TableHead>
                  <TableHead class="w-36">{{ t('transport') }}</TableHead>
                  <TableHead class="min-w-[280px]">URL</TableHead>
                  <TableHead class="w-36">Service ID</TableHead>
                  <TableHead class="w-28">{{ t('tool') }}</TableHead>
                  <TableHead class="w-24">{{ t('provider') }}</TableHead>
                  <TableHead class="w-20">{{ t('enabled') }}</TableHead>
                  <TableHead class="w-36">{{ t('search_update_time') }}</TableHead>
                  <TableHead class="w-20 text-center">{{ t('actions') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="importedLoading">
                  <TableCell colspan="9" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
                </TableRow>
                <template v-else>
                  <TableEmpty v-if="!importedItems.length" />
                  <TableRow v-for="row in importedItems" :key="row.id">
                    <TableCell class="font-medium text-sm truncate max-w-[160px]" :title="row.name">{{ row.name }}</TableCell>
                    <TableCell class="text-sm">{{ row.transport }}</TableCell>
                    <TableCell class="text-xs font-mono text-[var(--color-text-secondary)] truncate max-w-[280px]" :title="row.url">{{ row.url }}</TableCell>
                    <TableCell class="text-xs font-mono text-[var(--color-text-secondary)] truncate max-w-[140px]" :title="row.service_id">{{ row.service_id }}</TableCell>
                    <TableCell><span :class="badgeClass(row.allowed_tools?.length ? 'yellow' : 'gray')">{{ row.allowed_tools?.length ? t('selected_count_items', { count: row.allowed_tools.length }) : t('all_tools') }}</span></TableCell>
                    <TableCell><span :class="badgeClass('blue')">{{ row.provider_id || '-' }}</span></TableCell>
                    <TableCell><span :class="badgeClass(row.enabled ? 'green' : 'gray')">{{ row.enabled ? t('enabled') : t('disable') }}</span></TableCell>
                    <TableCell class="text-xs text-[var(--color-text-secondary)]">{{ row.updated_at }}</TableCell>
                    <TableCell class="text-center">
                      <DropdownMenu>
                        <DropdownMenuTrigger as-child>
                          <Button variant="ghost" size="icon" class="h-8 w-8"><MoreHorizontal class="h-4 w-4" /></Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem @click="openEditImportedDialog(row)">{{ t('edit') }}</DropdownMenuItem>
                          <DropdownMenuItem @click="openImportedToolsDialog(row)">{{ t('tool_selection') }}</DropdownMenuItem>
                          <DropdownMenuItem @click="toggleImportedEnabled(row)">{{ row.enabled ? t('disable') : t('enabled') }}</DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem class="text-destructive" @click="deleteImportedItem(row)">{{ t('delete') }}</DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                </template>
              </TableBody>
            </Table>
          </div>
          <div class="flex items-center justify-between px-4 py-2 border-t border-[var(--color-line)] text-sm text-[var(--color-text-secondary)]">
            <span>{{ importedTotal }} {{ t('total_items') }}</span>
            <div class="flex items-center gap-2">
              <Button variant="outline" size="sm" :disabled="importedPage <= 1" @click="loadImportedItems(importedPage - 1)">{{ t('prev') }}</Button>
              <span>{{ importedPage }} / {{ importedPageCount }}</span>
              <Button variant="outline" size="sm" :disabled="importedPage >= importedPageCount" @click="loadImportedItems(importedPage + 1)">{{ t('next') }}</Button>
            </div>
          </div>
        </div>
      </TabsContent>
    </Tabs>

    <!-- Service Detail Dialog -->
    <Dialog v-model:open="detailDialogVisible">
      <DialogContent class="max-w-[900px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('service_detail_title') }}</DialogTitle>
        </DialogHeader>
        <div>
          <div v-if="detailLoading" class="py-12 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
          <div v-else-if="!serviceDetail" class="py-12 text-center text-sm text-[var(--color-text-secondary)]">{{ t('no_service_detail') }}</div>
          <template v-else>
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2 mb-3 text-sm">
              <div><strong class="text-[var(--color-text)]">{{ t('service_label') }}</strong> <span class="text-[var(--color-text-secondary)]">{{ serviceDetail.name || '-' }}</span></div>
              <div><strong class="text-[var(--color-text)]">{{ t('source_market_label') }}</strong> <span class="text-[var(--color-text-secondary)]">{{ serviceDetail.market_name || '-' }}</span></div>
              <div><strong class="text-[var(--color-text)]">Service ID</strong> <span class="text-[var(--color-text-secondary)]">{{ serviceDetail.service_id || '-' }}</span></div>
            </div>
            <p v-if="serviceDetail.description" class="mb-3 text-sm text-[var(--color-text-secondary)] leading-relaxed">{{ serviceDetail.description }}</p>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('resource_name_col') }}</TableHead>
                  <TableHead class="w-36">{{ t('transport') }}</TableHead>
                  <TableHead>URL</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableEmpty v-if="!serviceDetail.endpoints?.length" />
                <TableRow v-for="ep in (serviceDetail.endpoints || [])" :key="ep.name">
                  <TableCell class="text-sm font-medium">{{ ep.name }}</TableCell>
                  <TableCell class="text-sm">{{ ep.transport }}</TableCell>
                  <TableCell class="text-xs font-mono text-[var(--color-text-secondary)] truncate max-w-[360px]" :title="ep.url">{{ ep.url }}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </template>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="detailDialogVisible = false">{{ t('close') }}</Button>
          <Button :disabled="detailImporting || !serviceDetail" @click="importFromDetail">{{ t('import_apply_hot') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Market Form Dialog -->
    <Dialog v-model:open="marketDialogVisible">
      <DialogContent class="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ editingMarket ? t('edit_mcp_market') : t('add_mcp_market') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-3 py-1">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('provider') }}</label>
            <Select v-model="marketForm.provider_id" @update:model-value="handleProviderChange">
              <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="provider in selectableProviderOptions" :key="provider.id" :value="provider.id">{{ provider.name }}</SelectItem>
              </SelectContent>
            </Select>
            <p v-if="currentProvider?.description" class="text-xs text-[var(--color-text-tertiary)] leading-relaxed">{{ currentProvider.description }}</p>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('name') }} <span class="text-destructive">*</span></label>
            <Input v-model="marketForm.name" :placeholder="t('mcp_market_name_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('catalog_url') }} <span class="text-destructive">*</span></label>
            <Input v-model="marketForm.catalog_url" placeholder="https://example.com/api/services" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('detail_url_template_label') }}</label>
            <Input v-model="marketForm.detail_url_template" :placeholder="t('detail_url_template_ph')" />
          </div>
          <div class="flex items-center gap-3">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('enabled') }}</label>
            <Switch v-model:checked="marketForm.enabled" />
          </div>
          <div class="border-t border-[var(--color-line)] pt-3 mt-1">
            <p class="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wide mb-2">{{ t('auth_config_divider') }}</p>
            <div class="grid gap-1.5">
              <label class="text-sm font-medium text-[var(--color-text)]">Token</label>
              <Input v-model="marketForm.auth.token" type="password"
                :placeholder="editingMarket ? t('token_keep_current', { current: editingMarket.token_mask || t('not_set') }) : t('enter_modao_token')" />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="marketDialogVisible = false">{{ t('cancel') }}</Button>
          <Button :disabled="marketSaving" @click="saveMarket">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Imported Service Form Dialog -->
    <Dialog v-model:open="importedDialogVisible">
      <DialogContent class="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ editingImported ? t('edit_import_service') : t('add_import_service') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-3 py-1">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('name') }} <span class="text-destructive">*</span></label>
            <Input v-model="importedForm.name" :placeholder="t('service_display_name_ph')" />
          </div>
          <div class="flex items-center gap-3">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('enabled') }}</label>
            <Switch v-model:checked="importedForm.enabled" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('transport') }} <span class="text-destructive">*</span></label>
            <Select v-model="importedForm.transport">
              <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="sse">SSE</SelectItem>
                <SelectItem value="streamablehttp">StreamableHTTP</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">URL <span class="text-destructive">*</span></label>
            <Input v-model="importedForm.url" placeholder="https://example.com/mcp" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('source_market') }}</label>
            <select v-model="importedForm.market_id" class="h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring">
              <option :value="null">{{ t('optional_label') }}</option>
              <option v-for="item in markets" :key="item.id" :value="item.id">{{ item.name }}</option>
            </select>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('provider') }}</label>
            <Input v-model="importedForm.provider_id" :placeholder="t('provider_id_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">Service ID</label>
            <Input v-model="importedForm.service_id" :placeholder="t('service_id_upstream_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('service_name_label') }}</label>
            <Input v-model="importedForm.service_name" :placeholder="t('service_name_upstream_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">Headers (JSON)</label>
            <textarea v-model="importedHeadersText" rows="4" :placeholder="t('headers_json_ph')"
              class="w-full dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none font-mono" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="importedDialogVisible = false">{{ t('cancel') }}</Button>
          <Button :disabled="importedSaving" @click="saveImportedItem">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Tool Selection Dialog -->
    <Dialog v-model:open="importedToolsDialogVisible">
      <DialogContent class="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ toolDialogTitle }}</DialogTitle>
        </DialogHeader>
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-muted)] p-4">
          <div class="flex items-start justify-between gap-4">
            <div class="flex flex-col gap-1.5">
              <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('tool_access_policy') }}</span>
              <span class="text-xs text-[var(--color-text-secondary)] leading-relaxed">{{ t('empty_tools_allowed') }}</span>
            </div>
            <div class="flex items-center gap-2.5 shrink-0">
              <span :class="badgeClass(importedToolMode === 'all' ? 'gray' : 'yellow')">
                {{ importedToolMode === 'all' ? t('all_tools') : t('selected_count_items', { count: importedSelectedTools.length }) }}
              </span>
              <Button variant="outline" size="sm" :disabled="importedToolsLoading" @click="refreshImportedTools">{{ t('probe_tools') }}</Button>
            </div>
          </div>
          <div class="flex gap-1.5 mt-3.5">
            <Button :variant="importedToolMode === 'all' ? 'default' : 'outline'" size="sm" @click="handleImportedToolModeChange('all')">{{ t('all_tools') }}</Button>
            <Button :variant="importedToolMode === 'selected' ? 'default' : 'outline'" size="sm" @click="handleImportedToolModeChange('selected')">{{ t('specify_tool_option') }}</Button>
          </div>
          <template v-if="importedToolMode === 'selected'">
            <div class="relative mt-3.5 max-w-xs">
              <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--color-text-tertiary)]" />
              <Input v-model="importedToolQuery" :placeholder="t('search_tool_ph')" class="pl-8 h-8" />
            </div>
            <div v-if="filteredImportedToolOptions.length === 0" class="mt-3.5 border border-dashed border-[var(--color-line)] rounded-xl p-4 text-center text-sm text-[var(--color-text-tertiary)]">
              {{ importedToolOptions.length === 0 ? t('no_tools_probed') : t('no_matching_tools') }}
            </div>
            <div v-else class="mt-3.5 grid gap-2.5" style="grid-template-columns: repeat(2, minmax(0, 1fr))">
              <label
                v-for="tool in filteredImportedToolOptions"
                :key="tool.name"
                :class="['flex items-start gap-2.5 rounded-xl border px-3.5 py-2.5 cursor-pointer transition-colors',
                  importedSelectedTools.includes(tool.name)
                    ? 'border-[var(--color-primary)] bg-blue-50/60 dark:bg-blue-900/15'
                    : 'border-[var(--color-line)] bg-[var(--color-surface)] hover:border-[var(--color-primary)]/40']"
              >
                <input type="checkbox" :checked="importedSelectedTools.includes(tool.name)"
                  @change="toggleImportedTool(tool.name)" class="mt-0.5 accent-[var(--color-primary)]" />
                <div class="flex flex-col gap-0.5 min-w-0">
                  <span class="text-xs font-semibold text-[var(--color-text)] truncate">{{ tool.name }}</span>
                  <span class="text-xs text-[var(--color-text-tertiary)] line-clamp-2">{{ tool.description || t('no_description') }}</span>
                </div>
              </label>
            </div>
          </template>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="importedToolsDialogVisible = false">{{ t('cancel') }}</Button>
          <Button :disabled="importedSaving" @click="saveImportedToolSelection">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
