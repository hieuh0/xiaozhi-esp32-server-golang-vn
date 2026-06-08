<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const userRoles = ref([])
const loading = ref(false)
const saving = ref(false)
const showCreateDialog = ref(false)
const editingRole = ref(null)

const llmConfigs = ref([])
const ttsConfigs = ref([])
const availableVoices = ref([])
const filteredVoices = ref([])
const voiceLoading = ref(false)
const previousTtsConfigId = ref(null)

const form = reactive({
  name: '',
  description: '',
  prompt: '',
  llm_config_id: null,
  tts_config_id: null,
  voice: ''
})

const isRoleActive = (role) => role?.status !== 'inactive'

const badgeClass = (active) => active
  ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
  : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'

const loadRoles = async () => {
  loading.value = true
  try {
    const response = await api.get('/user/roles')
    userRoles.value = response.data.data?.user_roles || []
  } catch {
    ElMessage.error(t('load_roles_failed'))
  } finally {
    loading.value = false
  }
}

const loadConfigs = async () => {
  try {
    const [llmRes, ttsRes] = await Promise.all([api.get('/user/llm-configs'), api.get('/user/tts-configs')])
    llmConfigs.value = llmRes.data.data || []
    ttsConfigs.value = ttsRes.data.data || []
  } catch (error) {
    console.error(t('load_config_list_failed'), error)
  }
}

const handleCardAction = (command, role) => {
  switch (command) {
    case 'edit': editRole(role); break
    case 'duplicate': duplicateRole(role); break
    case 'toggle-status': toggleRoleStatus(role); break
    case 'delete': deleteRole(role.id); break
  }
}

const toggleRoleStatus = async (role) => {
  if (!role?.id) return
  const action = isRoleActive(role) ? t('close') : t('enable')
  try {
    await api.patch(`/user/roles/${role.id}/toggle`)
    ElMessage.success(t('role_action_success', { action }))
    await loadRoles()
  } catch (error) {
    ElMessage.error(t('status_toggle_failed_prefix') + (error.response?.data?.error || error.message))
  }
}

const clearVoiceOptions = () => { availableVoices.value = []; filteredVoices.value = [] }

const loadVoices = async (provider) => {
  if (!provider) { clearVoiceOptions(); return }
  voiceLoading.value = true
  try {
    const params = { provider }
    if (form.tts_config_id) params.config_id = form.tts_config_id
    const response = await api.get('/user/voice-options', { params })
    availableVoices.value = response.data.data || []
    filteredVoices.value = availableVoices.value
  } catch {
    clearVoiceOptions()
  } finally {
    voiceLoading.value = false
  }
}

const handleTtsConfigChange = async () => {
  let previousProvider = null
  if (previousTtsConfigId.value) {
    const prevConfig = ttsConfigs.value.find((c) => c.config_id === previousTtsConfigId.value)
    previousProvider = prevConfig?.provider || null
  }
  if (!form.tts_config_id) {
    form.voice = ''
    previousTtsConfigId.value = null
    clearVoiceOptions()
    return
  }
  const ttsConfig = ttsConfigs.value.find((c) => c.config_id === form.tts_config_id)
  if (!ttsConfig?.provider) {
    form.voice = ''
    previousTtsConfigId.value = form.tts_config_id
    clearVoiceOptions()
    return
  }
  if (previousProvider && previousProvider !== ttsConfig.provider) form.voice = ''
  await loadVoices(ttsConfig.provider)
  if (form.voice && availableVoices.value.length > 0) {
    if (!availableVoices.value.some((v) => v.value === form.voice)) form.voice = ''
  }
  previousTtsConfigId.value = form.tts_config_id
}

const editRole = (role) => {
  editingRole.value = role
  Object.assign(form, { name: role.name, description: role.description || '', prompt: role.prompt || '', llm_config_id: role.llm_config_id || null, tts_config_id: role.tts_config_id || null, voice: role.voice || '' })
  previousTtsConfigId.value = form.tts_config_id
  handleTtsConfigChange()
  showCreateDialog.value = true
}

const duplicateRole = (role) => {
  editingRole.value = null
  Object.assign(form, { name: `${role.name} ${t('duplicate_suffix')}`, description: role.description || '', prompt: role.prompt || '', llm_config_id: role.llm_config_id || null, tts_config_id: role.tts_config_id || null, voice: role.voice || '' })
  previousTtsConfigId.value = form.tts_config_id
  handleTtsConfigChange()
  showCreateDialog.value = true
}

const handleSave = async () => {
  if (!form.name.trim()) { ElMessage.error(t('enter_role_name')); return }
  if (!form.prompt.trim()) { ElMessage.error(t('enter_system_prompt')); return }
  saving.value = true
  try {
    if (editingRole.value) {
      await api.put(`/user/roles/${editingRole.value.id}`, { ...form })
      ElMessage.success(t('update_success'))
    } else {
      await api.post('/user/roles', { ...form })
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

const deleteRole = async (id) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_role'), t('hint'), { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' })
    await api.delete(`/user/roles/${id}`)
    ElMessage.success(t('delete_success'))
    loadRoles()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('delete_failed'))
  }
}

const handleDialogClose = () => {
  showCreateDialog.value = false
  editingRole.value = null
  Object.assign(form, { name: '', description: '', prompt: '', llm_config_id: null, tts_config_id: null, voice: '' })
  previousTtsConfigId.value = null
  clearVoiceOptions()
}

onMounted(() => { loadRoles(); loadConfigs() })
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex justify-end">
      <Button @click="showCreateDialog = true">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('create_role') }}
      </Button>
    </div>

    <!-- Card grid -->
    <div v-if="loading" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</div>
    <div v-else-if="!userRoles.length" class="py-16 text-center">
      <p class="text-[var(--color-text-secondary)] mb-3">{{ t('no_roles_create') }}</p>
      <Button @click="showCreateDialog = true">{{ t('create_first_role') }}</Button>
    </div>
    <div v-else class="grid gap-3" style="grid-template-columns: repeat(auto-fill, minmax(280px, 340px));">
      <div
        v-for="role in userRoles" :key="role.id"
        class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-4 flex flex-col gap-3 hover:-translate-y-0.5 hover:shadow-md transition-all"
      >
        <!-- Card header -->
        <div class="flex items-center justify-between gap-2">
          <span class="font-bold text-[15px] text-[var(--color-text)] truncate">{{ role.name }}</span>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="icon" class="h-7 w-7 shrink-0" :aria-label="t('more_actions')">
                <MoreHorizontal class="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem @click="handleCardAction('edit', role)">{{ t('edit') }}</DropdownMenuItem>
              <DropdownMenuItem @click="handleCardAction('duplicate', role)">{{ t('copy') }}</DropdownMenuItem>
              <DropdownMenuItem @click="handleCardAction('toggle-status', role)">
                {{ isRoleActive(role) ? t('close') : t('enable') }}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem class="text-destructive" @click="handleCardAction('delete', role)">{{ t('delete') }}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <!-- Description -->
        <p class="text-sm text-[var(--color-text-secondary)] line-clamp-2 min-h-[2.5rem]">{{ role.description || t('no_description_alt') }}</p>

        <!-- Tags -->
        <div class="flex flex-wrap gap-1.5">
          <span :class="badgeClass(isRoleActive(role))">{{ isRoleActive(role) ? t('enable') : t('close') }}</span>
          <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800">
            LLM: {{ role.llm_config_id || t('default') }}
          </span>
          <span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800">
            TTS: {{ role.tts_config_id || t('default') }}
          </span>
          <span v-if="role.voice" class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-900/20 dark:text-yellow-300 dark:border-yellow-800">
            {{ t('voice_tag_prefix', { voice: role.voice }) }}
          </span>
        </div>

        <!-- Prompt preview -->
        <div class="mt-auto border border-[var(--color-line)] bg-[var(--color-surface-muted)] rounded-lg px-3 py-2">
          <p class="text-[10px] font-bold uppercase text-[var(--color-text-tertiary)] mb-1">Prompt</p>
          <p class="text-xs text-[var(--color-text-secondary)] line-clamp-3">{{ role.prompt || t('prompt_not_set') }}</p>
        </div>
      </div>
    </div>

    <!-- Create/Edit dialog -->
    <Dialog v-model:open="showCreateDialog" @update:open="(v) => !v && handleDialogClose()">
      <DialogContent class="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{{ editingRole ? t('edit_role') : t('create_role') }}</DialogTitle>
        </DialogHeader>

        <div class="grid gap-5 py-2">
          <!-- Basic info -->
          <div class="grid gap-3">
            <h4 class="text-sm font-bold text-[var(--color-text)]">{{ t('basic_info') }}</h4>
            <div class="grid gap-1.5">
              <label class="text-sm font-medium text-[var(--color-text)]">{{ t('role_name') }} <span class="text-destructive">*</span></label>
              <Input v-model="form.name" :placeholder="t('enter_role_name')" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-medium text-[var(--color-text)]">{{ t('description') }}</label>
              <textarea
                v-model="form.description"
                rows="3"
                :placeholder="t('enter_role_description')"
                class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none resize-none"
              />
            </div>
          </div>

          <hr class="border-[var(--color-line)]" />

          <!-- Prompt config -->
          <div class="grid gap-3">
            <h4 class="text-sm font-bold text-[var(--color-text)]">{{ t('prompt_config_section') }}</h4>
            <div class="grid gap-1.5">
              <label class="text-sm font-medium text-[var(--color-text)]">{{ t('system_prompt_label') }} <span class="text-destructive">*</span></label>
              <textarea
                v-model="form.prompt"
                rows="6"
                :placeholder="t('enter_system_prompt')"
                class="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 rounded-md border bg-transparent px-2.5 py-2 text-sm shadow-xs transition-[color,box-shadow] focus-visible:ring-3 focus-visible:outline-none resize-none"
              />
              <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('assistant_name_placeholder_hint') }}</p>
            </div>
          </div>

          <hr class="border-[var(--color-line)]" />

          <!-- Model config -->
          <div class="grid gap-3">
            <h4 class="text-sm font-bold text-[var(--color-text)]">{{ t('model_config_section') }}</h4>

            <div class="grid gap-1.5">
              <label class="text-sm font-medium text-[var(--color-text)]">{{ t('llm_config_label') }}</label>
              <Select v-model="form.llm_config_id">
                <SelectTrigger class="w-full">
                  <SelectValue :placeholder="t('select_llm_config_opt')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem :value="null">{{ t('select_llm_config_opt') }}</SelectItem>
                  <SelectItem
                    v-for="config in llmConfigs"
                    :key="config.id"
                    :value="config.config_id"
                    :disabled="!config.enabled"
                  >
                    {{ config.name }} ({{ config.config_id }}){{ config.is_default ? ` [${t('default')}]` : '' }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('leave_blank_use_default') }}</p>
            </div>

            <div class="grid gap-1.5">
              <label class="text-sm font-medium text-[var(--color-text)]">{{ t('tts_config_label') }}</label>
              <Select v-model="form.tts_config_id" @update:model-value="handleTtsConfigChange">
                <SelectTrigger class="w-full">
                  <SelectValue :placeholder="t('select_tts_config_opt')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem :value="null">{{ t('select_tts_config_opt') }}</SelectItem>
                  <SelectItem
                    v-for="config in ttsConfigs"
                    :key="config.id"
                    :value="config.config_id"
                    :disabled="!config.enabled"
                  >
                    {{ config.name }} ({{ config.config_id }}){{ config.is_default ? ` [${t('default')}]` : '' }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('leave_blank_use_default') }}</p>
            </div>

            <div v-if="form.tts_config_id" class="grid gap-1.5">
              <label class="text-sm font-medium text-[var(--color-text)]">{{ t('voice_timbre') }}</label>
              <Input
                v-model="form.voice"
                list="voice-datalist"
                :placeholder="t('select_or_enter_voice_custom')"
                :disabled="voiceLoading"
              />
              <datalist id="voice-datalist">
                <option v-for="voice in filteredVoices" :key="voice.value" :value="voice.value">{{ voice.label }}</option>
              </datalist>
              <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('voice_auto_load_hint') }}</p>
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
