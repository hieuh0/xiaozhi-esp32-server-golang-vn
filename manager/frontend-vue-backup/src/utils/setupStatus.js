import api from './api'

export const checkNeedsSetup = async () => {
  const response = await api.get('/setup/status', { silentError: true })
  return Boolean(response?.data?.needs_setup)
}
