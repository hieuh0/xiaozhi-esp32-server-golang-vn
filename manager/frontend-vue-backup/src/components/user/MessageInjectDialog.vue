<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  devices: { type: Array, default: () => [] },
  defaultDeviceId: { type: String, default: '' },
  lockDevice: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'success'])

const submitting = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive({ device_id: '', message: '', skip_llm: false, auto_listen: true })

const deviceSelectDisabled = computed(() => props.lockDevice && !!props.defaultDeviceId)

const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  return (Date.now() - new Date(lastActiveAt).getTime()) < 5 * 60 * 1000
}

const getDeviceNickName = (device) => {
  const n = String(device?.nick_name || '').trim()
  return n || String(device?.device_name || '').trim() || t('unnamed_device')
}

const getDeviceIdText = (device) => String(device?.device_name || '').trim() || '-'

const resetForm = () => {
  form.device_id = props.defaultDeviceId || ''
  form.message = ''
  form.skip_llm = false
  form.auto_listen = true
}

watch(() => [props.modelValue, props.defaultDeviceId], ([v]) => { if (v) resetForm() })

const handleSubmit = async () => {
  if (!form.device_id) { ElMessage.error(t('select_device')); return }
  if (!form.message.trim()) { ElMessage.error(t('enter_push_content')); return }
  submitting.value = true
  try {
    const res = await api.post('/user/devices/inject-message', {
      device_id: form.device_id,
      message: form.message,
      skip_llm: form.skip_llm,
      auto_listen: form.auto_listen
    })
    if (res.data?.success) {
      ElMessage.success(t('voice_push_success'))
      emit('success', res.data?.data || null)
      visible.value = false
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('voice_push_failed'))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="visible">
    <DialogContent class="max-w-[560px]">
      <DialogHeader>
        <DialogTitle>{{ t('voice_push') }}</DialogTitle>
      </DialogHeader>

      <div class="grid gap-4 py-2">
        <!-- Device select -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('select_device_prompt') }}</label>
          <Select v-model="form.device_id" :disabled="deviceSelectDisabled">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="t('select_push_voice_device')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="device in devices"
                :key="device.id || device.device_code"
                :value="device.device_name || ''"
              >
                <div class="flex items-center justify-between gap-3 w-full">
                  <span class="font-medium truncate">{{ getDeviceNickName(device) }}</span>
                  <span :class="['text-[10px] px-1.5 py-0.5 rounded-full font-medium', isDeviceOnline(device.last_active_at) ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-600']">
                    {{ isDeviceOnline(device.last_active_at) ? t('online') : t('offline') }}
                  </span>
                </div>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Message textarea -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('push_content') }}</label>
          <Textarea v-model="form.message" :placeholder="t('enter_broadcast_content')" maxlength="500" rows="4" />
          <p class="text-xs text-[var(--color-text-tertiary)] text-right">{{ form.message.length }}/500</p>
        </div>

        <!-- Direct broadcast switch -->
        <div class="flex items-center justify-between gap-4 px-4 py-3 rounded-xl bg-[var(--color-surface-muted)] border border-[var(--color-line)]">
          <div>
            <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('direct_broadcast') }} — {{ form.skip_llm ? t('enable') : t('close') }}</p>
            <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ form.skip_llm ? t('msg_direct_tts') : t('msg_via_llm') }}</p>
          </div>
          <Switch :model-value="form.skip_llm" @update:model-value="(v) => form.skip_llm = v" />
        </div>

        <!-- Return to idle switch -->
        <div class="flex items-center justify-between gap-4 px-4 py-3 rounded-xl bg-[var(--color-surface-muted)] border border-[var(--color-line)]">
          <div>
            <p class="text-sm font-semibold text-[var(--color-text)]">{{ t('switch_to_idle') }} — {{ !form.auto_listen ? t('enable') : t('close') }}</p>
            <p class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ !form.auto_listen ? t('broadcast_return_idle') : t('broadcast_continue_listen') }}</p>
          </div>
          <Switch :model-value="!form.auto_listen" @update:model-value="(v) => form.auto_listen = !v" />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="visible = false">{{ t('cancel') }}</Button>
        <Button :loading="submitting" @click="handleSubmit">
          {{ submitting ? t('pushing') : t('voice_push') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
