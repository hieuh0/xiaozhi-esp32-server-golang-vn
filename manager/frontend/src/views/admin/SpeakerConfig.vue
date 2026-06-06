<template>
  <div class="config-page">
    <el-card v-loading="loading" class="config-card">
      <el-alert
        :title="t('hint')"
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 20px;"
      >
        <template #default>
          {{ t('docker_env_hint') }}
        </template>
      </el-alert>
      
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item :label="t('service_address')" prop="base_url">
          <el-input 
            v-model="form.base_url" 
            :placeholder="t('http_service_address_ph')"
            style="width: 100%"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            {{ t('http_to_ws_hint') }}
          </div>
        </el-form-item>
        
        <el-form-item :label="t('recognition_threshold')" prop="threshold">
          <el-input-number 
            v-model="form.threshold" 
            :min="0" 
            :max="1" 
            :step="0.1" 
            :precision="2"
            placeholder="0.4"
            style="width: 100%"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            {{ t('recognition_threshold_hint') }}
          </div>
        </el-form-item>
        
        <el-form-item :label="t('enabled_status')">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      
      <div class="form-actions">
        <el-button type="primary" @click="handleSave" :loading="saving">
          {{ t('save_config') }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const formRef = ref()
const currentConfig = ref(null)

const form = reactive({
  base_url: 'http://192.168.208.214:8080',
  threshold: 0.4,
  enabled: true
})

const rules = {
  base_url: [
    { required: true, message: t('enter_service_address'), trigger: 'blur' },
    { 
      pattern: /^https?:\/\/.+/, 
      message: t('enter_valid_http_address'), 
      trigger: 'blur' 
    }
  ],
  threshold: [
    { required: true, message: t('enter_recognition_threshold'), trigger: 'blur' },
    { 
      type: 'number', 
      min: 0, 
      max: 1, 
      message: t('threshold_0_to_1_decimal'), 
      trigger: 'blur' 
    }
  ]
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/speaker-configs')
    const configs = response.data.data || []
    
    if (configs.length > 0) {
      // If configs exist, use the first one (should be only one)
      currentConfig.value = configs[0]
      const configObj = JSON.parse(configs[0].json_data || '{}')

      // Parse config
      if (configObj.service && configObj.service.base_url) {
        form.base_url = configObj.service.base_url
      } else if (configObj.base_url) {
        // Old format compatibility
        form.base_url = configObj.base_url
      }
      // Read threshold config
      if (configObj.service && configObj.service.threshold !== undefined) {
        form.threshold = configObj.service.threshold
      } else if (configObj.threshold !== undefined) {
        // Old format compatibility
        form.threshold = configObj.threshold
      } else {
        // Default value
        form.threshold = 0.4
      }
      // Toggle maps to json_data.enable (business-level enable), not the API-returned enabled column
      form.enabled = configObj.enable !== undefined ? configObj.enable : true
    }
  } catch (error) {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        // Build config data: toggle written to json_data.enable, used as the authoritative enable state
        const configData = {
          service: {
            base_url: form.base_url,
            threshold: form.threshold
          },
          enable: form.enabled
        }
        
        const saveData = {
          name: t('voiceprint_recognition_config'),
          config_id: 'asr_server',
          provider: 'asr_server',
          is_default: true,
          enabled: form.enabled,
          json_data: JSON.stringify(configData)
        }
        
        if (currentConfig.value) {
          // Update existing config
          await api.put(`/admin/speaker-configs/${currentConfig.value.id}`, saveData)
          ElMessage.success(t('config_update_success'))
        } else {
          // Create new config
          await api.post('/admin/speaker-configs', saveData)
          ElMessage.success(t('config_create_success'))
        }

        // Reload config
        await loadConfig()
      } catch (error) {
        ElMessage.error(t('save_failed_colon') + (error.response?.data?.message || error.message))
      } finally {
        saving.value = false
      }
    }
  })
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.config-card {
  max-width: 800px;
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--apple-text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}

.form-tip .el-icon {
  font-size: 14px;
  color: var(--apple-primary);
}

.form-actions {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
}
</style>
