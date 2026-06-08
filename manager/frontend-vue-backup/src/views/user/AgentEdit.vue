<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@lucide/vue'
import api from '@/utils/api'
import AgentForm from '../../components/common/AgentForm.vue'
import AgentRuntimeDiagnostics from '../../components/common/AgentRuntimeDiagnostics.vue'
import { agentToForm, createDefaultAgentForm } from '../../composables/useAgentFormOptions'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'

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
    const res = await api.get(`/user/agents/${route.params.id}`)
    form.value = agentToForm(res.data.data || {})
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('load_agent_config_failed'))
  } finally {
    loadingAgent.value = false
  }
}

const loadRoles = async () => {
  rolesLoading.value = true
  try {
    const res = await api.get('/user/roles')
    globalRoles.value = res.data.data?.global_roles || []
    userRoles.value = res.data.data?.user_roles || []
  } catch {
    globalRoles.value = []; userRoles.value = []
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
  if (applyingRoleConfig.value) { ElMessage.info(t('filling_role_config')); return }
  if (!agentFormRef.value) return
  const valid = await agentFormRef.value.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    await api.put(`/user/agents/${route.params.id}`, agentFormRef.value.buildPayload())
    ElMessage.success(t('save_success'))
    router.push('/agents')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('save_failed'))
  } finally {
    saving.value = false
  }
}

const goBack = () => router.push('/agents')

onMounted(async () => { await Promise.all([loadRoles(), loadAgent()]) })
</script>

<template>
  <div class="min-h-full py-2 pb-6">
    <!-- Header -->
    <div class="max-w-[1120px] mx-auto mb-3 flex items-center justify-between gap-4">
      <div class="flex items-center gap-2.5 min-w-0">
        <Button variant="ghost" size="sm" @click="goBack">
          <ArrowLeft class="w-4 h-4 mr-1" />{{ t('back') }}
        </Button>
        <h2 class="text-xl font-bold text-[var(--color-text)] truncate">{{ form.name || t('edit_agent') }}</h2>
      </div>
      <Button :disabled="saving" @click="handleSave">{{ t('save_config') }}</Button>
    </div>

    <!-- Role strip -->
    <div :class="['max-w-[1120px] mx-auto mb-3 min-h-[42px] flex items-center gap-2 overflow-x-auto pb-1', rolesLoading && 'opacity-60']">
      <button
        v-for="role in allRoles"
        :key="role.id"
        type="button"
        :class="['inline-flex items-center gap-2 px-2.5 py-2 rounded-lg text-xs border flex-none cursor-pointer transition-colors',
          selectedRoleId === role.id
            ? 'border-[var(--color-primary)] bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
            : 'border-[var(--color-line)] bg-[var(--color-surface)] text-[var(--color-text)] hover:border-[var(--color-primary)]']"
        @click="applyRoleConfig(role)"
      >
        <span>{{ role.name }}</span>
        <small class="text-[var(--color-text-secondary)]">{{ role.role_type === 'global' ? t('global') : t('mine') }}</small>
      </button>
      <span v-if="!rolesLoading && allRoles.length === 0" class="text-sm text-[var(--color-text-secondary)]">{{ t('no_roles_available') }}</span>
    </div>

    <!-- Form card -->
    <div :class="['max-w-[1120px] mx-auto p-5 border border-[var(--color-line)] rounded-xl bg-[var(--color-surface)]', loadingAgent && 'opacity-60']">
      <AgentForm ref="agentFormRef" v-model="form" mode="edit" />
    </div>

    <!-- Diagnostics card -->
    <div class="max-w-[1120px] mx-auto mt-3 p-5 border border-[var(--color-line)] rounded-xl bg-[var(--color-surface)]">
      <AgentRuntimeDiagnostics :agent-id="route.params.id" scope="user" preload-status />
    </div>
  </div>
</template>
