<script setup>
import { ArrowLeft, Plus, Monitor, Settings, Trash2, User, MessageCircle, PenLine, Check, X } from '@lucide/vue'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

defineProps({
  agents: { type: Array, default: () => [] },
  filteredDevices: { type: Array, default: () => [] },
  emptyDescription: { type: String, default: '' },
  showBackButton: { type: Boolean, default: false },
  editingDeviceId: { type: [String, Number], default: null },
  renamingDeviceId: { type: [String, Number], default: null },
  deviceNameInputRef: { type: Object, default: null },
  getDeviceDisplayName: { type: Function, required: true },
  getDeviceIdentityText: { type: Function, required: true },
  getDeviceAgentName: { type: Function, required: true },
  isDeviceOnline: { type: Function, required: true },
  formatDate: { type: Function, required: true }
})

const filterAgentId = defineModel('filterAgentId', { default: '' })
const editingDeviceName = defineModel('editingDeviceName', { default: '' })

const emit = defineEmits([
  'go-back', 'agent-filter-change', 'add-device',
  'start-name-edit', 'cancel-name-edit', 'save-name',
  'device-role', 'device-mcp', 'voice-push', 'delete-device'
])
</script>

<template>
  <div>
    <!-- Filter bar -->
    <div class="flex items-center justify-between gap-3 mb-4 px-4 py-3 rounded-2xl bg-[var(--color-surface)] border border-[var(--color-line)]">
      <div class="flex items-center gap-3 flex-wrap min-w-0">
        <Button v-if="showBackButton" variant="ghost" size="sm" @click="emit('go-back')">
          <ArrowLeft class="w-4 h-4 mr-1" />{{ t('back') }}
        </Button>
        <Select v-model="filterAgentId" @update:model-value="(v) => emit('agent-filter-change', v)">
          <SelectTrigger class="w-[220px]">
            <SelectValue :placeholder="t('filter_by_agent')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">{{ t('all_devices') }}</SelectItem>
            <SelectItem v-for="agent in agents" :key="agent.id" :value="String(agent.id)">{{ agent.name }}</SelectItem>
          </SelectContent>
        </Select>
        <span class="text-sm font-semibold text-[var(--color-text-secondary)]">{{ t('device_count_n', { count: filteredDevices.length }) }}</span>
      </div>
      <Button @click="emit('add-device')">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('add_device') }}
      </Button>
    </div>

    <!-- Empty state -->
    <div v-if="filteredDevices.length === 0" class="mt-10 text-center py-16 px-6 bg-[var(--color-surface)] border border-[var(--color-line)] rounded-2xl">
      <Monitor class="w-16 h-16 mx-auto mb-4 text-[var(--color-text-tertiary)]" />
      <h3 class="text-lg font-semibold text-[var(--color-text)] mb-2">{{ t('no_devices') }}</h3>
      <p class="text-sm text-[var(--color-text-secondary)] mb-6">{{ emptyDescription }}</p>
      <Button @click="emit('add-device')">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('add_first_device') }}
      </Button>
    </div>

    <!-- Device grid -->
    <div v-else class="mt-5 grid gap-4" style="grid-template-columns: repeat(auto-fill, minmax(280px, 340px));">
      <div v-for="device in filteredDevices" :key="device.id" class="bg-[var(--color-surface)] rounded-2xl p-4 border border-[var(--color-line)] flex flex-col h-full transition-all hover:-translate-y-0.5 hover:border-[var(--color-primary)]/20">
        <!-- Header -->
        <div class="flex items-start gap-3 mb-4">
          <div class="w-10 h-10 rounded-xl bg-gradient-to-b from-[#2e90ff] to-[#007aff] flex items-center justify-center text-white shrink-0">
            <Monitor class="w-5 h-5" />
          </div>
          <div class="flex-1 min-w-0">
            <!-- Name row -->
            <div class="flex items-center gap-2 mb-1">
              <Input
                v-if="editingDeviceId === device.id"
                v-model="editingDeviceName"
                class="h-7 text-sm flex-1 min-w-0"
                maxlength="50"
                :placeholder="t('enter_device_nickname')"
                @keydown.enter.prevent="emit('save-name', device)"
                @keydown.esc.prevent="emit('cancel-name-edit')"
              />
              <button
                v-else
                type="button"
                class="flex-1 min-w-0 text-left font-bold text-[var(--color-text)] text-sm truncate hover:text-[var(--color-primary)] cursor-text"
                :title="t('click_edit_nickname', { name: getDeviceDisplayName(device) })"
                @click="emit('start-name-edit', device)"
              >{{ getDeviceDisplayName(device) }}</button>
              <div class="flex items-center shrink-0 gap-0.5">
                <template v-if="editingDeviceId === device.id">
                  <Button variant="ghost" size="icon" class="h-6 w-6" :disabled="renamingDeviceId === device.id" :title="t('save_nickname')" @click="emit('save-name', device)">
                    <Check class="h-3 w-3 text-[var(--color-primary)]" />
                  </Button>
                  <Button variant="ghost" size="icon" class="h-6 w-6" :title="t('cancel_edit')" @click="emit('cancel-name-edit')">
                    <X class="h-3 w-3" />
                  </Button>
                </template>
                <Button v-else variant="ghost" size="icon" class="h-6 w-6 opacity-20 hover:opacity-100 transition-opacity" :title="t('edit_nickname_title')" @click="emit('start-name-edit', device)">
                  <PenLine class="h-3 w-3" />
                </Button>
              </div>
            </div>
            <p class="text-[11px] text-[var(--color-text-tertiary)] font-mono truncate">{{ getDeviceIdentityText(device) }}</p>
          </div>
          <!-- Status -->
          <div class="flex items-center gap-1.5 shrink-0">
            <span :class="['w-2 h-2 rounded-full', isDeviceOnline(device.last_active_at) ? 'bg-green-500' : 'bg-red-400']" />
            <span class="text-xs text-[var(--color-text-secondary)]">{{ isDeviceOnline(device.last_active_at) ? t('online') : t('offline') }}</span>
          </div>
        </div>

        <!-- Meta rows -->
        <div class="flex-1 grid gap-2 mb-4 text-xs">
          <div class="flex justify-between">
            <span class="text-[var(--color-text-secondary)]">{{ t('link_agent') }}</span>
            <span class="font-medium text-[var(--color-text)]">{{ getDeviceAgentName(device) }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-[var(--color-text-secondary)]">{{ t('activation_status') }}</span>
            <span :class="['inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium', device.activated ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400']">
              {{ device.activated ? t('activated') : t('not_activated') }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-[var(--color-text-secondary)]">{{ t('last_active') }}</span>
            <span class="text-[var(--color-text)]">{{ formatDate(device.last_active_at) }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-[var(--color-text-secondary)]">{{ t('created_at') }}</span>
            <span class="text-[var(--color-text)]">{{ formatDate(device.created_at) }}</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="grid grid-cols-2 gap-2">
          <Button variant="outline" size="sm" class="text-blue-700 border-blue-200 bg-blue-50 hover:bg-blue-100 dark:text-blue-400 dark:border-blue-800 dark:bg-blue-900/20" @click="emit('device-role', device.id)">
            <User class="w-3.5 h-3.5 mr-1" />{{ t('role') }}
          </Button>
          <Button variant="outline" size="sm" @click="emit('device-mcp', device)">
            <Settings class="w-3.5 h-3.5 mr-1" />MCP
          </Button>
          <Button variant="outline" size="sm" class="text-teal-700 border-teal-200 bg-teal-50 hover:bg-teal-100 dark:text-teal-400 dark:border-teal-800 dark:bg-teal-900/20" @click="emit('voice-push', device)">
            <MessageCircle class="w-3.5 h-3.5 mr-1" />{{ t('voice_notify') }}
          </Button>
          <Button variant="outline" size="sm" class="text-destructive border-destructive/30 hover:bg-destructive/10" @click="emit('delete-device', device)">
            <Trash2 class="w-3.5 h-3.5 mr-1" />{{ t('delete') }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
