<script setup>
import { Copy, Play, Download, Trash2, Plus } from '@lucide/vue'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

const { t } = useLocale()

defineProps({
  currentGroup: { type: Object, default: null },
  samples: { type: Array, default: () => [] },
  formatDate: { type: Function, required: true },
  formatFileSize: { type: Function, required: true },
  truncateId: { type: Function, required: true }
})

const visible = defineModel({ default: false })
const emit = defineEmits(['close', 'add-sample', 'verify-from-samples', 'play-sample', 'download-sample', 'delete-sample', 'copy'])
</script>

<template>
  <Sheet v-model:open="visible">
    <SheetContent side="right" class="w-full sm:max-w-[800px] overflow-y-auto p-0">
      <SheetHeader class="px-6 py-4 border-b border-[var(--color-line)]">
        <SheetTitle>{{ t('sample_management') }}</SheetTitle>
      </SheetHeader>

      <div v-if="currentGroup" class="p-6 grid gap-5">
        <!-- Group info -->
        <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-4 grid gap-2">
          <h3 class="font-bold text-[var(--color-text)]">{{ currentGroup.name }}</h3>
          <div v-if="currentGroup.prompt" class="border-t border-[var(--color-line)] pt-3">
            <p class="text-xs font-bold text-[var(--color-text-tertiary)] uppercase mb-1">Prompt</p>
            <p class="text-sm text-[var(--color-text)] whitespace-pre-wrap break-words">{{ currentGroup.prompt }}</p>
          </div>
          <div v-if="currentGroup.description" class="border-t border-[var(--color-line)] pt-3">
            <p class="text-xs font-bold text-[var(--color-text-tertiary)] uppercase mb-1">{{ t('desc_colon') }}</p>
            <p class="text-sm text-[var(--color-text)] whitespace-pre-wrap break-words">{{ currentGroup.description }}</p>
          </div>
        </div>

        <!-- Sample list -->
        <div class="grid gap-3">
          <div class="flex items-center justify-between gap-2">
            <h4 class="font-semibold text-[var(--color-text)]">{{ t('sample_list') }}</h4>
            <div class="flex gap-2">
              <Button variant="outline" size="sm" @click="emit('verify-from-samples')">
                <Play class="w-3.5 h-3.5 mr-1" />{{ t('verify_voiceprint_btn') }}
              </Button>
              <Button size="sm" @click="emit('add-sample')">
                <Plus class="w-3.5 h-3.5 mr-1" />{{ t('upload_new_sample') }}
              </Button>
            </div>
          </div>

          <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
            <div class="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>UUID</TableHead>
                    <TableHead>{{ t('filename_label') }}</TableHead>
                    <TableHead class="w-24">{{ t('file_size_col') }}</TableHead>
                    <TableHead class="w-20">{{ t('duration_col') }}</TableHead>
                    <TableHead>{{ t('created_at') }}</TableHead>
                    <TableHead class="w-36 text-center">{{ t('actions') }}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableEmpty v-if="!samples.length" />
                  <TableRow v-for="row in samples" :key="row.uuid">
                    <TableCell>
                      <div class="flex items-center gap-1">
                        <span class="font-mono text-xs" :title="row.uuid">{{ truncateId(row.uuid) }}</span>
                        <Button variant="ghost" size="icon" class="h-6 w-6" @click="emit('copy', row.uuid)">
                          <Copy class="h-3 w-3" />
                        </Button>
                      </div>
                    </TableCell>
                    <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.file_name }}</TableCell>
                    <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ formatFileSize(row.file_size) }}</TableCell>
                    <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ row.duration ? row.duration + 's' : '-' }}</TableCell>
                    <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ formatDate(row.created_at) }}</TableCell>
                    <TableCell>
                      <div class="flex items-center justify-center gap-1">
                        <Button variant="ghost" size="icon" class="h-7 w-7" :title="t('play')" @click="emit('play-sample', row)">
                          <Play class="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" class="h-7 w-7" :title="t('download_btn')" @click="emit('download-sample', row)">
                          <Download class="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive hover:text-destructive" :title="t('delete')" @click="emit('delete-sample', row)">
                          <Trash2 class="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          </div>
        </div>
      </div>
    </SheetContent>
  </Sheet>
</template>
