<template>
  <div class="config-page">
    <!-- Basic config section -->
    <el-card class="base-config-card" style="margin-bottom: 20px;">
      <template #header>
        <div class="card-header">
          <span>{{ t('basic_config') }}</span>
        </div>
      </template>
      
      <el-form
        ref="baseFormRef"
        :model="baseForm"
        :rules="baseRules"
        label-width="120px"
        style="max-width: 600px;"
      >
        <el-form-item :label="t('enable_auth')" prop="enable_auth">
          <el-switch v-model="baseForm.enable_auth" />
          <div class="form-tip">{{ t('vision_auth_hint') }}</div>
        </el-form-item>
        
        <el-form-item label="Vision URL" prop="vision_url">
          <el-input 
            v-model="baseForm.vision_url" 
            :placeholder="t('vision_url_ph')"
            style="width: 100%;"
          />
          <div class="form-tip">{{ t('vision_url_hint') }}</div>
        </el-form-item>
        
        <el-form-item>
          <el-button type="primary" @click="saveBaseConfig" :loading="baseSaving">{{ t('save_basic_config') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Config list section -->
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ t('model_config_list') }}</span>
          <el-button type="primary" @click="showDialog = true">
            <el-icon><Plus /></el-icon>
            {{ t('add_config') }}
          </el-button>
        </div>
      </template>

      <el-table :data="configs" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" :label="t('config_name')" />
        <el-table-column prop="provider" :label="t('provider')" />
        <el-table-column prop="enabled" :label="t('enabled_status')" width="80" align="center">
          <template #default="scope">
            <el-switch 
              v-model="scope.row.enabled" 
              @change="toggleEnable(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="is_default" :label="t('default_config')" width="80" align="center">
          <template #default="scope">
            <el-switch 
              v-model="scope.row.is_default" 
              @change="toggleDefault(scope.row)"
              :disabled="scope.row.is_default && getEnabledConfigs().length === 1"
            />
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('created_at')" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('actions')" width="180">
          <template #default="scope">
            <el-button size="small" @click="editConfig(scope.row)">{{ t('edit') }}</el-button>
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
    </el-card>

    <!-- Add/Edit config dialog -->
    <el-dialog
      v-model="showDialog"
      :title="editingConfig ? t('edit_vision_config') : t('add_vision_config')"
      width="700px"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item :label="t('provider')" prop="provider">
          <el-select v-model="form.provider" :placeholder="t('select_provider')" style="width: 100%">
            <el-option :label="t('aliyun_vision')" value="aliyun_vision" />
            <el-option :label="t('doubao_vision')" value="doubao_vision" />
          </el-select>
        </el-form-item>
        
        <el-form-item :label="t('config_name')" prop="name">
          <el-input v-model="form.name" :placeholder="t('enter_config_name')" />
        </el-form-item>
        
        <el-form-item :label="t('type')" prop="type">
          <el-input v-model="form.type" :placeholder="t('enter_type')" />
        </el-form-item>
        
        <el-form-item :label="t('model_name_label')" prop="model_name">
          <el-input v-model="form.model_name" :placeholder="t('enter_model_name')" />
        </el-form-item>
        
        <el-form-item :label="t('api_key')" prop="api_key">
          <el-input v-model="form.api_key" type="password" :placeholder="t('enter_api_password')" show-password />
        </el-form-item>
        
        <el-form-item :label="t('base_url')" prop="base_url">
          <el-input v-model="form.base_url" :placeholder="t('enter_base_url')" />
        </el-form-item>
        
        <el-form-item :label="t('max_tokens_label')" prop="max_tokens">
          <el-input-number v-model="form.max_tokens" :min="1" :max="100000" :placeholder="t('enter_max_tokens')" style="width: 100%" />
        </el-form-item>
        
        <el-form-item :label="t('temperature')" prop="temperature">
          <el-input-number v-model="form.temperature" :min="0" :max="2" :step="0.1" :placeholder="t('enter_temperature')" style="width: 100%" />
        </el-form-item>
        
        <el-form-item label="Top P" prop="top_p">
          <el-input-number v-model="form.top_p" :min="0" :max="1" :step="0.1" :placeholder="t('enter_top_p')" style="width: 100%" />
        </el-form-item>
        
        <el-form-item :label="t('timeout_seconds')" prop="timeout">
          <el-input-number v-model="form.timeout" :min="1" :max="300" :placeholder="t('enter_timeout')" style="width: 100%" />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="handleDialogClose">{{ t('cancel') }}</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          {{ t('save') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { resolveVisionProvider } from './forms/configProviderUtils'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const configs = ref([])
const loading = ref(false)
const saving = ref(false)
const baseSaving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)
const formRef = ref()
const baseFormRef = ref()

// Base config form
const baseForm = reactive({
  enable_auth: false,
  vision_url: ''
})

// Base config validation rules
const baseRules = {
  vision_url: [
    { required: true, message: t('enter_vision_url'), trigger: 'blur' },
    { type: 'url', message: t('enter_valid_url'), trigger: 'blur' }
  ]
}

const form = reactive({
  name: '',
  provider: 'aliyun_vision',
  is_default: false,
  enabled: true,
  type: 'openai',
  model_name: 'qwen-vl-max',
  api_key: '',
  base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  max_tokens: 1000,
  temperature: 0.1,
  top_p: 0.1,
  timeout: 30
})

const generateConfig = () => {
  return JSON.stringify({
    provider: form.provider,
    type: form.type,
    model_name: form.model_name,
    api_key: form.api_key,
    base_url: form.base_url,
    max_tokens: form.max_tokens,
    temperature: form.temperature,
    top_p: form.top_p,
    timeout: form.timeout
  })
}

const rules = {
  name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
  provider: [{ required: true, message: t('select_provider'), trigger: 'change' }],
  type: [{ required: true, message: t('enter_type'), trigger: 'blur' }],
  model_name: [{ required: true, message: t('enter_model_name'), trigger: 'blur' }],
  api_key: [{ required: true, message: t('enter_api_password'), trigger: 'blur' }],
  base_url: [
    { required: true, message: t('enter_base_url'), trigger: 'blur' },
    { type: 'url', message: t('enter_valid_url'), trigger: 'blur' }
  ],
  max_tokens: [{ required: true, message: t('enter_max_tokens'), trigger: 'blur' }],
  timeout: [{ required: true, message: t('enter_timeout'), trigger: 'blur' }]
}

const parseJsonData = (jsonData) => {
  try {
    return JSON.parse(jsonData || '{}')
  } catch (error) {
    return {}
  }
}

const normalizeVisionConfigRow = (config) => {
  const data = parseJsonData(config.json_data)
  return {
    ...config,
    provider: resolveVisionProvider(config.provider, config.config_id, data)
  }
}

// Load base config
const loadBaseConfig = async () => {
  try {
    const response = await api.get('/admin/vision-base-config')
    const data = response.data.data || {}
    baseForm.enable_auth = data.enable_auth || false
    baseForm.vision_url = data.vision_url || ''
  } catch (error) {
    console.error(t('load_base_config_failed'), error)
  }
}

// Save base config
const saveBaseConfig = async () => {
  if (!baseFormRef.value) return
  
  await baseFormRef.value.validate(async (valid) => {
    if (valid) {
      baseSaving.value = true
      try {
        await api.put('/admin/vision-base-config', {
          enable_auth: baseForm.enable_auth,
          vision_url: baseForm.vision_url
        })
        ElMessage.success(t('basic_config_save_success'))
      } catch (error) {
        ElMessage.error(t('save_failed_check_network'))
      } finally {
        baseSaving.value = false
      }
    }
  })
}

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/vision-configs')
    // Filter out the vision_base config to keep it out of the list
    const allConfigs = response.data.data || []
    configs.value = allConfigs.filter(config => config.config_id !== 'vision_base').map(normalizeVisionConfigRow)
  } catch (error) {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

const editConfig = (config) => {
  const normalizedConfig = normalizeVisionConfigRow(config)
  editingConfig.value = normalizedConfig
  form.name = normalizedConfig.name
  form.provider = normalizedConfig.provider
  form.is_default = normalizedConfig.is_default
  form.enabled = normalizedConfig.enabled
  
  try {
    const configData = parseJsonData(normalizedConfig.json_data)
    form.type = configData.type || ''
    form.model_name = configData.model_name || ''
    form.api_key = configData.api_key || ''
    form.base_url = configData.base_url || ''
    form.max_tokens = configData.max_tokens || 4096
    form.temperature = configData.temperature !== undefined ? configData.temperature : 0.7
    form.top_p = configData.top_p !== undefined ? configData.top_p : 0.9
    form.timeout = configData.timeout || 30
  } catch (error) {
    console.error(t('parse_config_failed'), error)
    ElMessage.warning(t('config_format_error_reset'))
  }
  
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        const isFirstConfig = !editingConfig.value && configs.value.length === 0
        
        const configData = {
          name: form.name,
          provider: form.provider,
          is_default: isFirstConfig || form.is_default,
          enabled: form.enabled !== undefined ? form.enabled : true,
          json_data: generateConfig()
        }
        
        if (editingConfig.value) {
          await api.put(`/admin/vision-configs/${editingConfig.value.id}`, configData)
          ElMessage.success(t('update_success'))
        } else {
          await api.post('/admin/vision-configs', configData)
          ElMessage.success(t('add_success'))
        }
        
        showDialog.value = false
        loadConfigs()
      } catch (error) {
        ElMessage.error(t('save_failed_check_network'))
      } finally {
        saving.value = false
      }
    }
  })
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
      provider: config.provider,
      is_default: config.is_default,
      enabled: config.enabled,
      json_data: config.json_data
    }
    
    await api.put(`/admin/vision-configs/${config.id}`, configData)
    ElMessage.success(config.is_default ? t('set_default_success') : t('cancel_default_success'))
    loadConfigs()
  } catch (error) {
    config.is_default = !config.is_default
    ElMessage.error(t('operation_failed'))
  }
}

const getEnabledConfigs = () => {
  return configs.value.filter(config => config.enabled)
}

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_config'), t('hint'), {
      confirmButtonText: t('confirm'),
      cancelButtonText: t('cancel'),
      type: 'warning'
    })
    
    await api.delete(`/admin/vision-configs/${id}`)
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
    provider: 'aliyun_vision',
    is_default: false,
    enabled: true,
    type: 'openai',
    model_name: 'qwen-vl-max',
    api_key: '',
    base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    max_tokens: 1000,
    temperature: 0.1,
    top_p: 0.1,
    timeout: 30
  })
  formRef.value?.clearValidate()
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
  loadBaseConfig()
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

.base-config-card {
  background: #f8f9fa;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  color: #333;
}

.form-tip {
  font-size: 12px;
  color: #666;
  margin-top: 4px;
  line-height: 1.4;
}
</style>
