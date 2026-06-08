<template>
  <div class="flex items-center gap-2.5 px-4 py-4 border-b border-[var(--color-line)] shrink-0">
    <img
      :src="appLogo"
      :alt="t('xiaozhi_management_system')"
      class="size-9 rounded-xl object-cover shrink-0"
    />
    <div class="min-w-0">
      <p class="text-[10px] font-bold uppercase tracking-widest text-[var(--color-primary)]">
        Control Center
      </p>
      <p class="truncate text-sm font-semibold text-[var(--color-text)]">
        {{ t('xiaozhi_management_system') }}
      </p>
      <p class="truncate text-xs text-[var(--color-text-secondary)]">
        {{ authStore.isAdmin ? t('admin_panel_title') : t('device_agent_workbench') }}
      </p>
    </div>
  </div>
</template>

<script setup>
import appLogo from '@/assets/brand/app-logo.webp'
import { useAuthStore } from '../stores/auth'
import { useLocale } from '../composables/useLocale'

const { t } = useLocale()
const authStore = useAuthStore()
</script>
