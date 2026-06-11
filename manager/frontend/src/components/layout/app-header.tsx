import { Menu, Sun, Moon, Monitor, ChevronDown, Wand2, Upload } from 'lucide-react'
import { Link, useRouterState } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem,
} from '@/components/ui/dropdown-menu'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useLocale } from '@/hooks/use-locale'
import { getPageTitle } from '@/utils/page-title'

const LANG_LABELS: Record<string, string> = { vi: '🇻🇳 VI', en: '🇬🇧 EN', zh: '🇨🇳 ZH' }
const THEME_ARIA: Record<string, string> = { dark: 'Switch to auto', auto: 'Switch to light', light: 'Switch to dark' }

interface Props {
  onToggleSidebar: () => void
}

export function AppHeader({ onToggleSidebar }: Props) {
  const { t, lang, setLang } = useLocale()
  const { mode, nextMode } = useThemeStore()
  const { user, isAdmin, logout } = useAuthStore()
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const initial = (user?.username || 'U').slice(0, 1).toUpperCase()
  const roleLabel = isAdmin ? t('admin') : t('normal_user')
  const eyebrow = isAdmin ? 'Admin Console' : 'User Workspace'

  const pageTitle = getPageTitle(pathname, isAdmin, t)

  const handleLogout = () => {
    logout()
    window.location.href = '/login'
  }

  return (
    <header className="flex items-center justify-between gap-4 px-5 h-16 shrink-0 border-b border-[var(--color-line)] bg-[var(--color-surface-1)]" style={{ boxShadow: '0 1px 0 var(--color-line)' }}>
      {/* Left */}
      <div className="flex items-center gap-3 min-w-0">
        <Button variant="ghost" size="icon" className="lg:hidden" aria-label="Toggle menu" onClick={onToggleSidebar}>
          <Menu className="size-5" />
        </Button>
        <div className="min-w-0">
          <p className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-primary)] font-mono">{eyebrow}</p>
          <h1 className="text-lg font-semibold font-display tracking-tight text-[var(--color-text)] leading-tight truncate">{pageTitle}</h1>
        </div>
      </div>

      {/* Right */}
      <div className="flex items-center gap-2 shrink-0">
        {/* Admin shortcuts */}
        {isAdmin && (
          <>
            <Link to="/admin/config-wizard">
              <Button variant="ghost" size="sm">
                <Wand2 className="size-4" /><span className="hidden xl:inline ml-1">{t('config_wizard')}</span>
              </Button>
            </Link>
            <Link to="/admin/ota-config">
              <Button variant="ghost" size="sm">
                <Upload className="size-4" /><span className="hidden xl:inline ml-1">{t('ota_config')}</span>
              </Button>
            </Link>
          </>
        )}

        {/* Theme toggle */}
        <Button variant="ghost" size="icon" aria-label={THEME_ARIA[mode]} onClick={nextMode}>
          {mode === 'light' ? <Sun className="size-4" /> : mode === 'dark' ? <Moon className="size-4" /> : <Monitor className="size-4" />}
        </Button>

        {/* Language switcher */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="gap-1">
              {LANG_LABELS[lang] ?? '🇻🇳 VI'}<ChevronDown className="size-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => setLang('vi')}>🇻🇳 Tiếng Việt</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setLang('en')}>🇬🇧 English</DropdownMenuItem>
            <DropdownMenuItem onClick={() => setLang('zh')}>🇨🇳 Chinese</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Profile dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              className="flex items-center gap-2 rounded-full border border-[var(--color-line)] bg-[var(--color-surface-1)] px-2.5 py-1.5 text-sm transition-colors hover:bg-[var(--color-surface-2)]"
            >
              <span className="flex size-7 items-center justify-center rounded-full bg-[var(--color-primary-soft)] text-xs font-bold text-[var(--color-primary)]">
                {initial}
              </span>
              <span className="hidden sm:flex flex-col items-start leading-tight">
                <strong className="text-xs font-semibold text-[var(--color-text)]">{user?.username}</strong>
                <small className="text-[10px] text-[var(--color-text-secondary)]">{roleLabel}</small>
              </span>
              <ChevronDown className="size-3 text-[var(--color-text-tertiary)]" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {!isAdmin && (
              <DropdownMenuItem asChild>
                <Link to="/user/api-tokens">API Token</Link>
              </DropdownMenuItem>
            )}
            <DropdownMenuItem className="text-[var(--color-danger)]" onClick={handleLogout}>
              {t('logout')}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
