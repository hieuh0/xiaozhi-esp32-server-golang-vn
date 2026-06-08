<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useLocale } from '../../composables/useLocale'
import { buildAgentPayload, useAgentFormOptions } from '../../composables/useAgentFormOptions'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const props = defineProps({
  modelValue: { type: Object, required: true },
  isAdmin: { type: Boolean, default: false },
  mode: { type: String, default: 'create' },
  labelPosition: { type: String, default: 'top' },
  labelWidth: { type: String, default: '120px' }
})

const emit = defineEmits(['update:modelValue'])

const form = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const targetUserId = computed(() => props.isAdmin ? Number(form.value.user_id || 0) : 0)

const {
  users, llmConfigs, ttsConfigs, knowledgeBases, mcpServiceOptions,
  voiceOptions, cloneVoices, loading,
  loadUsers, loadConfigs, loadKnowledgeBases, loadMcpServiceOptions,
  loadVoiceOptions, loadCloneVoices
} = useAgentFormOptions({ isAdmin: computed(() => props.isAdmin), targetUserId })

// ── Voice handling ──────────────────────────────────────────────────────────
const MAX_VISIBLE_VOICE_OPTIONS = 300
const voiceSearchKeyword = ref('')
const filteredVoiceOptions = ref([])
const previousTtsConfigId = ref(null)
const suppressTtsConfigWatch = ref(false)

const selectedTtsConfig = computed(() => ttsConfigs.value.find((c) => c.config_id === form.value.tts_config_id) || null)

const limitVoiceOptions = (voices) => {
  const list = Array.isArray(voices) ? voices : []
  const visible = list.slice(0, MAX_VISIBLE_VOICE_OPTIONS)
  const selected = String(form.value.voice || '').trim()
  if (selected && !visible.some((v) => v.value === selected)) {
    const opt = list.find((v) => v.value === selected)
    if (opt) visible.unshift(opt)
  }
  return visible
}

const syncFilteredVoiceOptions = () => {
  const keyword = String(voiceSearchKeyword.value || '').trim().toLowerCase()
  if (!keyword) { filteredVoiceOptions.value = limitVoiceOptions(voiceOptions.value); return }
  const matched = voiceOptions.value.filter((v) => {
    return String(v.label || '').toLowerCase().includes(keyword) || String(v.value || '').toLowerCase().includes(keyword)
  })
  filteredVoiceOptions.value = limitVoiceOptions(matched)
}

const refreshVoiceOptions = async ({ clearInvalid = true, previousConfigId = previousTtsConfigId.value } = {}) => {
  const provider = selectedTtsConfig.value?.provider
  if (!form.value.tts_config_id || !provider) {
    voiceOptions.value = []; filteredVoiceOptions.value = []; form.value.voice = null; previousTtsConfigId.value = null; return
  }
  const previousConfig = ttsConfigs.value.find((c) => c.config_id === previousConfigId)
  if (clearInvalid && previousConfig?.provider && previousConfig.provider !== provider) form.value.voice = null
  const voices = await loadVoiceOptions({ provider, configId: form.value.tts_config_id }).catch(() => [])
  await loadCloneVoices(form.value.tts_config_id).catch(() => [])
  if (clearInvalid && form.value.voice && voices.length) {
    if (!voices.some((v) => v.value === form.value.voice)) form.value.voice = null
  }
  syncFilteredVoiceOptions()
  previousTtsConfigId.value = form.value.tts_config_id
}

const setTtsConfig = async (configId, options = {}) => {
  const previousConfigId = form.value.tts_config_id
  suppressTtsConfigWatch.value = true
  form.value.tts_config_id = configId || null
  try {
    await refreshVoiceOptions({ clearInvalid: true, previousConfigId, ...options })
  } finally {
    Promise.resolve().then(() => { suppressTtsConfigWatch.value = false })
  }
}

const applyCloneVoice = async (clone) => {
  if (!clone?.tts_config_id || !clone?.provider_voice_id) return
  await setTtsConfig(clone.tts_config_id, { clearInvalid: false })
  form.value.voice = clone.provider_voice_id
}

// ── MCP services multi-select ───────────────────────────────────────────────
const selectedMcpServices = computed({
  get: () => String(form.value.mcp_service_names || '').split(',').map((i) => i.trim()).filter(Boolean),
  set: (items) => { form.value.mcp_service_names = Array.isArray(items) ? items.join(',') : '' }
})

const toggleMcpService = (svc) => {
  const curr = selectedMcpServices.value
  selectedMcpServices.value = curr.includes(svc) ? curr.filter((s) => s !== svc) : [...curr, svc]
}

// ── OpenClaw keyword tags ───────────────────────────────────────────────────
const newEnterKeyword = ref('')
const newExitKeyword = ref('')

const addEnterKeyword = () => {
  const kw = newEnterKeyword.value.trim()
  if (kw && !(form.value.openclaw_enter_keywords || []).includes(kw)) {
    form.value.openclaw_enter_keywords = [...(form.value.openclaw_enter_keywords || []), kw]
  }
  newEnterKeyword.value = ''
}
const removeEnterKeyword = (kw) => {
  form.value.openclaw_enter_keywords = (form.value.openclaw_enter_keywords || []).filter((k) => k !== kw)
}
const addExitKeyword = () => {
  const kw = newExitKeyword.value.trim()
  if (kw && !(form.value.openclaw_exit_keywords || []).includes(kw)) {
    form.value.openclaw_exit_keywords = [...(form.value.openclaw_exit_keywords || []), kw]
  }
  newExitKeyword.value = ''
}
const removeExitKeyword = (kw) => {
  form.value.openclaw_exit_keywords = (form.value.openclaw_exit_keywords || []).filter((k) => k !== kw)
}

// ── Lifecycle ───────────────────────────────────────────────────────────────
const userLabel = (user) => {
  const name = user?.username || user?.name || t('user_id_fallback', { id: user?.id })
  return `${name} (ID: ${user?.id})`
}

const loadTargetUserOptions = async () => {
  await Promise.all([
    loadKnowledgeBases().catch(() => []),
    loadCloneVoices(form.value.tts_config_id || '').catch(() => [])
  ])
  const validKbIds = new Set(knowledgeBases.value.map((i) => Number(i.id)))
  if (validKbIds.size) {
    form.value.knowledge_base_ids = (form.value.knowledge_base_ids || []).filter((id) => validKbIds.has(Number(id)))
  } else if (props.isAdmin && targetUserId.value) {
    form.value.knowledge_base_ids = []
  }
}

const applyDefaultConfigs = () => {
  if (props.mode !== 'create') return
  if (!form.value.llm_config_id) {
    const d = llmConfigs.value.find((c) => c.is_default)
    if (d) form.value.llm_config_id = d.config_id
  }
  if (!form.value.tts_config_id) {
    const d = ttsConfigs.value.find((c) => c.is_default)
    if (d) form.value.tts_config_id = d.config_id
  }
}

const reloadOptions = async () => {
  await Promise.all([
    props.isAdmin ? loadUsers().catch(() => []) : Promise.resolve([]),
    loadConfigs(),
    loadMcpServiceOptions().catch(() => [])
  ])
  applyDefaultConfigs()
  await loadTargetUserOptions()
  await refreshVoiceOptions({ clearInvalid: false })
}

watch(() => form.value.user_id, async (next, prev) => {
  if (!props.isAdmin || next === prev) return
  form.value.knowledge_base_ids = []; form.value.voice = null; previousTtsConfigId.value = null
  await loadTargetUserOptions()
  await refreshVoiceOptions({ clearInvalid: true })
})

watch(() => form.value.tts_config_id, async (next, prev) => {
  if (next === prev || suppressTtsConfigWatch.value) return
  await refreshVoiceOptions({ clearInvalid: true, previousConfigId: prev })
})

onMounted(() => reloadOptions())

// ── Exposed API ─────────────────────────────────────────────────────────────
const validate = () => {
  if (props.isAdmin && !form.value.user_id) {
    ElMessage.error(t('select_owner_user')); return Promise.reject(new Error('validation'))
  }
  if (!String(form.value.name || '').trim()) {
    ElMessage.error(t('enter_agent_name')); return Promise.reject(new Error('validation'))
  }
  if (!String(form.value.nickname || '').trim()) {
    ElMessage.error(t('enter_agent_nickname')); return Promise.reject(new Error('validation'))
  }
  return Promise.resolve(true)
}
const resetFields = () => {}
const clearValidate = () => {}
const buildPayload = () => buildAgentPayload(form.value, { isAdmin: props.isAdmin })
const hasLlmConfig = (id) => !id || llmConfigs.value.some((c) => c.config_id === id)
const hasTtsConfig = (id) => !id || ttsConfigs.value.some((c) => c.config_id === id)

defineExpose({ validate, resetFields, clearValidate, reloadOptions, refreshVoiceOptions, setTtsConfig, buildPayload, hasLlmConfig, hasTtsConfig })
</script>

<template>
  <div class="grid gap-4">
    <!-- Owner user (admin) -->
    <div v-if="isAdmin" class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('owner_user') }}</label>
      <Select v-model="form.user_id">
        <SelectTrigger class="w-full"><SelectValue :placeholder="t('select_owner_user')" /></SelectTrigger>
        <SelectContent>
          <SelectItem v-for="user in users" :key="user.id" :value="user.id">{{ userLabel(user) }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Name + Nickname -->
    <div class="grid grid-cols-2 gap-4">
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('agent_name') }}</label>
        <Input v-model="form.name" :placeholder="t('enter_admin_display_name')" maxlength="50" />
      </div>
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('agent_nickname') }}</label>
        <Input v-model="form.nickname" :placeholder="t('model_display_name_hint')" maxlength="50" />
      </div>
    </div>

    <!-- Custom prompt -->
    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('role_description') }}</label>
      <Textarea v-model="form.custom_prompt" :placeholder="t('enter_role_prompt')" rows="4" maxlength="10000" />
    </div>

    <!-- Knowledge bases (multi-select checkboxes) -->
    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('link_knowledge_base') }}</label>
      <div :class="['border border-[var(--color-line)] rounded-lg max-h-32 overflow-y-auto p-1.5', (isAdmin && !form.user_id) && 'opacity-50 pointer-events-none']">
        <div v-if="!knowledgeBases.length" class="text-xs text-[var(--color-text-tertiary)] p-2 text-center">{{ t('select_linked_knowledge_base') }}</div>
        <label v-for="kb in knowledgeBases" :key="kb.id" class="flex items-center gap-2 p-1.5 rounded hover:bg-[var(--color-surface-muted)] cursor-pointer text-sm">
          <input type="checkbox" :value="kb.id" v-model="form.knowledge_base_ids" class="accent-[var(--color-primary)]" />
          <span>{{ kb.name || t('kb_default_label', { id: kb.id }) }}</span>
        </label>
      </div>
    </div>

    <!-- LLM + TTS configs -->
    <div class="grid grid-cols-2 gap-4">
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('language_model') }}</label>
        <Select v-model="form.llm_config_id">
          <SelectTrigger class="w-full"><SelectValue :placeholder="t('select_language_model')" /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="cfg in llmConfigs" :key="cfg.config_id" :value="cfg.config_id">
              {{ cfg.is_default ? t('tts_default_label', { name: cfg.name }) : cfg.name }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('tts_config_label') }}</label>
        <Select v-model="form.tts_config_id">
          <SelectTrigger class="w-full"><SelectValue :placeholder="t('select_tts_config')" /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="cfg in ttsConfigs" :key="cfg.config_id" :value="cfg.config_id">
              {{ cfg.is_default ? t('tts_default_label', { name: cfg.name }) : cfg.name }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <!-- Voice (allow-create datalist) -->
    <div v-if="form.tts_config_id" class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('tts_voice') }}</label>
      <Input v-model="form.voice" list="agent-form-voice-datalist" :placeholder="t('select_or_enter_timbre')" />
      <datalist id="agent-form-voice-datalist">
        <option v-for="v in filteredVoiceOptions" :key="v.value" :value="v.value">{{ v.label || v.value }}</option>
      </datalist>
    </div>

    <!-- Clone voice presets -->
    <div v-if="cloneVoices.length" :class="['flex flex-wrap gap-1.5 -mt-2', loading.cloneVoices && 'opacity-50']">
      <button
        v-for="clone in cloneVoices"
        :key="clone.id"
        type="button"
        :class="['inline-flex items-center px-2.5 py-1 rounded-lg text-xs border transition-colors cursor-pointer',
          form.tts_config_id === clone.tts_config_id && form.voice === clone.provider_voice_id
            ? 'border-[var(--color-primary)] bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
            : 'border-[var(--color-line)] text-[var(--color-text-secondary)] hover:border-[var(--color-primary)]']"
        :title="`${clone.tts_config_name || clone.tts_config_id} · ${clone.provider_voice_id}`"
        @click="applyCloneVoice(clone)"
      >{{ clone.name || clone.provider_voice_id }}</button>
    </div>

    <!-- ASR speed + Memory mode + Speaker chat mode -->
    <div class="grid grid-cols-3 gap-4">
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('asr_speed') }}</label>
        <Select v-model="form.asr_speed">
          <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="normal">{{ t('normal') }}</SelectItem>
            <SelectItem value="patient">{{ t('patience') }}</SelectItem>
            <SelectItem value="fast">{{ t('fast') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('memory_mode') }}</label>
        <Select v-model="form.memory_mode">
          <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="none">{{ t('no_memory') }}</SelectItem>
            <SelectItem value="short">{{ t('short_memory') }}</SelectItem>
            <SelectItem value="long">{{ t('long_memory') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('voiceprint_chat_limit') }}</label>
        <Select v-model="form.speaker_chat_mode">
          <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="off">{{ t('close') }}</SelectItem>
            <SelectItem value="identified_only">{{ t('voiceprint_only_chat') }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <!-- MCP services multi-select -->
    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('mcp_service') }}</label>
      <div class="border border-[var(--color-line)] rounded-lg max-h-32 overflow-y-auto p-1.5">
        <div v-if="!mcpServiceOptions.length" class="text-xs text-[var(--color-text-tertiary)] p-2 text-center">{{ t('leave_blank_all_enabled') }}</div>
        <label v-for="svc in mcpServiceOptions" :key="svc" class="flex items-center gap-2 p-1.5 rounded hover:bg-[var(--color-surface-muted)] cursor-pointer text-sm">
          <input type="checkbox" :checked="selectedMcpServices.includes(svc)" @change="toggleMcpService(svc)" class="accent-[var(--color-primary)]" />
          <span>{{ svc }}</span>
        </label>
      </div>
    </div>

    <!-- OpenClaw panel -->
    <div class="border border-[var(--color-line)] rounded-xl p-4 bg-[var(--color-surface-muted)] grid gap-3">
      <div class="flex items-center justify-between gap-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ t('allow_openclaw_mode') }}</span>
        <Switch :model-value="!!form.openclaw_allowed" @update:model-value="(v) => form.openclaw_allowed = v" />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <!-- Enter keywords -->
        <div class="grid gap-1.5">
          <label class="text-xs font-medium text-[var(--color-text-secondary)]">{{ t('openclaw_enter_keyword') }}</label>
          <div class="flex flex-wrap gap-1 min-h-[28px]">
            <span v-for="kw in (form.openclaw_enter_keywords || [])" :key="kw" class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-xs bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800">
              {{ kw }}<button type="button" class="hover:opacity-70 ml-0.5" @click="removeEnterKeyword(kw)">×</button>
            </span>
          </div>
          <Input v-model="newEnterKeyword" class="h-7 text-xs" :placeholder="t('enter_press_add_keywords')" @keydown.enter.prevent="addEnterKeyword" />
        </div>
        <!-- Exit keywords -->
        <div class="grid gap-1.5">
          <label class="text-xs font-medium text-[var(--color-text-secondary)]">{{ t('openclaw_exit_keyword') }}</label>
          <div class="flex flex-wrap gap-1 min-h-[28px]">
            <span v-for="kw in (form.openclaw_exit_keywords || [])" :key="kw" class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-xs bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800">
              {{ kw }}<button type="button" class="hover:opacity-70 ml-0.5" @click="removeExitKeyword(kw)">×</button>
            </span>
          </div>
          <Input v-model="newExitKeyword" class="h-7 text-xs" :placeholder="t('enter_press_add_keywords')" @keydown.enter.prevent="addExitKeyword" />
        </div>
      </div>
    </div>
  </div>
</template>
