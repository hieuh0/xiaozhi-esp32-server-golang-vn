<script setup>
import { ref } from 'vue'
import { Upload, Mic, CircleCheck, CircleX, Play, Pause, RotateCcw } from '@lucide/vue'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

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

const nativeVerifyInput = ref()
const isDragging = ref(false)

const emit = defineEmits(['close', 'submit', 'file-change', 'file-remove', 'start-recording', 'stop-recording'])

const triggerFileInput = () => nativeVerifyInput.value?.click()

const handleDrop = (e) => {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) emit('file-change', { raw: file, name: file.name, size: file.size, uid: Date.now() })
}

const onNativeFileChange = (e) => {
  const file = e.target?.files?.[0]
  if (file) emit('file-change', { raw: file, name: file.name, size: file.size, uid: Date.now() })
}

// Compat: composable may call verifyUploadRef.clearFiles()
const verifyUploadRef = { clearFiles: () => { if (nativeVerifyInput.value) nativeVerifyInput.value.value = '' } }
defineExpose({ verifyFormRef: null, verifyUploadRef })
</script>

<template>
  <Dialog v-model:open="visible">
    <DialogContent class="max-w-[560px] max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ t('verify_group_title', { name: currentVerifyGroup?.name || '' }) }}</DialogTitle>
      </DialogHeader>

      <Tabs v-model="verifyMode" class="mt-2">
        <TabsList class="w-full">
          <TabsTrigger value="upload" class="flex-1">{{ t('upload_file') }}</TabsTrigger>
          <TabsTrigger value="record" class="flex-1">{{ t('record_audio') }}</TabsTrigger>
        </TabsList>

        <!-- Upload tab -->
        <TabsContent value="upload" class="mt-4">
          <div
            class="border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors"
            :class="isDragging ? 'border-[var(--color-primary)] bg-[var(--color-surface-muted)]' : 'border-[var(--color-line)] hover:border-[var(--color-primary)]'"
            @click="triggerFileInput"
            @dragover.prevent="isDragging = true"
            @dragleave.prevent="isDragging = false"
            @drop.prevent="handleDrop"
          >
            <Upload class="w-10 h-10 mx-auto mb-3 text-[var(--color-text-tertiary)]" />
            <p class="text-sm text-[var(--color-text-secondary)]">{{ t('drag_wav_hint') }}</p>
            <p class="text-xs text-[var(--color-text-tertiary)] mt-1">{{ t('wav_format_hint') }}</p>
          </div>
          <input ref="nativeVerifyInput" type="file" accept=".wav,audio/wav" class="hidden" @change="onNativeFileChange" />

          <div v-if="verifyForm.audioFile" class="flex items-center gap-2 p-3 mt-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-muted)] text-sm">
            <span class="flex-1 truncate">{{ verifyForm.audioFile.name }}</span>
            <span class="text-xs text-[var(--color-text-tertiary)]">({{ formatFileSize(verifyForm.audioFile.size) }})</span>
            <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive h-6 px-2" @click="emit('file-remove')">✕</Button>
          </div>
        </TabsContent>

        <!-- Record tab -->
        <TabsContent value="record" class="mt-4">
          <div class="min-h-[180px] flex flex-col items-center justify-center p-6 bg-[var(--color-surface-muted)] rounded-xl gap-3">
            <template v-if="!isVerifyRecording && !verifyRecordedBlob">
              <Mic class="w-12 h-12 text-[var(--color-primary)]" />
              <p class="text-sm text-[var(--color-text)]">{{ t('click_start_record') }}</p>
              <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('record_tip') }}</p>
            </template>
            <template v-else-if="isVerifyRecording">
              <div class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-red-500 animate-pulse" />
                <span class="text-sm font-medium text-red-500">{{ t('recording_in_progress') }}</span>
              </div>
              <div class="text-3xl font-bold font-mono text-[var(--color-text)]">{{ formatRecordTime(verifyRecordTime) }}</div>
              <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('click_stop_record') }}</p>
            </template>
            <template v-else-if="verifyRecordedBlob">
              <CircleCheck class="w-12 h-12 text-green-500" />
              <p class="text-sm text-[var(--color-text)]">{{ t('recording_complete') }}</p>
              <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('record_duration', { time: formatRecordTime(verifyRecordTime) }) }}</p>
              <audio :src="verifyRecordedBlobUrl" controls class="w-full max-w-[360px]" />
            </template>
          </div>
          <div class="flex justify-center gap-3 mt-3">
            <Button v-if="!isVerifyRecording && !verifyRecordedBlob" :disabled="!canRecord" @click="emit('start-recording')">
              <Play class="w-4 h-4 mr-1.5" />{{ t('start_recording') }}
            </Button>
            <Button v-if="isVerifyRecording" variant="destructive" @click="emit('stop-recording')">
              <Pause class="w-4 h-4 mr-1.5" />{{ t('stop_record') }}
            </Button>
            <Button v-if="verifyRecordedBlob" :disabled="!canRecord" @click="emit('start-recording')">
              <RotateCcw class="w-4 h-4 mr-1.5" />{{ t('re_record') }}
            </Button>
          </div>
        </TabsContent>
      </Tabs>

      <!-- Verify result -->
      <div v-if="verifyResult" class="mt-4">
        <hr class="border-[var(--color-line)] mb-4" />
        <div :class="['flex items-center gap-4 p-4 rounded-xl border', verifyResult.verified ? 'bg-green-50 border-green-200 dark:bg-green-900/20 dark:border-green-800' : 'bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-800']">
          <CircleCheck v-if="verifyResult.verified" class="w-10 h-10 text-green-500 shrink-0" />
          <CircleX v-else class="w-10 h-10 text-red-500 shrink-0" />
          <div>
            <p class="font-semibold text-base mb-1">{{ verifyResult.verified ? t('verification_passed') : t('verify_not_passed') }}</p>
            <p class="text-sm text-[var(--color-text-secondary)]">{{ t('confidence_label', { pct: (verifyResult.confidence * 100).toFixed(1) }) }}</p>
            <p class="text-sm text-[var(--color-text-secondary)]">{{ t('threshold_label_pct', { pct: (verifyResult.threshold * 100).toFixed(1) }) }}</p>
            <p v-if="verifyResult.message" class="text-xs text-[var(--color-text-tertiary)] mt-1">{{ verifyResult.message }}</p>
          </div>
        </div>
      </div>

      <DialogFooter class="mt-4">
        <Button variant="outline" @click="emit('close')">{{ t('cancel') }}</Button>
        <Button :disabled="verifying || !hasVerifyAudioFile" @click="emit('submit')">{{ t('verify') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
