<template>
  <div>
    <!-- Filter bar -->
    <div class="devices-filter-bar">
      <div class="filter-controls">
        <el-button v-if="showBackButton" @click="emit('go-back')" type="text" class="back-btn">
          <el-icon><ArrowLeft /></el-icon>
          {{ t('back') }}
        </el-button>
        <el-select
          v-model="filterAgentId"
          :placeholder="t('filter_by_agent')"
          clearable
          filterable
          class="agent-filter-select"
          @change="(v) => emit('agent-filter-change', v)"
        >
          <el-option :label="t('all_devices')" value="" />
          <el-option
            v-for="agent in agents"
            :key="agent.id"
            :label="agent.name"
            :value="String(agent.id)"
          />
        </el-select>
        <span class="devices-count">{{ t('device_count_n', { count: filteredDevices.length }) }}</span>
      </div>
      <el-button class="add-device-button" type="primary" @click="emit('add-device')">
        <el-icon><Plus /></el-icon>
        {{ t('add_device') }}
      </el-button>
    </div>

    <!-- Empty state -->
    <div v-if="filteredDevices.length === 0" class="empty-section">
      <el-card class="empty-card">
        <div class="empty-content">
          <el-icon size="64" color="var(--apple-text-tertiary)"><Monitor /></el-icon>
          <h3>{{ t('no_devices') }}</h3>
          <p>{{ emptyDescription }}</p>
          <div class="empty-actions">
            <el-button type="primary" size="large" @click="emit('add-device')">
              <el-icon><Plus /></el-icon>
              {{ t('add_first_device') }}
            </el-button>
          </div>
        </div>
      </el-card>
    </div>

    <!-- Device grid -->
    <div v-else class="devices-grid">
      <div v-for="device in filteredDevices" :key="device.id" class="device-item">
        <div class="device-card">
          <div class="device-header">
            <div class="device-icon">
              <el-icon size="28"><Monitor /></el-icon>
            </div>
            <div class="device-info">
              <div class="device-name-row">
                <el-input
                  v-if="editingDeviceId === device.id"
                  ref="deviceNameInputRef"
                  v-model="editingDeviceName"
                  class="device-name-input"
                  size="small"
                  maxlength="50"
                  show-word-limit
                  :placeholder="t('enter_device_nickname')"
                  @keydown.enter.prevent="emit('save-name', device)"
                  @keydown.esc.prevent="emit('cancel-name-edit')"
                />
                <button
                  v-else
                  type="button"
                  class="device-name-button"
                  :title="t('click_edit_nickname', { name: getDeviceDisplayName(device) })"
                  @click="emit('start-name-edit', device)"
                >
                  <span class="device-name">{{ getDeviceDisplayName(device) }}</span>
                </button>
                <div class="device-name-actions">
                  <template v-if="editingDeviceId === device.id">
                    <el-button
                      class="name-action-button" type="primary" :icon="Check" circle
                      :loading="renamingDeviceId === device.id" :title="t('save_nickname')"
                      @click="emit('save-name', device)"
                    />
                    <el-button
                      class="name-action-button" :icon="Close" circle
                      :title="t('cancel_edit')" @click="emit('cancel-name-edit')"
                    />
                  </template>
                  <el-button
                    v-else class="rename-icon-button" :icon="EditPen" circle
                    :title="t('edit_nickname_title')" @click="emit('start-name-edit', device)"
                  />
                </div>
              </div>
              <p class="device-identity" :title="getDeviceIdentityText(device)">{{ getDeviceIdentityText(device) }}</p>
            </div>
            <div class="device-status">
              <span :class="['status-dot', isDeviceOnline(device.last_active_at) ? 'online' : 'offline']"></span>
              <span class="status-text">{{ isDeviceOnline(device.last_active_at) ? t('online') : t('offline') }}</span>
            </div>
          </div>

          <div class="device-meta">
            <div class="meta-row">
              <span class="meta-label">{{ t('link_agent') }}</span>
              <span class="meta-value">{{ getDeviceAgentName(device) }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('device_type_label') }}</span>
              <span class="meta-value">{{ t('esp32_device') }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('activation_status') }}</span>
              <span class="meta-value">
                <el-tag :type="device.activated ? 'success' : 'warning'" size="small">
                  {{ device.activated ? t('activated') : t('not_activated') }}
                </el-tag>
              </span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('last_active') }}</span>
              <span class="meta-value">{{ formatDate(device.last_active_at) }}</span>
            </div>
            <div class="meta-row">
              <span class="meta-label">{{ t('created_at') }}</span>
              <span class="meta-value">{{ formatDate(device.created_at) }}</span>
            </div>
          </div>

          <div class="device-actions">
            <el-button class="device-action-button device-action-button-feature" size="small" @click="emit('device-role', device.id)">
              <el-icon><User /></el-icon>{{ t('role') }}
            </el-button>
            <el-button class="device-action-button" size="small" @click="emit('device-mcp', device)">
              <el-icon><Setting /></el-icon>MCP
            </el-button>
            <el-button class="device-action-button device-action-button-voice" size="small" @click="emit('voice-push', device)">
              <el-icon><ChatDotRound /></el-icon>{{ t('voice_notify') }}
            </el-button>
            <el-button class="device-action-button device-action-button-danger" size="small" @click="emit('delete-device', device)">
              <el-icon><Delete /></el-icon>{{ t('delete') }}
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ArrowLeft, Plus, Monitor, Setting, Delete, User, ChatDotRound, EditPen, Check, Close } from '@element-plus/icons-vue'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()

defineProps({
  agents: { type: Array, default: () => [] },
  filteredDevices: { type: Array, default: () => [] },
  emptyDescription: { type: String, default: '' },
  showBackButton: { type: Boolean, default: false },
  editingDeviceId: { type: [String, Number], default: null },
  renamingDeviceId: { type: [String, Number], default: null },
  deviceNameInputRef: { type: Object, default: null },
  getDeviceDisplayName: { type: Function, required: true },
  getDeviceIdentityText: { type: Function, required: true },
  getDeviceAgentName: { type: Function, required: true },
  isDeviceOnline: { type: Function, required: true },
  formatDate: { type: Function, required: true }
})

const filterAgentId = defineModel('filterAgentId', { default: '' })
const editingDeviceName = defineModel('editingDeviceName', { default: '' })

const emit = defineEmits([
  'go-back', 'agent-filter-change', 'add-device',
  'start-name-edit', 'cancel-name-edit', 'save-name',
  'device-role', 'device-mcp', 'voice-push', 'delete-device'
])
</script>

<style scoped>
.back-btn { padding: 8px; color: var(--apple-primary); }
.devices-filter-bar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 18px; padding: 14px 16px; border-radius: 20px; background: rgba(255,255,255,0.88); border: 1px solid rgba(255,255,255,0.9); box-shadow: var(--apple-shadow-sm); }
.filter-controls { min-width: 0; display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.agent-filter-select { width: 240px; }
.devices-count { color: var(--apple-text-secondary); font-size: 13px; font-weight: 600; }
.add-device-button { flex: none; margin-left: auto; }
.empty-section { margin-top: 40px; }
.empty-card { text-align: center; padding: 40px 20px; }
.empty-content h3 { margin: 20px 0 10px 0; color: var(--apple-text); }
.empty-content p { color: var(--apple-text-secondary); margin-bottom: 30px; }
.devices-grid { margin-top: 20px; display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 340px)); gap: 16px; justify-content: flex-start; }
.device-item { min-width: 0; }
.device-card { background: rgba(255,255,255,0.88); border-radius: 20px; padding: 16px; border: 1px solid rgba(229,229,234,0.72); box-shadow: var(--apple-shadow-md); transition: all 0.3s ease; height: 100%; display: flex; flex-direction: column; width: 100%; min-width: 0; max-width: 340px; }
.device-card:hover { transform: translateY(-2px); box-shadow: var(--apple-shadow-sm); border-color: rgba(0,122,255,0.18); }
.device-header { display: flex; align-items: flex-start; gap: 12px; margin-bottom: 16px; }
.device-icon { width: 42px; height: 42px; background: linear-gradient(180deg,#2e90ff 0%,#007aff 100%); border-radius: 12px; display: flex; align-items: center; justify-content: center; color: white; flex-shrink: 0; }
.device-info { flex: 1; min-width: 0; }
.device-name-row { display: grid; grid-template-columns: minmax(0,1fr) auto; align-items: center; gap: 8px; margin-bottom: 4px; min-width: 0; }
.device-name { margin: 0; font-size: 16px; font-weight: 700; color: var(--apple-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.device-name-button { display: inline-flex; align-items: center; min-width: 0; max-width: 100%; margin: 0; padding: 0; border: 0; background: transparent; text-align: left; cursor: text; }
.device-name-button:hover .device-name, .device-name-button:focus-visible .device-name { color: var(--apple-primary); }
.device-name-input { min-width: 0; }
.device-name-actions { display: inline-flex; align-items: center; flex-shrink: 0; gap: 2px; }
.device-name-actions :deep(.el-button) { min-height: auto; width: 26px; height: 26px; padding: 0; margin: 0; font-size: 12px; border-radius: 9px; }
.rename-icon-button { opacity: 0.28; color: var(--apple-text-tertiary); border-color: rgba(229,229,234,0.78); background: rgba(255,255,255,0.7); transition: opacity 0.2s ease, color 0.2s ease, border-color 0.2s ease, transform 0.2s ease; }
.device-name-row:hover .rename-icon-button, .device-card:hover .rename-icon-button, .rename-icon-button:focus-visible { opacity: 1; color: var(--apple-primary); border-color: rgba(0,122,255,0.28); transform: translateY(-1px); }
.name-action-button { box-shadow: none; }
.device-identity { margin: 0; font-size: 11px; color: rgba(107,114,128,0.74); font-family: monospace; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.device-status { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--apple-line-strong); }
.status-dot.online { background: var(--apple-success); }
.status-dot.offline { background: var(--apple-danger); }
.status-text { font-size: 12px; color: var(--apple-text-secondary); }
.device-meta { flex: 1; margin-bottom: 16px; }
.meta-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.meta-row:last-child { margin-bottom: 0; }
.meta-label { font-size: 12px; color: var(--apple-text-secondary); }
.meta-value { font-size: 12px; color: var(--apple-text); font-weight: 500; }
.device-actions { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 8px; margin-top: auto; }
.device-actions .el-button { min-width: 0; width: 100%; min-height: 34px; margin-left: 0; padding: 0 10px; justify-content: center; border-radius: 12px; border: 1px solid rgba(214,219,228,0.9); background: rgba(248,250,252,0.92); color: #4b5563; box-shadow: none; font-size: 12px; font-weight: 600; }
.device-actions .el-button + .el-button { margin-left: 0; }
.device-actions :deep(.el-button > span) { min-width: 0; width: 100%; display: inline-flex; align-items: center; justify-content: center; gap: 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.device-actions :deep(.el-icon) { flex: none; font-size: 13px; }
.device-action-button-feature { border-color: rgba(147,197,253,0.85); background: linear-gradient(180deg,rgba(239,246,255,0.98) 0%,rgba(219,234,254,0.9) 100%); color: #1d4ed8; }
.device-action-button-voice { border-color: rgba(153,246,228,0.95); background: rgba(240,253,250,0.96); color: #0f766e; }
.device-action-button-danger { border-color: rgba(244,191,191,0.95); background: rgba(255,245,245,0.96); color: #b42318; }
.device-actions .el-button:hover, .device-actions .el-button:focus { border-color: rgba(148,163,184,0.82); background: rgba(241,245,249,0.98); color: #334155; }
.device-action-button-feature:hover, .device-action-button-feature:focus { border-color: rgba(96,165,250,0.95); background: linear-gradient(180deg,rgba(219,234,254,0.98) 0%,rgba(191,219,254,0.92) 100%); color: #1e40af; }
.device-action-button-voice:hover, .device-action-button-voice:focus { color: #075985; background: rgba(236,254,255,0.98); border-color: rgba(34,211,238,0.58); }
.device-action-button-danger:hover, .device-action-button-danger:focus { border-color: rgba(248,113,113,0.78); background: rgba(254,242,242,0.98); color: #991b1b; }
@media (max-width: 768px) {
  .devices-filter-bar { align-items: stretch; flex-direction: column; }
  .filter-controls { align-items: stretch; flex-direction: column; }
  .agent-filter-select { width: 100%; }
  .devices-count { align-self: flex-start; }
  .add-device-button { width: 100%; margin-left: 0; }
  .devices-grid { grid-template-columns: 1fr; gap: 12px; }
  .rename-icon-button { opacity: 1; }
}
@media (max-width: 560px) {
  .devices-grid { gap: 10px; }
  .device-actions { grid-template-columns: 1fr; }
}
@media (min-width: 769px) and (max-width: 1180px) {
  .devices-grid { grid-template-columns: repeat(auto-fill,minmax(280px,340px)); }
}
</style>
