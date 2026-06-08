<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)

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

const validate = () => {
  if (!String(form.name || '').trim()) { ElMessage.error(t('enter_config_name')); return false }
  if (!String(form.broker || '').trim()) { ElMessage.error(t('enter_mqtt_broker')); return false }
  if (!form.type) { ElMessage.error(t('select_connection_type')); return false }
  if (!form.port || form.port < 1 || form.port > 65535) { ElMessage.error(t('port_range_error')); return false }
  if (!String(form.client_id || '').trim()) { ElMessage.error(t('enter_client_id')); return false }
  return true
}

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const generateConfig = () => ({
  enable: form.enable,
  broker: String(form.broker || '').trim(),
  type: form.type,
  port: Number(form.port),
  client_id: String(form.client_id || '').trim(),
  username: String(form.username || '').trim(),
  password: String(form.password || '')
})

const applyLoadedConfig = (config) => {
  configId.value = config?.id || null
  form.name = config?.name || t('mqtt_config_label')
  form.is_default = config?.is_default ?? true

  let configData = {}
  try {
    configData = JSON.parse(config?.json_data || '{}')
  } catch {
    ElMessage.warning(t('mqtt_config_format_error'))
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
  } catch {
    ElMessage.error(t('load_mqtt_config_failed'))
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!validate()) return

  saving.value = true
  try {
    const generatedConfigId = `mqtt_${String(form.name || '').replace(/[^a-zA-Z0-9]/g, '_').toLowerCase()}`
    const nextConfigPayload = generateConfig()
    let configData
    let isUpdate = false

    if (configId.value) {
      const response = await api.get('/admin/mqtt-configs')
      const configs = response.data?.data || []
      const existingConfig = configs.find(item => item.id === configId.value)

      if (existingConfig) {
        let existingData = {}
        try { existingData = JSON.parse(existingConfig.json_data || '{}') } catch { existingData = {} }
        configData = {
          name: form.name,
          config_id: generatedConfigId,
          is_default: true,
          json_data: JSON.stringify({ ...existingData, ...nextConfigPayload })
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

onMounted(() => { loadConfig() })
</script>

<template>
  <div class="grid gap-6 px-6 pb-8">
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <div v-else class="grid gap-6" style="grid-template-columns: minmax(0,1.45fr) minmax(320px,0.95fr);">

      <!-- Connection card -->
      <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
          <div>
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">Connection</p>
            <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('connection_params') }}</h3>
            <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('broker_setup_hint') }}</p>
          </div>
          <span :class="isCoreFieldsComplete
            ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800'
            : 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800'"
            class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1">
            {{ isCoreFieldsComplete ? t('params_complete') : t('pending_fill') }}
          </span>
        </div>
        <div class="p-6 grid grid-cols-2 gap-5">
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('config_name') }}</label>
            <Input v-model="form.name" :placeholder="t('mqtt_name_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('broker_address') }}</label>
            <Input v-model="form.broker" :placeholder="t('broker_address_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('connection_type') }}</label>
            <Select :model-value="form.type" @update:model-value="v => form.type = v">
              <SelectTrigger>
                <SelectValue :placeholder="t('select_connection_type')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in connectionTypeOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('port') }}</label>
            <NumberInput v-model="form.port" :min="1" :max="65535" />
          </div>
          <div class="grid gap-1.5 col-span-2">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('client_id_label') }}</label>
            <Input v-model="form.client_id" :placeholder="t('client_id_ph')" />
            <p class="text-xs text-[var(--color-text-secondary)]">{{ t('client_id_help') }}</p>
          </div>
        </div>
      </div>

      <!-- Authentication card -->
      <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
          <div>
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">Authentication</p>
            <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('auth_info') }}</h3>
            <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('auth_info_desc') }}</p>
          </div>
          <span :class="hasCredentials
            ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800'
            : 'bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)] border-[var(--color-line)]'"
            class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1">
            {{ hasCredentials ? t('credentials_filled') : t('can_be_empty') }}
          </span>
        </div>
        <div class="p-6 grid gap-5">
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('username') }}</label>
            <Input v-model="form.username" :placeholder="t('username_no_auth')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('password') }}</label>
            <Input v-model="form.password" type="password" :placeholder="t('username_no_auth')" />
          </div>
          <div class="grid gap-3.5 pt-4 border-t border-[var(--color-line)]">
            <div class="grid gap-1">
              <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('auth_note') }}</span>
              <p class="text-sm text-[var(--color-text-secondary)]">{{ t('auth_note_text') }}</p>
            </div>
            <div class="grid gap-1">
              <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('protocol_reminder') }}</span>
              <p class="text-sm text-[var(--color-text-secondary)]">{{ t('protocol_reminder_text') }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between gap-4">
      <p class="text-sm text-[var(--color-text-secondary)] max-w-[620px]">{{ t('mqtt_save_hint') }}</p>
      <div class="flex items-center gap-3">
        <Button variant="outline" :disabled="loading" @click="loadConfig">{{ t('reset_to_current') }}</Button>
        <Button :disabled="saving" @click="handleSave">{{ t('save_config') }}</Button>
      </div>
    </div>
  </div>
</template>
