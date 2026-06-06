<template>
  <div class="config-wizard">
    <el-steps :active="currentStep" finish-status="success" align-center class="wizard-steps">
      <el-step title="OTA" :description="t('service_address')" />
      <el-step title="VAD" :description="t('vad')" />
      <el-step title="ASR" :description="t('asr')" />
      <el-step title="LLM" :description="t('llm')" />
      <el-step title="TTS" :description="t('tts')" />
    </el-steps>

    <el-card class="step-card" shadow="hover">
      <template v-if="currentStep === 0">
        <div class="step-title">{{ t('ota_config') }}</div>
        <p class="step-hint">{{ t('domain_ip_hint') }}</p>
        <OTAConfigForm :model="otaForm" />
      </template>

      <template v-if="currentStep === 1">
        <div class="step-title">{{ t('vad_config') }}</div>
        <VADConfigForm ref="vadFormRef" :model="vadForm" :rules="vadFormRules" class="wizard-form" />
      </template>

      <template v-if="currentStep === 2">
        <div class="step-title">{{ t('asr_config') }}</div>
        <ASRConfigForm ref="asrFormRef" :model="asrForm" :rules="asrFormRules" class="wizard-form" />
      </template>

      <template v-if="currentStep === 3">
        <div class="step-title">{{ t('llm_config') }}</div>
        <LLMConfigForm ref="llmFormRef" :model="llmForm" :rules="llmFormRules" class="wizard-form" />
      </template>

      <template v-if="currentStep === 4">
        <div class="step-title">{{ t('tts_config') }}</div>
        <TTSConfigForm
          ref="ttsFormRef"
          :model="ttsForm"
          :rules="ttsFormRules"
          :voice-options="voiceOptions"
          :voice-loading="voiceLoading"
          class="wizard-form"
          @request-voice-options="handleTtsVoiceOptionsRequest"
        />
      </template>

      <template v-if="currentStep === 5">
        <div class="step-title">{{ t('config_done_title') }}</div>
        <p class="step-hint">{{ t('config_done_hint') }}</p>
        <div class="result-box">
          <div class="result-item">
            <span class="result-label">{{ t('ota_addr_label') }}</span>
            <el-input :model-value="finalOtaUrl" readonly>
              <template #append>
                <el-button @click="copyToClipboard(finalOtaUrl)" :icon="CopyDocument">{{ t('copy') }}</el-button>
              </template>
            </el-input>
          </div>
          <div class="result-item">
            <span class="result-label">{{ t('ws_addr_label') }}</span>
            <el-input :model-value="finalWsUrl" readonly>
              <template #append>
                <el-button @click="copyToClipboard(finalWsUrl)" :icon="CopyDocument">{{ t('copy') }}</el-button>
              </template>
            </el-input>
          </div>
          <div v-if="otaForm.enableMqttUdp && finalMqttEndpoint" class="result-item">
            <span class="result-label">{{ t('mqtt_endpoint_label') }}</span>
            <el-input :model-value="finalMqttEndpoint" readonly>
              <template #append>
                <el-button @click="copyToClipboard(finalMqttEndpoint)" :icon="CopyDocument">{{ t('copy') }}</el-button>
              </template>
            </el-input>
          </div>
          <div v-if="otaForm.enableMqttUdp && finalUdpEndpoint" class="result-item">
            <span class="result-label">{{ t('udp_info_label') }}</span>
            <el-input :model-value="finalUdpEndpoint" readonly>
              <template #append>
                <el-button @click="copyToClipboard(finalUdpEndpoint)" :icon="CopyDocument">{{ t('copy') }}</el-button>
              </template>
            </el-input>
          </div>
        </div>
        <div class="ota-test-section">
          <el-button type="warning" :loading="otaTestLoading" @click="runOtaTest">{{ t('ota_test') }}</el-button>
          <div v-if="otaTestResult !== null" class="ota-test-result">
            <span class="result-label">{{ t('ota_return_label') }}</span>
            <pre class="ota-test-json">{{ otaTestResult }}</pre>
          </div>
        </div>
      </template>

      <div class="step-actions">
        <el-button v-if="currentStep > 0 && currentStep < 5" @click="prevStep">{{ t('prev_step') }}</el-button>
        <el-button v-if="currentStep < 5" type="info" plain @click="skipStep">{{ t('skip_step') }}</el-button>
        <el-button
          v-if="currentStep >= 1 && currentStep <= 4"
          type="warning"
          plain
          :loading="testingStep"
          @click="testCurrentStepConfig"
        >{{ t('test_current_config') }}</el-button>
        <template v-if="currentStep < 5">
          <el-button type="primary" :loading="saving" @click="saveAndNext">
            {{ currentStep === 4 ? t('save_and_finish') : t('save_and_next') }}
          </el-button>
        </template>
        <template v-else>
          <el-button type="primary" @click="$router.push('/dashboard')">{{ t('back_to_home') }}</el-button>
          <el-button @click="currentStep = 0">{{ t('reconfigure') }}</el-button>
        </template>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { CopyDocument } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useConfigWizard } from '@/composables/useConfigWizard'
import { useLocale } from '@/composables/useLocale'
import OTAConfigForm from './forms/OTAConfigForm.vue'
import VADConfigForm from './forms/VADConfigForm.vue'
import ASRConfigForm from './forms/ASRConfigForm.vue'
import LLMConfigForm from './forms/LLMConfigForm.vue'
import TTSConfigForm from './forms/TTSConfigForm.vue'

const { t } = useLocale()
const authStore = useAuthStore()

const {
  currentStep, saving, testingStep, otaTestLoading, otaTestResult,
  otaForm, vadForm, vadFormRef, vadFormRules, asrForm, asrFormRef, asrFormRules,
  llmForm, llmFormRef, llmFormRules, ttsForm, ttsFormRef, ttsFormRules,
  voiceOptions, voiceLoading,
  finalOtaUrl, finalWsUrl, finalMqttEndpoint, finalUdpEndpoint,
  saveAndNext, skipStep, prevStep, testCurrentStepConfig, runOtaTest,
  handleTtsVoiceOptionsRequest, copyToClipboard, initialize
} = useConfigWizard()

onMounted(async () => {
  if (authStore.isAdmin) localStorage.setItem('admin_first_login_done', '1')
  await initialize()
})
</script>

<style scoped>
.config-wizard { padding: 20px; max-width: 820px; margin: 0 auto; }
.wizard-steps { margin-bottom: 24px; }
.step-card { padding: 24px; }
.step-title { font-size: 16px; font-weight: 600; margin-bottom: 8px; color: #303133; }
.step-hint { color: #909399; font-size: 13px; margin-bottom: 20px; }
.wizard-form { margin-bottom: 24px; }
.step-actions { display: flex; gap: 12px; margin-top: 24px; padding-top: 16px; border-top: 1px solid #ebeef5; }
.result-box { margin: 16px 0; }
.result-item { margin-bottom: 16px; }
.result-label { display: block; font-size: 13px; color: #606266; margin-bottom: 6px; }
.ota-test-section { margin-top: 24px; padding-top: 16px; border-top: 1px solid #ebeef5; }
.ota-test-result { margin-top: 12px; }
.ota-test-result .result-label { margin-bottom: 6px; }
.ota-test-json { margin: 0; padding: 12px; background: rgba(248,250,252,0.92); border: 1px solid rgba(229,229,234,0.72); border-radius: 12px; font-size: 12px; line-height: 1.5; white-space: pre-wrap; word-break: break-all; max-height: 280px; overflow: auto; }
</style>
