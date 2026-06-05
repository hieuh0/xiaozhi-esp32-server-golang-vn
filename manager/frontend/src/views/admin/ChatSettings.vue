<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button @click="loadSettings" :loading="loading">{{ t('refresh') }}</el-button>
      <el-button type="primary" @click="saveSettings" :loading="saving">{{ t('save_settings') }}</el-button>
    </div>

    <el-card v-loading="loading">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="180px" style="max-width: 720px;">
        <el-divider content-position="left">{{ t('identity_verification') }}</el-divider>
        <el-form-item :label="t('enable_device_activation')" prop="auth.enable">
          <el-switch v-model="form.auth.enable" />
        </el-form-item>
        <el-form-item :label="t('login_digit_verify')" prop="auth.login_captcha_enabled">
          <el-switch
            v-model="form.auth.login_captcha_enabled"
            active-:text="t('enable')"
            :inactive-text="t('close')"
          />
          <div class="form-help">
            {{ t('captcha_enabled_hint') }}</div>
        </el-form-item>

        <el-divider content-position="left">{{ t('chat_params') }}</el-divider>
        <el-form-item :label="t('session_max_idle_time')" prop="chat.max_idle_duration">
          <el-input-number v-model="form.chat.max_idle_duration" :min="0" :step="1000" style="width: 100%;" />
          <div class="form-help">
            {{ t('session_idle_hint') }}</div>
        </el-form-item>
        <el-form-item :label="t('sentence_end_silence_threshold')" prop="chat.chat_max_silence_duration">
          <el-input-number v-model="form.chat.chat_max_silence_duration" :min="0" :step="10" style="width: 100%;" />
          <div class="form-help">
            {{ t('chat_silence_hint') }}
          </div>
        </el-form-item>
        <el-form-item :label="t('realtime_interrupt_mode')" prop="chat.realtime_mode">
          <el-select v-model="form.chat.realtime_mode" style="width: 100%;">
            <el-option :value="1" :label="t('vad_interrupt_mode_1')" />
            <el-option :value="2" :label="t('asr_interrupt_mode')" />
            <el-option :value="3" :label="t('asr_voiceprint_interrupt')" />
            <el-option :value="4" :label="t('asr_result_interrupt')" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('global_system_prompt_desc')" prop="chat.global_system_prompt">
          <el-input
            v-model="form.chat.global_system_prompt"
            type="textarea"
            :rows="6"
            maxlength="8000"
            show-word-limit
            :placeholder="t('system_prompt_prefix_hint')"
          />
          <div class="form-help">
            {{ t('system_prompt_order_hint') }}</div>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)
const formRef = ref()

const form = reactive({
  auth: {
    enable: false,
    login_captcha_enabled: true
  },
  chat: {
    max_idle_duration: 30000,
    chat_max_silence_duration: 400,
    realtime_mode: 4,
    global_system_prompt: ''
  }
})

const rules = {
  'chat.max_idle_duration': [
    { required: true, message: t('enter_max_idle_time'), trigger: 'blur' }
  ],
  'chat.chat_max_silence_duration': [
    { required: true, message: t('enter_sentence_end_silence'), trigger: 'blur' }
  ],
  'chat.realtime_mode': [
    { required: true, message: t('select_interrupt_mode'), trigger: 'change' }
  ],
  'chat.global_system_prompt': [
    { max: 8000, message: t('global_system_prompt_max'), trigger: 'blur' }
  ]
}

const loadSettings = async () => {
  loading.value = true
  try {
    const res = await api.get('/admin/chat-settings')
    const data = res.data?.data || {}
    form.auth.enable = !!data.auth?.enable
    form.auth.login_captcha_enabled = data.auth?.login_captcha_enabled !== false
    form.chat.max_idle_duration = Number(data.chat?.max_idle_duration ?? 30000)
    form.chat.chat_max_silence_duration = Number(data.chat?.chat_max_silence_duration ?? 400)
    form.chat.realtime_mode = Number(data.chat?.realtime_mode ?? 4)
    form.chat.global_system_prompt = String(data.chat?.global_system_prompt ?? '')
  } catch (error) {
    ElMessage.error(t('load_chat_settings_failed'))
    console.error(error)
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    await api.put('/admin/chat-settings', {
      auth: {
        enable: !!form.auth.enable,
        login_captcha_enabled: form.auth.login_captcha_enabled !== false
      },
      chat: {
        max_idle_duration: Number(form.chat.max_idle_duration),
        chat_max_silence_duration: Number(form.chat.chat_max_silence_duration),
        realtime_mode: Number(form.chat.realtime_mode),
        global_system_prompt: String(form.chat.global_system_prompt || '')
      }
    })
    ElMessage.success(t('chat_settings_save_success'))
  } catch (error) {
    ElMessage.error(t('chat_settings_save_failed'))
    console.error(error)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.form-help {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}
</style>
