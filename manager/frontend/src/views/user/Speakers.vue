<template>
  <div class="speakers-page">
    <!-- Filter bar -->
    <div class="filter-bar">
      <el-select
        v-model="filterAgentId"
        :placeholder="t('filter_by_agent')"
        clearable
        style="width: 200px; margin-right: 10px;"
        @change="loadSpeakerGroups"
      >
        <el-option :label="t('all_agents')" value="" />
        <el-option
          v-for="agent in agents"
          :key="agent.id"
          :label="agent.name"
          :value="agent.id"
        />
      </el-select>
      <el-input
        v-model="searchKeyword"
        :placeholder="t('search_voiceprint_group')"
        clearable
        style="width: 250px;"
        @input="handleSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button class="create-group-button" type="primary" @click="handleAddGroup">
        <el-icon><Plus /></el-icon>
        {{ t('create_voiceprint_group') }}
      </el-button>
    </div>

    <!-- Voiceprint group list -->
    <div v-loading="loading" class="speakers-content">
      <el-table :data="filteredGroups" stripe style="width: 100%">
        <el-table-column prop="name" :label="t('voiceprint_group_name')" min-width="150" />
        <el-table-column prop="agent_name" :label="t('link_agent')" min-width="120" />
        <el-table-column label="Prompt" min-width="200">
          <template #default="{ row }">
            <el-popover
              placement="top"
              :width="300"
              trigger="hover"
              v-if="row.prompt"
            >
              <template #reference>
                <span class="prompt-text">{{ truncateText(row.prompt, 30) }}</span>
              </template>
              <div class="prompt-popover">{{ row.prompt }}</div>
            </el-popover>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="sample_count" :label="t('sample_count')" width="100" align="center">
          <template #default="{ row }">
            <el-tag type="info">{{ row.sample_count }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="t('created_at')" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('actions')" width="360" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
            <el-button
              type="success"
              size="small"
              @click="handleVerifyGroup(row)"
            >
              <el-icon><VideoPlay /></el-icon>{{ t('verify') }}</el-button>
            <el-button
              type="primary"
              size="small"
              @click="handleViewSamples(row)"
            >
              <el-icon><View /></el-icon>{{ t('manage_voiceprints') }}</el-button>
              <el-button
                type="primary"
                size="small"
                plain
                @click="handleEditGroup(row)"
              >
                  <el-icon><Edit /></el-icon>
                  {{ t('edit') }}
                </el-button>
              <el-button
                type="danger"
                size="small"
                @click="handleDeleteGroup(row)"
              >
                <el-icon><Delete /></el-icon>
                {{ t('delete') }}
                </el-button>
              </div>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="filteredGroups.length === 0 && !loading" class="empty-state">
        <el-empty :description="t('no_voiceprint_groups')" />
      </div>
    </div>

    <!-- Create/Edit voiceprint group dialog -->
    <el-dialog
      v-model="showGroupDialog"
      :title="groupDialogMode === 'add' ? t('create_voiceprint_group') : t('edit_voiceprint_group')"
      width="600px"
    >
      <el-form
        ref="groupFormRef"
        :model="groupForm"
        :rules="groupRules"
        label-width="100px"
      >
        <el-form-item :label="t('link_agent')" prop="agent_id">
          <el-select
            v-model="groupForm.agent_id"
            :placeholder="t('select_agent')"
            style="width: 100%"
          >
            <el-option
              v-for="agent in agents"
              :key="agent.id"
              :label="agent.name"
              :value="agent.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('voiceprint_name')" prop="name">
          <el-input
            v-model="groupForm.name"
            :placeholder="t('enter_voiceprint_name')"
            :maxlength="100"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="Prompt" prop="prompt">
          <el-input
            v-model="groupForm.prompt"
            type="textarea"
            :rows="4"
            :placeholder="t('role_prompt_ph')"
          />
        </el-form-item>
        <el-form-item :label="t('description')" prop="description">
          <el-input
            v-model="groupForm.description"
            type="textarea"
            :rows="3"
            :placeholder="t('desc_optional_ph')"
            :maxlength="200"
            show-word-limit
          />
        </el-form-item>
        <el-form-item :label="t('my_cloned_voice')" v-if="cloneVoicePresets.length > 0">
          <div class="clone-voice-line" v-loading="cloneVoicesLoading">
            <button
              v-for="clone in cloneVoicePresets"
              :key="clone.id"
              type="button"
              class="clone-voice-item"
              :class="{ active: isCloneVoiceSelected(clone) }"
              :title="`${clone.tts_config_name || clone.tts_config_id} · ${clone.provider_voice_id}`"
              @click="applyCloneVoice(clone)"
            >
              <span class="clone-voice-name">{{ clone.name || clone.provider_voice_id }}</span>
            </button>
          </div>
          <div class="form-help">{{ t('click_auto_fill') }}</div>
        </el-form-item>
        <el-form-item :label="t('tts_config_label')" prop="tts_config_id">
          <el-select
            v-model="groupForm.tts_config_id"
            :placeholder="t('select_tts_config_opt')"
            clearable
            style="width: 100%"
            @change="handleTtsConfigChange"
          >
            <el-option
              v-for="ttsConfig in ttsConfigs"
              :key="ttsConfig.config_id"
              :label="ttsConfig.is_default ? t('tts_default_label', { name: ttsConfig.name }) : ttsConfig.name"
              :value="ttsConfig.config_id"
            >
              <div class="config-option">
                {{ ttsConfig.name }}
                <el-tag v-if="ttsConfig.is_default" type="success" size="small" style="margin-left: 8px;">{{ t('default') }}</el-tag>
              </div>
              <span class="config-desc">{{ ttsConfig.provider || t('no_description_alt') }}</span>
            </el-option>
          </el-select>
          <div class="form-help" v-if="groupForm.tts_config_id">
            {{ getCurrentTtsConfigInfo() }}
          </div>
        </el-form-item>
        <el-form-item :label="t('voice_timbre')" prop="voice" v-if="groupForm.tts_config_id">
          <el-select
            v-model="groupForm.voice"
            :placeholder="t('select_or_enter_voice')"
            filterable
            allow-create
            clearable
            style="width: 100%"
          >
            <el-option
              v-for="voice in currentVoiceOptions"
              :key="voice.value"
              :label="voice.label"
              :value="voice.value"
            />
          </el-select>
          <div class="form-help">{{ t('current_tts_config_hint', { name: getCurrentTtsConfigName() }) }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGroupDialog = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmitGroup" :loading="submitting">
          {{ groupDialogMode === 'add' ? t('create') : t('save') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Sample management drawer -->
    <el-drawer
      v-model="showSampleDrawer"
      :title="t('sample_management')"
      :size="800"
      :before-close="handleCloseSampleDrawer"
    >
      <div v-if="currentGroup" class="sample-drawer">
        <!-- Voiceprint group info -->
        <el-card class="group-info-card" shadow="never">
          <div class="group-info">
            <h3>{{ currentGroup.name }}</h3>
            <div v-if="currentGroup.prompt" class="prompt-section">
              <strong>Prompt:</strong>
              <p>{{ currentGroup.prompt }}</p>
            </div>
            <div v-if="currentGroup.description" class="description-section">
              <strong>{{ t('desc_colon') }}</strong>
              <p>{{ currentGroup.description }}</p>
            </div>
          </div>
        </el-card>

        <!-- Sample list -->
        <div class="samples-section">
          <div class="samples-header">
            <h4>{{ t('sample_list') }}</h4>
            <div class="samples-header-actions">
              <el-button type="success" @click="handleVerifyFromSamples">
                <el-icon><VideoPlay /></el-icon>
                {{ t('verify_voiceprint_btn') }}
              </el-button>
              <el-button type="primary" @click="handleAddSample">
                <el-icon><Plus /></el-icon>
                {{ t('upload_new_sample') }}
              </el-button>
            </div>
          </div>

          <el-table :data="samples" stripe style="width: 100%">
            <el-table-column prop="uuid" label="UUID" min-width="200">
              <template #default="{ row }">
                <el-tooltip :content="row.uuid" placement="top">
                  <span class="uuid-text">{{ truncateId(row.uuid) }}</span>
                </el-tooltip>
                <el-button
                  type="text"
                  size="small"
                  @click="copyToClipboard(row.uuid)"
                  style="margin-left: 8px;"
                >
                  <el-icon><DocumentCopy /></el-icon>
                </el-button>
              </template>
            </el-table-column>
            <el-table-column prop="file_name" :label="t('filename_label')" min-width="150" />
            <el-table-column prop="file_size" :label="t('file_size_col')" width="100">
              <template #default="{ row }">
                {{ formatFileSize(row.file_size) }}
              </template>
            </el-table-column>
            <el-table-column prop="duration" :label="t('duration_col')" width="80">
              <template #default="{ row }">
                {{ row.duration ? row.duration + 's' : '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="created_at" :label="t('created_at')" width="180">
              <template #default="{ row }">
                {{ formatDate(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column :label="t('actions')" width="180" fixed="right">
              <template #default="{ row }">
                <el-button
                  type="primary"
                  size="small"
                  link
                  @click="handlePlaySample(row)"
                >
                  <el-icon><VideoPlay /></el-icon>
                  {{ t('play') }}
                </el-button>
                <el-button
                  type="primary"
                  size="small"
                  link
                  @click="handleDownloadSample(row)"
                >
                  <el-icon><Download /></el-icon>
                  {{ t('download_btn') }}
                </el-button>
                <el-button
                  type="danger"
                  size="small"
                  link
                  @click="handleDeleteSample(row)"
                >
                  <el-icon><Delete /></el-icon>
                  {{ t('delete') }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <div v-if="samples.length === 0" class="empty-samples">
            <el-empty :description="t('no_samples_upload')" />
          </div>
        </div>
      </div>
    </el-drawer>

    <!-- Upload sample dialog -->
    <el-dialog
      v-model="showUploadDialog"
      :title="t('add_voiceprint_sample')"
      width="600px"
      :before-close="handleCloseUploadDialog"
    >
      <el-tabs v-model="uploadMode" class="upload-tabs">
        <!-- Select from history -->
        <el-tab-pane :label="t('select_from_history')" name="history">
          <div class="history-section">
            <el-form :model="historyForm" label-width="100px">
              <el-form-item :label="t('agent')">
                <el-select
                  v-model="historyForm.agent_id"
                  :placeholder="t('select_agent')"
                  style="width: 100%"
                  @change="loadHistoryMessages"
                  clearable
                >
                  <el-option
                    v-for="agent in agents"
                    :key="agent.id"
                    :label="agent.name"
                    :value="agent.id"
                  />
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
                @row-click="handleSelectHistoryMessage"
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
                  <template #default="{ row }">
                    {{ formatDate(row.created_at) }}
                  </template>
                </el-table-column>
                <el-table-column :label="t('actions')" width="100">
                  <template #default="{ row }">
                    <el-button
                      type="primary"
                      size="small"
                      link
                      @click.stop="handlePreviewHistoryAudio(row)"
                    >
                      <el-icon><VideoPlay /></el-icon>
                      {{ t('preview_btn') }}
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </div>
        </el-tab-pane>
        
        <!-- Upload file -->
        <el-tab-pane :label="t('upload_file')" name="upload">
          <el-form
            ref="uploadFormRef"
            :model="uploadForm"
            :rules="uploadRules"
            label-width="0"
          >
            <el-form-item prop="audio">
          <el-upload
            ref="uploadRef"
            :auto-upload="false"
            :on-change="handleFileChange"
            :on-remove="handleFileRemove"
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

        <!-- Record audio -->
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
          <el-button 
                v-if="!isRecording && !recordedBlob"
            type="primary" 
            size="large"
                @click="startRecording"
                :disabled="!canRecord"
              >
                <el-icon><VideoPlay /></el-icon>
                {{ t('start_recording') }}
              </el-button>
              <el-button
                v-if="isRecording"
                type="danger"
                size="large"
                @click="stopRecording"
              >
                <el-icon><VideoPause /></el-icon>
                {{ t('stop_record') }}
              </el-button>
              <el-button
                v-if="recordedBlob"
                type="primary"
                size="large"
                @click="startRecording"
                :disabled="!canRecord"
              >
                <el-icon><Refresh /></el-icon>
                {{ t('re_record') }}
          </el-button>
        </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <template #footer>
        <el-button @click="handleCloseUploadDialog">{{ t('cancel') }}</el-button>
        <el-button
          type="primary"
          @click="handleSubmitSample"
          :loading="submitting"
          :disabled="!hasAudioFile"
        >
          {{ t('confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Verify voiceprint group dialog -->
    <el-dialog
      v-model="showVerifyDialog"
      :title="t('verify_group_title', { name: currentVerifyGroup?.name || '' })"
      width="600px"
      :before-close="handleCloseVerifyDialog"
    >
      <el-tabs v-model="verifyMode" class="verify-tabs">
        <!-- Upload file -->
        <el-tab-pane :label="t('upload_file')" name="upload">
          <el-form
            ref="verifyFormRef"
            :model="verifyForm"
            :rules="verifyRules"
            label-width="0"
          >
            <el-form-item prop="audio">
              <el-upload
                ref="verifyUploadRef"
                :auto-upload="false"
                :on-change="handleVerifyFileChange"
                :on-remove="handleVerifyFileRemove"
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

        <!-- Record audio -->
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
              <el-button
                v-if="!isVerifyRecording && !verifyRecordedBlob"
                type="primary"
                size="large"
                @click="startVerifyRecording"
                :disabled="!canRecord"
              >
                <el-icon><VideoPlay /></el-icon>
                {{ t('start_recording') }}
              </el-button>
              <el-button
                v-if="isVerifyRecording"
                type="danger"
                size="large"
                @click="stopVerifyRecording"
              >
                <el-icon><VideoPause /></el-icon>
                {{ t('stop_record') }}
              </el-button>
              <el-button
                v-if="verifyRecordedBlob"
                type="primary"
                size="large"
                @click="startVerifyRecording"
                :disabled="!canRecord"
              >
                <el-icon><Refresh /></el-icon>
                {{ t('re_record') }}
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
        <el-button @click="handleCloseVerifyDialog">{{ t('cancel') }}</el-button>
        <el-button
          type="primary"
          @click="handleSubmitVerify"
          :loading="verifying"
          :disabled="!hasVerifyAudioFile"
        >{{ t('verify') }}</el-button>
      </template>
    </el-dialog>

    <!-- Hidden audio player -->
    <audio ref="audioPlayer" style="display: none;" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Plus, 
  Edit, 
  Delete, 
  View,
  Search,
  UploadFilled,
  Document,
  DocumentCopy,
  VideoPlay,
  Download,
  Microphone,
  CircleCheck,
  CircleClose,
  Refresh,
  VideoPause
} from '@element-plus/icons-vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const submitting = ref(false)
const speakerGroups = ref([])
const agents = ref([])
const samples = ref([])
const filterAgentId = ref('')
const searchKeyword = ref('')

// Dialog state
const showGroupDialog = ref(false)
const groupDialogMode = ref('add') // 'add' | 'edit'
const currentGroup = ref(null)
const showSampleDrawer = ref(false)
const showUploadDialog = ref(false)
const uploadMode = ref('history') // 'upload' | 'record' | 'history'

// Verify dialog state
const showVerifyDialog = ref(false)
const verifyMode = ref('upload') // 'upload' | 'record'
const currentVerifyGroup = ref(null)
const verifying = ref(false)
const verifyResult = ref(null)

// Verify form
const verifyForm = reactive({
  audioFile: null,
  audio: null
})

// Verify file list (for el-upload component)
const verifyFileList = ref([])

const verifyRules = {
  audio: [
    {
      validator: (rule, value, callback) => {
        if (!verifyForm.audioFile && !verifyRecordedBlob.value) {
          callback(new Error(t('upload_or_record_audio')))
        } else {
          callback()
        }
      },
      trigger: ['change', 'blur']
    }
  ]
}

// Verify recording state
const isVerifyRecording = ref(false)
const verifyMediaRecorder = ref(null)
const verifyRecordedBlob = ref(null)
const verifyRecordedBlobUrl = ref('')
const verifyRecordTime = ref(0)
const verifyRecordTimer = ref(null)

// Recording state
const isRecording = ref(false)
const mediaRecorder = ref(null)
const recordedBlob = ref(null)
const recordedBlobUrl = ref('')
const recordTime = ref(0)
const recordTimer = ref(null)
const canRecord = ref(false)

// Form refs
const groupFormRef = ref()
const uploadFormRef = ref()
const uploadRef = ref()
const verifyFormRef = ref()
const verifyUploadRef = ref()
const audioPlayer = ref()

// Voiceprint group form
const groupForm = reactive({
  agent_id: null,
  name: '',
  prompt: '',
  description: '',
  tts_config_id: null,
  voice: null
})

const groupRules = {
  agent_id: [
    { required: true, message: t('select_linked_agent'), trigger: 'change' }
  ],
  name: [
    { required: true, message: t('enter_voiceprint_name'), trigger: 'blur' },
    { min: 1, max: 100, message: t('length_1_to_100'), trigger: 'blur' }
  ]
}

// TTS config state
const ttsConfigs = ref([])
const currentVoiceOptions = ref([])
const cloneVoicePresets = ref([])
const cloneVoicesLoading = ref(false)

// Upload form
const uploadForm = reactive({
  audioFile: null,
  audio: null
})

const uploadRules = {
  audio: [
    { 
      validator: (rule, value, callback) => {
        if (!uploadForm.audioFile && !recordedBlob.value) {
          callback(new Error(t('upload_or_record_audio')))
        } else {
          callback()
        }
      }, 
      trigger: ['change', 'blur']
    }
  ]
}

// History state
const loadingHistory = ref(false)
const historyMessages = ref([])
const historyForm = reactive({
  agent_id: null,
  selected_message_id: null
})

// Whether audio file is available
const hasAudioFile = computed(() => {
  if (uploadMode.value === 'history') {
    return historyForm.selected_message_id !== null
  }
  return uploadForm.audioFile !== null || recordedBlob.value !== null
})

// Filtered voiceprint group list
const filteredGroups = computed(() => {
  let result = speakerGroups.value

  // Filter by agent
  if (filterAgentId.value) {
    result = result.filter(g => g.agent_id === filterAgentId.value)
  }

  // Filter by keyword
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

// Load agent list
const loadAgents = async () => {
  try {
    const response = await api.get('/user/agents')
    agents.value = response.data.data || []
  } catch (error) {
    console.error(t('load_agent_list_failed_v2'), error)
    ElMessage.error(t('load_agent_list_failed'))
  }
}

// Load TTS config list
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

const applyCloneVoice = async (clone) => {
  if (!clone) return
  const ttsConfig = ttsConfigs.value.find(config => config.config_id === clone.tts_config_id)
  if (!ttsConfig) {
    return
  }
  groupForm.tts_config_id = clone.tts_config_id
  await handleTtsConfigChange(clone.tts_config_id)
  groupForm.voice = clone.provider_voice_id
}

// Load voice options when TTS config changes
const handleTtsConfigChange = async (configId) => {
  if (!configId) {
    currentVoiceOptions.value = []
    groupForm.voice = null
    return
  }
  
  const config = ttsConfigs.value.find(c => c.config_id === configId)
  if (!config) {
    currentVoiceOptions.value = []
    return
  }

  try {
    // Fetch full voice list for provider from backend
    const params = { provider: config.provider }
    // Always include config_id parameter
    if (configId) {
      params.config_id = configId
    }
    const response = await api.get('/user/voice-options', { params })
    currentVoiceOptions.value = response.data.data || []
  } catch (error) {
    console.error(t('load_voice_list_failed_c'), error)
    currentVoiceOptions.value = []
    ElMessage.warning(t('load_voice_list_failed'))
  }
}

// Extract voice options by provider
const extractVoiceOptions = (provider, config) => {
  const options = []

  if (!config) return options

  // Extract voices by TTS provider
  switch (provider) {
    case 'edge':
    case 'microsoft':
      // Edge TTS common voices
      if (config.voice) {
        options.push({ label: config.voice, value: config.voice })
      }
      // Add common voices
      const edgeVoices = [
        { label: t('voice_xiaoxiao'), value: 'zh-CN-XiaoxiaoNeural' },
        { label: t('voice_yunxi'), value: 'zh-CN-YunxiNeural' },
        { label: t('voice_yunyang'), value: 'zh-CN-YunyangNeural' },
        { label: t('voice_xiaoyi'), value: 'zh-CN-XiaoyiNeural' },
        { label: t('voice_yunjian'), value: 'zh-CN-YunjianNeural' },
        { label: t('voice_xiaochen'), value: 'zh-CN-XiaochenNeural' },
        { label: t('voice_xiaohan'), value: 'zh-CN-XiaohanNeural' }
      ]
      edgeVoices.forEach(v => {
        if (!options.find(o => o.value === v.value)) {
          options.push(v)
        }
      })
      break
      
    case 'doubao':
    case 'doubao_ws':
      // Doubao TTS voices
      if (config.voice) {
        options.push({ label: config.voice, value: config.voice })
      }
      const doubaoVoices = [
        { label: t('voice_sweet_female'), value: 'zh_female_shuangkuaisisi_moon_bigtts' },
        { label: t('voice_bv700v2_male'), value: 'BV700_V2_streaming' },
        { label: t('voice_bv001_female'), value: 'BV001_streaming' },
        { label: t('voice_bv002_male'), value: 'BV002_streaming' }
      ]
      doubaoVoices.forEach(v => {
        if (!options.find(o => o.value === v.value)) {
          options.push(v)
        }
      })
      break
      
    case 'cosyvoice':
      // CosyVoice uses spk_id
      if (config.spk_id) {
        options.push({ label: config.spk_id, value: config.spk_id })
      }
      const cosyVoices = [
        { label: t('chinese_female'), value: t('chinese_female') },
        { label: t('chinese_male'), value: t('chinese_male') },
        { label: t('cantonese_female'), value: t('cantonese_female') },
        { label: t('english_female'), value: t('english_female') },
        { label: t('english_male'), value: t('english_male') },
        { label: t('japanese_male'), value: t('japanese_male') },
        { label: t('korean_female'), value: t('korean_female') }
      ]
      cosyVoices.forEach(v => {
        if (!options.find(o => o.value === v.value)) {
          options.push(v)
        }
      })
      break
      
    case 'minimax':
      // Minimax TTS uses voice
      if (config.voice) {
        options.push({ label: config.voice, value: config.voice })
      }
      const minimaxVoices = [
        { label: t('voice_fresh_male'), value: 'male-qn-qingse' },
        { label: t('voice_fresh_female'), value: 'female-qn-qingse' },
        { label: t('voice_young_male'), value: 'male-shaonian' },
        { label: t('voice_young_female'), value: 'female-shaonian' },
        { label: t('voice_mature_male'), value: 'male-chengshu' },
        { label: t('voice_mature_female'), value: 'female-chengshu' },
        { label: t('voice_warm_male'), value: 'male-wennuan' },
        { label: t('voice_warm_female'), value: 'female-wennuan' },
        { label: t('voice_clear_male'), value: 'male-qinglang' },
        { label: t('voice_clear_female'), value: 'female-qinglang' },
        { label: t('voice_heavy_male'), value: 'male-houzhong' },
        { label: t('voice_heavy_female'), value: 'female-houzhong' }
      ]
      minimaxVoices.forEach(v => {
        if (!options.find(o => o.value === v.value)) {
          options.push(v)
        }
      })
      break
      
    default:
      // Other providers: try extracting from config
      if (config.voice) {
        options.push({ label: config.voice, value: config.voice })
      }
      if (config.spk_id) {
        options.push({ label: config.spk_id, value: config.spk_id })
      }
  }
  
  return options
}

// Get current TTS config name
const getCurrentTtsConfigName = () => {
  if (!groupForm.tts_config_id) return ''
  const config = ttsConfigs.value.find(c => c.config_id === groupForm.tts_config_id)
  return config ? config.name : ''
}

// Get current TTS config info
const getCurrentTtsConfigInfo = () => {
  if (!groupForm.tts_config_id) return ''
  const config = ttsConfigs.value.find(c => c.config_id === groupForm.tts_config_id)
  if (!config) return ''
  return t('tts_provider_label', { provider: config.provider || t('unknown') })
}

// Load voiceprint group list
const loadSpeakerGroups = async () => {
  try {
    loading.value = true
    const params = {}
    if (filterAgentId.value) {
      params.agent_id = filterAgentId.value
    }
    const response = await api.get('/user/speaker-groups', { params })
    speakerGroups.value = response.data.data || []
  } catch (error) {
    console.error(t('load_voiceprint_group_failed'), error)
    ElMessage.error(t('load_voiceprint_group_failed') + ' ' + (error.response?.data?.error || error.message))
  } finally {
    loading.value = false
  }
}

// Search handler
const handleSearch = () => {
  // Search is client-side filtering, no request needed
}

// Create voiceprint group
const handleAddGroup = async () => {
  groupDialogMode.value = 'add'
  resetGroupForm()
  await loadCloneVoicePresets()
  showGroupDialog.value = true
}

// Edit voiceprint group
const handleEditGroup = async (group) => {
  groupDialogMode.value = 'edit'
  currentGroup.value = group
  groupForm.agent_id = group.agent_id
  groupForm.name = group.name
  groupForm.prompt = group.prompt || ''
  groupForm.description = group.description || ''
  groupForm.tts_config_id = group.tts_config_id || null
  groupForm.voice = group.voice || null
  await loadCloneVoicePresets()

  // If TTS config set, load voice options
  if (groupForm.tts_config_id) {
    await handleTtsConfigChange(groupForm.tts_config_id)
  }
  
  showGroupDialog.value = true
}

// Submit voiceprint group form
const handleSubmitGroup = async () => {
  if (!groupFormRef.value) return

  try {
    await groupFormRef.value.validate()
    submitting.value = true

    if (groupDialogMode.value === 'add') {
      const response = await api.post('/user/speaker-groups', groupForm)
      ElMessage.success(t('create_success'))
      showGroupDialog.value = false
      await loadSpeakerGroups()
    } else {
      const response = await api.put(`/user/speaker-groups/${currentGroup.value.id}`, groupForm)
      ElMessage.success(t('update_success'))
      showGroupDialog.value = false
      await loadSpeakerGroups()
    }
  } catch (error) {
    if (error.fields) {
      // Form validation error
      return
    }
    console.error(t('submit_failed'), error)
    ElMessage.error(t('operation_failed_colon') + ' ' + (error.response?.data?.error || error.message))
  } finally {
    submitting.value = false
  }
}

// Verify voiceprint group
const handleVerifyGroup = async (group) => {
  // Reset previous data first
  resetVerifyForm()

  // Wait for DOM update
  await nextTick()
  
  currentVerifyGroup.value = group
  verifyResult.value = null
  verifyMode.value = 'upload'
  showVerifyDialog.value = true
  
  // Ensure upload component is cleared
  await nextTick()
  verifyUploadRef.value?.clearFiles()
  verifyFileList.value = []

  // Check browser recording support
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    stream.getTracks().forEach(track => track.stop())
    canRecord.value = true
  } catch (error) {
    console.warn(t('browser_recording_error'), error)
    canRecord.value = false
    if (verifyMode.value === 'record') {
      ElMessage.warning(t('browser_no_recording'))
      verifyMode.value = 'upload'
    }
  }
}

// Close verify dialog
const handleCloseVerifyDialog = () => {
  if (isVerifyRecording.value) {
    stopVerifyRecording()
  }
  resetVerifyForm()
  showVerifyDialog.value = false
}

// Verify file change handler
const handleVerifyFileChange = async (file, fileList) => {
  // Clear file list first to remove old file
  verifyFileList.value = []
  await nextTick()

  // Clear existing file if present
  if (verifyForm.audioFile) {
    verifyForm.audioFile = null
    verifyForm.audio = null
  }

  // Clear recording state
  if (verifyRecordedBlob.value) {
    if (verifyRecordedBlobUrl.value) {
      URL.revokeObjectURL(verifyRecordedBlobUrl.value)
      verifyRecordedBlobUrl.value = ''
    }
    verifyRecordedBlob.value = null
    verifyRecordTime.value = 0
  }
  
  // Clear verification result
  verifyResult.value = null

  const fileObj = file.raw || file
  if (!fileObj) {
    ElMessage.warning(t('invalid_file_object'))
    verifyUploadRef.value?.clearFiles()
    verifyForm.audioFile = null
    verifyFileList.value = []
    return
  }

  // Validate file type
  const fileName = fileObj.name || file.name || ''
  const fileType = fileObj.type || file.type || ''
  if (!fileType.includes('wav') && !fileName.toLowerCase().endsWith('.wav')) {
    ElMessage.warning(t('wav_only_upload'))
    verifyUploadRef.value?.clearFiles()
    verifyForm.audioFile = null
    verifyFileList.value = []
    return
  }

  // Validate file size (10MB limit)
  const fileSize = fileObj.size || file.size || 0
  if (fileSize > 10 * 1024 * 1024) {
    ElMessage.warning(t('file_size_limit'))
    verifyUploadRef.value?.clearFiles()
    verifyForm.audioFile = null
    verifyFileList.value = []
    return
  }

  // Set new file
  verifyForm.audioFile = file
  verifyForm.audio = file

  // Update file list display (latest file only)
  verifyFileList.value = [file]
  
  await nextTick()

  if (verifyFormRef.value) {
    verifyFormRef.value.clearValidate('audio')
  }
}

// Verify file remove handler
const handleVerifyFileRemove = () => {
  verifyForm.audioFile = null
  verifyForm.audio = null
  verifyFileList.value = []
  verifyResult.value = null // clear verification result
  if (verifyFormRef.value) {
    verifyFormRef.value.validateField('audio')
  }
}

// Start verify recording
const startVerifyRecording = async () => {
  try {
    // Stop previous recording if active
    if (verifyMediaRecorder.value && verifyMediaRecorder.value.state !== 'inactive') {
      verifyMediaRecorder.value.stop()
    }

    // Clean up previous recording
    if (verifyRecordedBlobUrl.value) {
      URL.revokeObjectURL(verifyRecordedBlobUrl.value)
      verifyRecordedBlobUrl.value = ''
    }
    verifyRecordedBlob.value = null
    verifyRecordTime.value = 0

    // Request microphone permission
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        channelCount: 1,
        sampleRate: 16000,
        echoCancellation: true,
        noiseSuppression: true
      }
    })

    // Create MediaRecorder
    const chunks = []
    const options = {
      mimeType: 'audio/webm;codecs=opus'
    }

    if (!MediaRecorder.isTypeSupported(options.mimeType)) {
      verifyMediaRecorder.value = new MediaRecorder(stream)
    } else {
      verifyMediaRecorder.value = new MediaRecorder(stream, options)
    }

    verifyMediaRecorder.value.ondataavailable = (e) => {
      if (e.data.size > 0) {
        chunks.push(e.data)
      }
    }

    verifyMediaRecorder.value.onstop = async () => {
      stream.getTracks().forEach(track => track.stop())
      
      try {
        // Convert recorded audio to WAV
        const blob = new Blob(chunks, { type: chunks[0]?.type || 'audio/webm' })
        const wavBlob = await convertToWav(blob)
        
        verifyRecordedBlob.value = wavBlob
        verifyRecordedBlobUrl.value = URL.createObjectURL(wavBlob)
        
        // Create File object for upload
        const fileName = `verify_recording_${Date.now()}.wav`
        const file = new File([wavBlob], fileName, { type: 'audio/wav' })
        verifyForm.audioFile = { raw: file, name: fileName, size: wavBlob.size }
        verifyForm.audio = file

        if (verifyFormRef.value) {
          verifyFormRef.value.clearValidate('audio')
        }
      } catch (error) {
        console.error(t('process_recording_failed_v2'), error)
        ElMessage.error(t('process_recording_failed'))
        verifyRecordedBlob.value = null
        verifyRecordedBlobUrl.value = ''
        verifyForm.audioFile = null
        verifyForm.audio = null
      }

      chunks.length = 0
    }

    // Start recording
    verifyMediaRecorder.value.start(100)
    isVerifyRecording.value = true

    // Start timer
    verifyRecordTimer.value = setInterval(() => {
      verifyRecordTime.value += 0.1
    }, 100)

    ElMessage.success(t('start_recording'))
  } catch (error) {
    console.error(t('recording_failed'), error)
    ElMessage.error(t('recording_failed') + ' ' + error.message)
    canRecord.value = false
  }
}

// Stop verify recording
const stopVerifyRecording = () => {
  if (verifyMediaRecorder.value && verifyMediaRecorder.value.state !== 'inactive') {
    verifyMediaRecorder.value.stop()
  }
  isVerifyRecording.value = false
  
  if (verifyRecordTimer.value) {
    clearInterval(verifyRecordTimer.value)
    verifyRecordTimer.value = null
  }

  ElMessage.success(t('recording_complete'))
}

// Submit verification
const handleSubmitVerify = async () => {
  if (!verifyFormRef.value) return

  try {
    await verifyFormRef.value.validate()

    if (!verifyForm.audioFile && !verifyRecordedBlob.value) {
      ElMessage.warning(t('upload_or_record_audio'))
      return
    }

    verifying.value = true
    verifyResult.value = null

    let file
    if (verifyForm.audioFile) {
      // Use uploaded file
      file = verifyForm.audioFile.raw || verifyForm.audioFile
    } else if (verifyRecordedBlob.value) {
      // Use recorded audio
      const fileName = `verify_recording_${Date.now()}.wav`
      file = new File([verifyRecordedBlob.value], fileName, { type: 'audio/wav' })
    } else {
      ElMessage.warning(t('upload_or_record_audio'))
      return
    }

    const formData = new FormData()
    formData.append('audio', file)

    const response = await api.post(`/user/speaker-groups/${currentVerifyGroup.value.id}/verify`, formData)
    
    if (response.data.success && response.data.data) {
      verifyResult.value = {
        verified: response.data.data.verified,
        confidence: response.data.data.confidence,
        threshold: response.data.data.threshold,
        message: response.data.data.message
      }
      
      if (verifyResult.value.verified) {
        ElMessage.success(t('verify_passed'))
      } else {
        ElMessage.warning(t('verify_not_passed'))
      }
    } else {
      ElMessage.error(t('verify_failed'))
    }
  } catch (error) {
    if (error.fields) {
      return
    }
    console.error(t('verify_failed_colon'), error)
    ElMessage.error(t('verify_failed_colon') + ' ' + (error.response?.data?.error || error.message))
  } finally {
    verifying.value = false
  }
}

// Reset verify form
const resetVerifyForm = () => {
  if (verifyFormRef.value) {
    verifyFormRef.value.resetFields()
  }
  if (verifyUploadRef.value) {
    verifyUploadRef.value.clearFiles()
  }
  verifyForm.audioFile = null
  verifyForm.audio = null
  
  // Clean up verify recording state
  if (isVerifyRecording.value) {
    stopVerifyRecording()
  }
  if (verifyRecordedBlobUrl.value) {
    URL.revokeObjectURL(verifyRecordedBlobUrl.value)
    verifyRecordedBlobUrl.value = ''
  }
  verifyRecordedBlob.value = null
  verifyRecordTime.value = 0
  verifyMode.value = 'upload'
  verifyResult.value = null
}

// Whether verify audio file is available
const hasVerifyAudioFile = computed(() => {
  return verifyForm.audioFile !== null || verifyRecordedBlob.value !== null
})

// Delete voiceprint group
const handleDeleteGroup = async (group) => {
  try {
    await ElMessageBox.confirm(
      t('confirm_delete_group', { name: group.name }),
      t('confirm_delete'),
      {
        confirmButtonText: t('confirm'),
        cancelButtonText: t('cancel'),
        type: 'warning'
      }
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

// View samples
const handleViewSamples = async (group) => {
  currentGroup.value = group
  showSampleDrawer.value = true
  await loadSamples(group.id)
}

// Verify voiceprint group from sample drawer
const handleVerifyFromSamples = () => {
  if (currentGroup.value) {
    showSampleDrawer.value = false
    handleVerifyGroup(currentGroup.value)
  }
}

// Load sample list
const loadSamples = async (groupId) => {
  try {
    const response = await api.get(`/user/speaker-groups/${groupId}/samples`)
    samples.value = response.data.data || []
  } catch (error) {
    console.error(t('load_sample_list_failed_v2'), error)
    ElMessage.error(t('load_sample_list_failed'))
  }
}

// Close sample drawer
const handleCloseSampleDrawer = () => {
  showSampleDrawer.value = false
  currentGroup.value = null
  samples.value = []
}

// Add sample
const handleAddSample = async () => {
  resetUploadForm()
  uploadMode.value = 'history'
  showUploadDialog.value = true

  // Initialize history form
  historyForm.agent_id = currentGroup.value?.agent_id || null
  historyForm.selected_message_id = null
  historyMessages.value = []

  // Auto-load history if group has an associated agent
  if (currentGroup.value?.agent_id) {
    historyForm.agent_id = currentGroup.value.agent_id
    await loadHistoryMessages()
  }

  // Check browser recording support
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    stream.getTracks().forEach(track => track.stop())
    canRecord.value = true
  } catch (error) {
    console.warn(t('browser_recording_error'), error)
    canRecord.value = false
    if (uploadMode.value === 'record') {
      ElMessage.warning(t('browser_no_recording'))
      uploadMode.value = 'upload'
    }
  }
}

// Close upload dialog
const handleCloseUploadDialog = () => {
  if (isRecording.value) {
    stopRecording()
  }
  resetUploadForm()
  showUploadDialog.value = false
}

// File change handler
const handleFileChange = (file) => {
  const fileObj = file.raw || file
  if (!fileObj) {
    ElMessage.warning(t('invalid_file_object'))
    uploadRef.value?.clearFiles()
    uploadForm.audioFile = null
      return
    }

  // Validate file type
  const fileName = fileObj.name || file.name || ''
  const fileType = fileObj.type || file.type || ''
  if (!fileType.includes('wav') && !fileName.toLowerCase().endsWith('.wav')) {
    ElMessage.warning(t('wav_only_upload'))
    uploadRef.value?.clearFiles()
    uploadForm.audioFile = null
    return
  }

  // Validate file size (10MB limit)
  const fileSize = fileObj.size || file.size || 0
  if (fileSize > 10 * 1024 * 1024) {
    ElMessage.warning(t('file_size_limit'))
    uploadRef.value?.clearFiles()
    uploadForm.audioFile = null
    return
  }

  uploadForm.audioFile = file
  uploadForm.audio = file

  if (uploadFormRef.value) {
    uploadFormRef.value.clearValidate('audio')
  }
}

// File remove handler
const handleFileRemove = () => {
  uploadForm.audioFile = null
  uploadForm.audio = null
  if (uploadFormRef.value) {
    uploadFormRef.value.validateField('audio')
  }
}

// Start recording
const startRecording = async () => {
  try {
    // Stop previous recording if active
    if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') {
      mediaRecorder.value.stop()
    }

    // Clean up previous recording
    if (recordedBlobUrl.value) {
      URL.revokeObjectURL(recordedBlobUrl.value)
      recordedBlobUrl.value = ''
    }
    recordedBlob.value = null
    recordTime.value = 0

    // Request microphone permission
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        channelCount: 1,
        sampleRate: 16000,
        echoCancellation: true,
        noiseSuppression: true
      }
    })

    // Create MediaRecorder (WAV format)
    const chunks = []
    const options = {
      mimeType: 'audio/webm;codecs=opus' // record as webm, then convert to WAV
    }

    // Check browser support
    if (!MediaRecorder.isTypeSupported(options.mimeType)) {
      // Fall back to default format if unsupported
      mediaRecorder.value = new MediaRecorder(stream)
      } else {
      mediaRecorder.value = new MediaRecorder(stream, options)
    }

    mediaRecorder.value.ondataavailable = (e) => {
      if (e.data.size > 0) {
        chunks.push(e.data)
      }
    }

    mediaRecorder.value.onstop = async () => {
      stream.getTracks().forEach(track => track.stop())
      
      try {
        // Convert recorded audio to WAV
        const blob = new Blob(chunks, { type: chunks[0]?.type || 'audio/webm' })
        const wavBlob = await convertToWav(blob)
        
        recordedBlob.value = wavBlob
        recordedBlobUrl.value = URL.createObjectURL(wavBlob)
        
        // Create File object for upload
        const fileName = `recording_${Date.now()}.wav`
        const file = new File([wavBlob], fileName, { type: 'audio/wav' })
        uploadForm.audioFile = { raw: file, name: fileName, size: wavBlob.size }
        uploadForm.audio = file

        if (uploadFormRef.value) {
          uploadFormRef.value.clearValidate('audio')
        }
      } catch (error) {
        console.error(t('process_recording_failed_v2'), error)
        ElMessage.error(t('process_recording_failed'))
        recordedBlob.value = null
        recordedBlobUrl.value = ''
        uploadForm.audioFile = null
        uploadForm.audio = null
      }

      chunks.length = 0
    }

    // Start recording
    mediaRecorder.value.start(100) // collect data every 100ms
    isRecording.value = true

    // Start timer
    recordTimer.value = setInterval(() => {
      recordTime.value += 0.1
    }, 100)

    ElMessage.success(t('start_recording'))
  } catch (error) {
    console.error(t('recording_failed'), error)
    ElMessage.error(t('recording_failed') + ' ' + error.message)
    canRecord.value = false
  }
}

// Stop recording
const stopRecording = () => {
  if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') {
    mediaRecorder.value.stop()
  }
  isRecording.value = false
  
  if (recordTimer.value) {
    clearInterval(recordTimer.value)
    recordTimer.value = null
  }

  ElMessage.success(t('recording_complete'))
}

// Convert audio to WAV format
const convertToWav = async (blob) => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = async (e) => {
      try {
        const audioContext = new (window.AudioContext || window.webkitAudioContext)()
        const arrayBuffer = e.target.result
        const audioBuffer = await audioContext.decodeAudioData(arrayBuffer)
        
        // Convert to WAV
        const wav = audioBufferToWav(audioBuffer)
        const wavBlob = new Blob([wav], { type: 'audio/wav' })
        resolve(wavBlob)
      } catch (error) {
        console.error(t('convert_wav_failed_v2'), error)
        // On failure, use raw blob (backend may support webm)
        reject(error)
      }
    }
    reader.onerror = reject
    reader.readAsArrayBuffer(blob)
  })
}

// Convert AudioBuffer to WAV format
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

  // WAV file header
  const writeString = (offset, string) => {
    for (let i = 0; i < string.length; i++) {
      view.setUint8(offset + i, string.charCodeAt(i))
    }
  }

  writeString(0, 'RIFF')
  view.setUint32(4, bufferSize - 8, true)
  writeString(8, 'WAVE')
  writeString(12, 'fmt ')
  view.setUint32(16, 16, true) // fmt chunk size
  view.setUint16(20, 1, true) // audio format (PCM)
  view.setUint16(22, numberOfChannels, true)
  view.setUint32(24, sampleRate, true)
  view.setUint32(28, byteRate, true)
  view.setUint16(32, blockAlign, true)
  view.setUint16(34, 16, true) // bits per sample
  writeString(36, 'data')
  view.setUint32(40, dataSize, true)

  // Write audio data
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

// Format recording duration
const formatRecordTime = (seconds) => {
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  const ms = Math.floor((seconds % 1) * 10)
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}.${ms}`
}

// Load chat history
const loadHistoryMessages = async () => {
  if (!historyForm.agent_id) {
    historyMessages.value = []
    return
  }

  try {
    loadingHistory.value = true
    const response = await api.get('/user/history/messages', {
      params: {
        agent_id: historyForm.agent_id,
        role: 'user',
        page: 1,
        page_size: 50
      }
    })
    
    // Only show messages with audio
    historyMessages.value = (response.data.data || []).filter(msg => msg.audio_path)
  } catch (error) {
    console.error(t('load_chat_history_failed'), error)
    ElMessage.error(t('load_chat_history_failed') + ' ' + (error.response?.data?.error || error.message))
    historyMessages.value = []
  } finally {
    loadingHistory.value = false
  }
}

// Select history message
const handleSelectHistoryMessage = (row) => {
  historyForm.selected_message_id = row.message_id
}

// Preview history audio
const handlePreviewHistoryAudio = async (message) => {
  try {
    const response = await api.get(`/user/history/messages/${message.id}/audio`, {
      responseType: 'blob'
    })
    
    const blob = new Blob([response.data], { type: 'audio/wav' })
    const blobUrl = URL.createObjectURL(blob)
    
    audioPlayer.value.src = blobUrl
    audioPlayer.value.play().catch(err => {
      console.error(t('play_failed_colon'), err)
      ElMessage.warning(t('play_failed_check_audio'))
    })
    
    audioPlayer.value.onended = () => {
      URL.revokeObjectURL(blobUrl)
    }
  } catch (error) {
    console.error(t('preview_failed'), error)
    ElMessage.error(t('preview_failed') + ' ' + (error.response?.data?.error || error.message))
  }
}

// Submit sample
const handleSubmitSample = async () => {
  if (uploadMode.value === 'history') {
    // Select from history
    if (!historyForm.selected_message_id) {
      ElMessage.warning(t('select_chat_history'))
      return
    }

    try {
      submitting.value = true
      const formData = new FormData()
      formData.append('message_id', historyForm.selected_message_id)

      await api.post(`/user/speaker-groups/${currentGroup.value.id}/samples`, formData)
      ElMessage.success(t('add_success'))
      handleCloseUploadDialog()
      await loadSamples(currentGroup.value.id)
      await loadSpeakerGroups() // refresh to update sample count
    } catch (error) {
      console.error(t('add_failed'), error)
      ElMessage.error(t('add_failed') + ' ' + (error.response?.data?.error || error.message))
    } finally {
      submitting.value = false
    }
    return
  }

  // Upload/record flow
  if (!uploadFormRef.value) return

  try {
    await uploadFormRef.value.validate()

    if (!uploadForm.audioFile && !recordedBlob.value) {
      ElMessage.warning(t('upload_or_record_audio'))
      return
    }

    submitting.value = true

    let file
    if (uploadForm.audioFile) {
      // Use uploaded file
      file = uploadForm.audioFile.raw || uploadForm.audioFile
    } else if (recordedBlob.value) {
      // Use recorded audio
      const fileName = `recording_${Date.now()}.wav`
      file = new File([recordedBlob.value], fileName, { type: 'audio/wav' })
    } else {
      ElMessage.warning(t('upload_or_record_audio'))
      return
    }

    const formData = new FormData()
    formData.append('audio', file)

    await api.post(`/user/speaker-groups/${currentGroup.value.id}/samples`, formData)
    ElMessage.success(t('upload_success'))
    handleCloseUploadDialog()
    await loadSamples(currentGroup.value.id)
    await loadSpeakerGroups() // Refresh list to update sample counts
  } catch (error) {
    if (error.fields) {
      return
    }
    console.error(t('upload_failed'), error)
    ElMessage.error(t('upload_failed') + ' ' + (error.response?.data?.error || error.message))
  } finally {
    submitting.value = false
  }
}

// Play sample
const handlePlaySample = async (sample) => {
  try {
    // Build audio file URL (backend must provide file access endpoint)
    // Fetch file via api.get and create blob URL
    const response = await api.get(
      `/user/speaker-groups/${currentGroup.value.id}/samples/${sample.id}/file`,
      {
        responseType: 'blob'
      }
    )
    
    // Create blob URL
    const blob = new Blob([response.data], { type: 'audio/wav' })
    const blobUrl = URL.createObjectURL(blob)

    audioPlayer.value.src = blobUrl
    audioPlayer.value.play().catch(err => {
      console.error(t('play_failed_colon'), err)
      ElMessage.warning(t('play_failed_check_audio'))
    })

    // Revoke blob URL on playback end
    audioPlayer.value.onended = () => {
      URL.revokeObjectURL(blobUrl)
    }
  } catch (error) {
    console.error(t('play_failed_colon'), error)
    ElMessage.error(t('play_failed_colon') + ' ' + (error.response?.data?.error || error.message))
  }
}

// Download sample
const handleDownloadSample = async (sample) => {
  try {
    // Fetch file via api.get and trigger download
    const response = await api.get(
      `/user/speaker-groups/${currentGroup.value.id}/samples/${sample.id}/file`,
      {
        responseType: 'blob'
      }
    )
    
    // Create blob URL and trigger download
    const blob = new Blob([response.data], { type: 'audio/wav' })
    const blobUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = sample.file_name || 'audio.wav'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)

    // Revoke blob URL
    setTimeout(() => {
      URL.revokeObjectURL(blobUrl)
    }, 100)
  } catch (error) {
    console.error(t('download_failed'), error)
    ElMessage.error(t('download_failed') + ' ' + (error.response?.data?.error || error.message))
  }
}

// Delete sample
const handleDeleteSample = async (sample) => {
  try {
    await ElMessageBox.confirm(
      t('confirm_delete_sample', { name: sample.file_name }),
      t('confirm_delete'),
      {
        confirmButtonText: t('confirm'),
        cancelButtonText: t('cancel'),
        type: 'warning'
      }
    )

    await api.delete(`/user/speaker-groups/${currentGroup.value.id}/samples/${sample.id}`)
    ElMessage.success(t('delete_success'))
    await loadSamples(currentGroup.value.id)
    await loadSpeakerGroups() // Refresh list to update sample counts
  } catch (error) {
    if (error !== 'cancel') {
      console.error(t('delete_failed_colon'), error)
      ElMessage.error(t('delete_failed_colon') + ' ' + (error.response?.data?.error || error.message))
    }
  }
}

// Copy to clipboard
const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('copied_to_clipboard'))
  } catch (error) {
    console.error(t('copy_failed_v2'), error)
    ElMessage.error(t('copy_failed'))
  }
}

// Reset form
const resetGroupForm = () => {
  if (groupFormRef.value) {
    groupFormRef.value.resetFields()
  }
  Object.assign(groupForm, {
    agent_id: null,
    name: '',
    prompt: '',
    description: '',
    tts_config_id: null,
    voice: null
  })
  currentGroup.value = null
  currentVoiceOptions.value = []
}

const resetUploadForm = () => {
  if (uploadFormRef.value) {
    uploadFormRef.value.resetFields()
  }
  if (uploadRef.value) {
    uploadRef.value.clearFiles()
  }
  uploadForm.audioFile = null
  uploadForm.audio = null

  // Clean up recording state
  if (isRecording.value) {
    stopRecording()
  }
  if (recordedBlobUrl.value) {
    URL.revokeObjectURL(recordedBlobUrl.value)
    recordedBlobUrl.value = ''
  }
  recordedBlob.value = null
  recordTime.value = 0
  uploadMode.value = 'history'

  // Clean up history state
  historyForm.agent_id = null
  historyForm.selected_message_id = null
  historyMessages.value = []
}

// Format date
const formatDate = (dateString) => {
  if (!dateString) return '-'
  return new Date(dateString).toLocaleString('zh-CN')
}

// Truncate ID for display
const truncateId = (id) => {
  if (!id) return '-'
  if (id.length > 20) {
    return id.substring(0, 10) + '...' + id.substring(id.length - 10)
  }
  return id
}

// Truncate text
const truncateText = (text, maxLength) => {
  if (!text) return '-'
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

// Format file size
const formatFileSize = (bytes) => {
  if (!bytes) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

onMounted(() => {
  loadAgents()
  loadSpeakerGroups()
  loadTtsConfigs()
  loadCloneVoicePresets()
})

// Clean up resources before unmount
onBeforeUnmount(() => {
  if (isRecording.value) {
    stopRecording()
  }
  if (recordedBlobUrl.value) {
    URL.revokeObjectURL(recordedBlobUrl.value)
  }
  if (recordTimer.value) {
    clearInterval(recordTimer.value)
  }
  if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') {
    mediaRecorder.value.stop()
  }
})
</script>

<style scoped>
.speakers-page {
  padding: 0;
}

.filter-bar {
  padding: 15px 20px;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  margin-bottom: 20px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.create-group-button {
  margin-left: auto;
}

.speakers-content {
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  padding: 20px;
}

.prompt-text {
  color: #606266;
  cursor: pointer;
}

.prompt-popover {
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.text-muted {
  color: #909399;
}

.uuid-text {
  font-family: monospace;
  font-size: 12px;
}

.empty-state {
  padding: 40px 0;
}

.sample-drawer {
  padding: 20px;
}

.group-info-card {
  margin-bottom: 20px;
}

.group-info h3 {
  margin: 0 0 15px 0;
  color: #303133;
}

.prompt-section,
.description-section {
  margin-top: 15px;
  padding-top: 15px;
  border-top: 1px solid #f0f0f0;
}

.prompt-section strong,
.description-section strong {
  display: block;
  margin-bottom: 8px;
  color: #606266;
}

.prompt-section p,
.description-section p {
  margin: 0;
  color: #303133;
  white-space: pre-wrap;
  word-break: break-word;
}

.samples-section {
  margin-top: 20px;
}

.samples-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.samples-header h4 {
  margin: 0;
  color: #303133;
}

.empty-samples {
  padding: 40px 0;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  padding: 8px 12px;
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  border-radius: 12px;
  font-size: 14px;
  color: var(--apple-text-secondary);
}

.file-size {
  color: #909399;
  font-size: 12px;
}

.clone-voice-line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  width: 100%;
}

.clone-voice-item {
  display: inline-flex;
  align-items: center;
  max-width: 220px;
  min-width: 0;
  padding: 4px 10px;
  border: 1px solid #d1d5db;
  border-radius: 999px;
  background: #f8fafc;
  color: #374151;
  cursor: pointer;
  transition: all 0.2s ease;
  line-height: 1.2;
  outline: none;
}

.clone-voice-item:hover {
  border-color: #93c5fd;
  background: #f1f7ff;
}

.clone-voice-item.active {
  border-color: #3b82f6;
  background: #e9f2ff;
  color: #1d4ed8;
  box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.1);
}

.clone-voice-name {
  font-size: 12px;
  font-weight: 500;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.el-upload-dragger) {
  width: 100%;
}

.action-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.action-buttons .el-button {
  margin: 0;
  white-space: nowrap;
}

/* Upload dialog styles */
.upload-tabs {
  margin-top: 10px;
}

.audio-upload {
  width: 100%;
}

.audio-upload :deep(.el-upload-dragger) {
  width: 100%;
  padding: 40px 20px;
}

.audio-upload :deep(.el-icon--upload) {
  font-size: 48px;
  color: var(--apple-primary);
  margin-bottom: 16px;
}

.audio-upload :deep(.el-upload__text) {
  font-size: 14px;
  color: #606266;
}

.audio-upload :deep(.el-upload__text em) {
  color: var(--apple-primary);
  font-style: normal;
}

.audio-upload :deep(.el-upload__tip) {
  margin-top: 12px;
  font-size: 12px;
  color: #909399;
}

/* Recording area styles */
.record-section {
  padding: 20px 0;
}

.record-status {
  min-height: 200px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 30px;
  background: #f5f7fa;
  border-radius: 8px;
  margin-bottom: 20px;
}

.record-ready,
.record-complete {
  text-align: center;
}

.record-ready p,
.record-complete p {
  margin: 12px 0 0 0;
  color: #303133;
  font-size: 16px;
}

.record-tip {
  margin-top: 8px !important;
  font-size: 14px !important;
  color: #909399 !important;
}

.record-recording {
  text-align: center;
}

.recording-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-bottom: 16px;
}

.recording-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--apple-danger);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(1.2);
  }
}

.recording-text {
  font-size: 16px;
  color: var(--apple-danger);
  font-weight: 500;
}

.record-time {
  font-size: 32px;
  font-weight: 600;
  color: #303133;
  font-family: 'Courier New', monospace;
  margin: 20px 0;
}

.record-preview {
  width: 100%;
  max-width: 400px;
  margin-top: 20px;
}

.record-controls {
  display: flex;
  justify-content: center;
  gap: 12px;
}

.record-controls .el-button {
  min-width: 120px;
}

/* History area styles */
.history-section {
  padding: 20px 0;
}

.history-list {
  margin-top: 20px;
}

.empty-history {
  padding: 40px 0;
}

.message-content {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-list :deep(.el-table__row) {
  cursor: pointer;
}

.history-list :deep(.el-table__row:hover) {
  background-color: #f5f7fa;
}
</style>
