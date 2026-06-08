<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { User } from '@lucide/vue'
import MobileNavBar from './MobileNavBar.vue'
import MobileTabBar from './MobileTabBar.vue'
import { useAuthStore } from '../stores/auth'
import { useLocale } from '../composables/useLocale'
import { Sheet, SheetContent } from '@/components/ui/sheet'

const { t } = useLocale()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const showUserMenu = ref(false)

const pageTitle = computed(() => {
  const key = route.meta?.title
  return key ? t(key) : t('xiaozhi_management_system')
})

const showBack = computed(() => {
  const hideBackPages = ['/dashboard', '/agents', '/user/speakers', '/more', '/login']
  const currentPath = route.path
  return !hideBackPages.some(p => currentPath === p || currentPath.startsWith(p + '/'))
})

const showTabBar = computed(() => {
  const currentPath = route.path
  if (['/login', '/setup', '/test', '/simple-login'].includes(currentPath)) return false
  if (currentPath.includes('/edit') || currentPath.includes('/detail') || currentPath.includes('/history')) return false
  return true
})

const roleText = computed(() => authStore.isAdmin ? t('admin') : t('normal_user'))

const handleGoMore = () => { router.push('/more'); showUserMenu.value = false }
const handleGoApiTokens = () => { router.push('/user/api-tokens'); showUserMenu.value = false }
const handleGoConfigWizard = () => { router.push('/admin/config-wizard'); showUserMenu.value = false }

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm(t('confirm_logout'), t('hint'), {
      confirmButtonText: t('confirm'),
      cancelButtonText: t('cancel'),
      type: 'warning'
    })
    authStore.logout()
    ElMessage.success(t('logged_out'))
    router.push('/login')
    showUserMenu.value = false
  } catch {
    // user cancelled
  }
}

watch(() => route.path, () => { showUserMenu.value = false })
</script>

<template>
  <div class="flex flex-col h-dvh bg-[var(--color-bg)] overflow-hidden">
    <MobileNavBar :title="pageTitle" :show-back="showBack">
      <template #right>
        <button
          type="button"
          @click="showUserMenu = true"
          class="flex items-center justify-center w-9 h-9 rounded-xl bg-blue-50 text-[var(--color-primary)] dark:bg-blue-900/30 mr-1"
        >
          <User class="w-4 h-4" />
        </button>
      </template>
    </MobileNavBar>

    <main class="flex-1 min-h-0 overflow-y-auto overscroll-contain -webkit-overflow-scrolling-touch">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>

    <MobileTabBar v-if="showTabBar" />

    <!-- User menu bottom sheet -->
    <Sheet v-model:open="showUserMenu">
      <SheetContent side="bottom" class="rounded-t-3xl px-0 pb-0" :show-close-button="false">
        <!-- User info -->
        <div class="flex items-center gap-4 px-5 py-5 border-b border-[var(--color-line)]">
          <div class="flex items-center justify-center w-14 h-14 rounded-2xl bg-blue-50 dark:bg-blue-900/30 text-[var(--color-primary)] shrink-0">
            <User class="w-7 h-7" />
          </div>
          <div>
            <div class="text-lg font-bold text-[var(--color-text)]">{{ authStore.user?.username }}</div>
            <div class="text-sm text-[var(--color-text-secondary)]">{{ roleText }}</div>
          </div>
        </div>

        <!-- Menu rows -->
        <div class="divide-y divide-[var(--color-line)]">
          <button type="button" @click="handleGoMore"
            class="flex items-center w-full px-5 py-4 text-sm text-[var(--color-text)] hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors text-left">
            {{ t('more_features') }}
            <span class="ml-auto text-[var(--color-text-tertiary)] text-base">›</span>
          </button>
          <button v-if="!authStore.isAdmin" type="button" @click="handleGoApiTokens"
            class="flex items-center w-full px-5 py-4 text-sm text-[var(--color-text)] hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors text-left">
            API Token
            <span class="ml-auto text-[var(--color-text-tertiary)] text-base">›</span>
          </button>
          <button v-if="authStore.isAdmin" type="button" @click="handleGoConfigWizard"
            class="flex items-center w-full px-5 py-4 text-sm text-[var(--color-text)] hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors text-left">
            {{ t('config_wizard') }}
            <span class="ml-auto text-[var(--color-text-tertiary)] text-base">›</span>
          </button>
          <button type="button" @click="handleLogout"
            class="flex items-center w-full px-5 py-4 text-sm text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 active:bg-red-100 transition-colors text-left">
            {{ t('logout') }}
          </button>
        </div>

        <div :style="{ height: 'max(16px, env(safe-area-inset-bottom))' }" />
      </SheetContent>
    </Sheet>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
