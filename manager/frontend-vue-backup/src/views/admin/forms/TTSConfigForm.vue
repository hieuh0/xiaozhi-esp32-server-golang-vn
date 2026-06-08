<script setup>
import { computed } from 'vue'
import { getTTSProviderOptions } from './ttsProviderOptions'
import XunfeiCommonConfig from './XunfeiCommonConfig.vue'
import { useLocale } from '../../../composables/useLocale'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import NumberInput from '@/components/ui/number-input.vue'

const { t } = useLocale()

const TTS_PROVIDER_OPTIONS = computed(() => getTTSProviderOptions(t))

const DOUBAO_MODEL_OPTIONS = [
  { label: t('doubao_tts_v11'), value: 'seed-tts-1.1' },
  { label: t('doubao_tts_v2_standard'), value: 'seed-tts-2.0-standard' },
  { label: t('doubao_tts_v2_expressive'), value: 'seed-tts-2.0-expressive' },
  { label: t('doubao_clone_v1'), value: 'seed-icl-1.0' },
  { label: t('doubao_clone_v2_standard'), value: 'seed-icl-2.0-standard' },
  { label: t('doubao_clone_v2_expressive'), value: 'seed-icl-2.0-expressive' }
]

const props = defineProps({
  model: { type: Object, required: true },
  voiceOptions: { type: Array, default: () => [] },
  voiceLoading: { type: Boolean, default: false }
})
const emit = defineEmits(['request-voice-options'])

const voiceOptionsList = computed(() => Array.isArray(props.voiceOptions) ? props.voiceOptions : [])

const indexTTSDocURL = 'https://github.com/hackers365/xiaozhi-esp32-server-golang/blob/main/doc/indextts_vllm_api.md'
const indexTTSReferenceURL = 'https://github.com/hackers365/index-tts-vllm/blob/master/api_server.py'

function handleIndexTTSVoiceFocus() {
  emit('request-voice-options', 'indextts_vllm')
}

function getJsonData() {
  const form = props.model
  const config = {}
  switch (form.provider) {
    case 'cosyvoice':
      config.api_url = form.cosyvoice?.api_url
      config.spk_id = form.cosyvoice?.spk_id
      config.frame_duration = form.cosyvoice?.frame_duration
      config.target_sr = form.cosyvoice?.target_sr
      config.audio_format = form.cosyvoice?.audio_format
      config.instruct_text = form.cosyvoice?.instruct_text
      break
    case 'doubao_ws':
      config.appid = form.doubao_ws?.appid
      config.access_token = form.doubao_ws?.access_token
      config.model = form.doubao_ws?.model || 'seed-tts-2.0-standard'
      config.resource_id = form.doubao_ws?.resource_id
      config.voice = form.doubao_ws?.voice
      config.ws_url = form.doubao_ws?.ws_url || 'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream'
      break
    case 'edge':
      config.voice = form.edge?.voice
      config.rate = form.edge?.rate
      config.volume = form.edge?.volume
      config.pitch = form.edge?.pitch
      config.connect_timeout = form.edge?.connect_timeout
      config.receive_timeout = form.edge?.receive_timeout
      break
    case 'edge_offline':
      config.server_url = form.edge_offline?.server_url
      config.timeout = form.edge_offline?.timeout
      config.sample_rate = form.edge_offline?.sample_rate
      config.channels = form.edge_offline?.channels
      config.frame_duration = form.edge_offline?.frame_duration
      break
    case 'aliyun_qwen':
      config.provider = 'aliyun_qwen'
      config.api_key = form.qwen_tts?.api_key
      config.api_url = form.qwen_tts?.api_url
      config.region = form.qwen_tts?.region
      config.model = form.qwen_tts?.model || 'qwen3-tts-flash'
      config.voice = form.qwen_tts?.voice || 'Cherry'
      config.language_type = form.qwen_tts?.language_type || 'Chinese'
      config.stream = form.qwen_tts?.stream
      config.frame_duration = form.qwen_tts?.frame_duration || 60
      break
    case 'openai':
      config.api_key = form.openai?.api_key
      config.api_url = form.openai?.api_url
      config.model = form.openai?.model
      config.voice = form.openai?.voice
      config.response_format = form.openai?.response_format
      config.speed = form.openai?.speed
      config.stream = form.openai?.stream
      config.frame_duration = form.openai?.frame_duration
      break
    case 'xunfei':
      config.provider = 'xunfei'
      config.app_id = form.xunfei?.app_id
      config.api_key = form.xunfei?.api_key
      config.api_secret = form.xunfei?.api_secret
      config.ws_url = form.xunfei?.ws_url
      config.voice = form.xunfei?.voice
      config.audio_encoding = form.xunfei?.audio_encoding || 'raw'
      config.sample_rate = form.xunfei?.sample_rate || 16000
      config.speed = form.xunfei?.speed ?? 50
      config.volume = form.xunfei?.volume ?? 50
      config.pitch = form.xunfei?.pitch ?? 50
      config.tte = form.xunfei?.tte || 'UTF8'
      config.reg = form.xunfei?.reg ?? 0
      config.rdn = form.xunfei?.rdn ?? 0
      config.frame_duration = form.xunfei?.frame_duration || 60
      config.connect_timeout = form.xunfei?.connect_timeout || 10
      config.read_timeout = form.xunfei?.read_timeout || 30
      break
    case 'xunfei_super_tts':
      config.provider = 'xunfei_super_tts'
      config.double_stream = true
      config.app_id = form.xunfei_super_tts?.app_id
      config.api_key = form.xunfei_super_tts?.api_key
      config.api_secret = form.xunfei_super_tts?.api_secret
      config.ws_url = form.xunfei_super_tts?.ws_url
      config.voice = form.xunfei_super_tts?.voice
      config.audio_encoding = form.xunfei_super_tts?.audio_encoding || 'raw'
      config.sample_rate = form.xunfei_super_tts?.sample_rate || 24000
      config.speed = form.xunfei_super_tts?.speed ?? 50
      config.volume = form.xunfei_super_tts?.volume ?? 50
      config.pitch = form.xunfei_super_tts?.pitch ?? 50
      config.bgs = form.xunfei_super_tts?.bgs ?? 0
      config.reg = form.xunfei_super_tts?.reg ?? 0
      config.rdn = form.xunfei_super_tts?.rdn ?? 0
      config.rhy = form.xunfei_super_tts?.rhy ?? 0
      config.oral_level = form.xunfei_super_tts?.oral_level || 'mid'
      config.spark_assist = form.xunfei_super_tts?.spark_assist ?? 1
      config.stop_split = form.xunfei_super_tts?.stop_split ?? 0
      config.remain = form.xunfei_super_tts?.remain ?? 0
      config.frame_duration = form.xunfei_super_tts?.frame_duration || 60
      config.connect_timeout = form.xunfei_super_tts?.connect_timeout || 10
      config.read_timeout = form.xunfei_super_tts?.read_timeout || 30
      break
    case 'indextts_vllm':
      config.provider = 'indextts_vllm'
      config.api_url = form.indextts_vllm?.api_url
      config.api_key = form.indextts_vllm?.api_key
      config.model = form.indextts_vllm?.model || 'indextts-vllm'
      config.voice = form.indextts_vllm?.voice
      config.response_format = 'wav'
      config.stream = false
      config.frame_duration = form.indextts_vllm?.frame_duration || 60
      break
    case 'zhipu':
      config.provider = 'zhipu'
      config.api_key = form.zhipu?.api_key
      config.api_url = form.zhipu?.api_url || 'https://open.bigmodel.cn/api/paas/v4/audio/speech'
      config.model = form.zhipu?.model || 'glm-tts'
      config.voice = form.zhipu?.voice
      config.response_format = form.zhipu?.response_format
      config.speed = form.zhipu?.speed
      config.volume = form.zhipu?.volume || 1.0
      config.stream = form.zhipu?.stream
      config.encode_format = form.zhipu?.encode_format || 'base64'
      config.frame_duration = form.zhipu?.frame_duration
      break
    case 'minimax':
      config.provider = 'minimax'
      config.api_key = form.minimax?.api_key
      config.model = form.minimax?.model || 'speech-2.8-hd'
      config.voice = form.minimax?.voice || 'male-qn-qingse'
      config.speed = form.minimax?.speed || 1.0
      config.vol = form.minimax?.vol || 1.0
      config.pitch = form.minimax?.pitch || 0
      config.sample_rate = form.minimax?.sample_rate || 32000
      config.bitrate = form.minimax?.bitrate || 128000
      config.format = form.minimax?.format || 'mp3'
      config.channel = form.minimax?.channel || 1
      break
  }
  return JSON.stringify(config)
}

function validate(callback) {
  const m = props.model
  let error = null
  if (!m.name?.trim()) error = t('enter_config_name')
  else if (!m.config_id?.trim()) error = t('enter_config_id')
  else if (!m.provider) error = t('select_provider')
  else if (m.provider === 'cosyvoice') {
    if (!m.cosyvoice?.api_url?.trim()) error = t('enter_api_url')
    else if (!m.cosyvoice?.spk_id?.trim()) error = t('enter_speaker_id')
  } else if (m.provider === 'doubao_ws') {
    if (!m.doubao_ws?.appid?.trim()) error = t('enter_app_id')
    else if (!m.doubao_ws?.access_token?.trim()) error = t('enter_access_token')
    else if (!m.doubao_ws?.model?.trim()) error = t('select_model')
    else if (!m.doubao_ws?.voice?.trim()) error = t('enter_voice_timbre')
    else if (!m.doubao_ws?.ws_url?.trim()) error = t('enter_websocket_url')
  } else if (m.provider === 'edge') {
    if (!m.edge?.voice?.trim()) error = t('enter_voice_timbre')
    else if (!m.edge?.rate?.trim()) error = t('enter_speech_rate')
    else if (!m.edge?.volume?.trim()) error = t('enter_volume')
  } else if (m.provider === 'edge_offline') {
    if (!m.edge_offline?.server_url?.trim()) error = t('enter_server_url')
  } else if (m.provider === 'openai') {
    if (!m.openai?.api_key?.trim()) error = t('enter_api_key')
  } else if (m.provider === 'xunfei') {
    if (!m.xunfei?.app_id?.trim()) error = t('enter_app_id')
    else if (!m.xunfei?.api_key?.trim()) error = t('enter_api_key')
    else if (!m.xunfei?.api_secret?.trim()) error = t('enter_api_secret')
    else if (!m.xunfei?.ws_url?.trim()) error = t('enter_websocket_url')
    else if (!m.xunfei?.voice?.trim()) error = t('enter_voice_timbre')
  } else if (m.provider === 'xunfei_super_tts') {
    if (!m.xunfei_super_tts?.app_id?.trim()) error = t('enter_app_id')
    else if (!m.xunfei_super_tts?.api_key?.trim()) error = t('enter_api_key')
    else if (!m.xunfei_super_tts?.api_secret?.trim()) error = t('enter_api_secret')
    else if (!m.xunfei_super_tts?.ws_url?.trim()) error = t('enter_websocket_url')
    else if (!m.xunfei_super_tts?.voice?.trim()) error = t('enter_voice_timbre')
  } else if (m.provider === 'zhipu') {
    if (!m.zhipu?.api_key?.trim()) error = t('enter_api_key')
  } else if (m.provider === 'minimax') {
    if (!m.minimax?.api_key?.trim()) error = t('enter_api_key')
  } else if (m.provider === 'aliyun_qwen') {
    if (!m.qwen_tts?.api_key?.trim()) error = t('enter_api_key')
  } else if (m.provider === 'indextts_vllm') {
    if (!m.indextts_vllm?.api_url?.trim()) error = t('enter_api_url')
  }
  const valid = !error
  if (callback) { callback(valid); return Promise.resolve(valid) }
  return valid ? Promise.resolve() : Promise.reject(new Error(error || 'Validation failed'))
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
          <SelectItem v-for="p in TTS_PROVIDER_OPTIONS" :key="p.value" :value="p.value">
            <span class="flex items-center gap-2">
              {{ p.label }}
              <span v-if="p.supports_voice_clone" class="text-xs font-medium text-green-600 dark:text-green-400">{{ t('support_clone') }}</span>
            </span>
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Config Name -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_name') }}</label>
      <Input v-model="model.name" :placeholder="t('enter_config_name')" />
    </div>

    <!-- Config ID -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_id') }}</label>
      <Input v-model="model.config_id" :placeholder="t('enter_unique_config_id')" />
    </div>

    <!-- Doubao WebSocket -->
    <template v-if="model.provider === 'doubao_ws'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Doubao WebSocket TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('app_id') }}</label>
        <Input v-model="model.doubao_ws.appid" :placeholder="t('enter_app_id')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('access_token') }}</label>
        <Input v-model="model.doubao_ws.access_token" type="password" :placeholder="t('enter_access_token')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Select v-model="model.doubao_ws.model">
          <SelectTrigger><SelectValue :placeholder="t('select_model')" /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="opt in DOUBAO_MODEL_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('resource_id') }}</label>
        <Input v-model="model.doubao_ws.resource_id" :placeholder="t('optional_instance_id_hint')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.doubao_ws.voice" list="doubao-ws-voice-list" :placeholder="voiceLoading ? 'Loading...' : t('select_timbre')" :disabled="voiceLoading" />
        <datalist id="doubao-ws-voice-list">
          <option v-for="opt in voiceOptionsList" :key="opt.value" :value="opt.value" />
        </datalist>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">WebSocket URL</label>
        <Input v-model="model.doubao_ws.ws_url" placeholder="wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream" />
      </div>
    </template>

    <!-- Edge TTS -->
    <template v-if="model.provider === 'edge'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Edge TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.edge.voice" list="edge-voice-list" :placeholder="voiceLoading ? 'Loading...' : t('select_timbre')" :disabled="voiceLoading" />
        <datalist id="edge-voice-list">
          <option v-for="opt in voiceOptionsList" :key="opt.value" :value="opt.value" />
        </datalist>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('speech_rate') }}</label>
        <Input v-model="model.edge.rate" :placeholder="t('enter_speech_rate_placeholder')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('volume') }}</label>
        <Input v-model="model.edge.volume" :placeholder="t('enter_volume_placeholder')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('pitch') }}</label>
        <Input v-model="model.edge.pitch" :placeholder="t('enter_pitch_placeholder')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('connection_timeout') }}</label>
        <NumberInput v-model="model.edge.connect_timeout" :min="1" :max="60" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('receive_timeout') }}</label>
        <NumberInput v-model="model.edge.receive_timeout" :min="1" :max="300" />
      </div>
    </template>

    <!-- Edge Offline -->
    <template v-if="model.provider === 'edge_offline'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Edge Offline TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('server_url_label') }}</label>
        <Input v-model="model.edge_offline.server_url" :placeholder="t('enter_server_url')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('timeout_label') }}</label>
        <NumberInput v-model="model.edge_offline.timeout" :min="1" :max="300" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <NumberInput v-model="model.edge_offline.sample_rate" :min="8000" :max="48000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('channel_count_label') }}</label>
        <NumberInput v-model="model.edge_offline.channels" :min="1" :max="8" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('frame_duration') }}</label>
        <NumberInput v-model="model.edge_offline.frame_duration" :min="1" :max="100" />
      </div>
    </template>

    <!-- Aliyun Qwen TTS -->
    <template v-if="model.provider === 'aliyun_qwen'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Aliyun Qwen TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
        <Input v-model="model.qwen_tts.api_key" type="password" :placeholder="t('enter_api_key')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('region') }}</label>
        <Select v-model="model.qwen_tts.region">
          <SelectTrigger><SelectValue :placeholder="t('select_region')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="beijing">{{ t('beijing') }}</SelectItem>
            <SelectItem value="singapore">{{ t('singapore') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Input v-model="model.qwen_tts.model" placeholder="qwen3-tts-flash" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.qwen_tts.voice" placeholder="Cherry" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('language_type') }}</label>
        <Select v-model="model.qwen_tts.language_type">
          <SelectTrigger><SelectValue :placeholder="t('select_language_type')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="Auto">{{ t('auto') }}</SelectItem>
            <SelectItem value="Chinese">{{ t('chinese_zh') }}</SelectItem>
            <SelectItem value="English">{{ t('english_en') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('use_streaming') }}</label>
        <Switch :checked="model.qwen_tts.stream" @update:checked="v => model.qwen_tts.stream = v" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('frame_duration') }}</label>
        <NumberInput v-model="model.qwen_tts.frame_duration" :min="1" :max="1000" />
      </div>
    </template>

    <!-- Zhipu TTS -->
    <template v-if="model.provider === 'zhipu'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Zhipu TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
        <Input v-model="model.zhipu.api_key" type="password" :placeholder="t('enter_api_key')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API URL</label>
        <Input v-model="model.zhipu.api_url" placeholder="https://open.bigmodel.cn/api/paas/v4/audio/speech" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Input v-model="model.zhipu.model" placeholder="glm-tts" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.zhipu.voice" list="zhipu-voice-list" :placeholder="voiceLoading ? 'Loading...' : t('select_timbre')" :disabled="voiceLoading" />
        <datalist id="zhipu-voice-list">
          <option v-for="opt in voiceOptionsList" :key="opt.value" :value="opt.value" />
        </datalist>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('response_format_label') }}</label>
        <Select v-model="model.zhipu.response_format">
          <SelectTrigger><SelectValue :placeholder="t('select_response_format')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="wav">WAV</SelectItem>
            <SelectItem value="pcm">PCM</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('volume') }}</label>
        <NumberInput v-model="model.zhipu.volume" :min="0" :max="10" :step="0.1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('speech_rate') }}</label>
        <NumberInput v-model="model.zhipu.speed" :min="0.5" :max="2.0" :step="0.1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('use_streaming') }}</label>
        <Switch :checked="model.zhipu.stream" @update:checked="v => model.zhipu.stream = v" />
      </div>
      <div v-if="model.zhipu.stream" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('encoding_format') }}</label>
        <Select v-model="model.zhipu.encode_format">
          <SelectTrigger><SelectValue :placeholder="t('select_encoding_format')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="base64">Base64</SelectItem>
            <SelectItem value="hex">Hex</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('frame_duration') }}</label>
        <NumberInput v-model="model.zhipu.frame_duration" :min="1" :max="1000" />
      </div>
    </template>

    <!-- Minimax TTS -->
    <template v-if="model.provider === 'minimax'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Minimax TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
        <Input v-model="model.minimax.api_key" type="password" :placeholder="t('enter_api_key')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Input v-model="model.minimax.model" placeholder="speech-2.8-hd" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.minimax.voice" list="minimax-voice-list" :placeholder="voiceLoading ? 'Loading...' : t('select_timbre')" :disabled="voiceLoading" />
        <datalist id="minimax-voice-list">
          <option v-for="opt in voiceOptionsList" :key="opt.value" :value="opt.value" />
        </datalist>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('speech_rate') }}</label>
        <NumberInput v-model="model.minimax.speed" :min="0.5" :max="2.0" :step="0.1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('volume') }}</label>
        <NumberInput v-model="model.minimax.vol" :min="0" :max="2" :step="0.1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('pitch') }}</label>
        <NumberInput v-model="model.minimax.pitch" :min="-12" :max="12" :step="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <NumberInput v-model="model.minimax.sample_rate" :min="8000" :max="48000" :step="1000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('bitrate') }}</label>
        <NumberInput v-model="model.minimax.bitrate" :min="32000" :max="320000" :step="16000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('audio_format') }}</label>
        <Select v-model="model.minimax.format">
          <SelectTrigger><SelectValue :placeholder="t('select_audio_format')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="mp3">MP3</SelectItem>
            <SelectItem value="wav">WAV</SelectItem>
            <SelectItem value="pcm">PCM</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('channel_count_label') }}</label>
        <NumberInput v-model="model.minimax.channel" :min="1" :max="2" />
      </div>
    </template>

    <!-- OpenAI TTS -->
    <template v-if="model.provider === 'openai'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">OpenAI TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
        <Input v-model="model.openai.api_key" type="password" :placeholder="t('enter_api_key')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API URL</label>
        <Input v-model="model.openai.api_url" placeholder="https://api.openai.com/v1/audio/speech" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Input v-model="model.openai.model" :placeholder="t('enter_model_default_tts1')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.openai.voice" list="openai-voice-list" :placeholder="voiceLoading ? 'Loading...' : t('select_timbre')" :disabled="voiceLoading" />
        <datalist id="openai-voice-list">
          <option v-for="opt in voiceOptionsList" :key="opt.value" :value="opt.value" />
        </datalist>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('response_format_label') }}</label>
        <Select v-model="model.openai.response_format">
          <SelectTrigger><SelectValue :placeholder="t('select_response_format')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="mp3">MP3</SelectItem>
            <SelectItem value="opus">Opus</SelectItem>
            <SelectItem value="aac">AAC</SelectItem>
            <SelectItem value="flac">FLAC</SelectItem>
            <SelectItem value="wav">WAV</SelectItem>
            <SelectItem value="pcm">PCM</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('speech_rate') }}</label>
        <NumberInput v-model="model.openai.speed" :min="0.25" :max="4.0" :step="0.1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('use_streaming') }}</label>
        <Switch :checked="model.openai.stream" @update:checked="v => model.openai.stream = v" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('frame_duration') }}</label>
        <NumberInput v-model="model.openai.frame_duration" :min="1" :max="1000" />
      </div>
    </template>

    <!-- Xunfei TTS -->
    <template v-if="model.provider === 'xunfei'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('xunfei') }} TTS</span>
        <Separator class="flex-1" />
      </div>
      <XunfeiCommonConfig :model-value="model" prefix="xunfei" default-ws-url="wss://tts-api.xfyun.cn/v2/tts" />
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.xunfei.voice" :placeholder="t('enter_voice_eg_xiaoyan')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('audio_encoding_label') }}</label>
        <Select v-model="model.xunfei.audio_encoding">
          <SelectTrigger><SelectValue :placeholder="t('select_audio_encoding')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="raw">RAW</SelectItem>
            <SelectItem value="opus">Opus</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <Select :model-value="String(model.xunfei.sample_rate)" @update:model-value="v => model.xunfei.sample_rate = Number(v)">
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="8000">8000 Hz</SelectItem>
            <SelectItem value="16000">16000 Hz</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('speech_rate') }}</label>
        <NumberInput v-model="model.xunfei.speed" :min="0" :max="100" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('volume') }}</label>
        <NumberInput v-model="model.xunfei.volume" :min="0" :max="100" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('pitch') }}</label>
        <NumberInput v-model="model.xunfei.pitch" :min="0" :max="100" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('text_encoding') }}</label>
        <Select v-model="model.xunfei.tte">
          <SelectTrigger><SelectValue :placeholder="t('select_text_encoding')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="UTF8">UTF8</SelectItem>
            <SelectItem value="UNICODE">UNICODE</SelectItem>
            <SelectItem value="GB2312">GB2312</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('digit_pronunciation') }}</label>
        <NumberInput v-model="model.xunfei.reg" :min="0" :max="2" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('digit_reading') }}</label>
        <NumberInput v-model="model.xunfei.rdn" :min="0" :max="2" />
      </div>
    </template>

    <!-- Xunfei Super TTS -->
    <template v-if="model.provider === 'xunfei_super_tts'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('xunfei') }} Super TTS</span>
        <Separator class="flex-1" />
      </div>
      <XunfeiCommonConfig :model-value="model" prefix="xunfei_super_tts" default-ws-url="wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6" />
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.xunfei_super_tts.voice" list="xunfei-super-voice-list" :placeholder="voiceLoading ? 'Loading...' : t('select_or_enter_voice_select')" :disabled="voiceLoading" />
        <datalist id="xunfei-super-voice-list">
          <option v-for="opt in voiceOptionsList" :key="opt.value" :value="opt.value" />
        </datalist>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('audio_encoding_label') }}</label>
        <Select v-model="model.xunfei_super_tts.audio_encoding">
          <SelectTrigger><SelectValue :placeholder="t('select_audio_encoding')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="raw">RAW</SelectItem>
            <SelectItem value="opus">Opus</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <Select :model-value="String(model.xunfei_super_tts.sample_rate)" @update:model-value="v => model.xunfei_super_tts.sample_rate = Number(v)">
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="8000">8000 Hz</SelectItem>
            <SelectItem value="16000">16000 Hz</SelectItem>
            <SelectItem value="24000">24000 Hz</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('speech_rate') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.speed" :min="0" :max="100" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('volume') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.volume" :min="0" :max="100" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('pitch') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.pitch" :min="0" :max="100" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('background_sound') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.bgs" :min="0" :max="10" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('digit_pronunciation') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.reg" :min="0" :max="2" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('digit_reading') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.rdn" :min="0" :max="2" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('prosody_enhance') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.rhy" :min="0" :max="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('oral_level') }}</label>
        <Select v-model="model.xunfei_super_tts.oral_level">
          <SelectTrigger><SelectValue :placeholder="t('select_oral_level')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="low">low</SelectItem>
            <SelectItem value="mid">mid</SelectItem>
            <SelectItem value="high">high</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Spark Assist</label>
        <NumberInput v-model="model.xunfei_super_tts.spark_assist" :min="0" :max="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('pause_split') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.stop_split" :min="0" :max="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('retain_oral') }}</label>
        <NumberInput v-model="model.xunfei_super_tts.remain" :min="0" :max="1" />
      </div>
      <p class="text-xs text-[var(--color-text-secondary)] pt-1 pb-2">{{ t('x4_series_hint') }}</p>
    </template>

    <!-- IndexTTS vLLM -->
    <template v-if="model.provider === 'indextts_vllm'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">IndexTTS vLLM</span>
        <Separator class="flex-1" />
      </div>
      <div class="w-full rounded-2xl border border-[var(--color-line)] p-3.5 bg-[var(--color-surface)] mb-2">
        <div class="mb-2">
          <div class="text-sm font-bold text-[var(--color-text)]">{{ t('indextts_guide_title') }}</div>
          <div class="mt-1 text-xs text-[var(--color-text-secondary)]">{{ t('indextts_guide_subtitle') }}</div>
        </div>
        <div class="flex items-center gap-2 mb-2.5 flex-wrap">
          <a :href="indexTTSDocURL" target="_blank" class="text-[var(--color-primary)] text-sm hover:underline">{{ t('indextts_project_docs') }}</a>
          <span class="text-[var(--color-text-tertiary)] text-xs">|</span>
          <a :href="indexTTSReferenceURL" target="_blank" class="text-[var(--color-text-secondary)] text-sm hover:underline">{{ t('indextts_reference_api') }}</a>
        </div>
        <div class="flex gap-2 flex-wrap">
          <span class="text-xs px-2 py-0.5 rounded-full border border-green-500/30 text-green-600 dark:text-green-400">/audio/speech</span>
          <span class="text-xs px-2 py-0.5 rounded-full border border-orange-500/30 text-orange-600 dark:text-orange-400">/audio/voices</span>
          <span class="text-xs px-2 py-0.5 rounded-full border border-[var(--color-line)] text-[var(--color-text-secondary)]">/audio/clone</span>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API URL</label>
        <Input v-model="model.indextts_vllm.api_url" placeholder="http://127.0.0.1:7860" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
        <Input v-model="model.indextts_vllm.api_key" type="password" :placeholder="t('optional_fill_as_needed')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Input v-model="model.indextts_vllm.model" placeholder="indextts-vllm" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('voice_timbre') }}</label>
        <Input v-model="model.indextts_vllm.voice" list="indextts-voice-list" :placeholder="voiceLoading ? 'Loading...' : t('select_timbre')" :disabled="voiceLoading" @focus="handleIndexTTSVoiceFocus" />
        <datalist id="indextts-voice-list">
          <option v-for="opt in voiceOptionsList" :key="opt.value" :value="opt.value" />
        </datalist>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('frame_duration') }}</label>
        <NumberInput v-model="model.indextts_vllm.frame_duration" :min="1" :max="1000" />
      </div>
    </template>

    <!-- CosyVoice TTS -->
    <template v-if="model.provider === 'cosyvoice'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">CosyVoice TTS</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API URL</label>
        <Input v-model="model.cosyvoice.api_url" :placeholder="t('enter_api_url')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('speaker_id') }}</label>
        <Input v-model="model.cosyvoice.spk_id" :placeholder="t('enter_speaker_id')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('frame_duration') }}</label>
        <NumberInput v-model="model.cosyvoice.frame_duration" :min="1" :max="1000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('target_sample_rate') }}</label>
        <NumberInput v-model="model.cosyvoice.target_sr" :min="8000" :max="48000" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('audio_format') }}</label>
        <Select v-model="model.cosyvoice.audio_format">
          <SelectTrigger><SelectValue :placeholder="t('select_audio_format')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="mp3">MP3</SelectItem>
            <SelectItem value="wav">WAV</SelectItem>
            <SelectItem value="pcm">PCM</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('instruct_text') }}</label>
        <Input v-model="model.cosyvoice.instruct_text" :placeholder="t('enter_instruct_text_opt')" />
      </div>
    </template>
  </div>
</template>
