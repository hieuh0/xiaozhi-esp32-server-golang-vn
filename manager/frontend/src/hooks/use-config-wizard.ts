import { useState } from 'react'
import { toast } from 'sonner'
import api from '@/utils/api'

export interface OtaForm {
  host: string; port: number; protocol: 'http' | 'https'
  signature_key: string; enableMqttUdp: boolean; mqttServerPort: number; udpPort: number
}

export interface GenericConfigForm {
  name: string; config_id: string; provider: string; json_data: string; _id: number | null
}

const defGeneric = (name: string, config_id: string, provider: string): GenericConfigForm =>
  ({ name, config_id, provider, json_data: '{}', _id: null })

type ConfigItem = { id: number; name: string; config_id: string; provider: string; json_data: string; is_default: boolean }

function loadIntoGeneric(data: { data?: ConfigItem[] } | undefined, setter: (p: Partial<GenericConfigForm>) => void) {
  const list = data?.data || []
  const config = list.find((c) => c.is_default) || list[0]
  if (config) setter({ name: config.name, config_id: config.config_id, provider: config.provider, json_data: config.json_data || '{}', _id: config.id })
}

export function useConfigWizard() {
  const [currentStep, setCurrentStep] = useState(0)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [otaTestResult, setOtaTestResult] = useState<string | null>(null)

  const [otaForm, setOtaFormRaw] = useState<OtaForm>({ host: '', port: 8989, protocol: 'http', signature_key: 'xiaozhi_ota_signature_key', enableMqttUdp: false, mqttServerPort: 1883, udpPort: 8990 })
  const [otaConfigId, setOtaConfigId] = useState<number | null>(null)
  const [vadForm, setVadFormRaw] = useState<GenericConfigForm>(defGeneric('Default VAD', 'ten_vad_default', 'ten_vad'))
  const [asrForm, setAsrFormRaw] = useState<GenericConfigForm>(defGeneric('FunASR ASR', 'funasr_default', 'funasr'))
  const [llmForm, setLlmFormRaw] = useState<GenericConfigForm>(defGeneric('Default LLM', 'openai_default', 'openai'))
  const [ttsForm, setTtsFormRaw] = useState<GenericConfigForm>(defGeneric('Default TTS', 'minimax_default', 'minimax'))

  const setOta = (p: Partial<OtaForm>) => setOtaFormRaw(f => ({ ...f, ...p }))
  const setVad = (p: Partial<GenericConfigForm>) => setVadFormRaw(f => ({ ...f, ...p }))
  const setAsr = (p: Partial<GenericConfigForm>) => setAsrFormRaw(f => ({ ...f, ...p }))
  const setLlm = (p: Partial<GenericConfigForm>) => setLlmFormRaw(f => ({ ...f, ...p }))
  const setTts = (p: Partial<GenericConfigForm>) => setTtsFormRaw(f => ({ ...f, ...p }))

  const host = otaForm.host.trim()
  const proto = otaForm.protocol
  const finalOtaUrl = host ? `${proto}://${host}:${otaForm.port}` : ''
  const finalWsUrl = host ? `${proto === 'https' ? 'wss' : 'ws'}://${host}:${otaForm.port}/xiaozhi/v1/` : ''
  const finalMqttEndpoint = (otaForm.enableMqttUdp && host) ? `${host}:${otaForm.mqttServerPort}` : ''
  const finalUdpEndpoint = (otaForm.enableMqttUdp && host) ? `${host}:${otaForm.udpPort}` : ''

  async function initialize() {
    try {
      const [otaR, vadR, asrR, llmR, ttsR] = await Promise.allSettled([
        api.get('/admin/ota-configs'), api.get('/admin/vad-configs'), api.get('/admin/asr-configs'),
        api.get('/admin/llm-configs'), api.get('/admin/tts-configs'),
      ])
      if (otaR.status === 'fulfilled') {
        const list: ConfigItem[] = otaR.value.data?.data || []
        const c = list.find(x => x.is_default) || list[0]
        if (c) {
          setOtaConfigId(c.id)
          const d = JSON.parse(c.json_data || '{}')
          const wsUrl: string = d.external?.websocket?.url || ''
          const m = wsUrl.match(/^(wss?):\/\/([^:/]+):?(\d+)?/)
          if (m) setOtaFormRaw(f => ({ ...f, protocol: m[1] === 'wss' ? 'https' : 'http', host: m[2], port: m[3] ? parseInt(m[3]) : 8989, signature_key: d.signature_key || f.signature_key, enableMqttUdp: !!(d.test?.mqtt?.enable || d.external?.mqtt?.enable) }))
        }
      }
      if (vadR.status === 'fulfilled') loadIntoGeneric(vadR.value.data, setVad)
      if (asrR.status === 'fulfilled') loadIntoGeneric(asrR.value.data, setAsr)
      if (llmR.status === 'fulfilled') loadIntoGeneric(llmR.value.data, setLlm)
      if (ttsR.status === 'fulfilled') loadIntoGeneric(ttsR.value.data, setTts)
    } catch (_) {}
  }

  async function saveOta(): Promise<boolean> {
    if (!host) { toast.warning('Enter domain or IP address'); return false }
    if (otaForm.enableMqttUdp) {
      try {
        const p = otaForm.mqttServerPort; const useTls = p === 8883
        const upsert = async (endpoint: string, pl: object) => {
          const r = await api.get(endpoint); const lst: ConfigItem[] = r.data?.data || []
          const ex = lst.find(c => c.is_default) || lst[0]
          if (ex?.id) await api.put(`${endpoint}/${ex.id}`, pl); else await api.post(endpoint, pl)
        }
        await upsert('/admin/mqtt-server-configs', { name: 'MQTT Server Config', config_id: 'mqtt_server_mqtt_server_config', provider: 'mqtt_server', enabled: true, is_default: true, json_data: JSON.stringify({ enable: true, listen_host: '0.0.0.0', listen_port: p, username: 'admin', password: 'admin123', signature_key: otaForm.signature_key || 'xiaozhi_ota_signature_key', enable_auth: true, tls: { enable: useTls, port: 8883, pem: '', key: '' } }) })
        await upsert('/admin/mqtt-configs', { name: 'MQTT Config', config_id: 'mqtt_wizard_default', is_default: true, json_data: JSON.stringify({ enable: true, broker: host, type: useTls ? 'ssl' : 'tcp', port: p, client_id: 'xiaozhi_manager', username: 'admin', password: 'admin123' }) })
        const udpP = otaForm.udpPort
        await upsert('/admin/udp-configs', { name: 'UDP Config', config_id: 'udp_wizard_default', is_default: true, json_data: JSON.stringify({ listen_host: '0.0.0.0', listen_port: udpP, external_host: host, external_port: udpP }) })
      } catch (e) { toast.error('MQTT/UDP save failed: ' + (e as Error).message); return false }
    }
    const mqttEndpoint = otaForm.enableMqttUdp ? finalMqttEndpoint : ''
    const payload = { name: 'OTA Config', config_id: 'ota_ota_config', provider: 'default', enabled: true, is_default: true, json_data: JSON.stringify({ signature_key: otaForm.signature_key || 'xiaozhi_ota_signature_key', test: { websocket: { url: finalWsUrl }, mqtt: { enable: otaForm.enableMqttUdp, endpoint: mqttEndpoint } }, external: { websocket: { url: finalWsUrl }, mqtt: { enable: otaForm.enableMqttUdp, endpoint: mqttEndpoint } } }, null, 2) }
    try {
      if (otaConfigId) await api.put(`/admin/ota-configs/${otaConfigId}`, payload)
      else { const r = await api.post('/admin/ota-configs', payload); setOtaConfigId(r.data?.data?.id ?? null) }
      toast.success('OTA config saved'); return true
    } catch (e) { toast.error('OTA save failed: ' + (e as Error).message); return false }
  }

  async function saveGeneric(endpoint: string, form: GenericConfigForm, setter: (p: Partial<GenericConfigForm>) => void, label: string): Promise<boolean> {
    if (!form.name.trim()) { toast.error('Enter config name'); return false }
    const payload = { name: form.name, config_id: form.config_id, provider: form.provider, json_data: form.json_data, enabled: true, is_default: true }
    try {
      if (form._id) await api.put(`${endpoint}/${form._id}`, payload)
      else { const r = await api.post(endpoint, payload); setter({ _id: r.data?.data?.id ?? null }) }
      toast.success(`${label} config saved`); return true
    } catch (e) { toast.error(`${label} save failed: ` + (e as Error).message); return false }
  }

  async function saveAndNext() {
    setSaving(true)
    try {
      let ok = false
      if (currentStep === 0) ok = await saveOta()
      else if (currentStep === 1) ok = await saveGeneric('/admin/vad-configs', vadForm, setVad, 'VAD')
      else if (currentStep === 2) ok = await saveGeneric('/admin/asr-configs', asrForm, setAsr, 'ASR')
      else if (currentStep === 3) ok = await saveGeneric('/admin/llm-configs', llmForm, setLlm, 'LLM')
      else if (currentStep === 4) ok = await saveGeneric('/admin/tts-configs', ttsForm, setTts, 'TTS')
      if (ok) setCurrentStep(s => Math.min(s + 1, 5))
    } finally { setSaving(false) }
  }

  function skipStep() { setCurrentStep(s => Math.min(s + 1, 5)) }
  function prevStep() { setCurrentStep(s => Math.max(s - 1, 0)) }

  async function runOtaTest() {
    setTesting(true); setOtaTestResult(null)
    try {
      const res = await api.post('/admin/configs/test', { types: ['ota'] }, { timeout: 30000 })
      const data = res.data?.data ?? res.data
      const ota = data?.ota
      if (ota && typeof ota === 'object') {
        const entry = Object.entries(ota).find(([k]) => !k.startsWith('_'))
        if (entry) {
          const v = entry[1] as { ok: boolean; message?: string; websocket?: { ok: boolean; message: string; first_packet_ms?: number }; mqtt_udp?: { ok: boolean; message: string }; ota_response?: string }
          let text = ''
          if (v.websocket) text += `WebSocket: ${v.websocket.ok ? '✓' : '✗'} ${v.websocket.message}${v.websocket.first_packet_ms != null ? ` (${v.websocket.first_packet_ms}ms)` : ''}\n`
          if (v.mqtt_udp) text += `MQTT UDP: ${v.mqtt_udp.ok ? '✓' : '✗'} ${v.mqtt_udp.message}\n`
          if (v.ota_response) text += `\n--- OTA Response ---\n${v.ota_response}`
          setOtaTestResult(text.trim() || 'No details available')
          v.ok ? toast.success(v.message || 'OTA test passed') : toast.warning(v.message || 'OTA test failed')
        }
      } else { setOtaTestResult(typeof data === 'string' ? data : JSON.stringify(data || {}, null, 2)) }
    } catch (e) { setOtaTestResult((e as Error).message || 'Request failed'); toast.error('OTA test failed') }
    finally { setTesting(false) }
  }

  async function copyToClipboard(text: string) {
    try { await navigator.clipboard.writeText(text); toast.success('Copied') }
    catch { toast.error('Copy failed') }
  }

  return {
    currentStep, saving, testing, otaTestResult,
    otaForm, setOta, vadForm, setVad, asrForm, setAsr, llmForm, setLlm, ttsForm, setTts,
    finalOtaUrl, finalWsUrl, finalMqttEndpoint, finalUdpEndpoint,
    initialize, saveAndNext, skipStep, prevStep, runOtaTest, copyToClipboard,
  }
}
