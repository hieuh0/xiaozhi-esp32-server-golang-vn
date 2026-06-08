<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal, RefreshCw } from '@lucide/vue'
import api from '../../utils/api'
import AgentForm from '../../components/common/AgentForm.vue'
import AgentRuntimeDiagnostics from '../../components/common/AgentRuntimeDiagnostics.vue'
import { agentToForm, createDefaultAgentForm } from '../../composables/useAgentFormOptions'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const { t } = useLocale()

const agents = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const showAddDialog = ref(false)
const editingAgent = ref(null)
const saving = ref(false)
const agentFormRef = ref(null)
const agentForm = ref(createDefaultAgentForm({ isAdmin: true }))
const showDiagnosticsDialog = ref(false)
const diagnosticAgent = ref(null)
const diagnosticPanel = ref('mcp')

const diagnosticsTitle = computed(() => {
  const name = diagnosticAgent.value?.name || t('agent_name_fallback', { id: diagnosticAgent.value?.id || '' })
  return diagnosticPanel.value === 'openclaw' ? `${name} - OpenClaw` : `${name} - MCP`
})

const loadAgents = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/agents', { params: { page: page.value, page_size: pageSize.value } })
    agents.value = response.data.data || []
    total.value = response.data.total || 0
  } catch {
    ElMessage.error(t('load_agent_list_failed'))
  } finally {
    loading.value = false
  }
}

const handleAgentAction = (command, row) => {
  switch (command) {
    case 'edit': editAgent(row); break
    case 'mcp': openDiagnostics(row, 'mcp'); break
    case 'openclaw': openDiagnostics(row, 'openclaw'); break
    case 'delete': deleteAgent(row); break
  }
}

const openAddDialog = () => {
  editingAgent.value = null
  agentForm.value = createDefaultAgentForm({ isAdmin: true })
  showAddDialog.value = true
}

const editAgent = (agent) => {
  editingAgent.value = agent
  agentForm.value = agentToForm(agent, { isAdmin: true })
  showAddDialog.value = true
}

const saveAgent = async () => {
  if (!agentFormRef.value) return
  const valid = await agentFormRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const payload = agentFormRef.value.buildPayload()
    if (editingAgent.value) {
      await api.put(`/admin/agents/${editingAgent.value.id}`, payload)
      ElMessage.success(t('agent_update_success'))
    } else {
      await api.post('/admin/agents', payload)
      ElMessage.success(t('agent_add_success'))
    }
    showAddDialog.value = false
    await loadAgents()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || (editingAgent.value ? t('agent_update_failed') : t('agent_add_failed')))
  } finally {
    saving.value = false
  }
}

const deleteAgent = async (agent) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_agent_msg', { name: agent.name }), t('confirm_delete'), {
      confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning'
    })
    await api.delete(`/admin/agents/${agent.id}`)
    ElMessage.success(t('agent_delete_success'))
    await loadAgents()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(error.response?.data?.error || t('agent_delete_failed'))
  }
}

const openDiagnostics = (agent, panel = 'mcp') => {
  diagnosticAgent.value = agent
  diagnosticPanel.value = panel
  showDiagnosticsDialog.value = true
}

const getVoiceText = (agent) => {
  const tts = agent.tts_config?.name || agent.tts_config?.provider || t('tts_not_set')
  return agent.voice ? `${tts} · ${agent.voice}` : tts
}

const getASRSpeedText = (speed) => ({ normal: t('normal'), patient: t('patience'), fast: t('fast') }[speed] || t('normal'))
const getMemoryModeText = (mode) => ({ none: t('no_memory'), short: t('short_memory'), long: t('long_memory') }[mode] || t('short_memory'))
const getSpeakerChatModeText = (mode) => ({ off: t('close'), identified_only: t('voiceprint_only') }[mode] || t('close'))

const badgeClass = (color) => {
  const map = {
    gray: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700',
    blue: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800',
    green: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800',
    yellow: 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800',
  }
  return map[color] || map.gray
}

const getASRSpeedBadge = (speed) => speed === 'fast' ? badgeClass('green') : speed === 'patient' ? badgeClass('yellow') : badgeClass('gray')
const getMemoryModeBadge = (mode) => mode === 'long' ? badgeClass('green') : mode === 'none' ? badgeClass('gray') : badgeClass('blue')
const getSpeakerChatModeBadge = (mode) => mode === 'identified_only' ? badgeClass('yellow') : badgeClass('gray')

const totalPages = () => Math.ceil(total.value / pageSize.value) || 1

onMounted(() => { loadAgents() })
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex items-center justify-end gap-2">
      <Button variant="outline" @click="loadAgents">
        <RefreshCw class="w-4 h-4 mr-1.5" />{{ t('refresh') }}
      </Button>
      <Button @click="openAddDialog">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('add_agent') }}
      </Button>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
      <div class="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead class="w-16">ID</TableHead>
              <TableHead>{{ t('name') }}</TableHead>
              <TableHead>{{ t('nickname') }}</TableHead>
              <TableHead class="w-32">{{ t('owner_user') }}</TableHead>
              <TableHead class="w-48">{{ t('role_description') }}</TableHead>
              <TableHead class="w-36">{{ t('language_model') }}</TableHead>
              <TableHead class="w-44">{{ t('tts_voice_col') }}</TableHead>
              <TableHead class="w-20 text-center">{{ t('knowledge_base_label') }}</TableHead>
              <TableHead class="w-20 text-center">{{ t('device') }}</TableHead>
              <TableHead class="w-24">{{ t('asr') }}</TableHead>
              <TableHead class="w-24">{{ t('memory_col') }}</TableHead>
              <TableHead class="w-32">{{ t('voiceprint_chat') }}</TableHead>
              <TableHead class="w-16 text-center">{{ t('actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="13" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableEmpty v-if="!agents.length" />
              <TableRow v-for="row in agents" :key="row.id">
                <TableCell class="text-xs font-mono text-[var(--color-text-secondary)]">{{ row.id }}</TableCell>
                <TableCell class="font-semibold">{{ row.name }}</TableCell>
                <TableCell class="text-[var(--color-text-secondary)]">{{ row.nickname || row.name }}</TableCell>
                <TableCell class="text-[var(--color-text-secondary)]">{{ row.username || t('user_id_fallback', { id: row.user_id }) }}</TableCell>
                <TableCell class="text-[var(--color-text-secondary)] text-xs truncate max-w-[190px]" :title="row.custom_prompt">{{ row.custom_prompt || t('not_set') }}</TableCell>
                <TableCell class="text-sm">{{ row.llm_config?.name || t('not_set') }}</TableCell>
                <TableCell class="text-sm truncate max-w-[170px]" :title="getVoiceText(row)">{{ getVoiceText(row) }}</TableCell>
                <TableCell class="text-center">{{ row.knowledge_base_ids?.length || 0 }}</TableCell>
                <TableCell class="text-center">{{ row.device_count || 0 }}</TableCell>
                <TableCell><span :class="getASRSpeedBadge(row.asr_speed)">{{ getASRSpeedText(row.asr_speed) }}</span></TableCell>
                <TableCell><span :class="getMemoryModeBadge(row.memory_mode)">{{ getMemoryModeText(row.memory_mode) }}</span></TableCell>
                <TableCell><span :class="getSpeakerChatModeBadge(row.speaker_chat_mode)">{{ getSpeakerChatModeText(row.speaker_chat_mode) }}</span></TableCell>
                <TableCell class="text-center">
                  <DropdownMenu>
                    <DropdownMenuTrigger as-child>
                      <Button variant="ghost" size="icon" class="h-8 w-8" :aria-label="t('more_actions')">
                        <MoreHorizontal class="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem @click="handleAgentAction('edit', row)">{{ t('edit') }}</DropdownMenuItem>
                      <DropdownMenuItem @click="handleAgentAction('mcp', row)">MCP</DropdownMenuItem>
                      <DropdownMenuItem @click="handleAgentAction('openclaw', row)">OpenClaw</DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem class="text-destructive" @click="handleAgentAction('delete', row)">{{ t('delete') }}</DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
      <span>{{ total }} {{ t('total_items') }}</span>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadAgents()">{{ t('prev') }}</Button>
        <span>{{ page }} / {{ totalPages() }}</span>
        <Button variant="outline" size="sm" :disabled="page >= totalPages()" @click="page++; loadAgents()">{{ t('next') }}</Button>
      </div>
    </div>

    <!-- Add/Edit agent dialog -->
    <Dialog v-model:open="showAddDialog">
      <DialogContent class="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ editingAgent ? t('edit_agent') : t('add_agent') }}</DialogTitle>
        </DialogHeader>
        <AgentForm ref="agentFormRef" v-model="agentForm" is-admin :mode="editingAgent ? 'edit' : 'create'" />
        <AgentRuntimeDiagnostics
          v-if="editingAgent"
          class="mt-4"
          :agent-id="editingAgent.id"
          scope="admin"
          preload-status
        />
        <DialogFooter>
          <Button variant="outline" @click="showAddDialog = false">{{ t('cancel') }}</Button>
          <Button :disabled="saving" @click="saveAgent">{{ editingAgent ? t('update') : t('add') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Diagnostics dialog -->
    <Dialog v-model:open="showDiagnosticsDialog">
      <DialogContent class="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ diagnosticsTitle }}</DialogTitle>
        </DialogHeader>
        <AgentRuntimeDiagnostics
          v-if="diagnosticAgent"
          :key="`${diagnosticAgent.id}-${diagnosticPanel}`"
          :agent-id="diagnosticAgent.id"
          scope="admin"
          :default-panels="[diagnosticPanel]"
        />
      </DialogContent>
    </Dialog>
  </div>
</template>
