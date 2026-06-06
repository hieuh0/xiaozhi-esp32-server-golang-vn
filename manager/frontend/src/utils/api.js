import axios from 'axios'
import { ElMessage } from 'element-plus'
import { tl } from './i18n-helper'

let _router = null

export function setRouter(router) {
  _router = router
}

const api = axios.create({
  baseURL: '/api',
  timeout: 10000
})

// Request interceptor
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

// Response interceptor
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    const silentError = error.config?.silentError === true
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (_router) {
        _router.push('/login')
      } else {
        window.location.href = '/login'
      }
    } else if (!silentError) {
      ElMessage.error(error.response?.data?.error || tl('request_failed'))
    }
    return Promise.reject(error)
  }
)

export default api
