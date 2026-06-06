<template>
  <div class="mobile-layout">
    <MobileNavBar
      :title="pageTitle"
      :show-back="showBack"
      class="mobile-nav-bar"
    >
      <template #right>
        <van-icon name="user-o" class="user-entry-icon" @click="handleUserClick" />
      </template>
    </MobileNavBar>

    <div class="mobile-content" :class="{ 'with-tabbar': showTabBar }">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>

    <MobileTabBar v-if="showTabBar" class="mobile-tabbar" />

    <!-- User menu popup -->
    <van-popup
      v-model:show="showUserMenu"
      position="bottom"
      :style="{ padding: '20px' }"
    >
      <div class="user-menu">
        <div class="user-info">
          <van-icon name="user-circle-o" size="48" />
          <div class="user-details">
            <div class="username">{{ authStore.user?.username }}</div>
            <div class="user-role">{{ roleText }}</div>
          </div>
        </div>
        <van-cell-group inset>
          <van-cell :title="t('more_features')" is-link @click="handleGoMore" />
          <van-cell v-if="!authStore.isAdmin" title="API Token" is-link @click="handleGoApiTokens" />
          <van-cell v-if="authStore.isAdmin" :title="t('config_wizard')" is-link @click="handleGoConfigWizard" />
          <van-cell :title="t('logout')" is-link @click="handleLogout" />
        </van-cell-group>
      </div>
    </van-popup>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { showConfirmDialog, showSuccessToast } from 'vant'
import MobileNavBar from './MobileNavBar.vue'
import MobileTabBar from './MobileTabBar.vue'
import { useAuthStore } from '../stores/auth'
import { isMobile } from '../utils/device'
import { useLocale } from '../composables/useLocale'
const { t } = useLocale()

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const showUserMenu = ref(false)

// Page title
const pageTitle = computed(() => {
  const key = route.meta?.title
  return key ? t(key) : t('xiaozhi_management_system')
})

// Show back button when not on home and not on a tab-bar page
const showBack = computed(() => {
  const hideBackPages = ['/dashboard', '/agents', '/user/speakers', '/more', '/login']
  const currentPath = route.path
  return !hideBackPages.some(path => currentPath === path || currentPath.startsWith(path + '/'))
})

// Show bottom tab bar
const showTabBar = computed(() => {
  const hideTabBarPages = [
    '/login',
    '/setup',
    '/test',
    '/simple-login'
  ]
  const currentPath = route.path

  // Hide tab bar on detail pages
  if (currentPath.includes('/edit') || currentPath.includes('/detail') || currentPath.includes('/history')) {
    return false
  }

  return !hideTabBarPages.includes(currentPath)
})

// Role label
const roleText = computed(() => {
  return authStore.isAdmin ? t('admin') : t('normal_user')
})

// User avatar click
const handleUserClick = () => {
  showUserMenu.value = true
}


const handleGoMore = () => {
  router.push('/more')
  showUserMenu.value = false
}

const handleGoApiTokens = () => {
  router.push('/user/api-tokens')
  showUserMenu.value = false
}

const handleGoConfigWizard = () => {
  router.push('/admin/config-wizard')
  showUserMenu.value = false
}

// Logout
const handleLogout = async () => {
  try {
    await showConfirmDialog({
      title: t('hint'),
      message: t('confirm_logout')
    })

    authStore.logout()
    showSuccessToast(t('logged_out'))
    router.push('/login')
    showUserMenu.value = false
  } catch {
    // User cancelled
  }
}

// Close user menu on route change
watch(
  () => route.path,
  () => {
    showUserMenu.value = false
  }
)
</script>

<style scoped>
.mobile-layout {
  height: 100dvh;
  background:
    radial-gradient(circle at top left, rgba(0, 122, 255, 0.08), transparent 28%),
    linear-gradient(180deg, #fbfcfe 0%, var(--apple-bg) 100%);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.mobile-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: 0;
  -webkit-overflow-scrolling: touch;
}

.mobile-content.with-tabbar {
  padding-bottom: calc(84px + env(safe-area-inset-bottom));
}

/* Page transition animation */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}


.user-entry-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 12px;
  background: rgba(0, 122, 255, 0.08);
  color: var(--apple-primary);
  font-size: 16px;
  line-height: 1;
}

.user-menu {
  padding: 4px 0 10px;
}

.user-info {
  display: flex;
  align-items: center;
  padding: 12px 16px 18px;
  margin-bottom: 16px;
}

.user-info .van-icon {
  margin-right: 16px;
  color: var(--apple-primary);
}

.user-details {
  flex: 1;
}

.username {
  font-size: 18px;
  font-weight: 700;
  color: var(--apple-text);
  margin-bottom: 4px;
}

.user-role {
  font-size: 14px;
  color: var(--apple-text-secondary);
}

:deep(.van-popup) {
  border-radius: 24px 24px 0 0;
  border: 1px solid rgba(255, 255, 255, 0.88);
}

:deep(.van-cell-group--inset) {
  margin: 0;
}
</style>
