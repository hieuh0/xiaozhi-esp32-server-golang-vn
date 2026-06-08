<script setup>
import { Search, Plus, Play, Eye, Edit, Trash2 } from '@lucide/vue'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

const { t } = useLocale()

defineProps({
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

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Filter bar -->
    <div class="flex items-center gap-2 flex-wrap">
      <Select v-model="filterAgentId" @update:model-value="emit('filter-change')">
        <SelectTrigger class="w-[200px]">
          <SelectValue :placeholder="t('filter_by_agent')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="">{{ t('all_agents') }}</SelectItem>
          <SelectItem v-for="agent in agents" :key="agent.id" :value="String(agent.id)">{{ agent.name }}</SelectItem>
        </SelectContent>
      </Select>
      <div class="relative">
        <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-tertiary)]" />
        <Input v-model="searchKeyword" :placeholder="t('search_voiceprint_group')" class="pl-8 w-[250px]" />
      </div>
      <Button class="ml-auto" @click="emit('add-group')">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('create_voiceprint_group') }}
      </Button>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
      <div class="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('voiceprint_group_name') }}</TableHead>
              <TableHead>{{ t('link_agent') }}</TableHead>
              <TableHead>Prompt</TableHead>
              <TableHead class="w-24 text-center">{{ t('sample_count') }}</TableHead>
              <TableHead>{{ t('created_at') }}</TableHead>
              <TableHead class="text-center">{{ t('actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="6" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableEmpty v-if="!filteredGroups.length" />
              <TableRow v-for="row in filteredGroups" :key="row.id">
                <TableCell class="font-semibold">{{ row.name }}</TableCell>
                <TableCell class="text-[var(--color-text-secondary)]">{{ row.agent_name }}</TableCell>
                <TableCell class="text-[var(--color-text-secondary)] text-sm">
                  <span v-if="row.prompt" :title="row.prompt" class="cursor-help">{{ truncateText(row.prompt, 30) }}</span>
                  <span v-else class="text-[var(--color-text-tertiary)]">-</span>
                </TableCell>
                <TableCell class="text-center">
                  <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700">{{ row.sample_count }}</span>
                </TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ formatDate(row.created_at) }}</TableCell>
                <TableCell>
                  <div class="flex items-center justify-center gap-1 flex-wrap">
                    <Button variant="outline" size="sm" @click="emit('verify-group', row)">
                      <Play class="w-3.5 h-3.5 mr-1" />{{ t('verify') }}
                    </Button>
                    <Button variant="outline" size="sm" @click="emit('view-samples', row)">
                      <Eye class="w-3.5 h-3.5 mr-1" />{{ t('manage_voiceprints') }}
                    </Button>
                    <Button variant="outline" size="sm" @click="emit('edit-group', row)">
                      <Edit class="w-3.5 h-3.5 mr-1" />{{ t('edit') }}
                    </Button>
                    <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" @click="emit('delete-group', row)">
                      <Trash2 class="w-3.5 h-3.5 mr-1" />{{ t('delete') }}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>
  </div>
</template>
