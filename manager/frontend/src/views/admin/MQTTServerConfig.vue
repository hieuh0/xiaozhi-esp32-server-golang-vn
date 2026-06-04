<template>
  <div class="mqtt-server-config">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      class="config-form"
      v-loading="loading"
    >
      <div class="config-layout">
        <el-card class="config-card config-card-main" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">MQTT Server</p>
                <h3>{{ t('listen_and_access') }}</h3>
                <p class="card-description">{{ t('mqtt_server_config_desc') }}配置内置 MQTT Server 的监听地址和启用状态，供设备与主程序接入。</p>
              </div>
              <el-tag :type="serverReady ? 'success' : 'warning'" effect="plain" round>
                {{ serverReady ? t('service_params_complete') : t('pending_fill') }}
              </el-tag>
            </div>
          </template>

          <div class="field-grid">
            <el-form-item :label="t('enabled_status')" prop="enable">
              <div class="switch-field">
                <div>
                  <div class="switch-title">启用内置 MQTT Server</div>
                  <div class="field-help">关闭后将不再监听设备侧 MQTT 连接。</div>
                </div>
                <el-switch v-model="form.enable" />
              </div>
            </el-form-item>

            <el-form-item label="监听主机" prop="listen_host">
              <el-input v-model="form.listen_host" placeholder="例如：0.0.0.0" />
            </el-form-item>

            <el-form-item label="监听端口" prop="listen_port">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </div>
        </el-card>

        <div class="side-stack">
          <el-card class="config-card" shadow="never">
            <template #header>
              <div class="card-head">
                <div>
                  <p class="card-kicker">Authentication</p>
                  <h3>认证与签名</h3>
                  <p class="card-description">如果启用鉴权，请填写主程序连接 MQTT Server 使用的管理员账号，并保持签名密钥与 OTA 一致。</p>
                </div>
                <el-tag :type="form.enable_auth ? 'warning' : 'info'" effect="plain" round>
                  {{ form.enable_auth ? t('auth_enabled') : t('anonymous_access') }}
                </el-tag>
              </div>
            </template>

            <div class="field-stack">
              <el-form-item label="启用认证" prop="enable_auth">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">校验 MQTT 用户名密码</div>
                    <div class="field-help">启用后会在客户端连接时校验用户名和密码。</div>
                  </div>
                  <el-switch v-model="form.enable_auth" />
                </div>
              </el-form-item>

              <el-form-item label="管理员用户" prop="username">
                <el-input v-model="form.username" placeholder="未启用鉴权可暂时留空" />
              </el-form-item>

              <el-form-item :label="t('admin_password')" prop="password">
                <el-input v-model="form.password" type="password" placeholder="未启用鉴权可暂时留空" show-password />
              </el-form-item>

              <el-form-item :label="t('signature_key')" prop="signature_key">
                <el-input v-model="form.signature_key" placeholder="请输入与 OTA 保持一致的签名密钥" />
                <div class="field-help">该密钥需要和 OTA 配置页的签名密钥保持一致。</div>
              </el-form-item>
            </div>
          </el-card>

          <el-card class="config-card" shadow="never">
            <template #header>
              <div class="card-head">
                <div>
                  <p class="card-kicker">MQTTS</p>
                  <h3>TLS 配置</h3>
                  <p class="card-description">需要设备通过 MQTTS 连接时，再启用 TLS 并补齐证书文件路径。</p>
                </div>
                <el-tag :type="form.tls.enable ? 'success' : 'info'" effect="plain" round>
                  {{ form.tls.enable ? t('tls_enabled') : t('tls_not_enabled') }}
                </el-tag>
              </div>
            </template>

            <div class="field-stack">
              <el-form-item label="启用 TLS" prop="tls.enable">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">允许设备通过 MQTTS 连接</div>
                    <div class="field-help">启用后请同时填写 TLS 端口、证书文件和密钥文件。</div>
                  </div>
                  <el-switch v-model="form.tls.enable" />
                </div>
              </el-form-item>

              <el-form-item label="TLS 端口" prop="tls.port">
                <el-input-number v-model="form.tls.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
              </el-form-item>

              <el-form-item label="证书文件" prop="tls.pem">
                <el-input v-model="form.tls.pem" placeholder="例如：certs/server.pem" />
              </el-form-item>

              <el-form-item label="密钥文件" prop="tls.key">
                <el-input v-model="form.tls.key" placeholder="例如：certs/server.key" />
              </el-form-item>
            </div>
          </el-card>
        </div>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          保存后会更新默认 MQTT Server 配置；如果同时启用 OTA 的 MQTT 下发，请确认签名密钥保持一致。
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
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)
const formRef = ref(null)

const createDefaultFormState = () => ({
  enable: true,
  listen_host: '0.0.0.0',
  listen_port: 1883,
  username: '',
  password: '',
  signature_key: 'xiaozhi_ota_signature_key',
  enable_auth: false,
  tls: {
    enable: false,
    port: 8883,
    pem: '',
    key: ''
  }
})

const form = reactive(createDefaultFormState())

const validateUsername = (_, value, callback) => {
  if (form.enable_auth && !String(value || '').trim()) {
    callback(new Error(t('auth_admin_username_required')))
    return
  }
  callback()
}

const validatePassword = (_, value, callback) => {
  if (form.enable_auth && !String(value || '').trim()) {
    callback(new Error(t('auth_admin_password_required')))
    return
  }
  callback()
}

const validateTlsPort = (_, value, callback) => {
  if (form.tls.enable && (!value || value < 1 || value > 65535)) {
    callback(new Error(t('tls_port_range')))
    return
  }
  callback()
}

const validateTlsPem = (_, value, callback) => {
  if (form.tls.enable && !String(value || '').trim()) {
    callback(new Error(t('tls_cert_required')))
    return
  }
  callback()
}

const validateTlsKey = (_, value, callback) => {
  if (form.tls.enable && !String(value || '').trim()) {
    callback(new Error(t('tls_key_required')))
    return
  }
  callback()
}

const rules = {
  listen_host: [{ required: true, message: t('enter_listen_host'), trigger: 'blur' }],
  listen_port: [
    { required: true, message: t('enter_listen_port'), trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: t('port_range_error'), trigger: 'blur' }
  ],
  username: [{ validator: validateUsername, trigger: 'blur' }],
  password: [{ validator: validatePassword, trigger: 'blur' }],
  signature_key: [{ required: true, message: t('enter_signature_key'), trigger: 'blur' }],
  'tls.port': [{ validator: validateTlsPort, trigger: 'blur' }],
  'tls.pem': [{ validator: validateTlsPem, trigger: 'blur' }],
  'tls.key': [{ validator: validateTlsKey, trigger: 'blur' }]
}

const serverReady = computed(() => {
  return Boolean(String(form.listen_host || '').trim() && Number(form.listen_port))
})

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const loadConfig = async () => {
  try {
    loading.value = true
    const response = await api.get('/admin/mqtt-server-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      const config = configs[0]
      configId.value = config.id

      let configData = {}
      try {
        configData = JSON.parse(config.json_data || '{}')
      } catch (error) {
        ElMessage.warning(t('mqtt_server_config_format_error'))
        configData = {}
      }

      form.enable = configData.enable !== undefined ? configData.enable : true
      form.listen_host = String(configData.listen_host || '0.0.0.0')
      form.listen_port = Number(configData.listen_port) > 0 ? Number(configData.listen_port) : 1883
      form.username = String(configData.username || '')
      form.password = String(configData.password || '')
      form.signature_key = String(configData.signature_key || 'xiaozhi_ota_signature_key')
      form.enable_auth = configData.enable_auth !== undefined ? configData.enable_auth : false
      form.tls.enable = configData.tls?.enable !== undefined ? configData.tls.enable : false
      form.tls.port = Number(configData.tls?.port) > 0 ? Number(configData.tls.port) : 8883
      form.tls.pem = String(configData.tls?.pem || '')
      form.tls.key = String(configData.tls?.key || '')
    } else {
      configId.value = null
      resetForm()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('load_mqtt_server_failed'))
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
    const payload = {
      name: t('mqtt_server_config_label'),
      config_id: 'mqtt_server_mqtt_server_config',
      provider: 'mqtt_server',
      json_data: JSON.stringify({
        enable: !!form.enable,
        listen_host: String(form.listen_host || '').trim(),
        listen_port: Number(form.listen_port),
        username: String(form.username || '').trim(),
        password: String(form.password || ''),
        signature_key: String(form.signature_key || '').trim(),
        enable_auth: !!form.enable_auth,
        tls: {
          enable: !!form.tls.enable,
          port: Number(form.tls.port),
          pem: String(form.tls.pem || '').trim(),
          key: String(form.tls.key || '').trim()
        }
      }),
      enabled: true,
      is_default: true
    }

    if (configId.value) {
      await api.put(`/admin/mqtt-server-configs/${configId.value}`, payload)
      ElMessage.success(t('mqtt_server_config_updated'))
    } else {
      const response = await api.post('/admin/mqtt-server-configs', payload)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success(t('mqtt_server_config_saved'))
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('save_mqtt_server_failed'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.mqtt-server-config {
  padding: 0 24px 32px;
}

.config-form {
  display: grid;
  gap: 24px;
}

.config-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(340px, 0.95fr);
  gap: 24px;
}

.side-stack {
  display: grid;
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

.card-kicker {
  display: block;
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--apple-text-tertiary);
}

.card-head h3 {
  margin: 8px 0 0;
  font-size: 22px;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: var(--apple-text);
}

.card-description,
.field-help,
.footer-note {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--apple-text-secondary);
}

.field-grid {
  display: grid;
  gap: 20px 18px;
}

.field-stack {
  display: grid;
  gap: 20px;
}

.switch-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px 18px;
  align-items: center;
}

.switch-title {
  font-size: 15px;
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
  max-width: 640px;
  margin: 0;
}

.footer-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
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
  .mqtt-server-config {
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

  .footer-actions {
    justify-content: stretch;
  }

  .footer-actions :deep(.el-button) {
    flex: 1;
  }
}
</style>
