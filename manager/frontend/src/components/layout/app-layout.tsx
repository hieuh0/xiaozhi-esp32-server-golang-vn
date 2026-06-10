import { useState } from 'react'
import { Outlet } from '@tanstack/react-router'
import { SidebarBrand } from './sidebar-brand'
import { SidebarNavItems } from './sidebar-nav-items'
import { AppHeader } from './app-header'
import { MobileLayout } from './mobile-layout'
import { useIsMobile } from '@/hooks/use-is-mobile'
import { ScrollArea } from '@/components/ui/scroll-area'

export function AppLayout() {
  const isMobile = useIsMobile()
  const [sidebarOpen, setSidebarOpen] = useState(false)

  if (isMobile) return <MobileLayout />

  return (
    <div className="flex h-dvh overflow-hidden bg-[var(--color-bg)]">
      {/* Desktop sidebar */}
      <aside className="w-60 shrink-0 flex flex-col border-r border-[var(--color-line)] bg-[var(--color-surface-1)]">
        <SidebarBrand />
        <ScrollArea className="flex-1 min-h-0 py-3 px-2">
          <SidebarNavItems />
        </ScrollArea>
      </aside>

      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/40 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}
      <aside
        className={`fixed inset-y-0 left-0 z-50 w-60 flex flex-col border-r border-[var(--color-line)] bg-[var(--color-surface-1)] transition-transform duration-200 lg:hidden ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <SidebarBrand />
        <ScrollArea className="flex-1 min-h-0 py-3 px-2">
          <SidebarNavItems onNavigate={() => setSidebarOpen(false)} />
        </ScrollArea>
      </aside>

      {/* Main content */}
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        <AppHeader onToggleSidebar={() => setSidebarOpen((v) => !v)} />
        <main className="flex-1 min-h-0 overflow-y-auto" style={{ overflowX: 'clip' }}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}
