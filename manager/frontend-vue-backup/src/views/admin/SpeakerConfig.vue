<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'
import { Switch } from '@/components/ui/switch'
import { Info } from '@lucide/vue'

const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const currentConfig = ref(null)

const form = reactive({
  base_url: 'http://192.168.208.214:8080',
  threshold: 0.4,
  enabled: true
})

const validate = () => {
  if (!String(form.base_url || '').trim()) { ElMessage.error(t('enter_service_address')); return false }
  if (!/^https?:\/\/.+/.test(form.base_url)) { ElMessage.error(t('enter_valid_http_address')); return false }
  if (form.threshold === undefined || form.threshold === null || form.threshold < 0 || form.threshold > 1) {
    ElMessage.error(t('threshold_0_to_1_decimal')); return false
  }
  return true
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/speaker-configs')
    const configs = response.data.data || []

    if (configs.length > 0) {
      currentConfig.value = configs[0]
      const configObj = JSON.parse(configs[0].json_data || '{}')

      if (configObj.service && configObj.service.base_url) {
        form.base_url = configObj.service.base_url
      } else if (configObj.base_url) {
        form.base_url = configObj.base_url
      }

      if (configObj.service && configObj.service.threshold !== undefined) {
        form.threshold = configObj.service.threshold
      } else if (configObj.threshold !== undefined) {
        form.threshold = configObj.threshold
      } else {
        form.threshold = 0.4
      }

      form.enabled = configObj.enable !== undefined ? configObj.enable : true
    }
  } catch {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!validate()) return

  saving.value = true
  try {
    const configData = {
      service: { base_url: form.base_url, threshold: form.threshold },
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
      await api.put(`/admin/speaker-configs/${currentConfig.value.id}`, saveData)
      ElMessage.success(t('config_update_success'))
    } else {
      await api.post('/admin/speaker-configs', saveData)
      ElMessage.success(t('config_create_success'))
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(t('save_failed_colon') + (error.response?.data?.message || error.message))
  } finally {
    saving.value = false
  }
}

onMounted(() => { loadConfig() })
</script>

<template>
  <div class="px-6 pb-8">
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <div v-else class="max-w-[800px] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
      <div class="px-6 pt-6 pb-0">
        <h3 class="text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('voiceprint_recognition_config') }}</h3>
      </div>

      <!-- Info alert -->
      <div class="mx-6 mt-5 flex gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 dark:border-blue-800 dark:bg-blue-900/20">
        <Info class="w-4 h-4 text-blue-500 shrink-0 mt-0.5" />
        <p class="text-sm text-blue-700 dark:text-blue-300">{{ t('docker_env_hint') }}</p>
      </div>

      <div class="p-6 grid gap-5">
        <div class="grid gap-1.5">
          <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('service_address') }}</label>
          <Input v-model="form.base_url" :placeholder="t('http_service_address_ph')" />
          <p class="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('http_to_ws_hint') }}
          </p>
        </div>

        <div class="grid gap-1.5">
          <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('recognition_threshold') }}</label>
          <NumberInput v-model="form.threshold" :min="0" :max="1" :step="0.1" :precision="2" />
          <p class="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('recognition_threshold_hint') }}
          </p>
        </div>

        <div class="flex items-center justify-between gap-4">
          <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('enabled_status') }}</label>
          <Switch :checked="form.enabled" @update:checked="v => form.enabled = v" />
        </div>

        <div class="pt-5 border-t border-[var(--color-line)]">
          <Button :disabled="saving" @click="handleSave">{{ t('save_config') }}</Button>
        </div>
      </div>
    </div>
  </div>
</template>
