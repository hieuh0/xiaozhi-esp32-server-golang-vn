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
        // Check if system needs initialization
        const response = await api.get('/setup/status')

        if (response.data.needs_setup) {
          // If setup is needed and not already on the setup page, redirect
          if (router.currentRoute.value.path !== '/setup') {
            router.push('/setup')
          }
        }
      } catch (error) {
        console.error(t('check_system_failed'), error)
        // Check failed (likely network issue) — do not force redirect
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


/* Mobile style optimizations */
@media (max-width: 767px) {
  /* Mobile font size optimization */
  body {
    font-size: 14px;
    -webkit-text-size-adjust: 100%;
    -webkit-tap-highlight-color: transparent;
  }

  /* Mobile scroll optimization */
  * {
    -webkit-overflow-scrolling: touch;
  }

  /* Mobile tap delay optimization */
  a, button, input, textarea {
    touch-action: manipulation;
  }

  /* Hide desktop-only elements */
  .desktop-only {
    display: none !important;
  }
}

/* Desktop styles */
@media (min-width: 768px) {
  /* Hide mobile-only elements */
  .mobile-only {
    display: none !important;
  }
}

/* Global animations */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.22s ease, transform 0.22s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

/* Mobile safe-area insets */
@supports (padding: max(0px)) {
  .mobile-safe-top {
    padding-top: max(20px, env(safe-area-inset-top));
  }
  
  .mobile-safe-bottom {
    padding-bottom: max(20px, env(safe-area-inset-bottom));
  }
}
</style>
