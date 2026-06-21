const SUPERTONIC_LANGS = [
  { value: 'ar', label: 'Arabic' }, { value: 'bg', label: 'Bulgarian' }, { value: 'hr', label: 'Croatian' },
  { value: 'cs', label: 'Czech' }, { value: 'da', label: 'Danish' }, { value: 'nl', label: 'Dutch' },
  { value: 'en', label: 'English' }, { value: 'et', label: 'Estonian' }, { value: 'fi', label: 'Finnish' },
  { value: 'fr', label: 'French' }, { value: 'de', label: 'German' }, { value: 'el', label: 'Greek' },
  { value: 'hi', label: 'Hindi' }, { value: 'hu', label: 'Hungarian' }, { value: 'id', label: 'Indonesian' },
  { value: 'it', label: 'Italian' }, { value: 'ja', label: 'Japanese' }, { value: 'ko', label: 'Korean' },
  { value: 'lv', label: 'Latvian' }, { value: 'lt', label: 'Lithuanian' }, { value: 'pl', label: 'Polish' },
  { value: 'pt', label: 'Portuguese' }, { value: 'ro', label: 'Romanian' }, { value: 'ru', label: 'Russian' },
  { value: 'sk', label: 'Slovak' }, { value: 'sl', label: 'Slovenian' }, { value: 'es', label: 'Spanish' },
  { value: 'sv', label: 'Swedish' }, { value: 'tr', label: 'Turkish' }, { value: 'uk', label: 'Ukrainian' },
  { value: 'vi', label: 'Vietnamese' },
]
const SUPERTONIC_PRESET_VOICES = ['M1','M2','M3','M4','M5','F1','F2','F3','F4','F5']

import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import api from '@/utils/api'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ComboInput } from '@/components/ui/combo-input'
import type { TtsFields } from './tts-config-form'

type T = (k: string) => string
const F = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="grid gap-1.5"><label className="text-sm font-medium text-[var(--color-text)]">{label}</label>{children}</div>
)
const NumInput = ({ value, min, max, step, onChange }: { value: number | string; min?: number; max?: number; step?: number; onChange: (v: number) => void }) => (
  <Input type="number" value={value} min={min} max={max} step={step} onChange={e => onChange(Number(e.target.value))} />
)
const VoiceSelect = ({ value, options, loading, onChange, placeholder }: { value: string; options: Array<{label:string;value:string}>; loading: boolean; onChange: (v: string) => void; placeholder: string }) => (
  <ComboInput value={value} onChange={onChange} options={options} loading={loading} placeholder={placeholder} />
)

type ModelStatus = 'idle' | 'checking' | 'found' | 'not_found' | 'downloading'

function SupertonicModelSection({ f, upd, t }: { f: TtsFields; upd: (p: Partial<TtsFields>) => void; t: T }) {
  const [status, setStatus] = useState<ModelStatus>('idle')

  useEffect(() => {
    setStatus('checking')
    const params: Record<string, string> = {}
    if (f.supertonic_onnx_dir) params.path = f.supertonic_onnx_dir
    api.get('/admin/supertonic-model', { params })
      .then(r => {
        if (r.data.exists) {
          setStatus('found')
          if (!f.supertonic_onnx_dir) upd({ supertonic_onnx_dir: r.data.onnx_dir })
        } else {
          setStatus('not_found')
          if (!f.supertonic_onnx_dir) upd({ supertonic_onnx_dir: r.data.default_path })
        }
      })
      .catch(() => setStatus('not_found'))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  async function handleDownload() {
    setStatus('downloading')
    try {
      const res = await api.post('/admin/supertonic-model/download',
        f.supertonic_onnx_dir ? { onnx_dir: f.supertonic_onnx_dir } : {},
        { timeout: 15 * 60 * 1000 }
      )
      upd({ supertonic_onnx_dir: res.data.onnx_dir })
      setStatus('found')
      toast.success(t('model_downloaded'))
    } catch (e: unknown) {
      setStatus('not_found')
      const msg = e instanceof Error ? e.message : String(e)
      toast.error(`${t('download_failed')}: ${msg}`)
    }
  }

  return (
    <>
      <F label={t('onnx_model_dir')}>
        <Input value={f.supertonic_onnx_dir} onChange={e => upd({ supertonic_onnx_dir: e.target.value })} placeholder="~/.cache/supertonic-model/onnx" />
      </F>
      <div className="flex items-center gap-2 text-sm min-h-[28px]">
        {status === 'checking' && <span className="text-[var(--color-text-muted)]">{t('checking_model')}…</span>}
        {status === 'found' && <span className="text-emerald-600">✓ {t('model_found')}</span>}
        {status === 'downloading' && <span className="text-[var(--color-text-muted)]">{t('downloading_model')}…</span>}
        {(status === 'not_found' || status === 'idle') && (
          <>
            <span className="text-amber-600">{t('model_not_found')}</span>
            <Button variant="outline" size="sm" onClick={handleDownload}>{t('download_model')}</Button>
          </>
        )}
      </div>
    </>
  )
}

export function TtsProviderFields({ f, upd, t, voiceOptions, voiceLoading }: {
  f: TtsFields; upd: (p: Partial<TtsFields>) => void; t: T
  voiceOptions: Array<{label: string; value: string}>; voiceLoading: boolean
}) {
  const p = f.provider
  return (
    <>
      {p === 'doubao_ws' && <>
        <F label={t('app_id')}><Input value={f.doubao_ws_appid} onChange={e => upd({ doubao_ws_appid: e.target.value })} placeholder={t('enter_app_id')} /></F>
        <F label={t('access_token')}><Input type="password" value={f.doubao_ws_token} onChange={e => upd({ doubao_ws_token: e.target.value })} placeholder={t('enter_access_token')} /></F>
        <F label={t('model')}>
          <Select value={f.doubao_ws_model} onValueChange={v => upd({ doubao_ws_model: v })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>{['seed-tts-2.0-standard','seed-tts-2.0-expressive','seed-tts-1.1','seed-icl-1.0','seed-icl-2.0-standard','seed-icl-2.0-expressive'].map(v => <SelectItem key={v} value={v}>{v}</SelectItem>)}</SelectContent>
          </Select>
        </F>
        <F label={t('resource_id')}><Input value={f.doubao_ws_resource_id} onChange={e => upd({ doubao_ws_resource_id: e.target.value })} /></F>
        <F label={t('voice_timbre')}><VoiceSelect value={f.doubao_ws_voice} options={voiceOptions} loading={voiceLoading} onChange={v => upd({ doubao_ws_voice: v })} placeholder={t('select_timbre')} /></F>
        <F label="WebSocket URL"><Input value={f.doubao_ws_url} onChange={e => upd({ doubao_ws_url: e.target.value })} placeholder="wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream" /></F>
      </>}
      {p === 'edge' && <>
        <F label={t('voice_timbre')}><VoiceSelect value={f.edge_voice} options={voiceOptions} loading={voiceLoading} onChange={v => upd({ edge_voice: v })} placeholder={t('select_timbre')} /></F>
        <F label={t('speech_rate')}><Input value={f.edge_rate} onChange={e => upd({ edge_rate: e.target.value })} placeholder={t('enter_speech_rate_placeholder')} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('volume')}><Input value={f.edge_volume} onChange={e => upd({ edge_volume: e.target.value })} placeholder="+0%" /></F>
          <F label={t('pitch')}><Input value={f.edge_pitch} onChange={e => upd({ edge_pitch: e.target.value })} placeholder="+0Hz" /></F>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('connection_timeout')}><NumInput value={f.edge_connect_timeout} min={1} max={60} onChange={v => upd({ edge_connect_timeout: v })} /></F>
          <F label={t('receive_timeout')}><NumInput value={f.edge_receive_timeout} min={1} max={300} onChange={v => upd({ edge_receive_timeout: v })} /></F>
        </div>
      </>}
      {p === 'edge_offline' && <>
        <F label={t('server_url_label')}><Input value={f.edge_offline_server_url} onChange={e => upd({ edge_offline_server_url: e.target.value })} placeholder="http://..." /></F>
        <div className="grid grid-cols-3 gap-3">
          <F label={t('timeout_label')}><NumInput value={f.edge_offline_timeout} min={1} max={300} onChange={v => upd({ edge_offline_timeout: v })} /></F>
          <F label={t('sample_rate')}><NumInput value={f.edge_offline_sample_rate} min={8000} max={48000} onChange={v => upd({ edge_offline_sample_rate: v })} /></F>
          <F label={t('channel_count_label')}><NumInput value={f.edge_offline_channels} min={1} max={8} onChange={v => upd({ edge_offline_channels: v })} /></F>
        </div>
        <F label={t('frame_duration')}><NumInput value={f.edge_offline_frame_duration} min={1} max={100} onChange={v => upd({ edge_offline_frame_duration: v })} /></F>
      </>}
      {p === 'aliyun_qwen' && <>
        <F label="API Key"><Input type="password" value={f.qwen_api_key} onChange={e => upd({ qwen_api_key: e.target.value })} placeholder={t('enter_api_key')} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('region')}>
            <Select value={f.qwen_region} onValueChange={v => upd({ qwen_region: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="beijing">{t('beijing')}</SelectItem><SelectItem value="singapore">{t('singapore')}</SelectItem></SelectContent>
            </Select>
          </F>
          <F label={t('model')}><Input value={f.qwen_model} onChange={e => upd({ qwen_model: e.target.value })} placeholder="qwen3-tts-flash" /></F>
        </div>
        <F label={t('voice_timbre')}><Input value={f.qwen_voice} onChange={e => upd({ qwen_voice: e.target.value })} placeholder="Cherry" /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('language_type')}>
            <Select value={f.qwen_language_type} onValueChange={v => upd({ qwen_language_type: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="Auto">{t('auto')}</SelectItem><SelectItem value="Chinese">{t('chinese_zh')}</SelectItem><SelectItem value="English">{t('english_en')}</SelectItem></SelectContent>
            </Select>
          </F>
          <F label={t('frame_duration')}><NumInput value={f.qwen_frame_duration} min={1} max={1000} onChange={v => upd({ qwen_frame_duration: v })} /></F>
        </div>
        <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.qwen_stream} onCheckedChange={v => upd({ qwen_stream: v })} /><span className="text-sm">{t('use_streaming')}</span></label>
      </>}
      {p === 'zhipu' && <>
        <F label="API Key"><Input type="password" value={f.zhipu_api_key} onChange={e => upd({ zhipu_api_key: e.target.value })} placeholder={t('enter_api_key')} /></F>
        <F label="API URL"><Input value={f.zhipu_api_url} onChange={e => upd({ zhipu_api_url: e.target.value })} placeholder="https://open.bigmodel.cn/api/paas/v4/audio/speech" /></F>
        <F label={t('model')}><Input value={f.zhipu_model} onChange={e => upd({ zhipu_model: e.target.value })} placeholder="glm-tts" /></F>
        <F label={t('voice_timbre')}><VoiceSelect value={f.zhipu_voice} options={voiceOptions} loading={voiceLoading} onChange={v => upd({ zhipu_voice: v })} placeholder={t('select_timbre')} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('response_format_label')}>
            <Select value={f.zhipu_response_format} onValueChange={v => upd({ zhipu_response_format: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="wav">WAV</SelectItem><SelectItem value="pcm">PCM</SelectItem></SelectContent>
            </Select>
          </F>
          <F label={t('frame_duration')}><NumInput value={f.zhipu_frame_duration} min={1} max={1000} onChange={v => upd({ zhipu_frame_duration: v })} /></F>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('volume')}><NumInput value={f.zhipu_volume} min={0} max={10} step={0.1} onChange={v => upd({ zhipu_volume: v })} /></F>
          <F label={t('speech_rate')}><NumInput value={f.zhipu_speed} min={0.5} max={2} step={0.1} onChange={v => upd({ zhipu_speed: v })} /></F>
        </div>
        <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.zhipu_stream} onCheckedChange={v => upd({ zhipu_stream: v })} /><span className="text-sm">{t('use_streaming')}</span></label>
        {f.zhipu_stream && <F label={t('encoding_format')}>
          <Select value={f.zhipu_encode_format} onValueChange={v => upd({ zhipu_encode_format: v })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent><SelectItem value="base64">Base64</SelectItem><SelectItem value="hex">Hex</SelectItem></SelectContent>
          </Select>
        </F>}
      </>}
      {p === 'minimax' && <>
        <F label="API Key"><Input type="password" value={f.minimax_api_key} onChange={e => upd({ minimax_api_key: e.target.value })} placeholder={t('enter_api_key')} /></F>
        <F label={t('model')}><Input value={f.minimax_model} onChange={e => upd({ minimax_model: e.target.value })} placeholder="speech-2.8-hd" /></F>
        <F label={t('voice_timbre')}><VoiceSelect value={f.minimax_voice} options={voiceOptions} loading={voiceLoading} onChange={v => upd({ minimax_voice: v })} placeholder={t('select_timbre')} /></F>
        <div className="grid grid-cols-3 gap-3">
          <F label={t('speech_rate')}><NumInput value={f.minimax_speed} min={0.5} max={2} step={0.1} onChange={v => upd({ minimax_speed: v })} /></F>
          <F label={t('volume')}><NumInput value={f.minimax_vol} min={0} max={2} step={0.1} onChange={v => upd({ minimax_vol: v })} /></F>
          <F label={t('pitch')}><NumInput value={f.minimax_pitch} min={-12} max={12} onChange={v => upd({ minimax_pitch: v })} /></F>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('audio_format')}>
            <Select value={f.minimax_format} onValueChange={v => upd({ minimax_format: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="mp3">MP3</SelectItem><SelectItem value="wav">WAV</SelectItem><SelectItem value="pcm">PCM</SelectItem></SelectContent>
            </Select>
          </F>
          <F label={t('sample_rate')}><NumInput value={f.minimax_sample_rate} min={8000} max={48000} step={1000} onChange={v => upd({ minimax_sample_rate: v })} /></F>
        </div>
      </>}
      {p === 'openai' && <>
        <F label="API Key"><Input type="password" value={f.openai_api_key} onChange={e => upd({ openai_api_key: e.target.value })} placeholder={t('enter_api_key')} /></F>
        <F label="API URL"><Input value={f.openai_api_url} onChange={e => upd({ openai_api_url: e.target.value })} placeholder="https://api.openai.com/v1/audio/speech" /></F>
        <F label={t('model')}><Input value={f.openai_model} onChange={e => upd({ openai_model: e.target.value })} placeholder={t('enter_model_default_tts1')} /></F>
        <F label={t('voice_timbre')}><VoiceSelect value={f.openai_voice} options={voiceOptions} loading={voiceLoading} onChange={v => upd({ openai_voice: v })} placeholder={t('select_timbre')} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('response_format_label')}>
            <Select value={f.openai_response_format} onValueChange={v => upd({ openai_response_format: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>{['mp3','opus','aac','flac','wav','pcm'].map(v => <SelectItem key={v} value={v}>{v.toUpperCase()}</SelectItem>)}</SelectContent>
            </Select>
          </F>
          <F label={t('speech_rate')}><NumInput value={f.openai_speed} min={0.25} max={4} step={0.1} onChange={v => upd({ openai_speed: v })} /></F>
        </div>
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 cursor-pointer"><Switch checked={f.openai_stream} onCheckedChange={v => upd({ openai_stream: v })} /><span className="text-sm">{t('use_streaming')}</span></label>
          <F label={t('frame_duration')}><NumInput value={f.openai_frame_duration} min={1} max={1000} onChange={v => upd({ openai_frame_duration: v })} /></F>
        </div>
      </>}
      {(p === 'xunfei' || p === 'xunfei_super_tts') && <XunfeiFields f={f} upd={upd} t={t} voiceOptions={voiceOptions} voiceLoading={voiceLoading} />}
      {p === 'indextts_vllm' && <>
        <p className="text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-2)] rounded-lg p-3">{t('indextts_guide_subtitle')}</p>
        <F label="API URL"><Input value={f.indextts_api_url} onChange={e => upd({ indextts_api_url: e.target.value })} placeholder="http://127.0.0.1:7860" /></F>
        <F label="API Key"><Input type="password" value={f.indextts_api_key} onChange={e => upd({ indextts_api_key: e.target.value })} placeholder={t('optional_fill_as_needed') ?? 'optional'} /></F>
        <F label={t('model')}><Input value={f.indextts_model} onChange={e => upd({ indextts_model: e.target.value })} placeholder="indextts-vllm" /></F>
        <F label={t('voice_timbre')}><VoiceSelect value={f.indextts_voice} options={voiceOptions} loading={voiceLoading} onChange={v => upd({ indextts_voice: v })} placeholder={t('select_timbre')} /></F>
        <F label={t('frame_duration')}><NumInput value={f.indextts_frame_duration} min={1} max={1000} onChange={v => upd({ indextts_frame_duration: v })} /></F>
      </>}
      {p === 'cosyvoice' && <>
        <F label="API URL"><Input value={f.cosyvoice_api_url} onChange={e => upd({ cosyvoice_api_url: e.target.value })} placeholder={t('enter_api_url')} /></F>
        <F label={t('speaker_id')}><Input value={f.cosyvoice_spk_id} onChange={e => upd({ cosyvoice_spk_id: e.target.value })} placeholder={t('enter_speaker_id')} /></F>
        <div className="grid grid-cols-2 gap-3">
          <F label={t('frame_duration')}><NumInput value={f.cosyvoice_frame_duration} min={1} max={1000} onChange={v => upd({ cosyvoice_frame_duration: v })} /></F>
          <F label={t('target_sample_rate')}><NumInput value={f.cosyvoice_target_sr} min={8000} max={48000} onChange={v => upd({ cosyvoice_target_sr: v })} /></F>
        </div>
        <F label={t('audio_format')}>
          <Select value={f.cosyvoice_audio_format} onValueChange={v => upd({ cosyvoice_audio_format: v })}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent><SelectItem value="mp3">MP3</SelectItem><SelectItem value="wav">WAV</SelectItem><SelectItem value="pcm">PCM</SelectItem></SelectContent>
          </Select>
        </F>
        <F label={t('instruct_text')}><Input value={f.cosyvoice_instruct_text} onChange={e => upd({ cosyvoice_instruct_text: e.target.value })} placeholder={t('enter_instruct_text_opt')} /></F>
      </>}
      {p === 'supertonic' && <>
        <SupertonicModelSection f={f} upd={upd} t={t} />
        <div className="grid grid-cols-2 gap-3">
          <F label={t('voice_timbre')}>
            <Select value={f.supertonic_voice} onValueChange={v => upd({ supertonic_voice: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {SUPERTONIC_PRESET_VOICES.map(v => <SelectItem key={v} value={v}>{v}</SelectItem>)}
                <SelectItem value="custom">{t('custom_voice_json')}</SelectItem>
              </SelectContent>
            </Select>
          </F>
          <F label={t('language')}>
            <Select value={f.supertonic_lang} onValueChange={v => upd({ supertonic_lang: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="na">{t('auto_detect')}</SelectItem>
                {SUPERTONIC_LANGS.map(l => <SelectItem key={l.value} value={l.value}>{l.label}</SelectItem>)}
              </SelectContent>
            </Select>
          </F>
        </div>
        {f.supertonic_voice === 'custom' && (
          <F label={t('custom_voice_json_path')}>
            <Input value={f.supertonic_voice_json_path} onChange={e => upd({ supertonic_voice_json_path: e.target.value })} placeholder="/path/to/my-voice.json" />
          </F>
        )}
        <div className="grid grid-cols-3 gap-3">
          <F label={t('quality_steps')}>
            <NumInput value={f.supertonic_steps} min={5} max={12} onChange={v => upd({ supertonic_steps: v })} />
          </F>
          <F label={t('speech_speed')}>
            <NumInput value={f.supertonic_speed} min={0.7} max={2.0} step={0.1} onChange={v => upd({ supertonic_speed: v })} />
          </F>
          <F label={t('silence_duration_s')}>
            <NumInput value={f.supertonic_silence} min={0} max={2} step={0.1} onChange={v => upd({ supertonic_silence: v })} />
          </F>
        </div>
        <F label={t('frame_duration')}>
          <NumInput value={f.supertonic_frame_duration} min={20} max={100} onChange={v => upd({ supertonic_frame_duration: v })} />
        </F>
      </>}
    </>
  )
}

function XunfeiFields({ f, upd, t, voiceOptions, voiceLoading }: { f: TtsFields; upd: (p: Partial<TtsFields>) => void; t: T; voiceOptions: Array<{label:string;value:string}>; voiceLoading: boolean }) {
  const isSuperTts = f.provider === 'xunfei_super_tts'
  const prefix = isSuperTts ? 'xunfei_super_' : 'xunfei_'
  const get = (k: string) => (f as unknown as Record<string, unknown>)[prefix + k]
  const set = (k: string, v: unknown) => upd({ [`${prefix}${k}`]: v } as Partial<TtsFields>)
  return <>
    <F label={t('app_id')}><Input value={get('app_id') as string || ''} onChange={e => set('app_id', e.target.value)} placeholder={t('enter_xunfei_app_id')} /></F>
    <F label="API Key"><Input type="password" value={get('api_key') as string || ''} onChange={e => set('api_key', e.target.value)} placeholder={t('enter_xunfei_api_key')} /></F>
    <F label="API Secret"><Input type="password" value={get('api_secret') as string || ''} onChange={e => set('api_secret', e.target.value)} /></F>
    <F label="WebSocket URL"><Input value={get('ws_url') as string || ''} onChange={e => set('ws_url', e.target.value)} placeholder={isSuperTts ? 'wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6' : 'wss://tts-api.xfyun.cn/v2/tts'} /></F>
    <F label={t('voice_timbre')}>
      {isSuperTts ? <VoiceSelect value={get('voice') as string || ''} options={voiceOptions} loading={voiceLoading} onChange={v => set('voice', v)} placeholder={t('select_timbre')} />
        : <Input value={get('voice') as string || ''} onChange={e => set('voice', e.target.value)} placeholder={t('enter_voice_eg_xiaoyan')} />}
    </F>
    <div className="grid grid-cols-3 gap-3">
      <F label={t('speech_rate')}><NumInput value={get('speed') as number ?? 50} min={0} max={100} onChange={v => set('speed', v)} /></F>
      <F label={t('volume')}><NumInput value={get('volume') as number ?? 50} min={0} max={100} onChange={v => set('volume', v)} /></F>
      <F label={t('pitch')}><NumInput value={get('pitch') as number ?? 50} min={0} max={100} onChange={v => set('pitch', v)} /></F>
    </div>
    <div className="grid grid-cols-2 gap-3">
      <F label={t('audio_encoding_label')}>
        <Select value={get('audio_encoding') as string || 'raw'} onValueChange={v => set('audio_encoding', v)}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectItem value="raw">RAW</SelectItem><SelectItem value="opus">Opus</SelectItem></SelectContent>
        </Select>
      </F>
      <F label={t('sample_rate')}>
        <Select value={String(get('sample_rate') ?? (isSuperTts ? 24000 : 16000))} onValueChange={v => set('sample_rate', Number(v))}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{(isSuperTts ? ['8000','16000','24000'] : ['8000','16000']).map(v => <SelectItem key={v} value={v}>{v}</SelectItem>)}</SelectContent>
        </Select>
      </F>
    </div>
    <div className="grid grid-cols-2 gap-3">
      <F label={t('connection_timeout')}><NumInput value={get('connect_timeout') as number ?? 10} min={1} max={60} onChange={v => set('connect_timeout', v)} /></F>
      <F label={t('timeout_label')}><NumInput value={get('read_timeout') as number ?? 30} min={1} max={300} onChange={v => set('read_timeout', v)} /></F>
    </div>
    <F label={t('frame_duration')}><NumInput value={get('frame_duration') as number ?? 60} min={1} max={1000} onChange={v => set('frame_duration', v)} /></F>
  </>
}
