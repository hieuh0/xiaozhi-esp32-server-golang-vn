<script setup>
import { computed, watch } from 'vue'
import { Info } from '@lucide/vue'
import { useLocale } from '../../../composables/useLocale'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import NumberInput from '@/components/ui/number-input.vue'

const { t } = useLocale()

const props = defineProps({
  model: { type: Object, required: true }
})

const ASR_PROVIDER_DEFAULTS = {
  funasr: {
    name: 'FunASR ASR',
    config_id: 'funasr_default',
    data: { host: '127.0.0.1', port: 10095, mode: 'offline', sample_rate: 16000, chunk_size: [5, 10, 5], chunk_interval: 10, max_connections: 100, timeout: 30, auto_end: false }
  },
  aliyun_funasr: {
    name: t('aliyun_funasr_asr'),
    config_id: 'aliyun_funasr_default',
    data: { api_key: '', ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/inference/', model: 'fun-asr-realtime', format: 'pcm', sample_rate: 16000, language_hints: ['zh'], vocabulary_id: '', disfluency_removal_enabled: false, timeout: 30 }
  },
  doubao: {
    name: t('doubao_asr'),
    config_id: 'doubao_default',
    data: { appid: '', access_token: '', ws_url: 'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async', resource_id: 'volc.bigasr.sauc.duration', model_name: 'bigmodel', end_window_size: 800, enable_punc: true, enable_itn: true, enable_ddc: false, chunk_duration: 200, timeout: 30 }
  },
  aliyun_qwen3: {
    name: t('aliyun_qwen3_asr'),
    config_id: 'aliyun_qwen3_default',
    data: { api_key: '', ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/realtime', model: 'qwen3-asr-flash-realtime', format: 'pcm', sample_rate: 16000, language: 'zh', auto_end: false, vad_threshold: 0.0, vad_silence_ms: 400, timeout: 30 }
  },
  xunfei: {
    name: t('xunfei_asr'),
    config_id: 'xunfei_default',
    data: { appid: '', api_key: '', api_secret: '', host: 'iat-api.xfyun.cn', path: '/v2/iat', domain: 'iat', language: 'zh_cn', accent: 'mandarin', sample_rate: 16000, timeout: 30 }
  }
}

const defaultNames = new Set([t('default_asr'), ...Object.values(ASR_PROVIDER_DEFAULTS).map(i => i.name)])
const defaultConfigIds = new Set(Object.values(ASR_PROVIDER_DEFAULTS).flatMap(i => [i.config_id, i.config_id.replace(/_default$/, '')]))

function cloneDefaultData(provider) {
  return JSON.parse(JSON.stringify(ASR_PROVIDER_DEFAULTS[provider]?.data || {}))
}

function normalizeLanguageHints(value) {
  if (Array.isArray(value)) return value.map(s => String(s).trim()).filter(Boolean)
  if (typeof value === 'string') return value.split(/[，,;；]/).map(s => s.trim()).filter(Boolean)
  return []
}

function ensureProviderData(provider) {
  if (!provider || !props.model || !ASR_PROVIDER_DEFAULTS[provider]) return
  const current = props.model[provider]
  props.model[provider] = { ...cloneDefaultData(provider), ...(current || {}) }
  if (provider === 'funasr' && !props.model.funasr.mode) props.model.funasr.mode = 'offline'
  if (provider === 'aliyun_funasr') {
    const hasLanguageHints = current && Object.prototype.hasOwnProperty.call(current, 'language_hints')
    const source = hasLanguageHints ? props.model.aliyun_funasr.language_hints : (props.model.aliyun_funasr.language || props.model.aliyun_funasr.language_hints)
    props.model.aliyun_funasr.language_hints = normalizeLanguageHints(source)
  }
}

function isDefaultish(value, knownValues) {
  return !String(value || '').trim() || knownValues.has(String(value || '').trim())
}

function applyProviderIdentity(provider) {
  if (!provider || !props.model || !ASR_PROVIDER_DEFAULTS[provider]) return
  const defaults = ASR_PROVIDER_DEFAULTS[provider]
  if (isDefaultish(props.model.name, defaultNames)) props.model.name = defaults.name
  if (isDefaultish(props.model.config_id, defaultConfigIds)) props.model.config_id = defaults.config_id
}

function onProviderChange(provider) {
  ensureProviderData(provider)
  applyProviderIdentity(provider)
}

watch(() => props.model?.provider, (provider) => { ensureProviderData(provider) }, { immediate: true })

const languageHintsString = computed({
  get: () => (props.model.aliyun_funasr?.language_hints || []).join(', '),
  set: (v) => { if (props.model.aliyun_funasr) props.model.aliyun_funasr.language_hints = normalizeLanguageHints(v) }
})

function getJsonData() {
  const m = props.model
  if (m.provider === 'funasr') return JSON.stringify(m.funasr || {})
  if (m.provider === 'aliyun_funasr') return JSON.stringify(m.aliyun_funasr || {})
  if (m.provider === 'doubao') return JSON.stringify(m.doubao || {})
  if (m.provider === 'aliyun_qwen3') return JSON.stringify(m.aliyun_qwen3 || {})
  if (m.provider === 'xunfei') return JSON.stringify(m.xunfei || {})
  return '{}'
}

function validate(callback) {
  const m = props.model
  let error = null
  if (!m.name?.trim()) error = t('enter_config_name')
  else if (!m.config_id?.trim()) error = t('enter_config_id')
  else if (!m.provider) error = t('select_provider')
  else if (m.provider === 'funasr' && !m.funasr?.host?.trim()) error = t('enter_host_address')
  else if (m.provider === 'aliyun_funasr') {
    if (!m.aliyun_funasr?.ws_url?.trim()) error = 'WS URL ' + t('required')
    else if (!m.aliyun_funasr?.model?.trim()) error = t('model') + ' ' + t('required')
  } else if (m.provider === 'doubao') {
    if (!m.doubao?.appid?.trim()) error = t('enter_app_id')
    else if (!m.doubao?.access_token?.trim()) error = t('enter_access_token')
  } else if (m.provider === 'aliyun_qwen3') {
    if (!m.aliyun_qwen3?.ws_url?.trim()) error = 'WS URL ' + t('required')
    else if (!m.aliyun_qwen3?.model?.trim()) error = t('model') + ' ' + t('required')
  } else if (m.provider === 'xunfei') {
    if (!m.xunfei?.appid?.trim()) error = t('enter_xunfei_app_id')
    else if (!m.xunfei?.api_key?.trim()) error = t('enter_xunfei_api_key')
    else if (!m.xunfei?.api_secret?.trim()) error = t('enter_xunfei_api_secret')
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
      <Select v-model="model.provider" @update:model-value="onProviderChange">
        <SelectTrigger><SelectValue :placeholder="t('select_provider')" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="funasr">FunASR</SelectItem>
          <SelectItem value="aliyun_funasr">Aliyun FunASR</SelectItem>
          <SelectItem value="doubao">{{ t('doubao') }}</SelectItem>
          <SelectItem value="aliyun_qwen3">Aliyun Qwen3</SelectItem>
          <SelectItem value="xunfei">{{ t('xunfei') }}</SelectItem>
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

    <!-- FunASR -->
    <template v-if="model.provider === 'funasr'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">FunASR</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('host_address') }}</label>
        <Input v-model="model.funasr.host" :placeholder="t('enter_host_address')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('port') }}</label>
        <NumberInput v-model="model.funasr.port" :min="1" :max="65535" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('mode') }}</label>
        <Select v-model="model.funasr.mode">
          <SelectTrigger><SelectValue :placeholder="t('select_mode')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="2pass">2pass</SelectItem>
            <SelectItem value="offline">offline</SelectItem>
            <SelectItem value="online">online</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <Select :model-value="String(model.funasr.sample_rate)" @update:model-value="v => model.funasr.sample_rate = Number(v)">
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="8000">8000 Hz</SelectItem>
            <SelectItem value="16000">16000 Hz</SelectItem>
            <SelectItem value="44100">44100 Hz</SelectItem>
            <SelectItem value="48000">48000 Hz</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('chunk_size') }}</label>
        <div class="space-y-1">
          <div class="flex gap-2">
            <NumberInput v-model="model.funasr.chunk_size[0]" :min="1" />
            <NumberInput v-model="model.funasr.chunk_size[1]" :min="1" />
            <NumberInput v-model="model.funasr.chunk_size[2]" :min="1" />
          </div>
          <p class="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('frame_size_format_hint') }}
          </p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('chunk_interval') }}</label>
        <NumberInput v-model="model.funasr.chunk_interval" :min="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('max_connections') }}</label>
        <NumberInput v-model="model.funasr.max_connections" :min="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('timeout_seconds') }}</label>
        <NumberInput v-model="model.funasr.timeout" :min="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-1">{{ t('auto_end') }}</label>
        <div class="space-y-1">
          <Switch :checked="model.funasr.auto_end" @update:checked="v => model.funasr.auto_end = v" />
          <p class="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('funasr_config_hint') }}
          </p>
        </div>
      </div>
    </template>

    <!-- Aliyun FunASR -->
    <template v-if="model.provider === 'aliyun_funasr'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Aliyun FunASR</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">API Key</label>
        <div class="space-y-1">
          <Input v-model="model.aliyun_funasr.api_key" type="password" :placeholder="t('optional_dashscope_key')" />
          <p class="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('optional_dashscope_fallback') }}
          </p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">WS URL</label>
        <Input v-model="model.aliyun_funasr.ws_url" placeholder="wss://dashscope.aliyuncs.com/api-ws/v1/inference/" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Input v-model="model.aliyun_funasr.model" placeholder="fun-asr-realtime" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('audio_format') }}</label>
        <Select v-model="model.aliyun_funasr.format">
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="pcm">pcm</SelectItem></SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <Select :model-value="String(model.aliyun_funasr.sample_rate)" @update:model-value="v => model.aliyun_funasr.sample_rate = Number(v)">
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="16000">16000 Hz</SelectItem></SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('language_hint') }}</label>
        <Input v-model="languageHintsString" placeholder="zh, en, ja, ko" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('vocab_id') }}</label>
        <Input v-model="model.aliyun_funasr.vocabulary_id" :placeholder="t('optional_empty')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('remove_filler_words') }}</label>
        <Switch :checked="model.aliyun_funasr.disfluency_removal_enabled" @update:checked="v => model.aliyun_funasr.disfluency_removal_enabled = v" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('timeout_seconds') }}</label>
        <NumberInput v-model="model.aliyun_funasr.timeout" :min="1" />
      </div>
    </template>

    <!-- Doubao -->
    <template v-if="model.provider === 'doubao'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('doubao') }}</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('app_id') }}</label>
        <Input v-model="model.doubao.appid" :placeholder="t('enter_app_id')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('access_token') }}</label>
        <Input v-model="model.doubao.access_token" type="password" :placeholder="t('enter_access_token')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">WebSocket URL</label>
        <Input v-model="model.doubao.ws_url" :placeholder="t('enter_websocket_url')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('resource_spec') }}</label>
        <Select v-model="model.doubao.resource_id">
          <SelectTrigger><SelectValue :placeholder="t('select_resource_spec')" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="volc.bigasr.sauc.duration">{{ t('doubao_stream_asr_v1_hourly') }}</SelectItem>
            <SelectItem value="volc.bigasr.sauc.concurrent">{{ t('doubao_stream_asr_v1_concurrent') }}</SelectItem>
            <SelectItem value="volc.seedasr.sauc.duration">{{ t('doubao_stream_asr_v2_hourly') }}</SelectItem>
            <SelectItem value="volc.seedasr.sauc.concurrent">{{ t('doubao_stream_asr_v2_concurrent') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('end_window_size') }}</label>
        <NumberInput v-model="model.doubao.end_window_size" :min="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('enable_punctuation') }}</label>
        <Switch :checked="model.doubao.enable_punc" @update:checked="v => model.doubao.enable_punc = v" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('enable_inverse_text_normalization') }}</label>
        <Switch :checked="model.doubao.enable_itn" @update:checked="v => model.doubao.enable_itn = v" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('enable_digit_detection') }}</label>
        <Switch :checked="model.doubao.enable_ddc" @update:checked="v => model.doubao.enable_ddc = v" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('chunk_duration_ms') }}</label>
        <NumberInput v-model="model.doubao.chunk_duration" :min="1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('timeout_seconds') }}</label>
        <NumberInput v-model="model.doubao.timeout" :min="1" />
      </div>
    </template>

    <!-- Xunfei -->
    <template v-if="model.provider === 'xunfei'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('xunfei') }}</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('app_id') }}</label>
        <Input v-model="model.xunfei.appid" :placeholder="t('enter_xunfei_app_id')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Key</label>
        <Input v-model="model.xunfei.api_key" type="password" :placeholder="t('enter_xunfei_api_key')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">API Secret</label>
        <Input v-model="model.xunfei.api_secret" type="password" :placeholder="t('enter_xunfei_api_secret')" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Host</label>
        <Input v-model="model.xunfei.host" placeholder="iat-api.xfyun.cn" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Path</label>
        <Input v-model="model.xunfei.path" placeholder="/v2/iat" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('business_domain') }}</label>
        <Input v-model="model.xunfei.domain" placeholder="iat" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('language') }}</label>
        <Input v-model="model.xunfei.language" placeholder="zh_cn" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('dialect') }}</label>
        <Input v-model="model.xunfei.accent" placeholder="mandarin" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('sample_rate') }}</label>
        <Select :model-value="String(model.xunfei.sample_rate)" @update:model-value="v => model.xunfei.sample_rate = Number(v)">
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="16000">16000 Hz</SelectItem></SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('timeout_seconds') }}</label>
        <NumberInput v-model="model.xunfei.timeout" :min="1" />
      </div>
    </template>

    <!-- Aliyun Qwen3 -->
    <template v-if="model.provider === 'aliyun_qwen3'">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">Aliyun Qwen3</span>
        <Separator class="flex-1" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">API Key</label>
        <div class="space-y-1">
          <Input v-model="model.aliyun_qwen3.api_key" type="password" :placeholder="t('optional_dashscope_key')" />
          <p class="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('optional_dashscope_fallback') }}
          </p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">WS URL</label>
        <Input v-model="model.aliyun_qwen3.ws_url" placeholder="wss://dashscope.aliyuncs.com/api-ws/v1/realtime" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('model') }}</label>
        <Input v-model="model.aliyun_qwen3.model" placeholder="qwen3-asr-flash-realtime" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('audio_format') }}</label>
        <Select v-model="model.aliyun_qwen3.format">
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="pcm">pcm</SelectItem>
            <SelectItem value="opus">opus</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-2">{{ t('sample_rate') }}</label>
        <div class="space-y-1">
          <Select :model-value="String(model.aliyun_qwen3.sample_rate)" @update:model-value="v => model.aliyun_qwen3.sample_rate = Number(v)">
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="8000">8000 Hz</SelectItem>
              <SelectItem value="16000">16000 Hz</SelectItem>
            </SelectContent>
          </Select>
          <p class="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('sample_rate_16000_hint') }}
          </p>
        </div>
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('language') }}</label>
        <Input v-model="model.aliyun_qwen3.language" placeholder="zh" />
      </div>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0 pt-1">{{ t('auto_end') }}</label>
        <div class="space-y-1">
          <Switch :checked="model.aliyun_qwen3.auto_end" @update:checked="v => model.aliyun_qwen3.auto_end = v" />
          <p class="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
            <Info class="w-3.5 h-3.5 text-[var(--color-primary)] shrink-0" />
            {{ t('server_vad_hint') }}
          </p>
        </div>
      </div>
      <template v-if="model.aliyun_qwen3?.auto_end">
        <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
          <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('vad_threshold') }}</label>
          <NumberInput v-model="model.aliyun_qwen3.vad_threshold" :min="0" :max="1" :step="0.1" />
        </div>
        <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
          <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('vad_silence_duration') }}</label>
          <NumberInput v-model="model.aliyun_qwen3.vad_silence_ms" :min="0" />
        </div>
      </template>
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('timeout_seconds') }}</label>
        <NumberInput v-model="model.aliyun_qwen3.timeout" :min="1" />
      </div>
    </template>
  </div>
</template>
