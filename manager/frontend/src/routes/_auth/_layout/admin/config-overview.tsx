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
    <div className="p-6 grid gap-8">
      <PageHeader eyebrow="ADMIN" title={t('config_management')} />
      {CONFIG_GROUPS.map((group) => (
        <div key={group.titleKey}>
          <p className="text-[10px] font-bold text-[var(--color-primary)] uppercase tracking-widest font-mono mb-4">
            {t(group.titleKey)}
          </p>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-4">
            {group.items.map((item) => (
              <button
                key={item.route}
                type="button"
                onClick={() => router.navigate({ to: item.route as never })}
                className="group flex flex-col gap-4 p-5 bg-[var(--color-surface-1)] border border-[var(--color-line)] rounded-xl cursor-pointer text-left hover:border-[var(--color-primary)]/40 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-[var(--shadow-card-hover)]"
              >
                <div className="w-10 h-10 rounded-lg flex items-center justify-center bg-[var(--color-surface-2)] border border-[var(--color-line)] text-[var(--color-primary)] transition-colors group-hover:bg-[var(--color-primary-soft)] group-hover:border-[var(--color-primary)]/40">
                  <item.icon className="w-5 h-5" />
                </div>
                <span className="text-sm font-semibold text-[var(--color-text)] leading-snug">{t(item.labelKey)}</span>
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
