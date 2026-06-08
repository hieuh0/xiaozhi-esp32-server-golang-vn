import { useTranslation } from 'react-i18next'
import i18n from '@/i18n'

export function useLocale() {
  const { t } = useTranslation()
  const lang = i18n.language
  const setLang = (l: string) => {
    i18n.changeLanguage(l)
    localStorage.setItem('lang', l)
  }
  return { t, lang, setLang }
}
