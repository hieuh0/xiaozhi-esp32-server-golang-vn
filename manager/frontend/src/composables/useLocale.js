import { computed } from 'vue'
import { useLocaleStore } from '../stores/locale'
import zh from '../locales/zh.js'
import vi from '../locales/vi.js'
import en from '../locales/en.js'

const maps = { zh, vi, en }

export function useLocale() {
  const store = useLocaleStore()
  // Pinia setup stores unwrap refs — store.lang is already a string, not a ref
  // Fallback chain: currentLang → zh → key itself
  const t = (key) => maps[store.lang]?.[key] ?? maps.zh[key] ?? key
  return {
    t,
    lang: computed(() => store.lang),
    setLang: store.setLang,
  }
}
