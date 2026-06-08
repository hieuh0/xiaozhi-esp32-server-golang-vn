import { createFileRoute, useRouter } from '@tanstack/react-router'
import { ChevronRight, Sun, Moon, Monitor, Globe } from 'lucide-react'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'

const THEME_ICONS: Record<ThemeMode, React.ElementType> = { dark: Moon, light: Sun, auto: Monitor }

const LANGS = [
  { code: 'vi', label: 'Tiếng Việt' },
  { code: 'en', label: 'English' },
  { code: 'zh', label: '中文' },
]

function NavList({ title, items }: { title: string; items: { label: string; path: string }[] }) {
  const router = useRouter()
  return (
    <div className="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface-1)]">
      <p className="px-4 pt-3 pb-1 text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wide">{title}</p>
      <div className="divide-y divide-[var(--color-line)]">
        {items.map(item => (
          <button
            key={item.path}
            type="button"
            onClick={() => router.navigate({ to: item.path as never })}
            className="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-2)] active:bg-[var(--color-surface-2)] transition-colors min-h-[56px]"
          >
            <span className="flex-1 text-sm text-[var(--color-text)] text-left">{item.label}</span>
            <ChevronRight className="w-4 h-4 text-[var(--color-text-tertiary)] shrink-0" />
          </button>
        ))}
      </div>
    </div>
  )
}

function MorePage() {
  const { t, lang, setLang } = useLocale()
  const { mode, nextMode } = useThemeStore()
  const { isAdmin } = useAuthStore()

  const ThemeIcon = THEME_ICONS[mode]
  const themeLabel = { dark: t('dark_mode'), light: t('light_mode'), auto: t('auto_mode') }[mode]
  const nextLang = () => {
    const idx = LANGS.findIndex(l => l.code === lang)
    setLang(LANGS[(idx + 1) % LANGS.length].code)
  }

  const commonItems = isAdmin
    ? [
        { label: t('config_wizard'), path: '/admin/config-wizard' },
        { label: t('resource_pool_stats'), path: '/admin/pool-stats' },
      ]
    : [
        { label: t('my_roles'), path: '/user/roles' },
        { label: t('voice_clone'), path: '/voice-clones' },
        { label: t('my_knowledge_base'), path: '/user/knowledge-bases' },
      ]

  const serviceItems = [
    { label: t('ota_config'), path: '/admin/ota-config' },
    { label: t('mqtt_config'), path: '/admin/mqtt-config' },
    { label: t('mqtt_server_config'), path: '/admin/mqtt-server-config' },
    { label: t('udp_config'), path: '/admin/udp-config' },
    { label: t('mcp_config'), path: '/admin/mcp-config' },
    { label: t('mcp_market'), path: '/admin/mcp-market' },
    { label: t('voiceprint_recognition_config'), path: '/admin/speaker-config' },
    { label: t('chat_settings'), path: '/admin/chat-settings' },
  ]

  const aiItems = [
    { label: t('vad_config'), path: '/admin/vad-config' },
    { label: t('asr_config'), path: '/admin/asr-config' },
    { label: t('llm_config'), path: '/admin/llm-config' },
    { label: t('tts_config'), path: '/admin/tts-config' },
    { label: t('vision_config'), path: '/admin/vision-config' },
    { label: t('memory_config'), path: '/admin/memory-config' },
    { label: t('knowledge_retrieval_config'), path: '/admin/knowledge-search-config' },
  ]

  const systemItems = [
    { label: t('global_role'), path: '/admin/global-roles' },
    { label: t('user_management'), path: '/admin/users' },
    { label: t('device_management'), path: '/admin/devices' },
    { label: t('agent_management'), path: '/admin/agents' },
  ]

  return (
    <div className="p-4 pb-8 space-y-3">
      <div className="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface-1)]">
        <button
          type="button"
          onClick={nextMode}
          className="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-2)] transition-colors min-h-[56px]"
        >
          <ThemeIcon className="w-5 h-5 text-[var(--color-text-secondary)] shrink-0" />
          <span className="flex-1 text-sm text-[var(--color-text)] text-left">{t('theme')}</span>
          <span className="text-sm text-[var(--color-text-secondary)]">{themeLabel}</span>
        </button>
      </div>

      <div className="rounded-2xl overflow-hidden border border-[var(--color-line)] bg-[var(--color-surface-1)]">
        <button
          type="button"
          onClick={nextLang}
          className="flex items-center w-full px-4 py-4 gap-3 hover:bg-[var(--color-surface-2)] transition-colors min-h-[56px]"
        >
          <Globe className="w-5 h-5 text-[var(--color-text-secondary)] shrink-0" />
          <span className="flex-1 text-sm text-[var(--color-text)] text-left">{t('language')}</span>
          <span className="text-sm text-[var(--color-text-secondary)]">{LANGS.find(l => l.code === lang)?.label ?? lang}</span>
        </button>
      </div>

      <NavList title={t('common_functions')} items={commonItems} />

      {isAdmin && (
        <>
          <NavList title={t('service_config')} items={serviceItems} />
          <NavList title={t('ai_config')} items={aiItems} />
          <NavList title={t('system_management')} items={systemItems} />
        </>
      )}
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/more')({
  component: MorePage,
})
