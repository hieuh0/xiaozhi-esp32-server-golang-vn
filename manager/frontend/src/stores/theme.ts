import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ThemeMode = 'light' | 'dark' | 'auto'

const CYCLE: Record<ThemeMode, ThemeMode> = { light: 'dark', dark: 'auto', auto: 'light' }

function applyTheme(mode: ThemeMode) {
  const html = document.documentElement
  html.classList.remove('dark', 'light')
  if (mode === 'dark') html.classList.add('dark')
  else if (mode === 'light') html.classList.add('light')
  else if (window.matchMedia('(prefers-color-scheme: dark)').matches) html.classList.add('dark')
}

interface ThemeState {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
  nextMode: () => void
  init: () => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      mode: 'auto' as ThemeMode,

      setMode: (mode) => {
        set({ mode })
        applyTheme(mode)
      },

      nextMode: () => {
        get().setMode(CYCLE[get().mode])
      },

      init: () => {
        applyTheme(get().mode)
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
          if (get().mode === 'auto') applyTheme('auto')
        })
      },
    }),
    { name: 'theme-mode', partialize: (s) => ({ mode: s.mode }) }
  )
)
