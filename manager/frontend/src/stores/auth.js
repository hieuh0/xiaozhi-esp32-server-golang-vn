import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'
import { useLocaleStore } from './locale'
import zh from '../locales/zh.js'
import vi from '../locales/vi.js'
import en from '../locales/en.js'

const _lm = { zh, vi, en }
function _tl(key) {
  try { const s = useLocaleStore(); return _lm[s.lang]?.[key] ?? _lm.zh[key] ?? key } catch { return _lm.zh[key] ?? key }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token'))
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
  const isValidating = ref(false) // 添加验证状态标记

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  const login = async (credentials) => {
    try {
      const response = await api.post('/login', credentials)
      const { token: newToken, user: userData } = response.data
      
      token.value = newToken
      user.value = userData
      
      localStorage.setItem('token', newToken)
      localStorage.setItem('user', JSON.stringify(userData))
      
      return { success: true, user: userData }
    } catch (error) {
      return { 
        success: false, 
        message: error.response?.data?.error || _tl('login_failed')
      }
    }
  }

  const register = async (userData) => {
    try {
      await api.post('/register', userData)
      return { success: true }
    } catch (error) {
      return { 
        success: false, 
        message: error.response?.data?.error || _tl('register_failed')
      }
    }
  }

  const logout = () => {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  const getProfile = async () => {
    // 如果正在验证中，避免重复调用
    if (isValidating.value) {
      return
    }
    
    isValidating.value = true
    try {
      const response = await api.get('/profile')
      user.value = response.data.user
      localStorage.setItem('user', JSON.stringify(response.data.user))
    } catch (error) {
      logout()
      throw error // 重新抛出错误，让路由守卫能够处理
    } finally {
      isValidating.value = false
    }
  }

  return {
    token,
    user,
    isAuthenticated,
    isAdmin,
    isValidating,
    login,
    register,
    logout,
    getProfile
  }
})