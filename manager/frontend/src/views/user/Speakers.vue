<template>
  <div class="speakers-page">
    <SpeakerGroupList
      v-model:filterAgentId="filterAgentId"
      v-model:searchKeyword="searchKeyword"
      :agents="agents"
      :filtered-groups="filteredGroups"
      :loading="loading"
      :format-date="formatDate"
      :truncate-text="truncateText"
      @filter-change="loadSpeakerGroups"
      @add-group="handleAddGroup(groupDialogRef)"
      @edit-group="(row) => handleEditGroup(row, groupDialogRef)"
      @delete-group="handleDeleteGroup"
      @view-samples="handleViewSamples"
      @verify-group="(row) => handleVerifyGroup(row, verifyDialogRef, verifyDialogRef)"
    />

    <SpeakerGroupDialog
      v-model="showGroupDialog"
      ref="groupDialogRef"
      :mode="groupDialogMode"
      :group-form="groupForm"
      :group-rules="groupRules"
      :agents="agents"
      :tts-configs="ttsConfigs"
      :current-voice-options="currentVoiceOptions"
      :clone-voice-presets="cloneVoicePresets"
      :clone-voices-loading="cloneVoicesLoading"
      :submitting="submitting"
      :current-tts-config-name="getCurrentTtsConfigName()"
      :current-tts-config-info="getCurrentTtsConfigInfo()"
      :is-clone-voice-selected="isCloneVoiceSelected"
      @submit="(ref) => handleSubmitGroup(ref)"
      @apply-clone-voice="applyCloneVoice"
      @tts-config-change="handleTtsConfigChange"
    />

    <SpeakerSampleDrawer
      v-model="showSampleDrawer"
      :current-group="currentGroup"
      :samples="samples"
      :format-date="formatDate"
      :format-file-size="formatFileSize"
      :truncate-id="truncateId"
      @close="handleCloseSampleDrawer"
      @add-sample="handleAddSample(uploadDialogRef, uploadDialogRef)"
      @verify-from-samples="handleVerifyFromSamples(verifyDialogRef, verifyDialogRef)"
      @play-sample="(s) => handlePlaySample(s, audioPlayer)"
      @download-sample="handleDownloadSample"
      @delete-sample="handleDeleteSample"
      @copy="copyToClipboard"
    />

    <SpeakerSampleUpload
      v-model="showUploadDialog"
      v-model:uploadMode="uploadMode"
      ref="uploadDialogRef"
      :agents="agents"
      :history-form="historyForm"
      :history-messages="historyMessages"
      :loading-history="loadingHistory"
      :upload-form="uploadForm"
      :upload-rules="uploadRules"
      :is-recording="isRecording"
      :recorded-blob="recordedBlob"
      :recorded-blob-url="recordedBlobUrl"
      :record-time="recordTime"
      :can-record="canRecord"
      :submitting="submitting"
      :has-audio-file="hasAudioFile"
      :format-date="formatDate"
      :format-file-size="formatFileSize"
      :format-record-time="formatRecordTime"
      :truncate-id="truncateId"
      :truncate-text="truncateText"
      @close="handleCloseUploadDialog(uploadDialogRef, uploadDialogRef)"
      @submit="handleSubmitSample(uploadDialogRef, uploadDialogRef)"
      @load-history="loadHistoryMessages"
      @select-history="handleSelectHistoryMessage"
      @preview-audio="(msg) => handlePreviewHistoryAudio(msg, audioPlayer)"
      @file-change="(f) => handleFileChange(f, uploadDialogRef, uploadDialogRef)"
      @file-remove="() => handleFileRemove(uploadDialogRef)"
      @start-recording="startRecording(uploadDialogRef)"
      @stop-recording="stopRecording"
    />

    <SpeakerVerifyDialog
      v-model="showVerifyDialog"
      v-model:verifyMode="verifyMode"
      ref="verifyDialogRef"
      :current-verify-group="currentVerifyGroup"
      :verify-form="verifyForm"
      :verify-rules="verifyRules"
      :verify-file-list="verifyFileList"
      :verify-result="verifyResult"
      :is-verify-recording="isVerifyRecording"
      :verify-recorded-blob="verifyRecordedBlob"
      :verify-recorded-blob-url="verifyRecordedBlobUrl"
      :verify-record-time="verifyRecordTime"
      :can-record="canRecord"
      :verifying="verifying"
      :has-verify-audio-file="hasVerifyAudioFile"
      :format-file-size="formatFileSize"
      :format-record-time="formatRecordTime"
      @close="handleCloseVerifyDialog(verifyDialogRef, verifyDialogRef)"
      @submit="handleSubmitVerify(verifyDialogRef)"
      @file-change="(f) => handleVerifyFileChange(f, verifyDialogRef, verifyDialogRef)"
      @file-remove="() => handleVerifyFileRemove(verifyDialogRef)"
      @start-recording="startVerifyRecording(verifyDialogRef)"
      @stop-recording="stopVerifyRecording"
    />

    <audio ref="audioPlayer" style="display: none;" />
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useSpeakers } from '../../composables/useSpeakers'
import SpeakerGroupList from '../../components/user/SpeakerGroupList.vue'
import SpeakerGroupDialog from '../../components/user/SpeakerGroupDialog.vue'
import SpeakerSampleDrawer from '../../components/user/SpeakerSampleDrawer.vue'
import SpeakerSampleUpload from '../../components/user/SpeakerSampleUpload.vue'
import SpeakerVerifyDialog from '../../components/user/SpeakerVerifyDialog.vue'

const speakers = useSpeakers()

// Destructure all state and methods from composable
const {
  loading, submitting, agents, samples, filterAgentId, searchKeyword,
  showGroupDialog, groupDialogMode, currentGroup, showSampleDrawer, showUploadDialog, uploadMode,
  showVerifyDialog, verifyMode, currentVerifyGroup, verifying, verifyResult,
  verifyForm, verifyFileList, verifyRules,
  isVerifyRecording, verifyRecordedBlob, verifyRecordedBlobUrl, verifyRecordTime,
  isRecording, recordedBlob, recordedBlobUrl, recordTime, canRecord,
  groupForm, groupRules,
  ttsConfigs, currentVoiceOptions, cloneVoicePresets, cloneVoicesLoading,
  uploadForm, uploadRules,
  loadingHistory, historyMessages, historyForm,
  hasAudioFile, hasVerifyAudioFile, filteredGroups,
  loadSpeakerGroups, loadHistoryMessages,
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
} = speakers

// Component refs (expose inner formRefs to composable handlers)
const groupDialogRef = ref()
const uploadDialogRef = ref()
const verifyDialogRef = ref()
const audioPlayer = ref()

onMounted(initialize)
onBeforeUnmount(cleanup)
</script>

<style scoped>
.speakers-page { padding: 0; }
</style>
