import { useEffect, useState } from 'react'
import { useLocale } from '@/hooks/use-locale'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { ConfigRow, ConfigForm } from './config-list-page'

interface AsrFields {
  provider: string; name: string; config_id: string; enabled: boolean; is_default: boolean
  funasr_host: string; funasr_port: number; funasr_mode: string; funasr_sample_rate: number
  funasr_chunk0: number; funasr_chunk1: number; funasr_chunk2: number
  funasr_chunk_interval: number; funasr_max_connections: number; funasr_timeout: number; funasr_auto_end: boolean
  aliyun_api_key: string; aliyun_ws_url: string; aliyun_model: string; aliyun_format: string
  aliyun_sample_rate: number; aliyun_language_hints: string; aliyun_vocabulary_id: string
  aliyun_disfluency: boolean; aliyun_timeout: number
  doubao_appid: string; doubao_token: string; doubao_ws_url: string; doubao_resource_id: string
  doubao_end_window: number; doubao_punc: boolean; doubao_itn: boolean; doubao_ddc: boolean
  doubao_chunk_duration: number; doubao_timeout: number
  qwen3_api_key: string; qwen3_ws_url: string; qwen3_model: string; qwen3_format: string
  qwen3_sample_rate: number; qwen3_language: string; qwen3_auto_end: boolean
  qwen3_vad_threshold: number; qwen3_vad_silence_ms: number; qwen3_timeout: number
  xunfei_appid: string; xunfei_api_key: string; xunfei_api_secret: string
  xunfei_host: string; xunfei_path: string; xunfei_domain: string; xunfei_language: string
  xunfei_accent: string; xunfei_sample_rate: number; xunfei_timeout: number
}

const D: AsrFields = {
  provider:'funasr', name:'', config_id:'', enabled:true, is_default:false,
  funasr_host:'127.0.0.1', funasr_port:10095, funasr_mode:'offline', funasr_sample_rate:16000,
  funasr_chunk0:5, funasr_chunk1:10, funasr_chunk2:5, funasr_chunk_interval:10, funasr_max_connections:100, funasr_timeout:30, funasr_auto_end:false,
  aliyun_api_key:'', aliyun_ws_url:'wss://dashscope.aliyuncs.com/api-ws/v1/inference/', aliyun_model:'fun-asr-realtime', aliyun_format:'pcm', aliyun_sample_rate:16000, aliyun_language_hints:'zh', aliyun_vocabulary_id:'', aliyun_disfluency:false, aliyun_timeout:30,
  doubao_appid:'', doubao_token:'', doubao_ws_url:'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async', doubao_resource_id:'volc.bigasr.sauc.duration', doubao_end_window:800, doubao_punc:true, doubao_itn:true, doubao_ddc:false, doubao_chunk_duration:200, doubao_timeout:30,
  qwen3_api_key:'', qwen3_ws_url:'wss://dashscope.aliyuncs.com/api-ws/v1/realtime', qwen3_model:'qwen3-asr-flash-realtime', qwen3_format:'pcm', qwen3_sample_rate:16000, qwen3_language:'zh', qwen3_auto_end:false, qwen3_vad_threshold:0.3, qwen3_vad_silence_ms:500, qwen3_timeout:30,
  xunfei_appid:'', xunfei_api_key:'', xunfei_api_secret:'', xunfei_host:'iat-api.xfyun.cn', xunfei_path:'/v2/iat', xunfei_domain:'iat', xunfei_language:'zh_cn', xunfei_accent:'mandarin', xunfei_sample_rate:16000, xunfei_timeout:30,
}

function parse(row: ConfigRow | null): AsrFields {
  if (!row) return { ...D }
  try {
    const d = JSON.parse(row.json_data || '{}')
    const prov = row.provider || 'funasr'
    const cs = Array.isArray(d.chunk_size) && d.chunk_size.length >= 3 ? d.chunk_size : [5,10,5]
    return {
      ...D, provider:prov, name:row.name, config_id:row.config_id, enabled:row.enabled!==false, is_default:!!row.is_default,
      funasr_host:d.host||'127.0.0.1', funasr_port:d.port||10095, funasr_mode:d.mode||'offline', funasr_sample_rate:d.sample_rate||16000,
      funasr_chunk0:cs[0], funasr_chunk1:cs[1], funasr_chunk2:cs[2], funasr_chunk_interval:d.chunk_interval||10, funasr_max_connections:d.max_connections||100, funasr_timeout:d.timeout||30, funasr_auto_end:!!d.auto_end,
      aliyun_api_key:d.api_key||'', aliyun_ws_url:d.ws_url||D.aliyun_ws_url, aliyun_model:d.model||'fun-asr-realtime', aliyun_format:d.format||'pcm', aliyun_sample_rate:d.sample_rate||16000, aliyun_language_hints:Array.isArray(d.language_hints)?d.language_hints.join(','):(d.language_hints||'zh'), aliyun_vocabulary_id:d.vocabulary_id||'', aliyun_disfluency:!!d.disfluency_removal_enabled, aliyun_timeout:d.timeout||30,
      doubao_appid:d.appid||'', doubao_token:d.access_token||'', doubao_ws_url:d.ws_url||D.doubao_ws_url, doubao_resource_id:d.resource_id||'volc.bigasr.sauc.duration', doubao_end_window:d.end_window_size||800, doubao_punc:d.enable_punc!==false, doubao_itn:d.enable_itn!==false, doubao_ddc:!!d.enable_ddc, doubao_chunk_duration:d.chunk_duration||200, doubao_timeout:d.timeout||30,
      qwen3_api_key:d.api_key||'', qwen3_ws_url:d.ws_url||D.qwen3_ws_url, qwen3_model:d.model||'qwen3-asr-flash-realtime', qwen3_format:d.format||'pcm', qwen3_sample_rate:d.sample_rate||16000, qwen3_language:d.language||'zh', qwen3_auto_end:!!d.auto_end, qwen3_vad_threshold:d.vad_threshold??0.3, qwen3_vad_silence_ms:d.vad_silence_ms||500, qwen3_timeout:d.timeout||30,
      xunfei_appid:d.appid||'', xunfei_api_key:d.api_key||'', xunfei_api_secret:d.api_secret||'', xunfei_host:d.host||'iat-api.xfyun.cn', xunfei_path:d.path||'/v2/iat', xunfei_domain:d.domain||'iat', xunfei_language:d.language||'zh_cn', xunfei_accent:d.accent||'mandarin', xunfei_sample_rate:d.sample_rate||16000, xunfei_timeout:d.timeout||30,
    }
  } catch { return { ...D } }
}

function serialize(f: AsrFields): string {
  const p = f.provider
  if (p==='funasr') return JSON.stringify({host:f.funasr_host,port:f.funasr_port,mode:f.funasr_mode,sample_rate:f.funasr_sample_rate,chunk_size:[f.funasr_chunk0,f.funasr_chunk1,f.funasr_chunk2],chunk_interval:f.funasr_chunk_interval,max_connections:f.funasr_max_connections,timeout:f.funasr_timeout,auto_end:f.funasr_auto_end})
  if (p==='aliyun_funasr') return JSON.stringify({api_key:f.aliyun_api_key,ws_url:f.aliyun_ws_url,model:f.aliyun_model,format:f.aliyun_format,sample_rate:f.aliyun_sample_rate,language_hints:f.aliyun_language_hints.split(',').map(s=>s.trim()).filter(Boolean),vocabulary_id:f.aliyun_vocabulary_id,disfluency_removal_enabled:f.aliyun_disfluency,timeout:f.aliyun_timeout})
  if (p==='doubao') return JSON.stringify({appid:f.doubao_appid,access_token:f.doubao_token,ws_url:f.doubao_ws_url,resource_id:f.doubao_resource_id,end_window_size:f.doubao_end_window,enable_punc:f.doubao_punc,enable_itn:f.doubao_itn,enable_ddc:f.doubao_ddc,chunk_duration:f.doubao_chunk_duration,timeout:f.doubao_timeout})
  if (p==='aliyun_qwen3') return JSON.stringify({api_key:f.qwen3_api_key,ws_url:f.qwen3_ws_url,model:f.qwen3_model,format:f.qwen3_format,sample_rate:f.qwen3_sample_rate,language:f.qwen3_language,auto_end:f.qwen3_auto_end,vad_threshold:f.qwen3_vad_threshold,vad_silence_ms:f.qwen3_vad_silence_ms,timeout:f.qwen3_timeout})
  if (p==='xunfei') return JSON.stringify({appid:f.xunfei_appid,api_key:f.xunfei_api_key,api_secret:f.xunfei_api_secret,host:f.xunfei_host,path:f.xunfei_path,domain:f.xunfei_domain,language:f.xunfei_language,accent:f.xunfei_accent,sample_rate:f.xunfei_sample_rate,timeout:f.xunfei_timeout})
  return '{}'
}

const F = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="grid gap-1.5"><label className="text-sm font-medium text-[var(--color-text)]">{label}</label>{children}</div>
)
const N = ({ v, min, max, step, onChange }: { v: number; min?: number; max?: number; step?: number; onChange: (n: number) => void }) => (
  <Input type="number" value={v} min={min} max={max} step={step} onChange={e => onChange(Number(e.target.value))} />
)

const ASR_PROVIDERS = [
  { label:'FunASR', value:'funasr' },
  { label:'aliyun_funasr_asr', value:'aliyun_funasr', labelKey:true },
  { label:'doubao_asr', value:'doubao', labelKey:true },
  { label:'aliyun_qwen3_asr', value:'aliyun_qwen3', labelKey:true },
  { label:'xunfei', value:'xunfei', labelKey:true },
]

export function AsrConfigForm({ form: _form, setForm, editing }: { form: ConfigForm; setForm: (p: Partial<ConfigForm>) => void; editing: ConfigRow | null }) {
  const { t } = useLocale()
  const [f, setF] = useState<AsrFields>(() => parse(editing))
  useEffect(() => { const parsed = parse(editing); setF(parsed) }, [editing])

  const upd = (patch: Partial<AsrFields>) => {
    const next = { ...f, ...patch }
    setF(next)
    setForm({ name: next.name, config_id: next.config_id, provider: next.provider, enabled: next.enabled, is_default: next.is_default, json_data: serialize(next) })
  }

  const p = f.provider
  return (
    <div className="grid gap-3">
      <F label={t('provider')}>
        <Select value={p} onValueChange={v => upd({ provider: v })}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{ASR_PROVIDERS.map(o => <SelectItem key={o.value} value={o.value}>{o.labelKey ? t(o.label) : o.label}</SelectItem>)}</SelectContent>
        </Select>
      </F>
      <div className="grid grid-cols-2 gap-3">
        <F label={t('config_name')}><Input value={f.name} onChange={e => upd({ name: e.target.value })} placeholder={t('enter_config_name')} /></F>
        <F label={t('config_id')}><Input value={f.config_id} onChange={e => upd({ config_id: e.target.value })} placeholder={t('enter_unique_config_id')} /></F>
      </div>
      {p==='funasr' && <>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('host_address')}><Input value={f.funasr_host} onChange={e => upd({ funasr_host: e.target.value })} /></F>
          <F label={t('port')}><N v={f.funasr_port} min={1} max={65535} onChange={v => upd({ funasr_port: v })} /></F>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('mode')}>
            <Select value={f.funasr_mode} onValueChange={v => upd({ funasr_mode: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>{['2pass','offline','online'].map(v=><SelectItem key={v} value={v}>{v}</SelectItem>)}</SelectContent>
            </Select>
          </F>
          <F label={t('sample_rate')}>
            <Select value={String(f.funasr_sample_rate)} onValueChange={v => upd({ funasr_sample_rate: Number(v) })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>{[8000,16000,44100,48000].map(v=><SelectItem key={v} value={String(v)}>{v}</SelectItem>)}</SelectContent>
            </Select>
          </F>
        </div>
        <F label={t('chunk_size')}>
          <div className="flex gap-2">
            <N v={f.funasr_chunk0} min={1} onChange={v => upd({ funasr_chunk0: v })} />
            <N v={f.funasr_chunk1} min={1} onChange={v => upd({ funasr_chunk1: v })} />
            <N v={f.funasr_chunk2} min={1} onChange={v => upd({ funasr_chunk2: v })} />
          </div>
          <p className="text-xs text-[var(--color-text-secondary)]">{t('frame_size_format_hint')}</p>
        </F>
        <div className="grid grid-cols-3 gap-3">
          <F label={t('chunk_interval')}><N v={f.funasr_chunk_interval} min={1} onChange={v => upd({ funasr_chunk_interval: v })} /></F>
          <F label={t('max_connections')}><N v={f.funasr_max_connections} min={1} onChange={v => upd({ funasr_max_connections: v })} /></F>
          <F label={t('timeout_seconds')}><N v={f.funasr_timeout} min={1} onChange={v => upd({ funasr_timeout: v })} /></F>
        </div>
        <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.funasr_auto_end} onCheckedChange={v => upd({ funasr_auto_end: v })} /><span className="text-sm">{t('auto_end')}</span></label>
      </>}
      {p==='aliyun_funasr' && <>
        <F label="API Key"><Input type="password" value={f.aliyun_api_key} onChange={e => upd({ aliyun_api_key: e.target.value })} placeholder={t('optional_dashscope_key')} /></F>
        <F label="WS URL"><Input value={f.aliyun_ws_url} onChange={e => upd({ aliyun_ws_url: e.target.value })} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('model')}><Input value={f.aliyun_model} onChange={e => upd({ aliyun_model: e.target.value })} placeholder="fun-asr-realtime" /></F>
          <F label={t('timeout_seconds')}><N v={f.aliyun_timeout} min={1} onChange={v => upd({ aliyun_timeout: v })} /></F>
        </div>
        <F label={t('language_hint')}><Input value={f.aliyun_language_hints} onChange={e => upd({ aliyun_language_hints: e.target.value })} placeholder="zh,en" /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('vocab_id')}><Input value={f.aliyun_vocabulary_id} onChange={e => upd({ aliyun_vocabulary_id: e.target.value })} placeholder={t('optional_empty') ?? ''} /></F>
          <F label={t('sample_rate')}>
            <Select value={String(f.aliyun_sample_rate)} onValueChange={v => upd({ aliyun_sample_rate: Number(v) })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="16000">16000</SelectItem></SelectContent>
            </Select>
          </F>
        </div>
        <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.aliyun_disfluency} onCheckedChange={v => upd({ aliyun_disfluency: v })} /><span className="text-sm">{t('remove_filler_words')}</span></label>
      </>}
      {p==='doubao' && <>
        <F label={t('app_id')}><Input value={f.doubao_appid} onChange={e => upd({ doubao_appid: e.target.value })} /></F>
        <F label={t('access_token')}><Input type="password" value={f.doubao_token} onChange={e => upd({ doubao_token: e.target.value })} /></F>
        <F label="WebSocket URL"><Input value={f.doubao_ws_url} onChange={e => upd({ doubao_ws_url: e.target.value })} /></F>
        <F label={t('resource_spec')}>
          <Select value={f.doubao_resource_id} onValueChange={v => upd({ doubao_resource_id: v })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>{[['volc.bigasr.sauc.duration','doubao_stream_asr_v1_hourly'],['volc.bigasr.sauc.concurrent','doubao_stream_asr_v1_concurrent'],['volc.seedasr.sauc.duration','doubao_stream_asr_v2_hourly'],['volc.seedasr.sauc.concurrent','doubao_stream_asr_v2_concurrent']].map(([v,k])=><SelectItem key={v} value={v}>{t(k)}</SelectItem>)}</SelectContent>
          </Select>
        </F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('end_window_size')}><N v={f.doubao_end_window} min={1} onChange={v => upd({ doubao_end_window: v })} /></F>
          <F label={t('timeout_seconds')}><N v={f.doubao_timeout} min={1} onChange={v => upd({ doubao_timeout: v })} /></F>
        </div>
        <div className="flex gap-4">
          <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.doubao_punc} onCheckedChange={v => upd({ doubao_punc: v })} /><span className="text-sm">{t('enable_punctuation')}</span></label>
          <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.doubao_itn} onCheckedChange={v => upd({ doubao_itn: v })} /><span className="text-sm">{t('enable_inverse_text_normalization')}</span></label>
          <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.doubao_ddc} onCheckedChange={v => upd({ doubao_ddc: v })} /><span className="text-sm">{t('enable_digit_detection')}</span></label>
        </div>
      </>}
      {p==='aliyun_qwen3' && <>
        <F label="API Key"><Input type="password" value={f.qwen3_api_key} onChange={e => upd({ qwen3_api_key: e.target.value })} placeholder={t('optional_dashscope_key')} /></F>
        <F label="WS URL"><Input value={f.qwen3_ws_url} onChange={e => upd({ qwen3_ws_url: e.target.value })} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('model')}><Input value={f.qwen3_model} onChange={e => upd({ qwen3_model: e.target.value })} placeholder="qwen3-asr-flash-realtime" /></F>
          <F label={t('language')}><Input value={f.qwen3_language} onChange={e => upd({ qwen3_language: e.target.value })} placeholder="zh" /></F>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('sample_rate')}>
            <Select value={String(f.qwen3_sample_rate)} onValueChange={v => upd({ qwen3_sample_rate: Number(v) })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="8000">8000</SelectItem><SelectItem value="16000">16000</SelectItem></SelectContent>
            </Select>
          </F>
          <F label={t('timeout_seconds')}><N v={f.qwen3_timeout} min={1} onChange={v => upd({ qwen3_timeout: v })} /></F>
        </div>
        <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.qwen3_auto_end} onCheckedChange={v => upd({ qwen3_auto_end: v })} /><span className="text-sm">{t('auto_end')}</span></label>
        {f.qwen3_auto_end && <div className="grid grid-cols-2 gap-3">
          <F label={t('vad_threshold')}><N v={f.qwen3_vad_threshold} min={0} max={1} step={0.1} onChange={v => upd({ qwen3_vad_threshold: v })} /></F>
          <F label={t('vad_silence_duration')}><N v={f.qwen3_vad_silence_ms} min={0} onChange={v => upd({ qwen3_vad_silence_ms: v })} /></F>
        </div>}
      </>}
      {p==='xunfei' && <>
        <F label={t('app_id')}><Input value={f.xunfei_appid} onChange={e => upd({ xunfei_appid: e.target.value })} placeholder={t('enter_xunfei_app_id')} /></F>
        <F label="API Key"><Input type="password" value={f.xunfei_api_key} onChange={e => upd({ xunfei_api_key: e.target.value })} placeholder={t('enter_xunfei_api_key')} /></F>
        <F label="API Secret"><Input type="password" value={f.xunfei_api_secret} onChange={e => upd({ xunfei_api_secret: e.target.value })} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label="Host"><Input value={f.xunfei_host} onChange={e => upd({ xunfei_host: e.target.value })} placeholder="iat-api.xfyun.cn" /></F>
          <F label="Path"><Input value={f.xunfei_path} onChange={e => upd({ xunfei_path: e.target.value })} placeholder="/v2/iat" /></F>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('business_domain')}><Input value={f.xunfei_domain} onChange={e => upd({ xunfei_domain: e.target.value })} placeholder="iat" /></F>
          <F label={t('language')}><Input value={f.xunfei_language} onChange={e => upd({ xunfei_language: e.target.value })} placeholder="zh_cn" /></F>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('dialect')}><Input value={f.xunfei_accent} onChange={e => upd({ xunfei_accent: e.target.value })} placeholder="mandarin" /></F>
          <F label={t('timeout_seconds')}><N v={f.xunfei_timeout} min={1} onChange={v => upd({ xunfei_timeout: v })} /></F>
        </div>
      </>}
      <div className="flex items-center gap-6 pt-1">
        <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.enabled} onCheckedChange={v => upd({ enabled: v })} /><span className="text-sm">{t('enabled_status')}</span></label>
        <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.is_default} onCheckedChange={v => upd({ is_default: v })} /><span className="text-sm">{t('default_config')}</span></label>
      </div>
    </div>
  )
}
