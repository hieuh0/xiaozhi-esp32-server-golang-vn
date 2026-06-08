<template>
  <div>
    <AgentDeviceGrid
      v-model:filterAgentId="filterAgentId"
      v-model:editingDeviceName="editingDeviceName"
      :agents="agents"
      :filtered-devices="filteredDevices"
      :empty-description="emptyDescription"
      :show-back-button="showBackButton"
      :editing-device-id="editingDeviceId"
      :editing-device-name="editingDeviceName"
      :renaming-device-id="renamingDeviceId"
      :device-name-input-ref="deviceNameInputRef"
      :get-device-display-name="getDeviceDisplayName"
      :get-device-identity-text="getDeviceIdentityText"
      :get-device-agent-name="getDeviceAgentName"
      :is-device-online="isDeviceOnline"
      :format-date="formatDate"
      @go-back="goBack"
      @agent-filter-change="handleAgentFilterChange"
      @add-device="openAddDeviceDialog"
      @start-name-edit="startDeviceNameEdit"
      @cancel-name-edit="cancelDeviceNameEdit"
      @save-name="saveDeviceName"
      @device-role="handleDeviceRole"
      @device-mcp="handleDeviceMcp"
      @voice-push="handleVoicePush"
      @delete-device="handleDeleteDevice"
    />

    <AgentDeviceDialog
      v-model:showAddDeviceDialog="showAddDeviceDialog"
      v-model:showMcpDialog="showMcpDialog"
      ref="dialogRef"
      :agents="agents"
      v-model:deviceForm="deviceForm"
      :fixed-agent-id="bindingAgentId"
      :adding-device="addingDevice"
      :mcp-loading="mcpLoading"
      :tools-loading="toolsLoading"
      :calling-tool="callingTool"
      :mcp-tools="mcpTools"
      :mcp-call-result="mcpCallResult"
      :mcp-call-form="mcpCallForm"
      v-model:showRoleConfigDialog="showRoleConfigDialog"
      :role-config-loading="roleConfigLoading"
      :current-device="currentDevice"
      v-model:selectedRoleId="selectedRoleId"
      :selected-role="selectedRole"
      :available-roles="availableRoles"
      @reset-form="resetAddDeviceForm"
      @add-device="handleAddDevice"
      @refresh-mcp="refreshDeviceMcpTools"
      @mcp-tool-change="handleMcpToolChange"
      @call-mcp-tool="callDeviceMcpTool"
      @role-select="handleRoleSelect"
      @apply-role="handleApplyRole"
      @close-role-config="handleCloseRoleConfig"
    />

    <MessageInjectDialog
      v-model="showVoicePushDialog"
      :devices="devices"
      :default-device-id="voicePushDeviceId"
      :lock-device="Boolean(voicePushDeviceId)"
      @success="handleVoicePushSuccess"
    />
  </div>
</template>

<script setup>
import { onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAgentDevices } from '../../composables/useAgentDevices'
import AgentDeviceGrid from '../../components/user/AgentDeviceGrid.vue'
import AgentDeviceDialog from '../../components/user/AgentDeviceDialog.vue'
import MessageInjectDialog from '../../components/user/MessageInjectDialog.vue'

const route = useRoute()

const {
  showBackButton, filterAgentId, bindingAgentId,
  agents, devices, showAddDeviceDialog, addingDevice, deviceFormRef, deviceForm,
  showVoicePushDialog, voicePushDeviceId,
  editingDeviceId, editingDeviceName, renamingDeviceId, deviceNameInputRef,
  showMcpDialog, mcpLoading, toolsLoading, callingTool, mcpTools, mcpCallResult, mcpCallForm,
  showRoleConfigDialog, roleConfigLoading, currentDevice, selectedRoleId, selectedRole, availableRoles,
  filteredDevices, emptyDescription,
  loadAgents, loadDevices, loadRoles,
  resetAddDeviceForm, openAddDeviceDialog, handleAddDevice,
  handleVoicePush, handleVoicePushSuccess, handleAgentFilterChange,
  getDeviceAgentName, getDeviceDisplayName, getDeviceIdentityText,
  startDeviceNameEdit, cancelDeviceNameEdit, saveDeviceName,
  handleDeviceMcp, refreshDeviceMcpTools, handleMcpToolChange, callDeviceMcpTool,
  handleDeviceRole, handleRoleSelect, handleApplyRole, handleCloseRoleConfig,
  handleDeleteDevice, goBack, formatDate, isDeviceOnline
} = useAgentDevices()

onMounted(() => {
  loadAgents()
  loadDevices()
  loadRoles()
})

watch(
  () => [route.params.id, route.query.agent_id],
  () => { filterAgentId.value = String(route.params.id || route.query.agent_id || '') }
)
</script>
