export const TTS_PROVIDERS_WITH_VOICE_CLONE = ['doubao_ws', 'minimax', 'cosyvoice', 'aliyun_qwen', 'indextts_vllm']

const voiceCloneProviderSet = new Set(TTS_PROVIDERS_WITH_VOICE_CLONE)

export function getTTSProviderOptions(t) {
  return [
    { label: t('tts_provider_doubao_ws'), value: 'doubao_ws' },
    { label: 'Edge TTS', value: 'edge' },
    { label: t('tts_provider_edge_offline'), value: 'edge_offline' },
    { label: 'CosyVoice', value: 'cosyvoice' },
    { label: t('xunfei'), value: 'xunfei' },
    { label: t('tts_provider_xunfei_super'), value: 'xunfei_super_tts' },
    { label: 'OpenAI', value: 'openai' },
    { label: t('tts_provider_qwen'), value: 'aliyun_qwen' },
    { label: t('tts_provider_zhipu'), value: 'zhipu' },
    { label: 'Minimax', value: 'minimax' },
    { label: 'IndexTTS(vLLM)', value: 'indextts_vllm' }
  ].map((item) => ({
    ...item,
    supports_voice_clone: voiceCloneProviderSet.has(item.value)
  }))
}

export const TTS_PROVIDERS_WITH_VOICES = ['minimax', 'edge', 'doubao', 'doubao_ws', 'zhipu', 'openai', 'indextts_vllm', 'xunfei_super_tts']
