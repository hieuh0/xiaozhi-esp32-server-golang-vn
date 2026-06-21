import { useEffect, useState } from 'react'
import { useLocale } from '@/hooks/use-locale'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { ConfigRow, ConfigForm } from './config-list-page'

const VAD_PROVIDERS = [
  { value: 'ten_vad',    label: 'TEN-VAD' },
  { value: 'silero_vad', label: 'Silero VAD' },
  { value: 'webrtc_vad', label: 'WebRTC VAD' },
]

interface VadFields {
  provider: string; name: string; config_id: string; enabled: boolean; is_default: boolean
  // ten_vad
  ten_hop_size: number; ten_threshold: number; ten_pool_size: number; ten_acquire_timeout_ms: number
  // silero_vad
  sil_model_path: string; sil_threshold: number; sil_min_silence_ms: number
  sil_sample_rate: number; sil_channels: number; sil_acquire_timeout_ms: number
  // webrtc_vad
  wrtc_pool_min: number; wrtc_pool_max: number; wrtc_pool_max_idle: number
  wrtc_sample_rate: number; wrtc_mode: number
}

const D: VadFields = {
  provider: 'ten_vad', name: '', config_id: '', enabled: true, is_default: false,
  ten_hop_size: 512, ten_threshold: 0.4, ten_pool_size: 10, ten_acquire_timeout_ms: 3000,
  sil_model_path: 'config/models/vad/silero_vad.onnx', sil_threshold: 0.5,
  sil_min_silence_ms: 100, sil_sample_rate: 16000, sil_channels: 1, sil_acquire_timeout_ms: 3000,
  wrtc_pool_min: 5, wrtc_pool_max: 1000, wrtc_pool_max_idle: 100, wrtc_sample_rate: 16000, wrtc_mode: 2,
}

function serialize(f: VadFields): string {
  if (f.provider === 'ten_vad') {
    return JSON.stringify({ hop_size: f.ten_hop_size, threshold: f.ten_threshold, pool_size: f.ten_pool_size, acquire_timeout_ms: f.ten_acquire_timeout_ms })
  }
  if (f.provider === 'silero_vad') {
    return JSON.stringify({ model_path: f.sil_model_path, threshold: f.sil_threshold, min_silence_duration_ms: f.sil_min_silence_ms, sample_rate: f.sil_sample_rate, channels: f.sil_channels, acquire_timeout_ms: f.sil_acquire_timeout_ms })
  }
  // webrtc_vad
  return JSON.stringify({ pool_min_size: f.wrtc_pool_min, pool_max_size: f.wrtc_pool_max, pool_max_idle: f.wrtc_pool_max_idle, vad_sample_rate: f.wrtc_sample_rate, vad_mode: f.wrtc_mode })
}

function parse(row: ConfigRow | null): VadFields {
  if (!row) return { ...D }
  try {
    const d = JSON.parse(row.json_data || '{}')
    const provider = row.provider || 'ten_vad'
    return {
      ...D, provider, name: row.name, config_id: row.config_id,
      enabled: row.enabled !== false, is_default: !!row.is_default,
      // ten_vad
      ten_hop_size: d.hop_size ?? D.ten_hop_size, ten_threshold: d.threshold ?? D.ten_threshold,
      ten_pool_size: d.pool_size ?? D.ten_pool_size, ten_acquire_timeout_ms: d.acquire_timeout_ms ?? D.ten_acquire_timeout_ms,
      // silero_vad
      sil_model_path: d.model_path ?? D.sil_model_path, sil_threshold: d.threshold ?? D.sil_threshold,
      sil_min_silence_ms: d.min_silence_duration_ms ?? D.sil_min_silence_ms,
      sil_sample_rate: d.sample_rate ?? D.sil_sample_rate, sil_channels: d.channels ?? D.sil_channels,
      sil_acquire_timeout_ms: d.acquire_timeout_ms ?? D.sil_acquire_timeout_ms,
      // webrtc_vad
      wrtc_pool_min: d.pool_min_size ?? D.wrtc_pool_min, wrtc_pool_max: d.pool_max_size ?? D.wrtc_pool_max,
      wrtc_pool_max_idle: d.pool_max_idle ?? D.wrtc_pool_max_idle,
      wrtc_sample_rate: d.vad_sample_rate ?? D.wrtc_sample_rate, wrtc_mode: d.vad_mode ?? D.wrtc_mode,
    }
  } catch { return { ...D } }
}

const F = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="grid gap-1.5"><label className="text-sm font-medium text-[var(--color-text)]">{label}</label>{children}</div>
)

export function VadConfigForm({ form: _form, setForm, editing }: { form: ConfigForm; setForm: (p: Partial<ConfigForm>) => void; editing: ConfigRow | null }) {
  const { t } = useLocale()
  const [f, setF] = useState<VadFields>(() => parse(editing))

  useEffect(() => { setF(parse(editing)) }, [editing])

  const upd = (patch: Partial<VadFields>) => {
    const next = { ...f, ...patch }
    setF(next)
    setForm({ name: next.name, config_id: next.config_id, provider: next.provider, enabled: next.enabled, is_default: next.is_default, json_data: serialize(next) })
  }

  return (
    <div className="grid gap-3">
      <F label={t('provider')}>
        <Select value={f.provider} onValueChange={v => upd({ provider: v })}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>{VAD_PROVIDERS.map(p => <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>)}</SelectContent>
        </Select>
      </F>

      <div className="grid grid-cols-2 gap-3">
        <F label={t('config_name')}><Input value={f.name} onChange={e => upd({ name: e.target.value })} placeholder={t('enter_config_name')} /></F>
        <F label={t('config_id')}><Input value={f.config_id} onChange={e => upd({ config_id: e.target.value })} placeholder={t('enter_unique_config_id')} /></F>
      </div>

      {f.provider === 'ten_vad' && (
        <>
          <div className="grid grid-cols-2 gap-3">
            <F label="Hop Size"><Input type="number" value={f.ten_hop_size} onChange={e => upd({ ten_hop_size: Number(e.target.value) })} /></F>
            <F label={t('threshold')}><Input type="number" value={f.ten_threshold} step={0.1} min={0} max={1} onChange={e => upd({ ten_threshold: Number(e.target.value) })} /></F>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <F label="Pool Size"><Input type="number" value={f.ten_pool_size} onChange={e => upd({ ten_pool_size: Number(e.target.value) })} /></F>
            <F label="Acquire Timeout (ms)"><Input type="number" value={f.ten_acquire_timeout_ms} onChange={e => upd({ ten_acquire_timeout_ms: Number(e.target.value) })} /></F>
          </div>
        </>
      )}

      {f.provider === 'silero_vad' && (
        <>
          <F label="Model Path"><Input value={f.sil_model_path} onChange={e => upd({ sil_model_path: e.target.value })} /></F>
          <div className="grid grid-cols-2 gap-3">
            <F label={t('threshold')}><Input type="number" value={f.sil_threshold} step={0.1} min={0} max={1} onChange={e => upd({ sil_threshold: Number(e.target.value) })} /></F>
            <F label="Min Silence (ms)"><Input type="number" value={f.sil_min_silence_ms} onChange={e => upd({ sil_min_silence_ms: Number(e.target.value) })} /></F>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <F label="Sample Rate"><Input type="number" value={f.sil_sample_rate} onChange={e => upd({ sil_sample_rate: Number(e.target.value) })} /></F>
            <F label="Channels"><Input type="number" value={f.sil_channels} min={1} max={2} onChange={e => upd({ sil_channels: Number(e.target.value) })} /></F>
            <F label="Acquire Timeout (ms)"><Input type="number" value={f.sil_acquire_timeout_ms} onChange={e => upd({ sil_acquire_timeout_ms: Number(e.target.value) })} /></F>
          </div>
        </>
      )}

      {f.provider === 'webrtc_vad' && (
        <>
          <div className="grid grid-cols-3 gap-3">
            <F label="Pool Min"><Input type="number" value={f.wrtc_pool_min} onChange={e => upd({ wrtc_pool_min: Number(e.target.value) })} /></F>
            <F label="Pool Max"><Input type="number" value={f.wrtc_pool_max} onChange={e => upd({ wrtc_pool_max: Number(e.target.value) })} /></F>
            <F label="Pool Max Idle"><Input type="number" value={f.wrtc_pool_max_idle} onChange={e => upd({ wrtc_pool_max_idle: Number(e.target.value) })} /></F>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <F label="Sample Rate"><Input type="number" value={f.wrtc_sample_rate} onChange={e => upd({ wrtc_sample_rate: Number(e.target.value) })} /></F>
            <F label="VAD Mode (0-3)">
              <Select value={String(f.wrtc_mode)} onValueChange={v => upd({ wrtc_mode: Number(v) })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="0">0 — Least Aggressive</SelectItem>
                  <SelectItem value="1">1 — Low</SelectItem>
                  <SelectItem value="2">2 — Medium</SelectItem>
                  <SelectItem value="3">3 — Most Aggressive</SelectItem>
                </SelectContent>
              </Select>
            </F>
          </div>
        </>
      )}

      <div className="flex items-center gap-6 pt-1">
        <label className="flex items-center gap-2 cursor-pointer">
          <Switch checked={f.enabled} onCheckedChange={v => upd({ enabled: v })} /><span className="text-sm">{t('enabled_status')}</span>
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <Switch checked={f.is_default} onCheckedChange={v => upd({ is_default: v })} /><span className="text-sm">{t('default_config')}</span>
        </label>
      </div>
    </div>
  )
}
