<template>
  <div>
    <!-- Filter bar -->
    <div class="filter-bar">
      <el-select
        v-model="filterAgentId"
        :placeholder="t('filter_by_agent')"
        clearable
        style="width: 200px; margin-right: 10px;"
        @change="emit('filter-change')"
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
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button class="create-group-button" type="primary" @click="emit('add-group')">
        <el-icon><Plus /></el-icon>
        {{ t('create_voiceprint_group') }}
      </el-button>
    </div>

    <!-- Voiceprint group table -->
    <div v-loading="loading" class="speakers-content">
      <el-table :data="filteredGroups" stripe style="width: 100%">
        <el-table-column prop="name" :label="t('voiceprint_group_name')" min-width="150" />
        <el-table-column prop="agent_name" :label="t('link_agent')" min-width="120" />
        <el-table-column label="Prompt" min-width="200">
          <template #default="{ row }">
            <el-popover placement="top" :width="300" trigger="hover" v-if="row.prompt">
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
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('actions')" width="360" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button type="success" size="small" @click="emit('verify-group', row)">
                <el-icon><VideoPlay /></el-icon>{{ t('verify') }}
              </el-button>
              <el-button type="primary" size="small" @click="emit('view-samples', row)">
                <el-icon><View /></el-icon>{{ t('manage_voiceprints') }}
              </el-button>
              <el-button type="primary" size="small" plain @click="emit('edit-group', row)">
                <el-icon><Edit /></el-icon>{{ t('edit') }}
              </el-button>
              <el-button type="danger" size="small" @click="emit('delete-group', row)">
                <el-icon><Delete /></el-icon>{{ t('delete') }}
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="filteredGroups.length === 0 && !loading" class="empty-state">
        <el-empty :description="t('no_voiceprint_groups')" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { Search, Plus, VideoPlay, View, Edit, Delete } from '@element-plus/icons-vue'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

const props = defineProps({
  agents: { type: Array, default: () => [] },
  filteredGroups: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  formatDate: { type: Function, required: true },
  truncateText: { type: Function, required: true }
})

const filterAgentId = defineModel('filterAgentId', { default: '' })
const searchKeyword = defineModel('searchKeyword', { default: '' })

const emit = defineEmits(['filter-change', 'add-group', 'edit-group', 'delete-group', 'view-samples', 'verify-group'])
</script>

<style scoped>
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
.create-group-button { margin-left: auto; }
.speakers-content { background: rgba(255, 255, 255, 0.88); border-radius: 8px; padding: 20px; }
.prompt-text { color: #606266; cursor: pointer; }
.prompt-popover { max-height: 200px; overflow-y: auto; white-space: pre-wrap; word-break: break-word; }
.text-muted { color: #909399; }
.empty-state { padding: 40px 0; }
.action-buttons { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.action-buttons .el-button { margin: 0; white-space: nowrap; }
</style>
