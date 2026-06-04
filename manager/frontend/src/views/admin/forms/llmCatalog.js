import { useLocaleStore } from '../../../stores/locale'
import zh from '../../../locales/zh'
import vi from '../../../locales/vi'
import en from '../../../locales/en'

const _maps = { zh, vi, en }
function _t(key) {
  try {
    const store = useLocaleStore()
    const lang = store.lang
    return _maps[lang]?.[key] ?? _maps.zh[key] ?? key
  } catch {
    return _maps.zh[key] ?? key
  }
}

function _opts(arr) {
  return arr.map(o => ({ ...o, label: _t(o.label) }))
}

const defaultOption = { label: 'default', value: 'default' }
const enableOption = { label: 'enable', value: 'enabled' }
const disableOption = { label: 'close', value: 'disabled' }
const clearHistoryOptions = [
  { label: 'default', value: 'default' },
  { label: 'option_clear', value: true },
  { label: 'option_retain', value: false }
]

function withDefault(options) {
  return [defaultOption, ...options]
}

function createModel(value, thinking, extra = {}) {
  return {
    value,
    label: value,
    thinking,
    ...extra
  }
}

const openAIReasoningStandard = withDefault([
  { label: 'level_minimal', value: 'minimal' },
  { label: 'level_low', value: 'low' },
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' }
])

const openAIReasoningCodex = withDefault([
  { label: 'close', value: 'none' },
  { label: 'level_low', value: 'low' },
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' }
])

const openAIReasoningCodexMax = withDefault([
  { label: 'close', value: 'none' },
  { label: 'level_low', value: 'low' },
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' },
  { label: 'level_very_high', value: 'xhigh' }
])

const openAIReasoningLegacy = withDefault([
  { label: 'level_low', value: 'low' },
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' }
])

const openAIReasoningHighOnly = withDefault([
  { label: 'level_high', value: 'high' }
])

const booleanThinkingOptions = withDefault([
  enableOption,
  disableOption
])

const doubaoReasoningOptions = withDefault([
  { label: 'close', value: 'minimal' },
  { label: 'level_low', value: 'low' },
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' }
])

const anthropicAdaptiveOptions = [
  { label: 'level_low', value: 'low' },
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' },
  { label: 'level_very_high', value: 'max' }
]

const openAIReasoningLatest = withDefault([
  { label: 'close', value: 'none' },
  { label: 'level_low', value: 'low' },
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' },
  { label: 'level_very_high', value: 'xhigh' }
])

const openAIReasoningLatestPro = withDefault([
  { label: 'level_medium', value: 'medium' },
  { label: 'level_high', value: 'high' },
  { label: 'level_very_high', value: 'xhigh' }
])

const openAIReasoningRequest = {
  allowMaxTokens: false,
  allowTemperature: false,
  allowTopP: false
}

const anthropicManualThinking = {
  label: 'deep_thinking',
  options: withDefault([{ label: 'manual_thinking', value: 'enabled' }]),
  showBudgetFor: ['enabled'],
  budgetMin: 1024,
  budgetRequiredFor: ['enabled']
}

const anthropicAdaptiveThinking = {
  label: 'deep_thinking',
  options: withDefault([
    { label: 'manual_thinking', value: 'enabled' },
    { label: 'adaptive_thinking', value: 'adaptive' }
  ]),
  showBudgetFor: ['enabled'],
  budgetMin: 1024,
  budgetRequiredFor: ['enabled'],
  showEffortFor: ['adaptive'],
  effortOptions: anthropicAdaptiveOptions
}

const zhipuThinkingConfig = {
  label: 'deep_thinking',
  options: booleanThinkingOptions,
  showClearThinkingFor: ['enabled'],
  clearThinkingOptions: clearHistoryOptions
}

const aliyunThinkingConfig = {
  label: 'deep_thinking',
  options: booleanThinkingOptions,
  showBudgetFor: ['enabled'],
  budgetMin: 1,
  budgetStep: 256
}

const siliconflowThinkingConfig = {
  label: 'deep_thinking',
  options: booleanThinkingOptions,
  showBudgetFor: ['enabled'],
  budgetMin: 128,
  budgetMax: 32768,
  budgetStep: 128
}

const providerTypeMap = {
  openai: 'openai',
  ollama: 'ollama',
  azure: 'openai',
  anthropic: 'openai',
  zhipu: 'openai',
  aliyun: 'openai',
  doubao: 'openai',
  siliconflow: 'openai',
  deepseek: 'openai',
  dify: 'dify',
  coze: 'coze'
}

const knownProviders = new Set(Object.keys(providerTypeMap))

const editableBaseURLProviders = new Set(['openai', 'ollama', 'azure', 'dify', 'coze'])

const catalog = {
  openai: {
    quickUrl: 'https://api.openai.com/v1',
    modelPlaceholderKey: 'select_or_enter_model_name',
    modelHintKey: 'model_hint_openai',
    models: [
      createModel('gpt-5.4', { label: 'thinking_intensity', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-pro', { label: 'thinking_intensity', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-mini', { label: 'thinking_intensity', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-nano', { label: 'thinking_intensity', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2', { label: 'thinking_intensity', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-pro', { label: 'thinking_intensity', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5-chat-latest', false, { hintKey: 'model_hint_gpt5_chat_latest' }),
      createModel('gpt-5-pro', { label: 'thinking_intensity', options: openAIReasoningHighOnly }, { request: openAIReasoningRequest }),
      createModel('gpt-5', { label: 'thinking_intensity', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-mini', { label: 'thinking_intensity', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-nano', { label: 'thinking_intensity', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5.3-codex', { label: 'thinking_intensity', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-codex', { label: 'thinking_intensity', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5-codex', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1', { label: 'thinking_intensity', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex', { label: 'thinking_intensity', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex-mini', { label: 'thinking_intensity', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex-max', { label: 'thinking_intensity', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('o3', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o4-mini', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3-mini', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o1', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest })
    ],
    fallbackThinking: {
      label: 'thinking_intensity',
      options: openAIReasoningCodex,
      hintKey: 'fallback_hint_openai'
    }
  },
  ollama: {
    quickUrl: 'http://127.0.0.1:11434/v1',
    modelPlaceholderKey: 'select_or_enter_model_name',
    modelHintKey: 'model_hint_ollama',
    models: [],
    fallbackThinking: null
  },
  azure: {
    quickUrl: 'https://your-resource-name.openai.azure.com/openai/v1/',
    modelPlaceholderKey: 'select_azure_model_or_deployment',
    modelHintKey: 'model_hint_azure',
    models: [
      createModel('gpt-5.4', { label: 'thinking_intensity', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-pro', { label: 'thinking_intensity', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2', { label: 'thinking_intensity', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-chat', false, { hintKey: 'model_hint_azure_gpt52_chat' }),
      createModel('gpt-5.3-codex', { label: 'thinking_intensity', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-codex', { label: 'thinking_intensity', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5-mini', { label: 'thinking_intensity', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-nano', { label: 'thinking_intensity', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-chat', { label: 'thinking_intensity', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-pro', { label: 'thinking_intensity', options: openAIReasoningHighOnly }, { request: openAIReasoningRequest }),
      createModel('o4-mini', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3-mini', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o1', { label: 'thinking_intensity', options: openAIReasoningLegacy }, { request: openAIReasoningRequest })
    ],
    fallbackThinking: {
      label: 'thinking_intensity',
      options: openAIReasoningCodex,
      hintKey: 'fallback_hint_azure'
    }
  },
  anthropic: {
    quickUrl: 'https://api.anthropic.com/v1/',
    modelPlaceholderKey: 'select_or_enter_model_name',
    modelHintKey: 'model_hint_anthropic',
    models: [
      createModel('claude-opus-4-6', anthropicAdaptiveThinking),
      createModel('claude-sonnet-4-6', anthropicAdaptiveThinking),
      createModel('claude-haiku-4-5', anthropicManualThinking),
      createModel('claude-3-7-sonnet', anthropicManualThinking),
      createModel('claude-sonnet-4', anthropicManualThinking),
      createModel('claude-opus-4', anthropicManualThinking),
      createModel('claude-opus-4-1', anthropicManualThinking)
    ],
    fallbackThinking: {
      ...anthropicAdaptiveThinking,
      hintKey: 'fallback_hint_anthropic'
    }
  },
  zhipu: {
    quickUrl: 'https://open.bigmodel.cn/api/paas/v4',
    modelPlaceholderKey: 'select_or_enter_model_name',
    modelHintKey: 'model_hint_zhipu',
    models: [
      createModel('glm-5', zhipuThinkingConfig),
      createModel('glm-4.7', zhipuThinkingConfig),
      createModel('glm-4.7-flashx', zhipuThinkingConfig),
      createModel('glm-4.7-flash', zhipuThinkingConfig),
      createModel('glm-4.6', zhipuThinkingConfig),
      createModel('glm-4.6v', zhipuThinkingConfig),
      createModel('glm-4.5', zhipuThinkingConfig),
      createModel('glm-4.5-air', zhipuThinkingConfig),
      createModel('glm-4.5-airx', zhipuThinkingConfig),
      createModel('glm-4.5v', zhipuThinkingConfig)
    ],
    fallbackThinking: {
      ...zhipuThinkingConfig,
      hintKey: 'fallback_hint_zhipu'
    }
  },
  aliyun: {
    quickUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    modelPlaceholderKey: 'select_or_enter_model_name',
    modelHintKey: 'model_hint_aliyun',
    models: [
      createModel('qwen-plus-latest', aliyunThinkingConfig),
      createModel('qwen-turbo-latest', aliyunThinkingConfig),
      createModel('qwen3-max', aliyunThinkingConfig),
      createModel('qwen3-235b-a22b', aliyunThinkingConfig),
      createModel('qwen3-30b-a3b', aliyunThinkingConfig),
      createModel('qwen3-next-80b-a3b-thinking', aliyunThinkingConfig),
      createModel('glm-4.7', aliyunThinkingConfig),
      createModel('glm-4.6', aliyunThinkingConfig),
      createModel('glm-4.5', aliyunThinkingConfig),
      createModel('glm-4.5-air', aliyunThinkingConfig),
      createModel('kimi-k2-thinking', aliyunThinkingConfig),
      createModel('qwen3-235b-a22b-thinking-2507', aliyunThinkingConfig, { versioned: true }),
      createModel('qwen3-30b-a3b-thinking-2507', aliyunThinkingConfig, { versioned: true }),
      createModel('kimi/kimi-k2.5', aliyunThinkingConfig, { versioned: true })
    ],
    fallbackThinking: {
      ...aliyunThinkingConfig,
      hintKey: 'fallback_hint_aliyun'
    }
  },
  doubao: {
    quickUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    modelPlaceholderKey: 'select_or_enter_model_id',
    modelHintKey: 'model_hint_doubao',
    models: [
      createModel('doubao-seed-2-0-pro-260215', { label: 'thinking_intensity', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Pro (doubao-seed-2-0-pro-260215)' }),
      createModel('doubao-seed-2-0-lite-260215', { label: 'thinking_intensity', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Lite (doubao-seed-2-0-lite-260215)' }),
      createModel('doubao-seed-2-0-mini-260215', { label: 'thinking_intensity', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Mini (doubao-seed-2-0-mini-260215)' }),
      createModel('doubao-seed-1-6-251015', { label: 'thinking_intensity', options: doubaoReasoningOptions }, { label: 'Doubao Seed 1.6 (doubao-seed-1-6-251015)' })
    ],
    fallbackThinking: {
      label: 'thinking_intensity',
      options: doubaoReasoningOptions,
      hintKey: 'fallback_hint_doubao'
    }
  },
  siliconflow: {
    quickUrl: 'https://api.siliconflow.cn/v1',
    modelPlaceholderKey: 'select_or_enter_model_name',
    modelHintKey: 'model_hint_siliconflow',
    models: [
      createModel('Pro/zai-org/GLM-5', siliconflowThinkingConfig),
      createModel('Pro/zai-org/GLM-4.7', siliconflowThinkingConfig),
      createModel('deepseek-ai/DeepSeek-V3.2', siliconflowThinkingConfig),
      createModel('Pro/deepseek-ai/DeepSeek-V3.2', siliconflowThinkingConfig),
      createModel('zai-org/GLM-4.6', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-8B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-14B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-32B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-30B-A3B', siliconflowThinkingConfig),
      createModel('tencent/Hunyuan-A13B-Instruct', siliconflowThinkingConfig),
      createModel('zai-org/GLM-4.5V', siliconflowThinkingConfig),
      createModel('deepseek-ai/DeepSeek-V3.1-Terminus', siliconflowThinkingConfig),
      createModel('Pro/deepseek-ai/DeepSeek-V3.1-Terminus', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-397B-A17B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-122B-A10B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-35B-A3B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-27B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-9B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-4B', siliconflowThinkingConfig)
    ],
    fallbackThinking: {
      ...siliconflowThinkingConfig,
      hintKey: 'fallback_hint_siliconflow'
    }
  },
  deepseek: {
    quickUrl: 'https://api.deepseek.com/v1',
    modelPlaceholderKey: 'select_or_enter_model_name',
    modelHintKey: 'model_hint_deepseek',
    models: [
      createModel('deepseek-chat', false, {
        hintKey: 'model_hint_deepseek_chat'
      }),
      createModel('deepseek-reasoner', false, {
        hintKey: 'model_hint_deepseek_reasoner'
      })
    ],
    fallbackThinking: {
      label: 'deep_thinking',
      options: booleanThinkingOptions,
      hintKey: 'fallback_hint_deepseek'
    }
  }
}

function cloneOptions(options = []) {
  return options.map(option => ({ ...option }))
}

function normalizeModelName(modelName) {
  return String(modelName || '').trim().toLowerCase()
}

export function resolveLLMProvider(provider, type) {
  const normalizedProvider = String(provider || '').trim().toLowerCase()
  const normalizedType = String(type || '').trim().toLowerCase()

  if (normalizedProvider === 'openai' && ['ollama', 'dify', 'coze'].includes(normalizedType)) {
    return normalizedType
  }
  if (knownProviders.has(normalizedProvider)) {
    return normalizedProvider
  }
  if (['ollama', 'dify', 'coze'].includes(normalizedType)) {
    return normalizedType
  }
  return 'openai'
}

export function getProviderFixedType(provider) {
  return providerTypeMap[provider] || 'openai'
}

export function isProviderBaseURLEditable(provider) {
  return editableBaseURLProviders.has(provider)
}

export function getProviderQuickUrl(provider) {
  return catalog[provider]?.quickUrl || ''
}

export function getProviderModelOptions(provider) {
  return (catalog[provider]?.models || []).map(model => ({
    label: model.versioned ? `${model.label} ${_t('versioned_suffix')}` : model.label,
    value: model.value
  }))
}

export function getProviderModelHint(provider) {
  const key = catalog[provider]?.modelHintKey
  return key ? _t(key) : ''
}

export function getProviderModelFieldLabel(provider) {
  if (provider === 'azure') {
    return _t('deployment_name')
  }
  if (provider === 'doubao') {
    return _t('model_id_label')
  }
  return _t('model_name_label')
}

export function getProviderModelPlaceholder(provider) {
  const key = catalog[provider]?.modelPlaceholderKey
  return _t(key || 'select_or_enter_model_name')
}

export function resolveProviderModel(provider, modelName) {
  const normalized = normalizeModelName(modelName)
  if (!normalized) {
    return null
  }

  const models = catalog[provider]?.models || []
  return models.find(model => normalizeModelName(model.value) === normalized) || null
}

export function getProviderRequestConfig(provider, modelName) {
  const model = resolveProviderModel(provider, modelName)
  return {
    allowMaxTokens: true,
    allowTemperature: true,
    allowTopP: true,
    temperatureMax: 2,
    ...(model?.request || {})
  }
}

function resolveHint(obj) {
  if (obj?.hintKey) return _t(obj.hintKey)
  return obj?.hint || ''
}

export function getProviderThinkingConfig(provider, modelName) {
  const model = resolveProviderModel(provider, modelName)
  if (model?.thinking === false) {
    return {
      visible: false,
      hint: resolveHint(model)
    }
  }

  const source = model?.thinking || catalog[provider]?.fallbackThinking
  if (!source) {
    return {
      visible: false,
      hint: resolveHint(model)
    }
  }

  return {
    visible: true,
    label: _t(source.label || 'deep_thinking'),
    options: _opts(cloneOptions(source.options)),
    showBudgetFor: [...(source.showBudgetFor || [])],
    budgetMin: source.budgetMin || 1,
    budgetMax: source.budgetMax || 100000,
    budgetStep: source.budgetStep || 1,
    budgetRequiredFor: [...(source.budgetRequiredFor || [])],
    showEffortFor: [...(source.showEffortFor || [])],
    effortOptions: _opts(cloneOptions(source.effortOptions || [])),
    showClearThinkingFor: [...(source.showClearThinkingFor || [])],
    clearThinkingOptions: _opts(cloneOptions(source.clearThinkingOptions || clearHistoryOptions)),
    hint: resolveHint(model) || resolveHint(source)
  }
}
