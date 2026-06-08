<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { MonitorSmartphone, ChevronRight } from '@lucide/vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()
const router = useRouter()

const devices = ref([])
const loading = ref(false)

const loadDevices = async () => {
  loading.value = true
  try {
    const res = await api.get('/user/devices')
    devices.value = res.data.data || []
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('load_device_list_failed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadDevices)
</script>

<template>
  <div class="p-4 pb-8">
    <h2 class="text-base font-bold text-[var(--color-text)] mb-4">{{ t('my_devices') }}</h2>

    <!-- Loading skeleton -->
    <div v-if="loading" class="space-y-3">
      <div v-for="i in 4" :key="i" class="h-[72px] rounded-2xl bg-[var(--color-surface-muted)] animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="devices.length === 0" class="py-16 flex flex-col items-center gap-3 text-[var(--color-text-secondary)]">
      <MonitorSmartphone class="w-14 h-14 opacity-25" />
      <p class="text-sm">{{ t('no_device_data') }}</p>
    </div>

    <!-- Device cards -->
    <div v-else class="space-y-2.5">
      <button
        v-for="device in devices"
        :key="device.id"
        type="button"
        @click="device.agent_id ? router.push(`/agents/${device.agent_id}/devices`) : null"
        class="flex items-center w-full gap-3 p-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-muted)] active:scale-[0.99] transition-all text-left"
        :class="device.agent_id ? 'cursor-pointer' : 'cursor-default'"
      >
        <div class="relative shrink-0">
          <div class="flex items-center justify-center w-11 h-11 rounded-xl bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)]">
            <MonitorSmartphone class="w-5 h-5" />
          </div>
          <!-- Online indicator -->
          <span
            class="absolute -top-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-[var(--color-surface)]"
            :class="device.is_online ? 'bg-green-500' : 'bg-[var(--color-text-tertiary)]'"
          />
        </div>
        <div class="flex-1 min-w-0">
          <div class="text-sm font-semibold text-[var(--color-text)] truncate">
            {{ device.nick_name || device.device_name || device.device_code }}
          </div>
          <div class="text-xs text-[var(--color-text-secondary)] mt-0.5 flex items-center gap-1">
            <span :class="device.is_online ? 'text-green-600 dark:text-green-400' : ''">
              {{ device.is_online ? t('online') : t('offline') }}
            </span>
            <span v-if="device.agent_name"> · {{ device.agent_name }}</span>
          </div>
        </div>
        <ChevronRight v-if="device.agent_id" class="w-4 h-4 text-[var(--color-text-tertiary)] shrink-0" />
      </button>
    </div>
  </div>
</template>
