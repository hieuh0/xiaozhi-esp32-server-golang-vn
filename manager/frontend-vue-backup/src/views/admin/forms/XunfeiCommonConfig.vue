<script setup>
import { useLocale } from '../../../composables/useLocale'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'

const { t } = useLocale()
defineProps({
  modelValue: { type: Object, required: true },
  prefix: { type: String, required: true },
  defaultWsUrl: { type: String, default: '' }
})
</script>

<template>
  <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
    <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('app_id') }}</label>
    <Input v-model="modelValue[prefix].app_id" :placeholder="t('enter_xunfei_app_id')" />
  </div>
  <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
    <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
    <Input v-model="modelValue[prefix].api_key" type="password" :placeholder="t('enter_api_key')" />
  </div>
  <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
    <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Secret</label>
    <Input v-model="modelValue[prefix].api_secret" type="password" :placeholder="t('enter_api_secret')" />
  </div>
  <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
    <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">WebSocket URL</label>
    <Input v-model="modelValue[prefix].ws_url" :placeholder="defaultWsUrl" />
  </div>
  <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
    <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('frame_duration') }}</label>
    <NumberInput v-model="modelValue[prefix].frame_duration" :min="1" :max="1000" />
  </div>
  <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
    <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('connection_timeout') }}</label>
    <NumberInput v-model="modelValue[prefix].connect_timeout" :min="1" :max="60" />
  </div>
  <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
    <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('read_timeout') }}</label>
    <NumberInput v-model="modelValue[prefix].read_timeout" :min="1" :max="300" />
  </div>
</template>
