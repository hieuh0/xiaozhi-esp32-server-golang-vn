// Composable: all state refs + API call functions for Speakers page
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../utils/api'
import { useLocale } from './useLocale'

export function useSpeakers() {
  const { t } = useLocale()

  // --- Core list state ---
  const loading = ref(false)
  const submitting = ref(false)
  const speakerGroups = ref([])
  const agents = ref([])
  const samples = ref([])
  const filterAgentId = ref('')
  const searchKeyword = ref('')

  // --- Dialog/drawer visibility ---
  const showGroupDialog = ref(false)
  const groupDialogMode = ref('add') // 'add' | 'edit'
  const currentGroup = ref(null)
  const showSampleDrawer = ref(false)
  const showUploadDialog = ref(false)
  const uploadMode = ref('history') // 'upload' | 'record' | 'history'

  // --- Verify dialog state ---
  const showVerifyDialog = ref(false)
  const verifyMode = ref('upload') // 'upload' | 'record'
  const currentVerifyGroup = ref(null)
  const verifying = ref(false)
  const verifyResult = ref(null)

  // --- Verify form ---
  const verifyForm = reactive({ audioFile: null, audio: null })
  const verifyFileList = ref([])

  // --- Verify recording ---
  const isVerifyRecording = ref(false)
  const verifyMediaRecorder = ref(null)
  const verifyRecordedBlob = ref(null)
  const verifyRecordedBlobUrl = ref('')
  const verifyRecordTime = ref(0)
  const verifyRecordTimer = ref(null)

  // --- Upload recording ---
  const isRecording = ref(false)
  const mediaRecorder = ref(null)
  const recordedBlob = ref(null)
  const recordedBlobUrl = ref('')
  const recordTime = ref(0)
  const recordTimer = ref(null)
  const canRecord = ref(false)

  // --- Group form ---
  const groupForm = reactive({
    agent_id: null, name: '', prompt: '', description: '',
    tts_config_id: null, voice: null
  })

  const groupRules = {
    agent_id: [{ required: true, message: () => t('select_linked_agent'), trigger: 'change' }],
    name: [
      { required: true, message: () => t('enter_voiceprint_name'), trigger: 'blur' },
      { min: 1, max: 100, message: () => t('length_1_to_100'), trigger: 'blur' }
    ]
  }

  // --- TTS state ---
  const ttsConfigs = ref([])
  const currentVoiceOptions = ref([])
  const cloneVoicePresets = ref([])
  const cloneVoicesLoading = ref(false)

  // --- Upload form ---
  const uploadForm = reactive({ audioFile: null, audio: null })

  const uploadRules = {
    audio: [{
      validator: (rule, value, callback) => {
        if (!uploadForm.audioFile && !recordedBlob.value) {
          callback(new Error(t('upload_or_record_audio')))
        } else { callback() }
      },
      trigger: ['change', 'blur']
    }]
  }

  // --- Verify rules ---
  const verifyRules = {
    audio: [{
      validator: (rule, value, callback) => {
        if (!verifyForm.audioFile && !verifyRecordedBlob.value) {
          callback(new Error(t('upload_or_record_audio')))
        } else { callback() }
      },
      trigger: ['change', 'blur']
    }]
  }

  // --- History state ---
  const loadingHistory = ref(false)
  const historyMessages = ref([])
  const historyForm = reactive({ agent_id: null, selected_message_id: null })

  // --- Computed ---
  const hasAudioFile = computed(() => {
    if (uploadMode.value === 'history') return historyForm.selected_message_id !== null
    return uploadForm.audioFile !== null || recordedBlob.value !== null
  })

  const hasVerifyAudioFile = computed(() => {
    return verifyForm.audioFile !== null || verifyRecordedBlob.value !== null
  })

  const filteredGroups = computed(() => {
    let result = speakerGroups.value
    if (filterAgentId.value) {
      result = result.filter(g => g.agent_id === filterAgentId.value)
    }
    if (searchKeyword.value) {
      const keyword = searchKeyword.value.toLowerCase()
      result = result.filter(g =>
        g.name.toLowerCase().includes(keyword) ||
        (g.prompt && g.prompt.toLowerCase().includes(keyword)) ||
        (g.description && g.description.toLowerCase().includes(keyword))
      )
    }
    return result
  })

  // --- API: load lists ---
  const loadAgents = async () => {
    try {
      const response = await api.get('/user/agents')
      agents.value = response.data.data || []
    } catch (error) {
      console.error(t('load_agent_list_failed_v2'), error)
      ElMessage.error(t('load_agent_list_failed'))
    }
  }

  const loadTtsConfigs = async () => {
    try {
      const response = await api.get('/user/tts-configs')
      ttsConfigs.value = response.data.data || []
    } catch (error) {
      console.error(t('load_tts_config_failed'), error)
      ElMessage.error(t('load_tts_config_failed'))
    }
  }

  const normalizeCloneStatus = (clone) => {
    const status = String(clone?.status || '').trim().toLowerCase()
    const taskStatus = String(clone?.task_status || '').trim().toLowerCase()
    if (status === 'failed' || taskStatus === 'failed') return 'failed'
    if (status === 'active' || taskStatus === 'succeeded') return 'active'
    if (taskStatus === 'queued' || taskStatus === 'processing') return taskStatus
    if (status === 'queued' || status === 'processing') return status
    return status || taskStatus || 'unknown'
  }

  const loadCloneVoicePresets = async () => {
    cloneVoicesLoading.value = true
    try {
      const response = await api.get('/user/voice-clones')
      const cloneList = response.data.data || []
      cloneVoicePresets.value = cloneList
        .filter(clone => normalizeCloneStatus(clone) === 'active')
        .filter(clone => clone?.tts_config_id && clone?.provider_voice_id)
        .map(clone => ({
          id: clone.id,
          name: clone.name || clone.provider_voice_id,
          provider_voice_id: clone.provider_voice_id,
          tts_config_id: clone.tts_config_id,
          tts_config_name: clone.tts_config_name || ''
        }))
    } catch (error) {
      console.error(t('load_clone_voice_failed'), error)
      cloneVoicePresets.value = []
    } finally {
      cloneVoicesLoading.value = false
    }
  }

  const isCloneVoiceSelected = (clone) => {
    return groupForm.tts_config_id === clone?.tts_config_id && groupForm.voice === clone?.provider_voice_id
  }

  const handleTtsConfigChange = async (configId) => {
    if (!configId) { currentVoiceOptions.value = []; groupForm.voice = null; return }
    const config = ttsConfigs.value.find(c => c.config_id === configId)
    if (!config) { currentVoiceOptions.value = []; return }
    try {
      const params = { provider: config.provider, config_id: configId }
      const response = await api.get('/user/voice-options', { params })
      currentVoiceOptions.value = response.data.data || []
    } catch (error) {
      console.error(t('load_voice_list_failed_c'), error)
      currentVoiceOptions.value = []
      ElMessage.warning(t('load_voice_list_failed'))
    }
  }

  const applyCloneVoice = async (clone) => {
    if (!clone) return
    const ttsConfig = ttsConfigs.value.find(config => config.config_id === clone.tts_config_id)
    if (!ttsConfig) return
    groupForm.tts_config_id = clone.tts_config_id
    await handleTtsConfigChange(clone.tts_config_id)
    groupForm.voice = clone.provider_voice_id
  }

  const getCurrentTtsConfigName = () => {
    if (!groupForm.tts_config_id) return ''
    const config = ttsConfigs.value.find(c => c.config_id === groupForm.tts_config_id)
    return config ? config.name : ''
  }

  const getCurrentTtsConfigInfo = () => {
    if (!groupForm.tts_config_id) return ''
    const config = ttsConfigs.value.find(c => c.config_id === groupForm.tts_config_id)
    if (!config) return ''
    return t('tts_provider_label', { provider: config.provider || t('unknown') })
  }

  const loadSpeakerGroups = async () => {
    try {
      loading.value = true
      const params = {}
      if (filterAgentId.value) params.agent_id = filterAgentId.value
      const response = await api.get('/user/speaker-groups', { params })
      speakerGroups.value = response.data.data || []
    } catch (error) {
      console.error(t('load_voiceprint_group_failed'), error)
      ElMessage.error(t('load_voiceprint_group_failed') + ' ' + (error.response?.data?.error || error.message))
    } finally {
      loading.value = false
    }
  }

  // --- Group CRUD ---
  const resetGroupForm = () => {
    Object.assign(groupForm, { agent_id: null, name: '', prompt: '', description: '', tts_config_id: null, voice: null })
    currentGroup.value = null
    currentVoiceOptions.value = []
  }

  const handleAddGroup = async (groupFormRef) => {
    groupDialogMode.value = 'add'
    resetGroupForm()
    if (groupFormRef?.value) groupFormRef.value.resetFields()
    await loadCloneVoicePresets()
    showGroupDialog.value = true
  }

  const handleEditGroup = async (group, groupFormRef) => {
    groupDialogMode.value = 'edit'
    currentGroup.value = group
    groupForm.agent_id = group.agent_id
    groupForm.name = group.name
    groupForm.prompt = group.prompt || ''
    groupForm.description = group.description || ''
    groupForm.tts_config_id = group.tts_config_id || null
    groupForm.voice = group.voice || null
    await loadCloneVoicePresets()
    if (groupForm.tts_config_id) await handleTtsConfigChange(groupForm.tts_config_id)
    showGroupDialog.value = true
  }

  const handleSubmitGroup = async (groupFormRef) => {
    if (!groupFormRef?.value) return
    try {
      await groupFormRef.value.validate()
      submitting.value = true
      if (groupDialogMode.value === 'add') {
        await api.post('/user/speaker-groups', groupForm)
        ElMessage.success(t('create_success'))
      } else {
        await api.put(`/user/speaker-groups/${currentGroup.value.id}`, groupForm)
        ElMessage.success(t('update_success'))
      }
      showGroupDialog.value = false
      await loadSpeakerGroups()
    } catch (error) {
      if (error.fields) return
      console.error(t('submit_failed'), error)
      ElMessage.error(t('operation_failed_colon') + ' ' + (error.response?.data?.error || error.message))
    } finally {
      submitting.value = false
    }
  }

  const handleDeleteGroup = async (group) => {
    try {
      await ElMessageBox.confirm(
        t('confirm_delete_group', { name: group.name }),
        t('confirm_delete'),
        { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' }
      )
      loading.value = true
      await api.delete(`/user/speaker-groups/${group.id}`)
      ElMessage.success(t('delete_success'))
      await loadSpeakerGroups()
    } catch (error) {
      if (error !== 'cancel') {
        console.error(t('delete_failed_colon'), error)
        ElMessage.error(t('delete_failed_colon') + ' ' + (error.response?.data?.error || error.message))
      }
    } finally {
      loading.value = false
    }
  }

  // --- Sample management ---
  const loadSamples = async (groupId) => {
    try {
      const response = await api.get(`/user/speaker-groups/${groupId}/samples`)
      samples.value = response.data.data || []
    } catch (error) {
      console.error(t('load_sample_list_failed_v2'), error)
      ElMessage.error(t('load_sample_list_failed'))
    }
  }

  const handleViewSamples = async (group) => {
    currentGroup.value = group
    showSampleDrawer.value = true
    await loadSamples(group.id)
  }

  const handleCloseSampleDrawer = () => {
    showSampleDrawer.value = false
    currentGroup.value = null
    samples.value = []
  }

  // --- Verify dialog ---
  const resetVerifyForm = (verifyFormRef, verifyUploadRef) => {
    if (verifyFormRef?.value) verifyFormRef.value.resetFields()
    if (verifyUploadRef?.value) verifyUploadRef.value.clearFiles()
    verifyForm.audioFile = null
    verifyForm.audio = null
    if (isVerifyRecording.value) stopVerifyRecording()
    if (verifyRecordedBlobUrl.value) { URL.revokeObjectURL(verifyRecordedBlobUrl.value); verifyRecordedBlobUrl.value = '' }
    verifyRecordedBlob.value = null
    verifyRecordTime.value = 0
    verifyMode.value = 'upload'
    verifyResult.value = null
  }

  const handleVerifyGroup = async (group, verifyFormRef, verifyUploadRef) => {
    resetVerifyForm(verifyFormRef, verifyUploadRef)
    await nextTick()
    currentVerifyGroup.value = group
    verifyResult.value = null
    verifyMode.value = 'upload'
    showVerifyDialog.value = true
    await nextTick()
    verifyUploadRef?.value?.clearFiles()
    verifyFileList.value = []
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      stream.getTracks().forEach(track => track.stop())
      canRecord.value = true
    } catch (error) {
      console.warn(t('browser_recording_error'), error)
      canRecord.value = false
    }
  }

  const handleCloseVerifyDialog = (verifyFormRef, verifyUploadRef) => {
    if (isVerifyRecording.value) stopVerifyRecording()
    resetVerifyForm(verifyFormRef, verifyUploadRef)
    showVerifyDialog.value = false
  }

  const handleVerifyFileChange = async (file, verifyFormRef, verifyUploadRef) => {
    verifyFileList.value = []
    await nextTick()
    if (verifyForm.audioFile) { verifyForm.audioFile = null; verifyForm.audio = null }
    if (verifyRecordedBlob.value) {
      if (verifyRecordedBlobUrl.value) { URL.revokeObjectURL(verifyRecordedBlobUrl.value); verifyRecordedBlobUrl.value = '' }
      verifyRecordedBlob.value = null; verifyRecordTime.value = 0
    }
    verifyResult.value = null
    const fileObj = file.raw || file
    if (!fileObj) { ElMessage.warning(t('invalid_file_object')); verifyUploadRef?.value?.clearFiles(); verifyForm.audioFile = null; verifyFileList.value = []; return }
    const fileName = fileObj.name || file.name || ''
    const fileType = fileObj.type || file.type || ''
    if (!fileType.includes('wav') && !fileName.toLowerCase().endsWith('.wav')) {
      ElMessage.warning(t('wav_only_upload')); verifyUploadRef?.value?.clearFiles(); verifyForm.audioFile = null; verifyFileList.value = []; return
    }
    const fileSize = fileObj.size || file.size || 0
    if (fileSize > 10 * 1024 * 1024) {
      ElMessage.warning(t('file_size_limit')); verifyUploadRef?.value?.clearFiles(); verifyForm.audioFile = null; verifyFileList.value = []; return
    }
    verifyForm.audioFile = file; verifyForm.audio = file; verifyFileList.value = [file]
    await nextTick()
    if (verifyFormRef?.value) verifyFormRef.value.clearValidate('audio')
  }

  const handleVerifyFileRemove = (verifyFormRef) => {
    verifyForm.audioFile = null; verifyForm.audio = null; verifyFileList.value = []; verifyResult.value = null
    if (verifyFormRef?.value) verifyFormRef.value.validateField('audio')
  }

  // --- WAV conversion helpers ---
  const audioBufferToWav = (buffer) => {
    const length = buffer.length
    const numberOfChannels = buffer.numberOfChannels
    const sampleRate = buffer.sampleRate
    const bytesPerSample = 2
    const blockAlign = numberOfChannels * bytesPerSample
    const byteRate = sampleRate * blockAlign
    const dataSize = length * blockAlign
    const bufferSize = 44 + dataSize
    const arrayBuffer = new ArrayBuffer(bufferSize)
    const view = new DataView(arrayBuffer)
    const writeString = (offset, string) => {
      for (let i = 0; i < string.length; i++) view.setUint8(offset + i, string.charCodeAt(i))
    }
    writeString(0, 'RIFF'); view.setUint32(4, bufferSize - 8, true)
    writeString(8, 'WAVE'); writeString(12, 'fmt ')
    view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, numberOfChannels, true)
    view.setUint32(24, sampleRate, true); view.setUint32(28, byteRate, true)
    view.setUint16(32, blockAlign, true); view.setUint16(34, 16, true)
    writeString(36, 'data'); view.setUint32(40, dataSize, true)
    let offset = 44
    for (let i = 0; i < length; i++) {
      for (let channel = 0; channel < numberOfChannels; channel++) {
        const sample = Math.max(-1, Math.min(1, buffer.getChannelData(channel)[i]))
        view.setInt16(offset, sample < 0 ? sample * 0x8000 : sample * 0x7FFF, true)
        offset += 2
      }
    }
    return arrayBuffer
  }

  const convertToWav = (blob) => new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = async (e) => {
      try {
        const audioContext = new (window.AudioContext || window.webkitAudioContext)()
        const audioBuffer = await audioContext.decodeAudioData(e.target.result)
        const wav = audioBufferToWav(audioBuffer)
        resolve(new Blob([wav], { type: 'audio/wav' }))
      } catch (error) {
        console.error(t('convert_wav_failed_v2'), error)
        reject(error)
      }
    }
    reader.onerror = reject
    reader.readAsArrayBuffer(blob)
  })

  // --- Recording (upload dialog) ---
  const startRecording = async (uploadFormRef) => {
    try {
      if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') mediaRecorder.value.stop()
      if (recordedBlobUrl.value) { URL.revokeObjectURL(recordedBlobUrl.value); recordedBlobUrl.value = '' }
      recordedBlob.value = null; recordTime.value = 0
      const stream = await navigator.mediaDevices.getUserMedia({ audio: { channelCount: 1, sampleRate: 16000, echoCancellation: true, noiseSuppression: true } })
      const chunks = []
      const options = { mimeType: 'audio/webm;codecs=opus' }
      mediaRecorder.value = MediaRecorder.isTypeSupported(options.mimeType) ? new MediaRecorder(stream, options) : new MediaRecorder(stream)
      mediaRecorder.value.ondataavailable = (e) => { if (e.data.size > 0) chunks.push(e.data) }
      mediaRecorder.value.onstop = async () => {
        stream.getTracks().forEach(track => track.stop())
        try {
          const blob = new Blob(chunks, { type: chunks[0]?.type || 'audio/webm' })
          const wavBlob = await convertToWav(blob)
          recordedBlob.value = wavBlob; recordedBlobUrl.value = URL.createObjectURL(wavBlob)
          const fileName = `recording_${Date.now()}.wav`
          uploadForm.audioFile = { raw: new File([wavBlob], fileName, { type: 'audio/wav' }), name: fileName, size: wavBlob.size }
          uploadForm.audio = uploadForm.audioFile.raw
          if (uploadFormRef?.value) uploadFormRef.value.clearValidate('audio')
        } catch (error) {
          console.error(t('process_recording_failed_v2'), error)
          ElMessage.error(t('process_recording_failed'))
          recordedBlob.value = null; recordedBlobUrl.value = ''; uploadForm.audioFile = null; uploadForm.audio = null
        }
        chunks.length = 0
      }
      mediaRecorder.value.start(100); isRecording.value = true
      recordTimer.value = setInterval(() => { recordTime.value += 0.1 }, 100)
      ElMessage.success(t('start_recording'))
    } catch (error) {
      console.error(t('recording_failed'), error)
      ElMessage.error(t('recording_failed') + ' ' + error.message)
      canRecord.value = false
    }
  }

  const stopRecording = () => {
    if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') mediaRecorder.value.stop()
    isRecording.value = false
    if (recordTimer.value) { clearInterval(recordTimer.value); recordTimer.value = null }
    ElMessage.success(t('recording_complete'))
  }

  // --- Recording (verify dialog) ---
  const startVerifyRecording = async (verifyFormRef) => {
    try {
      if (verifyMediaRecorder.value && verifyMediaRecorder.value.state !== 'inactive') verifyMediaRecorder.value.stop()
      if (verifyRecordedBlobUrl.value) { URL.revokeObjectURL(verifyRecordedBlobUrl.value); verifyRecordedBlobUrl.value = '' }
      verifyRecordedBlob.value = null; verifyRecordTime.value = 0
      const stream = await navigator.mediaDevices.getUserMedia({ audio: { channelCount: 1, sampleRate: 16000, echoCancellation: true, noiseSuppression: true } })
      const chunks = []
      const options = { mimeType: 'audio/webm;codecs=opus' }
      verifyMediaRecorder.value = MediaRecorder.isTypeSupported(options.mimeType) ? new MediaRecorder(stream, options) : new MediaRecorder(stream)
      verifyMediaRecorder.value.ondataavailable = (e) => { if (e.data.size > 0) chunks.push(e.data) }
      verifyMediaRecorder.value.onstop = async () => {
        stream.getTracks().forEach(track => track.stop())
        try {
          const blob = new Blob(chunks, { type: chunks[0]?.type || 'audio/webm' })
          const wavBlob = await convertToWav(blob)
          verifyRecordedBlob.value = wavBlob; verifyRecordedBlobUrl.value = URL.createObjectURL(wavBlob)
          const fileName = `verify_recording_${Date.now()}.wav`
          const file = new File([wavBlob], fileName, { type: 'audio/wav' })
          verifyForm.audioFile = { raw: file, name: fileName, size: wavBlob.size }; verifyForm.audio = file
          if (verifyFormRef?.value) verifyFormRef.value.clearValidate('audio')
        } catch (error) {
          console.error(t('process_recording_failed_v2'), error)
          ElMessage.error(t('process_recording_failed'))
          verifyRecordedBlob.value = null; verifyRecordedBlobUrl.value = ''; verifyForm.audioFile = null; verifyForm.audio = null
        }
        chunks.length = 0
      }
      verifyMediaRecorder.value.start(100); isVerifyRecording.value = true
      verifyRecordTimer.value = setInterval(() => { verifyRecordTime.value += 0.1 }, 100)
      ElMessage.success(t('start_recording'))
    } catch (error) {
      console.error(t('recording_failed'), error)
      ElMessage.error(t('recording_failed') + ' ' + error.message)
      canRecord.value = false
    }
  }

  const stopVerifyRecording = () => {
    if (verifyMediaRecorder.value && verifyMediaRecorder.value.state !== 'inactive') verifyMediaRecorder.value.stop()
    isVerifyRecording.value = false
    if (verifyRecordTimer.value) { clearInterval(verifyRecordTimer.value); verifyRecordTimer.value = null }
    ElMessage.success(t('recording_complete'))
  }

  // --- Submit verify ---
  const handleSubmitVerify = async (verifyFormRef) => {
    if (!verifyFormRef?.value) return
    try {
      await verifyFormRef.value.validate()
      if (!verifyForm.audioFile && !verifyRecordedBlob.value) { ElMessage.warning(t('upload_or_record_audio')); return }
      verifying.value = true; verifyResult.value = null
      let file
      if (verifyForm.audioFile) {
        file = verifyForm.audioFile.raw || verifyForm.audioFile
      } else {
        const fileName = `verify_recording_${Date.now()}.wav`
        file = new File([verifyRecordedBlob.value], fileName, { type: 'audio/wav' })
      }
      const formData = new FormData(); formData.append('audio', file)
      const response = await api.post(`/user/speaker-groups/${currentVerifyGroup.value.id}/verify`, formData)
      if (response.data.success && response.data.data) {
        verifyResult.value = {
          verified: response.data.data.verified,
          confidence: response.data.data.confidence,
          threshold: response.data.data.threshold,
          message: response.data.data.message
        }
        if (verifyResult.value.verified) ElMessage.success(t('verify_passed'))
        else ElMessage.warning(t('verify_not_passed'))
      } else {
        ElMessage.error(t('verify_failed'))
      }
    } catch (error) {
      if (error.fields) return
      console.error(t('verify_failed_colon'), error)
      ElMessage.error(t('verify_failed_colon') + ' ' + (error.response?.data?.error || error.message))
    } finally {
      verifying.value = false
    }
  }

  // --- Upload sample ---
  const resetUploadForm = (uploadFormRef, uploadRef) => {
    if (uploadFormRef?.value) uploadFormRef.value.resetFields()
    if (uploadRef?.value) uploadRef.value.clearFiles()
    uploadForm.audioFile = null; uploadForm.audio = null
    if (isRecording.value) stopRecording()
    if (recordedBlobUrl.value) { URL.revokeObjectURL(recordedBlobUrl.value); recordedBlobUrl.value = '' }
    recordedBlob.value = null; recordTime.value = 0; uploadMode.value = 'history'
    historyForm.agent_id = null; historyForm.selected_message_id = null; historyMessages.value = []
  }

  const handleAddSample = async (uploadFormRef, uploadRef) => {
    resetUploadForm(uploadFormRef, uploadRef)
    uploadMode.value = 'history'
    showUploadDialog.value = true
    historyForm.agent_id = currentGroup.value?.agent_id || null
    historyForm.selected_message_id = null
    historyMessages.value = []
    if (currentGroup.value?.agent_id) { historyForm.agent_id = currentGroup.value.agent_id; await loadHistoryMessages() }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      stream.getTracks().forEach(track => track.stop())
      canRecord.value = true
    } catch (error) {
      console.warn(t('browser_recording_error'), error)
      canRecord.value = false
    }
  }

  const handleCloseUploadDialog = (uploadFormRef, uploadRef) => {
    if (isRecording.value) stopRecording()
    resetUploadForm(uploadFormRef, uploadRef)
    showUploadDialog.value = false
  }

  const handleFileChange = (file, uploadFormRef, uploadRef) => {
    const fileObj = file.raw || file
    if (!fileObj) { ElMessage.warning(t('invalid_file_object')); uploadRef?.value?.clearFiles(); uploadForm.audioFile = null; return }
    const fileName = fileObj.name || file.name || ''
    const fileType = fileObj.type || file.type || ''
    if (!fileType.includes('wav') && !fileName.toLowerCase().endsWith('.wav')) {
      ElMessage.warning(t('wav_only_upload')); uploadRef?.value?.clearFiles(); uploadForm.audioFile = null; return
    }
    const fileSize = fileObj.size || file.size || 0
    if (fileSize > 10 * 1024 * 1024) { ElMessage.warning(t('file_size_limit')); uploadRef?.value?.clearFiles(); uploadForm.audioFile = null; return }
    uploadForm.audioFile = file; uploadForm.audio = file
    if (uploadFormRef?.value) uploadFormRef.value.clearValidate('audio')
  }

  const handleFileRemove = (uploadFormRef) => {
    uploadForm.audioFile = null; uploadForm.audio = null
    if (uploadFormRef?.value) uploadFormRef.value.validateField('audio')
  }

  const loadHistoryMessages = async () => {
    if (!historyForm.agent_id) { historyMessages.value = []; return }
    try {
      loadingHistory.value = true
      const response = await api.get('/user/history/messages', { params: { agent_id: historyForm.agent_id, role: 'user', page: 1, page_size: 50 } })
      historyMessages.value = (response.data.data || []).filter(msg => msg.audio_path)
    } catch (error) {
      console.error(t('load_chat_history_failed'), error)
      ElMessage.error(t('load_chat_history_failed') + ' ' + (error.response?.data?.error || error.message))
      historyMessages.value = []
    } finally {
      loadingHistory.value = false
    }
  }

  const handleSelectHistoryMessage = (row) => { historyForm.selected_message_id = row.message_id }

  const handlePreviewHistoryAudio = async (message, audioPlayer) => {
    try {
      const response = await api.get(`/user/history/messages/${message.id}/audio`, { responseType: 'blob' })
      const blob = new Blob([response.data], { type: 'audio/wav' })
      const blobUrl = URL.createObjectURL(blob)
      audioPlayer.value.src = blobUrl
      audioPlayer.value.play().catch(err => { console.error(t('play_failed_colon'), err); ElMessage.warning(t('play_failed_check_audio')) })
      audioPlayer.value.onended = () => URL.revokeObjectURL(blobUrl)
    } catch (error) {
      console.error(t('preview_failed'), error)
      ElMessage.error(t('preview_failed') + ' ' + (error.response?.data?.error || error.message))
    }
  }

  const handleSubmitSample = async (uploadFormRef, uploadRef) => {
    if (uploadMode.value === 'history') {
      if (!historyForm.selected_message_id) { ElMessage.warning(t('select_chat_history')); return }
      try {
        submitting.value = true
        const formData = new FormData(); formData.append('message_id', historyForm.selected_message_id)
        await api.post(`/user/speaker-groups/${currentGroup.value.id}/samples`, formData)
        ElMessage.success(t('add_success'))
        handleCloseUploadDialog(uploadFormRef, uploadRef)
        await loadSamples(currentGroup.value.id); await loadSpeakerGroups()
      } catch (error) {
        console.error(t('add_failed'), error)
        ElMessage.error(t('add_failed') + ' ' + (error.response?.data?.error || error.message))
      } finally { submitting.value = false }
      return
    }
    if (!uploadFormRef?.value) return
    try {
      await uploadFormRef.value.validate()
      if (!uploadForm.audioFile && !recordedBlob.value) { ElMessage.warning(t('upload_or_record_audio')); return }
      submitting.value = true
      let file
      if (uploadForm.audioFile) { file = uploadForm.audioFile.raw || uploadForm.audioFile }
      else { const fileName = `recording_${Date.now()}.wav`; file = new File([recordedBlob.value], fileName, { type: 'audio/wav' }) }
      const formData = new FormData(); formData.append('audio', file)
      await api.post(`/user/speaker-groups/${currentGroup.value.id}/samples`, formData)
      ElMessage.success(t('upload_success'))
      handleCloseUploadDialog(uploadFormRef, uploadRef)
      await loadSamples(currentGroup.value.id); await loadSpeakerGroups()
    } catch (error) {
      if (error.fields) return
      console.error(t('upload_failed'), error)
      ElMessage.error(t('upload_failed') + ' ' + (error.response?.data?.error || error.message))
    } finally { submitting.value = false }
  }

  // --- Sample actions ---
  const handlePlaySample = async (sample, audioPlayer) => {
    try {
      const response = await api.get(`/user/speaker-groups/${currentGroup.value.id}/samples/${sample.id}/file`, { responseType: 'blob' })
      const blob = new Blob([response.data], { type: 'audio/wav' })
      const blobUrl = URL.createObjectURL(blob)
      audioPlayer.value.src = blobUrl
      audioPlayer.value.play().catch(err => { console.error(t('play_failed_colon'), err); ElMessage.warning(t('play_failed_check_audio')) })
      audioPlayer.value.onended = () => URL.revokeObjectURL(blobUrl)
    } catch (error) {
      console.error(t('play_failed_colon'), error)
      ElMessage.error(t('play_failed_colon') + ' ' + (error.response?.data?.error || error.message))
    }
  }

  const handleDownloadSample = async (sample) => {
    try {
      const response = await api.get(`/user/speaker-groups/${currentGroup.value.id}/samples/${sample.id}/file`, { responseType: 'blob' })
      const blob = new Blob([response.data], { type: 'audio/wav' })
      const blobUrl = URL.createObjectURL(blob)
      const link = document.createElement('a'); link.href = blobUrl; link.download = sample.file_name || 'audio.wav'
      document.body.appendChild(link); link.click(); document.body.removeChild(link)
      setTimeout(() => URL.revokeObjectURL(blobUrl), 100)
    } catch (error) {
      console.error(t('download_failed'), error)
      ElMessage.error(t('download_failed') + ' ' + (error.response?.data?.error || error.message))
    }
  }

  const handleDeleteSample = async (sample) => {
    try {
      await ElMessageBox.confirm(
        t('confirm_delete_sample', { name: sample.file_name }),
        t('confirm_delete'),
        { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' }
      )
      await api.delete(`/user/speaker-groups/${currentGroup.value.id}/samples/${sample.id}`)
      ElMessage.success(t('delete_success'))
      await loadSamples(currentGroup.value.id); await loadSpeakerGroups()
    } catch (error) {
      if (error !== 'cancel') {
        console.error(t('delete_failed_colon'), error)
        ElMessage.error(t('delete_failed_colon') + ' ' + (error.response?.data?.error || error.message))
      }
    }
  }

  const handleVerifyFromSamples = (verifyFormRef, verifyUploadRef) => {
    if (currentGroup.value) { showSampleDrawer.value = false; handleVerifyGroup(currentGroup.value, verifyFormRef, verifyUploadRef) }
  }

  // --- Utilities ---
  const copyToClipboard = async (text) => {
    try { await navigator.clipboard.writeText(text); ElMessage.success(t('copied_to_clipboard')) }
    catch (error) { console.error(t('copy_failed_v2'), error); ElMessage.error(t('copy_failed')) }
  }

  const formatDate = (dateString) => { if (!dateString) return '-'; return new Date(dateString).toLocaleString('zh-CN') }
  const truncateId = (id) => { if (!id) return '-'; if (id.length > 20) return id.substring(0, 10) + '...' + id.substring(id.length - 10); return id }
  const truncateText = (text, maxLength) => { if (!text) return '-'; if (text.length <= maxLength) return text; return text.substring(0, maxLength) + '...' }
  const formatFileSize = (bytes) => { if (!bytes) return '0 B'; if (bytes < 1024) return bytes + ' B'; if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'; return (bytes / (1024 * 1024)).toFixed(2) + ' MB' }
  const formatRecordTime = (seconds) => { const mins = Math.floor(seconds / 60); const secs = Math.floor(seconds % 60); const ms = Math.floor((seconds % 1) * 10); return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}.${ms}` }

  // --- Lifecycle ---
  const initialize = () => { loadAgents(); loadSpeakerGroups(); loadTtsConfigs(); loadCloneVoicePresets() }

  const cleanup = () => {
    if (isRecording.value) stopRecording()
    if (recordedBlobUrl.value) URL.revokeObjectURL(recordedBlobUrl.value)
    if (recordTimer.value) clearInterval(recordTimer.value)
    if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') mediaRecorder.value.stop()
  }

  return {
    // State
    loading, submitting, speakerGroups, agents, samples, filterAgentId, searchKeyword,
    showGroupDialog, groupDialogMode, currentGroup, showSampleDrawer, showUploadDialog, uploadMode,
    showVerifyDialog, verifyMode, currentVerifyGroup, verifying, verifyResult,
    verifyForm, verifyFileList, verifyRules,
    isVerifyRecording, verifyRecordedBlob, verifyRecordedBlobUrl, verifyRecordTime,
    isRecording, recordedBlob, recordedBlobUrl, recordTime, canRecord,
    groupForm, groupRules,
    ttsConfigs, currentVoiceOptions, cloneVoicePresets, cloneVoicesLoading,
    uploadForm, uploadRules,
    loadingHistory, historyMessages, historyForm,
    // Computed
    hasAudioFile, hasVerifyAudioFile, filteredGroups,
    // Methods
    loadAgents, loadTtsConfigs, loadCloneVoicePresets, loadSpeakerGroups, loadSamples, loadHistoryMessages,
    handleSearch: () => {},
    handleAddGroup, handleEditGroup, handleSubmitGroup, handleDeleteGroup,
    handleViewSamples, handleCloseSampleDrawer, handleVerifyFromSamples,
    handleVerifyGroup, handleCloseVerifyDialog, handleVerifyFileChange, handleVerifyFileRemove,
    startVerifyRecording, stopVerifyRecording, handleSubmitVerify,
    handleAddSample, handleCloseUploadDialog, handleFileChange, handleFileRemove,
    startRecording, stopRecording, handleSubmitSample,
    handleSelectHistoryMessage, handlePreviewHistoryAudio,
    handlePlaySample, handleDownloadSample, handleDeleteSample,
    copyToClipboard, isCloneVoiceSelected, applyCloneVoice,
    handleTtsConfigChange, getCurrentTtsConfigName, getCurrentTtsConfigInfo,
    formatDate, truncateId, truncateText, formatFileSize, formatRecordTime,
    initialize, cleanup
  }
}
