import { useState } from 'react'
import { Outlet, Link, useRouterState, useRouter } from '@tanstack/react-router'
import { ChevronLeft, User, Home, Settings, Users, MoreHorizontal, Bot, Mic } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'
import { getPageTitle } from '@/utils/page-title'

const HIDE_BACK = ['/dashboard', '/agents', '/user/speakers', '/more', '/login']
const HIDE_TAB = ['/login', '/setup']

function MobileNavBar({ title, showBack }: { title: string; showBack: boolean; onUserClick: () => void }) {
  const router = useRouter()
  const { onUserClick: _unused, ..._ } = { onUserClick: () => {} }
  void _unused; void _
  return (
    <header className="flex items-center h-12 px-3 mx-3 mt-3 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface-1)] sticky top-3 z-10 shadow-sm">
      {showBack && (
        <button
          type="button"
          onClick={() => router.history.back()}
          className="flex items-center justify-center min-w-[44px] h-full -ml-1 mr-1 rounded-xl text-[var(--color-text)]"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>
      )}
      <h1 className="flex-1 text-base font-bold text-[var(--color-text)] truncate">{title}</h1>
    </header>
  )
}

function MobileTabBar() {
  const { t } = useLocale()
  const isAdmin = useAuthStore((s) => s.isAdmin)
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const isActive = (path: string) => pathname === path || pathname.startsWith(path + '/')

  const tabs = isAdmin
    ? [
        { path: '/dashboard', label: t('home'), icon: Home },
        { path: '/admin/config-overview', label: t('config'), icon: Settings },
        { path: '/admin/users', label: t('manage'), icon: Users },
        { path: '/more', label: t('more'), icon: MoreHorizontal },
      ]
    : [
        { path: '/agents', label: t('agent'), icon: Bot },
        { path: '/user/speakers', label: t('voiceprint'), icon: Mic },
        { path: '/more', label: t('more'), icon: MoreHorizontal },
      ]

  return (
    <nav
      className="mx-3 mb-3 flex rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface-1)] overflow-hidden shadow-sm"
      style={{ paddingBottom: 'env(safe-area-inset-bottom, 0px)' }}
    >
      {tabs.map((tab) => {
        const Icon = tab.icon
        return (
          <Link
            key={tab.path}
            to={tab.path}
            className={cn(
              'flex-1 flex flex-col items-center justify-center py-3 gap-1 transition-colors',
              isActive(tab.path)
                ? 'text-[var(--color-primary)]'
                : 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text)]'
            )}
          >
            <Icon className="w-5 h-5" />
            <span className="text-[10px] font-medium">{tab.label}</span>
          </Link>
        )
      })}
    </nav>
  )
}

export function MobileLayout() {
  const { t } = useLocale()
  const { user, isAdmin, logout } = useAuthStore()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const [showUserMenu, setShowUserMenu] = useState(false)
  const router = useRouter()

  const showBack = !HIDE_BACK.some((p) => pathname === p || pathname.startsWith(p + '/'))
  const showTab = !HIDE_TAB.includes(pathname) &&
    !pathname.includes('/edit') && !pathname.includes('/detail') && !pathname.includes('/history')

  const pageTitle = getPageTitle(pathname, isAdmin, t)

  const handleLogout = () => {
    logout()
    router.navigate({ to: '/login' })
    setShowUserMenu(false)
  }

  return (
    <div className="flex flex-col h-dvh bg-[var(--color-bg)] overflow-hidden">
      <div className="flex items-center">
        <div className="flex-1">
          <MobileNavBar title={pageTitle} showBack={showBack} onUserClick={() => setShowUserMenu(true)} />
        </div>
        <button
          type="button"
          onClick={() => setShowUserMenu(true)}
          className="flex items-center justify-center w-9 h-9 rounded-xl bg-blue-50 text-[var(--color-primary)] dark:bg-blue-900/30 mr-4 mt-3 shrink-0"
        >
          <User className="w-4 h-4" />
        </button>
      </div>

      <main className="flex-1 min-h-0 overflow-y-auto overscroll-contain">
        <Outlet />
      </main>

      {showTab && <MobileTabBar />}

      {/* User menu bottom sheet */}
      <Sheet open={showUserMenu} onOpenChange={setShowUserMenu}>
        <SheetContent side="bottom" className="rounded-t-3xl px-0 pb-0">
          <div className="flex items-center gap-4 px-5 py-5 border-b border-[var(--color-line)]">
            <div className="flex items-center justify-center w-14 h-14 rounded-2xl bg-blue-50 dark:bg-blue-900/30 text-[var(--color-primary)] shrink-0">
              <User className="w-7 h-7" />
            </div>
            <div>
              <div className="text-lg font-bold text-[var(--color-text)]">{user?.username}</div>
              <div className="text-sm text-[var(--color-text-secondary)]">
                {isAdmin ? t('admin') : t('normal_user')}
              </div>
            </div>
          </div>
          <div className="divide-y divide-[var(--color-line)]">
            <button type="button" onClick={() => { router.navigate({ to: '/more' }); setShowUserMenu(false) }}
              className="flex items-center w-full px-5 py-4 text-sm text-[var(--color-text)] hover:bg-[var(--color-surface-2)] transition-colors text-left">
              {t('more_features')}<span className="ml-auto text-[var(--color-text-tertiary)] text-base">›</span>
            </button>
            {!isAdmin && (
              <button type="button" onClick={() => { router.navigate({ to: '/user/api-tokens' }); setShowUserMenu(false) }}
                className="flex items-center w-full px-5 py-4 text-sm text-[var(--color-text)] hover:bg-[var(--color-surface-2)] transition-colors text-left">
                API Token<span className="ml-auto text-[var(--color-text-tertiary)] text-base">›</span>
              </button>
            )}
            {isAdmin && (
              <button type="button" onClick={() => { router.navigate({ to: '/admin/config-wizard' }); setShowUserMenu(false) }}
                className="flex items-center w-full px-5 py-4 text-sm text-[var(--color-text)] hover:bg-[var(--color-surface-2)] transition-colors text-left">
                {t('config_wizard')}<span className="ml-auto text-[var(--color-text-tertiary)] text-base">›</span>
              </button>
            )}
            <button type="button" onClick={handleLogout}
              className="flex items-center w-full px-5 py-4 text-sm text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors text-left">
              {t('logout')}
            </button>
          </div>
          <div style={{ height: 'max(16px, env(safe-area-inset-bottom))' }} />
        </SheetContent>
      </Sheet>
    </div>
  )
}
