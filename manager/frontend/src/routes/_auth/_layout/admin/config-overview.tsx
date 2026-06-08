import { createFileRoute, useRouter } from '@tanstack/react-router'
import {
  Upload, Wifi, Cpu, Settings, MessageCircle,
  Link, ShoppingCart, Mic, Eye, Brain, Search, Cloud, Monitor,
} from 'lucide-react'
import { useLocale } from '@/hooks/use-locale'
import { PageHeader } from '@/components/ui/page-header'

type ConfigItem = { labelKey: string; route: string; icon: React.ElementType }
type ConfigGroup = { titleKey: string; items: ConfigItem[] }

const CONFIG_GROUPS: ConfigGroup[] = [
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
      { labelKey: 'chat_settings', route: '/admin/chat-settings', icon: MessageCircle },
    ],
  },
  {
    titleKey: 'ai_config',
    items: [
      { labelKey: 'vad_config', route: '/admin/vad-config', icon: Cpu },
      { labelKey: 'asr_config', route: '/admin/asr-config', icon: Mic },
      { labelKey: 'llm_config', route: '/admin/llm-config', icon: Cloud },
      { labelKey: 'tts_config', route: '/admin/tts-config', icon: Settings },
      { labelKey: 'vision_config', route: '/admin/vision-config', icon: Eye },
      { labelKey: 'memory_config', route: '/admin/memory-config', icon: Brain },
      { labelKey: 'knowledge_retrieval_config', route: '/admin/knowledge-search-config', icon: Search },
    ],
  },
]

function ConfigOverviewPage() {
  const { t } = useLocale()
  const router = useRouter()

  return (
    <div className="p-6 grid gap-7">
      <PageHeader eyebrow="ADMIN" title={t('config_management')} />
      {CONFIG_GROUPS.map((group) => (
        <div key={group.titleKey}>
          <h2 className="text-xs font-bold text-[var(--color-text-secondary)] uppercase tracking-widest mb-3">
            {t(group.titleKey)}
          </h2>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(160px,1fr))] gap-3">
            {group.items.map((item) => (
              <button
                key={item.route}
                type="button"
                onClick={() => router.navigate({ to: item.route as never })}
                className="flex flex-col items-center justify-center gap-2.5 min-h-[88px] p-4 bg-[var(--color-surface-1)] border border-[var(--color-line)] rounded-xl cursor-pointer text-center hover:bg-[var(--color-primary-soft)] hover:border-[var(--color-primary)] transition-colors"
              >
                <item.icon className="w-[22px] h-[22px] text-[var(--color-primary)]" />
                <span className="text-[13px] font-semibold text-[var(--color-text)] leading-snug">{t(item.labelKey)}</span>
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/config-overview')({
  component: ConfigOverviewPage,
})
