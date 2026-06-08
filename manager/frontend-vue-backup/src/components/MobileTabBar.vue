<script setup>
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { Home, Settings, Users, MoreHorizontal, Bot, Mic } from '@lucide/vue'
import { useAuthStore } from '../stores/auth'
import { useLocale } from '../composables/useLocale'

const route = useRoute()
const authStore = useAuthStore()
const { t } = useLocale()

const tabs = computed(() => {
  if (authStore.isAdmin) {
    return [
      { path: '/dashboard', label: t('home'), icon: Home },
      { path: '/admin/config-overview', label: t('config'), icon: Settings },
      { path: '/admin/users', label: t('manage'), icon: Users },
      { path: '/more', label: t('more'), icon: MoreHorizontal }
    ]
  }
  return [
    { path: '/agents', label: t('agent'), icon: Bot },
    { path: '/user/speakers', label: t('voiceprint'), icon: Mic },
    { path: '/more', label: t('more'), icon: MoreHorizontal }
  ]
})

const isActive = (path) => {
  if (route.path === path) return true
  return route.path.startsWith(path + '/')
}
</script>

<template>
  <nav
    class="mx-3 mb-3 flex rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden shadow-sm"
    :style="{ paddingBottom: 'env(safe-area-inset-bottom, 0px)' }"
  >
    <RouterLink
      v-for="tab in tabs"
      :key="tab.path"
      :to="tab.path"
      class="flex-1 flex flex-col items-center justify-center py-3 gap-1 transition-colors"
      :class="isActive(tab.path)
        ? 'text-[var(--color-primary)]'
        : 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text)]'"
    >
      <component :is="tab.icon" class="w-5 h-5" />
      <span class="text-[10px] font-medium">{{ tab.label }}</span>
    </RouterLink>
  </nav>
</template>
