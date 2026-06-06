<template>
  <el-dialog
    v-model="visible"
    :title="mode === 'add' ? t('create_voiceprint_group') : t('edit_voiceprint_group')"
    width="600px"
  >
    <el-form ref="formRef" :model="groupForm" :rules="groupRules" label-width="100px">
      <el-form-item :label="t('link_agent')" prop="agent_id">
        <el-select v-model="groupForm.agent_id" :placeholder="t('select_agent')" style="width: 100%">
          <el-option v-for="agent in agents" :key="agent.id" :label="agent.name" :value="agent.id" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('voiceprint_name')" prop="name">
        <el-input v-model="groupForm.name" :placeholder="t('enter_voiceprint_name')" :maxlength="100" show-word-limit />
      </el-form-item>
      <el-form-item label="Prompt" prop="prompt">
        <el-input v-model="groupForm.prompt" type="textarea" :rows="4" :placeholder="t('role_prompt_ph')" />
      </el-form-item>
      <el-form-item :label="t('description')" prop="description">
        <el-input v-model="groupForm.description" type="textarea" :rows="3" :placeholder="t('desc_optional_ph')" :maxlength="200" show-word-limit />
      </el-form-item>
      <el-form-item :label="t('my_cloned_voice')" v-if="cloneVoicePresets.length > 0">
        <div class="clone-voice-line" v-loading="cloneVoicesLoading">
          <button
            v-for="clone in cloneVoicePresets"
            :key="clone.id"
            type="button"
            class="clone-voice-item"
            :class="{ active: isCloneVoiceSelected(clone) }"
            :title="`${clone.tts_config_name || clone.tts_config_id} · ${clone.provider_voice_id}`"
            @click="emit('apply-clone-voice', clone)"
          >
            <span class="clone-voice-name">{{ clone.name || clone.provider_voice_id }}</span>
          </button>
        </div>
        <div class="form-help">{{ t('click_auto_fill') }}</div>
      </el-form-item>
      <el-form-item :label="t('tts_config_label')" prop="tts_config_id">
        <el-select
          v-model="groupForm.tts_config_id"
          :placeholder="t('select_tts_config_opt')"
          clearable
          style="width: 100%"
          @change="(v) => emit('tts-config-change', v)"
        >
          <el-option
            v-for="ttsConfig in ttsConfigs"
            :key="ttsConfig.config_id"
            :label="ttsConfig.is_default ? t('tts_default_label', { name: ttsConfig.name }) : ttsConfig.name"
            :value="ttsConfig.config_id"
          >
            <div class="config-option">
              {{ ttsConfig.name }}
              <el-tag v-if="ttsConfig.is_default" type="success" size="small" style="margin-left: 8px;">{{ t('default') }}</el-tag>
            </div>
            <span class="config-desc">{{ ttsConfig.provider || t('no_description_alt') }}</span>
          </el-option>
        </el-select>
        <div class="form-help" v-if="groupForm.tts_config_id">{{ currentTtsConfigInfo }}</div>
      </el-form-item>
      <el-form-item :label="t('voice_timbre')" prop="voice" v-if="groupForm.tts_config_id">
        <el-select v-model="groupForm.voice" :placeholder="t('select_or_enter_voice')" filterable allow-create clearable style="width: 100%">
          <el-option v-for="voice in currentVoiceOptions" :key="voice.value" :label="voice.label" :value="voice.value" />
        </el-select>
        <div class="form-help">{{ t('current_tts_config_hint', { name: currentTtsConfigName }) }}</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">{{ t('cancel') }}</el-button>
      <el-button type="primary" @click="submit" :loading="submitting">
        {{ mode === 'add' ? t('create') : t('save') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref } from 'vue'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

const props = defineProps({
  mode: { type: String, default: 'add' },
  groupForm: { type: Object, required: true },
  groupRules: { type: Object, required: true },
  agents: { type: Array, default: () => [] },
  ttsConfigs: { type: Array, default: () => [] },
  currentVoiceOptions: { type: Array, default: () => [] },
  cloneVoicePresets: { type: Array, default: () => [] },
  cloneVoicesLoading: { type: Boolean, default: false },
  submitting: { type: Boolean, default: false },
  currentTtsConfigName: { type: String, default: '' },
  currentTtsConfigInfo: { type: String, default: '' },
  isCloneVoiceSelected: { type: Function, required: true }
})

const visible = defineModel({ default: false })
const emit = defineEmits(['submit', 'apply-clone-voice', 'tts-config-change'])

const formRef = ref()

// Expose formRef so parent can reset it
defineExpose({ formRef })

const submit = async () => {
  emit('submit', formRef)
}
</script>

<style scoped>
.clone-voice-line { display: flex; flex-wrap: wrap; gap: 6px; width: 100%; }
.clone-voice-item {
  display: inline-flex; align-items: center; max-width: 220px; min-width: 0;
  padding: 4px 10px; border: 1px solid #d1d5db; border-radius: 999px;
  background: #f8fafc; color: #374151; cursor: pointer; transition: all 0.2s ease;
  line-height: 1.2; outline: none;
}
.clone-voice-item:hover { border-color: #93c5fd; background: #f1f7ff; }
.clone-voice-item.active { border-color: #3b82f6; background: #e9f2ff; color: #1d4ed8; box-shadow: 0 0 0 1px rgba(59,130,246,0.1); }
.clone-voice-name { font-size: 12px; font-weight: 500; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.form-help { color: #909399; font-size: 12px; margin-top: 4px; }
.config-option { display: flex; align-items: center; }
.config-desc { font-size: 12px; color: #909399; }
</style>
