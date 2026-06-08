<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import NumberInput from '@/components/ui/number-input.vue'
import { Switch } from '@/components/ui/switch'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const loading = ref(false)
const saving = ref(false)

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

const validate = () => {
  if (form.chat.max_idle_duration === undefined || form.chat.max_idle_duration === null) {
    ElMessage.error(t('enter_max_idle_time')); return false
  }
  if (form.chat.chat_max_silence_duration === undefined || form.chat.chat_max_silence_duration === null) {
    ElMessage.error(t('enter_sentence_end_silence')); return false
  }
  if (!form.chat.realtime_mode) {
    ElMessage.error(t('select_interrupt_mode')); return false
  }
  if (String(form.chat.global_system_prompt || '').length > 8000) {
    ElMessage.error(t('global_system_prompt_max')); return false
  }
  return true
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
  } catch {
    ElMessage.error(t('load_chat_settings_failed'))
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  if (!validate()) return

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
  } catch {
    ElMessage.error(t('chat_settings_save_failed'))
  } finally {
    saving.value = false
  }
}

onMounted(() => { loadSettings() })
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex justify-end items-center gap-2">
      <Button variant="outline" :disabled="loading" @click="loadSettings">{{ t('refresh') }}</Button>
      <Button :disabled="saving" @click="saveSettings">{{ t('save_settings') }}</Button>
    </div>

    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <div v-else class="max-w-[720px] rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)]">
      <div class="p-6 grid gap-6">

        <!-- Identity verification section -->
        <div>
          <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-4">{{ t('identity_verification') }}</p>
          <div class="grid gap-5">
            <div class="flex items-center justify-between gap-4">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('enable_device_activation') }}</label>
              <Switch :checked="form.auth.enable" @update:checked="v => form.auth.enable = v" />
            </div>
            <div class="grid gap-1.5">
              <div class="flex items-center justify-between gap-4">
                <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('login_digit_verify') }}</label>
                <Switch :checked="form.auth.login_captcha_enabled" @update:checked="v => form.auth.login_captcha_enabled = v" />
              </div>
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('captcha_enabled_hint') }}</p>
            </div>
          </div>
        </div>

        <!-- Chat params section -->
        <div class="pt-6 border-t border-[var(--color-line)]">
          <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-4">{{ t('chat_params') }}</p>
          <div class="grid gap-5">
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('session_max_idle_time') }}</label>
              <NumberInput v-model="form.chat.max_idle_duration" :min="0" :step="1000" />
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('session_idle_hint') }}</p>
            </div>

            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('sentence_end_silence_threshold') }}</label>
              <NumberInput v-model="form.chat.chat_max_silence_duration" :min="0" :step="10" />
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('chat_silence_hint') }}</p>
            </div>

            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('realtime_interrupt_mode') }}</label>
              <Select :model-value="String(form.chat.realtime_mode)" @update:model-value="v => form.chat.realtime_mode = Number(v)">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">{{ t('vad_interrupt_mode_1') }}</SelectItem>
                  <SelectItem value="2">{{ t('asr_interrupt_mode') }}</SelectItem>
                  <SelectItem value="3">{{ t('asr_voiceprint_interrupt') }}</SelectItem>
                  <SelectItem value="4">{{ t('asr_result_interrupt') }}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('global_system_prompt_desc') }}</label>
              <textarea
                v-model="form.chat.global_system_prompt"
                rows="6"
                maxlength="8000"
                :placeholder="t('system_prompt_prefix_hint')"
                class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 w-full resize-y min-h-[120px]"
              />
              <p class="text-xs text-[var(--color-text-secondary)] text-right">
                {{ String(form.chat.global_system_prompt || '').length }} / 8000
              </p>
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('system_prompt_order_hint') }}</p>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
