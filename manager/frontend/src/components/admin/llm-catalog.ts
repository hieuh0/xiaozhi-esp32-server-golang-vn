// Port of llmCatalog.js — provider data and utility functions for LLM config forms

type T = (k: string) => string

export interface ModelEntry {
  value: string
  label?: string
  thinking: ThinkingDef | false | null
  request?: Partial<RequestConfig>
  hintKey?: string
  versioned?: boolean
}

export interface ThinkingDef {
  label?: string
  options?: Array<{ label: string; value: string }>
  showBudgetFor?: string[]
  budgetMin?: number
  budgetMax?: number
  budgetStep?: number
  budgetRequiredFor?: string[]
  showEffortFor?: string[]
  effortOptions?: Array<{ label: string; value: string }>
  showClearThinkingFor?: string[]
  clearThinkingOptions?: Array<{ label: string; value: string | boolean }>
  hintKey?: string
}

export interface RequestConfig {
  allowMaxTokens: boolean
  allowTemperature: boolean
  allowTopP: boolean
  temperatureMax: number
}

export interface ThinkingConfig {
  visible: boolean
  label: string
  options: Array<{ label: string; value: string }>
  showBudgetFor: string[]
  budgetMin: number
  budgetMax: number
  budgetStep: number
  budgetRequiredFor: string[]
  showEffortFor: string[]
  effortOptions: Array<{ label: string; value: string }>
  showClearThinkingFor: string[]
  clearThinkingOptions: Array<{ label: string; value: string | boolean }>
  hint: string
}

const providerTypeMap: Record<string, string> = {
  openai: 'openai', ollama: 'ollama', azure: 'openai', anthropic: 'openai',
  zhipu: 'openai', aliyun: 'openai', doubao: 'openai', siliconflow: 'openai',
  deepseek: 'openai', dify: 'dify', coze: 'coze',
}
const knownProviders = new Set(Object.keys(providerTypeMap))
const editableBaseURLProviders = new Set(['openai', 'ollama', 'azure', 'dify', 'coze'])

// Reasoning option sets
const mkOpts = (arr: Array<[string, string]>) => arr.map(([value, label]) => ({ value, label }))
const reasoningStandard = mkOpts([['default','default'],['minimal','level_minimal'],['low','level_low'],['medium','level_medium'],['high','level_high']])
const reasoningCodex = mkOpts([['default','default'],['none','close'],['low','level_low'],['medium','level_medium'],['high','level_high']])
const reasoningCodexMax = mkOpts([['default','default'],['none','close'],['low','level_low'],['medium','level_medium'],['high','level_high'],['xhigh','level_very_high']])
const reasoningLegacy = mkOpts([['default','default'],['low','level_low'],['medium','level_medium'],['high','level_high']])
const reasoningHighOnly = mkOpts([['default','default'],['high','level_high']])
const reasoningLatest = mkOpts([['default','default'],['none','close'],['low','level_low'],['medium','level_medium'],['high','level_high'],['xhigh','level_very_high']])
const reasoningLatestPro = mkOpts([['default','default'],['medium','level_medium'],['high','level_high'],['xhigh','level_very_high']])
const boolOpts = mkOpts([['default','default'],['enabled','enable'],['disabled','close']])
const doubaoOpts = mkOpts([['default','default'],['minimal','close'],['low','level_low'],['medium','level_medium'],['high','level_high']])
const anthropicAdaptiveOpts = mkOpts([['low','level_low'],['medium','level_medium'],['high','level_high'],['max','level_very_high']])
const clearHistoryOpts = [{ value: 'default' as string | boolean, label: 'default' },{ value: true, label: 'option_clear' },{ value: false, label: 'option_retain' }]
const openAIReasoningRequest = { allowMaxTokens: false, allowTemperature: false, allowTopP: false }

const anthropicManual: ThinkingDef = { label:'deep_thinking', options: mkOpts([['default','default'],['enabled','manual_thinking']]), showBudgetFor:['enabled'], budgetMin:1024, budgetRequiredFor:['enabled'] }
const anthropicAdaptive: ThinkingDef = { ...anthropicManual, options: mkOpts([['default','default'],['enabled','manual_thinking'],['adaptive','adaptive_thinking']]), showEffortFor:['adaptive'], effortOptions:anthropicAdaptiveOpts }
const zhipuThinking: ThinkingDef = { label:'deep_thinking', options:boolOpts, showClearThinkingFor:['enabled'], clearThinkingOptions:clearHistoryOpts }
const aliyunThinking: ThinkingDef = { label:'deep_thinking', options:boolOpts, showBudgetFor:['enabled'], budgetMin:1, budgetStep:256 }
const siliconflowThinking: ThinkingDef = { label:'deep_thinking', options:boolOpts, showBudgetFor:['enabled'], budgetMin:128, budgetMax:32768, budgetStep:128 }

function m(value: string, thinking: ModelEntry['thinking'], extra: Partial<ModelEntry> = {}): ModelEntry {
  return { value, label: extra.label || value, thinking, ...extra }
}

const catalog: Record<string, { quickUrl: string; modelPlaceholderKey: string; modelHintKey?: string; models: ModelEntry[]; fallbackThinking: ThinkingDef | null }> = {
  openai: { quickUrl:'https://api.openai.com/v1', modelPlaceholderKey:'select_or_enter_model_name', modelHintKey:'model_hint_openai', models:[
    m('gpt-5.4',{label:'thinking_intensity',options:reasoningLatest},{request:openAIReasoningRequest}),
    m('gpt-5.4-pro',{label:'thinking_intensity',options:reasoningLatestPro},{request:openAIReasoningRequest}),
    m('gpt-5',{label:'thinking_intensity',options:reasoningStandard},{request:openAIReasoningRequest}),
    m('gpt-5-mini',{label:'thinking_intensity',options:reasoningStandard},{request:openAIReasoningRequest}),
    m('o3',{label:'thinking_intensity',options:reasoningLegacy},{request:openAIReasoningRequest}),
    m('o4-mini',{label:'thinking_intensity',options:reasoningLegacy},{request:openAIReasoningRequest}),
    m('o3-mini',{label:'thinking_intensity',options:reasoningLegacy},{request:openAIReasoningRequest}),
    m('o1',{label:'thinking_intensity',options:reasoningLegacy},{request:openAIReasoningRequest}),
    m('gpt-5.3-codex',{label:'thinking_intensity',options:reasoningCodexMax},{request:openAIReasoningRequest}),
    m('gpt-5.1',{label:'thinking_intensity',options:reasoningCodex},{request:openAIReasoningRequest}),
    m('gpt-5-pro',{label:'thinking_intensity',options:reasoningHighOnly},{request:openAIReasoningRequest}),
  ], fallbackThinking:{label:'thinking_intensity',options:reasoningCodex,hintKey:'fallback_hint_openai'}},
  ollama: { quickUrl:'http://127.0.0.1:11434/v1', modelPlaceholderKey:'select_or_enter_model_name', modelHintKey:'model_hint_ollama', models:[], fallbackThinking:null },
  azure: { quickUrl:'https://your-resource-name.openai.azure.com/openai/v1/', modelPlaceholderKey:'select_azure_model_or_deployment', modelHintKey:'model_hint_azure', models:[
    m('gpt-5.4',{label:'thinking_intensity',options:reasoningLatest},{request:openAIReasoningRequest}),
    m('gpt-5',{label:'thinking_intensity',options:reasoningStandard},{request:openAIReasoningRequest}),
    m('o4-mini',{label:'thinking_intensity',options:reasoningLegacy},{request:openAIReasoningRequest}),
    m('o3',{label:'thinking_intensity',options:reasoningLegacy},{request:openAIReasoningRequest}),
  ], fallbackThinking:{label:'thinking_intensity',options:reasoningCodex,hintKey:'fallback_hint_azure'}},
  anthropic: { quickUrl:'https://api.anthropic.com/v1/', modelPlaceholderKey:'select_or_enter_model_name', modelHintKey:'model_hint_anthropic', models:[
    m('claude-opus-4-6',anthropicAdaptive), m('claude-sonnet-4-6',anthropicAdaptive), m('claude-haiku-4-5',anthropicManual),
    m('claude-3-7-sonnet',anthropicManual), m('claude-sonnet-4',anthropicManual), m('claude-opus-4',anthropicManual),
  ], fallbackThinking:{...anthropicAdaptive,hintKey:'fallback_hint_anthropic'}},
  zhipu: { quickUrl:'https://open.bigmodel.cn/api/paas/v4', modelPlaceholderKey:'select_or_enter_model_name', modelHintKey:'model_hint_zhipu', models:[
    m('glm-5',zhipuThinking), m('glm-4.7',zhipuThinking), m('glm-4.7-flash',zhipuThinking), m('glm-4.6',zhipuThinking),
  ], fallbackThinking:{...zhipuThinking,hintKey:'fallback_hint_zhipu'}},
  aliyun: { quickUrl:'https://dashscope.aliyuncs.com/compatible-mode/v1', modelPlaceholderKey:'select_or_enter_model_name', modelHintKey:'model_hint_aliyun', models:[
    m('qwen-plus-latest',aliyunThinking), m('qwen-turbo-latest',aliyunThinking), m('qwen3-max',aliyunThinking),
    m('qwen3-235b-a22b',aliyunThinking), m('qwen3-30b-a3b',aliyunThinking),
  ], fallbackThinking:{...aliyunThinking,hintKey:'fallback_hint_aliyun'}},
  doubao: { quickUrl:'https://ark.cn-beijing.volces.com/api/v3', modelPlaceholderKey:'select_or_enter_model_id', modelHintKey:'model_hint_doubao', models:[
    m('doubao-seed-2-0-pro-260215',{label:'thinking_intensity',options:doubaoOpts},{label:'Doubao Seed 2.0 Pro'}),
    m('doubao-seed-2-0-lite-260215',{label:'thinking_intensity',options:doubaoOpts},{label:'Doubao Seed 2.0 Lite'}),
    m('doubao-seed-1-6-251015',{label:'thinking_intensity',options:doubaoOpts},{label:'Doubao Seed 1.6'}),
  ], fallbackThinking:{label:'thinking_intensity',options:doubaoOpts,hintKey:'fallback_hint_doubao'}},
  siliconflow: { quickUrl:'https://api.siliconflow.cn/v1', modelPlaceholderKey:'select_or_enter_model_name', modelHintKey:'model_hint_siliconflow', models:[
    m('Qwen/Qwen3-8B',siliconflowThinking), m('Qwen/Qwen3-32B',siliconflowThinking),
    m('deepseek-ai/DeepSeek-V3.2',siliconflowThinking), m('Pro/deepseek-ai/DeepSeek-V3.2',siliconflowThinking),
  ], fallbackThinking:{...siliconflowThinking,hintKey:'fallback_hint_siliconflow'}},
  deepseek: { quickUrl:'https://api.deepseek.com/v1', modelPlaceholderKey:'select_or_enter_model_name', modelHintKey:'model_hint_deepseek', models:[
    m('deepseek-chat',false,{hintKey:'model_hint_deepseek_chat'}), m('deepseek-reasoner',false,{hintKey:'model_hint_deepseek_reasoner'}),
  ], fallbackThinking:{label:'deep_thinking',options:boolOpts,hintKey:'fallback_hint_deepseek'}},
}

export function resolveLLMProvider(provider: string, type?: string): string {
  const p = (provider || '').trim().toLowerCase()
  const tp = (type || '').trim().toLowerCase()
  if (p === 'openai' && ['ollama','dify','coze'].includes(tp)) return tp
  if (knownProviders.has(p)) return p
  if (['ollama','dify','coze'].includes(tp)) return tp
  return 'openai'
}

export function getProviderFixedType(provider: string): string {
  return providerTypeMap[provider] || 'openai'
}

export function isProviderBaseURLEditable(provider: string): boolean {
  return editableBaseURLProviders.has(provider)
}

export function getProviderQuickUrl(provider: string): string {
  return catalog[provider]?.quickUrl || ''
}

export function getProviderModelOptions(provider: string): Array<{ label: string; value: string }> {
  return (catalog[provider]?.models || []).map(m => ({ label: m.label || m.value, value: m.value }))
}

export function getProviderModelHint(provider: string, modelName: string, t: T): string {
  const model = resolveProviderModel(provider, modelName)
  if (model?.hintKey) return t(model.hintKey)
  const key = catalog[provider]?.modelHintKey
  return key ? t(key) : ''
}

export function getProviderModelFieldLabel(provider: string, t: T): string {
  if (provider === 'azure') return t('deployment_name')
  if (provider === 'doubao') return t('model_id_label')
  return t('model_name_label')
}

export function getProviderModelPlaceholder(provider: string, t: T): string {
  const key = catalog[provider]?.modelPlaceholderKey || 'select_or_enter_model_name'
  return t(key)
}

export function getProviderRequestConfig(provider: string, modelName: string): RequestConfig {
  const model = resolveProviderModel(provider, modelName)
  return { allowMaxTokens:true, allowTemperature:true, allowTopP:true, temperatureMax:2, ...(model?.request || {}) }
}

function resolveProviderModel(provider: string, modelName: string): ModelEntry | null {
  const n = (modelName || '').trim().toLowerCase()
  if (!n) return null
  return (catalog[provider]?.models || []).find(m => m.value.toLowerCase() === n) || null
}

function resolveHint(obj: { hintKey?: string } | null | undefined, t: T): string {
  if (obj?.hintKey) return t(obj.hintKey)
  return ''
}

export function getProviderThinkingConfig(provider: string, modelName: string, t: T): ThinkingConfig & { visible: boolean } {
  const empty = { visible:false, label:'', options:[], showBudgetFor:[], budgetMin:1, budgetMax:100000, budgetStep:1, budgetRequiredFor:[], showEffortFor:[], effortOptions:[], showClearThinkingFor:[], clearThinkingOptions:[], hint:'' }
  const model = resolveProviderModel(provider, modelName)
  if (model?.thinking === false) return { ...empty, hint: resolveHint(model, t) }
  const source = (model?.thinking as ThinkingDef) || catalog[provider]?.fallbackThinking
  if (!source) return { ...empty, hint: resolveHint(model, t) }
  const translateOpts = (opts: Array<{label:string;value:string|boolean}>) => opts.map(o => ({ ...o, label: t(o.label as string) }))
  return {
    visible: true,
    label: t(source.label || 'deep_thinking'),
    options: translateOpts(source.options || []) as Array<{label:string;value:string}>,
    showBudgetFor: source.showBudgetFor || [],
    budgetMin: source.budgetMin || 1,
    budgetMax: source.budgetMax || 100000,
    budgetStep: source.budgetStep || 1,
    budgetRequiredFor: source.budgetRequiredFor || [],
    showEffortFor: source.showEffortFor || [],
    effortOptions: translateOpts(source.effortOptions || []) as Array<{label:string;value:string}>,
    showClearThinkingFor: source.showClearThinkingFor || [],
    clearThinkingOptions: translateOpts(source.clearThinkingOptions || clearHistoryOpts),
    hint: resolveHint(source, t) || resolveHint(model, t),
  }
}

export const LLM_PROVIDERS = [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Ollama', value: 'ollama' },
  { label: 'Azure OpenAI', value: 'azure' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'zhipu_ai', value: 'zhipu', labelKey: true },
  { label: 'aliyun', value: 'aliyun', labelKey: true },
  { label: 'doubao', value: 'doubao', labelKey: true },
  { label: 'silicon_flow', value: 'siliconflow', labelKey: true },
  { label: 'deepseek_label', value: 'deepseek', labelKey: true },
  { label: 'Dify', value: 'dify' },
  { label: 'Coze', value: 'coze' },
]
