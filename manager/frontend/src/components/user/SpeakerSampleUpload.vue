<template>
  <el-dialog
    v-model="visible"
    :title="t('add_voiceprint_sample')"
    width="600px"
    :before-close="() => emit('close')"
  >
    <el-tabs v-model="uploadMode" class="upload-tabs">
      <!-- History tab -->
      <el-tab-pane :label="t('select_from_history')" name="history">
        <div class="history-section">
          <el-form :model="historyForm" label-width="100px">
            <el-form-item :label="t('agent')">
              <el-select
                v-model="historyForm.agent_id"
                :placeholder="t('select_agent')"
                style="width: 100%"
                clearable
                @change="emit('load-history')"
              >
                <el-option v-for="agent in agents" :key="agent.id" :label="agent.name" :value="agent.id" />
              </el-select>
            </el-form-item>
          </el-form>
          <div v-loading="loadingHistory" class="history-list">
            <div v-if="historyMessages.length === 0 && !loadingHistory" class="empty-history">
              <el-empty :description="t('no_chat_history_select')" />
            </div>
            <el-table
              v-else
              :data="historyMessages"
              row-key="message_id"
              stripe
              style="width: 100%"
              max-height="400"
              @row-click="(row) => emit('select-history', row)"
            >
              <el-table-column :label="t('select_col')" width="80" align="center">
                <template #default="{ row }">
                  <el-radio
                    :model-value="historyForm.selected_message_id"
                    :label="row.message_id"
                    @change="historyForm.selected_message_id = row.message_id"
                  />
                </template>
              </el-table-column>
              <el-table-column prop="content" :label="t('message_content_col')" min-width="200">
                <template #default="{ row }">
                  <div class="message-content">{{ truncateText(row.content, 50) }}</div>
                </template>
              </el-table-column>
              <el-table-column prop="device_id" :label="t('device_id_col')" width="150">
                <template #default="{ row }">
                  <el-tooltip :content="row.device_id" placement="top">
                    <span>{{ truncateId(row.device_id) }}</span>
                  </el-tooltip>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" :label="t('time_col')" width="180">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
              <el-table-column :label="t('actions')" width="100">
                <template #default="{ row }">
                  <el-button type="primary" size="small" link @click.stop="emit('preview-audio', row)">
                    <el-icon><VideoPlay /></el-icon>{{ t('preview_btn') }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </el-tab-pane>

      <!-- Upload tab -->
      <el-tab-pane :label="t('upload_file')" name="upload">
        <el-form ref="uploadFormRef" :model="uploadForm" :rules="uploadRules" label-width="0">
          <el-form-item prop="audio">
            <el-upload
              ref="uploadRef"
              :auto-upload="false"
              :on-change="(file) => emit('file-change', file)"
              :on-remove="() => emit('file-remove')"
              :limit="1"
              accept=".wav,audio/wav"
              drag
              class="audio-upload"
            >
              <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
              <div class="el-upload__text">{{ t('drag_wav_hint') }}</div>
              <template #tip>
                <div class="el-upload__tip">{{ t('wav_format_hint') }}</div>
              </template>
            </el-upload>
            <div v-if="uploadForm.audioFile" class="file-info">
              <el-icon><Document /></el-icon>
              <span>{{ uploadForm.audioFile.name }}</span>
              <span class="file-size">({{ formatFileSize(uploadForm.audioFile.size) }})</span>
            </div>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- Record tab -->
      <el-tab-pane :label="t('record_audio')" name="record">
        <div class="record-section">
          <div class="record-status">
            <div v-if="!isRecording && !recordedBlob" class="record-ready">
              <el-icon size="48" color="var(--apple-primary)"><Microphone /></el-icon>
              <p>{{ t('click_start_record') }}</p>
              <p class="record-tip">{{ t('record_tip') }}</p>
            </div>
            <div v-else-if="isRecording" class="record-recording">
              <div class="recording-indicator">
                <span class="recording-dot"></span>
                <span class="recording-text">{{ t('recording_in_progress') }}</span>
              </div>
              <div class="record-time">{{ formatRecordTime(recordTime) }}</div>
              <p class="record-tip">{{ t('click_stop_record') }}</p>
            </div>
            <div v-else-if="recordedBlob" class="record-complete">
              <el-icon size="48" color="var(--apple-success)"><CircleCheck /></el-icon>
              <p>{{ t('recording_complete') }}</p>
              <p class="record-tip">{{ t('record_duration', { time: formatRecordTime(recordTime) }) }}</p>
              <audio :src="recordedBlobUrl" controls class="record-preview"></audio>
            </div>
          </div>
          <div class="record-controls">
            <el-button v-if="!isRecording && !recordedBlob" type="primary" size="large" :disabled="!canRecord" @click="emit('start-recording')">
              <el-icon><VideoPlay /></el-icon>{{ t('start_recording') }}
            </el-button>
            <el-button v-if="isRecording" type="danger" size="large" @click="emit('stop-recording')">
              <el-icon><VideoPause /></el-icon>{{ t('stop_record') }}
            </el-button>
            <el-button v-if="recordedBlob" type="primary" size="large" :disabled="!canRecord" @click="emit('start-recording')">
              <el-icon><Refresh /></el-icon>{{ t('re_record') }}
            </el-button>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <el-button @click="emit('close')">{{ t('cancel') }}</el-button>
      <el-button type="primary" @click="emit('submit')" :loading="submitting" :disabled="!hasAudioFile">
        {{ t('confirm') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref } from 'vue'
import { UploadFilled, Document, VideoPlay, Microphone, CircleCheck, VideoPause, Refresh } from '@element-plus/icons-vue'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

defineProps({
  agents: { type: Array, default: () => [] },
  historyForm: { type: Object, required: true },
  historyMessages: { type: Array, default: () => [] },
  loadingHistory: { type: Boolean, default: false },
  uploadForm: { type: Object, required: true },
  uploadRules: { type: Object, required: true },
  isRecording: { type: Boolean, default: false },
  recordedBlob: { type: Object, default: null },
  recordedBlobUrl: { type: String, default: '' },
  recordTime: { type: Number, default: 0 },
  canRecord: { type: Boolean, default: false },
  submitting: { type: Boolean, default: false },
  hasAudioFile: { type: Boolean, default: false },
  formatDate: { type: Function, required: true },
  formatFileSize: { type: Function, required: true },
  formatRecordTime: { type: Function, required: true },
  truncateId: { type: Function, required: true },
  truncateText: { type: Function, required: true }
})

const visible = defineModel({ default: false })
const uploadMode = defineModel('uploadMode', { default: 'history' })

const uploadFormRef = ref()
const uploadRef = ref()

defineExpose({ uploadFormRef, uploadRef })

const emit = defineEmits([
  'close', 'submit', 'load-history', 'select-history', 'preview-audio',
  'file-change', 'file-remove', 'start-recording', 'stop-recording'
])
</script>

<style scoped>
.upload-tabs { margin-top: 10px; }
.audio-upload { width: 100%; }
.audio-upload :deep(.el-upload-dragger) { width: 100%; padding: 40px 20px; }
.audio-upload :deep(.el-icon--upload) { font-size: 48px; color: var(--apple-primary); margin-bottom: 16px; }
.audio-upload :deep(.el-upload__text) { font-size: 14px; color: #606266; }
.audio-upload :deep(.el-upload__tip) { margin-top: 12px; font-size: 12px; color: #909399; }
.file-info { display: flex; align-items: center; gap: 8px; margin-top: 8px; padding: 8px 12px; background: rgba(248,250,252,0.92); border: 1px solid rgba(229,229,234,0.72); border-radius: 12px; font-size: 14px; }
.file-size { color: #909399; font-size: 12px; }
.history-section { padding: 20px 0; }
.history-list { margin-top: 20px; }
.empty-history { padding: 40px 0; }
.message-content { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.history-list :deep(.el-table__row) { cursor: pointer; }
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
</style>
