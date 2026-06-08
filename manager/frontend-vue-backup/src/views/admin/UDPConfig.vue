<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'

const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)

const createDefaultFormState = () => ({
  name: t('udp_config_label'),
  is_default: true,
  external_host: '192.168.0.208',
  external_port: 8990,
  listen_host: '0.0.0.0',
  listen_port: 8990
})

const form = reactive(createDefaultFormState())

const listenReady = computed(() => {
  return Boolean(String(form.listen_host || '').trim() && Number(form.listen_port))
})

const externalReady = computed(() => {
  return Boolean(String(form.external_host || '').trim() && Number(form.external_port))
})

const validate = () => {
  if (!String(form.name || '').trim()) { ElMessage.error(t('enter_config_name')); return false }
  if (!String(form.listen_host || '').trim()) { ElMessage.error(t('enter_listen_host')); return false }
  if (!form.listen_port || form.listen_port < 1 || form.listen_port > 65535) { ElMessage.error(t('port_range_error')); return false }
  if (!String(form.external_host || '').trim()) { ElMessage.error(t('enter_external_host')); return false }
  if (!form.external_port || form.external_port < 1 || form.external_port > 65535) { ElMessage.error(t('port_range_error')); return false }
  return true
}

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/udp-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      const config = configs[0]
      configId.value = config.id

      let configData = {}
      try {
        configData = JSON.parse(config.json_data || '{}')
      } catch {
        ElMessage.warning(t('udp_config_format_error'))
      }

      form.name = config.name || t('udp_config_label')
      form.is_default = config.is_default ?? true
      form.external_host = String(configData.external_host || '192.168.0.208')
      form.external_port = Number(configData.external_port) > 0 ? Number(configData.external_port) : 8990
      form.listen_host = String(configData.listen_host || '0.0.0.0')
      form.listen_port = Number(configData.listen_port) > 0 ? Number(configData.listen_port) : 8990
    } else {
      configId.value = null
      resetForm()
    }
  } catch {
    ElMessage.error(t('load_udp_config_failed'))
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!validate()) return

  saving.value = true
  try {
    const payload = {
      name: form.name,
      config_id: `udp_${String(form.name || '').replace(/[^a-zA-Z0-9]/g, '_').toLowerCase()}`,
      is_default: form.is_default,
      json_data: JSON.stringify({
        external_host: String(form.external_host || '').trim(),
        external_port: Number(form.external_port),
        listen_host: String(form.listen_host || '').trim(),
        listen_port: Number(form.listen_port)
      })
    }

    if (configId.value) {
      await api.put(`/admin/udp-configs/${configId.value}`, payload)
      ElMessage.success(t('udp_config_updated'))
    } else {
      const response = await api.post('/admin/udp-configs', payload)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success(t('udp_config_saved'))
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || t('save_udp_failed'))
  } finally {
    saving.value = false
  }
}

onMounted(() => { loadConfig() })
</script>

<template>
  <div class="grid gap-6 px-6 pb-8">
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <div v-else class="grid gap-6" style="grid-template-columns: minmax(0,1.4fr) minmax(320px,0.9fr);">

      <!-- Listen card -->
      <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
          <div>
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">UDP Server</p>
            <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('service_listen') }}</h3>
            <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('udp_server_config_desc') }}</p>
          </div>
          <span :class="listenReady
            ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800'
            : 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800'"
            class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1">
            {{ listenReady ? t('listen_params_complete') : t('pending_fill') }}
          </span>
        </div>
        <div class="p-6 grid grid-cols-2 gap-5">
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('config_name') }}</label>
            <Input v-model="form.name" :placeholder="t('udp_name_ph')" />
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

      <!-- Announce address card -->
      <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
        <div class="flex items-start justify-between px-6 pt-6 pb-0 gap-4">
          <div>
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)]">Announce Address</p>
            <h3 class="mt-2 text-xl font-semibold tracking-tight text-[var(--color-text)]">{{ t('terminal_publish_addr') }}</h3>
            <p class="mt-1 text-sm text-[var(--color-text-secondary)]">{{ t('terminal_publish_desc') }}</p>
          </div>
          <span :class="externalReady
            ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-800'
            : 'bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-400 dark:border-yellow-800'"
            class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold shrink-0 mt-1">
            {{ externalReady ? t('address_complete') : t('pending_fill') }}
          </span>
        </div>
        <div class="p-6 grid gap-5">
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('external_host') }}</label>
            <Input v-model="form.external_host" :placeholder="t('external_host_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('external_port') }}</label>
            <NumberInput v-model="form.external_port" :min="1" :max="65535" />
            <p class="text-xs text-[var(--color-text-secondary)]">{{ t('terminal_addr_note') }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between gap-4">
      <p class="text-sm text-[var(--color-text-secondary)] max-w-[620px]">{{ t('udp_save_hint') }}</p>
      <div class="flex items-center gap-3">
        <Button variant="outline" :disabled="loading" @click="loadConfig">{{ t('reset_to_current') }}</Button>
        <Button :disabled="saving" @click="handleSave">{{ t('save_config') }}</Button>
      </div>
    </div>
  </div>
</template>
