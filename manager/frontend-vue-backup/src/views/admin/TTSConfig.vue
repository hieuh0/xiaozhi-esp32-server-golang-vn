<script setup>
import { ref, reactive, onMounted, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { testSingleConfig, testWithData, parseJsonData } from '../../utils/configTest'
import TTSConfigForm from './forms/TTSConfigForm.vue'
import { TTS_PROVIDERS_WITH_VOICES } from './forms/ttsProviderOptions'
import { resolveTTSProvider } from './forms/configProviderUtils'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell, TableEmpty } from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

const { t } = useLocale()
const { formatDate } = useFormatDate()

const configs = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const testingId = ref(null)
const testingAll = ref(false)
const testingCurrent = ref(false)
const testResults = ref({})
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)
const formRef = ref()

const voiceOptions = ref([])
const voiceLoading = ref(false)

const form = reactive({
  name: '',
  config_id: '',
  provider: 'doubao_ws',
  is_default: false,
  enabled: true,
  double_stream: false,
  cosyvoice: { api_url: 'https://tts.linkerai.cn/tts', spk_id: 'spk_id', frame_duration: 60, target_sr: 24000, audio_format: 'mp3', instruct_text: t('hello') },
  qwen_tts: { api_key: '', api_url: 'https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation', region: 'beijing', model: 'qwen3-tts-flash', voice: 'Cherry', language_type: 'Chinese', stream: true, frame_duration: 60 },
  doubao: { appid: '6886011847', access_token: 'access_token', model: 'seed-tts-2.0-standard', voice: 'BV001_streaming', api_url: 'https://openspeech.bytedance.com/api/v3/tts/unidirectional' },
  doubao_ws: { appid: '6886011847', access_token: 'access_token', model: 'seed-tts-2.0-standard', resource_id: '', voice: '', ws_url: 'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream' },
  edge: { voice: 'zh-CN-XiaoxiaoNeural', rate: '+0%', volume: '+0%', pitch: '+0Hz', connect_timeout: 10, receive_timeout: 60 },
  edge_offline: { server_url: 'ws://localhost:8080/tts', timeout: 30, sample_rate: 16000, channels: 1, frame_duration: 20 },
  openai: { api_key: '', api_url: 'https://api.openai.com/v1/audio/speech', model: 'tts-1', voice: 'alloy', response_format: 'mp3', speed: 1.0, stream: true, frame_duration: 60 },
  xunfei: { app_id: '', api_key: '', api_secret: '', ws_url: 'wss://tts-api.xfyun.cn/v2/tts', voice: 'xiaoyan', audio_encoding: 'raw', sample_rate: 16000, speed: 50, volume: 50, pitch: 50, tte: 'UTF8', reg: 0, rdn: 0, frame_duration: 60, connect_timeout: 10, read_timeout: 30 },
  xunfei_super_tts: { app_id: '', api_key: '', api_secret: '', ws_url: 'wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6', voice: 'x6_lingxiaoxue_pro', audio_encoding: 'raw', sample_rate: 24000, speed: 50, volume: 50, pitch: 50, bgs: 0, reg: 0, rdn: 0, rhy: 0, oral_level: 'mid', spark_assist: 1, stop_split: 0, remain: 0, frame_duration: 60, connect_timeout: 10, read_timeout: 30 },
  indextts_vllm: { api_url: 'http://127.0.0.1:7860', api_key: '', model: 'indextts-vllm', voice: '', frame_duration: 60 },
  zhipu: { api_key: '', api_url: 'https://open.bigmodel.cn/api/paas/v4/audio/speech', model: 'glm-tts', voice: 'tongtong', response_format: 'pcm', speed: 1.0, volume: 1.0, stream: true, encode_format: 'base64', frame_duration: 60 },
  minimax: { api_key: '', model: 'speech-2.8-hd', voice: 'male-qn-qingse', speed: 1.0, vol: 1.0, pitch: 0, sample_rate: 32000, bitrate: 128000, format: 'mp3', channel: 1 }
})

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/tts-configs', { params: { page: page.value, page_size: pageSize.value } })
    configs.value = (response.data.data || []).map(normalizeTTSConfigRow)
    total.value = response.data.total || 0
  } catch {
    ElMessage.error(t('load_config_failed'))
  } finally {
    loading.value = false
  }
}

function normalizeTTSConfigRow(row) {
  const data = parseJsonData(row?.json_data)
  return { ...row, provider: resolveTTSProvider(row?.provider, row?.config_id, data) }
}

const editConfig = (config) => {
  config = normalizeTTSConfigRow(config)
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.is_default = config.is_default
  form.enabled = config.enabled
  form.double_stream = false

  try {
    const configData = JSON.parse(config.json_data || '{}')
    form.double_stream = configData.double_stream === true
    switch (config.provider) {
      case 'cosyvoice':
        form.cosyvoice.api_url = configData.api_url || ''
        form.cosyvoice.spk_id = configData.spk_id || ''
        form.cosyvoice.frame_duration = configData.frame_duration || 60
        form.cosyvoice.target_sr = configData.target_sr || 24000
        form.cosyvoice.audio_format = configData.audio_format || 'mp3'
        form.cosyvoice.instruct_text = configData.instruct_text || ''
        break
      case 'doubao':
        form.doubao.appid = configData.appid || ''
        form.doubao.access_token = configData.access_token || ''
        form.doubao.model = configData.model || 'seed-tts-2.0-standard'
        form.doubao.voice = configData.voice || ''
        form.doubao.api_url = configData.api_url || 'https://openspeech.bytedance.com/api/v3/tts/unidirectional'
        break
      case 'doubao_ws':
        form.doubao_ws.appid = configData.appid || ''
        form.doubao_ws.access_token = configData.access_token || ''
        form.doubao_ws.model = configData.model || 'seed-tts-2.0-standard'
        form.doubao_ws.resource_id = configData.resource_id || ''
        form.doubao_ws.voice = configData.voice || ''
        form.doubao_ws.ws_url = configData.ws_url || (configData.ws_host ? `wss://${configData.ws_host}/api/v3/tts/unidirectional/stream` : 'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream')
        break
      case 'edge':
        form.edge.voice = configData.voice || ''
        form.edge.rate = configData.rate || '+0%'
        form.edge.volume = configData.volume || '+0%'
        form.edge.pitch = configData.pitch || '+0Hz'
        form.edge.connect_timeout = configData.connect_timeout || 10
        form.edge.receive_timeout = configData.receive_timeout || 60
        break
      case 'edge_offline':
        form.edge_offline.server_url = configData.server_url || ''
        form.edge_offline.timeout = configData.timeout || 30
        form.edge_offline.sample_rate = configData.sample_rate || 16000
        form.edge_offline.channels = configData.channels || 1
        form.edge_offline.frame_duration = configData.frame_duration || 20
        break
      case 'aliyun_qwen':
        form.qwen_tts.api_key = configData.api_key || ''
        form.qwen_tts.api_url = configData.api_url || 'https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation'
        form.qwen_tts.region = configData.region || 'beijing'
        form.qwen_tts.model = configData.model || 'qwen3-tts-flash'
        form.qwen_tts.voice = configData.voice || 'Cherry'
        form.qwen_tts.language_type = configData.language_type || 'Chinese'
        form.qwen_tts.stream = configData.stream !== undefined ? configData.stream : true
        form.qwen_tts.frame_duration = configData.frame_duration || 60
        break
      case 'openai':
        form.openai.api_key = configData.api_key || ''
        form.openai.api_url = configData.api_url || 'https://api.openai.com/v1/audio/speech'
        form.openai.model = configData.model || 'tts-1'
        form.openai.voice = configData.voice || 'alloy'
        form.openai.response_format = configData.response_format || 'mp3'
        form.openai.speed = configData.speed || 1.0
        form.openai.stream = configData.stream !== undefined ? configData.stream : true
        form.openai.frame_duration = configData.frame_duration || 60
        break
      case 'xunfei':
        form.xunfei.app_id = configData.app_id || ''
        form.xunfei.api_key = configData.api_key || ''
        form.xunfei.api_secret = configData.api_secret || ''
        form.xunfei.ws_url = configData.ws_url || 'wss://tts-api.xfyun.cn/v2/tts'
        form.xunfei.voice = configData.voice || 'xiaoyan'
        form.xunfei.audio_encoding = configData.audio_encoding || 'raw'
        form.xunfei.sample_rate = configData.sample_rate || 16000
        form.xunfei.speed = configData.speed ?? 50
        form.xunfei.volume = configData.volume ?? 50
        form.xunfei.pitch = configData.pitch ?? 50
        form.xunfei.tte = configData.tte || 'UTF8'
        form.xunfei.reg = configData.reg ?? 0
        form.xunfei.rdn = configData.rdn ?? 0
        form.xunfei.frame_duration = configData.frame_duration || 60
        form.xunfei.connect_timeout = configData.connect_timeout || 10
        form.xunfei.read_timeout = configData.read_timeout || 30
        break
      case 'xunfei_super_tts':
        form.xunfei_super_tts.app_id = configData.app_id || ''
        form.xunfei_super_tts.api_key = configData.api_key || ''
        form.xunfei_super_tts.api_secret = configData.api_secret || ''
        form.xunfei_super_tts.ws_url = configData.ws_url || 'wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6'
        form.xunfei_super_tts.voice = configData.voice || 'x6_lingxiaoxue_pro'
        form.xunfei_super_tts.audio_encoding = configData.audio_encoding || 'raw'
        form.xunfei_super_tts.sample_rate = configData.sample_rate || 24000
        form.xunfei_super_tts.speed = configData.speed ?? 50
        form.xunfei_super_tts.volume = configData.volume ?? 50
        form.xunfei_super_tts.pitch = configData.pitch ?? 50
        form.xunfei_super_tts.bgs = configData.bgs ?? 0
        form.xunfei_super_tts.reg = configData.reg ?? 0
        form.xunfei_super_tts.rdn = configData.rdn ?? 0
        form.xunfei_super_tts.rhy = configData.rhy ?? 0
        form.xunfei_super_tts.oral_level = configData.oral_level || 'mid'
        form.xunfei_super_tts.spark_assist = configData.spark_assist ?? 1
        form.xunfei_super_tts.stop_split = configData.stop_split ?? 0
        form.xunfei_super_tts.remain = configData.remain ?? 0
        form.xunfei_super_tts.frame_duration = configData.frame_duration || 60
        form.xunfei_super_tts.connect_timeout = configData.connect_timeout || 10
        form.xunfei_super_tts.read_timeout = configData.read_timeout || 30
        break
      case 'indextts_vllm':
        form.indextts_vllm.api_url = configData.api_url || 'http://127.0.0.1:7860'
        form.indextts_vllm.api_key = configData.api_key || ''
        form.indextts_vllm.model = configData.model || 'indextts-vllm'
        form.indextts_vllm.voice = configData.voice || ''
        form.indextts_vllm.frame_duration = configData.frame_duration || 60
        break
      case 'zhipu':
        form.zhipu.api_key = configData.api_key || ''
        form.zhipu.api_url = configData.api_url || 'https://open.bigmodel.cn/api/paas/v4/audio/speech'
        form.zhipu.model = configData.model || 'glm-tts'
        form.zhipu.voice = configData.voice || 'tongtong'
        form.zhipu.response_format = configData.response_format || 'pcm'
        form.zhipu.speed = configData.speed || 1.0
        form.zhipu.volume = configData.volume || 1.0
        form.zhipu.stream = configData.stream !== undefined ? configData.stream : true
        form.zhipu.encode_format = configData.encode_format || 'base64'
        form.zhipu.frame_duration = configData.frame_duration || 60
        break
      case 'minimax':
        form.minimax.api_key = configData.api_key || ''
        form.minimax.model = configData.model || 'speech-2.8-hd'
        form.minimax.voice = configData.voice || 'male-qn-qingse'
        form.minimax.speed = configData.speed || 1.0
        form.minimax.vol = configData.vol || configData.volume || 1.0
        form.minimax.pitch = configData.pitch || 0
        form.minimax.sample_rate = configData.sample_rate || 32000
        form.minimax.bitrate = configData.bitrate || 128000
        form.minimax.format = configData.format || 'mp3'
        form.minimax.channel = configData.channel || 1
        break
    }
  } catch (error) {
    console.error(t('parse_config_json_failed'), error)
  }
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch (_) {
    return
  }
  saving.value = true
  try {
    const isFirstConfig = !editingConfig.value && configs.value.length === 0
    const configData = {
      name: form.name,
      config_id: form.config_id,
      provider: form.provider,
      is_default: isFirstConfig || form.is_default,
      enabled: form.enabled !== undefined ? form.enabled : true,
      json_data: formRef.value.getJsonData()
    }
    if (editingConfig.value) {
      await api.put(`/admin/tts-configs/${editingConfig.value.id}`, configData)
      ElMessage.success(t('config_update_success'))
    } else {
      await api.post('/admin/tts-configs', configData)
      ElMessage.success(t('config_create_success'))
    }
    showDialog.value = false
    loadConfigs()
  } catch (error) {
    ElMessage.error(t('save_failed_colon') + (error.response?.data?.message || error.message))
  } finally {
    saving.value = false
  }
}

const toggleEnable = async (config) => {
  try {
    await api.post(`/admin/configs/${config.id}/toggle`)
    ElMessage.success(config.enabled ? t('enabled_success') : t('disable_success'))
  } catch {
    config.enabled = !config.enabled
    ElMessage.error(t('operation_failed'))
  }
}

const toggleDefault = async (config) => {
  try {
    if (!config.enabled) {
      ElMessage.warning(t('enable_config_before_default'))
      config.is_default = false
      return
    }
    await api.put(`/admin/tts-configs/${config.id}`, { name: config.name, config_id: config.config_id, provider: config.provider, is_default: config.is_default, enabled: config.enabled, json_data: config.json_data })
    ElMessage.success(config.is_default ? t('set_default_success') : t('cancel_default_success'))
    loadConfigs()
  } catch {
    config.is_default = !config.is_default
    ElMessage.error(t('operation_failed'))
  }
}

const getEnabledConfigs = () => configs.value.filter(c => c.enabled)

function formatTestResultLabel(r) {
  if (!r?.ok) return t('error')
  return r.first_packet_ms != null ? t('correct_result', { ms: r.first_packet_ms }) : t('correct')
}
function formatTestResultTip(r) {
  if (!r?.ok) return ''
  return r.first_packet_ms != null ? t('passed_result', { ms: r.first_packet_ms }) : t('passed')
}
function formatTestMessage(result) {
  const base = result.message || ''
  return result.first_packet_ms != null ? `${base} ${result.first_packet_ms}ms` : base
}

const testConfig = async (row) => {
  testingId.value = row.config_id
  try {
    const result = await testSingleConfig('tts', row.config_id)
    testResults.value = { ...testResults.value, [row.config_id]: result }
    if (result.ok) {
      ElMessage.success(`${row.name || row.config_id}${t('label_colon')}${formatTestMessage(result)}`)
    } else {
      ElMessage.warning(`${row.name || row.config_id}${t('label_colon')}${result.message}`)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || t('test_request_failed_v2'))
  } finally {
    testingId.value = null
  }
}

const testAllConfigs = async () => {
  const list = getEnabledConfigs()
  if (!list.length) { ElMessage.warning(t('no_enabled_config')); return }
  testingAll.value = true
  testResults.value = {}
  let okCount = 0
  try {
    for (const row of list) {
      try {
        const result = await testSingleConfig('tts', row.config_id)
        testResults.value = { ...testResults.value, [row.config_id]: result }
        if (result.ok) okCount++
      } catch {
        testResults.value = { ...testResults.value, [row.config_id]: { ok: false, message: t('request_failed') } }
      }
    }
    ElMessage.success(t('all_tests_complete_msg', { ok: okCount, total: list.length }))
  } catch (err) {
    ElMessage.error(err.response?.data?.error || t('test_request_failed_v2'))
  } finally {
    testingAll.value = false
  }
}

const testCurrentConfig = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch (_) {
    return
  }
  const configId = form.config_id?.trim()
  if (!configId) { ElMessage.warning(t('fill_config_id')); return }
  const payload = { name: form.name, config_id: configId, provider: form.provider, is_default: form.is_default, ...parseJsonData(formRef.value.getJsonData()) }
  testingCurrent.value = true
  try {
    const result = await testWithData('tts', { [configId]: payload })
    if (result.ok) {
      ElMessage.success(formatTestMessage(result) || t('test_passed'))
    } else {
      ElMessage.warning(result.message || t('test_not_passed'))
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || t('test_request_failed_v2'))
  } finally {
    testingCurrent.value = false
  }
}

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_config'), t('hint'), { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' })
    await api.delete(`/admin/tts-configs/${id}`)
    ElMessage.success(t('delete_success'))
    loadConfigs()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('delete_failed'))
  }
}

const resetForm = () => {
  editingConfig.value = null
  Object.assign(form, {
    name: '', config_id: '', provider: 'doubao_ws', is_default: false, enabled: true,
    cosyvoice: { api_url: 'https://tts.linkerai.top/tts', spk_id: 'spk_id', frame_duration: 60, target_sr: 24000, audio_format: 'mp3', instruct_text: t('hello') },
    qwen_tts: { api_key: '', api_url: 'https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation', region: 'beijing', model: 'qwen3-tts-flash', voice: 'Cherry', language_type: 'Chinese', stream: true, frame_duration: 60 },
    doubao: { appid: '6886011847', access_token: 'access_token', model: 'seed-tts-2.0-standard', voice: 'BV001_streaming', api_url: 'https://openspeech.bytedance.com/api/v3/tts/unidirectional' },
    doubao_ws: { appid: '6886011847', access_token: 'access_token', model: 'seed-tts-2.0-standard', resource_id: '', voice: '', ws_url: 'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream' },
    edge: { voice: 'zh-CN-XiaoxiaoNeural', rate: '+0%', volume: '+0%', pitch: '+0Hz', connect_timeout: 10, receive_timeout: 60 },
    edge_offline: { server_url: 'ws://localhost:8080/tts', timeout: 30, sample_rate: 16000, channels: 1, frame_duration: 20 },
    openai: { api_key: '', api_url: 'https://api.openai.com/v1/audio/speech', model: 'tts-1', voice: 'alloy', response_format: 'mp3', speed: 1.0, stream: true, frame_duration: 60 },
    xunfei: { app_id: '', api_key: '', api_secret: '', ws_url: 'wss://tts-api.xfyun.cn/v2/tts', voice: 'xiaoyan', audio_encoding: 'raw', sample_rate: 16000, speed: 50, volume: 50, pitch: 50, tte: 'UTF8', reg: 0, rdn: 0, frame_duration: 60, connect_timeout: 10, read_timeout: 30 },
    xunfei_super_tts: { app_id: '', api_key: '', api_secret: '', ws_url: 'wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6', voice: 'x6_lingxiaoxue_pro', audio_encoding: 'raw', sample_rate: 24000, speed: 50, volume: 50, pitch: 50, bgs: 0, reg: 0, rdn: 0, rhy: 0, oral_level: 'mid', spark_assist: 1, stop_split: 0, remain: 0, frame_duration: 60, connect_timeout: 10, read_timeout: 30 },
    indextts_vllm: { api_url: 'http://127.0.0.1:7860', api_key: '', model: 'indextts-vllm', voice: '', frame_duration: 60 },
    zhipu: { api_key: '', api_url: 'https://open.bigmodel.cn/api/paas/v4/audio/speech', model: 'glm-tts', voice: 'tongtong', response_format: 'pcm', speed: 1.0, volume: 1.0, stream: true, frame_duration: 60 },
    minimax: { api_key: '', model: 'speech-2.8-hd', voice: 'male-qn-qingse', speed: 1.0, vol: 1.0, pitch: 0, sample_rate: 32000, bitrate: 128000, format: 'mp3', channel: 1 }
  })
}

const openCreateDialog = () => {
  resetForm()
  showDialog.value = true
}

const handleAction = (command, row) => {
  if (command === 'edit') editConfig(row)
  else if (command === 'test') testConfig(row)
  else if (command === 'delete') deleteConfig(row.id)
}

const loadVoiceOptions = async (provider, options = {}) => {
  const trigger = options?.trigger || 'auto'
  if (!provider) { voiceOptions.value = []; return }
  if (provider === 'indextts_vllm' && trigger !== 'dropdown') { voiceOptions.value = []; return }
  if (!TTS_PROVIDERS_WITH_VOICES.includes(provider)) { voiceOptions.value = []; return }
  voiceLoading.value = true
  try {
    const params = { provider, config_id: form.config_id || undefined }
    if (provider === 'indextts_vllm') {
      const apiURL = String(form.indextts_vllm?.api_url || '').trim()
      const apiKey = String(form.indextts_vllm?.api_key || '').trim()
      if (apiURL) params.api_url = apiURL
      if (apiKey) params.api_key = apiKey
    }
    const response = await api.get('/user/voice-options', { params })
    voiceOptions.value = response.data.data || []
  } catch (error) {
    console.error(t('load_voice_list_failed_c'), error)
    voiceOptions.value = []
  } finally {
    voiceLoading.value = false
  }
}

const handleVoiceOptionsRequest = (provider) => {
  if (!showDialog.value) return
  loadVoiceOptions(provider || form.provider, { trigger: 'dropdown' })
}

watch(() => form.provider, (newProvider) => {
  if (showDialog.value) loadVoiceOptions(newProvider)
}, { immediate: false })

watch(showDialog, (isOpen) => {
  if (isOpen) {
    nextTick(() => loadVoiceOptions(form.provider))
  } else {
    resetForm()
  }
})

onMounted(() => { loadConfigs() })
</script>

<template>
  <div class="ck-page">
    <!-- Top actions -->
    <div class="flex justify-end items-center gap-2 flex-wrap mb-5">
      <Button variant="outline" :disabled="testingAll || !getEnabledConfigs().length" @click="testAllConfigs">
        {{ t('test_all') }}
      </Button>
      <Button @click="openCreateDialog">
        <Plus class="w-4 h-4 mr-1" />
        {{ t('add_config') }}
      </Button>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-[var(--color-line)] overflow-hidden">
      <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
      <Table v-else>
        <TableHeader>
          <TableRow>
            <TableHead class="w-16">ID</TableHead>
            <TableHead>{{ t('config_name') }}</TableHead>
            <TableHead class="w-36">{{ t('config_id') }}</TableHead>
            <TableHead>{{ t('provider') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('enabled_status') }}</TableHead>
            <TableHead class="w-20 text-center">{{ t('default_config') }}</TableHead>
            <TableHead class="w-28 text-center">{{ t('test_result') }}</TableHead>
            <TableHead class="w-44">{{ t('created_at') }}</TableHead>
            <TableHead class="w-16 text-center">{{ t('actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableEmpty v-if="!configs.length" :colspan="9" />
          <TableRow v-for="row in configs" :key="row.id">
            <TableCell class="text-[var(--color-text-secondary)] text-xs">{{ row.id }}</TableCell>
            <TableCell class="font-medium">{{ row.name }}</TableCell>
            <TableCell class="text-xs text-[var(--color-text-secondary)]">{{ row.config_id }}</TableCell>
            <TableCell>{{ row.provider }}</TableCell>
            <TableCell class="text-center">
              <Switch :checked="row.enabled" @update:checked="v => { row.enabled = v; toggleEnable(row) }" />
            </TableCell>
            <TableCell class="text-center">
              <Switch :checked="row.is_default" :disabled="row.is_default && getEnabledConfigs().length === 1" @update:checked="v => { row.is_default = v; toggleDefault(row) }" />
            </TableCell>
            <TableCell class="text-center">
              <template v-if="testResults[row.config_id]">
                <TooltipProvider v-if="testResults[row.config_id].ok">
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <span class="text-xs font-medium text-green-600 dark:text-green-400 cursor-default">{{ formatTestResultLabel(testResults[row.config_id]) }}</span>
                    </TooltipTrigger>
                    <TooltipContent>{{ formatTestResultTip(testResults[row.config_id]) }}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <TooltipProvider v-else>
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <span class="text-xs font-medium text-red-500 cursor-help">{{ t('error') }}</span>
                    </TooltipTrigger>
                    <TooltipContent>{{ testResults[row.config_id].message }}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </template>
              <span v-else class="text-xs text-[var(--color-text-tertiary)]">-</span>
            </TableCell>
            <TableCell class="text-xs text-[var(--color-text-secondary)]">{{ formatDate(row.created_at) }}</TableCell>
            <TableCell class="text-center">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="icon" class="h-7 w-7" :aria-label="t('more_actions')">
                    <MoreHorizontal class="w-4 h-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem @click="handleAction('edit', row)">{{ t('edit') }}</DropdownMenuItem>
                  <DropdownMenuItem @click="handleAction('test', row)">{{ t('test') }}</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem class="text-destructive focus:text-destructive" @click="handleAction('delete', row)">{{ t('delete') }}</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <!-- Pagination -->
    <div v-if="total > pageSize" class="flex items-center justify-end gap-2 text-sm mt-4">
      <span class="text-[var(--color-text-secondary)]">{{ total }} items</span>
      <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadConfigs()">Prev</Button>
      <span class="text-[var(--color-text-secondary)]">{{ page }}</span>
      <Button variant="outline" size="sm" :disabled="page * pageSize >= total" @click="page++; loadConfigs()">Next</Button>
    </div>

    <!-- Add/Edit Dialog -->
    <Dialog v-model:open="showDialog">
      <DialogContent class="max-w-[620px]">
        <DialogHeader>
          <DialogTitle>{{ editingConfig ? t('edit_tts_config') : t('add_tts_config') }}</DialogTitle>
        </DialogHeader>
        <div class="max-h-[65vh] overflow-y-auto pr-1">
          <TTSConfigForm
            ref="formRef"
            :model="form"
            :voice-options="voiceOptions"
            :voice-loading="voiceLoading"
            @request-voice-options="handleVoiceOptionsRequest"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showDialog = false">{{ t('cancel') }}</Button>
          <Button variant="outline" :disabled="testingCurrent" @click="testCurrentConfig">{{ t('test') }}</Button>
          <Button :disabled="saving" @click="handleSave">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
