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
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon>
        {{ t('add_config') }}  添加配置
      </el-button>
    </div>

    <el-table :data="configs" style="width: 100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" :label="t('config_name')" />
      <el-table-column prop="config_id" :label="t('config_id')" width="150" />
      <el-table-column prop="provider" :label="t('type')">
        <template #default="scope">
          {{ getProviderLabel(scope.row.provider) }}
        </template>
      </el-table-column>
      <el-table-column label="思考" width="100" align="center">
        <template #default="scope">
          <el-tag
            v-if="getThinkingLabel(scope.row)"
            size="small"
            effect="plain"
            :type="testResults[scope.row.config_id]?.ok ? 'success' : 'info'"
            class="thinking-tag"
          >
            {{ getThinkingLabel(scope.row) }}
          </el-tag>
          <span v-else class="test-result test-none">-</span>
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
      <el-table-column label="耗时" width="100" align="center">
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
            @click="testConfig(scope.row, 'llm')"
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
      :title="editingConfig ? '编辑LLM配置' : '添加LLM配置'"
      width="600px"
      @close="handleDialogClose"
    >
      <LLMConfigForm ref="formRef" :model="form" :rules="rules" />
      
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
import { ref, reactive, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { testSingleConfig, testWithData, parseJsonData } from '../../utils/configTest'
import LLMConfigForm from './forms/LLMConfigForm.vue'
import { getProviderFixedType, getProviderThinkingConfig, isProviderBaseURLEditable, resolveLLMProvider } from './forms/llmCatalog'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const configs = ref([])
const testingId = ref(null)
const testingAll = ref(false)
const testingCurrent = ref(false)
const testResults = ref({}) // config_id -> { ok, message }
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

const rules = {
  name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
  config_id: [{ required: true, message: t('enter_config_id'), trigger: 'blur' }],
  provider: [{ required: true, message: t('select_provider'), trigger: 'change' }],
  model_name: [{
    required: true,
    message: t('enter_model_name'),
    trigger: 'change'
  }, {
    validator: (_, value, callback) => {
      const provider = resolveLLMProvider(form.provider, form.type)
      const providerType = getProviderFixedType(provider)
      if ((providerType === 'openai' || providerType === 'ollama') && !value) {
        callback(new Error(t('enter_model_name')))
        return
      }
      callback()
    },
    trigger: 'change'
  }],
  api_key: [{
    validator: (_, value, callback) => {
      const provider = resolveLLMProvider(form.provider, form.type)
      if (getProviderFixedType(provider) !== 'ollama' && !value) {
        callback(new Error(t('enter_api_password')))
        return
      }
      callback()
    },
    trigger: 'blur'
  }],
  base_url: [{
    validator: (_, value, callback) => {
      const provider = resolveLLMProvider(form.provider, form.type)
      if (isProviderBaseURLEditable(provider) && !value) {
        callback(new Error(t('enter_base_url')))
        return
      }
      callback()
    },
    trigger: 'blur'
  }],
  max_tokens: [{
    validator: (_, value, callback) => {
      const provider = resolveLLMProvider(form.provider, form.type)
      const providerType = getProviderFixedType(provider)
      if ((providerType === 'openai' || providerType === 'ollama') && (!value || Number(value) < 1 || Number(value) > 100000)) {
        callback(new Error('max_tokens必须在1-100000之间'))
        return
      }
      callback()
    },
    trigger: 'blur'
  }],
  bot_id: [{
    validator: (_, value, callback) => {
      const provider = resolveLLMProvider(form.provider, form.type)
      if (getProviderFixedType(provider) === 'coze' && !value) {
        callback(new Error('请输入Coze Bot ID'))
        return
      }
      callback()
    },
    trigger: 'blur'
  }],
  temperature: [{ type: 'number', min: 0, max: 2, message: t('temperature_range_error'), trigger: 'blur' }],
  top_p: [{ type: 'number', min: 0, max: 1, message: t('top_p_range_error'), trigger: 'blur' }]
}

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/llm-configs')
    configs.value = (response.data.data || []).map((row) => {
      const configObj = parseJsonData(row?.json_data)
      return {
        ...row,
        provider: resolveLLMProvider(row?.provider, configObj?.type)
      }
    })
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
  
  // 解析配置JSON并填充到对应字段
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
    console.error('解析配置JSON失败:', error)
  }
  
  showDialog.value = true
  nextTick(() => {
    formRef.value?.clearValidate?.()
  })
}

const openCreateDialog = () => {
  resetForm()
  showDialog.value = true
  nextTick(() => {
    formRef.value?.clearValidate?.()
  })
}

const handleSave = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        // 检查是否是首次添加配置
        const isFirstConfig = !editingConfig.value && configs.value.length === 0
        
        const configData = {
          name: form.name,
          config_id: form.config_id,
          provider: form.provider,
          is_default: isFirstConfig || form.is_default, // 首次添加自动设为默认
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
    
    await api.put(`/admin/llm-configs/${config.id}`, configData)
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
  return r.first_packet_ms != null ? `${r.first_packet_ms}ms` : '通过'
}
function formatTestResultTip(r) {
  if (!r?.ok) return ''
  const parts = []
  if (r.first_packet_ms != null) parts.push(`首包 ${r.first_packet_ms}ms`)
  if (r.reasoning_content_returned) parts.push('检测到上游返回思考内容')
  return parts.length ? parts.join('，') : '通过'
}
function formatTestMessage(result) {
  const base = result.message || ''
  const suffix = []
  if (result.first_packet_ms != null) suffix.push(`${result.first_packet_ms}ms`)
  if (result.reasoning_content_returned) suffix.push('检测到上游返回思考内容')
  return suffix.length ? `${base} ${suffix.join(' · ')}` : base
}

function formatDraftTestLabel(name, configId) {
  return name?.trim() || configId?.trim() || '当前配置'
}

function getThinkingLabel(row) {
  const config = parseJsonData(row?.json_data)
  const provider = resolveLLMProvider(row?.provider, config?.type)
  const mode = String(config?.thinking?.mode || '').trim()
  if (!mode || mode === 'default') {
    return ''
  }

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
        const result = await testSingleConfig('llm', row.config_id)
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
    const result = await testWithData('llm', { provider: configId, [configId]: payload })
    const label = formatDraftTestLabel(form.name, configId)
    if (result.ok) {
      ElMessage.success(`${label}：${formatTestMessage(result) || '测试通过'}`)
    } else {
      ElMessage.warning(`${label}：${result.message || '测试未通过'}`)
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

const handleDialogClose = () => {
  showDialog.value = false
  resetForm()
  nextTick(() => {
    formRef.value?.clearValidate?.()
  })
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

.thinking-tag {
  max-width: 84px;
}

.test-result { font-size: 12px; }
.test-result.test-ok { color: var(--el-color-success); }
.test-result.test-err { color: var(--el-color-danger); cursor: help; }
.test-result.test-none { color: var(--el-text-color-placeholder); }
</style>
