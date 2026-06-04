<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/utils/api'

export default {
  name: 'App',
  setup() {
    const router = useRouter()

    const checkSystemStatus = async () => {
      try {
        // 检查系统是否需要初始化
        const response = await api.get('/setup/status')
        
        if (response.data.needs_setup) {
          // 如果需要初始化且当前不在引导页面，则跳转到引导页面
          if (router.currentRoute.value.path !== '/setup') {
            router.push('/setup')
          }
        }
      } catch (error) {
        console.error(t('check_system_failed'), error)
        // 如果检查失败，可能是网络问题，不强制跳转
      }
    }

    onMounted(() => {
      checkSystemStatus()
    })
  }
}
</script>

<style>
#app {
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  height: 100dvh;
}

html,
body {
  height: 100%;
}

body {
  margin: 0;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}


/* 移动端样式优化 */
@media (max-width: 767px) {
  /* 移动端字体大小优化 */
  body {
    font-size: 14px;
    -webkit-text-size-adjust: 100%;
    -webkit-tap-highlight-color: transparent;
  }
  
  /* 移动端滚动优化 */
  * {
    -webkit-overflow-scrolling: touch;
  }
  
  /* 移动端点击延迟优化 */
  a, button, input, textarea {
    touch-action: manipulation;
  }
  
  /* 隐藏桌面端元素 */
  .desktop-only {
    display: none !important;
  }
}

/* 桌面端样式 */
@media (min-width: 768px) {
  /* 隐藏移动端元素 */
  .mobile-only {
    display: none !important;
  }
}

/* 全局动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.22s ease, transform 0.22s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

/* 移动端安全区域适配 */
@supports (padding: max(0px)) {
  .mobile-safe-top {
    padding-top: max(20px, env(safe-area-inset-top));
  }
  
  .mobile-safe-bottom {
    padding-bottom: max(20px, env(safe-area-inset-bottom));
  }
}
</style>
