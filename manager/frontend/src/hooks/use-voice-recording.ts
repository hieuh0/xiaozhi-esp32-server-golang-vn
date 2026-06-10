import { useRef, useState } from 'react'

export const getAudioDurationSeconds = (file: File | Blob): Promise<number> =>
  new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const audio = new Audio()
    audio.preload = 'metadata'
    audio.onloadedmetadata = () => {
      const d = audio.duration
      URL.revokeObjectURL(url)
      if (!Number.isFinite(d) || d <= 0) { reject(new Error('read_audio_duration_failed')); return }
      resolve(d)
    }
    audio.onerror = () => { URL.revokeObjectURL(url); reject(new Error('parse_audio_failed')) }
    audio.src = url
  })

const audioBufferToWav = (buffer: AudioBuffer): ArrayBuffer => {
  const { length, numberOfChannels, sampleRate } = buffer
  const bytesPerSample = 2
  const blockAlign = numberOfChannels * bytesPerSample
  const byteRate = sampleRate * blockAlign
  const dataSize = length * blockAlign
  const bufferSize = 44 + dataSize
  const ab = new ArrayBuffer(bufferSize)
  const view = new DataView(ab)
  const ws = (offset: number, s: string) => { for (let i = 0; i < s.length; i++) view.setUint8(offset + i, s.charCodeAt(i)) }
  ws(0, 'RIFF'); view.setUint32(4, bufferSize - 8, true); ws(8, 'WAVE'); ws(12, 'fmt ')
  view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, numberOfChannels, true)
  view.setUint32(24, sampleRate, true); view.setUint32(28, byteRate, true)
  view.setUint16(32, blockAlign, true); view.setUint16(34, 16, true); ws(36, 'data'); view.setUint32(40, dataSize, true)
  let offset = 44
  for (let i = 0; i < length; i++) {
    for (let ch = 0; ch < numberOfChannels; ch++) {
      const s = Math.max(-1, Math.min(1, buffer.getChannelData(ch)[i]))
      view.setInt16(offset, s < 0 ? s * 0x8000 : s * 0x7FFF, true); offset += 2
    }
  }
  return ab
}

const convertToWav = async (blob: Blob): Promise<Blob> => {
  const arrayBuffer = await blob.arrayBuffer()
  const ctx = new AudioContext()
  try {
    const audioBuffer = await ctx.decodeAudioData(arrayBuffer)
    return new Blob([audioBufferToWav(audioBuffer)], { type: 'audio/wav' })
  } finally {
    await ctx.close()
  }
}

export interface UseVoiceRecordingOptions {
  onDurationCheck?: (durationSec: number) => boolean
  onError?: (msg: string) => void
}

export function useVoiceRecording(opts?: UseVoiceRecordingOptions) {
  const [isRecording, setIsRecording] = useState(false)
  const [recordBlob, setRecordBlob] = useState<Blob | null>(null)
  const [recordPreviewUrl, setRecordPreviewUrl] = useState('')
  const [durationSec, setDurationSec] = useState(0)
  const recorderRef = useRef<MediaRecorder | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const chunksRef = useRef<BlobPart[]>([])
  const optsRef = useRef(opts)
  optsRef.current = opts

  const reset = () => {
    setRecordBlob(null)
    setRecordPreviewUrl(prev => { if (prev) URL.revokeObjectURL(prev); return '' })
    setDurationSec(0)
    setIsRecording(false)
  }

  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      streamRef.current = stream
      chunksRef.current = []
      const mimeType = 'audio/webm;codecs=opus'
      const recorder = MediaRecorder.isTypeSupported(mimeType)
        ? new MediaRecorder(stream, { mimeType })
        : new MediaRecorder(stream)
      recorderRef.current = recorder
      recorder.ondataavailable = (e) => { if (e.data?.size > 0) chunksRef.current.push(e.data) }
      recorder.onstop = async () => {
        try {
          const raw = new Blob(chunksRef.current, { type: (chunksRef.current[0] as Blob)?.type || 'audio/webm' })
          const wav = await convertToWav(raw)
          const duration = await getAudioDurationSeconds(wav)
          if (optsRef.current?.onDurationCheck && !optsRef.current.onDurationCheck(duration)) return
          setRecordBlob(wav)
          setDurationSec(duration)
          setRecordPreviewUrl(prev => { if (prev) URL.revokeObjectURL(prev); return URL.createObjectURL(wav) })
        } catch (err) {
          optsRef.current?.onError?.((err as Error).message || 'Recording convert failed')
        } finally {
          streamRef.current?.getTracks().forEach(t => t.stop())
          streamRef.current = null
        }
      }
      recorder.start()
      setIsRecording(true)
    } catch (err) {
      optsRef.current?.onError?.((err as Error).message || 'Recording failed')
    }
  }

  const stopRecording = () => { recorderRef.current?.stop(); setIsRecording(false) }

  return { isRecording, recordBlob, recordPreviewUrl, durationSec, startRecording, stopRecording, reset }
}
