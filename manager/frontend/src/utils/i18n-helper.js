import { useLocaleStore } from '../stores/locale'
import zh from '../locales/zh.js'
import vi from '../locales/vi.js'
import en from '../locales/en.js'

const _lm = { zh, vi, en }

/**
 * Translate a key outside Vue component context (non-reactive, safe for interceptors/stores).
 * Falls back to zh, then the key itself. Supports {param} interpolation via params object.
 */
export function tl(key, params) {
  try {
    const s = useLocaleStore()
    let str = _lm[s.lang]?.[key] ?? _lm.zh[key] ?? key
    if (params) Object.entries(params).forEach(([k, v]) => { str = str.replaceAll(`{${k}}`, v) })
    return str
  } catch {
    return _lm.zh[key] ?? key
  }
}
