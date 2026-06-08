<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const otaTestingTest = ref(false)
const otaTestingExternal = ref(false)
const configId = ref(null)

const createDefaultState = () => ({
  signature_key: 'xiaozhi_ota_signature_key',
  test: {
    websocket: { url: 'ws://127.0.0.1:8989/xiaozhi/v1/' },
    mqtt: { enable: true, endpoint: '127.0.0.1:1883' }
  },
  external: {
    websocket: { url: 'ws://127.0.0.1:8989/xiaozhi/v1/' },
    mqtt: { enable: false, endpoint: '127.0.0.1:1883' }
  }
})

const form = reactive(createDefaultState())

const validate = () => {
  if (!String(form.signature_key || '').trim()) { ElMessage.error(t('enter_signature_key')); return false }
  if (!String(form.test.websocket.url || '').trim()) { ElMessage.error(t('enter_test_ws_url')); return false }
  if (form.test.mqtt.enable && !String(form.test.mqtt.endpoint || '').trim()) { ElMessage.error(t('mqtt_endpoint_required')); return false }
  if (!String(form.external.websocket.url || '').trim()) { ElMessage.error(t('enter_external_ws_url')); return false }
  if (form.external.mqtt.enable && !String(form.external.mqtt.endpoint || '').trim()) { ElMessage.error(t('mqtt_endpoint_required')); return false }
  return true
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
    websocket: { url: String(form.test.websocket.url || '').trim() },
    mqtt: { enable: !!form.test.mqtt.enable, endpoint: String(form.test.mqtt.endpoint || '').trim() }
  },
  external: {
    websocket: { url: String(form.external.websocket.url || '').trim() },
    mqtt: { enable: !!form.external.mqtt.enable, endpoint: String(form.external.mqtt.endpoint || '').trim() }
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
            websocket: { url: configData.test?.websocket?.url || 'ws://127.0.0.1:8989/xiaozhi/v1/' },
            mqtt: {
              enable: configData.test?.mqtt?.enable !== undefined ? configData.test.mqtt.enable : true,
              endpoint: configData.test?.mqtt?.endpoint || '127.0.0.1:1883'
            }
          },
          external: {
            websocket: { url: configData.external?.websocket?.url || 'ws://127.0.0.1:8989/xiaozhi/v1/' },
            mqtt: {
              enable: configData.external?.mqtt?.enable !== undefined ? configData.external.mqtt.enable : false,
              endpoint: configData.external?.mqtt?.endpoint || '127.0.0.1:1883'
            }
          }
        })
      } catch {
        ElMessage.warning(t('ota_config_format_error'))
        applyState(createDefaultState())
      }
    } else {
      configId.value = null
      applyState(createDefaultState())
    }
  } catch {
    ElMessage.error(t('load_ota_config_failed'))
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  if (!validate()) return

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

    let message = `WebSocket: ${wsMsg}`
    if (wsMs != null) message += ` (${wsMs}ms)`

    if (mqttEnabled) {
      message += ` | MQTT UDP: ${mqttMsg}`
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

onMounted(() => { loadConfig() })
</script>

<template>
  <div class="grid gap-6 px-6 pb-8">
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <template v-else>

      <!-- Signature key card -->
      <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        <div class="px-6 pt-6 pb-0">
          <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">OTA Base</p>
          <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('signature_constraints') }}</h3>
          <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('ota_mqtt_password_hint') }}</p>
        </div>
        <div class="p-6">
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('signature_key') }}</label>
            <Input v-model="form.signature_key" type="password" :placeholder="t('ota_sig_key_ph')" />
            <p class="text-xs text-[var(--color-text-secondary)]">{{ t('ota_sig_key_hint') }}</p>
          </div>
        </div>
      </div>

      <!-- Environment grid -->
      <div class="grid gap-6" style="grid-template-columns: repeat(2, minmax(0,1fr));">

        <!-- Test environment card -->
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
          <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">Test</p>
              <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('test_env_delivery') }}</h3>
              <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('test_env_desc') }}</p>
            </div>
            <div class="flex items-center gap-2 shrink-0 mt-1">
              <span class="inline-flex items-center rounded-full border border-yellow-200 bg-yellow-50 px-2.5 py-0.5 text-xs font-semibold text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800">
                {{ t('test_env_tag') }}
              </span>
              <Button size="sm" variant="outline" :disabled="otaTestingTest" @click="testOtaEnv('test')">
                {{ otaTestingTest ? '...' : t('test_env_tag') }}
              </Button>
            </div>
          </div>
          <div class="p-6 grid gap-6">
            <!-- WebSocket section -->
            <div class="grid gap-3">
              <p class="text-sm font-bold text-[var(--color-text)]">{{ t('ws_delivery') }}</p>
              <div class="grid gap-1.5">
                <label class="text-sm font-semibold text-[var(--color-text)]">WebSocket URL</label>
                <Input v-model="form.test.websocket.url" :placeholder="t('ws_test_url_ph')" />
              </div>
            </div>
            <!-- MQTT section -->
            <div class="grid gap-3 pt-6 border-t border-[var(--color-line)]">
              <p class="text-sm font-bold text-[var(--color-text)]">{{ t('mqtt_delivery') }}</p>
              <div class="flex items-center justify-between gap-4">
                <div>
                  <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('priority_mqtt') }}</p>
                  <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ t('priority_mqtt_help') }}</p>
                </div>
                <Switch :checked="form.test.mqtt.enable" @update:checked="v => form.test.mqtt.enable = v" />
              </div>
              <div class="grid gap-1.5">
                <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('mqtt_endpoint') }}</label>
                <Input v-model="form.test.mqtt.endpoint" :disabled="!form.test.mqtt.enable" :placeholder="t('mqtt_test_endpoint_ph')" />
                <p class="text-xs text-[var(--color-text-secondary)]">{{ t('mqtt_udp_note') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- External environment card -->
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
          <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">External</p>
              <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('external_env_delivery') }}</h3>
              <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('external_env_desc') }}</p>
            </div>
            <div class="flex items-center gap-2 shrink-0 mt-1">
              <span class="inline-flex items-center rounded-full border border-green-200 bg-green-50 px-2.5 py-0.5 text-xs font-semibold text-green-700 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800">
                {{ t('prod_env_tag') }}
              </span>
              <Button size="sm" variant="outline" :disabled="otaTestingExternal" @click="testOtaEnv('external')">
                {{ otaTestingExternal ? '...' : t('test_env_tag') }}
              </Button>
            </div>
          </div>
          <div class="p-6 grid gap-6">
            <!-- WebSocket section -->
            <div class="grid gap-3">
              <p class="text-sm font-bold text-[var(--color-text)]">{{ t('ws_delivery') }}</p>
              <div class="grid gap-1.5">
                <label class="text-sm font-semibold text-[var(--color-text)]">WebSocket URL</label>
                <Input v-model="form.external.websocket.url" :placeholder="t('wss_external_url_ph')" />
              </div>
            </div>
            <!-- MQTT section -->
            <div class="grid gap-3 pt-6 border-t border-[var(--color-line)]">
              <p class="text-sm font-bold text-[var(--color-text)]">{{ t('mqtt_delivery') }}</p>
              <div class="flex items-center justify-between gap-4">
                <div>
                  <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('production_mqtt') }}</p>
                  <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ t('production_mqtt_help') }}</p>
                </div>
                <Switch :checked="form.external.mqtt.enable" @update:checked="v => form.external.mqtt.enable = v" />
              </div>
              <div class="grid gap-1.5">
                <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('mqtt_endpoint') }}</label>
                <Input v-model="form.external.mqtt.endpoint" :disabled="!form.external.mqtt.enable" :placeholder="t('mqtt_external_endpoint_ph')" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between gap-4">
        <p class="text-sm text-[var(--color-text-secondary)] max-w-[700px]">{{ t('ota_save_hint') }}</p>
        <div class="flex items-center gap-3">
          <Button variant="outline" :disabled="loading" @click="loadConfig">{{ t('reset_to_current') }}</Button>
          <Button :disabled="saving" @click="saveConfig">{{ t('save_config') }}</Button>
        </div>
      </div>
    </template>
  </div>
</template>
