<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button
        type="warning"
        plain
        :loading="testingAll"
        @click="testAllConfigs"
        :disabled="!getEnabledConfigs().length"
      >
        测试全部
      </el-button>
      <el-button type="primary" @click="showDialog = true">
        <el-icon><Plus /></el-icon>
        {{ t('add_config') }}  添加配置
      </el-button>
    </div>

    <el-table :data="configs" style="width: 100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" :label="t('config_name')" />
      <el-table-column prop="config_id" :label="t('config_id')" width="150" />
      <el-table-column prop="provider" :label="t('provider')">
        <template #default="scope">
          {{ scope.row.provider }}
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="启用状态" width="80" align="center">
        <template #default="scope">
          <el-switch 
            v-model="scope.row.enabled" 
            @change="toggleEnable(scope.row)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="is_default" label="默认配置" width="80" align="center">
        <template #default="scope">
          <el-switch 
            v-model="scope.row.is_default" 
            @change="toggleDefault(scope.row)"
            :disabled="scope.row.is_default && getEnabledConfigs().length === 1"
          />
        </template>
      </el-table-column>
      <el-table-column label="测试结果" width="120" align="center">
        <template #default="scope">
          <template v-if="testResults[scope.row.config_id]">
            <el-tooltip v-if="testResults[scope.row.config_id].ok" :content="formatTestResultTip(testResults[scope.row.config_id])" placement="top">
              <span class="test-result test-ok">{{ formatTestResultLabel(testResults[scope.row.config_id]) }}</span>
            </el-tooltip>
            <el-tooltip v-else :content="testResults[scope.row.config_id].message" placement="top" :show-after="200">
              <span class="test-result test-err">错误</span>
            </el-tooltip>
          </template>
          <span v-else class="test-result test-none">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="scope">
          {{ formatDate(scope.row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260">
        <template #default="scope">
          <el-button size="small" @click="editConfig(scope.row)">编辑</el-button>
          <el-button
            size="small"
            type="warning"
            :loading="testingId === scope.row.config_id"
            @click="testConfig(scope.row, 'vad')"
          >
            测试
          </el-button>
          <el-button
            size="small"
            type="danger"
            @click="deleteConfig(scope.row.id)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑配置弹窗 -->
    <el-dialog
      v-model="showDialog"
      :title="editingConfig ? '编辑VAD配置' : '添加VAD配置'"
      width="600px"
      @close="handleDialogClose"
    >
      <VADConfigForm ref="formRef" :model="form" :rules="rules" />
      
      <template #footer>
        <el-button @click="handleDialogClose">取消</el-button>
        <el-button type="warning" plain @click="testCurrentConfig" :loading="testingCurrent">
          测试
        </el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { testSingleConfig, testWithData, parseJsonData } from '../../utils/configTest'
import VADConfigForm from './forms/VADConfigForm.vue'
import { resolveVADProvider } from './forms/configProviderUtils'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const configs = ref([])
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
  provider: 'ten_vad',
  is_default: false,
  enabled: true,
  webrtc_vad: {
    pool_min_size: 5,
    pool_max_size: 1000,
    pool_max_idle: 100,
    vad_sample_rate: 16000,
    vad_mode: 2
  },
  silero_vad: {
    model_path: 'config/models/vad/silero_vad.onnx',
    threshold: 0.5,
    min_silence_duration_ms: 100,
    sample_rate: 16000,
    channels: 1,
    acquire_timeout_ms: 3000
  },
  ten_vad: {
    hop_size: 320,
    threshold: 0.3,
    pool_size: 10,
    acquire_timeout_ms: 3000
  }
})

const rules = {
  name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
  config_id: [{ required: true, message: t('enter_config_id'), trigger: 'blur' }],
  provider: [{ required: true, message: t('select_provider'), trigger: 'change' }],
  'webrtc_vad.pool_min_size': [{ required: true, message: t('enter_min_pool_size'), trigger: 'blur' }],
  'webrtc_vad.pool_max_size': [{ required: true, message: t('enter_max_pool_size'), trigger: 'blur' }],
  'webrtc_vad.pool_max_idle': [{ required: true, message: t('enter_max_idle_connections'), trigger: 'blur' }],
  'webrtc_vad.vad_sample_rate': [{ required: true, message: t('select_vad_sample_rate'), trigger: 'change' }],
  'webrtc_vad.vad_mode': [{ required: true, message: t('select_vad_mode'), trigger: 'change' }],
  'silero_vad.model_path': [{ required: true, message: t('enter_model_path'), trigger: 'blur' }],
  'silero_vad.threshold': [{ required: true, message: t('enter_threshold'), trigger: 'blur' }],
  'silero_vad.min_silence_duration_ms': [{ required: true, message: t('enter_min_silence_duration'), trigger: 'blur' }],
  'silero_vad.sample_rate': [{ required: true, message: t('select_sample_rate'), trigger: 'change' }],
  'silero_vad.channels': [{ required: true, message: t('select_channel_count'), trigger: 'change' }],
  'silero_vad.acquire_timeout_ms': [{ required: true, message: t('enter_fetch_timeout'), trigger: 'blur' }],
  'ten_vad.hop_size': [{ required: true, message: t('enter_frame_shift'), trigger: 'blur' }],
  'ten_vad.threshold': [{ required: true, message: t('enter_vad_threshold'), trigger: 'blur' }],
  'ten_vad.pool_size': [{ required: true, message: t('enter_pool_size'), trigger: 'blur' }],
  'ten_vad.acquire_timeout_ms': [{ required: true, message: t('enter_fetch_timeout'), trigger: 'blur' }]
}

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/vad-configs')
    configs.value = (response.data.data || []).map(normalizeVADConfigRow)
  } catch (error) {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

function normalizeVADConfigRow(row) {
  const data = parseJsonData(row?.json_data)
  return {
    ...row,
    provider: resolveVADProvider(row?.provider, row?.config_id, data)
  }
}

const editConfig = (config) => {
  config = normalizeVADConfigRow(config)
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.is_default = config.is_default
  form.enabled = config.enabled
  
  // 解析配置JSON并填充到对应字段
  try {
    const configObj = JSON.parse(config.json_data || '{}')
    if (configObj.webrtc_vad) {
      form.webrtc_vad = { ...form.webrtc_vad, ...configObj.webrtc_vad }
    } else if (configObj.silero_vad) {
      form.silero_vad = { ...form.silero_vad, ...configObj.silero_vad }
    } else if (configObj.ten_vad) {
      form.ten_vad = { ...form.ten_vad, ...configObj.ten_vad }
    } else {
      if (config.provider === 'webrtc_vad') {
        form.webrtc_vad = { ...form.webrtc_vad, ...configObj }
      } else if (config.provider === 'silero_vad') {
        form.silero_vad = { ...form.silero_vad, ...configObj }
      } else if (config.provider === 'ten_vad') {
        form.ten_vad = { ...form.ten_vad, ...configObj }
      }
    }
  } catch (error) {
    console.error('解析配置JSON失败:', error)
  }
  
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        // 如果是新增配置且当前没有任何配置，则自动设为默认配置
        const isFirstConfig = !editingConfig.value && configs.value.length === 0
        
        const configData = {
          name: form.name,
          config_id: form.config_id,
          provider: form.provider,
          is_default: isFirstConfig || form.is_default, // 首次添加时自动设为默认
          enabled: form.enabled !== undefined ? form.enabled : true,
          json_data: formRef.value.getJsonData()
        }

        if (editingConfig.value) {
          await api.put(`/admin/vad-configs/${editingConfig.value.id}`, configData)
          ElMessage.success(t('config_update_success'))
        } else {
          await api.post('/admin/vad-configs', configData)
          ElMessage.success(t('config_create_success'))
        }
        
        showDialog.value = false
        loadConfigs()
      } catch (error) {
        ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
      } finally {
        saving.value = false
      }
    }
  })
}

const toggleEnable = async (config) => {
  try {
    await api.post(`/admin/configs/${config.id}/toggle`)
    ElMessage.success(`${config.enabled ? '启用' : '禁用'}成功`)
  } catch (error) {
    // 恢复开关状态
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
    
    await api.put(`/admin/vad-configs/${config.id}`, configData)
    ElMessage.success(config.is_default ? '设为默认成功' : '取消默认成功')
    
    // 刷新列表以更新其他配置的默认状态
    loadConfigs()
  } catch (error) {
    // 恢复开关状态
    config.is_default = !config.is_default
    ElMessage.error(t('operation_failed'))
  }
}

const getEnabledConfigs = () => {
  return configs.value.filter(config => config.enabled)
}

function formatTestResultLabel(r) {
  if (!r?.ok) return '错误'
  return r.first_packet_ms != null ? `正确 ${r.first_packet_ms}ms` : '正确'
}
function formatTestResultTip(r) {
  if (!r?.ok) return ''
  return r.first_packet_ms != null ? `通过，耗时 ${r.first_packet_ms}ms` : '通过'
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
      ElMessage.success(`${row.name || row.config_id}：${formatTestMessage(result)}`)
    } else {
      ElMessage.warning(`${row.name || row.config_id}：${result.message}`)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '测试请求失败')
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
        const result = await testSingleConfig('vad', row.config_id)
        testResults.value = { ...testResults.value, [row.config_id]: result }
        if (result.ok) okCount++
      } catch (_) {
        testResults.value = { ...testResults.value, [row.config_id]: { ok: false, message: t('request_failed') } }
      }
    }
    ElMessage.success(`全部测试完成：${okCount}/${list.length} 通过`)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '测试请求失败')
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
    const result = await testWithData('vad', { [configId]: payload })
    if (result.ok) {
      ElMessage.success(formatTestMessage(result) || '测试通过')
    } else {
      ElMessage.warning(result.message || '测试未通过')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || '测试请求失败')
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
    
    await api.delete(`/admin/vad-configs/${id}`)
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
  Object.assign(form, {
    name: '',
    config_id: '',
    provider: 'ten_vad',
    is_default: false,
    enabled: true,
    webrtc_vad: {
      pool_min_size: 5,
      pool_max_size: 1000,
      pool_max_idle: 100,
      vad_sample_rate: 16000,
      vad_mode: 2
    },
    silero_vad: {
      model_path: 'config/models/vad/silero_vad.onnx',
      threshold: 0.5,
      min_silence_duration_ms: 100,
      sample_rate: 16000,
      channels: 1,
      acquire_timeout_ms: 3000
    },
    ten_vad: {
      hop_size: 320,
      threshold: 0.3,
      pool_size: 10,
      acquire_timeout_ms: 3000
    }
  })
}

const handleDialogClose = () => {
  showDialog.value = false
  resetForm()
  if (formRef.value) {
    formRef.value.resetFields()
  }
}

const formatDate = (dateString) => {
  return new Date(dateString).toLocaleString('zh-CN')
}

onMounted(() => {
  loadConfigs()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.test-result { font-size: 12px; }
.test-result.test-ok { color: var(--el-color-success); }
.test-result.test-err { color: var(--el-color-danger); cursor: help; }
.test-result.test-none { color: var(--el-text-color-placeholder); }
</style>
