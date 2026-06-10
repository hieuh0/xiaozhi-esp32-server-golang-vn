import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { ConfigRow, ConfigForm } from './config-list-page'
import { getTTSProviderOptions, TTS_PROVIDERS_WITH_VOICES } from './tts-provider-options'
import { TtsProviderFields } from './tts-provider-fields'

export interface TtsFields {
  provider: string; name: string; config_id: string; enabled: boolean; is_default: boolean
  doubao_ws_appid: string; doubao_ws_token: string; doubao_ws_model: string; doubao_ws_resource_id: string; doubao_ws_voice: string; doubao_ws_url: string
  edge_voice: string; edge_rate: string; edge_volume: string; edge_pitch: string; edge_connect_timeout: number; edge_receive_timeout: number
  edge_offline_server_url: string; edge_offline_timeout: number; edge_offline_sample_rate: number; edge_offline_channels: number; edge_offline_frame_duration: number
  qwen_api_key: string; qwen_region: string; qwen_model: string; qwen_voice: string; qwen_language_type: string; qwen_stream: boolean; qwen_frame_duration: number
  zhipu_api_key: string; zhipu_api_url: string; zhipu_model: string; zhipu_voice: string; zhipu_response_format: string; zhipu_volume: number; zhipu_speed: number; zhipu_stream: boolean; zhipu_encode_format: string; zhipu_frame_duration: number
  minimax_api_key: string; minimax_model: string; minimax_voice: string; minimax_speed: number; minimax_vol: number; minimax_pitch: number; minimax_sample_rate: number; minimax_bitrate: number; minimax_format: string; minimax_channel: number
  openai_api_key: string; openai_api_url: string; openai_model: string; openai_voice: string; openai_response_format: string; openai_speed: number; openai_stream: boolean; openai_frame_duration: number
  xunfei_app_id: string; xunfei_api_key: string; xunfei_api_secret: string; xunfei_ws_url: string; xunfei_voice: string; xunfei_audio_encoding: string; xunfei_sample_rate: number; xunfei_speed: number; xunfei_volume: number; xunfei_pitch: number; xunfei_connect_timeout: number; xunfei_read_timeout: number; xunfei_frame_duration: number
  xunfei_super_app_id: string; xunfei_super_api_key: string; xunfei_super_api_secret: string; xunfei_super_ws_url: string; xunfei_super_voice: string; xunfei_super_audio_encoding: string; xunfei_super_sample_rate: number; xunfei_super_speed: number; xunfei_super_volume: number; xunfei_super_pitch: number; xunfei_super_connect_timeout: number; xunfei_super_read_timeout: number; xunfei_super_frame_duration: number
  indextts_api_url: string; indextts_api_key: string; indextts_model: string; indextts_voice: string; indextts_frame_duration: number
  cosyvoice_api_url: string; cosyvoice_spk_id: string; cosyvoice_frame_duration: number; cosyvoice_target_sr: number; cosyvoice_audio_format: string; cosyvoice_instruct_text: string
  supertonic_onnx_dir: string; supertonic_voice: string; supertonic_voice_json_path: string
  supertonic_lang: string; supertonic_steps: number; supertonic_speed: number
  supertonic_silence: number; supertonic_frame_duration: number
}

const D: TtsFields = {
  provider: 'edge', name: '', config_id: '', enabled: true, is_default: false,
  doubao_ws_appid:'', doubao_ws_token:'', doubao_ws_model:'seed-tts-2.0-standard', doubao_ws_resource_id:'', doubao_ws_voice:'', doubao_ws_url:'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream',
  edge_voice:'', edge_rate:'', edge_volume:'', edge_pitch:'', edge_connect_timeout:10, edge_receive_timeout:60,
  edge_offline_server_url:'', edge_offline_timeout:30, edge_offline_sample_rate:16000, edge_offline_channels:1, edge_offline_frame_duration:20,
  qwen_api_key:'', qwen_region:'beijing', qwen_model:'qwen3-tts-flash', qwen_voice:'Cherry', qwen_language_type:'Auto', qwen_stream:false, qwen_frame_duration:60,
  zhipu_api_key:'', zhipu_api_url:'https://open.bigmodel.cn/api/paas/v4/audio/speech', zhipu_model:'glm-tts', zhipu_voice:'', zhipu_response_format:'wav', zhipu_volume:1, zhipu_speed:1, zhipu_stream:false, zhipu_encode_format:'base64', zhipu_frame_duration:60,
  minimax_api_key:'', minimax_model:'speech-2.8-hd', minimax_voice:'male-qn-qingse', minimax_speed:1, minimax_vol:1, minimax_pitch:0, minimax_sample_rate:32000, minimax_bitrate:128000, minimax_format:'mp3', minimax_channel:1,
  openai_api_key:'', openai_api_url:'', openai_model:'', openai_voice:'', openai_response_format:'mp3', openai_speed:1, openai_stream:false, openai_frame_duration:60,
  xunfei_app_id:'', xunfei_api_key:'', xunfei_api_secret:'', xunfei_ws_url:'wss://tts-api.xfyun.cn/v2/tts', xunfei_voice:'', xunfei_audio_encoding:'raw', xunfei_sample_rate:16000, xunfei_speed:50, xunfei_volume:50, xunfei_pitch:50, xunfei_connect_timeout:10, xunfei_read_timeout:30, xunfei_frame_duration:60,
  xunfei_super_app_id:'', xunfei_super_api_key:'', xunfei_super_api_secret:'', xunfei_super_ws_url:'wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6', xunfei_super_voice:'', xunfei_super_audio_encoding:'raw', xunfei_super_sample_rate:24000, xunfei_super_speed:50, xunfei_super_volume:50, xunfei_super_pitch:50, xunfei_super_connect_timeout:10, xunfei_super_read_timeout:30, xunfei_super_frame_duration:60,
  indextts_api_url:'', indextts_api_key:'', indextts_model:'indextts-vllm', indextts_voice:'', indextts_frame_duration:60,
  cosyvoice_api_url:'', cosyvoice_spk_id:'', cosyvoice_frame_duration:60, cosyvoice_target_sr:22050, cosyvoice_audio_format:'wav', cosyvoice_instruct_text:'',
  supertonic_onnx_dir:'', supertonic_voice:'M1', supertonic_voice_json_path:'',
  supertonic_lang:'na', supertonic_steps:8, supertonic_speed:1.0,
  supertonic_silence:0.3, supertonic_frame_duration:60,
}

function serialize(f: TtsFields): string {
  const p = f.provider
  let cfg: Record<string, unknown> = {}
  if (p==='doubao_ws') cfg={appid:f.doubao_ws_appid,access_token:f.doubao_ws_token,model:f.doubao_ws_model,resource_id:f.doubao_ws_resource_id,voice:f.doubao_ws_voice,ws_url:f.doubao_ws_url}
  else if (p==='edge') cfg={voice:f.edge_voice,rate:f.edge_rate,volume:f.edge_volume,pitch:f.edge_pitch,connect_timeout:f.edge_connect_timeout,receive_timeout:f.edge_receive_timeout}
  else if (p==='edge_offline') cfg={server_url:f.edge_offline_server_url,timeout:f.edge_offline_timeout,sample_rate:f.edge_offline_sample_rate,channels:f.edge_offline_channels,frame_duration:f.edge_offline_frame_duration}
  else if (p==='aliyun_qwen') cfg={provider:'aliyun_qwen',api_key:f.qwen_api_key,region:f.qwen_region,model:f.qwen_model,voice:f.qwen_voice,language_type:f.qwen_language_type,stream:f.qwen_stream,frame_duration:f.qwen_frame_duration}
  else if (p==='zhipu') cfg={provider:'zhipu',api_key:f.zhipu_api_key,api_url:f.zhipu_api_url,model:f.zhipu_model,voice:f.zhipu_voice,response_format:f.zhipu_response_format,volume:f.zhipu_volume,speed:f.zhipu_speed,stream:f.zhipu_stream,encode_format:f.zhipu_encode_format,frame_duration:f.zhipu_frame_duration}
  else if (p==='minimax') cfg={provider:'minimax',api_key:f.minimax_api_key,model:f.minimax_model,voice:f.minimax_voice,speed:f.minimax_speed,vol:f.minimax_vol,pitch:f.minimax_pitch,sample_rate:f.minimax_sample_rate,bitrate:f.minimax_bitrate,format:f.minimax_format,channel:f.minimax_channel}
  else if (p==='openai') cfg={api_key:f.openai_api_key,api_url:f.openai_api_url,model:f.openai_model,voice:f.openai_voice,response_format:f.openai_response_format,speed:f.openai_speed,stream:f.openai_stream,frame_duration:f.openai_frame_duration}
  else if (p==='xunfei') cfg={provider:'xunfei',app_id:f.xunfei_app_id,api_key:f.xunfei_api_key,api_secret:f.xunfei_api_secret,ws_url:f.xunfei_ws_url,voice:f.xunfei_voice,audio_encoding:f.xunfei_audio_encoding,sample_rate:f.xunfei_sample_rate,speed:f.xunfei_speed,volume:f.xunfei_volume,pitch:f.xunfei_pitch,connect_timeout:f.xunfei_connect_timeout,read_timeout:f.xunfei_read_timeout,frame_duration:f.xunfei_frame_duration}
  else if (p==='xunfei_super_tts') cfg={provider:'xunfei_super_tts',double_stream:true,app_id:f.xunfei_super_app_id,api_key:f.xunfei_super_api_key,api_secret:f.xunfei_super_api_secret,ws_url:f.xunfei_super_ws_url,voice:f.xunfei_super_voice,audio_encoding:f.xunfei_super_audio_encoding,sample_rate:f.xunfei_super_sample_rate,speed:f.xunfei_super_speed,volume:f.xunfei_super_volume,pitch:f.xunfei_super_pitch,connect_timeout:f.xunfei_super_connect_timeout,read_timeout:f.xunfei_super_read_timeout,frame_duration:f.xunfei_super_frame_duration}
  else if (p==='indextts_vllm') cfg={provider:'indextts_vllm',api_url:f.indextts_api_url,api_key:f.indextts_api_key,model:f.indextts_model,voice:f.indextts_voice,response_format:'wav',stream:false,frame_duration:f.indextts_frame_duration}
  else if (p==='cosyvoice') cfg={api_url:f.cosyvoice_api_url,spk_id:f.cosyvoice_spk_id,frame_duration:f.cosyvoice_frame_duration,target_sr:f.cosyvoice_target_sr,audio_format:f.cosyvoice_audio_format,instruct_text:f.cosyvoice_instruct_text}
  else if (p==='supertonic') cfg={
    onnx_dir:f.supertonic_onnx_dir,
    voice:f.supertonic_voice,
    voice_json_path:f.supertonic_voice_json_path,
    lang:f.supertonic_lang,
    steps:f.supertonic_steps,
    speed:f.supertonic_speed,
    silence_duration:f.supertonic_silence,
    frame_duration:f.supertonic_frame_duration,
  }
  return JSON.stringify(cfg)
}

function parse(row: ConfigRow | null): TtsFields {
  if (!row) return { ...D }
  try {
    const d = JSON.parse(row.json_data || '{}')
    const prov = row.provider || 'edge'
    return {
      ...D, provider: prov, name: row.name, config_id: row.config_id, enabled: row.enabled !== false, is_default: !!row.is_default,
      doubao_ws_appid: d.appid||'', doubao_ws_token: d.access_token||'', doubao_ws_model: d.model||'seed-tts-2.0-standard', doubao_ws_resource_id: d.resource_id||'', doubao_ws_voice: d.voice||'', doubao_ws_url: d.ws_url||D.doubao_ws_url,
      edge_voice: d.voice||'', edge_rate: d.rate||'', edge_volume: d.volume||'', edge_pitch: d.pitch||'', edge_connect_timeout: d.connect_timeout||10, edge_receive_timeout: d.receive_timeout||60,
      edge_offline_server_url: d.server_url||'', edge_offline_timeout: d.timeout||30, edge_offline_sample_rate: d.sample_rate||16000, edge_offline_channels: d.channels||1, edge_offline_frame_duration: d.frame_duration||20,
      qwen_api_key: d.api_key||'', qwen_region: d.region||'beijing', qwen_model: d.model||'qwen3-tts-flash', qwen_voice: d.voice||'Cherry', qwen_language_type: d.language_type||'Auto', qwen_stream: !!d.stream, qwen_frame_duration: d.frame_duration||60,
      zhipu_api_key: d.api_key||'', zhipu_api_url: d.api_url||D.zhipu_api_url, zhipu_model: d.model||'glm-tts', zhipu_voice: d.voice||'', zhipu_response_format: d.response_format||'wav', zhipu_volume: d.volume??1, zhipu_speed: d.speed??1, zhipu_stream: !!d.stream, zhipu_encode_format: d.encode_format||'base64', zhipu_frame_duration: d.frame_duration||60,
      minimax_api_key: d.api_key||'', minimax_model: d.model||'speech-2.8-hd', minimax_voice: d.voice||'', minimax_speed: d.speed||1, minimax_vol: d.vol||1, minimax_pitch: d.pitch||0, minimax_sample_rate: d.sample_rate||32000, minimax_bitrate: d.bitrate||128000, minimax_format: d.format||'mp3', minimax_channel: d.channel||1,
      openai_api_key: d.api_key||'', openai_api_url: d.api_url||'', openai_model: d.model||'', openai_voice: d.voice||'', openai_response_format: d.response_format||'mp3', openai_speed: d.speed||1, openai_stream: !!d.stream, openai_frame_duration: d.frame_duration||60,
      xunfei_app_id: d.app_id||'', xunfei_api_key: d.api_key||'', xunfei_api_secret: d.api_secret||'', xunfei_ws_url: d.ws_url||D.xunfei_ws_url, xunfei_voice: d.voice||'', xunfei_audio_encoding: d.audio_encoding||'raw', xunfei_sample_rate: d.sample_rate||16000, xunfei_speed: d.speed??50, xunfei_volume: d.volume??50, xunfei_pitch: d.pitch??50, xunfei_connect_timeout: d.connect_timeout||10, xunfei_read_timeout: d.read_timeout||30, xunfei_frame_duration: d.frame_duration||60,
      xunfei_super_app_id: d.app_id||'', xunfei_super_api_key: d.api_key||'', xunfei_super_api_secret: d.api_secret||'', xunfei_super_ws_url: d.ws_url||D.xunfei_super_ws_url, xunfei_super_voice: d.voice||'', xunfei_super_audio_encoding: d.audio_encoding||'raw', xunfei_super_sample_rate: d.sample_rate||24000, xunfei_super_speed: d.speed??50, xunfei_super_volume: d.volume??50, xunfei_super_pitch: d.pitch??50, xunfei_super_connect_timeout: d.connect_timeout||10, xunfei_super_read_timeout: d.read_timeout||30, xunfei_super_frame_duration: d.frame_duration||60,
      indextts_api_url: d.api_url||'', indextts_api_key: d.api_key||'', indextts_model: d.model||'indextts-vllm', indextts_voice: d.voice||'', indextts_frame_duration: d.frame_duration||60,
      cosyvoice_api_url: d.api_url||'', cosyvoice_spk_id: d.spk_id||'', cosyvoice_frame_duration: d.frame_duration||60, cosyvoice_target_sr: d.target_sr||22050, cosyvoice_audio_format: d.audio_format||'wav', cosyvoice_instruct_text: d.instruct_text||'',
      supertonic_onnx_dir: d.onnx_dir||'', supertonic_voice: d.voice||'M1', supertonic_voice_json_path: d.voice_json_path||'',
      supertonic_lang: d.lang||'na', supertonic_steps: d.steps||8, supertonic_speed: d.speed||1.0,
      supertonic_silence: d.silence_duration||0.3, supertonic_frame_duration: d.frame_duration||60,
    }
  } catch { return { ...D } }
}

const F = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="grid gap-1.5"><label className="text-sm font-medium text-[var(--color-text)]">{label}</label>{children}</div>
)

export function TtsConfigForm({ form, setForm, editing }: { form: ConfigForm; setForm: (p: Partial<ConfigForm>) => void; editing: ConfigRow | null }) {
  const { t } = useLocale()
  const [f, setF] = useState<TtsFields>(() => parse(editing))
  const [voiceOptions, setVoiceOptions] = useState<Array<{label:string;value:string}>>([])
  const [voiceLoading, setVoiceLoading] = useState(false)
  const [isTesting, setIsTesting] = useState(false)
  const providerRef = useRef(f.provider)

  useEffect(() => { const parsed = parse(editing); setF(parsed); providerRef.current = parsed.provider }, [editing])

  useEffect(() => {
    const prov = f.provider
    if (!TTS_PROVIDERS_WITH_VOICES.includes(prov)) { setVoiceOptions([]); return }
    setVoiceLoading(true)
    const params: Record<string, string> = { provider: prov }
    if (f.config_id) params.config_id = f.config_id
    if (prov === 'indextts_vllm' && f.indextts_api_url) params.api_url = f.indextts_api_url
    api.get('/user/voice-options', { params })
      .then(r => setVoiceOptions(r.data.data || []))
      .catch(() => toast.error(t('load_voice_list_failed')))
      .finally(() => setVoiceLoading(false))
  }, [f.provider]) // eslint-disable-line react-hooks/exhaustive-deps

  const upd = (patch: Partial<TtsFields>) => {
    const next = { ...f, ...patch }
    setF(next)
    setForm({ name: next.name, config_id: next.config_id, provider: next.provider, enabled: next.enabled, is_default: next.is_default, json_data: serialize(next) })
  }

  async function handleTest() {
    const cid = f.config_id || '_test_draft'
    setIsTesting(true)
    try {
      const cfgItem = { provider: f.provider, name: f.name, is_default: f.is_default, ...JSON.parse(serialize(f)) }
      const res = await api.post('/admin/configs/test', {
        types: ['tts'],
        data: { tts: { [cid]: cfgItem } },
        config_ids: { tts: [cid] },
      }, { timeout: 30000 })
      const ttsResult = res.data?.data?.tts
      if (!ttsResult) { toast.error(t('test_failed')); return }
      if (ttsResult._no_client) { toast.error(t('main_server_not_connected')); return }
      const errEntry = ttsResult._error
      if (errEntry) { toast.error(`${t('test_failed')}: ${errEntry.message || ''}`); return }
      const entry = ttsResult[cid]
      if (entry?.ok) {
        toast.success(`${t('tts_test_ok')}${entry.first_packet_ms ? ` (${entry.first_packet_ms}ms)` : ''}`)
      } else {
        toast.error(`${t('test_failed')}: ${entry?.message || ''}`)
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      toast.error(`${t('test_failed')}: ${msg}`)
    } finally {
      setIsTesting(false)
    }
  }

  const providerOptions = getTTSProviderOptions(t)

  return (
    <div className="grid gap-3">
      <F label={t('provider')}>
        <Select value={f.provider} onValueChange={v => upd({ provider: v })}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{providerOptions.map(p => (
            <SelectItem key={p.value} value={p.value}>
              <span className="flex items-center gap-2">
                {p.label}
                {p.supportsVoiceClone && <span className="text-xs px-1.5 py-0.5 rounded border border-emerald-300 text-emerald-600 leading-none">{t('support_clone')}</span>}
              </span>
            </SelectItem>
          ))}</SelectContent>
        </Select>
      </F>
      <div className="grid grid-cols-2 gap-3">
        <F label={t('config_name')}><Input value={f.name} onChange={e => upd({ name: e.target.value })} placeholder={t('enter_config_name')} /></F>
        <F label={t('config_id')}><Input value={f.config_id} onChange={e => upd({ config_id: e.target.value })} placeholder={t('enter_unique_config_id')} /></F>
      </div>
      <TtsProviderFields f={f} upd={upd} t={t} voiceOptions={voiceOptions} voiceLoading={voiceLoading} />
      <div className="flex items-center justify-between pt-1">
        <div className="flex items-center gap-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <Switch checked={f.enabled} onCheckedChange={v => upd({ enabled: v })} /><span className="text-sm">{t('enabled_status')}</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <Switch checked={f.is_default} onCheckedChange={v => upd({ is_default: v })} /><span className="text-sm">{t('default_config')}</span>
          </label>
        </div>
        <Button variant="outline" size="sm" onClick={handleTest} disabled={isTesting}>
          {isTesting ? '...' : t('test_current_config')}
        </Button>
      </div>
    </div>
  )
}
