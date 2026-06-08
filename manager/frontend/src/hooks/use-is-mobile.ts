import { useState, useEffect } from 'react'
import { isMobile } from '@/utils/device'

export function useIsMobile(): boolean {
  const [mobile, setMobile] = useState(() => isMobile())
  useEffect(() => {
    const handler = () => setMobile(isMobile())
    window.addEventListener('resize', handler)
    return () => window.removeEventListener('resize', handler)
  }, [])
  return mobile
}
