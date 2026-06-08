import { createFileRoute } from '@tanstack/react-router'
import { AppLayout } from '@/components/layout/app-layout'

export const Route = createFileRoute('/_auth/_layout')({
  component: AppLayout,
})
