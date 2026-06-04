<template>
  <el-form ref="formRef" :model="model" :rules="rules" label-width="120px">
    <el-form-item :label="t('provider')" prop="provider">
      <el-select v-model="model.provider" :placeholder="t('select_provider')" style="width: 100%">
        <el-option label="TEN VAD" value="ten_vad" />
        <el-option label="Silero VAD" value="silero_vad" />
      </el-select>
    </el-form-item>
    <el-form-item :label="t('config_name')" prop="name">
      <el-input v-model="model.name" :placeholder="t('enter_config_name')" />
    </el-form-item>
    <el-form-item :label="t('config_id')" prop="config_id">
      <el-input v-model="model.config_id" :placeholder="t('enter_unique_config_id')" />
    </el-form-item>
    <template v-if="model.provider === 'webrtc_vad'">
      <el-divider content-position="left">{{ t('webrtc_vad_config') }}</el-divider>
      <el-form-item :label="t('min_pool_size')" prop="webrtc_vad.pool_min_size">
        <el-input-number v-model="model.webrtc_vad.pool_min_size" :min="1" :max="1000" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('max_pool_size')" prop="webrtc_vad.pool_max_size">
        <el-input-number v-model="model.webrtc_vad.pool_max_size" :min="1" :max="10000" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('max_idle_connections')" prop="webrtc_vad.pool_max_idle">
        <el-input-number v-model="model.webrtc_vad.pool_max_idle" :min="1" :max="1000" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('vad_sample_rate')" prop="webrtc_vad.vad_sample_rate">
        <el-select v-model="model.webrtc_vad.vad_sample_rate" style="width: 100%">
          <el-option label="8000 Hz" :value="8000" />
          <el-option label="16000 Hz" :value="16000" />
          <el-option label="32000 Hz" :value="32000" />
          <el-option label="48000 Hz" :value="48000" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('vad_mode')" prop="webrtc_vad.vad_mode">
        <el-select v-model="model.webrtc_vad.vad_mode" style="width: 100%">
          <el-option :label="t('mode_quality_priority')" :value="0" />
          <el-option :label="t('mode_low_latency')" :value="1" />
          <el-option :label="t('mode_balanced')" :value="2" />
          <el-option :label="t('mode_high_accuracy')" :value="3" />
        </el-select>
      </el-form-item>
    </template>
    <template v-if="model.provider === 'silero_vad'">
      <el-divider content-position="left">{{ t('silero_vad_config') }}</el-divider>
      <el-form-item :label="t('model_path_label')" prop="silero_vad.model_path">
        <el-input v-model="model.silero_vad.model_path" :placeholder="t('model_path_input')" />
      </el-form-item>
      <el-form-item :label="t('threshold')" prop="silero_vad.threshold">
        <el-input-number v-model="model.silero_vad.threshold" :min="0" :max="1" :step="0.1" :precision="2" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('min_silence_duration_ms')" prop="silero_vad.min_silence_duration_ms">
        <el-input-number v-model="model.silero_vad.min_silence_duration_ms" :min="10" :max="5000" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('sample_rate')" prop="silero_vad.sample_rate">
        <el-select v-model="model.silero_vad.sample_rate" style="width: 100%">
          <el-option label="8000 Hz" :value="8000" />
          <el-option label="16000 Hz" :value="16000" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('channel_count_label')" prop="silero_vad.channels">
        <el-select v-model="model.silero_vad.channels" style="width: 100%">
          <el-option :label="t('mono_channel')" :value="1" />
          <el-option :label="t('stereo_channel')" :value="2" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('session_count')" prop="silero_vad.pool_size">
        <el-input-number v-model="model.silero_vad.pool_size" :min="1" :max="100" style="width: 100%" />
        <div style="font-size: 12px; color: #909399; margin-top: 4px;">{{ t('default_cpu_cores') }}</div>
      </el-form-item>
      <el-form-item :label="t('fetch_timeout_ms')" prop="silero_vad.acquire_timeout_ms">
        <el-input-number v-model="model.silero_vad.acquire_timeout_ms" :min="100" :max="30000" style="width: 100%" />
      </el-form-item>
    </template>
    <template v-if="model.provider === 'ten_vad'">
      <el-divider content-position="left">{{ t('ten_vad_config') }}</el-divider>
      <el-form-item :label="t('hop_size_label')" prop="ten_vad.hop_size">
        <el-input-number v-model="model.ten_vad.hop_size" :min="128" :max="1024" style="width: 100%" />
        <div style="font-size: 12px; color: #909399; margin-top: 4px;">{{ t('hop_size_default') }}</div>
      </el-form-item>
      <el-form-item :label="t('vad_threshold_label')" prop="ten_vad.threshold">
        <el-input-number v-model="model.ten_vad.threshold" :min="0" :max="1" :step="0.1" :precision="2" style="width: 100%" />
        <div style="font-size: 12px; color: #909399; margin-top: 4px;">{{ t('vad_threshold_recommended') }}</div>
      </el-form-item>
      <el-form-item :label="t('pool_size_label')" prop="ten_vad.pool_size">
        <el-input-number v-model="model.ten_vad.pool_size" :min="1" :max="100" style="width: 100%" />
        <div style="font-size: 12px; color: #909399; margin-top: 4px;">{{ t('pool_size_recommended') }}</div>
      </el-form-item>
      <el-form-item :label="t('fetch_timeout_ms')" prop="ten_vad.acquire_timeout_ms">
        <el-input-number v-model="model.ten_vad.acquire_timeout_ms" :min="100" :max="30000" style="width: 100%" />
        <div style="font-size: 12px; color: #909399; margin-top: 4px;">{{ t('timeout_recommended') }}</div>
      </el-form-item>
    </template>
  </el-form>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useLocale } from '../../../composables/useLocale'

const { t } = useLocale()

const props = defineProps({
  model: { type: Object, required: true },
  rules: { type: Object, default: () => ({}) }
})

const formRef = ref()

function getJsonData() {
  const m = props.model
  if (m.provider === 'webrtc_vad') return JSON.stringify(m.webrtc_vad || {})
  if (m.provider === 'silero_vad') return JSON.stringify(m.silero_vad || {})
  if (m.provider === 'ten_vad') return JSON.stringify(m.ten_vad || {})
  return '{}'
}

function validate(callback) {
  return formRef.value?.validate(callback)
}

function resetFields() {
  formRef.value?.resetFields()
}

defineExpose({ validate, getJsonData, resetFields })
</script>
