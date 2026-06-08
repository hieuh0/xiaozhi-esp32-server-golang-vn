<script setup>
import { useLocale } from '../../../composables/useLocale'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import NumberInput from '@/components/ui/number-input.vue'

const { t } = useLocale()

const props = defineProps({
  model: { type: Object, required: true }
})

function getJsonData() {
  const m = props.model
  if (m.provider === 'webrtc_vad') return JSON.stringify(m.webrtc_vad || {})
  if (m.provider === 'silero_vad') return JSON.stringify(m.silero_vad || {})
  if (m.provider === 'ten_vad') return JSON.stringify(m.ten_vad || {})
  return '{}'
}

function validate(callback) {
  const m = props.model
  let error = null
  if (!m.name?.trim()) error = t('enter_config_name')
  else if (!m.config_id?.trim()) error = t('enter_config_id')
  else if (!m.provider) error = t('select_provider')

  if (callback) {
    callback(!error)
    return Promise.resolve(!error)
  }
  return error ? Promise.reject(new Error(error)) : Promise.resolve()
}

function resetFields() {}

defineExpose({ validate, getJsonData, resetFields })
</script>

<template>
  <div class="space-y-1 py-1">
    <!-- Provider -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('provider') }}</label>
      <Select v-model="model.provider">
        <SelectTrigger><SelectValue :placeholder="t('select_provider')" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="ten_vad">TEN VAD</SelectItem>
          <SelectItem value="silero_vad">Silero VAD</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Config name -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_name') }}</label>
      <Input v-model="model.name" :placeholder="t('enter_config_name')" />
    </div>

    <!-- Config ID -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_id') }}</label>
      <Input v-model="model.config_id" :placeholder="t('enter_unique_config_id')" />
    </div>

    <!-- WebRTC VAD (legacy) -->
    <template v-if="model.provider === 'webrtc_vad'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('webrtc_vad_config') }}</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('min_pool_size') }}</label>
        <NumberInput v-model="model.webrtc_vad.pool_min_size" :min="1" :max="1000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('max_pool_size') }}</label>
        <NumberInput v-model="model.webrtc_vad.pool_max_size" :min="1" :max="10000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('max_idle_connections') }}</label>
        <NumberInput v-model="model.webrtc_vad.pool_max_idle" :min="1" :max="1000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('vad_sample_rate') }}</label>
        <Select
          :model-value="String(model.webrtc_vad.vad_sample_rate)"
          @update:model-value="v => model.webrtc_vad.vad_sample_rate = Number(v)"
        >
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="8000">8000 Hz</SelectItem>
            <SelectItem value="16000">16000 Hz</SelectItem>
            <SelectItem value="32000">32000 Hz</SelectItem>
            <SelectItem value="48000">48000 Hz</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('vad_mode') }}</label>
        <Select
          :model-value="String(model.webrtc_vad.vad_mode)"
          @update:model-value="v => model.webrtc_vad.vad_mode = Number(v)"
        >
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="0">{{ t('mode_quality_priority') }}</SelectItem>
            <SelectItem value="1">{{ t('mode_low_latency') }}</SelectItem>
            <SelectItem value="2">{{ t('mode_balanced') }}</SelectItem>
            <SelectItem value="3">{{ t('mode_high_accuracy') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </template>

    <!-- Silero VAD -->
    <template v-if="model.provider === 'silero_vad'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('silero_vad_config') }}</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model_path_label') }}</label>
        <Input v-model="model.silero_vad.model_path" :placeholder="t('model_path_input')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('threshold') }}</label>
        <NumberInput v-model="model.silero_vad.threshold" :min="0" :max="1" :step="0.1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('min_silence_duration_ms') }}</label>
        <NumberInput v-model="model.silero_vad.min_silence_duration_ms" :min="10" :max="5000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <Select
          :model-value="String(model.silero_vad.sample_rate)"
          @update:model-value="v => model.silero_vad.sample_rate = Number(v)"
        >
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="8000">8000 Hz</SelectItem>
            <SelectItem value="16000">16000 Hz</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('channel_count_label') }}</label>
        <Select
          :model-value="String(model.silero_vad.channels)"
          @update:model-value="v => model.silero_vad.channels = Number(v)"
        >
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="1">{{ t('mono_channel') }}</SelectItem>
            <SelectItem value="2">{{ t('stereo_channel') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('session_count') }}</label>
        <div class="space-y-1">
          <NumberInput v-model="model.silero_vad.pool_size" :min="1" :max="100" />
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('default_cpu_cores') }}</p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('fetch_timeout_ms') }}</label>
        <NumberInput v-model="model.silero_vad.acquire_timeout_ms" :min="100" :max="30000" />
      </div>
    </template>

    <!-- TEN VAD -->
    <template v-if="model.provider === 'ten_vad'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('ten_vad_config') }}</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('hop_size_label') }}</label>
        <div class="space-y-1">
          <NumberInput v-model="model.ten_vad.hop_size" :min="128" :max="1024" />
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('hop_size_default') }}</p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('vad_threshold_label') }}</label>
        <div class="space-y-1">
          <NumberInput v-model="model.ten_vad.threshold" :min="0" :max="1" :step="0.1" />
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('vad_threshold_recommended') }}</p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('pool_size_label') }}</label>
        <div class="space-y-1">
          <NumberInput v-model="model.ten_vad.pool_size" :min="1" :max="100" />
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('pool_size_recommended') }}</p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('fetch_timeout_ms') }}</label>
        <div class="space-y-1">
          <NumberInput v-model="model.ten_vad.acquire_timeout_ms" :min="100" :max="30000" />
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('timeout_recommended') }}</p>
        </div>
      </div>
    </template>
  </div>
</template>
