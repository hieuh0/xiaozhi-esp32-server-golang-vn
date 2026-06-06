<template>
  <div class="agent-config">
    <div class="agent-edit-header">
      <div class="header-left">
        <el-button text @click="goBack">
          <el-icon><ArrowLeft /></el-icon>
          {{ t('back') }}</el-button>
        <h2>{{ form.name || t('edit_agent') }}</h2>
      </div>
      <el-button type="primary" @click="handleSave" :loading="saving">{{ t('save_config') }}</el-button>
    </div>

    <div class="role-strip" v-loading="rolesLoading">
      <button
        v-for="role in allRoles"
        :key="role.id"
        type="button"
        class="role-chip"
        :class="{ active: selectedRoleId === role.id }"
        @click="applyRoleConfig(role)"
      >
        <span>{{ role.name }}</span>
        <small>{{ role.role_type === 'global' ? t('global') : t('mine') }}</small>
      </button>
      <span v-if="!rolesLoading && allRoles.length === 0" class="role-empty">{{ t('no_roles_available') }}</span>
    </div>

    <div class="form-card" v-loading="loadingAgent">
      <AgentForm ref="agentFormRef" v-model="form" mode="edit" />
    </div>

    <div class="diagnostics-card">
      <AgentRuntimeDiagnostics :agent-id="route.params.id" scope="user" preload-status />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import api from '@/utils/api'
import AgentForm from '../../components/common/AgentForm.vue'
import AgentRuntimeDiagnostics from '../../components/common/AgentRuntimeDiagnostics.vue'
import { agentToForm, createDefaultAgentForm } from '../../composables/useAgentFormOptions'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const route = useRoute()
const router = useRouter()

const form = ref(createDefaultAgentForm())
const agentFormRef = ref(null)
const saving = ref(false)
const applyingRoleConfig = ref(false)
const loadingAgent = ref(false)
const rolesLoading = ref(false)
const selectedRoleId = ref(null)
const globalRoles = ref([])
const userRoles = ref([])

const isRoleEnabled = (role) => role?.status === 'active' || !role?.status
const allRoles = computed(() => [...globalRoles.value, ...userRoles.value].filter(isRoleEnabled))

const loadAgent = async () => {
  loadingAgent.value = true
  try {
    const response = await api.get(`/user/agents/${route.params.id}`)
    form.value = agentToForm(response.data.data || {})
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('load_agent_config_failed'))
  } finally {
    loadingAgent.value = false
  }
}

const loadRoles = async () => {
  rolesLoading.value = true
  try {
    const response = await api.get('/user/roles')
    globalRoles.value = response.data.data?.global_roles || []
    userRoles.value = response.data.data?.user_roles || []
  } catch (error) {
    globalRoles.value = []
    userRoles.value = []
  } finally {
    rolesLoading.value = false
  }
}

const applyRoleConfig = async (role) => {
  if (!role) return
  applyingRoleConfig.value = true
  try {
    selectedRoleId.value = role.id
    await agentFormRef.value?.reloadOptions?.()
    form.value.custom_prompt = role.prompt || ''

    if (role.llm_config_id && agentFormRef.value?.hasLlmConfig?.(role.llm_config_id)) {
      form.value.llm_config_id = role.llm_config_id
    }

    if (role.tts_config_id && agentFormRef.value?.hasTtsConfig?.(role.tts_config_id)) {
      await agentFormRef.value?.setTtsConfig?.(role.tts_config_id, { clearInvalid: true })
    } else {
      await agentFormRef.value?.setTtsConfig?.(null, { clearInvalid: true })
    }

    form.value.voice = role.voice || null
    ElMessage.info(t('role_config_applied'))
  } finally {
    applyingRoleConfig.value = false
  }
}

const handleSave = async () => {
  if (applyingRoleConfig.value) {
    ElMessage.info(t('filling_role_config'))
    return
  }
  if (!agentFormRef.value) return
  const valid = await agentFormRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    await api.put(`/user/agents/${route.params.id}`, agentFormRef.value.buildPayload())
    ElMessage.success(t('save_success'))
    router.push('/agents')
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('save_failed'))
  } finally {
    saving.value = false
  }
}

const goBack = () => {
  router.push('/agents')
}

onMounted(async () => {
  await Promise.all([loadRoles(), loadAgent()])
})
</script>

<style scoped>
.agent-config {
  min-height: 100%;
  padding: 8px 0 24px;
}

.agent-edit-header {
  max-width: 1120px;
  margin: 0 auto 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.header-left {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-left h2 {
  margin: 0;
  color: var(--apple-text, #1d1d1f);
  font-size: 22px;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.role-strip {
  max-width: 1120px;
  min-height: 42px;
  margin: 0 auto 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  padding: 2px 0 6px;
}

.role-chip {
  border: 1px solid rgba(0, 122, 255, 0.18);
  border-radius: 8px;
  background: #fff;
  color: var(--apple-text, #1d1d1f);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: none;
  font-size: 13px;
  line-height: 1;
  padding: 9px 11px;
}

.role-chip small {
  color: var(--apple-text-secondary, #6b7280);
}

.role-chip.active {
  border-color: #409eff;
  background: #ecf5ff;
  color: #1677d2;
}

.role-empty {
  color: var(--apple-text-secondary, #6b7280);
  font-size: 13px;
}

.form-card,
.diagnostics-card {
  max-width: 1120px;
  margin: 0 auto;
  padding: 22px;
  border: 1px solid rgba(229, 229, 234, 0.78);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.035);
}

.diagnostics-card {
  margin-top: 14px;
}

@media (max-width: 760px) {
  .agent-edit-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .agent-edit-header .el-button {
    width: 100%;
  }

  .form-card,
  .diagnostics-card {
    padding: 14px;
  }
}
</style>
