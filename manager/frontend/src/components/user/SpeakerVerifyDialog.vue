<template>
  <el-dialog
    v-model="visible"
    :title="t('verify_group_title', { name: currentVerifyGroup?.name || '' })"
    width="600px"
    :before-close="() => emit('close')"
  >
    <el-tabs v-model="verifyMode" class="verify-tabs">
      <!-- Upload tab -->
      <el-tab-pane :label="t('upload_file')" name="upload">
        <el-form ref="verifyFormRef" :model="verifyForm" :rules="verifyRules" label-width="0">
          <el-form-item prop="audio">
            <el-upload
              ref="verifyUploadRef"
              :auto-upload="false"
              :on-change="(file) => emit('file-change', file)"
              :on-remove="() => emit('file-remove')"
              :limit="1"
              accept=".wav,audio/wav"
              drag
              class="audio-upload"
              :file-list="verifyFileList"
            >
              <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
              <div class="el-upload__text">{{ t('drag_wav_hint') }}</div>
              <template #tip>
                <div class="el-upload__tip">{{ t('wav_format_hint') }}</div>
              </template>
            </el-upload>
            <div v-if="verifyForm.audioFile" class="file-info">
              <el-icon><Document /></el-icon>
              <span>{{ verifyForm.audioFile.name }}</span>
              <span class="file-size">({{ formatFileSize(verifyForm.audioFile.size) }})</span>
            </div>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- Record tab -->
      <el-tab-pane :label="t('record_audio')" name="record">
        <div class="record-section">
          <div class="record-status">
            <div v-if="!isVerifyRecording && !verifyRecordedBlob" class="record-ready">
              <el-icon size="48" color="var(--apple-primary)"><Microphone /></el-icon>
              <p>{{ t('click_start_record') }}</p>
              <p class="record-tip">{{ t('record_tip') }}</p>
            </div>
            <div v-else-if="isVerifyRecording" class="record-recording">
              <div class="recording-indicator">
                <span class="recording-dot"></span>
                <span class="recording-text">{{ t('recording_in_progress') }}</span>
              </div>
              <div class="record-time">{{ formatRecordTime(verifyRecordTime) }}</div>
              <p class="record-tip">{{ t('click_stop_record') }}</p>
            </div>
            <div v-else-if="verifyRecordedBlob" class="record-complete">
              <el-icon size="48" color="var(--apple-success)"><CircleCheck /></el-icon>
              <p>{{ t('recording_complete') }}</p>
              <p class="record-tip">{{ t('record_duration', { time: formatRecordTime(verifyRecordTime) }) }}</p>
              <audio :src="verifyRecordedBlobUrl" controls class="record-preview"></audio>
            </div>
          </div>
          <div class="record-controls">
            <el-button v-if="!isVerifyRecording && !verifyRecordedBlob" type="primary" size="large" :disabled="!canRecord" @click="emit('start-recording')">
              <el-icon><VideoPlay /></el-icon>{{ t('start_recording') }}
            </el-button>
            <el-button v-if="isVerifyRecording" type="danger" size="large" @click="emit('stop-recording')">
              <el-icon><VideoPause /></el-icon>{{ t('stop_record') }}
            </el-button>
            <el-button v-if="verifyRecordedBlob" type="primary" size="large" :disabled="!canRecord" @click="emit('start-recording')">
              <el-icon><Refresh /></el-icon>{{ t('re_record') }}
            </el-button>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- Verification result -->
    <div v-if="verifyResult" class="verify-result">
      <el-divider>{{ t('verify_result_divider') }}</el-divider>
      <div :class="['result-content', verifyResult.verified ? 'result-success' : 'result-failed']">
        <div class="result-icon">
          <el-icon v-if="verifyResult.verified" size="48" color="var(--apple-success)"><CircleCheck /></el-icon>
          <el-icon v-else size="48" color="var(--apple-danger)"><CircleClose /></el-icon>
        </div>
        <div class="result-info">
          <div class="result-status">
            {{ verifyResult.verified ? t('verification_passed') : t('verify_not_passed') }}
          </div>
          <div class="result-details">
            <div>{{ t('confidence_label', { pct: (verifyResult.confidence * 100).toFixed(1) }) }}</div>
            <div>{{ t('threshold_label_pct', { pct: (verifyResult.threshold * 100).toFixed(1) }) }}</div>
          </div>
          <div class="result-message">{{ verifyResult.message }}</div>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="emit('close')">{{ t('cancel') }}</el-button>
      <el-button type="primary" @click="emit('submit')" :loading="verifying" :disabled="!hasVerifyAudioFile">
        {{ t('verify') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref } from 'vue'
import { UploadFilled, Document, VideoPlay, Microphone, CircleCheck, CircleClose, VideoPause, Refresh } from '@element-plus/icons-vue'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

defineProps({
  currentVerifyGroup: { type: Object, default: null },
  verifyForm: { type: Object, required: true },
  verifyRules: { type: Object, required: true },
  verifyFileList: { type: Array, default: () => [] },
  verifyResult: { type: Object, default: null },
  isVerifyRecording: { type: Boolean, default: false },
  verifyRecordedBlob: { type: Object, default: null },
  verifyRecordedBlobUrl: { type: String, default: '' },
  verifyRecordTime: { type: Number, default: 0 },
  canRecord: { type: Boolean, default: false },
  verifying: { type: Boolean, default: false },
  hasVerifyAudioFile: { type: Boolean, default: false },
  formatFileSize: { type: Function, required: true },
  formatRecordTime: { type: Function, required: true }
})

const visible = defineModel({ default: false })
const verifyMode = defineModel('verifyMode', { default: 'upload' })

const verifyFormRef = ref()
const verifyUploadRef = ref()

defineExpose({ verifyFormRef, verifyUploadRef })

const emit = defineEmits(['close', 'submit', 'file-change', 'file-remove', 'start-recording', 'stop-recording'])
</script>

<style scoped>
.audio-upload { width: 100%; }
.audio-upload :deep(.el-upload-dragger) { width: 100%; padding: 40px 20px; }
.audio-upload :deep(.el-icon--upload) { font-size: 48px; color: var(--apple-primary); margin-bottom: 16px; }
.audio-upload :deep(.el-upload__text) { font-size: 14px; color: #606266; }
.audio-upload :deep(.el-upload__tip) { margin-top: 12px; font-size: 12px; color: #909399; }
.file-info { display: flex; align-items: center; gap: 8px; margin-top: 8px; padding: 8px 12px; background: rgba(248,250,252,0.92); border: 1px solid rgba(229,229,234,0.72); border-radius: 12px; font-size: 14px; }
.file-size { color: #909399; font-size: 12px; }
.record-section { padding: 20px 0; }
.record-status { min-height: 200px; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 30px; background: #f5f7fa; border-radius: 8px; margin-bottom: 20px; }
.record-ready, .record-complete { text-align: center; }
.record-ready p, .record-complete p { margin: 12px 0 0 0; color: #303133; font-size: 16px; }
.record-tip { margin-top: 8px !important; font-size: 14px !important; color: #909399 !important; }
.record-recording { text-align: center; }
.recording-indicator { display: flex; align-items: center; justify-content: center; gap: 8px; margin-bottom: 16px; }
.recording-dot { width: 12px; height: 12px; border-radius: 50%; background: var(--apple-danger); animation: pulse 1.5s ease-in-out infinite; }
@keyframes pulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.5; transform: scale(1.2); } }
.recording-text { font-size: 16px; color: var(--apple-danger); font-weight: 500; }
.record-time { font-size: 32px; font-weight: 600; color: #303133; font-family: 'Courier New', monospace; margin: 20px 0; }
.record-preview { width: 100%; max-width: 400px; margin-top: 20px; }
.record-controls { display: flex; justify-content: center; gap: 12px; }
.record-controls .el-button { min-width: 120px; }
.verify-result { margin-top: 16px; }
.result-content { display: flex; align-items: center; gap: 20px; padding: 20px; border-radius: 8px; }
.result-success { background: rgba(103,194,58,0.08); border: 1px solid rgba(103,194,58,0.3); }
.result-failed { background: rgba(245,108,108,0.08); border: 1px solid rgba(245,108,108,0.3); }
.result-status { font-size: 18px; font-weight: 600; margin-bottom: 8px; }
.result-details { color: #606266; font-size: 14px; margin-bottom: 8px; }
.result-message { color: #909399; font-size: 13px; }
</style>
