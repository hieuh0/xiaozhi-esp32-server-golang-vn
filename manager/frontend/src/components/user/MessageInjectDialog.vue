<template>
  <el-dialog
    v-model="visible"
    :title="t('voice_push')"
    width="620px"
    class="inject-message-dialog"
    :close-on-click-modal="false"
    @closed="resetForm"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
    >
      <el-form-item :label="t('select_device_prompt')" prop="device_id">
        <el-select
          v-model="form.device_id"
          :placeholder="t('select_push_voice_device')"
          style="width: 100%"
          filterable
          :disabled="deviceSelectDisabled"
          popper-class="inject-device-select-popper"
        >
          <el-option
            v-for="device in devices"
            :key="device.id || device.device_code"
            :label="getDeviceOptionLabel(device)"
            :value="device.device_name || ''"
          >
            <div class="device-option">
              <div class="device-option-header">
                <span class="device-name">{{ getDeviceNickName(device) }}</span>
                <el-tag :type="isDeviceOnline(device.last_active_at) ? 'success' : 'danger'" size="small">
                  {{ isDeviceOnline(device.last_active_at) ? '在线' : '离线' }}
                </el-tag>
              </div>
              <div class="device-code">设备ID: {{ getDeviceIdText(device) }}</div>
              <div v-if="device.device_code" class="device-code">激活码: {{ device.device_code }}</div>
              <div class="device-agent">智能体: {{ device.agent_name || '未绑定' }}</div>
            </div>
          </el-option>
        </el-select>
      </el-form-item>

      <el-form-item :label="t('push_content')" prop="message">
        <el-input
          v-model="form.message"
          type="textarea"
          :rows="4"
          :placeholder="t('enter_broadcast_content')"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>

      <el-form-item :label="t('direct_broadcast')" prop="skip_llm">
        <div class="switch-field">
          <div class="switch-copy">
            <div class="switch-title">{{ directPlayback ? '开启' : t('close') }}</div>
            <div class="switch-desc">
              {{ directPlayback ? '消息将直接转语音播报，不经过 LLM 推理。' : '消息将先经过 LLM 处理，再进行播报。' }}
            </div>
          </div>
          <el-switch
            v-model="directPlayback"
            inline-prompt
            active-text="开启"
            :inactive-text="t('close')"
          />
        </div>
      </el-form-item>

      <el-form-item :label="t('switch_to_idle')" prop="auto_listen">
        <div class="switch-field">
          <div class="switch-copy">
            <div class="switch-title">{{ returnToIdleAfterPlayback ? '开启' : t('close') }}</div>
            <div class="switch-desc">
              {{ returnToIdleAfterPlayback ? '播报完成后回到空闲，适合广播通知和单向播报。' : '播报完成后继续监听，可直接进入下一轮对话。' }}
            </div>
          </div>
          <el-switch
            v-model="returnToIdleAfterPlayback"
            inline-prompt
            active-text="开启"
            :inactive-text="t('close')"
          />
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ submitting ? '推送中...' : t('voice_push') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  devices: {
    type: Array,
    default: () => []
  },
  defaultDeviceId: {
    type: String,
    default: ''
  },
  lockDevice: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const formRef = ref()
const submitting = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const directPlayback = computed({
  get: () => form.skip_llm,
  set: (value) => {
    form.skip_llm = value
  }
})

const returnToIdleAfterPlayback = computed({
  get: () => !form.auto_listen,
  set: (value) => {
    form.auto_listen = !value
  }
})

const deviceSelectDisabled = computed(() => props.lockDevice && !!props.defaultDeviceId)

const form = reactive({
  device_id: '',
  message: '',
  skip_llm: false,
  auto_listen: true
})

const rules = {
  device_id: [
    { required: true, message: t('select_device'), trigger: 'change' }
  ],
  message: [
    { required: true, message: t('enter_push_content'), trigger: 'blur' },
    { min: 1, max: 500, message: t('push_content_length'), trigger: 'blur' }
  ]
}

const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  const lastActive = new Date(lastActiveAt)
  return (Date.now() - lastActive.getTime()) < 5 * 60 * 1000
}

const getDeviceNickName = (device) => {
  const nickName = String(device?.nick_name || '').trim()
  if (nickName) return nickName
  return String(device?.device_name || '').trim() || '未命名设备'
}

const getDeviceIdText = (device) => String(device?.device_name || '').trim() || '-'

const getDeviceOptionLabel = (device) => {
  const nickName = getDeviceNickName(device)
  const deviceId = getDeviceIdText(device)
  return `${nickName} (${deviceId})`
}

const resetForm = () => {
  form.device_id = props.defaultDeviceId || ''
  form.message = ''
  form.skip_llm = false
  form.auto_listen = true
  formRef.value?.clearValidate?.()
}

watch(
  () => [props.modelValue, props.defaultDeviceId],
  ([visible]) => {
    if (!visible) return
    resetForm()
  }
)

const closeDialog = () => {
  visible.value = false
}

const handleSubmit = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    const response = await api.post('/user/devices/inject-message', {
      device_id: form.device_id,
      message: form.message,
      skip_llm: form.skip_llm,
      auto_listen: form.auto_listen
    })
    if (response.data?.success) {
      ElMessage.success(t('voice_push_success'))
      emit('success', response.data?.data || null)
      closeDialog()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '语音推送失败')
  } finally {
    submitting.value = false
  }
}

const handleClose = () => {
  resetForm()
  closeDialog()
}
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.device-option {
  padding: 8px 0;
}

.device-option-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  gap: 12px;
}

.device-name {
  font-weight: 600;
  color: var(--apple-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.device-code,
.device-agent {
  font-size: 12px;
  color: rgba(107, 114, 128, 0.72);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.switch-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
  padding: 14px 16px;
  border-radius: 18px;
  background: rgba(248, 250, 252, 0.9);
  border: 1px solid rgba(229, 229, 234, 0.72);
}

.switch-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.switch-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--apple-text);
}

.switch-desc {
  font-size: 12px;
  line-height: 1.5;
  color: var(--apple-text-secondary);
}

:deep(.inject-device-select-popper .el-select-dropdown__item) {
  height: auto;
  line-height: 1.4;
  padding-top: 8px;
  padding-bottom: 8px;
  white-space: normal;
}

@media (max-width: 768px) {
  .dialog-footer {
    flex-wrap: wrap;
  }

  .dialog-footer .el-button {
    flex: 1;
    min-width: 120px;
  }

  .switch-field {
    align-items: flex-start;
  }
}
</style>
