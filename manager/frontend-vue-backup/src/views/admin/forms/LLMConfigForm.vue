<script setup>
import { computed, watch } from 'vue'
import { Info } from '@lucide/vue'
import { useLocale } from '../../../composables/useLocale'
import {
  getProviderFixedType,
  getProviderModelFieldLabel,
  getProviderModelHint,
  getProviderModelOptions,
  getProviderModelPlaceholder,
  getProviderQuickUrl,
  getProviderRequestConfig,
  getProviderThinkingConfig,
  isProviderBaseURLEditable,
  resolveLLMProvider
} from './llmCatalog'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import NumberInput from '@/components/ui/number-input.vue'
import { Alert, AlertDescription } from '@/components/ui/alert'

const { t } = useLocale()

const props = defineProps({
  model: { type: Object, required: true }
})

const resolvedProvider = computed(() => resolveLLMProvider(props.model?.provider, props.model?.type))
const effectiveType = computed(() => getProviderFixedType(resolvedProvider.value))
const isOpenAIOrOllama = computed(() => effectiveType.value === 'openai' || effectiveType.value === 'ollama')
const isOllama = computed(() => effectiveType.value === 'ollama')
const isDify = computed(() => effectiveType.value === 'dify')
const isCoze = computed(() => effectiveType.value === 'coze')
const showBaseURL = computed(() => isProviderBaseURLEditable(resolvedProvider.value))
const apiKeyRequired = computed(() => !isOllama.value)

const defaultThinkingMode = 'default'

const provider = computed(() => resolvedProvider.value || '')
const modelFieldLabel = computed(() => getProviderModelFieldLabel(provider.value))
const modelPlaceholder = computed(() => getProviderModelPlaceholder(provider.value))
const modelOptions = computed(() => getProviderModelOptions(provider.value))
const thinkingDefinition = computed(() => getProviderThinkingConfig(provider.value, props.model?.model_name))
const modelHint = computed(() => {
  if (!thinkingDefinition.value?.visible && thinkingDefinition.value?.hint) {
    return thinkingDefinition.value.hint
  }
  return getProviderModelHint(provider.value)
})
const requestConfig = computed(() => getProviderRequestConfig(provider.value, props.model?.model_name))

const requestCapabilityHint = computed(() => {
  const blockedFields = []
  if (!requestConfig.value.allowMaxTokens) blockedFields.push('max_tokens')
  if (!requestConfig.value.allowTemperature) blockedFields.push('temperature')
  if (!requestConfig.value.allowTopP) blockedFields.push('top_p')
  if (!blockedFields.length) return ''
  return t('blocked_fields_warning', { fields: blockedFields.join(t('enum_join_sep')) })
})

const thinkingConfig = computed(() => {
  const config = thinkingDefinition.value
  if (!config?.visible) {
    return {
      visible: false,
      label: t('deep_thinking'),
      options: [],
      showBudget: false,
      budgetMin: 1, budgetMax: 100000, budgetStep: 1, budgetRequired: false,
      showEffort: false, effortOptions: [],
      showClearThinking: false, clearThinkingOptions: [],
      hint: config?.hint || ''
    }
  }
  const showBudget = config.showBudgetFor.includes(props.model?.thinking_mode)
  const showEffort = config.showEffortFor.includes(props.model?.thinking_mode)
  const showClearThinking = config.showClearThinkingFor.includes(props.model?.thinking_mode)
  const budgetRequired = config.budgetRequiredFor.includes(props.model?.thinking_mode)
  return {
    visible: true,
    label: config.label,
    options: config.options,
    showBudget, budgetMin: config.budgetMin, budgetMax: config.budgetMax, budgetStep: config.budgetStep, budgetRequired,
    showEffort, effortOptions: config.effortOptions,
    showClearThinking, clearThinkingOptions: config.clearThinkingOptions,
    hint: config.hint || t('deep_thinking_warning')
  }
})

function normalizeThinkingState(p) {
  if (!props.model) return
  const config = getProviderThinkingConfig(p, props.model?.model_name)
  if (!config?.visible) {
    props.model.thinking_mode = defaultThinkingMode
    props.model.thinking_budget_tokens = null
    props.model.thinking_clear_thinking = 'default'
    return
  }
  const options = config.options.map(o => o.value)
  if (!options.includes(props.model.thinking_mode)) props.model.thinking_mode = defaultThinkingMode
  if (props.model.thinking_budget_tokens !== null && props.model.thinking_budget_tokens !== undefined && props.model.thinking_budget_tokens !== '') {
    const budgetValue = Number(props.model.thinking_budget_tokens)
    if (Number.isNaN(budgetValue) || budgetValue < (config.budgetMin || 1)) props.model.thinking_budget_tokens = null
  }
  if (!props.model.thinking_effort) props.model.thinking_effort = 'medium'
  if (props.model.thinking_clear_thinking === undefined || props.model.thinking_clear_thinking === null || props.model.thinking_clear_thinking === '') {
    props.model.thinking_clear_thinking = 'default'
  }
}

function applyProviderDefaults(value, forceEditableURL = false, resetModel = false) {
  if (!value || !props.model) return
  props.model.type = getProviderFixedType(value)
  if (isProviderBaseURLEditable(value)) {
    const quickUrl = getProviderQuickUrl(value)
    if (forceEditableURL || !props.model.base_url) props.model.base_url = quickUrl
  } else {
    props.model.base_url = ''
  }
  if (resetModel) props.model.model_name = ''
  if (value === 'dify' && (!props.model.model_name || resetModel)) props.model.model_name = 'dify'
  if (value === 'coze') {
    if (!props.model.model_name || resetModel) props.model.model_name = 'coze'
    if (!props.model.connector_id) props.model.connector_id = '1024'
  }
}

watch(() => props.model?.provider, (value) => {
  if (!value || !props.model) return
  applyProviderDefaults(value, false)
  normalizeThinkingState(value)
}, { immediate: true })

watch(() => props.model?.model_name, () => {
  if (!provider.value) return
  normalizeThinkingState(provider.value)
}, { immediate: true })

function onProviderChange(value) {
  if (!value || !props.model) return
  applyProviderDefaults(value, true, true)
}

function getJsonData() {
  const m = props.model
  const providerName = resolveLLMProvider(m?.provider, m?.type)
  const providerType = getProviderFixedType(providerName)
  const thinking = buildThinkingPayload(m)
  if (providerType === 'dify') {
    const config = { api_key: m.api_key, user_prefix: m.user_prefix }
    if (m.base_url) config.base_url = m.base_url
    return JSON.stringify(config, null, 2)
  }
  if (providerType === 'coze') {
    const config = { api_key: m.api_key, bot_id: m.bot_id, user_prefix: m.user_prefix, connector_id: m.connector_id }
    if (m.base_url) config.base_url = m.base_url
    return JSON.stringify(config, null, 2)
  }
  const config = { model_name: m.model_name, api_key: m.api_key }
  if (isProviderBaseURLEditable(providerName) && m.base_url) config.base_url = m.base_url
  if (requestConfig.value.allowMaxTokens && m.max_tokens !== undefined && m.max_tokens !== null && m.max_tokens !== '') config.max_tokens = m.max_tokens
  if (requestConfig.value.allowTemperature && m.temperature !== undefined && m.temperature !== null) config.temperature = m.temperature
  if (requestConfig.value.allowTopP && m.top_p !== undefined && m.top_p !== null) config.top_p = m.top_p
  if (thinking) config.thinking = thinking
  return JSON.stringify(config, null, 2)
}

function buildThinkingPayload(model) {
  const providerName = resolveLLMProvider(model?.provider, model?.type)
  const config = getProviderThinkingConfig(providerName, model?.model_name)
  if (!config?.visible) return undefined
  const mode = model?.thinking_mode || defaultThinkingMode
  if (mode === defaultThinkingMode) return undefined
  const payload = { mode }
  if ((config.showBudgetFor || []).includes(mode) && model?.thinking_budget_tokens !== null && model?.thinking_budget_tokens !== undefined && model?.thinking_budget_tokens !== '') {
    payload.budget_tokens = Number(model.thinking_budget_tokens)
  }
  if ((config.showEffortFor || []).includes(mode) && model?.thinking_effort) payload.effort = model.thinking_effort
  if ((config.showClearThinkingFor || []).includes(mode) && typeof model?.thinking_clear_thinking === 'boolean') {
    payload.clear_thinking = model.thinking_clear_thinking
  }
  return payload
}

function parseClearThinkingValue(v) {
  if (v === 'true') return true
  if (v === 'false') return false
  return v
}

function validate(callback) {
  const m = props.model
  let error = null
  if (!m.name?.trim()) error = t('enter_config_name')
  else if (!m.config_id?.trim()) error = t('enter_config_id')
  else if (!m.provider) error = t('select_provider')
  else if (isOpenAIOrOllama.value && !m.model_name?.trim()) error = t('enter_model_name')
  else if (apiKeyRequired.value && !m.api_key?.trim()) error = t('enter_api_password')
  else if (showBaseURL.value && !m.base_url?.trim()) error = t('enter_base_url')
  else if (isCoze.value && !m.bot_id?.trim()) error = t('enter_coze_bot_id_v2')
  else if (isOpenAIOrOllama.value && requestConfig.value.allowMaxTokens && (!m.max_tokens || Number(m.max_tokens) < 1 || Number(m.max_tokens) > 100000)) error = t('max_tokens_range')
  else if (thinkingConfig.value.showBudget && thinkingConfig.value.budgetRequired && (m.thinking_budget_tokens === null || m.thinking_budget_tokens === undefined || m.thinking_budget_tokens === '')) error = t('thinking_budget_required')
  else if (thinkingConfig.value.showBudget && m.thinking_budget_tokens !== null && m.thinking_budget_tokens !== undefined && m.thinking_budget_tokens !== '' && Number(m.thinking_budget_tokens) < thinkingConfig.value.budgetMin) error = t('budget_min_validation', { min: thinkingConfig.value.budgetMin })

  const valid = !error
  if (callback) {
    callback(valid)
    return Promise.resolve(valid)
  }
  return valid ? Promise.resolve() : Promise.reject(new Error(error))
}

function resetFields() {}

defineExpose({ validate, getJsonData, resetFields })
</script>

<template>
  <div class="space-y-1 py-1">
    <!-- Provider -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('provider') }}</label>
      <Select v-model="model.provider" @update:model-value="onProviderChange">
        <SelectTrigger><SelectValue :placeholder="t('select_provider')" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="openai">OpenAI</SelectItem>
          <SelectItem value="ollama">Ollama</SelectItem>
          <SelectItem value="azure">Azure OpenAI</SelectItem>
          <SelectItem value="anthropic">Anthropic</SelectItem>
          <SelectItem value="zhipu">{{ t('zhipu_ai') }}</SelectItem>
          <SelectItem value="aliyun">{{ t('aliyun') }}</SelectItem>
          <SelectItem value="doubao">{{ t('doubao') }}</SelectItem>
          <SelectItem value="siliconflow">{{ t('silicon_flow') }}</SelectItem>
          <SelectItem value="deepseek">{{ t('deepseek_label') }}</SelectItem>
          <SelectItem value="dify">Dify</SelectItem>
          <SelectItem value="coze">Coze</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Config name -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_name') }}</label>
      <Input v-model="model.name" :placeholder="t('enter_config_name')" />
    </div>

    <!-- Config ID -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('config_id') }}</label>
      <Input v-model="model.config_id" :placeholder="t('enter_unique_config_id')" />
    </div>

    <!-- Model name (free-text with datalist suggestions) -->
    <template v-if="isOpenAIOrOllama">
      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ modelFieldLabel }}</label>
        <div>
          <Input v-model="model.model_name" :placeholder="modelPlaceholder" list="llm-model-list" />
          <datalist id="llm-model-list">
            <option v-for="option in modelOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </datalist>
        </div>
      </div>
      <div v-if="modelHint" class="grid grid-cols-[140px_1fr] gap-3 items-start py-1">
        <span />
        <Alert variant="info" class="py-2">
          <Info class="w-4 h-4" />
          <AlertDescription class="text-xs">{{ modelHint }}</AlertDescription>
        </Alert>
      </div>
    </template>

    <!-- API key -->
    <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">
        {{ t('api_key') }}
        <span v-if="!apiKeyRequired" class="text-[10px] text-[var(--color-text-tertiary)] block">{{ t('optional') }}</span>
      </label>
      <Input v-model="model.api_key" type="password" :placeholder="t('enter_api_password')" />
    </div>

    <!-- Base URL -->
    <div v-if="showBaseURL" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('base_url') }}</label>
      <Input v-model="model.base_url" :placeholder="t('enter_base_url')" />
    </div>

    <!-- Bot ID (Coze) -->
    <div v-if="isCoze" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Bot ID</label>
      <Input v-model="model.bot_id" :placeholder="t('enter_coze_bot_id')" />
    </div>

    <!-- User prefix (Dify/Coze) -->
    <div v-if="isDify || isCoze" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('user_prefix') }}</label>
      <Input v-model="model.user_prefix" :placeholder="t('optional_default_xiaozhi')" />
    </div>

    <!-- Connector ID (Coze) -->
    <div v-if="isCoze" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
      <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Connector ID</label>
      <Input v-model="model.connector_id" :placeholder="t('optional_default_1024')" />
    </div>

    <!-- OpenAI/Ollama request params -->
    <template v-if="isOpenAIOrOllama">
      <div v-if="requestConfig.allowMaxTokens" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">max_tokens</label>
        <NumberInput v-model="model.max_tokens" :min="1" :max="100000" />
      </div>
      <div v-if="requestConfig.allowTemperature" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('temperature') }}</label>
        <NumberInput v-model="model.temperature" :min="0" :max="requestConfig.temperatureMax" :step="0.1" />
      </div>
      <div v-if="requestConfig.allowTopP" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">Top P</label>
        <NumberInput v-model="model.top_p" :min="0" :max="1" :step="0.1" />
      </div>
      <div v-if="requestCapabilityHint" class="grid grid-cols-[140px_1fr] gap-3 items-start py-1">
        <span />
        <Alert variant="info" class="py-2">
          <Info class="w-4 h-4" />
          <AlertDescription class="text-xs">{{ requestCapabilityHint }}</AlertDescription>
        </Alert>
      </div>
    </template>

    <!-- Thinking config -->
    <template v-if="thinkingConfig.visible">
      <div class="flex items-center gap-3 py-3">
        <span class="text-sm font-semibold text-[var(--color-text)]">{{ thinkingConfig.label }}</span>
        <Separator class="flex-1" />
      </div>

      <div class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ thinkingConfig.label }}</label>
        <Select v-model="model.thinking_mode">
          <SelectTrigger><SelectValue :placeholder="t('select_deep_thinking_mode')" /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="option in thinkingConfig.options" :key="option.value" :value="option.value">
              {{ option.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div v-if="thinkingConfig.showEffort" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('thinking_level') }}</label>
        <Select v-model="model.thinking_effort">
          <SelectTrigger><SelectValue :placeholder="t('select_thinking_depth')" /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="option in thinkingConfig.effortOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div v-if="thinkingConfig.showBudget" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('thinking_budget') }}</label>
        <NumberInput
          v-model="model.thinking_budget_tokens"
          :min="thinkingConfig.budgetMin"
          :max="thinkingConfig.budgetMax"
          :step="thinkingConfig.budgetStep"
        />
      </div>

      <div v-if="thinkingConfig.showClearThinking" class="grid grid-cols-[140px_1fr] gap-3 items-center py-2">
        <label class="text-sm text-[var(--color-text-secondary)] text-right shrink-0">{{ t('history_thinking_chain') }}</label>
        <Select
          :model-value="String(model.thinking_clear_thinking)"
          @update:model-value="v => model.thinking_clear_thinking = parseClearThinkingValue(v)"
        >
          <SelectTrigger><SelectValue :placeholder="t('select_thinking_chain_mode')" /></SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="option in thinkingConfig.clearThinkingOptions"
              :key="String(option.value)"
              :value="String(option.value)"
            >
              {{ option.label }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div class="grid grid-cols-[140px_1fr] gap-3 items-start py-1">
        <span />
        <Alert class="py-2 border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20">
          <Info class="w-4 h-4 text-amber-600 dark:text-amber-400" />
          <AlertDescription class="text-xs text-amber-700 dark:text-amber-300">{{ thinkingConfig.hint }}</AlertDescription>
        </Alert>
      </div>
    </template>
  </div>
</template>
