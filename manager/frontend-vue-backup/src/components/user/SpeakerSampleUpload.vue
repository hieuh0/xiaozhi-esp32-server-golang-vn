<script setup>
import { ref } from 'vue'
import { Upload, File, Play, Mic, CircleCheck, Pause, RotateCcw } from '@lucide/vue'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

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

const nativeFileInput = ref()
const isDragging = ref(false)

const emit = defineEmits([
  'close', 'submit', 'load-history', 'select-history', 'preview-audio',
  'file-change', 'file-remove', 'start-recording', 'stop-recording'
])

const triggerFileInput = () => nativeFileInput.value?.click()

const handleDrop = (e) => {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) emit('file-change', { raw: file, name: file.name, size: file.size, uid: Date.now() })
}

const onNativeFileChange = (e) => {
  const file = e.target?.files?.[0]
  if (file) emit('file-change', { raw: file, name: file.name, size: file.size, uid: Date.now() })
}

// Expose clearFiles() so composable can call uploadDialogRef.value.uploadRef.clearFiles()
const uploadRef = {
  clearFiles: () => {
    if (nativeFileInput.value) nativeFileInput.value.value = ''
  }
}

defineExpose({ uploadFormRef: null, uploadRef })
</script>

<template>
  <Dialog v-model:open="visible">
    <DialogContent class="max-w-[600px] max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ t('add_voiceprint_sample') }}</DialogTitle>
      </DialogHeader>

      <Tabs v-model="uploadMode" class="mt-2">
        <TabsList class="w-full">
          <TabsTrigger value="history" class="flex-1">{{ t('select_from_history') }}</TabsTrigger>
          <TabsTrigger value="upload" class="flex-1">{{ t('upload_file') }}</TabsTrigger>
          <TabsTrigger value="record" class="flex-1">{{ t('record_audio') }}</TabsTrigger>
        </TabsList>

        <!-- History tab -->
        <TabsContent value="history" class="mt-4">
          <div class="grid gap-3">
            <Select v-model="historyForm.agent_id" @update:model-value="emit('load-history')">
              <SelectTrigger class="w-full">
                <SelectValue :placeholder="t('select_agent')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="agent in agents" :key="agent.id" :value="String(agent.id)">{{ agent.name }}</SelectItem>
              </SelectContent>
            </Select>

            <div v-if="loadingHistory" class="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
            <div v-else-if="!historyMessages.length" class="py-8 text-center text-sm text-[var(--color-text-secondary)]">{{ t('no_chat_history_select') }}</div>
            <div v-else class="rounded-xl border border-[var(--color-line)] overflow-hidden max-h-[400px] overflow-y-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead class="w-16 text-center">{{ t('select_col') }}</TableHead>
                    <TableHead>{{ t('message_content_col') }}</TableHead>
                    <TableHead class="w-32">{{ t('device_id_col') }}</TableHead>
                    <TableHead class="w-36">{{ t('time_col') }}</TableHead>
                    <TableHead class="w-20 text-center">{{ t('actions') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow
                    v-for="row in historyMessages"
                    :key="row.message_id"
                    class="cursor-pointer"
                    @click="historyForm.selected_message_id = row.message_id; emit('select-history', row)"
                  >
                    <TableCell class="text-center">
                      <input
                        type="radio"
                        :checked="historyForm.selected_message_id === row.message_id"
                        @change="historyForm.selected_message_id = row.message_id"
                        class="accent-[var(--color-primary)]"
                      />
                    </TableCell>
                    <TableCell class="text-sm max-w-[200px] truncate" :title="row.content">{{ truncateText(row.content, 50) }}</TableCell>
                    <TableCell class="text-xs font-mono text-[var(--color-text-secondary)]" :title="row.device_id">{{ truncateId(row.device_id) }}</TableCell>
                    <TableCell class="text-xs text-[var(--color-text-secondary)]">{{ formatDate(row.created_at) }}</TableCell>
                    <TableCell class="text-center">
                      <Button variant="ghost" size="sm" @click.stop="emit('preview-audio', row)">
                        <Play class="w-3.5 h-3.5 mr-1" />{{ t('preview_btn') }}
                      </Button>
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          </div>
        </TabsContent>

        <!-- Upload tab -->
        <TabsContent value="upload" class="mt-4">
          <div class="grid gap-3">
            <!-- Drop zone -->
            <div
              class="border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors"
              :class="isDragging
                ? 'border-[var(--color-primary)] bg-[var(--color-surface-muted)]'
                : 'border-[var(--color-line)] hover:border-[var(--color-primary)]'"
              @click="triggerFileInput"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <Upload class="w-10 h-10 mx-auto mb-3 text-[var(--color-text-tertiary)]" />
              <p class="text-sm text-[var(--color-text-secondary)]">{{ t('drag_wav_hint') }}</p>
              <p class="text-xs text-[var(--color-text-tertiary)] mt-1">{{ t('wav_format_hint') }}</p>
            </div>
            <input ref="nativeFileInput" type="file" accept=".wav,audio/wav" class="hidden" @change="onNativeFileChange" />

            <!-- Selected file display -->
            <div v-if="uploadForm.audioFile" class="flex items-center gap-2 p-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-muted)] text-sm">
              <File class="w-4 h-4 text-[var(--color-text-tertiary)] shrink-0" />
              <span class="flex-1 truncate">{{ uploadForm.audioFile.name }}</span>
              <span class="text-xs text-[var(--color-text-tertiary)]">({{ formatFileSize(uploadForm.audioFile.size) }})</span>
              <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive h-6 px-2" @click="emit('file-remove')">✕</Button>
            </div>
          </div>
        </TabsContent>

        <!-- Record tab -->
        <TabsContent value="record" class="mt-4">
          <div class="grid gap-4">
            <!-- Record status -->
            <div class="min-h-[180px] flex flex-col items-center justify-center p-6 bg-[var(--color-surface-muted)] rounded-xl gap-3">
              <template v-if="!isRecording && !recordedBlob">
                <Mic class="w-12 h-12 text-[var(--color-primary)]" />
                <p class="text-sm text-[var(--color-text)]">{{ t('click_start_record') }}</p>
                <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('record_tip') }}</p>
              </template>
              <template v-else-if="isRecording">
                <div class="flex items-center gap-2">
                  <span class="w-3 h-3 rounded-full bg-red-500 animate-pulse" />
                  <span class="text-sm font-medium text-red-500">{{ t('recording_in_progress') }}</span>
                </div>
                <div class="text-3xl font-bold font-mono text-[var(--color-text)]">{{ formatRecordTime(recordTime) }}</div>
                <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('click_stop_record') }}</p>
              </template>
              <template v-else-if="recordedBlob">
                <CircleCheck class="w-12 h-12 text-green-500" />
                <p class="text-sm text-[var(--color-text)]">{{ t('recording_complete') }}</p>
                <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('record_duration', { time: formatRecordTime(recordTime) }) }}</p>
                <audio :src="recordedBlobUrl" controls class="w-full max-w-[380px]" />
              </template>
            </div>

            <!-- Record controls -->
            <div class="flex justify-center gap-3">
              <Button v-if="!isRecording && !recordedBlob" :disabled="!canRecord" @click="emit('start-recording')">
                <Play class="w-4 h-4 mr-1.5" />{{ t('start_recording') }}
              </Button>
              <Button v-if="isRecording" variant="destructive" @click="emit('stop-recording')">
                <Pause class="w-4 h-4 mr-1.5" />{{ t('stop_record') }}
              </Button>
              <Button v-if="recordedBlob" :disabled="!canRecord" @click="emit('start-recording')">
                <RotateCcw class="w-4 h-4 mr-1.5" />{{ t('re_record') }}
              </Button>
            </div>
          </div>
        </TabsContent>
      </Tabs>

      <DialogFooter class="mt-4">
        <Button variant="outline" @click="emit('close')">{{ t('cancel') }}</Button>
        <Button :disabled="submitting || !hasAudioFile" @click="emit('submit')">{{ t('confirm') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
