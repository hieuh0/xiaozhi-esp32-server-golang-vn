<template>
  <div class="mcp-market-page">
    <el-tabs v-model="activeTab" class="market-tabs">
      <el-tab-pane name="discover">
        <template #label>
          <span>{{ t('market_discovery') }}</span>
        </template>

        <el-row :gutter="16">
          <el-col :xs="24" :lg="11">
            <el-card shadow="never" class="panel-card">
              <template #header>
                <div class="panel-header">
                  <span>{{ t('mcp_market_title') }}</span>
                  <div>
                    <el-button type="primary" size="small" @click="openCreateDialog">{{ t('add_connection') }}</el-button>
                    <el-button size="small" @click="loadMarkets">
                      <el-icon><Refresh /></el-icon>
                    </el-button>
                  </div>
                </div>
              </template>

              <el-table :data="markets" stripe v-loading="marketsLoading" height="560">
                <el-table-column prop="name" :label="t('name')" min-width="140" />
                <el-table-column prop="provider_id" :label="t('provider')" width="130">
                  <template #default="{ row }">
                    <el-tag size="small">{{ row.provider_id || 'generic' }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="catalog_url" :label="t('catalog_url')" min-width="220" show-overflow-tooltip />
                <el-table-column :label="t('auth')" width="120">
                  <template #default="{ row }">
                    <el-tag size="small" :type="row.has_token ? 'success' : 'info'">
                      {{ row.auth_type || 'none' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('status')" width="90">
                  <template #default="{ row }">
                    <el-tag size="small" :type="row.enabled ? 'success' : 'info'">
                      {{ row.enabled ? t('enabled') : t('disable') }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="t('actions')" width="96" fixed="right">
                  <template #default="{ row }">
                    <el-dropdown trigger="click" @command="(cmd) => handleMarketAction(cmd, row)">
                      <el-button link type="primary" class="market-action-btn">
                        <el-icon><MoreFilled /></el-icon>
                      </el-button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="edit">{{ t('edit') }}</el-dropdown-item>
                          <el-dropdown-item command="test">{{ t('test') }}</el-dropdown-item>
                          <el-dropdown-item command="delete" divided>{{ t('delete') }}</el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </el-col>

          <el-col :xs="24" :lg="13">
            <el-card shadow="never" class="panel-card">
              <template #header>
                <div class="panel-header">
                  <span>{{ t('aggregate_service_list') }}</span>
                  <div class="search-actions">
                    <el-input
                      v-model="serviceQuery"
                      :placeholder="t('search_placeholder_mcp')"
                      clearable
                      size="small"
                      style="width: 240px"
                      @keyup.enter="loadServices(1)"
                    >
                      <template #append>
                        <el-button @click="loadServices(1)">
                          <el-icon><Search /></el-icon>
                        </el-button>
                      </template>
                    </el-input>
                    <el-button size="small" @click="loadServices(servicePage)">
                      <el-icon><Refresh /></el-icon>
                    </el-button>
                  </div>
                </div>
              </template>

              <el-table :data="services" stripe v-loading="servicesLoading" height="500">
                <el-table-column prop="name" :label="t('service')" min-width="180" show-overflow-tooltip />
                <el-table-column prop="market_name" :label="t('source_market')" min-width="120" show-overflow-tooltip />
                <el-table-column prop="service_id" label="Service ID" min-width="180" show-overflow-tooltip />
                <el-table-column :label="t('actions')" width="90" fixed="right">
                  <template #default="{ row }">
                    <el-button link type="primary" @click.stop="loadServiceDetail(row)">{{ t('detail_btn') }}</el-button>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination-wrap">
                <el-pagination
                  layout="prev, pager, next, total"
                  :current-page="servicePage"
                  :page-size="servicePageSize"
                  :total="serviceTotal"
                  @current-change="loadServices"
                />
              </div>

              <el-alert
                v-if="serviceWarnings.length > 0"
                type="warning"
                :closable="false"
                :title="t('partial_market_failed_title')"
                class="warning-alert"
              >
                <template #default>
                  <div v-for="(warn, idx) in serviceWarnings" :key="idx">{{ warn }}</div>
                </template>
              </el-alert>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane name="imported">
        <template #label>
          <div class="tab-label-with-badge">
            <span>{{ t('mcp_market_imported') }}</span>
            <el-badge :value="importedTotal" :max="999" class="tab-badge" />
          </div>
        </template>

        <el-card shadow="never" class="panel-card">
          <template #header>
            <div class="panel-header">
              <span>{{ t('mcp_market_imported') }}</span>
              <div class="search-actions">
                <el-input
                  v-model="importedQuery"
                  :placeholder="t('search_name_id_url')"
                  clearable
                  size="small"
                  style="width: 320px"
                  @keyup.enter="loadImportedItems(1)"
                >
                  <template #append>
                    <el-button @click="loadImportedItems(1)">
                      <el-icon><Search /></el-icon>
                    </el-button>
                  </template>
                </el-input>
                <el-button size="small" @click="loadImportedItems(importedPage)">
                  <el-icon><Refresh /></el-icon>
                </el-button>
                <el-button type="primary" size="small" @click="openCreateImportedDialog">{{ t('new_service') }}</el-button>
              </div>
            </div>
          </template>

          <el-table :data="importedItems" stripe v-loading="importedLoading" height="560">
            <el-table-column prop="name" :label="t('name')" min-width="160" show-overflow-tooltip />
            <el-table-column prop="transport" :label="t('transport')" width="140" />
            <el-table-column prop="url" label="URL" min-width="320" show-overflow-tooltip />
            <el-table-column prop="service_id" label="Service ID" min-width="180" show-overflow-tooltip />
            <el-table-column :label="t('tool')" width="120">
              <template #default="{ row }">
                <el-tag size="small" :type="row.allowed_tools?.length ? 'warning' : 'info'">
                  {{ row.allowed_tools?.length ? t('selected_count_items', { count: row.allowed_tools.length }) : t('all_tools') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="provider_id" :label="t('provider')" width="120">
              <template #default="{ row }">
                <el-tag size="small">{{ row.provider_id || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="enabled" :label="t('enabled')" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.enabled ? 'success' : 'info'">
                  {{ row.enabled ? t('enabled') : t('disable') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="updated_at" :label="t('search_update_time')" width="180" />
            <el-table-column :label="t('actions')" width="280" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openEditImportedDialog(row)">{{ t('edit') }}</el-button>
                <el-button link type="primary" @click="openImportedToolsDialog(row)">{{ t('tool_selection') }}</el-button>
                <el-button link :type="row.enabled ? 'warning' : 'success'" @click="toggleImportedEnabled(row)">
                  {{ row.enabled ? t('disable') : t('enabled') }}
                </el-button>
                <el-button link type="danger" @click="deleteImportedItem(row)">{{ t('delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="pagination-wrap">
            <el-pagination
              layout="prev, pager, next, total"
              :current-page="importedPage"
              :page-size="importedPageSize"
              :total="importedTotal"
              @current-change="loadImportedItems"
            />
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="detailDialogVisible" :title="t('service_detail_title')" width="900px">
      <div v-loading="detailLoading">
        <el-empty v-if="!serviceDetail && !detailLoading" :description="t('no_service_detail')" />
        <template v-else-if="serviceDetail">
          <div class="detail-grid">
            <div><strong>{{ t('service_label') }}</strong>{{ serviceDetail.name || '-' }}</div>
            <div><strong>{{ t('source_market_label') }}</strong>{{ serviceDetail.market_name || '-' }}</div>
            <div><strong>Service ID：</strong>{{ serviceDetail.service_id || '-' }}</div>
          </div>
          <div v-if="serviceDetail.description" class="detail-desc">{{ serviceDetail.description }}</div>
          <el-table :data="serviceDetail.endpoints || []" size="small" stripe>
            <el-table-column prop="name" :label="t('resource_name_col')" min-width="120" show-overflow-tooltip />
            <el-table-column prop="transport" :label="t('transport')" width="140" />
            <el-table-column prop="url" label="URL" min-width="360" show-overflow-tooltip />
          </el-table>
        </template>
      </div>
      <template #footer>
        <el-button @click="detailDialogVisible = false">{{ t('close') }}</el-button>
        <el-button type="primary" :loading="detailImporting" :disabled="!serviceDetail" @click="importFromDetail">
          {{ t('import_apply_hot') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="importedDialogVisible" :title="editingImported ? t('edit_import_service') : t('add_import_service')" width="700px">
      <el-form ref="importedFormRef" :model="importedForm" :rules="importedRules" label-width="120px">
        <el-form-item :label="t('name')" prop="name">
          <el-input v-model="importedForm.name" :placeholder="t('service_display_name_ph')" />
        </el-form-item>
        <el-form-item :label="t('enabled')">
          <el-switch v-model="importedForm.enabled" />
        </el-form-item>
        <el-form-item :label="t('transport')" prop="transport">
          <el-select v-model="importedForm.transport" style="width: 100%">
            <el-option label="SSE" value="sse" />
            <el-option label="StreamableHTTP" value="streamablehttp" />
          </el-select>
        </el-form-item>
        <el-form-item label="URL" prop="url">
          <el-input v-model="importedForm.url" placeholder="https://example.com/mcp" />
        </el-form-item>
        <el-form-item :label="t('source_market')">
          <el-select v-model="importedForm.market_id" clearable filterable style="width: 100%" :placeholder="t('optional_label')">
            <el-option v-for="item in markets" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('provider')">
          <el-input v-model="importedForm.provider_id" :placeholder="t('provider_id_ph')" />
        </el-form-item>
        <el-form-item label="Service ID">
          <el-input v-model="importedForm.service_id" :placeholder="t('service_id_upstream_ph')" />
        </el-form-item>
        <el-form-item :label="t('service_name_label')">
          <el-input v-model="importedForm.service_name" :placeholder="t('service_name_upstream_ph')" />
        </el-form-item>
        <el-form-item label="Headers(JSON)">
          <el-input
            v-model="importedHeadersText"
            type="textarea"
            :rows="4"
            :placeholder="t('headers_json_ph')"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="importedDialogVisible = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="importedSaving" @click="saveImportedItem">{{ t('save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="importedToolsDialogVisible" :title="toolDialogTitle" width="760px">
      <div class="tool-selector-card">
        <div class="tool-selector-header">
          <div class="tool-selector-meta">
            <span class="tool-selector-title">{{ t('tool_access_policy') }}</span>
            <span class="tool-selector-tip">{{ t('empty_tools_allowed') }}</span>
          </div>
          <div class="tool-selector-actions">
            <el-tag size="small" :type="importedToolMode === 'all' ? 'info' : 'warning'">
              {{ importedToolMode === 'all' ? t('all_tools') : t('selected_count_items', { count: importedSelectedTools.length }) }}
            </el-tag>
            <el-button size="small" :loading="importedToolsLoading" @click="refreshImportedTools">{{ t('probe_tools') }}</el-button>
          </div>
        </div>

        <el-radio-group v-model="importedToolMode" size="small" class="tool-mode-group" @change="handleImportedToolModeChange">
          <el-radio-button label="all">{{ t('all_tools') }}</el-radio-button>
          <el-radio-button label="selected">{{ t('specify_tool_option') }}</el-radio-button>
        </el-radio-group>

        <template v-if="importedToolMode === 'selected'">
          <div class="tool-picker-search">
            <el-input v-model="importedToolQuery" clearable :placeholder="t('search_tool_ph')">
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </div>

          <div v-if="filteredImportedToolOptions.length === 0" class="tool-picker-empty">
            {{ importedToolOptions.length === 0 ? t('no_tools_probed') : t('no_matching_tools') }}
          </div>
          <el-checkbox-group v-else v-model="importedSelectedTools" class="tool-grid">
            <el-checkbox
              v-for="tool in filteredImportedToolOptions"
              :key="tool.name"
              :label="tool.name"
              border
              class="tool-tile"
            >
              <div class="tool-tile-body">
                <span class="tool-tile-name">{{ tool.name }}</span>
                <span class="tool-tile-desc">{{ tool.description || t('no_description') }}</span>
              </div>
            </el-checkbox>
          </el-checkbox-group>
        </template>
      </div>

      <template #footer>
        <el-button @click="importedToolsDialogVisible = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="importedSaving" @click="saveImportedToolSelection">{{ t('save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="marketDialogVisible" :title="editingMarket ? t('edit_mcp_market') : t('add_mcp_market')" width="640px">
      <el-form ref="marketFormRef" :model="marketForm" :rules="marketRules" label-width="130px">
        <el-form-item :label="t('provider')">
          <el-select v-model="marketForm.provider_id" style="width: 100%" @change="handleProviderChange">
            <el-option v-for="provider in selectableProviderOptions" :key="provider.id" :label="provider.name" :value="provider.id" />
          </el-select>
          <div v-if="currentProvider?.description" class="provider-desc">{{ currentProvider.description }}</div>
        </el-form-item>
        <el-form-item :label="t('name')" prop="name">
          <el-input v-model="marketForm.name" :placeholder="t('mcp_market_name_ph')" />
        </el-form-item>
        <el-form-item :label="t('catalog_url')" prop="catalog_url">
          <el-input v-model="marketForm.catalog_url" placeholder="https://example.com/api/services" />
        </el-form-item>
        <el-form-item :label="t('detail_url_template_label')" prop="detail_url_template">
          <el-input v-model="marketForm.detail_url_template" :placeholder="t('detail_url_template_ph')" />
        </el-form-item>
        <el-form-item :label="t('enabled')">
          <el-switch v-model="marketForm.enabled" />
        </el-form-item>

        <el-divider>{{ t('auth_config_divider') }}</el-divider>
        <el-form-item label="Token">
          <el-input
            v-model="marketForm.auth.token"
            :placeholder="editingMarket ? t('token_keep_current', { current: editingMarket.token_mask || t('not_set') }) : t('enter_modao_token')"
            show-password
            clearable
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="marketDialogVisible = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="marketSaving" @click="saveMarket">{{ t('save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Search, MoreFilled } from '@element-plus/icons-vue'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const activeTab = ref('discover')

const markets = ref([])
const marketsLoading = ref(false)
const providerOptions = ref([])
const marketDialogVisible = ref(false)
const marketSaving = ref(false)
const editingMarket = ref(null)
const marketFormRef = ref()

const marketForm = reactive({
  name: '',
  provider_id: 'modelscope',
  catalog_url: '',
  detail_url_template: '',
  enabled: true,
  auth: {
    type: 'bearer',
    token: '',
    header_name: 'Authorization'
  }
})

const marketRules = {
  name: [{ required: true, message: t('enter_name'), trigger: 'blur' }],
  catalog_url: [{ required: true, message: t('enter_directory_url'), trigger: 'blur' }]
}

const selectableProviderOptions = computed(() => {
  return providerOptions.value.filter(item => item.id !== 'generic')
})

const currentProvider = computed(() => {
  return selectableProviderOptions.value.find(item => item.id === marketForm.provider_id) || null
})

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
const importedFormRef = ref()
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
  name: '',
  enabled: true,
  transport: 'streamablehttp',
  url: '',
  allowed_tools: [],
  market_id: null,
  provider_id: '',
  service_id: '',
  service_name: ''
})

const importedRules = {
  name: [{ required: true, message: t('enter_name'), trigger: 'blur' }],
  transport: [{ required: true, message: t('select_transport_type'), trigger: 'change' }],
  url: [{ required: true, message: t('enter_url'), trigger: 'blur' }]
}

const toolDialogTitle = computed(() => {
  return importedToolTarget.value ? t('tool_selection_title', { name: importedToolTarget.value.name }) : t('tool_selection')
})

const filteredImportedToolOptions = computed(() => {
  const query = importedToolQuery.value.trim().toLowerCase()
  if (!query) return importedToolOptions.value
  return importedToolOptions.value.filter((tool) => {
    return tool.name.toLowerCase().includes(query) || (tool.description || '').toLowerCase().includes(query)
  })
})

const getDefaultProviderId = () => {
  if (selectableProviderOptions.value.length === 0) return 'modelscope'
  if (selectableProviderOptions.value.some(item => item.id === 'modelscope')) return 'modelscope'
  return selectableProviderOptions.value[0].id
}

const loadProviders = async () => {
  try {
    const resp = await api.get('/admin/mcp-market/providers')
    providerOptions.value = resp.data.data || []
    if (!marketForm.provider_id) {
      marketForm.provider_id = getDefaultProviderId()
    }
    if (!selectableProviderOptions.value.some(item => item.id === marketForm.provider_id)) {
      marketForm.provider_id = getDefaultProviderId()
    }
  } catch (error) {
    providerOptions.value = [{ id: 'modelscope', name: t('modelscope') }]
    marketForm.provider_id = marketForm.provider_id || 'modelscope'
    ElMessage.error(error.response?.data?.error || t('load_provider_failed'))
  }
}

const applyProviderPreset = (providerId, force = false) => {
  const provider = selectableProviderOptions.value.find(item => item.id === providerId)
  if (!provider) return

  if (force || !marketForm.catalog_url) {
    marketForm.catalog_url = provider.catalog_url || ''
  }
  if (force || !marketForm.detail_url_template) {
    marketForm.detail_url_template = provider.detail_url_template || ''
  }
  if (force || !marketForm.auth.type) {
    marketForm.auth.type = 'bearer'
  }
  marketForm.auth.header_name = 'Authorization'

  if (force) {
    marketForm.auth.token = ''
  }

  if (!editingMarket.value && (force || !marketForm.name) && provider.id === 'modelscope') {
    marketForm.name = t('modao_mcp_market')
  }
}

const handleProviderChange = (providerId) => {
  applyProviderPreset(providerId, true)
}

const handleMarketAction = async (command, row) => {
  if (command === 'edit') {
    openEditDialog(row)
    return
  }
  if (command === 'test') {
    await testMarket(row)
    return
  }
  if (command === 'delete') {
    await deleteMarket(row)
  }
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
  marketForm.name = ''
  marketForm.provider_id = getDefaultProviderId()
  marketForm.catalog_url = ''
  marketForm.detail_url_template = ''
  marketForm.enabled = true
  marketForm.auth.type = 'bearer'
  marketForm.auth.token = ''
  marketForm.auth.header_name = 'Authorization'
}

const openCreateDialog = () => {
  editingMarket.value = null
  resetMarketForm()
  applyProviderPreset(marketForm.provider_id, true)
  marketDialogVisible.value = true
}

const openEditDialog = (row) => {
  editingMarket.value = row
  marketForm.name = row.name
  const rowProviderId = row.provider_id || getDefaultProviderId()
  marketForm.provider_id = selectableProviderOptions.value.some(item => item.id === rowProviderId)
    ? rowProviderId
    : getDefaultProviderId()
  marketForm.catalog_url = row.catalog_url
  marketForm.detail_url_template = row.detail_url_template || ''
  marketForm.enabled = !!row.enabled
  marketForm.auth.type = 'bearer'
  marketForm.auth.header_name = 'Authorization'
  marketForm.auth.token = ''
  marketDialogVisible.value = true
}

const saveMarket = async () => {
  if (!marketFormRef.value) return
  const valid = await marketFormRef.value.validate().catch(() => false)
  if (!valid) return

  const payload = {
    name: marketForm.name,
    provider_id: marketForm.provider_id,
    catalog_url: marketForm.catalog_url,
    detail_url_template: marketForm.detail_url_template,
    enabled: marketForm.enabled,
    auth: {
      type: 'bearer',
      token: marketForm.auth.token,
      header_name: 'Authorization'
    }
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
    marketDialogVisible.value = false
    await loadMarkets()
    await loadServices(1)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('save_failed'))
  } finally {
    marketSaving.value = false
  }
}

const deleteMarket = async (row) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_mcp_market', { name: row.name }), t('hint'), {
      type: 'warning',
      confirmButtonText: t('delete'),
      cancelButtonText: t('cancel')
    })
    await api.delete(`/admin/mcp-markets/${row.id}`)
    ElMessage.success(t('delete_success'))
    await loadMarkets()
    await loadServices(1)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || t('delete_failed'))
    }
  }
}

const testMarket = async (row) => {
  try {
    const resp = await api.post(`/admin/mcp-markets/${row.id}/test`)
    const count = resp.data?.data?.service_count ?? 0
    ElMessage.success(t('connection_success_count', { count }))
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('connection_test_failed'))
  }
}

const loadServices = async (page = 1) => {
  servicePage.value = page
  servicesLoading.value = true
  try {
    const resp = await api.get('/admin/mcp-market/services', {
      params: {
        q: serviceQuery.value,
        page: servicePage.value,
        page_size: servicePageSize.value
      }
    })
    const data = resp.data.data || {}
    services.value = data.items || []
    serviceTotal.value = data.total || 0
    serviceWarnings.value = data.warnings || []
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('load_agg_service_failed'))
  } finally {
    servicesLoading.value = false
  }
}

const loadServiceDetail = async (row) => {
  detailDialogVisible.value = true
  detailLoading.value = true
  serviceDetail.value = null
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
  if (!row?.market_id || !row?.service_id) {
    ElMessage.error(t('service_id_missing'))
    return
  }

  detailImporting.value = true
  try {
    const payload = {
      market_id: row.market_id,
      service_id: row.service_id,
      name_override: ''
    }
    const resp = await api.post('/admin/mcp-market/import', payload)
    const result = resp.data.data || {}
    ElMessage.success(t('import_success_count', { count: result.imported_count || 0 }))
    await loadServices(servicePage.value)
    await loadImportedItems(1)
    detailDialogVisible.value = false
    activeTab.value = 'imported'
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('import_failed'))
  } finally {
    detailImporting.value = false
  }
}

const loadImportedItems = async (page = 1) => {
  importedPage.value = page
  importedLoading.value = true
  try {
    const resp = await api.get('/admin/mcp-market/imported-services', {
      params: {
        q: importedQuery.value,
        page: importedPage.value,
        page_size: importedPageSize.value
      }
    })
    const data = resp.data.data || {}
    importedItems.value = data.items || []
    importedTotal.value = data.total || 0
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
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error(t('headers_must_json'))
    }
    return parsed
  } catch (error) {
    throw new Error(t('headers_invalid_json'))
  }
}

const resetImportedForm = () => {
  importedForm.name = ''
  importedForm.enabled = true
  importedForm.transport = 'streamablehttp'
  importedForm.url = ''
  importedForm.market_id = null
  importedForm.provider_id = ''
  importedForm.service_id = ''
  importedForm.service_name = ''
  importedHeadersText.value = ''
}

const mergeImportedToolOptions = (tools = [], selected = []) => {
  const merged = new Map()

  ;(tools || []).forEach((tool) => {
    if (!tool?.name) return
    merged.set(tool.name, {
      name: tool.name,
      description: tool.description || ''
    })
  })

  ;(selected || []).forEach((name) => {
    if (!name || merged.has(name)) return
    merged.set(name, {
      name,
      description: t('current_config_selected')
    })
  })

  importedToolOptions.value = Array.from(merged.values()).sort((a, b) => a.name.localeCompare(b.name))
}

const loadImportedToolOptions = async (serviceId) => {
  if (!serviceId) {
    mergeImportedToolOptions([], importedSelectedTools.value)
    return
  }

  importedToolsLoading.value = true
  try {
    const resp = await api.get(`/admin/mcp-market/imported-services/${serviceId}/tools`)
    const data = resp.data?.data || {}
    mergeImportedToolOptions(data.tools || [], importedSelectedTools.value)
  } catch (error) {
    mergeImportedToolOptions([], importedSelectedTools.value)
    ElMessage.error(error.response?.data?.error || t('load_tool_list_failed'))
  } finally {
    importedToolsLoading.value = false
  }
}

const openCreateImportedDialog = () => {
  editingImported.value = null
  resetImportedForm()
  importedDialogVisible.value = true
}

const openEditImportedDialog = (row) => {
  editingImported.value = row
  importedForm.name = row.name || ''
  importedForm.enabled = !!row.enabled
  importedForm.transport = row.transport || 'streamablehttp'
  importedForm.url = row.url || ''
  importedForm.market_id = row.market_id || null
  importedForm.provider_id = row.provider_id || ''
  importedForm.service_id = row.service_id || ''
  importedForm.service_name = row.service_name || ''
  importedHeadersText.value = row.headers ? JSON.stringify(row.headers, null, 2) : ''
  importedDialogVisible.value = true
}

const syncImportedToolMode = (selected = importedSelectedTools.value) => {
  importedToolMode.value = selected.length > 0 ? 'selected' : 'all'
}

const handleImportedToolModeChange = (mode) => {
  importedToolQuery.value = ''
  if (mode === 'all') {
    importedSelectedTools.value = []
  }
}

const openImportedToolsDialog = async (row) => {
  importedToolTarget.value = row
  importedSelectedTools.value = Array.isArray(row.allowed_tools) ? [...row.allowed_tools] : []
  importedToolQuery.value = ''
  syncImportedToolMode(importedSelectedTools.value)
  mergeImportedToolOptions([], importedSelectedTools.value)
  importedToolsDialogVisible.value = true
  await loadImportedToolOptions(row.id)
}

const refreshImportedTools = async () => {
  if (!importedToolTarget.value?.id) {
    ElMessage.warning(t('select_imported_service'))
    return
  }
  await loadImportedToolOptions(importedToolTarget.value.id)
}

const saveImportedItem = async () => {
  if (!importedFormRef.value) return
  const valid = await importedFormRef.value.validate().catch(() => false)
  if (!valid) return

  let headers = null
  try {
    headers = parseImportedHeaders()
  } catch (error) {
    ElMessage.error(error.message)
    return
  }

  const payload = {
    name: importedForm.name,
    enabled: importedForm.enabled,
    transport: importedForm.transport,
    url: importedForm.url,
    headers,
    allowed_tools: editingImported.value?.allowed_tools || [],
    market_id: importedForm.market_id || null,
    provider_id: importedForm.provider_id,
    service_id: importedForm.service_id,
    service_name: importedForm.service_name
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
    importedDialogVisible.value = false
    await loadImportedItems(importedPage.value)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('save_failed'))
  } finally {
    importedSaving.value = false
  }
}

const saveImportedToolSelection = async () => {
  if (!importedToolTarget.value) return
  if (importedToolMode.value === 'selected' && importedSelectedTools.value.length === 0) {
    ElMessage.warning(t('select_at_least_one_tool'))
    return
  }

  const row = importedToolTarget.value
  const payload = {
    name: row.name,
    enabled: row.enabled,
    transport: row.transport,
    url: row.url,
    headers: row.headers || null,
    allowed_tools: importedToolMode.value === 'all' ? [] : importedSelectedTools.value,
    market_id: row.market_id || null,
    provider_id: row.provider_id || '',
    service_id: row.service_id || '',
    service_name: row.service_name || ''
  }

  importedSaving.value = true
  try {
    await api.put(`/admin/mcp-market/imported-services/${row.id}`, payload)
    ElMessage.success(t('tool_strategy_updated'))
    importedToolsDialogVisible.value = false
    importedToolTarget.value = null
    importedToolQuery.value = ''
    await loadImportedItems(importedPage.value)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('save_failed'))
  } finally {
    importedSaving.value = false
  }
}

const toggleImportedEnabled = async (row) => {
  const payload = {
    name: row.name,
    enabled: !row.enabled,
    transport: row.transport,
    url: row.url,
    headers: row.headers || null,
    allowed_tools: row.allowed_tools || [],
    market_id: row.market_id || null,
    provider_id: row.provider_id || '',
    service_id: row.service_id || '',
    service_name: row.service_name || ''
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
      type: 'warning',
      confirmButtonText: t('delete'),
      cancelButtonText: t('cancel')
    })
    await api.delete(`/admin/mcp-market/imported-services/${row.id}`)
    ElMessage.success(t('delete_success'))
    await loadImportedItems(importedPage.value)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || t('delete_failed'))
    }
  }
}

onMounted(async () => {
  await loadProviders()
  await loadMarkets()
  await loadServices(1)
  await loadImportedItems(1)
})
</script>

<style scoped>
.mcp-market-page {
  padding: 20px;
}

.market-tabs {
  --el-tabs-header-height: 44px;
}

.tab-label-with-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.tab-badge {
  line-height: 1;
}

.market-action-btn {
  padding: 0;
  min-width: 22px;
}

.panel-card {
  margin-bottom: 16px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.search-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.pagination-wrap {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
}

.warning-alert {
  margin-top: 12px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(180px, 1fr));
  gap: 8px 12px;
  margin-bottom: 10px;
}

.detail-desc {
  margin-bottom: 12px;
  color: #4b5563;
  line-height: 1.6;
}

.provider-desc {
  margin-top: 6px;
  line-height: 1.5;
  color: #6b7280;
  font-size: 12px;
}

.tool-selector-card {
  border: 1px solid #e5edf5;
  border-radius: 16px;
  padding: 16px;
  background: linear-gradient(180deg, #fbfdff 0%, #f4f8fc 100%);
}

.tool-selector-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.tool-selector-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tool-selector-title {
  color: #172033;
  font-size: 15px;
  font-weight: 600;
}

.tool-selector-tip {
  color: #667085;
  font-size: 12px;
  line-height: 1.5;
}

.tool-selector-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.tool-mode-group {
  margin-top: 14px;
}

.tool-picker-search {
  margin-top: 14px;
  max-width: 320px;
}

.tool-picker-empty {
  margin-top: 14px;
  border: 1px dashed #d6deeb;
  border-radius: 12px;
  padding: 18px 16px;
  color: #667085;
  font-size: 13px;
  text-align: center;
  background: rgba(255, 255, 255, 0.72);
}

.tool-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.tool-tile {
  margin-right: 0;
  min-width: 0;
}

.tool-tile-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  overflow: hidden;
}

.tool-tile-name {
  display: -webkit-box;
  color: #182230;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: break-word;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.tool-tile-desc {
  display: -webkit-box;
  color: #667085;
  font-size: 12px;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: break-word;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

:deep(.tool-grid .el-checkbox.is-bordered) {
  width: 100%;
  height: auto;
  margin-right: 0;
  padding: 12px 14px;
  border-radius: 14px;
  align-items: flex-start;
  background: rgba(255, 255, 255, 0.9);
  border-color: #d7e1ec;
}

:deep(.tool-grid .el-checkbox__label) {
  padding-left: 10px;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

:deep(.tool-grid .el-checkbox.is-bordered.is-checked) {
  background: #eff7ff;
  border-color: #7ab8ff;
  box-shadow: 0 0 0 1px rgba(64, 158, 255, 0.08);
}

@media (max-width: 992px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }

  .search-actions {
    flex-wrap: wrap;
  }

  .panel-header {
    flex-wrap: wrap;
  }

  .tool-selector-header {
    flex-direction: column;
  }

  .tool-selector-actions {
    width: 100%;
    justify-content: space-between;
  }

  .tool-grid {
    grid-template-columns: 1fr;
  }

  .tool-picker-search {
    max-width: none;
  }
}
</style>
