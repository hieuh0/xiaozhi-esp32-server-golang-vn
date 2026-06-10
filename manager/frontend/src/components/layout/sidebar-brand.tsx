import appLogo from '@/assets/brand/app-logo.webp'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'

export function SidebarBrand() {
  const { t } = useLocale()
  const isAdmin = useAuthStore((s) => s.isAdmin)

  return (
    <div className="flex items-center gap-2.5 px-4 h-16 border-b border-[var(--color-line)] shrink-0">
      <img
        src={appLogo}
        alt={t('xiaozhi_management_system')}
        className="size-9 rounded-xl object-cover shrink-0 ring-2 ring-[var(--color-primary)]/20"
      />
      <div className="min-w-0">
        <p className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-primary)]">
          Control Center
        </p>
        <p className="truncate text-sm font-semibold text-[var(--color-text)]">
          {t('xiaozhi_management_system')}
        </p>
        <p className="truncate text-xs text-[var(--color-text-secondary)]">
          {isAdmin ? t('admin_panel_title') : t('device_agent_workbench')}
        </p>
      </div>
    </div>
  )
}
