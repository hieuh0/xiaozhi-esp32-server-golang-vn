<template>
  <div v-if="!isMobileDevice" class="layout-shell" :class="{ 'is-form-heavy-route': isFormHeavyRoute }">
    <aside class="sidebar-shell">
      <div class="sidebar-card apple-surface">
        <div class="brand-panel">
          <img class="brand-mark" :src="appLogo" :alt="t('xiaozhi_management_system')" />
          <div class="brand-copy">
            <p class="brand-eyebrow">Control Center</p>
            <h3>{{ t('xiaozhi_management_system') }}</h3>
            <p>{{ authStore.isAdmin ? 'AI 服务与设备管理台' : '设备与智能体工作台' }}</p>
          </div>
        </div>

        <div class="sidebar-meta">
          <span class="apple-chip is-primary">{{ authStore.isAdmin ? '管理员模式' : '用户模式' }}</span>
          <span class="apple-chip is-success">{{ t('online') }}</span>
        </div>

        <el-scrollbar class="sidebar-scroll">
          <el-menu
            :default-active="$route.path"
            class="sidebar-menu"
            router
            unique-opened
            :collapse-transition="false"
          >
            <el-menu-item v-if="authStore.isAdmin" index="/dashboard">
              <el-icon><House /></el-icon>
              <span>{{ t('dashboard') }}</span>
            </el-menu-item>

            <el-menu-item v-if="!authStore.isAdmin" index="/agents">
              <el-icon><Connection /></el-icon>
              <span>{{ t('agent_management') }}</span>
            </el-menu-item>

            <el-menu-item v-if="!authStore.isAdmin" index="/user/devices">
              <el-icon><Iphone /></el-icon>
              <span>{{ t('device_list') }}</span>
            </el-menu-item>

            <el-menu-item v-if="!authStore.isAdmin" index="/user/roles">
              <el-icon><User /></el-icon>
              <span>{{ t('my_roles') }}</span>
            </el-menu-item>

            <el-menu-item v-if="!authStore.isAdmin" index="/speakers">
              <el-icon><Microphone /></el-icon>
              <span>{{ t('voiceprint_management') }}</span>
            </el-menu-item>

            <el-menu-item v-if="!authStore.isAdmin" index="/voice-clones">
              <el-icon><Microphone /></el-icon>
              <span>{{ t('voice_clone') }}</span>
            </el-menu-item>

            <el-menu-item v-if="!authStore.isAdmin" index="/user/knowledge-bases">
              <el-icon><Document /></el-icon>
              <span>{{ t('my_knowledge_base') }}</span>
            </el-menu-item>

            <el-sub-menu v-if="authStore.isAdmin" index="/admin/service-config">
              <template #title>
                <el-icon><Tools /></el-icon>
                <span>{{ t('service_config') }}</span>
              </template>
              <el-menu-item index="/admin/ota-config">{{ t('ota_config') }}</el-menu-item>
              <el-menu-item index="/admin/mqtt-config">{{ t('mqtt_config') }}</el-menu-item>
              <el-menu-item index="/admin/mqtt-server-config">{{ t('mqtt_server_config') }}</el-menu-item>
              <el-menu-item index="/admin/udp-config">{{ t('udp_config') }}</el-menu-item>
              <el-sub-menu index="/admin/mcp-config-group" class="menu-sub-child">
                <template #title>{{ t('mcp_config') }}</template>
                <el-menu-item class="menu-grandchild" index="/admin/mcp-config">配置</el-menu-item>
                <el-menu-item class="menu-grandchild" index="/admin/mcp-market">{{ t('mcp_market') }}</el-menu-item>
              </el-sub-menu>
              <el-menu-item index="/admin/speaker-config">{{ t('voiceprint_recognition_config') }}</el-menu-item>
              <el-menu-item index="/admin/chat-settings">{{ t('chat_settings') }}</el-menu-item>
            </el-sub-menu>

            <el-sub-menu v-if="authStore.isAdmin" index="/admin/ai-config">
              <template #title>
                <el-icon><Cpu /></el-icon>
                <span>{{ t('ai_config') }}</span>
              </template>
              <el-menu-item index="/admin/vad-config">{{ t('vad_config') }}</el-menu-item>
              <el-menu-item index="/admin/asr-config">{{ t('asr_config') }}</el-menu-item>
              <el-menu-item index="/admin/llm-config">{{ t('llm_config') }}</el-menu-item>
              <el-menu-item index="/admin/tts-config">{{ t('tts_config') }}</el-menu-item>
              <el-menu-item index="/admin/vision-config">{{ t('vision_config') }}</el-menu-item>
              <el-menu-item index="/admin/memory-config">{{ t('memory_config') }}</el-menu-item>
              <el-menu-item index="/admin/knowledge-search-config">{{ t('knowledge_retrieval_config') }}</el-menu-item>
            </el-sub-menu>

            <el-menu-item v-if="authStore.isAdmin" index="/voice-clones">
              <el-icon><Microphone /></el-icon>
              <span>{{ t('voice_clone') }}</span>
            </el-menu-item>

            <el-menu-item v-if="authStore.isAdmin" index="/admin/pool-stats">
              <el-icon><DataAnalysis /></el-icon>
              <span>{{ t('resource_pool_stats') }}</span>
            </el-menu-item>

            <el-menu-item v-if="authStore.isAdmin" index="/admin/global-roles">
              <el-icon><Setting /></el-icon>
              <span>{{ t('global_role') }}</span>
            </el-menu-item>

            <el-menu-item v-if="authStore.isAdmin" index="/admin/users">
              <el-icon><UserFilled /></el-icon>
              <span>{{ t('user_management') }}</span>
            </el-menu-item>

            <el-menu-item v-if="authStore.isAdmin" index="/admin/devices">
              <el-icon><Iphone /></el-icon>
              <span>{{ t('device_management') }}</span>
            </el-menu-item>

            <el-menu-item v-if="authStore.isAdmin" index="/admin/agents">
              <el-icon><Connection /></el-icon>
              <span>{{ t('agent_management') }}</span>
            </el-menu-item>
          </el-menu>
        </el-scrollbar>
      </div>
    </aside>

    <div class="content-shell">
      <AppHeader
        :title="currentPageTitle"
        :eyebrow="authStore.isAdmin ? 'Admin Console' : 'User Workspace'"
        :username="authStore.user?.username || ''"
        :role-label="authStore.isAdmin ? '管理员' : '普通用户'"
        :initial="usernameInitial"
        :is-admin="authStore.isAdmin"
        :show-admin-shortcuts="authStore.isAdmin"
        @command="handleCommand"
      />

      <main class="main-shell">
        <router-view />
      </main>
    </div>
  </div>

  <MobileLayout v-else />
</template>

<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import { isMobile } from '../utils/device'
import AppHeader from './AppHeader.vue'
import MobileLayout from './MobileLayout.vue'
import appLogo from '@/assets/brand/app-logo.webp'
import {
  House,
  Setting,
  User,
  Tools,
  Cpu,
  UserFilled,
  Iphone,
  Connection,
  Microphone,
  DataAnalysis,
  Document
} from '@element-plus/icons-vue'
import { useLocale } from '../composables/useLocale'


const { t } = useLocale()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const isMobileDevice = computed(() => isMobile())

const currentPageTitle = computed(() => route.meta?.title || (authStore.isAdmin ? t('dashboard') : '我的智能体'))
const isFormHeavyRoute = computed(() => ['AgentEdit', 'UserAgentEdit'].includes(String(route.name || '')))

const usernameInitial = computed(() => {
  const username = authStore.user?.username || 'U'
  return username.slice(0, 1).toUpperCase()
})

const handleCommand = async (command) => {
  if (command === 'api-tokens') {
    router.push('/user/api-tokens')
    return
  }

  if (command === 'logout') {
    try {
      await ElMessageBox.confirm(t('confirm_logout'), t('hint'), {
        confirmButtonText: t('confirm'),
        cancelButtonText: t('cancel'),
        type: 'warning'
      })

      authStore.logout()
      ElMessage.success(t('logged_out'))
      router.push('/login')
    } catch {
      // 用户取消
    }
  }
}
</script>

<style scoped>
.layout-shell {
  height: 100dvh;
  min-height: 0;
  padding: 20px;
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 16px;
  overflow: hidden;
}

.sidebar-shell {
  min-width: 0;
}

.sidebar-card {
  height: calc(100dvh - 40px);
  padding: 12px;
  border-radius: 26px;
  display: flex;
  flex-direction: column;
}

.brand-panel {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 14px;
}

.brand-mark {
  width: 38px;
  height: 38px;
  border-radius: 14px;
  display: block;
  object-fit: cover;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.36), 0 12px 22px rgba(0, 122, 255, 0.24);
  flex: none;
}

.brand-copy h3 {
  margin: 0;
  font-size: 15px;
  line-height: 1.2;
}

.brand-copy {
  min-width: 0;
}

.brand-copy h3,
.brand-copy p {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-copy p {
  margin: 3px 0 0;
  color: var(--apple-text-secondary);
  font-size: 11px;
  line-height: 1.35;
}

.brand-eyebrow {
  margin: 0 0 4px !important;
  color: var(--apple-primary) !important;
  font-size: 9px !important;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sidebar-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 14px;
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
  margin: 0 -4px -4px;
  padding-right: 4px;
}

.sidebar-menu {
  background: transparent;
  border-right: 0;
  padding: 2px 0 12px;
}

.sidebar-menu :deep(.el-menu-item),
.sidebar-menu :deep(.el-sub-menu__title) {
  height: 40px;
  margin-bottom: 4px;
  border-radius: 14px;
  color: var(--apple-text-secondary);
  font-weight: 600;
  padding: 0 12px !important;
  font-size: 13px;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
}

.sidebar-menu :deep(.el-sub-menu__title) {
  padding-right: 28px !important;
}

.sidebar-menu :deep(.el-menu-item span),
.sidebar-menu :deep(.el-sub-menu__title span) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-menu :deep(.el-menu-item:hover),
.sidebar-menu :deep(.el-sub-menu__title:hover) {
  color: var(--apple-text);
  background: rgba(255, 255, 255, 0.82);
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  color: var(--apple-primary);
  background: rgba(0, 122, 255, 0.1);
  box-shadow: inset 0 0 0 1px rgba(0, 122, 255, 0.08);
}

.sidebar-menu :deep(.el-sub-menu .el-menu-item) {
  height: 36px;
  margin: 3px 0 3px 6px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.48);
  padding-left: 46px !important;
}

.sidebar-menu :deep(.menu-sub-child > .el-sub-menu__title) {
  height: 36px;
  margin: 3px 0 3px 6px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.48);
  padding-left: 46px !important;
  padding-right: 28px !important;
}

.sidebar-menu :deep(.menu-sub-child .menu-grandchild) {
  margin-left: 24px;
  padding-left: 46px !important;
  background: rgba(255, 255, 255, 0.34);
}

.sidebar-menu :deep(.el-menu-item .el-icon),
.sidebar-menu :deep(.el-sub-menu__title .el-icon) {
  margin-right: 8px;
  font-size: 15px;
  flex: none;
}

.sidebar-menu :deep(.el-sub-menu__icon-arrow) {
  right: 8px;
}

.content-shell {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.main-shell {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 0 4px 4px 0;
  scrollbar-gutter: stable;
  -webkit-overflow-scrolling: touch;
}

.layout-shell.is-form-heavy-route :deep(.apple-surface) {
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
  box-shadow: var(--apple-shadow-sm);
}

@media (max-width: 1360px) {
  .layout-shell {
    grid-template-columns: 208px minmax(0, 1fr);
  }
}
</style>
