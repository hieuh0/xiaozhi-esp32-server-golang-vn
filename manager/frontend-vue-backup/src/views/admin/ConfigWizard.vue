<script setup>
import { onMounted } from 'vue'
import { Check, Copy } from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { useConfigWizard } from '@/composables/useConfigWizard'
import { useLocale } from '@/composables/useLocale'
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import OTAConfigForm from './forms/OTAConfigForm.vue'
import VADConfigForm from './forms/VADConfigForm.vue'
import ASRConfigForm from './forms/ASRConfigForm.vue'
import LLMConfigForm from './forms/LLMConfigForm.vue'
import TTSConfigForm from './forms/TTSConfigForm.vue'

const { t } = useLocale()
const authStore = useAuthStore()
const router = useRouter()

const {
  currentStep, saving, testingStep, otaTestLoading, otaTestResult,
  otaForm, vadForm, vadFormRef, vadFormRules, asrForm, asrFormRef, asrFormRules,
  llmForm, llmFormRef, llmFormRules, ttsForm, ttsFormRef, ttsFormRules,
  voiceOptions, voiceLoading,
  finalOtaUrl, finalWsUrl, finalMqttEndpoint, finalUdpEndpoint,
  saveAndNext, skipStep, prevStep, testCurrentStepConfig, runOtaTest,
  handleTtsVoiceOptionsRequest, copyToClipboard, initialize
} = useConfigWizard()

const wizardSteps = [
  { title: 'OTA', desc: () => t('service_address') },
  { title: 'VAD', desc: () => t('vad') },
  { title: 'ASR', desc: () => t('asr') },
  { title: 'LLM', desc: () => t('llm') },
  { title: 'TTS', desc: () => t('tts') },
]

onMounted(async () => {
  if (authStore.isAdmin) localStorage.setItem('admin_first_login_done', '1')
  await initialize()
})
</script>

<template>
  <div class="px-6 pb-8 max-w-[860px] mx-auto">
    <!-- Step indicator -->
    <div class="flex items-center justify-center gap-1 mb-6 flex-wrap">
      <template v-for="(step, i) in wizardSteps" :key="i">
        <div class="flex items-center gap-1.5">
          <div :class="['w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold border-2 transition-colors',
            i < currentStep
              ? 'bg-[var(--color-primary)] border-[var(--color-primary)] text-white'
              : i === currentStep
              ? 'border-[var(--color-primary)] text-[var(--color-primary)] bg-transparent'
              : 'border-[var(--color-line)] text-[var(--color-text-tertiary)] bg-transparent']">
            <Check v-if="i < currentStep" class="w-3.5 h-3.5" />
            <span v-else>{{ i + 1 }}</span>
          </div>
          <div class="hidden sm:flex flex-col leading-none">
            <span :class="['text-xs font-semibold', i <= currentStep ? 'text-[var(--color-text)]' : 'text-[var(--color-text-tertiary)]']">{{ step.title }}</span>
            <span class="text-[10px] text-[var(--color-text-tertiary)] mt-0.5">{{ step.desc() }}</span>
          </div>
        </div>
        <div v-if="i < wizardSteps.length - 1"
          :class="['h-0.5 w-8 mx-1 transition-colors', i < currentStep ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-line)]']" />
      </template>
    </div>

    <!-- Step card -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
      <!-- Step 0: OTA -->
      <template v-if="currentStep === 0">
        <div class="text-base font-semibold text-[var(--color-text)] mb-1">{{ t('ota_config') }}</div>
        <p class="text-sm text-[var(--color-text-secondary)] mb-5">{{ t('domain_ip_hint') }}</p>
        <OTAConfigForm :model="otaForm" />
      </template>

      <!-- Step 1: VAD -->
      <template v-if="currentStep === 1">
        <div class="text-base font-semibold text-[var(--color-text)] mb-5">{{ t('vad_config') }}</div>
        <VADConfigForm ref="vadFormRef" :model="vadForm" :rules="vadFormRules" class="mb-6" />
      </template>

      <!-- Step 2: ASR -->
      <template v-if="currentStep === 2">
        <div class="text-base font-semibold text-[var(--color-text)] mb-5">{{ t('asr_config') }}</div>
        <ASRConfigForm ref="asrFormRef" :model="asrForm" :rules="asrFormRules" class="mb-6" />
      </template>

      <!-- Step 3: LLM -->
      <template v-if="currentStep === 3">
        <div class="text-base font-semibold text-[var(--color-text)] mb-5">{{ t('llm_config') }}</div>
        <LLMConfigForm ref="llmFormRef" :model="llmForm" :rules="llmFormRules" class="mb-6" />
      </template>

      <!-- Step 4: TTS -->
      <template v-if="currentStep === 4">
        <div class="text-base font-semibold text-[var(--color-text)] mb-5">{{ t('tts_config') }}</div>
        <TTSConfigForm ref="ttsFormRef" :model="ttsForm" :rules="ttsFormRules"
          :voice-options="voiceOptions" :voice-loading="voiceLoading" class="mb-6"
          @request-voice-options="handleTtsVoiceOptionsRequest" />
      </template>

      <!-- Step 5: Result -->
      <template v-if="currentStep === 5">
        <div class="text-base font-semibold text-[var(--color-text)] mb-1">{{ t('config_done_title') }}</div>
        <p class="text-sm text-[var(--color-text-secondary)] mb-5">{{ t('config_done_hint') }}</p>
        <div class="grid gap-4 mb-6">
          <div>
            <label class="block text-sm text-[var(--color-text-secondary)] mb-1.5">{{ t('ota_addr_label') }}</label>
            <div class="flex gap-2">
              <Input :model-value="finalOtaUrl" readonly class="flex-1 font-mono text-xs" />
              <Button variant="outline" size="sm" @click="copyToClipboard(finalOtaUrl)">
                <Copy class="w-3.5 h-3.5 mr-1" />{{ t('copy') }}
              </Button>
            </div>
          </div>
          <div>
            <label class="block text-sm text-[var(--color-text-secondary)] mb-1.5">{{ t('ws_addr_label') }}</label>
            <div class="flex gap-2">
              <Input :model-value="finalWsUrl" readonly class="flex-1 font-mono text-xs" />
              <Button variant="outline" size="sm" @click="copyToClipboard(finalWsUrl)">
                <Copy class="w-3.5 h-3.5 mr-1" />{{ t('copy') }}
              </Button>
            </div>
          </div>
          <div v-if="otaForm.enableMqttUdp && finalMqttEndpoint">
            <label class="block text-sm text-[var(--color-text-secondary)] mb-1.5">{{ t('mqtt_endpoint_label') }}</label>
            <div class="flex gap-2">
              <Input :model-value="finalMqttEndpoint" readonly class="flex-1 font-mono text-xs" />
              <Button variant="outline" size="sm" @click="copyToClipboard(finalMqttEndpoint)">
                <Copy class="w-3.5 h-3.5 mr-1" />{{ t('copy') }}
              </Button>
            </div>
          </div>
          <div v-if="otaForm.enableMqttUdp && finalUdpEndpoint">
            <label class="block text-sm text-[var(--color-text-secondary)] mb-1.5">{{ t('udp_info_label') }}</label>
            <div class="flex gap-2">
              <Input :model-value="finalUdpEndpoint" readonly class="flex-1 font-mono text-xs" />
              <Button variant="outline" size="sm" @click="copyToClipboard(finalUdpEndpoint)">
                <Copy class="w-3.5 h-3.5 mr-1" />{{ t('copy') }}
              </Button>
            </div>
          </div>
        </div>
        <div class="border-t border-[var(--color-line)] pt-4">
          <Button variant="outline" :disabled="otaTestLoading" @click="runOtaTest" class="mb-3">{{ t('ota_test') }}</Button>
          <div v-if="otaTestResult !== null">
            <label class="block text-sm text-[var(--color-text-secondary)] mb-1.5">{{ t('ota_return_label') }}</label>
            <pre class="p-3 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-muted)] text-xs leading-relaxed whitespace-pre-wrap break-all max-h-[280px] overflow-auto">{{ otaTestResult }}</pre>
          </div>
        </div>
      </template>

      <!-- Navigation -->
      <div class="flex gap-2 mt-6 pt-4 border-t border-[var(--color-line)]">
        <Button v-if="currentStep > 0 && currentStep < 5" variant="outline" @click="prevStep">{{ t('prev_step') }}</Button>
        <Button v-if="currentStep < 5" variant="outline" @click="skipStep">{{ t('skip_step') }}</Button>
        <Button v-if="currentStep >= 1 && currentStep <= 4" variant="outline" :disabled="testingStep" @click="testCurrentStepConfig">
          {{ t('test_current_config') }}
        </Button>
        <template v-if="currentStep < 5">
          <Button :disabled="saving" @click="saveAndNext">
            {{ currentStep === 4 ? t('save_and_finish') : t('save_and_next') }}
          </Button>
        </template>
        <template v-else>
          <Button @click="router.push('/dashboard')">{{ t('back_to_home') }}</Button>
          <Button variant="outline" @click="currentStep = 0">{{ t('reconfigure') }}</Button>
        </template>
      </div>
    </div>
  </div>
</template>
