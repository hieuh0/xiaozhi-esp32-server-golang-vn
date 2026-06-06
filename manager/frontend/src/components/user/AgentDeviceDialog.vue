<template>
  <!-- Bind device dialog -->
  <el-dialog
    v-model="showAddDeviceDialog"
    :title="t('bind_device')"
    width="520px"
    :close-on-click-modal="false"
    @closed="emit('reset-form')"
  >
    <DeviceForm
      ref="deviceFormRef"
      :model-value="deviceForm"
      @update:modelValue="emit('update:deviceForm', $event)"
      mode="bind"
      :agents="agents"
      :fixed-agent-id="fixedAgentId"
    />
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="showAddDeviceDialog = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="addingDevice" @click="emit('add-device')">
          {{ addingDevice ? t('binding') : t('bind_device') }}
        </el-button>
      </div>
    </template>
  </el-dialog>

  <!-- MCP tools dialog -->
  <el-dialog v-model="showMcpDialog" :title="t('device_mcp_tools')" width="760px">
    <div v-loading="mcpLoading">
      <div class="mcp-tools-header">
        <el-button size="small" type="primary" @click="emit('refresh-mcp')" :loading="toolsLoading">
          {{ t('refresh_tool_list') }}
        </el-button>
      </div>
      <div v-if="mcpTools.length === 0" class="tools-empty">{{ t('no_tool_data') }}</div>
      <div v-else class="tools-tags">
        <el-tag v-for="tool in mcpTools" :key="tool.name" class="tool-tag">{{ tool.name }}</el-tag>
      </div>
      <el-divider />
      <el-form :model="mcpCallForm" label-width="90px">
        <el-form-item :label="t('tool')">
          <el-select v-model="mcpCallForm.tool_name" :placeholder="t('select_tool')" style="width:100%" @change="(v) => emit('mcp-tool-change', v)">
            <el-option v-for="tool in mcpTools" :key="tool.name" :label="tool.name" :value="tool.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('args_json_label')">
          <el-input v-model="mcpCallForm.argumentsText" type="textarea" :rows="6" :placeholder="t('mcp_args_placeholder')" />
        </el-form-item>
      </el-form>
      <el-button type="primary" @click="emit('call-mcp-tool')" :loading="callingTool">{{ t('call_tool') }}</el-button>
      <el-divider />
      <div class="mcp-result-box">{{ mcpCallResult || t('no_call_results') }}</div>
    </div>
  </el-dialog>

  <!-- Role config dialog -->
  <el-dialog v-model="showRoleConfigDialog" :title="t('device_role_config_title')" width="700px" @close="emit('close-role-config')">
    <div v-loading="roleConfigLoading">
      <div class="role-config-content">
        <el-alert :title="t('role_config_note')" type="info" :closable="false" style="margin-bottom: 16px">
          {{ t('role_override_desc') }}
        </el-alert>
        <el-form label-width="120px">
          <el-form-item :label="t('current_role_label')">
            <div v-if="currentDevice.role_id">
              <el-tag type="success" size="large">{{ t('role_linked') }}</el-tag>
              <div class="current-role-info">
                <p><strong>{{ t('role_id_label') }}</strong> {{ currentDevice.role_id }}</p>
              </div>
            </div>
            <el-tag v-else type="info" size="large">{{ t('role_not_linked') }}</el-tag>
          </el-form-item>
          <el-form-item :label="t('select_role_label')">
            <el-select
              v-model="selectedRoleId"
              :placeholder="t('select_role_opt_ph')"
              style="width: 100%"
              clearable
              filterable
              @change="(v) => emit('role-select', v)"
            >
              <el-option v-for="role in availableRoles" :key="role.id" :label="role.name" :value="role.id">
                <div class="role-option-item">
                  <div class="role-option-main">
                    <span>{{ role.name }}</span>
                    <el-tag v-if="role.role_type === 'global'" size="small" type="success">{{ t('global') }}</el-tag>
                  </div>
                  <el-tag size="small" type="info">LLM: {{ role.llm_config_id || t('default') }}</el-tag>
                </div>
              </el-option>
            </el-select>
            <div class="form-help">{{ t('role_override_help') }}</div>
          </el-form-item>
          <el-form-item :label="t('role_detail_label')" v-if="selectedRole">
            <el-card class="role-preview-card">
              <div class="role-preview-content">
                <p><strong>{{ t('name_colon') }}</strong> {{ selectedRole.name }}</p>
                <p v-if="selectedRole.description"><strong>{{ t('desc_colon') }}</strong> {{ selectedRole.description }}</p>
                <el-divider />
                <p><strong>Prompt:</strong></p>
                <p class="prompt-preview">{{ selectedRole.prompt.substring(0, 200) }}{{ selectedRole.prompt.length > 200 ? '...' : '' }}</p>
                <div class="role-configs-preview">
                  <el-tag size="small">LLM: {{ selectedRole.llm_config_id || t('default') }}</el-tag>
                  <el-tag size="small">TTS: {{ selectedRole.tts_config_id || t('default') }}</el-tag>
                  <el-tag v-if="selectedRole.voice" size="small">{{ t('voice_colon') }}{{ selectedRole.voice }}</el-tag>
                </div>
              </div>
            </el-card>
          </el-form-item>
        </el-form>
      </div>
    </div>
    <template #footer>
      <el-button @click="emit('close-role-config')">{{ t('cancel') }}</el-button>
      <el-button
        type="primary"
        @click="emit('apply-role')"
        :loading="roleConfigLoading"
        :disabled="!selectedRoleId && !currentDevice.role_id"
      >
        {{ selectedRoleId ? t('apply_role') : t('cancel_role') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref } from 'vue'
import DeviceForm from '../common/DeviceForm.vue'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

defineProps({
  agents: { type: Array, default: () => [] },
  deviceForm: { type: Object, required: true },
  fixedAgentId: { type: String, default: null },
  addingDevice: { type: Boolean, default: false },
  mcpLoading: { type: Boolean, default: false },
  toolsLoading: { type: Boolean, default: false },
  callingTool: { type: Boolean, default: false },
  mcpTools: { type: Array, default: () => [] },
  mcpCallResult: { type: String, default: '' },
  mcpCallForm: { type: Object, required: true },
  roleConfigLoading: { type: Boolean, default: false },
  currentDevice: { type: Object, default: () => ({}) },
  selectedRole: { type: Object, default: null },
  availableRoles: { type: Array, default: () => [] }
})

const showAddDeviceDialog = defineModel('showAddDeviceDialog', { default: false })
const showMcpDialog = defineModel('showMcpDialog', { default: false })
const showRoleConfigDialog = defineModel('showRoleConfigDialog', { default: false })
const selectedRoleId = defineModel('selectedRoleId', { default: null })

const deviceFormRef = ref()
defineExpose({ deviceFormRef })

const emit = defineEmits([
  'reset-form', 'add-device', 'refresh-mcp', 'mcp-tool-change', 'call-mcp-tool',
  'role-select', 'apply-role', 'close-role-config', 'update:deviceForm'
])
</script>

<style scoped>
.dialog-footer { display: flex; justify-content: center; gap: 12px; }
.dialog-footer .el-button { min-width: 80px; }
.mcp-tools-header { margin-bottom: 12px; }
.tools-tags { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
.tools-empty { color: var(--apple-text-secondary); margin: 8px 0 16px; }
.mcp-result-box { white-space: pre-wrap; font-family: monospace; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 10px; min-height: 80px; }
.role-config-content { padding: 20px 0; }
.current-role-info { margin-bottom: 16px; }
.current-role-info p { margin: 4px 0; color: var(--apple-text-secondary); }
.role-option-item { display: flex; align-items: flex-start; justify-content: space-between; flex-direction: column; gap: 6px; padding: 8px 12px; border-radius: 6px; margin-bottom: 8px; }
.role-option-main { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.role-preview-card { background: rgba(248,250,252,0.92); border: 1px solid rgba(229,229,234,0.72); }
.role-preview-content { font-size: 14px; }
.role-preview-content p { margin: 8px 0; }
.role-preview-content strong { color: var(--apple-text); margin-right: 8px; }
.role-configs-preview { display: flex; gap: 8px; flex-wrap: wrap; }
.prompt-preview { background: rgba(248,250,252,0.92); border: 1px solid rgba(229,229,234,0.72); padding: 12px; border-radius: 14px; font-size: 13px; color: var(--apple-text-secondary); line-height: 1.6; }
.form-help { color: #909399; font-size: 12px; margin-top: 4px; }
</style>
