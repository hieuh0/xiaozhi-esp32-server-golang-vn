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
          如果是docker-compose环境部署会读取环境变量中的api地址，无需进行配置
        </template>
      </el-alert>
      
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="服务地址" prop="base_url">
          <el-input 
            v-model="form.base_url" 
            placeholder="请输入HTTP服务地址，如：http://192.168.208.214:8080"
            style="width: 100%"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            请输入HTTP地址，系统会自动转换为WebSocket地址
          </div>
        </el-form-item>
        
        <el-form-item label="识别阈值" prop="threshold">
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
            声纹识别阈值，范围 0.0-1.0，默认 0.4。值越大识别越严格
          </div>
        </el-form-item>
        
        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      
      <div class="form-actions">
        <el-button type="primary" @click="handleSave" :loading="saving">
          保存配置
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
      // 如果有配置，使用第一个（应该只有一个）
      currentConfig.value = configs[0]
      const configObj = JSON.parse(configs[0].json_data || '{}')
      
      // 解析配置
      if (configObj.service && configObj.service.base_url) {
        form.base_url = configObj.service.base_url
      } else if (configObj.base_url) {
        // 兼容旧格式
        form.base_url = configObj.base_url
      }
      // 读取阈值配置
      if (configObj.service && configObj.service.threshold !== undefined) {
        form.threshold = configObj.service.threshold
      } else if (configObj.threshold !== undefined) {
        // 兼容旧格式
        form.threshold = configObj.threshold
      } else {
        // 默认值
        form.threshold = 0.4
      }
      // 开关对应 json_data.enable（业务启用），不使用接口返回的 enabled 列
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
        // 构建配置数据：开关写入 json_data.enable，对外输出以该字段为准
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
          // 更新现有配置
          await api.put(`/admin/speaker-configs/${currentConfig.value.id}`, saveData)
          ElMessage.success(t('config_update_success'))
        } else {
          // 创建新配置
          await api.post('/admin/speaker-configs', saveData)
          ElMessage.success(t('config_create_success'))
        }
        
        // 重新加载配置
        await loadConfig()
      } catch (error) {
        ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
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
