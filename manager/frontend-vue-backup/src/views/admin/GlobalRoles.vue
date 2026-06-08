<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'

const { t } = useLocale()

const roles = ref([])
const loading = ref(false)
const saving = ref(false)
const showCreateDialog = ref(false)
const editingRole = ref(null)
const llmConfigs = ref([])
const ttsConfigs = ref([])
const availableVoices = ref([])
const voiceLoading = ref(false)
const previousTtsConfigId = ref(null)
const voiceFilter = ref('')

const form = reactive({
  name: '',
  description: '',
  prompt: '',
  llm_config_id: null,
  tts_config_id: null,
  voice: '',
  status: 'active',
  sort_order: 0,
  is_default: false
})

const isRoleActive = (role) => role?.status !== 'inactive'

const validate = () => {
  if (!String(form.name || '').trim()) { ElMessage.error(t('enter_role_name')); return false }
  if (!String(form.prompt || '').trim()) { ElMessage.error(t('enter_system_prompt')); return false }
  return true
}

const loadRoles = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/roles/global')
    roles.value = response.data.data || []
  } catch {
    ElMessage.error(t('load_roles_failed'))
  } finally {
    loading.value = false
  }
}

const loadConfigs = async () => {
  try {
    const [llmRes, ttsRes] = await Promise.all([
      api.get('/admin/llm-configs'),
      api.get('/admin/tts-configs')
    ])
    llmConfigs.value = llmRes.data.data || []
    ttsConfigs.value = ttsRes.data.data || []
  } catch (error) {
    console.error(t('load_config_list_failed'), error)
  }
}

const clearVoiceOptions = () => { availableVoices.value = [] }

const loadVoices = async (provider) => {
  if (!provider) { clearVoiceOptions(); return }
  voiceLoading.value = true
  try {
    const params = { provider }
    if (form.tts_config_id) params.config_id = form.tts_config_id
    const response = await api.get('/user/voice-options', { params })
    availableVoices.value = response.data.data || []
  } catch {
    clearVoiceOptions()
  } finally {
    voiceLoading.value = false
  }
}

const handleTtsConfigChange = async (newVal) => {
  let previousProvider = null
  if (previousTtsConfigId.value) {
    const prevConfig = ttsConfigs.value.find((c) => c.config_id === previousTtsConfigId.value)
    previousProvider = prevConfig?.provider || null
  }

  if (!newVal) {
    form.voice = ''
    previousTtsConfigId.value = null
    clearVoiceOptions()
    return
  }

  const ttsConfig = ttsConfigs.value.find((c) => c.config_id === newVal)
  if (!ttsConfig?.provider) {
    form.voice = ''
    previousTtsConfigId.value = newVal
    clearVoiceOptions()
    return
  }

  if (previousProvider && previousProvider !== ttsConfig.provider) form.voice = ''

  await loadVoices(ttsConfig.provider)

  if (form.voice && availableVoices.value.length > 0) {
    if (!availableVoices.value.some((v) => v.value === form.voice)) form.voice = ''
  }

  previousTtsConfigId.value = newVal
}

const filteredVoices = () => {
  if (!voiceFilter.value) return availableVoices.value
  const kw = voiceFilter.value.toLowerCase()
  return availableVoices.value.filter(v => v.label.toLowerCase().includes(kw) || v.value.toLowerCase().includes(kw))
}

const resetForm = () => {
  editingRole.value = null
  Object.assign(form, {
    name: '', description: '', prompt: '',
    llm_config_id: null, tts_config_id: null, voice: '',
    status: 'active', sort_order: 0, is_default: false
  })
  previousTtsConfigId.value = null
  clearVoiceOptions()
  voiceFilter.value = ''
}

const handleDialogClose = () => {
  showCreateDialog.value = false
  resetForm()
}

const editRole = (role) => {
  editingRole.value = role
  Object.assign(form, {
    name: role.name,
    description: role.description || '',
    prompt: role.prompt || '',
    llm_config_id: role.llm_config_id || null,
    tts_config_id: role.tts_config_id || null,
    voice: role.voice || '',
    status: role.status || 'active',
    sort_order: role.sort_order || 0,
    is_default: role.is_default || false
  })
  previousTtsConfigId.value = form.tts_config_id
  handleTtsConfigChange(form.tts_config_id)
  showCreateDialog.value = true
}

const duplicateRole = (role) => {
  editingRole.value = null
  Object.assign(form, {
    name: `${role.name} ${t('duplicate_suffix')}`,
    description: role.description || '',
    prompt: role.prompt || '',
    llm_config_id: role.llm_config_id || null,
    tts_config_id: role.tts_config_id || null,
    voice: role.voice || '',
    status: role.status || 'active',
    sort_order: role.sort_order || 0,
    is_default: false
  })
  previousTtsConfigId.value = form.tts_config_id
  handleTtsConfigChange(form.tts_config_id)
  showCreateDialog.value = true
}

const handleSave = async () => {
  if (!validate()) return
  saving.value = true
  try {
    const data = { ...form }
    if (editingRole.value) {
      await api.put(`/admin/roles/global/${editingRole.value.id}`, data)
      ElMessage.success(t('update_success'))
    } else {
      await api.post('/admin/roles/global', data)
      ElMessage.success(t('create_success'))
    }
    showCreateDialog.value = false
    loadRoles()
  } catch (error) {
    ElMessage.error(t('save_failed_colon') + (error.response?.data?.error || error.message))
  } finally {
    saving.value = false
  }
}

const toggleRoleStatus = async (role) => {
  if (!role?.id) return
  const action = isRoleActive(role) ? t('close') : t('enable')
  try {
    await api.patch(`/admin/roles/global/${role.id}/toggle`)
    ElMessage.success(t('role_action_success', { action }))
    await loadRoles()
  } catch (error) {
    ElMessage.error(t('status_toggle_failed_prefix') + (error.response?.data?.error || error.message))
  }
}

const setDefaultRole = async (role) => {
  if (!role?.id || role.is_default) return
  try {
    await api.patch(`/admin/roles/global/${role.id}/default`)
    ElMessage.success(t('set_as_default_role'))
    await loadRoles()
  } catch (error) {
    ElMessage.error(t('set_default_failed_prefix') + (error.response?.data?.error || error.message))
  }
}

const deleteRole = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_global_role'), t('hint'), {
      confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning'
    })
    await api.delete(`/admin/roles/global/${id}`)
    ElMessage.success(t('delete_success'))
    loadRoles()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('delete_failed'))
  }
}

const handleCardAction = (command, role) => {
  switch (command) {
    case 'edit': editRole(role); break
    case 'duplicate': duplicateRole(role); break
    case 'toggle-status': toggleRoleStatus(role); break
    case 'set-default': setDefaultRole(role); break
    case 'delete': deleteRole(role.id); break
  }
}

onMounted(() => { loadRoles(); loadConfigs() })
</script>

<template>
  <div class="px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex justify-end mb-5">
      <Button @click="showCreateDialog = true">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('create_global_role') }}
      </Button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>

    <!-- Empty -->
    <div v-else-if="!roles.length" class="flex flex-col items-center gap-3 py-16 text-[var(--color-text-secondary)]">
      <p class="text-sm">{{ t('no_global_role') }}</p>
      <Button @click="showCreateDialog = true">{{ t('create_first_global_role') }}</Button>
    </div>

    <!-- Card grid -->
    <div v-else class="grid gap-3" style="grid-template-columns: repeat(auto-fill, minmax(280px, 340px));">
      <div v-for="role in roles" :key="role.id"
        class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:-translate-y-0.5 transition-transform duration-200">
        <!-- Card header -->
        <div class="flex items-center justify-between gap-3 px-4 pt-4 pb-3 border-b border-[var(--color-line)]">
          <span class="font-bold text-[15px] text-[var(--color-text)] truncate">{{ role.name }}</span>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <button class="flex items-center justify-center w-7 h-7 rounded-full text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text)] transition-colors cursor-pointer">
                <MoreHorizontal class="w-4 h-4" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem @click="handleCardAction('edit', role)">{{ t('edit') }}</DropdownMenuItem>
              <DropdownMenuItem @click="handleCardAction('duplicate', role)">{{ t('copy') }}</DropdownMenuItem>
              <DropdownMenuItem @click="handleCardAction('toggle-status', role)">{{ isRoleActive(role) ? t('close') : t('enable') }}</DropdownMenuItem>
              <DropdownMenuItem :disabled="role.is_default" @click="!role.is_default && handleCardAction('set-default', role)">
                {{ role.is_default ? t('set_as_default_done') : t('set_as_default') }}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem class="text-destructive" @click="handleCardAction('delete', role)">{{ t('delete') }}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <!-- Card body -->
        <div class="px-4 py-3 flex flex-col gap-2.5 min-h-[170px]">
          <p class="text-sm text-[var(--color-text-secondary)] line-clamp-2 m-0">{{ role.description || t('no_description_alt') }}</p>

          <!-- Tags -->
          <div class="flex flex-wrap gap-1.5">
            <span :class="isRoleActive(role)
              ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
              : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'">
              {{ isRoleActive(role) ? t('enable') : t('close') }}
            </span>
            <span v-if="role.is_default" class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800">
              {{ t('default_role_tag') }}
            </span>
            <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800">
              LLM: {{ role.llm_config_id || t('default') }}
            </span>
            <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800">
              TTS: {{ role.tts_config_id || t('default') }}
            </span>
            <span v-if="role.voice" class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-orange-50 text-orange-700 border-orange-200 dark:bg-orange-900/20 dark:text-orange-300 dark:border-orange-800">
              {{ t('voice_tag_prefix', { voice: role.voice }) }}
            </span>
          </div>

          <!-- Prompt preview -->
          <div class="mt-auto border border-[var(--color-line)] bg-[var(--color-surface-muted)] rounded-lg px-2.5 py-2">
            <p class="m-0 mb-1 text-[11px] font-bold uppercase tracking-wide text-[var(--color-text-tertiary)]">Prompt</p>
            <p class="m-0 text-xs text-[var(--color-text)] line-clamp-3">{{ role.prompt || t('prompt_not_set') }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Dialog -->
    <Dialog v-model:open="showCreateDialog" @update:open="v => { if (!v) resetForm() }">
      <DialogContent class="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ editingRole ? t('edit_global_role') : t('create_global_role') }}</DialogTitle>
        </DialogHeader>

        <div class="grid gap-6 py-2">
          <!-- Basic info section -->
          <div class="grid gap-4">
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-0">{{ t('basic_info') }}</p>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('role_name') }}</label>
              <Input v-model="form.name" :placeholder="t('enter_role_name')" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('description') }}</label>
              <textarea v-model="form.description" rows="3" :placeholder="t('enter_role_description')"
                class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none placeholder:text-muted-foreground w-full resize-y" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('sort_order') }}</label>
              <NumberInput v-model="form.sort_order" :min="0" :step="1" :precision="0" />
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('sort_order_hint_num') }}</p>
            </div>
            <div class="flex items-center justify-between gap-4">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('default_role_tag') }}</label>
              <Switch :checked="form.is_default" @update:checked="v => form.is_default = v" />
            </div>
          </div>

          <div class="border-t border-[var(--color-line)]" />

          <!-- Prompt section -->
          <div class="grid gap-4">
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-0">{{ t('prompt_config_section') }}</p>
            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('system_prompt_label') }}</label>
              <textarea v-model="form.prompt" rows="6" :placeholder="t('enter_system_prompt')"
                class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none placeholder:text-muted-foreground w-full resize-y" />
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('assistant_name_placeholder_hint') }}</p>
            </div>
          </div>

          <div class="border-t border-[var(--color-line)]" />

          <!-- Model config section -->
          <div class="grid gap-4">
            <p class="text-xs font-bold tracking-widest uppercase text-[var(--color-text-tertiary)] mb-0">{{ t('model_config_section') }}</p>

            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('llm_config_label') }}</label>
              <Select :model-value="form.llm_config_id || ''" @update:model-value="v => form.llm_config_id = v || null">
                <SelectTrigger><SelectValue :placeholder="t('select_llm_config_opt')" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="">{{ t('default') }}</SelectItem>
                  <SelectItem v-for="c in llmConfigs" :key="c.id" :value="c.config_id" :disabled="!c.enabled">
                    {{ c.name }} ({{ c.config_id }}){{ c.is_default ? ` · ${t('default')}` : '' }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('leave_blank_use_default') }}</p>
            </div>

            <div class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('tts_config_label') }}</label>
              <Select :model-value="form.tts_config_id || ''" @update:model-value="v => { form.tts_config_id = v || null; handleTtsConfigChange(v || null) }">
                <SelectTrigger><SelectValue :placeholder="t('select_tts_config_opt')" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="">{{ t('default') }}</SelectItem>
                  <SelectItem v-for="c in ttsConfigs" :key="c.id" :value="c.config_id" :disabled="!c.enabled">
                    {{ c.name }} ({{ c.config_id }}){{ c.is_default ? ` · ${t('default')}` : '' }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('leave_blank_use_default') }}</p>
            </div>

            <div v-if="form.tts_config_id" class="grid gap-1.5">
              <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('voice_timbre') }}</label>
              <Input
                v-model="form.voice"
                list="voice-datalist"
                :placeholder="t('select_or_enter_voice_custom')"
                :disabled="voiceLoading"
              />
              <datalist id="voice-datalist">
                <option v-for="v in filteredVoices()" :key="v.value" :value="v.value">{{ v.label }}</option>
              </datalist>
              <p class="text-xs text-[var(--color-text-secondary)]">{{ t('voice_auto_load_hint') }}</p>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="handleDialogClose">{{ t('cancel') }}</Button>
          <Button :disabled="saving" @click="handleSave">{{ t('save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
