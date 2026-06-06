<template>
  <el-drawer
    v-model="visible"
    :title="t('sample_management')"
    :size="800"
    :before-close="() => emit('close')"
  >
    <div v-if="currentGroup" class="sample-drawer">
      <!-- Group info card -->
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
            <el-button type="success" @click="emit('verify-from-samples')">
              <el-icon><VideoPlay /></el-icon>
              {{ t('verify_voiceprint_btn') }}
            </el-button>
            <el-button type="primary" @click="emit('add-sample')">
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
              <el-button type="text" size="small" @click="emit('copy', row.uuid)" style="margin-left: 8px;">
                <el-icon><DocumentCopy /></el-icon>
              </el-button>
            </template>
          </el-table-column>
          <el-table-column prop="file_name" :label="t('filename_label')" min-width="150" />
          <el-table-column prop="file_size" :label="t('file_size_col')" width="100">
            <template #default="{ row }">{{ formatFileSize(row.file_size) }}</template>
          </el-table-column>
          <el-table-column prop="duration" :label="t('duration_col')" width="80">
            <template #default="{ row }">{{ row.duration ? row.duration + 's' : '-' }}</template>
          </el-table-column>
          <el-table-column prop="created_at" :label="t('created_at')" width="180">
            <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
          </el-table-column>
          <el-table-column :label="t('actions')" width="180" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" size="small" link @click="emit('play-sample', row)">
                <el-icon><VideoPlay /></el-icon>{{ t('play') }}
              </el-button>
              <el-button type="primary" size="small" link @click="emit('download-sample', row)">
                <el-icon><Download /></el-icon>{{ t('download_btn') }}
              </el-button>
              <el-button type="danger" size="small" link @click="emit('delete-sample', row)">
                <el-icon><Delete /></el-icon>{{ t('delete') }}
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
</template>

<script setup>
import { VideoPlay, Plus, DocumentCopy, Download, Delete } from '@element-plus/icons-vue'
import { useLocale } from '../../composables/useLocale'

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

<style scoped>
.sample-drawer { padding: 20px; }
.group-info-card { margin-bottom: 20px; }
.group-info h3 { margin: 0 0 15px 0; color: #303133; }
.prompt-section, .description-section { margin-top: 15px; padding-top: 15px; border-top: 1px solid #f0f0f0; }
.prompt-section strong, .description-section strong { display: block; margin-bottom: 8px; color: #606266; }
.prompt-section p, .description-section p { margin: 0; color: #303133; white-space: pre-wrap; word-break: break-word; }
.samples-section { margin-top: 20px; }
.samples-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
.samples-header h4 { margin: 0; color: #303133; }
.samples-header-actions { display: flex; gap: 8px; }
.empty-samples { padding: 40px 0; }
.uuid-text { font-family: monospace; font-size: 12px; }
</style>
