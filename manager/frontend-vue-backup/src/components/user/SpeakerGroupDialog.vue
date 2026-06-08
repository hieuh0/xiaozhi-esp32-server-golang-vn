<script setup>
import { ref } from 'vue'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

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
defineExpose({ formRef })

const submit = () => emit('submit', formRef)
</script>

<template>
  <Dialog v-model:open="visible">
    <DialogContent class="max-w-[580px] max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ mode === 'add' ? t('create_voiceprint_group') : t('edit_voiceprint_group') }}</DialogTitle>
      </DialogHeader>

      <div class="grid gap-4 py-2">
        <!-- Agent select -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('link_agent') }}</label>
          <Select v-model="groupForm.agent_id">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="t('select_agent')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="agent in agents" :key="agent.id" :value="agent.id">{{ agent.name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Name -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('voiceprint_name') }}</label>
          <Input v-model="groupForm.name" :placeholder="t('enter_voiceprint_name')" maxlength="100" />
        </div>

        <!-- Prompt -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">Prompt</label>
          <Textarea v-model="groupForm.prompt" :placeholder="t('role_prompt_ph')" rows="4" />
        </div>

        <!-- Description -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('description') }}</label>
          <Textarea v-model="groupForm.description" :placeholder="t('desc_optional_ph')" rows="3" maxlength="200" />
        </div>

        <!-- Clone voice presets -->
        <div v-if="cloneVoicePresets.length > 0" class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('my_cloned_voice') }}</label>
          <div :class="['flex flex-wrap gap-1.5', cloneVoicesLoading && 'opacity-50']">
            <button
              v-for="clone in cloneVoicePresets"
              :key="clone.id"
              type="button"
              :class="['inline-flex items-center px-3 py-1 rounded-full text-xs border transition-colors cursor-pointer',
                isCloneVoiceSelected(clone)
                  ? 'border-[var(--color-primary)] bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                  : 'border-[var(--color-line)] bg-[var(--color-surface)] text-[var(--color-text-secondary)] hover:border-[var(--color-primary)]']"
              :title="`${clone.tts_config_name || clone.tts_config_id} · ${clone.provider_voice_id}`"
              @click="emit('apply-clone-voice', clone)"
            >{{ clone.name || clone.provider_voice_id }}</button>
          </div>
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('click_auto_fill') }}</p>
        </div>

        <!-- TTS config -->
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('tts_config_label') }}</label>
          <Select v-model="groupForm.tts_config_id" @update:model-value="(v) => emit('tts-config-change', v)">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="t('select_tts_config_opt')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="cfg in ttsConfigs" :key="cfg.config_id" :value="cfg.config_id">
                {{ cfg.is_default ? t('tts_default_label', { name: cfg.name }) : cfg.name }}
              </SelectItem>
            </SelectContent>
          </Select>
          <p v-if="groupForm.tts_config_id && currentTtsConfigInfo" class="text-xs text-[var(--color-text-tertiary)]">{{ currentTtsConfigInfo }}</p>
        </div>

        <!-- Voice (allow-create via datalist) -->
        <div v-if="groupForm.tts_config_id" class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('voice_timbre') }}</label>
          <Input v-model="groupForm.voice" list="speaker-group-voice-datalist" :placeholder="t('select_or_enter_voice')" />
          <datalist id="speaker-group-voice-datalist">
            <option v-for="v in currentVoiceOptions" :key="v.value" :value="v.value">{{ v.label }}</option>
          </datalist>
          <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('current_tts_config_hint', { name: currentTtsConfigName }) }}</p>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="visible = false">{{ t('cancel') }}</Button>
        <Button :disabled="submitting" @click="submit">
          {{ mode === 'add' ? t('create') : t('save') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
