import { useState } from 'react'
import { Link, useRouterState } from '@tanstack/react-router'
import {
  LayoutDashboard, Settings, Cpu, Mic, Bot, Users, Smartphone,
  BarChart2, Shield, BookOpen, ChevronDown,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'

interface NavLeaf { label: string; path: string; icon?: React.ElementType }
interface NavGroup { label: string; icon: React.ElementType; children: { label: string; path: string }[] }
type NavItem = NavLeaf | NavGroup

function isGroup(item: NavItem): item is NavGroup {
  return 'children' in item
}

interface Props {
  className?: string
  onNavigate?: () => void
}

export function SidebarNavItems({ className, onNavigate }: Props) {
  const { t } = useLocale()
  const isAdmin = useAuthStore((s) => s.isAdmin)
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const [open, setOpen] = useState<Record<string, boolean>>({})

  const isActive = (path: string) => pathname === path || pathname.startsWith(path + '/')
  const isGroupActive = (item: NavGroup) => item.children.some((c) => isActive(c.path))

  const adminItems: NavItem[] = [
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
      ],
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
      ],
    },
    { label: t('voice_clone'), icon: Mic, path: '/voice-clones' },
    { label: t('resource_pool_stats'), icon: BarChart2, path: '/admin/pool-stats' },
    { label: t('global_role'), icon: Shield, path: '/admin/global-roles' },
    { label: t('user_management'), icon: Users, path: '/admin/users' },
    { label: t('device_management'), icon: Smartphone, path: '/admin/devices' },
    { label: t('agent_management'), icon: Bot, path: '/admin/agents' },
  ]

  const userItems: NavItem[] = [
    { label: t('agent_management'), icon: Bot, path: '/agents' },
    { label: t('device_list'), icon: Smartphone, path: '/user/devices' },
    { label: t('my_roles'), icon: Shield, path: '/user/roles' },
    { label: t('voiceprint_management'), icon: Mic, path: '/speakers' },
    { label: t('voice_clone'), icon: Mic, path: '/voice-clones' },
    { label: t('my_knowledge_base'), icon: BookOpen, path: '/user/knowledge-bases' },
  ]

  const items = isAdmin ? adminItems : userItems

  return (
    <nav className={cn('flex flex-col gap-0.5', className)}>
      {items.map((item) => {
        if (!isGroup(item)) {
          const Icon = item.icon
          return (
            <Link
              key={item.path}
              to={item.path}
              className={cn(
                'flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isActive(item.path)
                  ? 'bg-[var(--color-primary-soft)] text-[var(--color-primary)] font-semibold'
                  : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-text)]'
              )}
              onClick={onNavigate}
            >
              {Icon && <Icon className="size-4 shrink-0" />}
              <span className="truncate">{item.label}</span>
            </Link>
          )
        }

        const Icon = item.icon
        const active = isGroupActive(item)
        const isOpen = open[item.label] ?? active

        return (
          <Collapsible key={item.label} open={isOpen} onOpenChange={(v) => setOpen((p) => ({ ...p, [item.label]: v }))}>
            <CollapsibleTrigger className={cn(
              'flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              active
                ? 'bg-[var(--color-surface-2)] text-[var(--color-text)] font-semibold'
                : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-text)]'
            )}>
              <span className="flex items-center gap-2.5">
                <Icon className="size-4 shrink-0" />
                <span className="truncate">{item.label}</span>
              </span>
              <ChevronDown className={cn('size-3.5 shrink-0 transition-transform', isOpen && 'rotate-180')} />
            </CollapsibleTrigger>
            <CollapsibleContent className="ml-4 mt-0.5 flex flex-col gap-0.5 border-l border-[var(--color-line)] pl-3">
              {item.children.map((child) => (
                <Link
                  key={child.path}
                  to={child.path}
                  className={cn(
                    'rounded-md px-2 py-1.5 text-xs transition-colors truncate block',
                    isActive(child.path)
                      ? 'text-[var(--color-primary)] font-semibold'
                      : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
                  )}
                  onClick={onNavigate}
                >
                  {child.label}
                </Link>
              ))}
            </CollapsibleContent>
          </Collapsible>
        )
      })}
    </nav>
  )
}
