<template>
  <div class="ota-config">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      class="config-form"
      v-loading="loading"
    >
      <el-card class="config-card" shadow="never">
        <template #header>
          <div class="card-head">
            <div>
              <p class="card-kicker">OTA Base</p>
              <h3>{{ t('signature_constraints') }}</h3>
              <p class="card-description">{{ t('ota_mqtt_password_hint') }}</p>
            </div>
          </div>
        </template>

        <div class="field-grid">
          <el-form-item :label="t('signature_key')" prop="signature_key" class="field-span-full">
            <el-input v-model="form.signature_key" :placeholder="t('ota_sig_key_ph')" show-password />
            <div class="field-help">{{ t('ota_sig_key_hint') }}</div>
          </el-form-item>
        </div>
      </el-card>

      <div class="environment-grid">
        <el-card class="config-card" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Test</p>
                <h3>{{ t('test_env_delivery') }}</h3>
                <p class="card-description">{{ t('test_env_desc') }}</p>
              </div>
              <div class="card-actions">
                <el-tag type="warning" effect="plain" round>{{ t('test_env_tag') }}</el-tag>
                <el-button size="small" :loading="otaTestingTest" @click="testOtaEnv('test')">{{ t('test_env_tag') }}</el-button>
              </div>
            </div>
          </template>

          <div class="section-stack">
            <section class="config-section">
              <div class="section-title">{{ t('ws_delivery') }}</div>
              <el-form-item label="WebSocket URL" prop="test.websocket.url">
                <el-input v-model="form.test.websocket.url" :placeholder="t('ws_test_url_ph')" />
              </el-form-item>
            </section>

            <section class="config-section">
              <div class="section-title">{{ t('mqtt_delivery') }}</div>
              <el-form-item :label="t('mqtt_enabled_status')">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">{{ t('priority_mqtt') }}</div>
                    <div class="field-help">{{ t('priority_mqtt_help') }}</div>
                  </div>
                  <el-switch v-model="form.test.mqtt.enable" />
                </div>
              </el-form-item>

              <el-form-item :label="t('mqtt_endpoint')" prop="test.mqtt.endpoint">
                <el-input
                  v-model="form.test.mqtt.endpoint"
                  :disabled="!form.test.mqtt.enable"
                  :placeholder="t('mqtt_test_endpoint_ph')"
                />
                <div class="field-help">{{ t('mqtt_udp_note') }}</div>
              </el-form-item>
            </section>
          </div>
        </el-card>

        <el-card class="config-card" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">External</p>
                <h3>{{ t('external_env_delivery') }}</h3>
                <p class="card-description">{{ t('external_env_desc') }}</p>
              </div>
              <div class="card-actions">
                <el-tag type="success" effect="plain" round>{{ t('prod_env_tag') }}</el-tag>
                <el-button size="small" :loading="otaTestingExternal" @click="testOtaEnv('external')">{{ t('test_env_tag') }}</el-button>
              </div>
            </div>
          </template>

          <div class="section-stack">
            <section class="config-section">
              <div class="section-title">{{ t('ws_delivery') }}</div>
              <el-form-item label="WebSocket URL" prop="external.websocket.url">
                <el-input v-model="form.external.websocket.url" :placeholder="t('wss_external_url_ph')" />
              </el-form-item>
            </section>

            <section class="config-section">
              <div class="section-title">{{ t('mqtt_delivery') }}</div>
              <el-form-item :label="t('mqtt_enabled_status')">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">{{ t('production_mqtt') }}</div>
                    <div class="field-help">{{ t('production_mqtt_help') }}</div>
                  </div>
                  <el-switch v-model="form.external.mqtt.enable" />
                </div>
              </el-form-item>

              <el-form-item :label="t('mqtt_endpoint')" prop="external.mqtt.endpoint">
                <el-input
                  v-model="form.external.mqtt.endpoint"
                  :disabled="!form.external.mqtt.enable"
                  :placeholder="t('mqtt_external_endpoint_ph')"
                />
              </el-form-item>
            </section>
          </div>
        </el-card>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          {{ t('ota_save_hint') }}
        </p>
        <div class="footer-actions">
          <el-button plain :loading="loading" @click="loadConfig">{{ t('reset_to_current') }}</el-button>
          <el-button type="primary" :loading="saving" @click="saveConfig">{{ t('save_config') }}</el-button>
        </div>
      </div>
    </el-form>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const otaTestingTest = ref(false)
const otaTestingExternal = ref(false)
const configId = ref(null)
const formRef = ref()

const createDefaultState = () => ({
  signature_key: 'xiaozhi_ota_signature_key',
  test: {
    websocket: {
      url: 'ws://127.0.0.1:8989/xiaozhi/v1/'
    },
    mqtt: {
      enable: true,
      endpoint: '127.0.0.1:1883'
    }
  },
  external: {
    websocket: {
      url: 'ws://127.0.0.1:8989/xiaozhi/v1/'
    },
    mqtt: {
      enable: false,
      endpoint: '127.0.0.1:1883'
    }
  }
})

const form = reactive(createDefaultState())

const rules = {
  signature_key: [
    { required: true, message: t('enter_signature_key'), trigger: 'blur' }
  ],
  'test.websocket.url': [
    { required: true, message: t('enter_test_ws_url'), trigger: 'blur' }
  ],
  'test.mqtt.endpoint': [
    {
      validator: (_, value, callback) => {
        if (form.test.mqtt.enable && !String(value || '').trim()) {
          callback(new Error(t('mqtt_endpoint_required')))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ],
  'external.websocket.url': [
    { required: true, message: t('enter_external_ws_url'), trigger: 'blur' }
  ],
  'external.mqtt.endpoint': [
    {
      validator: (_, value, callback) => {
        if (form.external.mqtt.enable && !String(value || '').trim()) {
          callback(new Error(t('mqtt_endpoint_required')))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}

const applyState = (state) => {
  form.signature_key = state.signature_key
  form.test.websocket.url = state.test.websocket.url
  form.test.mqtt.enable = state.test.mqtt.enable
  form.test.mqtt.endpoint = state.test.mqtt.endpoint
  form.external.websocket.url = state.external.websocket.url
  form.external.mqtt.enable = state.external.mqtt.enable
  form.external.mqtt.endpoint = state.external.mqtt.endpoint
}

const buildConfigObject = () => ({
  signature_key: String(form.signature_key || '').trim(),
  test: {
    websocket: {
      url: String(form.test.websocket.url || '').trim()
    },
    mqtt: {
      enable: !!form.test.mqtt.enable,
      endpoint: String(form.test.mqtt.endpoint || '').trim()
    }
  },
  external: {
    websocket: {
      url: String(form.external.websocket.url || '').trim()
    },
    mqtt: {
      enable: !!form.external.mqtt.enable,
      endpoint: String(form.external.mqtt.endpoint || '').trim()
    }
  }
})

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/ota-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      const config = configs[0]
      configId.value = config.id

      try {
        const configData = JSON.parse(config.json_data || '{}')
        applyState({
          signature_key: configData.signature_key || 'xiaozhi_ota_signature_key',
          test: {
            websocket: {
              url: configData.test?.websocket?.url || 'ws://127.0.0.1:8989/xiaozhi/v1/'
            },
            mqtt: {
              enable: configData.test?.mqtt?.enable !== undefined ? configData.test.mqtt.enable : true,
              endpoint: configData.test?.mqtt?.endpoint || '127.0.0.1:1883'
            }
          },
          external: {
            websocket: {
              url: configData.external?.websocket?.url || 'ws://127.0.0.1:8989/xiaozhi/v1/'
            },
            mqtt: {
              enable: configData.external?.mqtt?.enable !== undefined ? configData.external.mqtt.enable : false,
              endpoint: configData.external?.mqtt?.endpoint || '127.0.0.1:1883'
            }
          }
        })
      } catch (error) {
        ElMessage.warning(t('ota_config_format_error'))
        applyState(createDefaultState())
      }
    } else {
      configId.value = null
      applyState(createDefaultState())
    }
  } catch (error) {
    ElMessage.error(t('load_ota_config_failed'))
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    const configData = {
      name: t('ota_config_label'),
      config_id: 'ota_ota_config',
      json_data: JSON.stringify(buildConfigObject()),
      enabled: true,
      is_default: true
    }

    if (configId.value) {
      await api.put(`/admin/ota-configs/${configId.value}`, configData)
      ElMessage.success(t('ota_config_updated'))
    } else {
      const response = await api.post('/admin/ota-configs', configData)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success(t('ota_config_saved'))
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('save_ota_failed'))
  } finally {
    saving.value = false
  }
}

const testOtaEnv = async (env) => {
  const envConfig = env === 'test' ? form.test : form.external
  const mqttEnabled = envConfig.mqtt.enable
  const payload = buildConfigObject()
  const loadingRef = env === 'test' ? otaTestingTest : otaTestingExternal

  loadingRef.value = true
  try {
    const body = { types: ['ota'], data: { ota: { ota_ota_config: payload } } }
    const res = await api.post('/admin/configs/test', body, { timeout: 30000 })
    const data = res.data?.data ?? res.data
    const otaResult = data?.ota?.ota_ota_config
    const label = env === 'test' ? t('test_env') : t('external_env')

    if (!otaResult) {
      ElMessage.error(t('label_no_result', { label }))
      return
    }

    const wsResult = otaResult.websocket || {}
    const wsOk = wsResult.ok || false
    const wsMsg = wsResult.message || t('websocket_test_failed')
    const wsMs = wsResult.first_packet_ms

    const mqttResult = otaResult.mqtt_udp
    let mqttOk = true
    let mqttMsg = ''
    let mqttMs = 0

    if (mqttEnabled && mqttResult) {
      mqttOk = mqttResult.ok || false
      mqttMsg = mqttResult.message || t('mqtt_udp_test_failed')
      mqttMs = mqttResult.first_packet_ms || 0
    } else if (mqttEnabled) {
      mqttOk = false
      mqttMsg = t('mqtt_udp_no_result')
    }

    let message = wsOk ? `WebSocket: ${wsMsg}` : `WebSocket: ${wsMsg}`
    if (wsMs != null) message += ` (${wsMs}ms)`

    if (mqttEnabled) {
      message += ' | '
      message += `MQTT UDP: ${mqttMsg}`
      if (mqttMs != null) message += ` (${mqttMs}ms)`
    }

    if (wsOk && (!mqttEnabled || mqttOk)) {
      ElMessage.success(`${label}：${message}`)
    } else {
      ElMessage.warning(`${label}：${message}`)
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('test_request_failed_v2'))
  } finally {
    loadingRef.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.ota-config {
  padding: 0 24px 32px;
}

.config-form {
  display: grid;
  gap: 24px;
}

.environment-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
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

.card-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
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

.field-span-full {
  grid-column: 1 / -1;
}

.section-stack {
  display: grid;
  gap: 24px;
}

.config-section {
  display: grid;
  gap: 18px;
}

.config-section + .config-section {
  padding-top: 24px;
  border-top: 1px solid rgba(229, 229, 234, 0.72);
}

.section-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--apple-text);
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
  max-width: 700px;
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
  .environment-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .ota-config {
    padding: 0 16px 24px;
  }

  :deep(.el-card__body) {
    padding: 20px;
  }

  :deep(.el-card__header) {
    padding: 20px 20px 0;
  }

  .card-head,
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
