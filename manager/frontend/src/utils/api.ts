import axios from 'axios'

let _navigate: ((path: string) => void) | null = null

/** Call once after router is created to enable programmatic 401 redirect */
export function setNavigate(fn: (path: string) => void) {
  _navigate = fn
}

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (_navigate) _navigate('/login')
      else window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api
