import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useLocaleStore } from '../stores/locale'
import zh from '../locales/zh.js'
import vi from '../locales/vi.js'
import en from '../locales/en.js'

const _lm = { zh, vi, en }
function _tl(key) {
  try { const s = useLocaleStore(); return _lm[s.lang]?.[key] ?? _lm.zh[key] ?? key } catch { return _lm.zh[key] ?? key }
}

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    const silentError = error.config?.silentError === true
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
    } else if (!silentError) {
      ElMessage.error(error.response?.data?.error || _tl('request_failed'))
    }
    return Promise.reject(error)
  }
)

export default api
