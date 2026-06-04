<template>
  <div class="mqtt-config">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      class="config-form"
      v-loading="loading"
    >
      <div class="config-layout">
        <el-card class="config-card" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Connection</p>
                <h3>{{ t('connection_params') }}连接参数</h3>
                <p class="card-description">{{ t('broker_setup_hint') }}</p>
              </div>
              <el-tag :type="isCoreFieldsComplete ? 'success' : 'warning'" effect="plain" round>
                {{ isCoreFieldsComplete ? t('params_complete') : t('pending_fill') }}
              </el-tag>
            </div>
          </template>

          <div class="field-grid">
            <el-form-item :label="t('config_name')" prop="name">
              <el-input v-model="form.name" placeholder="例如：默认 MQTT 连接" />
            </el-form-item>

            <el-form-item label="Broker 地址" prop="broker">
              <el-input v-model="form.broker" placeholder="例如：mqtt://127.0.0.1 或 broker.example.com" />
            </el-form-item>

            <el-form-item label="连接类型" prop="type">
              <el-select v-model="form.type" :placeholder="t('select_connection_type')" style="width: 100%">
                <el-option
                  v-for="option in connectionTypeOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </el-form-item>

            <el-form-item :label="t('port')" prop="port">
              <el-input-number
                v-model="form.port"
                :min="1"
                :max="65535"
                controls-position="right"
                style="width: 100%"
              />
            </el-form-item>

            <el-form-item label="客户端 ID" prop="client_id" class="field-span-full">
              <el-input
                v-model="form.client_id"
                placeholder="请输入主程序连接 Broker 时使用的 Client ID"
              />
              <div class="field-help">
                建议使用稳定且可识别的 Client ID，便于在 Broker 侧定位连接来源。
              </div>
            </el-form-item>
          </div>
        </el-card>

        <el-card class="config-card config-card-side" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Authentication</p>
                <h3>认证信息</h3>
                <p class="card-description">如果 Broker 开启账号密码认证，请填写具有订阅权限的凭证。</p>
              </div>
              <el-tag :type="hasCredentials ? 'success' : 'info'" effect="plain" round>
                {{ hasCredentials ? t('credentials_filled') : t('can_be_empty') }}
              </el-tag>
            </div>
          </template>

          <div class="field-stack">
            <el-form-item :label="t('username')" prop="username">
              <el-input v-model="form.username" placeholder="未启用鉴权可留空" />
            </el-form-item>

            <el-form-item :label="t('password')" prop="password">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="未启用鉴权可留空"
                show-password
              />
            </el-form-item>
          </div>

          <div class="helper-panel">
            <div class="helper-item">
              <span>鉴权说明</span>
              <p>如果只填写了用户名或密码，也会一并保存；建议按 Broker 的实际策略成对维护。</p>
            </div>
            <div class="helper-item">
              <span>协议提醒</span>
              <p>常见组合是 TCP/1883、SSL/TLS/8883、WebSocket/8083，最终以你的 Broker 设置为准。</p>
            </div>
          </div>
        </el-card>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          保存后会更新默认 MQTT Client 配置，供主程序后续连接 Broker 使用。
        </p>
        <div class="footer-actions">
          <el-button plain :loading="loading" @click="loadConfig">重置为当前配置</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">保存配置</el-button>
        </div>
      </div>
    </el-form>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)
const formRef = ref()

const createDefaultFormState = () => ({
  name: t('mqtt_config_label'),
  is_default: true,
  enable: true,
  broker: '',
  type: 'tcp',
  port: 1883,
  client_id: '',
  username: '',
  password: ''
})

const form = reactive(createDefaultFormState())

const connectionTypeOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'WebSocket', value: 'websocket' },
  { label: 'SSL/TLS', value: 'ssl' }
]

const rules = {
  name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
  broker: [{ required: true, message: t('enter_mqtt_broker'), trigger: 'blur' }],
  type: [{ required: true, message: t('select_connection_type'), trigger: 'change' }],
  port: [
    { required: true, message: t('enter_port_number'), trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: t('port_range_error'), trigger: 'blur' }
  ],
  client_id: [{ required: true, message: t('enter_client_id'), trigger: 'blur' }]
}

const hasCredentials = computed(() => {
  return Boolean(String(form.username || '').trim() || String(form.password || '').trim())
})

const isCoreFieldsComplete = computed(() => {
  return Boolean(
    String(form.broker || '').trim() &&
    String(form.client_id || '').trim() &&
    form.type &&
    Number(form.port)
  )
})

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const generateConfig = () => {
  return {
    enable: form.enable,
    broker: String(form.broker || '').trim(),
    type: form.type,
    port: Number(form.port),
    client_id: String(form.client_id || '').trim(),
    username: String(form.username || '').trim(),
    password: String(form.password || '')
  }
}

const applyLoadedConfig = (config) => {
  configId.value = config?.id || null
  form.name = config?.name || t('mqtt_config_label')
  form.is_default = config?.is_default ?? true

  let configData = {}
  try {
    configData = JSON.parse(config?.json_data || '{}')
  } catch (error) {
    ElMessage.warning(t('mqtt_config_format_error'))
    configData = {}
  }

  form.enable = typeof configData.enable === 'boolean' ? configData.enable : true
  form.broker = String(configData.broker || '')
  form.type = String(configData.type || 'tcp')
  form.port = Number(configData.port) > 0 ? Number(configData.port) : 1883
  form.client_id = String(configData.client_id || '')
  form.username = String(configData.username || '')
  form.password = String(configData.password || '')
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/mqtt-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      applyLoadedConfig(configs[0])
    } else {
      configId.value = null
      resetForm()
    }
  } catch (error) {
    ElMessage.error(t('load_mqtt_config_failed'))
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    const generatedConfigId = `mqtt_${String(form.name || '')
      .replace(/[^a-zA-Z0-9]/g, '_')
      .toLowerCase()}`

    const nextConfigPayload = generateConfig()
    let configData
    let isUpdate = false

    if (configId.value) {
      const response = await api.get('/admin/mqtt-configs')
      const configs = response.data?.data || []
      const existingConfig = configs.find(item => item.id === configId.value)

      if (existingConfig) {
        let existingData = {}
        try {
          existingData = JSON.parse(existingConfig.json_data || '{}')
        } catch {
          existingData = {}
        }

        configData = {
          name: form.name,
          config_id: generatedConfigId,
          is_default: true,
          json_data: JSON.stringify({
            ...existingData,
            ...nextConfigPayload
          })
        }
        isUpdate = true
      }
    }

    if (!configData) {
      configData = {
        name: form.name,
        config_id: generatedConfigId,
        is_default: true,
        json_data: JSON.stringify(nextConfigPayload)
      }
    }

    if (isUpdate) {
      await api.put(`/admin/mqtt-configs/${configId.value}`, configData)
      ElMessage.success(t('mqtt_config_updated'))
    } else {
      const response = await api.post('/admin/mqtt-configs', configData)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success(t('mqtt_config_saved'))
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('save_mqtt_failed'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.mqtt-config {
  display: grid;
  padding: 0 24px 32px;
}

.footer-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
}

.card-kicker {
  display: block;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.card-kicker {
  color: var(--apple-text-tertiary);
}

.card-description,
.field-help,
.helper-item p,
.footer-note {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--apple-text-secondary);
}

.config-form {
  display: grid;
  gap: 24px;
}

.config-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.95fr);
  gap: 24px;
}

.config-card {
  border: 1px solid rgba(255, 255, 255, 0.88);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: var(--apple-shadow-md);
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.card-head h3 {
  margin: 8px 0 0;
  font-size: 22px;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: var(--apple-text);
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px 18px;
}

.field-span-full {
  grid-column: 1 / -1;
}

.field-stack {
  display: grid;
  gap: 20px;
}

.helper-panel {
  display: grid;
  gap: 14px;
  margin-top: 8px;
  padding-top: 18px;
  border-top: 1px solid rgba(229, 229, 234, 0.72);
}

.helper-item span {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--apple-text);
}

.footer-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 0 4px;
}

.footer-note {
  max-width: 620px;
  margin: 0;
}

:deep(.el-card__header) {
  padding: 24px 24px 0;
  border-bottom: none;
  background: transparent;
}

:deep(.el-card__body) {
  padding: 24px;
}

:deep(.el-form-item) {
  margin-bottom: 0;
}

:deep(.el-form-item__label) {
  font-size: 14px;
  font-weight: 600;
  color: var(--apple-text);
}

@media (max-width: 1180px) {
  .config-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .mqtt-config {
    padding: 0 16px 24px;
  }

  :deep(.el-card__body) {
    padding: 20px;
  }

  :deep(.el-card__header) {
    padding: 20px 20px 0;
  }

  .footer-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .field-grid {
    grid-template-columns: 1fr;
  }

  .footer-actions {
    justify-content: stretch;
  }

  .footer-actions :deep(.el-button) {
    flex: 1;
  }
}
</style>
