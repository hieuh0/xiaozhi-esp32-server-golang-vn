import appLogo from '@/assets/brand/app-logo.svg'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'

export function SidebarBrand() {
  const { t } = useLocale()
  const isAdmin = useAuthStore((s) => s.isAdmin)

  return (
    <div className="flex items-center gap-2.5 px-4 h-16 border-b border-[var(--color-line)] shrink-0">
      <div
        className="size-9 rounded-xl flex items-center justify-center shrink-0 overflow-hidden"
        style={{ background: 'var(--color-primary-soft)', border: '1px solid color-mix(in srgb, var(--color-primary) 30%, transparent)' }}
      >
        <img
          src={appLogo}
          alt={t('xiaozhi_management_system')}
          className="size-7 object-cover"
        />
      </div>
      <div className="min-w-0">
        <p className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-primary)] font-mono leading-none mb-0.5">
          Control Center
        </p>
        <p className="truncate text-sm font-semibold text-[var(--color-text)] leading-tight">
          {t('xiaozhi_management_system')}
        </p>
        <p className="truncate text-[11px] text-[var(--color-text-tertiary)] leading-tight">
          {isAdmin ? t('admin_panel_title') : t('device_agent_workbench')}
        </p>
      </div>
    </div>
  )
}
