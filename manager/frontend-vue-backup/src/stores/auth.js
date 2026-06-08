import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'
import { tl } from '../utils/i18n-helper'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token'))
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
  const isValidating = ref(false) // Validation in-progress flag

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
        message: error.response?.data?.error || tl('login_failed')
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
        message: error.response?.data?.error || tl('register_failed')
      }
    }
  }

  const logout = () => {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  let _validationPromise = null

  const getProfile = async () => {
    if (_validationPromise) return _validationPromise

    _validationPromise = (async () => {
      isValidating.value = true
      try {
        const response = await api.get('/profile')
        user.value = response.data.user
        localStorage.setItem('user', JSON.stringify(response.data.user))
      } catch (error) {
        logout()
        throw error
      } finally {
        isValidating.value = false
        _validationPromise = null
      }
    })()

    return _validationPromise
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