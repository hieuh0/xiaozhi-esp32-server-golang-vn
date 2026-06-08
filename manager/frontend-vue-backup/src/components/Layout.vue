<template>
  <div v-if="!isMobileDevice" class="flex h-dvh overflow-hidden bg-[var(--color-bg)]">
    <!-- Desktop sidebar -->
    <aside class="hidden lg:flex flex-col w-60 shrink-0 border-r border-[var(--color-line)] bg-[var(--color-surface-muted)]">
      <SidebarBrand />
      <SidebarNavItems :items="navItems" class="flex-1 overflow-y-auto px-3 py-3" />
    </aside>

    <!-- Mobile Sheet -->
    <Sheet v-model:open="sidebarOpen">
      <SheetContent side="left" class="w-60 p-0 flex flex-col">
        <SidebarBrand />
        <SidebarNavItems :items="navItems" class="flex-1 overflow-y-auto px-3 py-3" @navigate="sidebarOpen = false" />
      </SheetContent>
    </Sheet>

    <!-- Main area -->
    <div class="flex flex-col flex-1 min-w-0">
      <AppHeader
        :title="currentPageTitle"
        :eyebrow="authStore.isAdmin ? 'Admin Console' : 'User Workspace'"
        :username="authStore.user?.username || ''"
        :role-label="authStore.isAdmin ? t('admin') : t('normal_user')"
        :initial="usernameInitial"
        :is-admin="authStore.isAdmin"
        :show-admin-shortcuts="authStore.isAdmin"
        @command="handleCommand"
        @toggle-sidebar="sidebarOpen = !sidebarOpen"
      />
      <main id="main-content" class="flex-1 overflow-y-auto overscroll-contain p-6">
        <RouterView />
      </main>
    </div>
  </div>

  <MobileLayout v-else />
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import {
  LayoutDashboard, Settings, Cpu, Mic, Bot, Users, Smartphone,
  BarChart2, Shield, BookOpen, Wand2, Upload
} from '@lucide/vue'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import AppHeader from './AppHeader.vue'
import MobileLayout from './MobileLayout.vue'
import SidebarBrand from './SidebarBrand.vue'
import SidebarNavItems from './SidebarNavItems.vue'
import { useAuthStore } from '../stores/auth'
import { isMobile } from '../utils/device'
import { useLocale } from '../composables/useLocale'

const { t } = useLocale()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const sidebarOpen = ref(false)
const isMobileDevice = computed(() => isMobile())

const currentPageTitle = computed(() => {
  const key = route.meta?.title
  return key ? t(key) : (authStore.isAdmin ? t('dashboard') : t('my_agents'))
})

const usernameInitial = computed(() =>
  (authStore.user?.username || 'U').slice(0, 1).toUpperCase()
)

const adminNavItems = computed(() => [
  { label: t('dashboard'), icon: LayoutDashboard, path: '/dashboard' },
  {
    label: t('service_config'), icon: Settings,
    children: [
      { label: t('ota_config'), path: '/admin/ota-config' },
      { label: t('mqtt_config'), path: '/admin/mqtt-config' },
      { label: t('mqtt_server_config'), path: '/admin/mqtt-server-config' },
      { label: t('udp_config'), path: '/admin/udp-config' },
      { label: t('mcp_config'), path: '/admin/mcp-config' },
      { label: t('mcp_market'), path: '/admin/mcp-market' },
      { label: t('voiceprint_recognition_config'), path: '/admin/speaker-config' },
      { label: t('chat_settings'), path: '/admin/chat-settings' },
    ]
  },
  {
    label: t('ai_config'), icon: Cpu,
    children: [
      { label: t('vad_config'), path: '/admin/vad-config' },
      { label: t('asr_config'), path: '/admin/asr-config' },
      { label: t('llm_config'), path: '/admin/llm-config' },
      { label: t('tts_config'), path: '/admin/tts-config' },
      { label: t('vision_config'), path: '/admin/vision-config' },
      { label: t('memory_config'), path: '/admin/memory-config' },
      { label: t('knowledge_retrieval_config'), path: '/admin/knowledge-search-config' },
    ]
  },
  { label: t('voice_clone'), icon: Mic, path: '/voice-clones' },
  { label: t('resource_pool_stats'), icon: BarChart2, path: '/admin/pool-stats' },
  { label: t('global_role'), icon: Shield, path: '/admin/global-roles' },
  { label: t('user_management'), icon: Users, path: '/admin/users' },
  { label: t('device_management'), icon: Smartphone, path: '/admin/devices' },
  { label: t('agent_management'), icon: Bot, path: '/admin/agents' },
])

const userNavItems = computed(() => [
  { label: t('agent_management'), icon: Bot, path: '/agents' },
  { label: t('device_list'), icon: Smartphone, path: '/user/devices' },
  { label: t('my_roles'), icon: Shield, path: '/user/roles' },
  { label: t('voiceprint_management'), icon: Mic, path: '/speakers' },
  { label: t('voice_clone'), icon: Mic, path: '/voice-clones' },
  { label: t('my_knowledge_base'), icon: BookOpen, path: '/user/knowledge-bases' },
])

const navItems = computed(() => authStore.isAdmin ? adminNavItems.value : userNavItems.value)

const handleCommand = async (command) => {
  if (command === 'api-tokens') { router.push('/user/api-tokens'); return }
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
    } catch { /* cancelled */ }
  }
}
</script>
