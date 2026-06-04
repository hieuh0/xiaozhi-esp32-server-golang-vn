import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useLocaleStore = defineStore('locale', () => {
  const lang = ref(localStorage.getItem('lang') || 'vi')
  const setLang = (l) => {
    lang.value = l
    localStorage.setItem('lang', l)
  }
  return { lang, setLang }
})
