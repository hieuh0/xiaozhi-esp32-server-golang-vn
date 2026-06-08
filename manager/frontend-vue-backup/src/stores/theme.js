import { defineStore } from 'pinia'

const prefersDark = () => window.matchMedia('(prefers-color-scheme: dark)').matches

function applyTheme(mode) {
  const html = document.documentElement
  html.classList.remove('dark', 'light')
  if (mode === 'dark') html.classList.add('dark')
  else if (mode === 'light') html.classList.add('light')
  // auto: mirror system preference so Tailwind dark: classes also work
  else if (prefersDark()) html.classList.add('dark')
}

export const useThemeStore = defineStore('theme', {
  state: () => ({
    mode: 'auto' // 'auto' | 'light' | 'dark'
  }),
  actions: {
    setMode(mode) {
      this.mode = mode
      localStorage.setItem('theme-mode', mode)
      applyTheme(mode)
    },
    nextMode() {
      const cycle = { dark: 'auto', auto: 'light', light: 'dark' }
      this.setMode(cycle[this.mode])
    },
    init() {
      const saved = localStorage.getItem('theme-mode') || 'auto'
      this.mode = saved
      applyTheme(saved)
      // Re-sync .dark class when system preference changes while in auto mode
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (this.mode === 'auto') applyTheme('auto')
      })
    }
  }
})
