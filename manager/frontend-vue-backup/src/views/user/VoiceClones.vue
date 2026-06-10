<script setup>
import { computed, nextTick, ref, onBeforeUnmount, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../utils/api'
import { useAuthStore } from '../../stores/auth'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

const { t } = useLocale()
const authStore = useAuthStore()
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const submitting = ref(false)
const createDialogVisible = ref(false)
const audioDialogVisible = ref(false)
const editDialogVisible = ref(false)
const voiceClones = ref([])
const currentAudios = ref([])
const ttsConfigs = ref([])
const MIN_AUDIO_DURATION_SECONDS = 10
const cloneEnabledProviders = ['doubao', 'doubao_ws', 'minimax', 'cosyvoice', 'aliyun_qwen', 'indextts_vllm']
const pendingStatuses = ['queued', 'processing']
let clonePollingTimer = null
const clonePollingBusy = ref(false)
const editSubmitting = ref(false)
const retrySubmittingID = ref(null)
const previewUploadSubmittingID = ref(null)
const previewClonedSubmittingID = ref(null)
const appendAudioSubmittingID = ref(null)
const shareSubmittingID = ref(null)
const deleteSubmittingID = ref(null)
const appendAudioInputRef = ref(null)
const appendAudioTargetClone = ref(null)
const previewPlayerVisible = ref(false)
const previewPlayerRef = ref(null)
const previewPlayerURL = ref('')
const previewPlayerSourceLabel = ref('')
const previewPlayerCloneLabel = ref('')
const previewPlayerPlaying = ref(false)
const previewPlayerCurrentTime = ref(0)
const previewPlayerDuration = ref(0)

const form = ref({ name: '', tts_config_id: '', source_type: 'upload', transcript: '', transcript_lang: 'zh-CN', audioFile: null, recordBlob: null, audioDurationSec: 0 })
const editForm = ref({ id: null, originalName: '', name: '', provider: '', ttsConfigDisplay: '', providerVoiceID: '', statusText: '', createdAtText: '', lastError: '' })
const capability = ref({ enabled: true, requires_transcript: false, min_text_len: 0, max_text_len: 0 })

const cloneEnabledConfigs = computed(() => ttsConfigs.value.filter(item => cloneEnabledProviders.includes(item.provider)))
const selectedCloneConfig = computed(() => cloneEnabledConfigs.value.find(item => item.config_id === form.value.tts_config_id) || null)
const currentCloneProvider = computed(() => selectedCloneConfig.value?.provider || '')
const normalizeProvider = (provider) => String(provider || '').trim().toLowerCase()

const resolveChargeNotice = (provider, scene = 'create') => {
  const normalized = normalizeProvider(provider)
  if (normalized === 'aliyun_qwen') return { message: scene === 'create' ? t('billing_qwen_per_voice') : t('billing_qwen_confirm'), type: 'warning' }
  if (normalized === 'minimax') return { message: scene === 'create' ? t('billing_minimax_first_fee') : t('billing_minimax_first_preview'), type: 'warning' }
  if (normalized === 'cosyvoice') return { message: scene === 'create' ? t('billing_cosyvoice_free') : t('billing_cosyvoice_confirm'), type: 'info' }
  return { message: '', type: 'info' }
}

const createChargeNotice = computed(() => resolveChargeNotice(currentCloneProvider.value, 'create'))
const requiresMinimaxDuration = computed(() => currentCloneProvider.value === 'minimax')
const isAliyunQwenProvider = computed(() => currentCloneProvider.value === 'aliyun_qwen')
const qwenCloneRuntimeModel = 'qwen3-tts-vc-2026-01-22'
const uploadAcceptTypes = computed(() => isAliyunQwenProvider.value ? '.wav,.mp3,.m4a,audio/wav,audio/wave,audio/mpeg,audio/mp4,audio/x-m4a' : '.wav,audio/wav,audio/wave')
const audioRequirementText = computed(() => {
  if (requiresMinimaxDuration.value) return t('wav_min_dur_require', { n: MIN_AUDIO_DURATION_SECONDS })
  if (isAliyunQwenProvider.value) return t('audio_duration_requirement')
  return t('wav_requirement')
})

const isRecording = ref(false)
const mediaRecorder = ref(null)
const recordChunks = ref([])
const recordPreviewUrl = ref('')
const { formatDate } = useFormatDate()

const totalPages = () => Math.ceil(total.value / pageSize.value) || 1

const badgeClass = (type) => {
  const map = {
    gray: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700',
    blue: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800',
    green: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800',
    yellow: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800',
    red: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800',
  }
  return map[type] || map.gray
}

const parseMetaJSON = (metaJSON) => { try { return JSON.parse(metaJSON || '') } catch { return {} } }
const normalizeCloneStatus = (row) => {
  const status = String(row?.status || '').trim().toLowerCase()
  const taskStatus = String(row?.task_status || '').trim().toLowerCase()
  if (status === 'failed' || taskStatus === 'failed') return 'failed'
  if (status === 'active' || taskStatus === 'succeeded') return 'active'
  if (taskStatus === 'queued' || taskStatus === 'processing') return taskStatus
  if (status === 'queued' || status === 'processing') return status
  return status || taskStatus || 'unknown'
}
const formatCloneStatus = (row) => {
  const status = normalizeCloneStatus(row)
  if (status === 'queued') return t('queuing')
  if (status === 'processing') return t('processing')
  if (status === 'active') return t('success')
  if (status === 'failed') return t('failed')
  return t('unknown')
}
const getCloneStatusBadge = (row) => {
  const status = normalizeCloneStatus(row)
  if (status === 'queued') return badgeClass('blue')
  if (status === 'processing') return badgeClass('yellow')
  if (status === 'active') return badgeClass('green')
  if (status === 'failed') return badgeClass('red')
  return badgeClass('gray')
}
const getCloneLastError = (row) => {
  const status = normalizeCloneStatus(row)
  if (status !== 'failed') return '-'
  if (row?.task_last_error) return row.task_last_error
  return parseMetaJSON(row?.meta_json).last_error || '-'
}
const canRetryClone = (row) => normalizeCloneStatus(row) === 'failed'
const canPreviewClonedVoice = (row) => normalizeCloneStatus(row) === 'active'
const canAppendRefAudio = (row) => normalizeCloneStatus(row) === 'active' && normalizeProvider(row?.provider) === 'indextts_vllm'

const formatPlayerTime = (seconds) => {
  const value = Number(seconds || 0)
  if (!Number.isFinite(value) || value < 0) return '00:00'
  const total = Math.floor(value)
  return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
}

const pauseAllOtherAudios = () => {
  const current = previewPlayerRef.value
  document.querySelectorAll('audio').forEach(audioEl => { if (audioEl !== current) { try { audioEl.pause() } catch {} } })
}
const revokePreviewPlayerURL = () => { if (previewPlayerURL.value) { URL.revokeObjectURL(previewPlayerURL.value); previewPlayerURL.value = '' } }
const stopPreviewPlayback = () => {
  const audioEl = previewPlayerRef.value
  if (!audioEl) return
  audioEl.pause(); audioEl.currentTime = 0; previewPlayerCurrentTime.value = 0
}
const closePreviewPlayerDialog = () => {
  stopPreviewPlayback(); previewPlayerPlaying.value = false
  previewPlayerCurrentTime.value = 0; previewPlayerDuration.value = 0
  previewPlayerSourceLabel.value = ''; previewPlayerCloneLabel.value = ''
  revokePreviewPlayerURL()
}
const setPreviewPlayerSource = async (blob, sourceLabel, cloneLabel) => {
  stopPreviewPlayback(); revokePreviewPlayerURL()
  previewPlayerURL.value = URL.createObjectURL(blob)
  previewPlayerSourceLabel.value = sourceLabel; previewPlayerCloneLabel.value = cloneLabel
  previewPlayerCurrentTime.value = 0; previewPlayerDuration.value = 0
  previewPlayerVisible.value = true
  await nextTick(); pauseAllOtherAudios()
  const audioEl = previewPlayerRef.value
  if (!audioEl) return
  try { await audioEl.play() } catch { ElMessage.info(t('audio_loaded_click_play')) }
}
const togglePreviewPlayback = async () => {
  const audioEl = previewPlayerRef.value
  if (!audioEl) return
  if (audioEl.paused) { pauseAllOtherAudios(); await audioEl.play() } else { audioEl.pause() }
}
const onPreviewAudioPlay = () => { pauseAllOtherAudios(); previewPlayerPlaying.value = true }
const onPreviewAudioPause = () => { previewPlayerPlaying.value = false }
const onPreviewAudioEnded = () => { previewPlayerPlaying.value = false; previewPlayerCurrentTime.value = previewPlayerDuration.value }
const onPreviewAudioTimeUpdate = () => { const audioEl = previewPlayerRef.value; if (audioEl) previewPlayerCurrentTime.value = Number(audioEl.currentTime || 0) }
const onPreviewAudioLoadedMetadata = () => { const audioEl = previewPlayerRef.value; if (audioEl) previewPlayerDuration.value = Number(audioEl.duration || 0) }

const hasPendingCloneTask = (row) => pendingStatuses.includes(normalizeCloneStatus(row))
const clearClonePollingTimer = () => { if (clonePollingTimer) { window.clearTimeout(clonePollingTimer); clonePollingTimer = null } }
const scheduleClonePolling = () => {
  if (clonePollingTimer) return
  clonePollingTimer = window.setTimeout(async () => {
    clonePollingTimer = null
    if (!voiceClones.value.some(hasPendingCloneTask)) return
    if (clonePollingBusy.value) { scheduleClonePolling(); return }
    clonePollingBusy.value = true
    try { await loadVoiceClones(true) } finally {
      clonePollingBusy.value = false
      if (voiceClones.value.some(hasPendingCloneTask)) scheduleClonePolling()
    }
  }, 2000)
}

const loadVoiceClones = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const res = await api.get('/user/voice-clones', { params: { page: page.value, page_size: pageSize.value } })
    voiceClones.value = res.data.data || []; total.value = res.data.total || 0
  } finally {
    if (!silent) loading.value = false
    if (voiceClones.value.some(hasPendingCloneTask)) scheduleClonePolling()
    else clearClonePollingTimer()
  }
}

const loadTtsConfigs = async () => { const res = await api.get('/user/tts-configs'); ttsConfigs.value = res.data.data || [] }

const openCreateDialog = async () => {
  createDialogVisible.value = true
  await loadTtsConfigs()
  if (!cloneEnabledConfigs.value.length) { form.value.tts_config_id = ''; return }
  if (!cloneEnabledConfigs.value.find(item => item.config_id === form.value.tts_config_id)) {
    form.value.tts_config_id = cloneEnabledConfigs.value[0].config_id
  }
  await onConfigChange(form.value.tts_config_id)
}

const onConfigChange = async (configId) => {
  const cfg = cloneEnabledConfigs.value.find(item => item.config_id === configId)
  if (!cfg) { capability.value = { enabled: true, requires_transcript: false, min_text_len: 0, max_text_len: 0 }; return }
  const res = await api.get('/user/voice-clone/capabilities', { params: { provider: cfg.provider } })
  capability.value = res.data.data || capability.value
}

const isWavFile = (file) => { const n = (file?.name || '').toLowerCase(); const t = (file?.type || '').toLowerCase(); return t.includes('audio/wav') || t.includes('audio/wave') || n.endsWith('.wav') }
const isSupportedAliyunQwenAudio = (file) => { const n = (file?.name || '').toLowerCase(); const t = (file?.type || '').toLowerCase(); return n.endsWith('.wav') || n.endsWith('.mp3') || n.endsWith('.m4a') || t.includes('audio/wav') || t.includes('audio/wave') || t.includes('audio/mpeg') || t.includes('audio/mp4') || t.includes('audio/x-m4a') }
const isSupportedUploadAudio = (file) => isAliyunQwenProvider.value ? isSupportedAliyunQwenAudio(file) : isWavFile(file)

const getAudioDurationSeconds = (blobOrFile) => new Promise((resolve, reject) => {
  const url = URL.createObjectURL(blobOrFile)
  const audio = new Audio()
  audio.preload = 'metadata'
  audio.onloadedmetadata = () => { const d = Number(audio.duration || 0); URL.revokeObjectURL(url); if (!Number.isFinite(d) || d <= 0) reject(new Error(t('read_audio_duration_failed'))); else resolve(d) }
  audio.onerror = () => { URL.revokeObjectURL(url); reject(new Error(t('parse_audio_failed'))) }
  audio.src = url
})

const handleFileChange = async (event) => {
  const file = event.target.files?.[0] || null
  if (!file) { form.value.audioFile = null; form.value.audioDurationSec = 0; return }
  if (!isSupportedUploadAudio(file)) { ElMessage.warning(isAliyunQwenProvider.value ? t('wav_mp3_m4a_only') : t('wav_only')); form.value.audioFile = null; form.value.audioDurationSec = 0; event.target.value = ''; return }
  if (!requiresMinimaxDuration.value) { form.value.audioFile = file; form.value.audioDurationSec = 0; return }
  try {
    const duration = await getAudioDurationSeconds(file)
    if (duration < MIN_AUDIO_DURATION_SECONDS) { ElMessage.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: duration.toFixed(2) })); form.value.audioFile = null; form.value.audioDurationSec = 0; event.target.value = ''; return }
    form.value.audioFile = file; form.value.audioDurationSec = duration
  } catch (error) { ElMessage.warning(error.message || t('read_audio_duration_fail')); form.value.audioFile = null; form.value.audioDurationSec = 0; event.target.value = '' }
}

const convertToWav = async (blob) => {
  const arrayBuffer = await blob.arrayBuffer()
  const audioContext = new (window.AudioContext || window.webkitAudioContext)()
  try { const audioBuffer = await audioContext.decodeAudioData(arrayBuffer); return new Blob([audioBufferToWav(audioBuffer)], { type: 'audio/wav' }) } finally { await audioContext.close() }
}

const audioBufferToWav = (buffer) => {
  const length = buffer.length, numberOfChannels = buffer.numberOfChannels, sampleRate = buffer.sampleRate
  const bytesPerSample = 2, blockAlign = numberOfChannels * bytesPerSample, byteRate = sampleRate * blockAlign
  const dataSize = length * blockAlign, bufferSize = 44 + dataSize
  const arrayBuffer = new ArrayBuffer(bufferSize), view = new DataView(arrayBuffer)
  const writeString = (offset, str) => { for (let i = 0; i < str.length; i++) view.setUint8(offset + i, str.charCodeAt(i)) }
  writeString(0, 'RIFF'); view.setUint32(4, bufferSize - 8, true); writeString(8, 'WAVE'); writeString(12, 'fmt ')
  view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, numberOfChannels, true)
  view.setUint32(24, sampleRate, true); view.setUint32(28, byteRate, true); view.setUint16(32, blockAlign, true)
  view.setUint16(34, 16, true); writeString(36, 'data'); view.setUint32(40, dataSize, true)
  let offset = 44
  for (let i = 0; i < length; i++) { for (let channel = 0; channel < numberOfChannels; channel++) { const sample = Math.max(-1, Math.min(1, buffer.getChannelData(channel)[i])); view.setInt16(offset, sample < 0 ? sample * 0x8000 : sample * 0x7FFF, true); offset += 2 } }
  return arrayBuffer
}

const startRecording = async () => {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  recordChunks.value = []; form.value.audioDurationSec = 0
  const options = { mimeType: 'audio/webm;codecs=opus' }
  const recorder = MediaRecorder.isTypeSupported(options.mimeType) ? new MediaRecorder(stream, options) : new MediaRecorder(stream)
  mediaRecorder.value = recorder
  recorder.ondataavailable = (evt) => { if (evt.data?.size > 0) recordChunks.value.push(evt.data) }
  recorder.onstop = async () => {
    const blob = new Blob(recordChunks.value, { type: recordChunks.value[0]?.type || 'audio/webm' })
    try {
      const wavBlob = await convertToWav(blob)
      const duration = await getAudioDurationSeconds(wavBlob)
      if (requiresMinimaxDuration.value && duration < MIN_AUDIO_DURATION_SECONDS) {
        ElMessage.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: duration.toFixed(2) }))
        form.value.recordBlob = null; form.value.audioDurationSec = 0
        if (recordPreviewUrl.value) { URL.revokeObjectURL(recordPreviewUrl.value); recordPreviewUrl.value = '' }
      } else {
        form.value.recordBlob = wavBlob; form.value.audioDurationSec = duration
        if (recordPreviewUrl.value) URL.revokeObjectURL(recordPreviewUrl.value)
        recordPreviewUrl.value = URL.createObjectURL(wavBlob)
      }
    } catch { ElMessage.error(t('recording_convert_failed')); form.value.recordBlob = null; form.value.audioDurationSec = 0; if (recordPreviewUrl.value) { URL.revokeObjectURL(recordPreviewUrl.value); recordPreviewUrl.value = '' } }
    stream.getTracks().forEach(track => track.stop())
  }
  recorder.start(); isRecording.value = true
}

const stopRecording = () => { if (mediaRecorder.value) mediaRecorder.value.stop(); isRecording.value = false }

const submitClone = async () => {
  if (!form.value.tts_config_id) { ElMessage.warning(t('select_clonable_tts')); return }
  const createNotice = resolveChargeNotice(currentCloneProvider.value, 'create')
  if (createNotice.message) {
    try { await ElMessageBox.confirm(createNotice.message, t('create_clone_reminder'), { confirmButtonText: t('i_understand_continue'), cancelButtonText: t('cancel'), type: createNotice.type }) } catch { return }
  }
  if (capability.value.requires_transcript && !form.value.transcript.trim()) { ElMessage.warning(t('provider_requires_audio_text')); return }
  const fd = new FormData()
  fd.append('name', form.value.name); fd.append('tts_config_id', form.value.tts_config_id)
  fd.append('source_type', form.value.source_type); fd.append('transcript', form.value.transcript); fd.append('transcript_lang', form.value.transcript_lang)
  if (form.value.source_type === 'upload') {
    if (!form.value.audioFile) { ElMessage.warning(t('upload_audio_file')); return }
    let duration = form.value.audioDurationSec
    if (requiresMinimaxDuration.value && !duration) { try { duration = await getAudioDurationSeconds(form.value.audioFile) } catch (error) { ElMessage.warning(error.message || t('read_audio_duration_fail')); return } }
    if (requiresMinimaxDuration.value && duration < MIN_AUDIO_DURATION_SECONDS) { ElMessage.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: duration.toFixed(2) })); return }
    fd.append('audio_file', form.value.audioFile)
  } else {
    if (!form.value.recordBlob) { ElMessage.warning(t('record_first')); return }
    let duration = form.value.audioDurationSec
    if (requiresMinimaxDuration.value && !duration) { try { duration = await getAudioDurationSeconds(form.value.recordBlob) } catch (error) { ElMessage.warning(error.message || t('read_duration_failed')); return } }
    if (requiresMinimaxDuration.value && duration < MIN_AUDIO_DURATION_SECONDS) { ElMessage.warning(t('audio_min_dur_warning', { min: MIN_AUDIO_DURATION_SECONDS, cur: duration.toFixed(2) })); return }
    fd.append('audio_blob', form.value.recordBlob, `recording_${Date.now()}.wav`)
  }
  submitting.value = true
  try {
    const res = await api.post('/user/voice-clones', fd, { timeout: 120000 })
    const queued = res.status === 202 || pendingStatuses.includes(normalizeCloneStatus(res.data?.data || {}))
    ElMessage.success(queued ? t('clone_task_submitted') : t('clone_voice_created'))
    createDialogVisible.value = false; await loadVoiceClones()
  } finally { submitting.value = false }
}

const loadAudios = async (clone) => { const res = await api.get(`/user/voice-clones/${clone.id}/audios`); currentAudios.value = res.data.data || []; audioDialogVisible.value = true }

const openEditDialog = (clone) => {
  if (!clone) return
  editForm.value = { id: clone.id, originalName: String(clone.name || ''), name: String(clone.name || ''), provider: String(clone.provider || '-'), ttsConfigDisplay: `${clone.tts_config_name || '-'} (${clone.tts_config_id || '-'})`, providerVoiceID: String(clone.provider_voice_id || '-'), statusText: formatCloneStatus(clone), createdAtText: formatDate(clone.created_at), lastError: String(getCloneLastError(clone) === '-' ? '' : getCloneLastError(clone)) }
  editDialogVisible.value = true
}

const resetEditForm = () => { editForm.value = { id: null, originalName: '', name: '', provider: '', ttsConfigDisplay: '', providerVoiceID: '', statusText: '', createdAtText: '', lastError: '' }; editSubmitting.value = false }

const submitEditClone = async () => {
  const cloneID = editForm.value.id; if (!cloneID) return
  const nextName = String(editForm.value.name || '').trim()
  if (!nextName) { ElMessage.warning(t('name_required')); return }
  if ([...nextName].length > 100) { ElMessage.warning(t('name_max_length')); return }
  if (nextName === String(editForm.value.originalName || '').trim()) { editDialogVisible.value = false; return }
  editSubmitting.value = true
  try { await api.put(`/user/voice-clones/${cloneID}`, { name: nextName }); ElMessage.success(t('name_update_success')); editDialogVisible.value = false; await loadVoiceClones(true) } finally { editSubmitting.value = false }
}

const retryClone = async (clone) => {
  if (!clone?.id || !canRetryClone(clone) || retrySubmittingID.value) return
  retrySubmittingID.value = clone.id
  try { await api.post(`/user/voice-clones/${clone.id}/retry`); ElMessage.success(t('reclone_task_submitted')); await loadVoiceClones(true) } finally { retrySubmittingID.value = null }
}

const toggleSharedToAll = async (clone, nextValue) => {
  if (!authStore.isAdmin || !clone?.id) return
  shareSubmittingID.value = clone.id
  try { await api.put(`/user/voice-clones/${clone.id}`, { shared_to_all: !!nextValue }); clone.shared_to_all = !!nextValue; ElMessage.success(nextValue ? t('enabled_for_all') : t('sharing_closed')) } finally { shareSubmittingID.value = null }
}

const deleteClone = async (clone) => {
  if (!clone?.id || deleteSubmittingID.value) return
  try { await ElMessageBox.confirm(t('confirm_delete_clone_voice', { name: clone.name || clone.provider_voice_id || clone.id }), t('delete_cloned_voice'), { type: 'warning', confirmButtonText: t('delete'), cancelButtonText: t('cancel') }) } catch { return }
  deleteSubmittingID.value = clone.id
  try { await api.delete(`/user/voice-clones/${clone.id}`); ElMessage.success(t('delete_success')); await loadVoiceClones(true) } finally { deleteSubmittingID.value = null }
}

const openAppendAudioDialog = (clone) => {
  if (!clone?.id || !canAppendRefAudio(clone) || appendAudioSubmittingID.value) return
  appendAudioTargetClone.value = clone
  const input = appendAudioInputRef.value
  if (!input) { ElMessage.error(t('file_picker_not_ready')); return }
  input.value = ''; input.click()
}

const handleAppendAudioFileChange = async (event) => {
  const file = event?.target?.files?.[0]; const clone = appendAudioTargetClone.value
  if (!file || !clone?.id) { appendAudioTargetClone.value = null; return }
  appendAudioSubmittingID.value = clone.id
  try { const fd = new FormData(); fd.append('source_type', 'upload'); fd.append('audio_file', file); await api.post(`/user/voice-clones/${clone.id}/append-audio`, fd, { timeout: 120000 }); ElMessage.success(t('append_ref_audio_success')); await loadVoiceClones(true) } catch (error) { ElMessage.error(error?.response?.data?.error || t('append_ref_audio_failed')) } finally { appendAudioSubmittingID.value = null; appendAudioTargetClone.value = null; if (event?.target) event.target.value = '' }
}

const playAudio = async (audio) => {
  const response = await api.get(`/user/voice-clones/audios/${audio.id}/file`, { responseType: 'blob' })
  await setPreviewPlayerSource(response.data, t('original_audio'), String(audio?.file_name || '') || t('clone_original_audio'))
}

const previewUploadedAudio = async (clone) => {
  if (!clone?.id || previewUploadSubmittingID.value) return
  previewUploadSubmittingID.value = clone.id
  try {
    const res = await api.get(`/user/voice-clones/${clone.id}/audios`); const audios = res.data.data || []
    if (!audios.length) { ElMessage.warning(t('no_uploaded_audio')); return }
    const audioRes = await api.get(`/user/voice-clones/audios/${audios[0].id}/file`, { responseType: 'blob' })
    await setPreviewPlayerSource(audioRes.data, t('original_audio'), String(clone?.name || t('clone_task')))
  } catch (error) { ElMessage.error(error?.response?.data?.error || t('preview_upload_audio_failed')) } finally { previewUploadSubmittingID.value = null }
}

const previewClonedVoice = async (clone) => {
  if (!clone?.id || !canPreviewClonedVoice(clone) || previewClonedSubmittingID.value) return
  const previewNotice = resolveChargeNotice(clone?.provider, 'preview')
  if (previewNotice.message) { try { await ElMessageBox.confirm(previewNotice.message, t('preview_clone_reminder'), { confirmButtonText: t('continue_preview'), cancelButtonText: t('cancel'), type: previewNotice.type }) } catch { return } }
  previewClonedSubmittingID.value = clone.id
  try { const response = await api.get(`/user/voice-clones/${clone.id}/preview`, { responseType: 'blob' }); await setPreviewPlayerSource(response.data, t('preview_clone'), String(clone?.name || t('clone_task'))) } catch (error) { ElMessage.error(error?.response?.data?.error || t('preview_clone_audio_failed')) } finally { previewClonedSubmittingID.value = null }
}

onMounted(loadVoiceClones)
onBeforeUnmount(() => { clearClonePollingTimer(); closePreviewPlayerDialog() })
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex justify-end">
      <Button @click="openCreateDialog">{{ t('create_clone_voice') }}</Button>
    </div>

    <!-- Hidden append audio input -->
    <input ref="appendAudioInputRef" type="file" :accept="uploadAcceptTypes" class="hidden" @change="handleAppendAudioFileChange" />

    <!-- Main table -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
      <div class="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('name') }}</TableHead>
              <TableHead class="w-24">{{ t('provider') }}</TableHead>
              <TableHead>{{ t('tts_config_label') }}</TableHead>
              <TableHead>{{ t('clone_voice_id') }}</TableHead>
              <TableHead v-if="authStore.isAdmin" class="w-32 text-center">{{ t('share_to_all_col') }}</TableHead>
              <TableHead class="w-24">{{ t('task_status') }}</TableHead>
              <TableHead>{{ t('failure_reason') }}</TableHead>
              <TableHead class="w-40">{{ t('created_at') }}</TableHead>
              <TableHead>{{ t('actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell :colspan="authStore.isAdmin ? 9 : 8" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableEmpty v-if="!voiceClones.length" />
              <TableRow v-for="row in voiceClones" :key="row.id">
                <TableCell class="font-medium text-sm">{{ row.name }}</TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.provider }}</TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)] max-w-[180px] truncate" :title="`${row.tts_config_name || '-'} (${row.tts_config_id || '-'})`">
                  {{ `${row.tts_config_name || '-'} (${row.tts_config_id || '-'})` }}
                </TableCell>
                <TableCell class="text-xs font-mono text-[var(--color-text-secondary)] max-w-[160px] truncate" :title="row.provider_voice_id">{{ row.provider_voice_id }}</TableCell>
                <TableCell v-if="authStore.isAdmin" class="text-center">
                  <Switch
                    :model-value="!!row.shared_to_all"
                    :disabled="normalizeCloneStatus(row) !== 'active' || shareSubmittingID === row.id"
                    @update:model-value="(val) => toggleSharedToAll(row, val)"
                  />
                </TableCell>
                <TableCell><span :class="getCloneStatusBadge(row)">{{ formatCloneStatus(row) }}</span></TableCell>
                <TableCell class="text-xs text-[var(--color-text-secondary)] max-w-[140px] truncate" :title="getCloneLastError(row)">{{ getCloneLastError(row) }}</TableCell>
                <TableCell class="text-xs text-[var(--color-text-secondary)]">{{ formatDate(row.created_at) }}</TableCell>
                <TableCell>
                  <div class="flex flex-wrap gap-1">
                    <Button variant="outline" size="sm" :disabled="previewUploadSubmittingID === row.id" @click="previewUploadedAudio(row)">{{ t('original_audio') }}</Button>
                    <Button v-if="canPreviewClonedVoice(row)" variant="outline" size="sm" :disabled="previewClonedSubmittingID === row.id" @click="previewClonedVoice(row)">{{ t('preview_clone') }}</Button>
                    <Button variant="outline" size="sm" @click="openEditDialog(row)">{{ t('edit') }}</Button>
                    <Button v-if="canRetryClone(row)" variant="outline" size="sm" :disabled="retrySubmittingID === row.id" @click="retryClone(row)">{{ t('re_clone') }}</Button>
                    <Button v-if="canAppendRefAudio(row)" variant="outline" size="sm" :disabled="appendAudioSubmittingID === row.id" @click="openAppendAudioDialog(row)">{{ t('append_reference_audio') }}</Button>
                    <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" :disabled="deleteSubmittingID === row.id" @click="deleteClone(row)">{{ t('delete') }}</Button>
                  </div>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
      <span>{{ total }} {{ t('total_items') }}</span>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadVoiceClones()">{{ t('prev') }}</Button>
        <span>{{ page }} / {{ totalPages() }}</span>
        <Button variant="outline" size="sm" :disabled="page >= totalPages()" @click="page++; loadVoiceClones()">{{ t('next') }}</Button>
      </div>
    </div>

    <!-- Create dialog -->
    <Dialog v-model:open="createDialogVisible">
      <DialogContent class="max-w-[680px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('create_clone_voice') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('clone_name_label') }}</label>
            <Input v-model="form.name" :placeholder="t('clone_name_optional_ph')" />
          </div>

          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('tts_config_label') }} <span class="text-destructive">*</span></label>
            <Select v-model="form.tts_config_id" @update:model-value="onConfigChange">
              <SelectTrigger class="w-full">
                <SelectValue :placeholder="t('select_cloneable_tts_ph')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="cfg in cloneEnabledConfigs" :key="cfg.config_id" :value="cfg.config_id">
                  {{ cfg.name }} ({{ cfg.config_id }})
                </SelectItem>
              </SelectContent>
            </Select>
            <p v-if="isAliyunQwenProvider" class="text-xs text-[var(--color-text-tertiary)]">{{ t('qwen_clone_hint', { model: qwenCloneRuntimeModel }) }}</p>
            <div v-if="createChargeNotice.message" class="rounded-lg border border-yellow-200 bg-yellow-50 text-yellow-800 p-3 text-xs dark:bg-yellow-900/20 dark:border-yellow-800 dark:text-yellow-300">
              {{ createChargeNotice.message }}
            </div>
          </div>

          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('audio_source') }}</label>
            <div class="flex gap-2">
              <Button :variant="form.source_type === 'upload' ? 'default' : 'outline'" size="sm" @click="form.source_type = 'upload'">{{ t('upload_audio') }}</Button>
              <Button :variant="form.source_type === 'record' ? 'default' : 'outline'" size="sm" @click="form.source_type = 'record'">{{ t('browser_record') }}</Button>
            </div>
          </div>

          <div v-if="form.source_type === 'upload'" class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('audio_file_label') }} <span class="text-destructive">*</span></label>
            <input type="file" :accept="uploadAcceptTypes" class="text-sm" @change="handleFileChange" />
            <p class="text-xs text-[var(--color-text-tertiary)]">{{ audioRequirementText }}</p>
          </div>

          <div v-else class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('browser_record') }} <span class="text-destructive">*</span></label>
            <div class="flex gap-2">
              <Button variant="outline" size="sm" :disabled="isRecording" @click="startRecording">{{ t('start_recording') }}</Button>
              <Button variant="destructive" size="sm" :disabled="!isRecording" @click="stopRecording">{{ t('stop_recording_btn') }}</Button>
            </div>
            <audio v-if="recordPreviewUrl" :src="recordPreviewUrl" controls class="w-full mt-1" />
            <p class="text-xs text-[var(--color-text-tertiary)]">{{ audioRequirementText }}</p>
          </div>

          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">
              {{ capability.requires_transcript ? t('audio_corresponding_text_req') : t('audio_corresponding_text') }}
            </label>
            <textarea
              v-model="form.transcript"
              rows="4"
              :placeholder="capability.requires_transcript ? t('provider_requires_audio_text') : t('optional_submit')"
              class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none resize-none"
            />
            <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('text_char_require', { min: capability.min_text_len || 0, max: capability.max_text_len || 4000 }) }}</p>
          </div>

          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('text_language') }}</label>
            <Select v-model="form.transcript_lang">
              <SelectTrigger class="w-[220px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="zh-CN">{{ t('chinese_lang_option') }}</SelectItem>
                <SelectItem value="en-US">{{ t('english_lang_option') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="createDialogVisible = false">{{ t('cancel') }}</Button>
          <Button :disabled="submitting" @click="submitClone">{{ t('submit_clone') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Audio list dialog -->
    <Dialog v-model:open="audioDialogVisible">
      <DialogContent class="max-w-[720px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ t('clone_original_audio') }}</DialogTitle>
        </DialogHeader>
        <div class="rounded-xl border border-[var(--color-line)] overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead class="w-24">{{ t('source_label') }}</TableHead>
                <TableHead>{{ t('filename_label') }}</TableHead>
                <TableHead>{{ t('corresponded_text') }}</TableHead>
                <TableHead class="w-24 text-center">{{ t('play') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableEmpty v-if="!currentAudios.length" />
              <TableRow v-for="audio in currentAudios" :key="audio.id">
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ audio.source_type }}</TableCell>
                <TableCell class="text-sm">{{ audio.file_name }}</TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)] max-w-[240px] truncate" :title="audio.transcript">{{ audio.transcript }}</TableCell>
                <TableCell class="text-center">
                  <Button variant="ghost" size="sm" @click="playAudio(audio)">{{ t('play') }}</Button>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Edit dialog -->
    <Dialog v-model:open="editDialogVisible" @update:open="(v) => !v && resetEditForm()">
      <DialogContent class="max-w-[620px]">
        <DialogHeader>
          <DialogTitle>{{ t('edit_clone_voice') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('name') }}</label>
            <Input v-model="editForm.name" maxlength="100" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text-secondary)]">{{ t('provider') }}</label>
            <Input :value="editForm.provider" readonly class="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text-secondary)]">{{ t('tts_config_label') }}</label>
            <Input :value="editForm.ttsConfigDisplay" readonly class="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text-secondary)]">{{ t('clone_voice_id') }}</label>
            <Input :value="editForm.providerVoiceID" readonly class="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text-secondary)]">{{ t('task_status') }}</label>
            <Input :value="editForm.statusText" readonly class="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text-secondary)]">{{ t('created_at') }}</label>
            <Input :value="editForm.createdAtText" readonly class="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]" />
          </div>
          <div v-if="editForm.lastError" class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text-secondary)]">{{ t('failure_reason') }}</label>
            <textarea :value="editForm.lastError" readonly rows="3" class="dark:bg-input/30 border-input rounded-md border bg-[var(--color-surface-muted)] px-2.5 py-2 text-sm text-[var(--color-text-secondary)] resize-none focus-visible:outline-none" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="editDialogVisible = false">{{ t('cancel') }}</Button>
          <Button :disabled="editSubmitting" @click="submitEditClone">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Preview player dialog -->
    <Dialog v-model:open="previewPlayerVisible" @update:open="(v) => !v && closePreviewPlayerDialog()">
      <DialogContent class="max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{{ t('audio_preview_title') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="flex items-center gap-2">
            <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700">{{ previewPlayerSourceLabel || '-' }}</span>
            <span class="text-sm text-[var(--color-text-secondary)] truncate">{{ previewPlayerCloneLabel || '-' }}</span>
          </div>
          <audio
            ref="previewPlayerRef"
            class="w-full"
            :src="previewPlayerURL"
            controls
            preload="metadata"
            @play="onPreviewAudioPlay"
            @pause="onPreviewAudioPause"
            @ended="onPreviewAudioEnded"
            @timeupdate="onPreviewAudioTimeUpdate"
            @loadedmetadata="onPreviewAudioLoadedMetadata"
          />
          <div class="flex items-center gap-3">
            <Button :disabled="!previewPlayerURL" @click="togglePreviewPlayback">
              {{ previewPlayerPlaying ? t('pause') : t('play') }}
            </Button>
            <Button variant="outline" :disabled="!previewPlayerURL" @click="stopPreviewPlayback">{{ t('stop_btn') }}</Button>
            <span class="text-xs text-[var(--color-text-tertiary)] font-mono">{{ formatPlayerTime(previewPlayerCurrentTime) }} / {{ formatPlayerTime(previewPlayerDuration) }}</span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
