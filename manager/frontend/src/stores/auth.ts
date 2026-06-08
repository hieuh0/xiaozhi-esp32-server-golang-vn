import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import api from '@/utils/api'

interface User {
  username: string
  role: string
  [key: string]: unknown
}

interface AuthState {
  token: string | null
  user: User | null
  isAdmin: boolean
  isValidating: boolean
  login: (credentials: { username: string; password: string; captchaId?: string; captchaAnswer?: string }) => Promise<{ success: boolean; user?: User; message?: string }>
  register: (data: { username: string; email: string; password: string; captchaId?: string; captchaAnswer?: string }) => Promise<{ success: boolean; message?: string }>
  logout: () => void
  getProfile: () => Promise<void>
}

let _validationPromise: Promise<void> | null = null

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      isAdmin: false,
      isValidating: false,

      login: async (credentials) => {
        try {
          const response = await api.post<{ token: string; user: User }>('/login', credentials)
          const { token, user } = response.data
          set({ token, user, isAdmin: user.role === 'admin' })
          localStorage.setItem('token', token)
          return { success: true, user }
        } catch (error: unknown) {
          const err = error as { response?: { data?: { error?: string } } }
          return { success: false, message: err.response?.data?.error || 'Login failed' }
        }
      },

      register: async (data) => {
        try {
          await api.post('/register', data)
          return { success: true }
        } catch (error: unknown) {
          const err = error as { response?: { data?: { error?: string } } }
          return { success: false, message: err.response?.data?.error || 'Registration failed' }
        }
      },

      logout: () => {
        set({ token: null, user: null, isAdmin: false })
        localStorage.removeItem('token')
        localStorage.removeItem('user')
      },

      getProfile: async () => {
        if (_validationPromise) return _validationPromise
        _validationPromise = (async () => {
          set({ isValidating: true })
          try {
            const response = await api.get<{ user: User }>('/profile')
            const user = response.data.user
            set({ user, isAdmin: user.role === 'admin' })
          } catch {
            get().logout()
            throw new Error('Session expired')
          } finally {
            set({ isValidating: false })
            _validationPromise = null
          }
        })()
        return _validationPromise
      },
    }),
    {
      name: 'auth',
      partialize: (state) => ({ token: state.token, user: state.user, isAdmin: state.isAdmin }),
    }
  )
)
