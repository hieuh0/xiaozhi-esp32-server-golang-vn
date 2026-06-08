<template>
  <div class="flex flex-col gap-7">
    <div v-for="group in configGroups" :key="group.titleKey">
      <h2 class="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-widest mb-3">{{ t(group.titleKey) }}</h2>
      <div class="grid grid-cols-[repeat(auto-fill,minmax(160px,1fr))] gap-3">
        <button
          v-for="item in group.items"
          :key="item.route"
          class="flex flex-col items-center justify-center gap-2.5 min-h-[88px] p-4 bg-[var(--color-surface-strong)] border border-[var(--color-line)] rounded-xl cursor-pointer text-center hover:bg-[var(--color-primary-soft)] hover:border-[var(--color-primary)] transition-colors"
          @click="router.push(item.route)"
        >
          <component :is="item.icon" class="w-[22px] h-[22px] text-[var(--color-primary)]" />
          <span class="text-[13px] font-semibold text-[var(--color-text)] leading-snug">{{ t(item.labelKey) }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useLocale } from '../../composables/useLocale'
import {
  Upload, Wifi, Cpu, Settings, MessageCircle,
  Link, ShoppingCart, Mic, BarChart3, Cloud, Monitor
} from '@lucide/vue'

const router = useRouter()
const { t } = useLocale()

const configGroups = [
  {
    titleKey: 'service_config',
    items: [
      { labelKey: 'ota_config', route: '/admin/ota-config', icon: Upload },
      { labelKey: 'mqtt_config', route: '/admin/mqtt-config', icon: Wifi },
      { labelKey: 'mqtt_server_config_management', route: '/admin/mqtt-server-config', icon: Wifi },
      { labelKey: 'udp_config', route: '/admin/udp-config', icon: Monitor },
      { labelKey: 'mcp_config_management', route: '/admin/mcp-config', icon: Link },
      { labelKey: 'mcp_market', route: '/admin/mcp-market', icon: ShoppingCart },
      { labelKey: 'voiceprint_recognition_config', route: '/admin/speaker-config', icon: Mic },
      { labelKey: 'chat_settings', route: '/admin/chat-settings', icon: MessageCircle }
    ]
  },
  {
    titleKey: 'ai_config',
    items: [
      { labelKey: 'vad_config', route: '/admin/vad-config', icon: Cpu },
      { labelKey: 'asr_config', route: '/admin/asr-config', icon: Mic },
      { labelKey: 'llm_config', route: '/admin/llm-config', icon: Cloud },
      { labelKey: 'tts_config', route: '/admin/tts-config', icon: Settings },
      { labelKey: 'vision_config', route: '/admin/vision-config', icon: BarChart3 },
      { labelKey: 'memory_config', route: '/admin/memory-config', icon: BarChart3 },
      { labelKey: 'knowledge_retrieval_config', route: '/admin/knowledge-search-config', icon: BarChart3 }
    ]
  }
]
</script>
