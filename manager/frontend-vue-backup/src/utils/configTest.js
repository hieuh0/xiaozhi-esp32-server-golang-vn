import api from './api'
import { useLocaleStore } from '../stores/locale'
import zh from '../locales/zh.js'
import vi from '../locales/vi.js'
import en from '../locales/en.js'

const localeMaps = { zh, vi, en }
function tl(key) {
  try {
    const store = useLocaleStore()
    return localeMaps[store.lang]?.[key] ?? localeMaps.zh[key] ?? key
  } catch {
    return localeMaps.zh[key] ?? key
  }
}

/** Normalize a response entry into a unified result (with first_packet_ms) */
function normItem(item) {
  if (!item || typeof item !== 'object') return { ok: false, message: '', first_packet_ms: undefined, reasoning_content_returned: false }
  const ms = item.first_packet_ms
  return {
    ok: !!item.ok,
    message: item.message || '',
    first_packet_ms: typeof ms === 'number' ? ms : (ms != null ? Number(ms) : undefined),
    reasoning_content_returned: !!item.reasoning_content_returned
  }
}

/**
 * Test a single config or all configs of a type
 * @param {string} type - Type: ota | vad | asr | llm | tts
 * @param {string} [configId] - Optional; if given, only that config_id is tested
 * @returns {Promise<{ ok: boolean, message: string, first_packet_ms?: number }>} Single result, or first/summary for multiple
 */
export async function testSingleConfig(type, configId) {
  const body = {
    types: [type],
    config_ids: configId ? { [type]: [configId] } : {}
  }
  const res = await api.post('/admin/configs/test', body, { timeout: 30000 })
  const data = res.data?.data ?? res.data
  const typeResult = data?.[type]
  if (!typeResult || typeof typeResult !== 'object') {
    return { ok: false, message: tl('no_test_result') }
  }
  const entries = Object.entries(typeResult).filter(([k]) => !k.startsWith('_'))
  if (configId && typeResult[configId]) {
    return normItem(typeResult[configId])
  }
  if (entries.length === 0) {
    const err = typeResult._error || typeResult._no_client || typeResult._none
    const msg = err && typeof err === 'object' ? (err.message || '').trim() : ''
    const fallback = typeResult._none ? tl('not_configured_or_disabled') : tl('no_results')
    return { ok: false, message: msg || fallback }
  }
  return normItem(entries[0][1])
}

/**
 * Test all configs of a type; returns results keyed by config_id (for “test all” row display)
 * @param {string} type - Type: vad | asr | llm | tts
 * @returns {Promise<Record<string, { ok: boolean, message: string, first_packet_ms?: number }>>} config_id -> { ok, message, first_packet_ms? }
 */
export async function testAllConfigs(type) {
  const body = { types: [type] }
  const res = await api.post('/admin/configs/test', body, { timeout: 60000 })
  const data = res.data?.data ?? res.data
  const typeResult = data?.[type]
  const out = {}
  if (!typeResult || typeof typeResult !== 'object') {
    return out
  }
  const err = typeResult._error || typeResult._no_client || typeResult._none
  const errMsg = err && typeof err === 'object' ? (err.message || '').trim() : tl('no_test_result')
  for (const [k, v] of Object.entries(typeResult)) {
    if (k.startsWith('_')) continue
    out[k] = normItem(v)
  }
  if (Object.keys(out).length === 0 && errMsg) {
    out._global = { ok: false, message: errMsg }
  }
  return out
}

/**
 * Convert getJsonData() return value to a plain object (form returns JSON string)
 * @param {string|object} jsonData - Value returned by getJsonData()
 * @returns {object}
 */
export function parseJsonData(jsonData) {
  if (jsonData == null) return {}
  if (typeof jsonData === 'object') return jsonData
  if (typeof jsonData !== 'string') return {}
  try {
    return JSON.parse(jsonData) || {}
  } catch {
    return {}
  }
}

/**
 * Test with custom data (unsaved draft / current wizard step)
 * @param {string} type - Type: ota | vad | asr | llm | tts
 * @param {Record<string, object>} typeData - config_id -> config object for this type, matching data[type] shape
 * @returns {Promise<{ ok: boolean, message: string, first_packet_ms?: number }>} Single result (only one config supported)
 */
export async function testWithData(type, typeData) {
  const body = { types: [type], data: { [type]: typeData } }
  const res = await api.post('/admin/configs/test', body, { timeout: 30000 })
  const data = res.data?.data ?? res.data
  const typeResult = data?.[type]
  if (!typeResult || typeof typeResult !== 'object') {
    return { ok: false, message: tl('no_test_result') }
  }
  const err = typeResult._error || typeResult._no_client
  if (err && typeof err === 'object' && err.message) {
    return { ok: false, message: err.message }
  }
  const entries = Object.entries(typeResult).filter(([k]) => !k.startsWith('_'))
  if (entries.length === 0) {
    return { ok: false, message: typeResult._none?.message || tl('no_results') }
  }
  return normItem(entries[0][1])
}
