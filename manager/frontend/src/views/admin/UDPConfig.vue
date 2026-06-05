<template>
  <div class="udp-config">
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
                <p class="card-kicker">UDP Server</p>
                <h3>{{ t('service_listen') }}</h3>
                <p class="card-description">{{ t('udp_server_config_desc') }}</p>
              </div>
              <el-tag :type="listenReady ? 'success' : 'warning'" effect="plain" round>
                {{ listenReady ? t('listen_params_complete') : t('pending_fill') }}
              </el-tag>
            </div>
          </template>

          <div class="field-grid">
            <el-form-item :label="t('config_name')" prop="name">
              <el-input v-model="form.name" :placeholder="t('udp_name_ph')" />
            </el-form-item>

            <el-form-item :label="t('listen_host')" prop="listen_host">
              <el-input v-model="form.listen_host" :placeholder="t('enter_listen_host_eg')" />
            </el-form-item>

            <el-form-item :label="t('listen_port')" prop="listen_port">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </div>
        </el-card>

        <el-card class="config-card config-card-side" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Announce Address</p>
                <h3>{{ t('terminal_publish_addr') }}</h3>
                <p class="card-description">{{ t('terminal_publish_desc') }}</p>
              </div>
              <el-tag :type="externalReady ? 'success' : 'warning'" effect="plain" round>
                {{ externalReady ? t('address_complete') : t('pending_fill') }}
              </el-tag>
            </div>
          </template>

          <div class="field-stack">
            <el-form-item :label="t('external_host')" prop="external_host">
              <el-input v-model="form.external_host" :placeholder="t('external_host_ph')" />
            </el-form-item>

            <el-form-item :label="t('external_port')" prop="external_port">
              <el-input-number v-model="form.external_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
              <div class="field-help">
                {{ t('terminal_addr_note') }}
              </div>
            </el-form-item>
          </div>
        </el-card>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          {{ t('udp_save_hint') }}
        </p>
        <div class="footer-actions">
          <el-button plain :loading="loading" @click="loadConfig">{{ t('reset_to_current') }}</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">{{ t('save_config') }}</el-button>
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
  name: t('udp_config_label'),
  is_default: true,
  external_host: '192.168.0.208',
  external_port: 8990,
  listen_host: '0.0.0.0',
  listen_port: 8990
})

const form = reactive(createDefaultFormState())

const rules = {
  name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
  external_host: [{ required: true, message: t('enter_external_host'), trigger: 'blur' }],
  external_port: [
    { required: true, message: t('enter_external_port'), trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: t('port_range_error'), trigger: 'blur' }
  ],
  listen_host: [{ required: true, message: t('enter_listen_host'), trigger: 'blur' }],
  listen_port: [
    { required: true, message: t('enter_listen_port'), trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: t('port_range_error'), trigger: 'blur' }
  ]
}

const listenReady = computed(() => {
  return Boolean(String(form.listen_host || '').trim() && Number(form.listen_port))
})

const externalReady = computed(() => {
  return Boolean(String(form.external_host || '').trim() && Number(form.external_port))
})

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
      } catch (error) {
        ElMessage.warning(t('udp_config_format_error'))
        configData = {}
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
  } catch (error) {
    ElMessage.error(t('load_udp_config_failed'))
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

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.udp-config {
  padding: 0 24px 32px;
}

.config-form {
  display: grid;
  gap: 24px;
}

.config-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(320px, 0.9fr);
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
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px 18px;
}

.field-stack {
  display: grid;
  gap: 20px;
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
  .udp-config {
    padding: 0 16px 24px;
  }

  :deep(.el-card__body) {
    padding: 20px;
  }

  :deep(.el-card__header) {
    padding: 20px 20px 0;
  }

  .field-grid {
    grid-template-columns: 1fr;
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
