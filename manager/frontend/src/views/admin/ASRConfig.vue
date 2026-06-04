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
        {{ t('test_all') }}</el-button>
      <el-button type="primary" @click="showDialog = true">
        <el-icon><Plus /></el-icon>
        {{ t('add_config') }}
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
            @click="testConfig(scope.row, 'asr')"
          >
            测试
          </el-button>
          <el-button
            size="small"
            type="danger"
            @click="deleteConfig(scope.row.id)"
          >
            {{ t('delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑配置弹窗 -->
    <el-dialog
      v-model="showDialog"
      :title="editingConfig ? '编辑ASR配置' : '添加ASR配置'"
      width="720px"
      @close="handleDialogClose"
    >
      <ASRConfigForm ref="formRef" :model="form" :rules="rules" />
      
      <template #footer>
        <el-button @click="handleDialogClose">{{ t('cancel') }}</el-button>
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
import ASRConfigForm from './forms/ASRConfigForm.vue'
import { resolveASRProvider } from './forms/configProviderUtils'
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

const validateAliyunPcm = (rule, value, callback) => {
  if (value !== 'pcm') {
    callback(new Error('格式必须为pcm'))
    return
  }
  callback()
}

const validateAliyun16000 = (rule, value, callback) => {
  if (Number(value) !== 16000) {
    callback(new Error('采样率必须为16000'))
    return
  }
  callback()
}

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

// 按当前 provider 动态规则，避免未显示的 doubao/funasr 字段触发必填导致保存不发请求
const rules = computed(() => {
  const base = {
    name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
    config_id: [{ required: true, message: t('enter_config_id'), trigger: 'blur' }],
    provider: [{ required: true, message: t('select_provider'), trigger: 'change' }]
  }
  if (form.provider === 'funasr') {
    return {
      ...base,
      'funasr.host': [{ required: true, message: t('enter_host_address'), trigger: 'blur' }],
      'funasr.port': [{ required: true, message: t('enter_port'), trigger: 'blur' }],
      'funasr.mode': [{ required: true, message: t('select_mode'), trigger: 'change' }],
      'funasr.sample_rate': [{ required: true, message: t('select_sample_rate'), trigger: 'change' }],
      'funasr.chunk_size': [{ required: true, message: t('enter_chunk_size'), trigger: 'blur' }],
      'funasr.chunk_interval': [{ required: true, message: t('enter_chunk_interval'), trigger: 'blur' }],
      'funasr.max_connections': [{ required: true, message: t('enter_max_connections'), trigger: 'blur' }],
      'funasr.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }]
    }
  }
  if (form.provider === 'aliyun_funasr') {
    return {
      ...base,
      'aliyun_funasr.ws_url': [{ required: true, message: t('enter_ws_url'), trigger: 'blur' }],
      'aliyun_funasr.model': [{ required: true, message: t('enter_model_name'), trigger: 'blur' }],
      'aliyun_funasr.format': [
        { required: true, message: t('select_audio_format'), trigger: 'change' },
        { validator: validateAliyunPcm, trigger: 'change' }
      ],
      'aliyun_funasr.sample_rate': [
        { required: true, message: t('select_sample_rate'), trigger: 'change' },
        { validator: validateAliyun16000, trigger: 'change' }
      ],
      'aliyun_funasr.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }]
    }
  }
  if (form.provider === 'doubao') {
    return {
      ...base,
      'doubao.appid': [{ required: true, message: t('enter_app_id'), trigger: 'blur' }],
      'doubao.access_token': [{ required: true, message: t('enter_access_token'), trigger: 'blur' }],
      'doubao.ws_url': [{ required: true, message: t('enter_websocket_url'), trigger: 'blur' }],
      'doubao.resource_id': [{ required: true, message: t('select_resource_spec'), trigger: 'change' }],
      'doubao.end_window_size': [{ required: true, message: t('enter_end_window_size'), trigger: 'blur' }],
      'doubao.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }]
    }
  }
  if (form.provider === 'aliyun_qwen3') {
    return {
      ...base,
      'aliyun_qwen3.ws_url': [{ required: true, message: t('enter_ws_url'), trigger: 'blur' }],
      'aliyun_qwen3.model': [{ required: true, message: t('enter_model_name'), trigger: 'blur' }],
      'aliyun_qwen3.format': [{ required: true, message: t('select_audio_format'), trigger: 'change' }],
      'aliyun_qwen3.sample_rate': [{ required: true, message: t('select_sample_rate'), trigger: 'change' }],
      'aliyun_qwen3.language': [{ required: true, message: t('enter_language'), trigger: 'blur' }],
      'aliyun_qwen3.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }]
    }
  }
  if (form.provider === 'xunfei') {
    return {
      ...base,
      'xunfei.appid': [{ required: true, message: t('enter_app_id'), trigger: 'blur' }],
      'xunfei.api_key': [{ required: true, message: t('enter_api_key'), trigger: 'blur' }],
      'xunfei.api_secret': [{ required: true, message: t('enter_api_secret'), trigger: 'blur' }],
      'xunfei.host': [{ required: true, message: t('enter_host'), trigger: 'blur' }],
      'xunfei.path': [{ required: true, message: t('enter_path'), trigger: 'blur' }],
      'xunfei.sample_rate': [{ required: true, message: t('enter_sample_rate'), trigger: 'change' }],
      'xunfei.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }]
    }
  }
  return base
})

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/asr-configs')
    configs.value = (response.data.data || []).map(normalizeASRConfigRow)
  } catch (error) {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

function normalizeASRConfigRow(row) {
  const data = parseJsonData(row?.json_data)
  return {
    ...row,
    provider: resolveASRProvider(row?.provider, row?.config_id, data)
  }
}

const editConfig = (config) => {
  config = normalizeASRConfigRow(config)
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.is_default = config.is_default
  form.enabled = config.enabled
  
  // 解析配置JSON并填充到对应字段
  try {
    const configObj = JSON.parse(config.json_data || '{}')
    
    // 兼容新旧格式：检查是否是包装格式（包含provider层）还是直接格式
    if (configObj.funasr) {
      // 旧格式：包含provider层
      const funasrConfig = { ...form.funasr, ...configObj.funasr }
      // 兼容chunk_size：如果是单个数字或无效格式，转换为默认值 [5, 10, 5]
      if (typeof funasrConfig.chunk_size === 'number') {
        funasrConfig.chunk_size = [5, 10, 5]
      } else if (!Array.isArray(funasrConfig.chunk_size) || funasrConfig.chunk_size.length !== 3) {
        funasrConfig.chunk_size = [5, 10, 5]
      }
      form.funasr = funasrConfig
    } else if (configObj.aliyun_funasr) {
      // 旧格式：包含provider层
      form.aliyun_funasr = { ...form.aliyun_funasr, ...configObj.aliyun_funasr }
    } else if (configObj.doubao) {
      // 旧格式：包含provider层
      form.doubao = { ...form.doubao, ...configObj.doubao }
    } else if (config.provider === 'funasr' && configObj.host) {
      // 新格式：直接包含配置内容
      const funasrConfig = { ...form.funasr, ...configObj }
      // 兼容chunk_size：如果是单个数字或无效格式，转换为默认值 [5, 10, 5]
      if (typeof funasrConfig.chunk_size === 'number') {
        funasrConfig.chunk_size = [5, 10, 5]
      } else if (!Array.isArray(funasrConfig.chunk_size) || funasrConfig.chunk_size.length !== 3) {
        funasrConfig.chunk_size = [5, 10, 5]
      }
      form.funasr = funasrConfig
    } else if (config.provider === 'aliyun_funasr' && (configObj.ws_url || configObj.model || configObj.api_key)) {
      // 新格式：直接包含配置内容
      form.aliyun_funasr = { ...form.aliyun_funasr, ...configObj }
    } else if (config.provider === 'doubao' && (configObj.appid || configObj.access_token)) {
      // 新格式：直接包含配置内容
      form.doubao = { ...form.doubao, ...configObj }
    } else if (configObj.aliyun_qwen3) {
      // 旧格式：包含provider层
      form.aliyun_qwen3 = { ...form.aliyun_qwen3, ...configObj.aliyun_qwen3 }
    } else if (config.provider === 'aliyun_qwen3' && (configObj.ws_url || configObj.model || configObj.api_key)) {
      // 新格式：直接包含配置内容
      form.aliyun_qwen3 = { ...form.aliyun_qwen3, ...configObj }
    } else if (configObj.xunfei) {
      form.xunfei = { ...form.xunfei, ...configObj.xunfei }
    } else if (config.provider === 'xunfei' && (configObj.appid || configObj.api_key || configObj.api_secret)) {
      form.xunfei = { ...form.xunfei, ...configObj }
    }
  } catch (error) {
    console.error('解析配置JSON失败:', error)
  }
  
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) {
    ElMessage.warning(t('form_not_ready'))
    return
  }
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
          enabled: form.enabled !== undefined ? form.enabled : true, // 确保enabled字段存在
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
        ElMessage.error('保存失败: ' + msg)
      } finally {
        saving.value = false
      }
    }
  })
}

const toggleEnable = async (config) => {
  try {
    await api.post(`/admin/configs/${config.id}/toggle`)
    ElMessage.success(`${config.enabled ? t('enabled') : '禁用'}成功`)
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
    
    await api.put(`/admin/asr-configs/${config.id}`, configData)
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
        const result = await testSingleConfig('asr', row.config_id)
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
    const result = await testWithData('asr', { [configId]: payload })
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
    
    await api.delete(`/admin/asr-configs/${id}`)
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
  form.funasr = {
    host: 'localhost',
    port: 10095,
    mode: 'offline',
    sample_rate: 16000,
    chunk_size: [5, 10, 5],
    chunk_interval: 10,
    max_connections: 100,
    timeout: 30,
    auto_end: false
  }
  form.aliyun_funasr = {
    api_key: '',
    ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/inference/',
    model: 'fun-asr-realtime',
    format: 'pcm',
    sample_rate: 16000,
    language_hints: ['zh'],
    vocabulary_id: '',
    disfluency_removal_enabled: false,
    timeout: 30
  }
  form.doubao = {
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
  }
  form.aliyun_qwen3 = {
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
  }
  form.xunfei = {
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
