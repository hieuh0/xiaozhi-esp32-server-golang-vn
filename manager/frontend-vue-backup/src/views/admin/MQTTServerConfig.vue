<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'
import { Switch } from '@/components/ui/switch'

const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)

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

const serverReady = computed(() => {
  return Boolean(String(form.listen_host || '').trim() && Number(form.listen_port))
})

const validate = () => {
  if (!String(form.listen_host || '').trim()) { ElMessage.error(t('enter_listen_host')); return false }
  if (!form.listen_port || form.listen_port < 1 || form.listen_port > 65535) { ElMessage.error(t('port_range_error')); return false }
  if (!String(form.signature_key || '').trim()) { ElMessage.error(t('enter_signature_key')); return false }
  if (form.enable_auth) {
    if (!String(form.username || '').trim()) { ElMessage.error(t('auth_admin_username_required')); return false }
    if (!String(form.password || '').trim()) { ElMessage.error(t('auth_admin_password_required')); return false }
  }
  if (form.tls.enable) {
    if (!form.tls.port || form.tls.port < 1 || form.tls.port > 65535) { ElMessage.error(t('tls_port_range')); return false }
    if (!String(form.tls.pem || '').trim()) { ElMessage.error(t('tls_cert_required')); return false }
    if (!String(form.tls.key || '').trim()) { ElMessage.error(t('tls_key_required')); return false }
  }
  return true
}

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/mqtt-server-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      const config = configs[0]
      configId.value = config.id

      let configData = {}
      try {
        configData = JSON.parse(config.json_data || '{}')
      } catch {
        ElMessage.warning(t('mqtt_server_config_format_error'))
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
  if (!validate()) return

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

onMounted(() => { loadConfig() })
</script>

<template>
  <div class="grid gap-6 px-6 pb-8">
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <div v-else class="grid gap-6" style="grid-template-columns: minmax(0,1.25fr) minmax(340px,0.95fr);">

      <!-- Server card -->
      <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
          <div>
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">MQTT Server</p>
            <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('listen_and_access') }}</h3>
            <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('mqtt_server_config_desc') }}</p>
          </div>
          <span :class="serverReady
            ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800'
            : 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800'"
            class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1">
            {{ serverReady ? t('service_params_complete') : t('pending_fill') }}
          </span>
        </div>
        <div class="p-6 grid gap-5">
          <!-- Enable switch -->
          <div class="flex items-center justify-between gap-4">
            <div>
              <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('enable_mqtt_server') }}</p>
              <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ t('enable_mqtt_server_help') }}</p>
            </div>
            <Switch :checked="form.enable" @update:checked="v => form.enable = v" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('listen_host') }}</label>
            <Input v-model="form.listen_host" :placeholder="t('enter_listen_host_eg')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('listen_port') }}</label>
            <NumberInput v-model="form.listen_port" :min="1" :max="65535" />
          </div>
        </div>
      </div>

      <!-- Side stack -->
      <div class="grid gap-6 content-start">

        <!-- Authentication card -->
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
          <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">Authentication</p>
              <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('auth_and_signing') }}</h3>
              <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('auth_signing_desc') }}</p>
            </div>
            <span :class="form.enable_auth
              ? 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800'
              : 'bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)] border-[var(--color-line)]'"
              class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1">
              {{ form.enable_auth ? t('auth_enabled') : t('anonymous_access') }}
            </span>
          </div>
          <div class="p-6 grid gap-5">
            <div class="flex items-center justify-between gap-4">
              <div>
                <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('verify_mqtt_auth') }}</p>
                <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ t('verify_mqtt_auth_help') }}</p>
              </div>
              <Switch :checked="form.enable_auth" @update:checked="v => form.enable_auth = v" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('admin_user') }}</label>
              <Input v-model="form.username" :placeholder="t('username_no_auth_temp')" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('admin_password') }}</label>
              <Input v-model="form.password" type="password" :placeholder="t('username_no_auth_temp')" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('signature_key') }}</label>
              <Input v-model="form.signature_key" :placeholder="t('signature_key_ph')" />
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('signature_key_note_ota') }}</p>
            </div>
          </div>
        </div>

        <!-- TLS card -->
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
          <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
            <div>
              <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">MQTTS</p>
              <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('tls_config') }}</h3>
              <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('tls_config_desc') }}</p>
            </div>
            <span :class="form.tls.enable
              ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800'
              : 'bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)] border-[var(--color-line)]'"
              class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1">
              {{ form.tls.enable ? t('tls_enabled') : t('tls_not_enabled') }}
            </span>
          </div>
          <div class="p-6 grid gap-5">
            <div class="flex items-center justify-between gap-4">
              <div>
                <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('allow_devices_mqtts') }}</p>
                <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ t('allow_devices_mqtts_help') }}</p>
              </div>
              <Switch :checked="form.tls.enable" @update:checked="v => form.tls.enable = v" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('tls_port') }}</label>
              <NumberInput v-model="form.tls.port" :min="1" :max="65535" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('cert_file') }}</label>
              <Input v-model="form.tls.pem" :placeholder="t('cert_file_ph')" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('key_file') }}</label>
              <Input v-model="form.tls.key" :placeholder="t('key_file_ph')" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between gap-4">
      <p class="text-sm text-[var(--color-text-secondary)] max-w-[640px]">{{ t('mqtt_server_save_hint') }}</p>
      <div class="flex items-center gap-3">
        <Button variant="outline" :disabled="loading" @click="loadConfig">{{ t('reset_to_current') }}</Button>
        <Button :disabled="saving" @click="handleSave">{{ t('save_config') }}</Button>
      </div>
    </div>
  </div>
</template>
