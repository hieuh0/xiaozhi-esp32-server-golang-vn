import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import { Toaster } from 'sonner'
import { routeTree } from './routeTree.gen'
import { setNavigate, setLogout } from '@/utils/api'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { AppQueryProvider } from '@/providers/query-provider'
import './styles/globals.css'
import './i18n'

const router = createRouter({ routeTree })

setNavigate((path) => router.navigate({ to: path as '/' }))
setLogout(() => useAuthStore.getState().logout())
useThemeStore.getState().init()

declare module '@tanstack/react-router' {
  interface Register { router: typeof router }
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppQueryProvider>
      <RouterProvider router={router} />
      <Toaster richColors position="top-right" />
    </AppQueryProvider>
  </StrictMode>,
)
