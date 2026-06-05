// 前端诊断脚本
console.log('=== 前端诊断开始 ===')

// 检查基本环境
console.log('1. 检查基本环境:')
console.log('   - Vue版本:', typeof window.Vue !== 'undefined' ? 'Vue已加载' : 'Vue未加载')
console.log('   - 当前URL:', window.location.href)
console.log('   - User Agent:', navigator.userAgent)

// 检查localStorage
console.log('2. 检查本地存储:')
console.log('   - Token:', localStorage.getItem('token'))
console.log('   - User:', localStorage.getItem('user'))

// 检查网络连接
console.log('3. 检查后端连接:')
fetch('http://localhost:8080/api/profile')
  .then(response => {
    console.log('   - 后端响应状态:', response.status)
    if (response.status === 401) {
      console.log('   - 后端正常运行（返回401未认证错误）')
    }
  })
  .catch(error => {
    console.log('   - 后端连接失败:', error.message)
  })

// 检查路由
console.log('4. 可用的测试路由:')
console.log('   - /test - 基础测试页面')
console.log('   - /simple-login - 简化登录页面')
console.log('   - /login - 完整登录页面')

console.log('=== 前端诊断结束 ===')
console.log('请在浏览器控制台中查看上述信息')

// 导出到全局，方便在控制台中调用
window.diagnose = () => {
  console.clear()
  // 重新执行诊断
  location.reload()
}

