const UA_MOBILE = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i
const UA_TABLET = /iPad|Android/i

export const isMobile = (): boolean => {
  const ua = navigator.userAgent || navigator.vendor
  return UA_MOBILE.test(ua) || window.innerWidth < 768
}

export const isTablet = (): boolean => {
  const ua = navigator.userAgent || navigator.vendor
  return UA_TABLET.test(ua) && window.innerWidth >= 768 && window.innerWidth < 1024
}

export const isDesktop = (): boolean => !isMobile() && !isTablet()

export const getDeviceType = (): 'mobile' | 'tablet' | 'desktop' => {
  if (isMobile()) return 'mobile'
  if (isTablet()) return 'tablet'
  return 'desktop'
}

export const onResize = (callback: (type: 'mobile' | 'tablet' | 'desktop') => void): (() => void) => {
  let ticking = false
  const handler = () => {
    if (!ticking) {
      window.requestAnimationFrame(() => { callback(getDeviceType()); ticking = false })
      ticking = true
    }
  }
  window.addEventListener('resize', handler)
  return () => window.removeEventListener('resize', handler)
}
