<script setup>
import { ref } from 'vue'
import DeviceForm from '../common/DeviceForm.vue'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

defineProps({
  agents: { type: Array, default: () => [] },
  deviceForm: { type: Object, required: true },
  fixedAgentId: { type: String, default: null },
  addingDevice: { type: Boolean, default: false },
  mcpLoading: { type: Boolean, default: false },
  toolsLoading: { type: Boolean, default: false },
  callingTool: { type: Boolean, default: false },
  mcpTools: { type: Array, default: () => [] },
  mcpCallResult: { type: String, default: '' },
  mcpCallForm: { type: Object, required: true },
  roleConfigLoading: { type: Boolean, default: false },
  currentDevice: { type: Object, default: () => ({}) },
  selectedRole: { type: Object, default: null },
  availableRoles: { type: Array, default: () => [] }
})

const showAddDeviceDialog = defineModel('showAddDeviceDialog', { default: false })
const showMcpDialog = defineModel('showMcpDialog', { default: false })
const showRoleConfigDialog = defineModel('showRoleConfigDialog', { default: false })
const selectedRoleId = defineModel('selectedRoleId', { default: null })

const deviceFormRef = ref()
defineExpose({ deviceFormRef })

const emit = defineEmits([
  'reset-form', 'add-device', 'refresh-mcp', 'mcp-tool-change', 'call-mcp-tool',
  'role-select', 'apply-role', 'close-role-config', 'update:deviceForm'
])
</script>

<template>
  <!-- Bind device dialog -->
  <Dialog v-model:open="showAddDeviceDialog" @update:open="(v) => !v && emit('reset-form')">
    <DialogContent class="max-w-[520px] max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ t('bind_device') }}</DialogTitle>
      </DialogHeader>
      <DeviceForm
        ref="deviceFormRef"
        :model-value="deviceForm"
        @update:modelValue="emit('update:deviceForm', $event)"
        mode="bind"
        :agents="agents"
        :fixed-agent-id="fixedAgentId"
      />
      <DialogFooter class="mt-4">
        <Button variant="outline" @click="showAddDeviceDialog = false">{{ t('cancel') }}</Button>
        <Button :disabled="addingDevice" @click="emit('add-device')">
          {{ addingDevice ? t('binding') : t('bind_device') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- MCP tools dialog -->
  <Dialog v-model:open="showMcpDialog">
    <DialogContent class="max-w-[720px] max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ t('device_mcp_tools') }}</DialogTitle>
      </DialogHeader>
      <div :class="['grid gap-4', mcpLoading && 'opacity-60 pointer-events-none']">
        <div class="flex items-center gap-2">
          <Button size="sm" variant="outline" :disabled="toolsLoading" @click="emit('refresh-mcp')">{{ t('refresh_tool_list') }}</Button>
        </div>
        <div v-if="!mcpTools.length" class="text-sm text-[var(--color-text-secondary)] py-2">{{ t('no_tool_data') }}</div>
        <div v-else class="flex flex-wrap gap-1.5">
          <span v-for="tool in mcpTools" :key="tool.name" class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-[var(--color-text)]">{{ tool.name }}</span>
        </div>
        <hr class="border-[var(--color-line)]" />
        <div class="grid gap-3">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('tool') }}</label>
            <Select v-model="mcpCallForm.tool_name" @update:model-value="(v) => emit('mcp-tool-change', v)">
              <SelectTrigger class="w-full"><SelectValue :placeholder="t('select_tool')" /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="tool in mcpTools" :key="tool.name" :value="tool.name">{{ tool.name }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('args_json_label') }}</label>
            <Textarea v-model="mcpCallForm.argumentsText" :placeholder="t('mcp_args_placeholder')" rows="6" class="font-mono text-sm" />
          </div>
          <Button :disabled="callingTool" @click="emit('call-mcp-tool')">{{ t('call_tool') }}</Button>
        </div>
        <hr class="border-[var(--color-line)]" />
        <pre class="min-h-[60px] p-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-xs font-mono whitespace-pre-wrap break-all">{{ mcpCallResult || t('no_call_results') }}</pre>
      </div>
    </DialogContent>
  </Dialog>

  <!-- Role config dialog -->
  <Dialog v-model:open="showRoleConfigDialog" @update:open="(v) => !v && emit('close-role-config')">
    <DialogContent class="max-w-[660px] max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ t('device_role_config_title') }}</DialogTitle>
      </DialogHeader>
      <div :class="['grid gap-4 py-2', roleConfigLoading && 'opacity-60 pointer-events-none']">
        <!-- Info banner -->
        <div class="flex gap-2 p-3 rounded-lg bg-blue-50 border border-blue-200 text-sm text-blue-700 dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-400">
          <span>{{ t('role_config_note') }}: {{ t('role_override_desc') }}</span>
        </div>

        <!-- Current role -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('current_role_label') }}</label>
          <div v-if="currentDevice.role_id" class="text-sm">
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-green-100 text-green-700 border border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800">{{ t('role_linked') }}</span>
            <p class="mt-1 text-[var(--color-text-secondary)]"><strong>{{ t('role_id_label') }}</strong> {{ currentDevice.role_id }}</p>
          </div>
          <span v-else class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-muted)] border border-[var(--color-line)] text-[var(--color-text-secondary)]">{{ t('role_not_linked') }}</span>
        </div>

        <!-- Role select -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('select_role_label') }}</label>
          <Select v-model="selectedRoleId" @update:model-value="(v) => emit('role-select', v)">
            <SelectTrigger class="w-full"><SelectValue :placeholder="t('select_role_opt_ph')" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="role in availableRoles" :key="role.id" :value="role.id">{{ role.name }}</SelectItem>
            </SelectContent>
          </Select>
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('role_override_help') }}</p>
        </div>

        <!-- Role preview -->
        <div v-if="selectedRole" class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-4 grid gap-2 text-sm">
          <p><strong class="text-[var(--color-text)]">{{ t('name_colon') }}</strong> {{ selectedRole.name }}</p>
          <p v-if="selectedRole.description"><strong class="text-[var(--color-text)]">{{ t('desc_colon') }}</strong> {{ selectedRole.description }}</p>
          <hr class="border-[var(--color-line)]" />
          <p class="font-semibold">Prompt:</p>
          <p class="text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface-muted)] p-3 rounded-lg whitespace-pre-wrap">{{ selectedRole.prompt.substring(0, 200) }}{{ selectedRole.prompt.length > 200 ? '...' : '' }}</p>
          <div class="flex gap-1.5 flex-wrap">
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-muted)] border border-[var(--color-line)]">LLM: {{ selectedRole.llm_config_id || t('default') }}</span>
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-muted)] border border-[var(--color-line)]">TTS: {{ selectedRole.tts_config_id || t('default') }}</span>
            <span v-if="selectedRole.voice" class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-[var(--color-surface-muted)] border border-[var(--color-line)]">{{ t('voice_colon') }}{{ selectedRole.voice }}</span>
          </div>
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="emit('close-role-config')">{{ t('cancel') }}</Button>
        <Button :disabled="roleConfigLoading || (!selectedRoleId && !currentDevice.role_id)" @click="emit('apply-role')">
          {{ selectedRoleId ? t('apply_role') : t('cancel_role') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
