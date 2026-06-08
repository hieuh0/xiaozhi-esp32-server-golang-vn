<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Sun, Moon, Monitor, ChevronRight } from '@lucide/vue'
import { useAuthStore } from '../../stores/auth'
import { useThemeStore } from '../../stores/theme'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()
const themeStore = useThemeStore()
const router = useRouter()
const authStore = useAuthStore()

const themeLabel = computed(() => ({
  dark: t('dark_mode'),
  light: t('light_mode'),
  auto: t('auto_mode')
})[themeStore.mode])

const themeIcon = computed(() => ({
  dark: Moon,
  light: Sun,
  auto: Monitor
})[themeStore.mode])

const commonItems = computed(() => {
  if (authStore.isAdmin) {
    return [
      { title: t('config_wizard'), desc: t('first_deploy_hint'), path: '/admin/config-wizard' },
      { title: t('resource_pool_stats'), desc: t('view_resource_pool'), path: '/admin/pool-stats' }
    ]
  }
  return [
    { title: t('my_roles'), desc: t('manage_personal_roles'), path: '/user/roles' },
    { title: t('voice_clone'), desc: t('manage_voice_clone'), path: '/voice-clones' },
    { title: t('my_knowledge_base'), desc: t('manage_kb_docs'), path: '/user/knowledge-bases' }
  ]
})

const serviceItems = [
  { title: t('ota_config'), path: '/admin/ota-config' },
  { title: t('mqtt_config'), path: '/admin/mqtt-config' },
  { title: t('mqtt_server_config'), path: '/admin/mqtt-server-config' },
  { title: t('udp_config'), path: '/admin/udp-config' },
  { title: t('mcp_config'), path: '/admin/mcp-config' },
  { title: t('mcp_market'), path: '/admin/mcp-market' },
  { title: t('voiceprint_recognition_config'), path: '/admin/speaker-config' },
  { title: t('chat_settings'), path: '/admin/chat-settings' }
]

const aiItems = [
  { title: t('vad_config'), path: '/admin/vad-config' },
  { title: t('asr_config'), path: '/admin/asr-config' },
  { title: t('llm_config'), path: '/admin/llm-config' },
  { title: t('tts_config'), path: '/admin/tts-config' },
  { title: t('vision_config'), path: '/admin/vision-config' },
  { title: t('memory_config'), path: '/admin/memory-config' },
  { title: t('knowledge_retrieval_config'), path: '/admin/knowledge-search-config' }
]

const systemItems = [
  { title: t('global_role'), path: '/admin/global-roles' },
  { title: t('user_management'), path: '/admin/users' },
  { title: t('device_management'), path: '/admin/devices' },
  { title: t('agent_management'), path: '/admin/agents' }
]

const go = (path) => router.push(path)
</script>

<template>
  <div class="p-4 pb-8 space-y-3">
    <!-- Theme toggle -->
    <div class="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface)]">
      <button type="button" @click="themeStore.nextMode()"
        class="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors min-h-[56px]">
        <component :is="themeIcon" class="w-5 h-5 text-[var(--color-text-secondary)] shrink-0" />
        <span class="flex-1 text-sm text-[var(--color-text)] text-left">{{ t('theme') }}</span>
        <span class="text-sm text-[var(--color-text-secondary)]">{{ themeLabel }}</span>
      </button>
    </div>

    <!-- Common functions -->
    <div v-if="commonItems.length" class="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface)]">
      <p class="px-4 pt-3 pb-1 text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wide">{{ t('common_functions') }}</p>
      <div class="divide-y divide-[var(--color-line)]">
        <button v-for="item in commonItems" :key="item.path" type="button" @click="go(item.path)"
          class="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors min-h-[56px]">
          <div class="flex-1 text-left min-w-0">
            <div class="text-sm text-[var(--color-text)]">{{ item.title }}</div>
            <div v-if="item.desc" class="text-xs text-[var(--color-text-secondary)] mt-0.5">{{ item.desc }}</div>
          </div>
          <ChevronRight class="w-4 h-4 text-[var(--color-text-tertiary)] shrink-0" />
        </button>
      </div>
    </div>

    <template v-if="authStore.isAdmin">
      <!-- Service config -->
      <div class="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface)]">
        <p class="px-4 pt-3 pb-1 text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wide">{{ t('service_config') }}</p>
        <div class="divide-y divide-[var(--color-line)]">
          <button v-for="item in serviceItems" :key="item.path" type="button" @click="go(item.path)"
            class="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors min-h-[56px]">
            <span class="flex-1 text-sm text-[var(--color-text)] text-left">{{ item.title }}</span>
            <ChevronRight class="w-4 h-4 text-[var(--color-text-tertiary)]" />
          </button>
        </div>
      </div>

      <!-- AI config -->
      <div class="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface)]">
        <p class="px-4 pt-3 pb-1 text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wide">{{ t('ai_config') }}</p>
        <div class="divide-y divide-[var(--color-line)]">
          <button v-for="item in aiItems" :key="item.path" type="button" @click="go(item.path)"
            class="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors min-h-[56px]">
            <span class="flex-1 text-sm text-[var(--color-text)] text-left">{{ item.title }}</span>
            <ChevronRight class="w-4 h-4 text-[var(--color-text-tertiary)]" />
          </button>
        </div>
      </div>

      <!-- System management -->
      <div class="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface)]">
        <p class="px-4 pt-3 pb-1 text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wide">{{ t('system_management') }}</p>
        <div class="divide-y divide-[var(--color-line)]">
          <button v-for="item in systemItems" :key="item.path" type="button" @click="go(item.path)"
            class="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-muted)] active:bg-[var(--color-surface-strong)] transition-colors min-h-[56px]">
            <span class="flex-1 text-sm text-[var(--color-text)] text-left">{{ item.title }}</span>
            <ChevronRight class="w-4 h-4 text-[var(--color-text-tertiary)]" />
          </button>
        </div>
      </div>
    </template>
  </div>
</template>
