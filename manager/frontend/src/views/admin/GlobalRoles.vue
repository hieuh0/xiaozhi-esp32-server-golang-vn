<template>
  <div class="roles-page">
    <div class="page-actions">
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon>
        {{ t('create_global_role') }}</el-button>
    </div>

    <div class="roles-grid" v-loading="loading">
      <div v-for="role in roles" :key="role.id" class="role-col">
        <el-card class="role-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="role-name">{{ role.name }}</span>
              <el-dropdown @command="(cmd) => handleCardAction(cmd, role)">
                <el-icon class="more-icon"><MoreFilled /></el-icon>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="edit">
                      <el-icon><Edit /></el-icon>
                      {{ t('edit') }}</el-dropdown-item>
                    <el-dropdown-item command="duplicate">
                      <el-icon><CopyDocument /></el-icon>
                      {{ t('copy') }}</el-dropdown-item>
                    <el-dropdown-item command="toggle-status">
                      <el-icon><SwitchButton /></el-icon>
                      {{ isRoleActive(role) ? t('close') : t('enable') }}
                    </el-dropdown-item>
                    <el-dropdown-item command="set-default" :disabled="role.is_default">
                      <el-icon><Star /></el-icon>
                      {{ role.is_default ? t('set_as_default_done') : t('set_as_default') }}
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" divided>
                      <el-icon><Delete /></el-icon>
                      {{ t('delete') }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>

          <div class="role-content">
            <p class="description">{{ role.description || t('no_description_alt') }}</p>

            <div class="role-config">
              <el-tag size="small" :type="isRoleActive(role) ? 'success' : 'info'">
                {{ isRoleActive(role) ? t('enable') : t('close') }}
              </el-tag>
              <el-tag v-if="role.is_default" size="small" type="warning">{{ t('default_role_tag') }}</el-tag>
              <el-tag size="small" type="primary">LLM: {{ role.llm_config_id || t('default') }}</el-tag>
              <el-tag size="small" type="success">TTS: {{ role.tts_config_id || t('default') }}</el-tag>
              <el-tag v-if="role.voice" size="small" type="warning">{{ t('voice_tag_prefix', { voice: role.voice }) }}</el-tag>
            </div>

            <div class="role-prompt">
              <p class="prompt-label">Prompt</p>
              <p class="prompt-text">{{ role.prompt || t('prompt_not_set') }}</p>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <el-empty v-if="!loading && roles.length === 0" :description="t('no_global_role')">
      <el-button type="primary" @click="showCreateDialog = true">{{ t('create_first_global_role') }}</el-button>
    </el-empty>

    <el-dialog
      v-model="showCreateDialog"
      :title="editingRole ? t('edit_global_role') : t('create_global_role')"
      width="800px"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <div class="dialog-sections">
          <section class="dialog-section">
            <h4 class="dialog-section-title">{{ t('basic_info') }}</h4>
            <el-form-item :label="t('role_name')" prop="name">
              <el-input v-model="form.name" :placeholder="t('enter_role_name')" />
            </el-form-item>

            <el-form-item :label="t('description')" prop="description">
              <el-input
                v-model="form.description"
                type="textarea"
                :rows="3"
                :placeholder="t('enter_role_description')"
              />
            </el-form-item>

            <el-form-item :label="t('sort_order')">
              <el-input-number
                v-model="form.sort_order"
                :min="0"
                :step="1"
                style="width: 100%"
                :placeholder="t('sort_order_hint_num')"
              />
            </el-form-item>

            <el-form-item :label="t('default_role_tag')">
              <el-switch v-model="form.is_default" />
            </el-form-item>
          </section>

          <el-divider />

          <section class="dialog-section">
            <h4 class="dialog-section-title">{{ t('prompt_config_section') }}</h4>
            <el-form-item :label="t('system_prompt_label')" prop="prompt">
              <el-input
                v-model="form.prompt"
                type="textarea"
                :rows="6"
                :placeholder="t('enter_system_prompt')"
              />
              <div class="prompt-tips">
                <el-text size="small" type="info">
                  {{ t('assistant_name_placeholder_hint') }}
                </el-text>
              </div>
            </el-form-item>
          </section>

          <el-divider />

          <section class="dialog-section">
            <h4 class="dialog-section-title">{{ t('model_config_section') }}</h4>
            <el-form-item :label="t('llm_config_label')">
              <el-select v-model="form.llm_config_id" :placeholder="t('select_llm_config_opt')" clearable style="width: 100%">
                <el-option
                  v-for="config in llmConfigs"
                  :key="config.id"
                  :label="`${config.name} (${config.config_id})`"
                  :value="config.config_id"
                  :disabled="!config.enabled"
                >
                  <span>{{ config.name }}</span>
                  <el-tag v-if="config.is_default" size="small" type="success" style="margin-left: 8px">{{ t('default') }}</el-tag>
                </el-option>
              </el-select>
              <div class="form-tip">
                <el-text size="small" type="info">{{ t('leave_blank_use_default') }}</el-text>
              </div>
            </el-form-item>

            <el-form-item :label="t('tts_config_label')">
              <el-select
                v-model="form.tts_config_id"
                :placeholder="t('select_tts_config_opt')"
                clearable
                style="width: 100%"
                @change="handleTtsConfigChange"
              >
                <el-option
                  v-for="config in ttsConfigs"
                  :key="config.id"
                  :label="`${config.name} (${config.config_id})`"
                  :value="config.config_id"
                  :disabled="!config.enabled"
                >
                  <span>{{ config.name }}</span>
                  <el-tag v-if="config.is_default" size="small" type="success" style="margin-left: 8px">{{ t('default') }}</el-tag>
                </el-option>
              </el-select>
              <div class="form-tip">
                <el-text size="small" type="info">{{ t('leave_blank_use_default') }}</el-text>
              </div>
            </el-form-item>

            <el-form-item :label="t('voice_timbre')" v-if="form.tts_config_id">
              <el-select
                v-model="form.voice"
                :placeholder="t('select_or_enter_voice_custom')"
                clearable
                filterable
                allow-create
                default-first-option
                reserve-keyword
                :loading="voiceLoading"
                :filter-method="filterVoice"
                style="width: 100%"
              >
                <el-option
                  v-for="voice in filteredVoices"
                  :key="voice.value"
                  :label="voice.label"
                  :value="voice.value"
                >
                  <span>{{ voice.label }}</span>
                  <span class="apple-option-value">{{ voice.value }}</span>
                </el-option>
              </el-select>
              <div class="form-tip">
                <el-text size="small" type="info">{{ t('voice_auto_load_hint') }}</el-text>
              </div>
            </el-form-item>
          </section>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="handleDialogClose">{{ t('cancel') }}</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          {{ t('save') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreFilled, Edit, CopyDocument, Delete, SwitchButton, Star } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const roles = ref([])
const loading = ref(false)
const saving = ref(false)
const showCreateDialog = ref(false)
const editingRole = ref(null)
const formRef = ref()

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
  voice: '',
  status: 'active',
  sort_order: 0,
  is_default: false
})

const rules = {
  name: [{ required: true, message: t('enter_role_name'), trigger: 'blur' }],
  prompt: [{ required: true, message: t('enter_system_prompt'), trigger: 'blur' }]
}

const isRoleActive = (role) => role?.status !== 'inactive'

const loadRoles = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/roles/global')
    roles.value = response.data.data || []
  } catch (error) {
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

const handleCardAction = (command, role) => {
  switch (command) {
    case 'edit':
      editRole(role)
      break
    case 'duplicate':
      duplicateRole(role)
      break
    case 'toggle-status':
      toggleRoleStatus(role)
      break
    case 'set-default':
      setDefaultRole(role)
      break
    case 'delete':
      deleteRole(role.id)
      break
  }
}

const clearVoiceOptions = () => {
  availableVoices.value = []
  filteredVoices.value = []
}

const filterVoice = (val) => {
  if (!val) {
    filteredVoices.value = availableVoices.value
    return
  }

  const keyword = val.toLowerCase()
  filteredVoices.value = availableVoices.value.filter((voice) =>
    voice.label.toLowerCase().includes(keyword) || voice.value.toLowerCase().includes(keyword)
  )
}

const loadVoices = async (provider) => {
  if (!provider) {
    clearVoiceOptions()
    return
  }

  voiceLoading.value = true
  try {
    const params = { provider }
    if (form.tts_config_id) {
      params.config_id = form.tts_config_id
    }
    const response = await api.get('/user/voice-options', { params })
    availableVoices.value = response.data.data || []
    filteredVoices.value = availableVoices.value
  } catch (error) {
    clearVoiceOptions()
    console.error(t('load_voice_list_failed_nc'), error)
  } finally {
    voiceLoading.value = false
  }
}

const handleTtsConfigChange = async () => {
  let previousProvider = null
  if (previousTtsConfigId.value) {
    const prevConfig = ttsConfigs.value.find((config) => config.config_id === previousTtsConfigId.value)
    previousProvider = prevConfig?.provider || null
  }

  if (!form.tts_config_id) {
    form.voice = ''
    previousTtsConfigId.value = null
    clearVoiceOptions()
    return
  }

  const ttsConfig = ttsConfigs.value.find((config) => config.config_id === form.tts_config_id)
  if (!ttsConfig || !ttsConfig.provider) {
    form.voice = ''
    previousTtsConfigId.value = form.tts_config_id
    clearVoiceOptions()
    return
  }

  if (previousProvider && previousProvider !== ttsConfig.provider) {
    form.voice = ''
  }

  await loadVoices(ttsConfig.provider)

  if (form.voice && availableVoices.value.length > 0) {
    const voiceExists = availableVoices.value.some((voice) => voice.value === form.voice)
    if (!voiceExists) {
      form.voice = ''
    }
  }

  previousTtsConfigId.value = form.tts_config_id
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
  handleTtsConfigChange()
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
  handleTtsConfigChange()
  showCreateDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
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
  })
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
      confirmButtonText: t('confirm'),
      cancelButtonText: t('cancel'),
      type: 'warning'
    })

    await api.delete(`/admin/roles/global/${id}`)
    ElMessage.success(t('delete_success'))
    loadRoles()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('delete_failed'))
    }
  }
}

const resetForm = () => {
  editingRole.value = null
  Object.assign(form, {
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
  previousTtsConfigId.value = null
  clearVoiceOptions()
}

const handleDialogClose = () => {
  showCreateDialog.value = false
  resetForm()
  if (formRef.value) {
    formRef.value.resetFields()
  }
}

onMounted(() => {
  loadRoles()
  loadConfigs()
})
</script>

<style scoped>
.roles-page {
  padding: 20px;
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.roles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 340px));
  gap: 12px;
  align-items: stretch;
  justify-content: flex-start;
}

.role-col {
  display: flex;
  min-width: 0;
}

.role-card {
  width: 100%;
  max-width: 340px;
  border-radius: 12px;
  border: 1px solid #ebeef5;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.role-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.08);
}

:deep(.role-card .el-card__header) {
  padding: 12px 14px;
}

:deep(.role-card .el-card__body) {
  padding: 12px 14px 14px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.role-name {
  font-weight: 700;
  font-size: 15px;
  color: #111827;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.more-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 16px;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  color: #6b7280;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.more-icon:hover {
  color: #1f2937;
  background: #f3f4f6;
}

.role-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 170px;
}

.description {
  color: #4b5563;
  font-size: 14px;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.role-config {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.role-prompt {
  margin-top: auto;
  border: 1px solid #e5e7eb;
  background: #f9fafb;
  border-radius: 8px;
  padding: 8px 10px;
}

.prompt-label {
  margin: 0 0 4px 0;
  color: #6b7280;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.prompt-text {
  margin: 0;
  color: #374151;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
}

.prompt-tips {
  margin-top: 4px;
}

.form-tip {
  margin-top: 4px;
}

.dialog-sections {
  display: flex;
  flex-direction: column;
}

.dialog-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dialog-section-title {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 700;
  color: #374151;
}

:deep(.dialog-sections .el-divider--horizontal) {
  margin: 8px 0 16px;
}
</style>
