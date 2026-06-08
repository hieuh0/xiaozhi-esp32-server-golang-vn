<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { buildDevicePayload, useAgentFormOptions } from '../../composables/useAgentFormOptions'
import { useLocale } from '../../composables/useLocale'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'

const { t } = useLocale()

const props = defineProps({
  modelValue: { type: Object, required: true },
  isAdmin: { type: Boolean, default: false },
  mode: { type: String, default: 'create' },
  fixedAgentId: { type: [Number, String, null], default: null },
  agents: { type: Array, default: () => [] },
  labelPosition: { type: String, default: 'top' },
  labelWidth: { type: String, default: '110px' }
})

const emit = defineEmits(['update:modelValue'])

const form = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const targetUserId = computed(() => props.isAdmin ? Number(form.value.user_id || 0) : 0)
const isBindMode = computed(() => props.mode === 'bind')
const hasFixedAgent = computed(() => props.fixedAgentId !== null && props.fixedAgentId !== undefined && props.fixedAgentId !== '')

const { users, agents, loading, loadUsers, loadAgents } = useAgentFormOptions({
  isAdmin: computed(() => props.isAdmin),
  targetUserId
})

const displayAgents = computed(() => {
  const source = props.agents.length ? props.agents : agents.value
  if (props.isAdmin && targetUserId.value) return source.filter((a) => Number(a.user_id) === targetUserId.value)
  return source
})

const agentLabel = (agent) => {
  if (!props.isAdmin) return agent.name || t('agent_id_fallback', { id: agent.id })
  const username = agent.username ? ` · ${agent.username}` : ''
  return `${agent.name || t('agent_id_fallback', { id: agent.id })} ${t('agent_user_suffix', { user_id: agent.user_id, username })}`
}

const userLabel = (user) => {
  const name = user?.username || user?.name || t('user_id_fallback', { id: user?.id })
  return `${name} (ID: ${user?.id})`
}

const reloadOptions = async () => {
  await Promise.all([
    props.isAdmin ? loadUsers().catch(() => []) : Promise.resolve([]),
    loadAgents().catch(() => [])
  ])
}

watch(() => form.value.user_id, async (next, prev) => {
  if (!props.isAdmin || next === prev) return
  form.value.agent_id = 0
  await loadAgents().catch(() => [])
})

watch(() => props.fixedAgentId, (value) => {
  if (hasFixedAgent.value) form.value.agent_id = Number(value)
}, { immediate: true })

onMounted(() => reloadOptions())

const validate = () => {
  if (isBindMode.value) {
    if (props.isAdmin && !form.value.user_id) return Promise.reject(new Error(t('select_owner_user')))
    if (!hasFixedAgent.value && !form.value.agent_id) return Promise.reject(new Error(t('select_target_agent')))
    if (!String(form.value.identifier || '').trim()) return Promise.reject(new Error(t('enter_device_code_mac')))
  } else {
    if (props.isAdmin && !form.value.user_id) return Promise.reject(new Error(t('select_owner_user')))
    const deviceName = String(form.value.device_name || '').trim()
    const deviceCode = String(form.value.device_code || '').trim()
    if (props.isAdmin && !deviceName && !deviceCode) return Promise.reject(new Error(t('device_id_or_code')))
    if (!props.isAdmin && !deviceName) return Promise.reject(new Error(t('enter_device_id')))
  }
  return Promise.resolve(true)
}
const resetFields = () => {}
const clearValidate = () => {}
const buildPayload = () => buildDevicePayload(form.value, { isAdmin: props.isAdmin, mode: props.mode })

defineExpose({ validate, resetFields, clearValidate, reloadOptions, buildPayload })
</script>

<template>
  <div class="grid gap-4">
    <!-- Owner user (admin) -->
    <div v-if="isAdmin" class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('owner_user') }}</label>
      <Select v-model="form.user_id">
        <SelectTrigger class="w-full">
          <SelectValue :placeholder="t('select_owner_user')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="user in users" :key="user.id" :value="user.id">{{ userLabel(user) }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Target agent (bind mode) -->
    <div v-if="isBindMode && !hasFixedAgent" class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('target_agent') }}</label>
      <Select v-model="form.agent_id">
        <SelectTrigger class="w-full">
          <SelectValue :placeholder="t('select_agent_to_bind')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="agent in displayAgents" :key="agent.id" :value="agent.id">{{ agent.name || t('agent_id_fallback', { id: agent.id }) }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <!-- Identifier (bind mode) -->
    <div v-if="isBindMode" class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('device_verify_code_mac') }}</label>
      <Input v-model="form.identifier" :placeholder="t('enter_6digit_or_mac')" autocomplete="off" />
      <div class="flex items-center gap-2 flex-wrap text-xs text-[var(--color-text-tertiary)]">
        <span>{{ t('example') }}</span>
        <code class="px-1.5 py-0.5 rounded bg-[var(--color-surface-muted)] font-mono">123456</code>
        <code class="px-1.5 py-0.5 rounded bg-[var(--color-surface-muted)] font-mono">28:0A:C6:1D:3B:E8</code>
      </div>
    </div>

    <!-- Nickname -->
    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('device_nickname') }}</label>
      <Input v-model="form.nick_name" :placeholder="t('device_name_example')" maxlength="50" />
    </div>

    <template v-if="!isBindMode">
      <div class="grid grid-cols-2 gap-4">
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('device_identifier') }}</label>
          <Input v-model="form.device_name" :placeholder="t('device_mac_reported')" />
        </div>
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('activation_code') }}</label>
          <Input v-model="form.device_code" :placeholder="t('device_activation_code')" />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div v-if="isAdmin" class="flex items-center gap-3">
          <Switch :model-value="!!form.activated" @update:model-value="(v) => form.activated = v" />
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('activation_status') }}</label>
        </div>
        <div class="grid gap-1.5">
          <label class="text-sm font-medium text-[var(--color-text)]">{{ t('link_agent') }}</label>
          <Select v-model="form.agent_id" :disabled="isAdmin && !form.user_id">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="t('select_agent')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="0">{{ t('no_agent_linked') }}</SelectItem>
              <SelectItem v-for="agent in displayAgents" :key="agent.id" :value="agent.id">{{ agentLabel(agent) }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </template>
  </div>
</template>
