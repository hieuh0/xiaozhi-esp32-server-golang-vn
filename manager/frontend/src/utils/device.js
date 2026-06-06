/**
 * Device detection utilities
 * Determines current device type for responsive layout
 */

/**
 * Returns true if the current device is mobile
 * @returns {boolean}
 */
export const isMobile = () => {
  // Check via User-Agent
  const userAgent = navigator.userAgent || navigator.vendor || window.opera
  const mobileRegex = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i
  const isMobileUA = mobileRegex.test(userAgent)

  // Fallback: check screen width
  const isMobileWidth = window.innerWidth < 768
  
  return isMobileUA || isMobileWidth
}

/**
 * Returns true if the current device is a tablet
 * @returns {boolean}
 */
export const isTablet = () => {
  const userAgent = navigator.userAgent || navigator.vendor || window.opera
  return /iPad|Android/i.test(userAgent) && window.innerWidth >= 768 && window.innerWidth < 1024
}

/**
 * Returns true if the current device is desktop
 * @returns {boolean}
 */
export const isDesktop = () => {
  return !isMobile() && !isTablet()
}

/**
 * Returns true if running inside WeChat browser
 * @returns {boolean}
 */
export const isWeChat = () => {
  const userAgent = navigator.userAgent || ''
  return /MicroMessenger/i.test(userAgent)
}

/**
 * Returns the current device type
 * @returns {'mobile' | 'tablet' | 'desktop'}
 */
export const getDeviceType = () => {
  if (isMobile()) {
    return 'mobile'
  } else if (isTablet()) {
    return 'tablet'
  } else {
    return 'desktop'
  }
}

/**
 * Listen to window resize events
 * @param {Function} callback - Called with the new device type on resize
 * @returns {Function} Unsubscribe function
 */
export const onResize = (callback) => {
  let ticking = false
  
  const handler = () => {
    if (!ticking) {
      window.requestAnimationFrame(() => {
        callback(getDeviceType())
        ticking = false
      })
      ticking = true
    }
  }
  
  window.addEventListener('resize', handler)
  
  // Return unsubscribe function
  return () => {
    window.removeEventListener('resize', handler)
  }
}
