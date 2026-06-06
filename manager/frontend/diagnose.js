// Frontend diagnostics script
console.log('=== Frontend Diagnostics Start ===')

// Check basic environment
console.log('1. Check basic environment:')
console.log('   - Vue version:', typeof window.Vue !== 'undefined' ? 'Vue loaded' : 'Vue not loaded')
console.log('   - Current URL:', window.location.href)
console.log('   - User Agent:', navigator.userAgent)

// Check localStorage
console.log('2. Check local storage:')
console.log('   - Token:', localStorage.getItem('token'))
console.log('   - User:', localStorage.getItem('user'))

// Check network connection
console.log('3. Check backend connection:')
fetch('http://localhost:8080/api/profile')
  .then(response => {
    console.log('   - Backend response status:', response.status)
    if (response.status === 401) {
      console.log('   - Backend running normally (returned 401 unauthenticated)')
    }
  })
  .catch(error => {
    console.log('   - Backend connection failed:', error.message)
  })

// Check routes
console.log('4. Available test routes:')
console.log('   - /test - Basic test page')
console.log('   - /simple-login - Simplified login page')
console.log('   - /login - Full login page')

console.log('=== Frontend Diagnostics End ===')
console.log('Check the above information in the browser console')

// Export to global for easy console invocation
window.diagnose = () => {
  console.clear()
  // Re-run diagnostics
  location.reload()
}
