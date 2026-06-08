// Composable: all state, form models, validation rules, step navigation and API calls for ConfigWizard
import { ref, reactive, computed, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../utils/api'
import { testWithData, parseJsonData } from '../utils/configTest'
import { resolveASRProvider, resolveTTSProvider, resolveVADProvider } from '../views/admin/forms/configProviderUtils'
import { getProviderFixedType, resolveLLMProvider } from '../views/admin/forms/llmCatalog'
import { useLocale } from './useLocale'

export function useConfigWizard() {
  const { t } = useLocale()

  const currentStep = ref(0)
  const saving = ref(false)
  const testingStep = ref(false)
  const otaTestLoading = ref(false)
  const otaTestResult = ref(null)

  // Config IDs for update vs create
  const otaConfigId = ref(null)
  const vadConfigId = ref(null)
  const asrConfigId = ref(null)
  const llmConfigId = ref(null)
  const ttsConfigId = ref(null)

  // --- OTA form ---
  const otaForm = reactive({
    host: '', port: 8989, protocol: 'http',
    signature_key: 'xiaozhi_ota_signature_key',
    enableMqttUdp: false, mqttServerPort: 1883, udpPort: 8990
  })

  // --- VAD form ---
  const vadForm = reactive({
    name: t('default_vad'), config_id: 'ten_vad_default', provider: 'ten_vad',
    webrtc_vad: { pool_min_size: 5, pool_max_size: 1000, pool_max_idle: 100, vad_sample_rate: 16000, vad_mode: 2 },
    silero_vad: { model_path: 'config/models/vad/silero_vad.onnx', threshold: 0.5, min_silence_duration_ms: 100, sample_rate: 16000, channels: 1, acquire_timeout_ms: 3000 },
    ten_vad: { hop_size: 320, threshold: 0.3, pool_size: 10, acquire_timeout_ms: 3000 }
  })
  const vadFormRef = ref()
  const vadFormRules = {
    name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
    config_id: [{ required: true, message: t('enter_config_id'), trigger: 'blur' }],
    provider: [{ required: true, message: t('select_provider'), trigger: 'change' }],
    'silero_vad.model_path': [{ required: true, message: t('enter_model_path'), trigger: 'blur' }],
    'silero_vad.threshold': [{ required: true, message: t('enter_threshold'), trigger: 'blur' }],
    'silero_vad.min_silence_duration_ms': [{ required: true, message: t('enter_min_silence_duration'), trigger: 'blur' }],
    'silero_vad.sample_rate': [{ required: true, message: t('select_sample_rate'), trigger: 'change' }],
    'silero_vad.channels': [{ required: true, message: t('select_channel_count'), trigger: 'change' }],
    'silero_vad.acquire_timeout_ms': [{ required: true, message: t('enter_fetch_timeout'), trigger: 'blur' }],
    'ten_vad.hop_size': [{ required: true, message: t('enter_frame_shift'), trigger: 'blur' }],
    'ten_vad.threshold': [{ required: true, message: t('enter_vad_threshold'), trigger: 'blur' }],
    'ten_vad.pool_size': [{ required: true, message: t('enter_pool_size'), trigger: 'blur' }],
    'ten_vad.acquire_timeout_ms': [{ required: true, message: t('enter_fetch_timeout'), trigger: 'blur' }]
  }

  // --- ASR form ---
  const asrForm = reactive({
    name: 'FunASR ASR', config_id: 'funasr_default', provider: 'funasr',
    funasr: { host: '127.0.0.1', port: 10095, mode: 'offline', sample_rate: 16000, chunk_size: [5, 10, 5], chunk_interval: 10, max_connections: 100, timeout: 30, auto_end: false },
    aliyun_funasr: { api_key: '', ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/inference/', model: 'fun-asr-realtime', format: 'pcm', sample_rate: 16000, language_hints: ['zh'], vocabulary_id: '', disfluency_removal_enabled: false, timeout: 30 },
    doubao: { appid: '', access_token: '', ws_url: 'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async', resource_id: 'volc.bigasr.sauc.duration', model_name: 'bigmodel', end_window_size: 800, enable_punc: true, enable_itn: true, enable_ddc: false, chunk_duration: 200, timeout: 30 },
    aliyun_qwen3: { api_key: '', ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/realtime', model: 'qwen3-asr-flash-realtime', format: 'pcm', sample_rate: 16000, language: 'zh', auto_end: false, vad_threshold: 0.0, vad_silence_ms: 400, timeout: 30 },
    xunfei: { appid: '', api_key: '', api_secret: '', host: 'iat-api.xfyun.cn', path: '/v2/iat', domain: 'iat', language: 'zh_cn', accent: 'mandarin', sample_rate: 16000, timeout: 30 }
  })
  const asrFormRef = ref()
  const validateAliyunPcm = (rule, value, callback) => { value !== 'pcm' ? callback(new Error(t('format_must_pcm'))) : callback() }
  const validateAliyun16000 = (rule, value, callback) => { Number(value) !== 16000 ? callback(new Error(t('sample_rate_must_16000'))) : callback() }
  const asrFormRules = {
    name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
    config_id: [{ required: true, message: t('enter_config_id'), trigger: 'blur' }],
    provider: [{ required: true, message: t('select_provider'), trigger: 'change' }],
    'funasr.host': [{ required: true, message: t('enter_host_address'), trigger: 'blur' }],
    'funasr.port': [{ required: true, message: t('enter_port'), trigger: 'blur' }],
    'aliyun_funasr.ws_url': [{ required: true, message: t('enter_ws_url'), trigger: 'blur' }],
    'aliyun_funasr.model': [{ required: true, message: t('enter_model_name'), trigger: 'blur' }],
    'aliyun_funasr.format': [{ required: true, message: t('select_audio_format'), trigger: 'change' }, { validator: validateAliyunPcm, trigger: 'change' }],
    'aliyun_funasr.sample_rate': [{ required: true, message: t('select_sample_rate'), trigger: 'change' }, { validator: validateAliyun16000, trigger: 'change' }],
    'aliyun_funasr.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }],
    'doubao.appid': [{ required: true, message: t('enter_app_id'), trigger: 'blur' }],
    'doubao.access_token': [{ required: true, message: t('enter_access_token'), trigger: 'blur' }],
    'doubao.ws_url': [{ required: true, message: t('enter_websocket_url'), trigger: 'blur' }],
    'doubao.resource_id': [{ required: true, message: t('select_resource_spec'), trigger: 'change' }],
    'aliyun_qwen3.ws_url': [{ required: true, message: t('enter_ws_url'), trigger: 'blur' }],
    'aliyun_qwen3.model': [{ required: true, message: t('enter_model_name'), trigger: 'blur' }],
    'aliyun_qwen3.format': [{ required: true, message: t('select_audio_format'), trigger: 'change' }],
    'aliyun_qwen3.sample_rate': [{ required: true, message: t('select_sample_rate'), trigger: 'change' }],
    'aliyun_qwen3.language': [{ required: true, message: t('enter_language'), trigger: 'blur' }],
    'aliyun_qwen3.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }],
    'xunfei.appid': [{ required: true, message: t('enter_app_id'), trigger: 'blur' }],
    'xunfei.api_key': [{ required: true, message: t('enter_api_key'), trigger: 'blur' }],
    'xunfei.api_secret': [{ required: true, message: t('enter_api_secret'), trigger: 'blur' }],
    'xunfei.host': [{ required: true, message: t('enter_host'), trigger: 'blur' }],
    'xunfei.path': [{ required: true, message: t('enter_path'), trigger: 'blur' }],
    'xunfei.domain': [{ required: true, message: t('enter_business_domain'), trigger: 'blur' }],
    'xunfei.language': [{ required: true, message: t('enter_language'), trigger: 'blur' }],
    'xunfei.accent': [{ required: true, message: t('enter_dialect'), trigger: 'blur' }],
    'xunfei.sample_rate': [{ required: true, message: t('select_sample_rate'), trigger: 'change' }],
    'xunfei.timeout': [{ required: true, message: t('enter_timeout'), trigger: 'blur' }]
  }

  // --- LLM form ---
  const llmForm = reactive({
    name: t('default_llm'), config_id: 'openai_default', provider: 'openai', type: 'openai',
    model_name: 'gpt-3.5-turbo', api_key: '', base_url: 'https://api.openai.com/v1',
    max_tokens: 4000, temperature: 0.7, top_p: 0.9,
    thinking_mode: 'default', thinking_budget_tokens: null, thinking_effort: 'medium', thinking_clear_thinking: 'default'
  })
  const llmFormRef = ref()
  const getResolvedLLMType = (provider, type) => getProviderFixedType(resolveLLMProvider(provider, type))
  const getResolvedLLMProvider = (provider, type) => resolveLLMProvider(provider, type)
  const llmFormRules = {
    name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
    config_id: [{ required: true, message: t('enter_config_id'), trigger: 'blur' }],
    provider: [{ required: true, message: t('select_provider'), trigger: 'change' }],
    model_name: [
      { required: true, message: t('enter_model_name'), trigger: 'change' },
      { validator: (_, value, callback) => { const pt = getResolvedLLMType(llmForm.provider, llmForm.type); if ((pt === 'openai' || pt === 'ollama') && !value) { callback(new Error(t('enter_model_name'))); return } callback() }, trigger: 'change' }
    ],
    api_key: [{ validator: (_, value, callback) => { if (getResolvedLLMType(llmForm.provider, llmForm.type) !== 'ollama' && !value) { callback(new Error(t('enter_api_password'))); return } callback() }, trigger: 'blur' }],
    base_url: [{ validator: (_, value, callback) => { if (getResolvedLLMType(llmForm.provider, llmForm.type) !== 'coze' && !value) { callback(new Error(t('enter_base_url'))); return } callback() }, trigger: 'blur' }],
    max_tokens: [{ validator: (_, value, callback) => { const pt = getResolvedLLMType(llmForm.provider, llmForm.type); if ((pt === 'openai' || pt === 'ollama') && (!value || Number(value) < 1 || Number(value) > 100000)) { callback(new Error(t('max_tokens_range'))); return } callback() }, trigger: 'blur' }]
  }

  // --- TTS form ---
  const ttsForm = reactive({
    name: t('default_tts'), config_id: 'minimax_default', provider: 'minimax', double_stream: false,
    qwen_tts: { api_key: '', api_url: 'https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation', region: 'beijing', model: 'qwen3-tts-flash', voice: 'Cherry', language_type: 'Chinese', stream: true, frame_duration: 60 },
    doubao_ws: { appid: '', access_token: '', model: 'seed-tts-2.0-standard', resource_id: '', voice: '', ws_url: 'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream' },
    edge: { voice: 'zh-CN-XiaoxiaoNeural', rate: '+0%', volume: '+0%', pitch: '+0Hz', connect_timeout: 10, receive_timeout: 60 },
    edge_offline: { server_url: 'ws://localhost:8080/tts', timeout: 30, sample_rate: 16000, channels: 1, frame_duration: 20 },
    openai: { api_key: '', api_url: 'https://api.openai.com/v1/audio/speech', model: 'tts-1', voice: 'alloy', response_format: 'mp3', speed: 1.0, stream: true, frame_duration: 60 },
    xunfei: { app_id: '', api_key: '', api_secret: '', ws_url: 'wss://tts-api.xfyun.cn/v2/tts', voice: 'xiaoyan', audio_encoding: 'raw', sample_rate: 16000, speed: 50, volume: 50, pitch: 50, tte: 'UTF8', reg: 0, rdn: 0, frame_duration: 60, connect_timeout: 10, read_timeout: 30 },
    xunfei_super_tts: { app_id: '', api_key: '', api_secret: '', ws_url: 'wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6', voice: 'x6_lingxiaoxue_pro', audio_encoding: 'raw', sample_rate: 24000, speed: 50, volume: 50, pitch: 50, bgs: 0, reg: 0, rdn: 0, rhy: 0, oral_level: 'mid', spark_assist: 1, stop_split: 0, remain: 0, frame_duration: 60, connect_timeout: 10, read_timeout: 30 },
    zhipu: { api_key: '', api_url: 'https://open.bigmodel.cn/api/paas/v4/audio/speech', model: 'glm-tts', voice: 'tongtong', response_format: 'pcm', speed: 1.0, volume: 1.0, stream: true, encode_format: 'base64', frame_duration: 60 },
    minimax: { api_key: '', model: 'speech-2.8-hd', voice: 'male-qn-qingse', speed: 1.0, vol: 1.0, pitch: 0, sample_rate: 32000, bitrate: 128000, format: 'mp3', channel: 1 }
  })
  const ttsFormRef = ref()
  const voiceOptions = ref([])
  const voiceLoading = ref(false)
  const ttsFormRules = {
    name: [{ required: true, message: t('enter_config_name'), trigger: 'blur' }],
    config_id: [{ required: true, message: t('enter_config_id'), trigger: 'blur' }],
    provider: [{ required: true, message: t('select_provider'), trigger: 'change' }],
    'doubao_ws.appid': [{ required: true, message: t('enter_app_id'), trigger: 'blur' }],
    'doubao_ws.access_token': [{ required: true, message: t('enter_access_token'), trigger: 'blur' }],
    'doubao_ws.model': [{ required: true, message: t('select_model'), trigger: 'change' }],
    'doubao_ws.ws_url': [{ required: true, message: t('enter_websocket_url'), trigger: 'blur' }],
    'xunfei.app_id': [{ required: true, message: t('enter_app_id'), trigger: 'blur' }],
    'xunfei.api_key': [{ required: true, message: t('enter_api_key'), trigger: 'blur' }],
    'xunfei.api_secret': [{ required: true, message: t('enter_api_secret'), trigger: 'blur' }],
    'xunfei.ws_url': [{ required: true, message: t('enter_websocket_url'), trigger: 'blur' }],
    'xunfei.voice': [{ required: true, message: t('enter_voice_timbre'), trigger: 'blur' }],
    'xunfei_super_tts.app_id': [{ required: true, message: t('enter_app_id'), trigger: 'blur' }],
    'xunfei_super_tts.api_key': [{ required: true, message: t('enter_api_key'), trigger: 'blur' }],
    'xunfei_super_tts.api_secret': [{ required: true, message: t('enter_api_secret'), trigger: 'blur' }],
    'xunfei_super_tts.ws_url': [{ required: true, message: t('enter_websocket_url'), trigger: 'blur' }],
    'xunfei_super_tts.voice': [{ required: true, message: t('enter_voice_timbre'), trigger: 'blur' }],
    'minimax.api_key': [{ required: true, message: t('enter_api_key'), trigger: 'blur' }],
    'qwen_tts.api_key': [{ required: true, message: t('enter_api_key'), trigger: 'blur' }]
  }

  // --- Computed URLs ---
  const finalOtaUrl = computed(() => {
    if (!otaForm.host?.trim()) return t('ota_domain_required')
    return `${otaForm.protocol === 'https' ? 'https' : 'http'}://${otaForm.host.trim()}:${otaForm.port}`
  })
  const finalWsUrl = computed(() => {
    if (!otaForm.host?.trim()) return t('ota_domain_required')
    return `${otaForm.protocol === 'https' ? 'wss' : 'ws'}://${otaForm.host.trim()}:${otaForm.port}/xiaozhi/v1/`
  })
  const finalMqttEndpoint = computed(() => {
    if (!otaForm.enableMqttUdp || !otaForm.host?.trim()) return ''
    return `${otaForm.host.trim()}:${otaForm.mqttServerPort}`
  })
  const finalUdpEndpoint = computed(() => {
    if (!otaForm.enableMqttUdp || !otaForm.host?.trim()) return ''
    return `${otaForm.host.trim()}:${otaForm.udpPort}`
  })

  function buildWsUrl() {
    if (!otaForm.host?.trim()) return ''
    return `${otaForm.protocol === 'https' ? 'wss' : 'ws'}://${otaForm.host.trim()}:${otaForm.port}/xiaozhi/v1/`
  }

  // --- MQTT/UDP save helpers ---
  const MQTT_SERVER_DEFAULT_USER = 'admin'
  const MQTT_SERVER_DEFAULT_PASS = 'admin123'

  async function saveMqttServerConfig() {
    const port = Number(otaForm.mqttServerPort) || 1883
    const useTls = port === 8883
    const configData = { enable: true, listen_host: '0.0.0.0', listen_port: port, username: MQTT_SERVER_DEFAULT_USER, password: MQTT_SERVER_DEFAULT_PASS, signature_key: otaForm.signature_key?.trim() || 'xiaozhi_ota_signature_key', enable_auth: true, tls: { enable: useTls, port: 8883, pem: '', key: '' } }
    const payload = { name: t('mqtt_server_config_label'), config_id: 'mqtt_server_mqtt_server_config', provider: 'mqtt_server', json_data: JSON.stringify(configData), enabled: true, is_default: true }
    const res = await api.get('/admin/mqtt-server-configs')
    const list = res.data?.data || []
    const existing = list.find(c => c.is_default) || list[0]
    if (existing?.id) await api.put(`/admin/mqtt-server-configs/${existing.id}`, payload)
    else await api.post('/admin/mqtt-server-configs', payload)
  }

  async function saveMqttConfig() {
    const host = otaForm.host?.trim() || '127.0.0.1'
    const port = Number(otaForm.mqttServerPort) || 1883
    const useTls = port === 8883
    const resGet = await api.get('/admin/mqtt-configs')
    const list = resGet.data?.data || []
    const existing = list.find(c => c.is_default) || list[0]
    let configData
    if (existing?.id) {
      const existingData = JSON.parse(existing.json_data || '{}')
      Object.assign(existingData, { enable: true, broker: host, type: useTls ? 'ssl' : 'tcp', port, client_id: existingData.client_id || 'xiaozhi_manager', username: MQTT_SERVER_DEFAULT_USER, password: MQTT_SERVER_DEFAULT_PASS })
      configData = existingData
    } else {
      configData = { enable: true, broker: host, type: useTls ? 'ssl' : 'tcp', port, client_id: 'xiaozhi_manager', username: MQTT_SERVER_DEFAULT_USER, password: MQTT_SERVER_DEFAULT_PASS }
    }
    const payload = { name: t('mqtt_config_label'), config_id: 'mqtt_wizard_default', is_default: true, json_data: JSON.stringify(configData) }
    if (existing?.id) await api.put(`/admin/mqtt-configs/${existing.id}`, payload)
    else await api.post('/admin/mqtt-configs', payload)
  }

  async function saveUdpConfig() {
    const host = otaForm.host?.trim() || '0.0.0.0'
    const port = Number(otaForm.udpPort) || 8990
    const configData = { listen_host: '0.0.0.0', listen_port: port, external_host: host, external_port: port }
    const payload = { name: t('udp_config_label'), config_id: 'udp_wizard_default', is_default: true, json_data: JSON.stringify(configData) }
    const res = await api.get('/admin/udp-configs')
    const list = res.data?.data || []
    const existing = list.find(c => c.is_default) || list[0]
    if (existing?.id) await api.put(`/admin/udp-configs/${existing.id}`, payload)
    else await api.post('/admin/udp-configs', payload)
  }

  // --- Save per step ---
  async function saveOta() {
    const wsUrl = buildWsUrl()
    if (!wsUrl) { ElMessage.warning(t('fill_domain_or_ip')); return false }
    if (otaForm.enableMqttUdp) {
      if (!otaForm.host?.trim()) { ElMessage.warning(t('fill_domain_or_ip')); return false }
      const mqttPort = Number(otaForm.mqttServerPort)
      const udpPort = Number(otaForm.udpPort)
      if (!mqttPort || mqttPort < 1 || mqttPort > 65535) { ElMessage.warning(t('enter_valid_mqtt_port')); return false }
      if (!udpPort || udpPort < 1 || udpPort > 65535) { ElMessage.warning(t('enter_valid_udp_port')); return false }
      try { await saveMqttServerConfig(); await saveMqttConfig(); await saveUdpConfig() }
      catch (e) { ElMessage.error(t('save_failed_mqtt_udp') + (e.response?.data?.message || e.message)); return false }
    }
    const mqttEndpoint = otaForm.enableMqttUdp ? finalMqttEndpoint.value : ''
    const payload = { name: t('ota_config_label'), config_id: 'ota_ota_config', provider: 'default', json_data: JSON.stringify({ signature_key: otaForm.signature_key?.trim() || 'xiaozhi_ota_signature_key', test: { websocket: { url: wsUrl }, mqtt: { enable: otaForm.enableMqttUdp, endpoint: mqttEndpoint } }, external: { websocket: { url: wsUrl }, mqtt: { enable: otaForm.enableMqttUdp, endpoint: mqttEndpoint } } }, null, 2), enabled: true, is_default: true }
    try {
      if (otaConfigId.value) { await api.put(`/admin/ota-configs/${otaConfigId.value}`, payload) }
      else { const res = await api.post('/admin/ota-configs', payload); otaConfigId.value = res.data?.data?.id ?? null }
      ElMessage.success(otaForm.enableMqttUdp ? t('ota_mqtt_udp_saved') : t('ota_config_saved'))
      return true
    } catch (e) { ElMessage.error(t('save_failed_ota') + (e.response?.data?.message || e.message)); return false }
  }

  async function saveVad() {
    if (!vadFormRef.value) return false
    try { await vadFormRef.value.validate() } catch (_) { return false }
    const payload = { name: vadForm.name, config_id: vadForm.config_id, provider: vadForm.provider, json_data: vadFormRef.value.getJsonData(), enabled: true, is_default: true }
    try {
      if (vadConfigId.value) { await api.put(`/admin/vad-configs/${vadConfigId.value}`, payload) }
      else { const res = await api.post('/admin/vad-configs', payload); vadConfigId.value = res.data?.data?.id ?? null }
      ElMessage.success(t('vad_config_saved')); return true
    } catch (e) { ElMessage.error(t('save_failed_vad') + (e.response?.data?.message || e.message)); return false }
  }

  async function saveAsr() {
    if (!asrFormRef.value) return false
    try { await asrFormRef.value.validate() } catch (_) { return false }
    const payload = { name: asrForm.name, config_id: asrForm.config_id, provider: asrForm.provider, json_data: asrFormRef.value.getJsonData(), enabled: true, is_default: true }
    try {
      if (asrConfigId.value) { await api.put(`/admin/asr-configs/${asrConfigId.value}`, payload) }
      else { const res = await api.post('/admin/asr-configs', payload); asrConfigId.value = res.data?.data?.id ?? null }
      ElMessage.success(t('asr_config_saved')); return true
    } catch (e) { ElMessage.error(t('save_failed_asr') + (e.response?.data?.message || e.message)); return false }
  }

  async function saveLlm() {
    if (!llmFormRef.value) return false
    try { await llmFormRef.value.validate() } catch (_) { return false }
    const payload = { name: llmForm.name, config_id: llmForm.config_id, provider: llmForm.provider, json_data: llmFormRef.value.getJsonData(), enabled: true, is_default: true }
    try {
      if (llmConfigId.value) { await api.put(`/admin/llm-configs/${llmConfigId.value}`, payload) }
      else { const res = await api.post('/admin/llm-configs', payload); llmConfigId.value = res.data?.data?.id ?? null }
      ElMessage.success(t('llm_config_saved')); return true
    } catch (e) { ElMessage.error(t('save_failed_llm') + (e.response?.data?.message || e.message)); return false }
  }

  async function saveTts() {
    if (!ttsFormRef.value) return false
    try { await ttsFormRef.value.validate() } catch (_) { return false }
    const payload = { name: ttsForm.name, config_id: ttsForm.config_id, provider: ttsForm.provider, json_data: ttsFormRef.value.getJsonData(), enabled: true, is_default: true }
    try {
      if (ttsConfigId.value) { await api.put(`/admin/tts-configs/${ttsConfigId.value}`, payload) }
      else { const res = await api.post('/admin/tts-configs', payload); ttsConfigId.value = res.data?.data?.id ?? null }
      ElMessage.success(t('tts_config_saved')); return true
    } catch (e) { ElMessage.error(t('save_failed_tts') + (e.response?.data?.message || e.message)); return false }
  }

  // --- Load existing configs ---
  async function loadOtaIfExists() {
    try {
      const res = await api.get('/admin/ota-configs')
      const list = res.data?.data || []
      const config = list.find(c => c.is_default) || list[0]
      if (!config) return
      otaConfigId.value = config.id
      const data = JSON.parse(config.json_data || '{}')
      if (data.signature_key) otaForm.signature_key = data.signature_key
      const ext = data.external?.websocket?.url || ''
      if (ext) {
        const m = ext.match(/^(wss?):\/\/([^:/]+):?(\d+)?/)
        if (m) { otaForm.protocol = m[1] === 'wss' ? 'https' : 'http'; otaForm.host = m[2]; otaForm.port = m[3] ? parseInt(m[3], 10) : 8989 }
      }
      const mqttEnabled = data.test?.mqtt?.enable || data.external?.mqtt?.enable
      otaForm.enableMqttUdp = !!mqttEnabled
      const endpoint = data.test?.mqtt?.endpoint || data.external?.mqtt?.endpoint || ''
      if (mqttEnabled && endpoint) { const parts = endpoint.split(':'); if (parts.length >= 2 && parts[1]) otaForm.mqttServerPort = parseInt(parts[1], 10) || 1883 }
    } catch (_) {}
  }

  async function loadVadIfExists() {
    try {
      const res = await api.get('/admin/vad-configs')
      const list = res.data?.data || []
      const config = list.find(c => c.is_default) || list[0]
      if (!config) return
      vadConfigId.value = config.id; vadForm.name = config.name; vadForm.config_id = config.config_id
      const data = JSON.parse(config.json_data || '{}')
      const provider = resolveVADProvider(config.provider, config.config_id, data)
      vadForm.provider = provider
      if (provider === 'webrtc_vad') Object.assign(vadForm.webrtc_vad, data.webrtc_vad || data)
      else if (provider === 'silero_vad') Object.assign(vadForm.silero_vad, data.silero_vad || data)
      else Object.assign(vadForm.ten_vad, data.ten_vad || data)
    } catch (_) {}
  }

  async function loadAsrIfExists() {
    try {
      const res = await api.get('/admin/asr-configs')
      const list = res.data?.data || []
      const config = list.find(c => c.is_default) || list[0]
      if (!config) return
      asrConfigId.value = config.id; asrForm.name = config.name; asrForm.config_id = config.config_id
      const data = JSON.parse(config.json_data || '{}')
      const provider = resolveASRProvider(config.provider, config.config_id, data)
      asrForm.provider = provider
      if (provider === 'doubao') Object.assign(asrForm.doubao, data.doubao || data)
      else if (provider === 'aliyun_funasr') Object.assign(asrForm.aliyun_funasr, data.aliyun_funasr || data)
      else if (provider === 'aliyun_qwen3') Object.assign(asrForm.aliyun_qwen3, data.aliyun_qwen3 || data)
      else if (provider === 'xunfei') Object.assign(asrForm.xunfei, data.xunfei || data)
      else {
        const obj = data.funasr || data; const funasr = { ...asrForm.funasr }
        if (typeof obj.chunk_size === 'number') funasr.chunk_size = [5, 10, 5]
        else if (Array.isArray(obj.chunk_size) && obj.chunk_size.length === 3) funasr.chunk_size = [...obj.chunk_size]
        Object.assign(funasr, obj); asrForm.funasr = funasr
      }
    } catch (_) {}
  }

  async function loadLlmIfExists() {
    try {
      const res = await api.get('/admin/llm-configs')
      const list = res.data?.data || []
      const config = list.find(c => c.is_default) || list[0]
      if (!config) return
      llmConfigId.value = config.id; llmForm.name = config.name; llmForm.config_id = config.config_id
      const data = JSON.parse(config.json_data || '{}')
      llmForm.provider = getResolvedLLMProvider(config.provider, data.type)
      llmForm.type = getResolvedLLMType(config.provider, data.type)
      if (data.model_name !== undefined) llmForm.model_name = data.model_name
      if (data.api_key !== undefined) llmForm.api_key = data.api_key
      if (data.base_url !== undefined) llmForm.base_url = data.base_url
      if (data.max_tokens !== undefined) llmForm.max_tokens = data.max_tokens
      if (data.temperature !== undefined) llmForm.temperature = data.temperature
      if (data.top_p !== undefined) llmForm.top_p = data.top_p
      if (data.thinking?.mode !== undefined) llmForm.thinking_mode = data.thinking.mode
      if (data.thinking?.budget_tokens !== undefined) llmForm.thinking_budget_tokens = Number(data.thinking.budget_tokens) || null
      if (data.thinking?.effort !== undefined) llmForm.thinking_effort = data.thinking.effort
      if (data.thinking?.clear_thinking !== undefined) llmForm.thinking_clear_thinking = data.thinking.clear_thinking
    } catch (_) {}
  }

  async function loadTtsIfExists() {
    try {
      const res = await api.get('/admin/tts-configs')
      const list = res.data?.data || []
      const config = list.find(c => c.is_default) || list[0]
      if (!config) return
      ttsConfigId.value = config.id; ttsForm.name = config.name; ttsForm.config_id = config.config_id
      const data = JSON.parse(config.json_data || '{}')
      const p = resolveTTSProvider(config.provider, config.config_id, data)
      ttsForm.provider = p
      if (p === 'doubao_ws') {
        Object.assign(ttsForm.doubao_ws, data)
        if (!String(ttsForm.doubao_ws.ws_url || '').trim()) ttsForm.doubao_ws.ws_url = data.ws_host ? `wss://${data.ws_host}/api/v3/tts/unidirectional/stream` : 'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream'
        if (!String(ttsForm.doubao_ws.resource_id || '').trim()) ttsForm.doubao_ws.resource_id = data.resource_id || ''
      } else if (p === 'edge') Object.assign(ttsForm.edge, data)
      else if (p === 'edge_offline') Object.assign(ttsForm.edge_offline, data)
      else if (p === 'aliyun_qwen') Object.assign(ttsForm.qwen_tts, data)
      else if (p === 'openai') Object.assign(ttsForm.openai, data)
      else if (p === 'xunfei') Object.assign(ttsForm.xunfei, data)
      else if (p === 'xunfei_super_tts') Object.assign(ttsForm.xunfei_super_tts, data)
      else if (p === 'zhipu') Object.assign(ttsForm.zhipu, data)
      else if (p === 'minimax') Object.assign(ttsForm.minimax, data)
    } catch (_) {}
  }

  // --- Navigation ---
  async function saveAndNext() {
    saving.value = true
    let ok = false
    try {
      if (currentStep.value === 0) ok = await saveOta()
      else if (currentStep.value === 1) ok = await saveVad()
      else if (currentStep.value === 2) ok = await saveAsr()
      else if (currentStep.value === 3) ok = await saveLlm()
      else if (currentStep.value === 4) ok = await saveTts()
      if (ok) currentStep.value = currentStep.value === 4 ? 5 : currentStep.value + 1
    } finally { saving.value = false }
  }

  function skipStep() { currentStep.value = currentStep.value === 4 ? 5 : currentStep.value + 1 }
  function prevStep() { if (currentStep.value > 0) currentStep.value-- }

  // --- Test helpers ---
  function formatTestMessage(result) {
    const base = result.message || ''
    const suffix = []
    if (result.first_packet_ms != null) suffix.push(`${result.first_packet_ms}ms`)
    if (result.reasoning_content_returned) suffix.push(t('upstream_thinking_detected'))
    return suffix.length ? `${base} ${suffix.join(' · ')}` : base
  }

  function formatDraftTestLabel(name, configId) {
    return name?.trim() || configId?.trim() || t('current_config')
  }

  async function testCurrentStepConfig() {
    const step = currentStep.value
    const testMap = {
      1: { formRef: vadFormRef, form: vadForm, type: 'vad', buildPayload: () => ({ name: vadForm.name, config_id: vadForm.config_id, provider: vadForm.provider, is_default: vadForm.is_default, ...parseJsonData(vadFormRef.value.getJsonData()) }) },
      2: { formRef: asrFormRef, form: asrForm, type: 'asr', buildPayload: () => ({ name: asrForm.name, config_id: asrForm.config_id, provider: asrForm.provider, is_default: asrForm.is_default, ...parseJsonData(asrFormRef.value.getJsonData()) }) },
      3: { formRef: llmFormRef, form: llmForm, type: 'llm', buildPayload: () => ({ name: llmForm.name, config_id: llmForm.config_id, provider: llmForm.provider, is_default: llmForm.is_default, ...parseJsonData(llmFormRef.value.getJsonData()) }) },
      4: { formRef: ttsFormRef, form: ttsForm, type: 'tts', buildPayload: () => ({ name: ttsForm.name, config_id: ttsForm.config_id, provider: ttsForm.provider, is_default: ttsForm.is_default, ...parseJsonData(ttsFormRef.value.getJsonData()) }) }
    }
    const entry = testMap[step]
    if (!entry || !entry.formRef.value) return
    try { await entry.formRef.value.validate() } catch (_) { return }
    const configId = entry.form.config_id?.trim() || `${entry.type}_wizard`
    const payload = entry.buildPayload()
    testingStep.value = true
    try {
      const testPayload = entry.type === 'llm' ? { provider: configId, [configId]: payload } : { [configId]: payload }
      const result = await testWithData(entry.type, testPayload)
      const label = formatDraftTestLabel(entry.form.name, configId)
      if (result.ok) ElMessage.success(`${label}：${formatTestMessage(result) || t('test_passed')}`)
      else ElMessage.warning(`${label}：${result.message || t('test_not_passed')}`)
    } catch (err) {
      ElMessage.warning(err.response?.data?.error || t('test_request_failed_v2'))
    } finally { testingStep.value = false }
  }

  // --- OTA test ---
  function formatOtaResponseDisplay(str) {
    if (str == null || str === '') return ''
    const s = String(str).trim()
    if (!s) return ''
    try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
  }

  async function runOtaTest() {
    otaTestLoading.value = true; otaTestResult.value = null
    try {
      const res = await api.post('/admin/configs/test', { types: ['ota'] }, { timeout: 30000 })
      const data = res.data?.data ?? res.data
      const ota = data?.ota
      if (ota && typeof ota === 'object') {
        const entry = Object.entries(ota).find(([k]) => !k.startsWith('_'))
        if (entry) {
          const [, v] = entry
          let displayText = ''
          if (v.websocket) { const ws = v.websocket; displayText += `WebSocket: ${ws.ok ? '✓' : '✗'} ${ws.message}${ws.first_packet_ms != null ? ` (${ws.first_packet_ms}ms)` : ''}\n` }
          if (v.mqtt_udp) { const mqtt = v.mqtt_udp; displayText += `MQTT UDP: ${mqtt.ok ? '✓' : '✗'} ${mqtt.message}${mqtt.first_packet_ms != null ? ` (${mqtt.first_packet_ms}ms)` : ''}\n` }
          if (v.ota_response !== undefined && v.ota_response !== '') displayText += `\n--- ${t('ota_return_label')} ---\n${formatOtaResponseDisplay(v.ota_response)}`
          otaTestResult.value = displayText.trim() || t('detail_not_available')
          v.ok ? ElMessage.success(v.message || t('ota_test_passed')) : ElMessage.warning(v.message || t('ota_test_failed'))
        } else { otaTestResult.value = t('ota_test_no_result') }
      } else { otaTestResult.value = typeof data === 'string' ? data : JSON.stringify(data || {}, null, 2) }
    } catch (e) {
      const errorMsg = (e.response?.data && typeof e.response.data === 'object') ? JSON.stringify(e.response.data, null, 2) : (e.response?.data?.message || e.message || t('request_failed'))
      otaTestResult.value = errorMsg; ElMessage.error(t('ota_test_request_failed'))
    } finally { otaTestLoading.value = false }
  }

  // --- TTS voice list ---
  async function loadTtsVoiceOptions(provider) {
    if (!provider) { voiceOptions.value = []; return }
    const providersWithVoices = ['minimax', 'edge', 'doubao', 'doubao_ws', 'zhipu', 'openai', 'xunfei_super_tts']
    if (!providersWithVoices.includes(provider)) { voiceOptions.value = []; return }
    voiceLoading.value = true
    try {
      const response = await api.get('/user/voice-options', { params: { provider } })
      voiceOptions.value = response.data.data || []
    } catch (error) { console.error(t('load_voice_list_failed_c'), error); voiceOptions.value = [] }
    finally { voiceLoading.value = false }
  }

  function handleTtsVoiceOptionsRequest(provider) { loadTtsVoiceOptions(provider || ttsForm.provider) }

  async function copyToClipboard(text) {
    try { await navigator.clipboard.writeText(text); ElMessage.success(t('copied_to_clipboard')) }
    catch { ElMessage.error(t('copy_failed')) }
  }

  // --- Load all on mount ---
  async function initialize() {
    await loadOtaIfExists(); await loadVadIfExists(); await loadAsrIfExists(); await loadLlmIfExists(); await loadTtsIfExists()
  }

  // Watch TTS step to load voices
  watch(currentStep, (step) => { if (step === 4 && ttsForm.provider) nextTick(() => loadTtsVoiceOptions(ttsForm.provider)) }, { immediate: true })
  watch(() => ttsForm.provider, (provider) => { if (currentStep.value === 4 && provider) loadTtsVoiceOptions(provider) })

  return {
    currentStep, saving, testingStep, otaTestLoading, otaTestResult,
    otaForm, vadForm, vadFormRef, vadFormRules, asrForm, asrFormRef, asrFormRules,
    llmForm, llmFormRef, llmFormRules, ttsForm, ttsFormRef, ttsFormRules,
    voiceOptions, voiceLoading,
    finalOtaUrl, finalWsUrl, finalMqttEndpoint, finalUdpEndpoint,
    saveAndNext, skipStep, prevStep, testCurrentStepConfig, runOtaTest,
    handleTtsVoiceOptionsRequest, copyToClipboard, initialize
  }
}
