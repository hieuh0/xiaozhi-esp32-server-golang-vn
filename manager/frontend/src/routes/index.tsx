import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuthStore } from '@/stores/auth'

export const Route = createFileRoute('/')({
  beforeLoad: () => {
    const { user } = useAuthStore.getState()
    const dest = user?.role === 'admin' ? '/dashboard' : '/agents'
    throw redirect({ to: dest })
  },
})
