import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { isMobile } from '../utils/device'

// Dynamically load login component based on device type
const getLoginComponent = () => {
  return isMobile()
    ? import('../views/mobile/MobileLogin.vue')
    : import('../views/Login.vue')
}

const routes = [
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('../views/Setup.vue')
  },
  {
    path: '/test',
    name: 'Test',
    component: () => import('../views/Test.vue')
  },
  {
    path: '/test-route',
    name: 'TestRoute',
    component: () => import('../views/TestRoute.vue')
  },
  {
    path: '/simple-login',
    name: 'SimpleLogin',
    component: () => import('../views/SimpleLogin.vue')
  },
  {
    path: '/login',
    name: 'Login',
    component: getLoginComponent
  },

  {
    path: '/openapi-docs',
    name: 'OpenAPIDocs',
    component: () => import('../views/OpenAPIDocs.vue'),
    meta: { title: 'view_public_openapi' }
  },
  {
    path: '/',
    name: 'Layout',
    component: () => import('../components/Layout.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: '/dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: 'dashboard', requiresAdmin: true }
      },
      // Admin routes
      {
        path: '/admin',
        name: 'Admin',
        meta: { requiresAuth: true, requiresAdmin: true },
        children: [
          {
            path: 'config-wizard',
            name: 'ConfigWizard',
            component: () => import('../views/admin/ConfigWizard.vue'),
            meta: { title: 'config_wizard' }
          },
          {
            path: 'vad-config',
            name: 'VADConfig',
            component: () => import('../views/admin/VADConfig.vue'),
            meta: { title: 'vad_config' }
          },
          {
            path: 'asr-config',
            name: 'ASRConfig',
            component: () => import('../views/admin/ASRConfig.vue'),
            meta: { title: 'asr_config' }
          },
          {
            path: 'llm-config',
            name: 'LLMConfig',
            component: () => import('../views/admin/LLMConfig.vue'),
            meta: { title: 'llm_config' }
          },
          {
            path: 'tts-config',
            name: 'TTSConfig',
            component: () => import('../views/admin/TTSConfig.vue'),
            meta: { title: 'tts_config' }
          },
          {
            path: 'speaker-config',
            name: 'SpeakerConfig',
            component: () => import('../views/admin/SpeakerConfig.vue'),
            meta: { title: 'speaker_config' }
          },
          {
            path: 'ota-config',
            name: 'OTAConfig',
            component: () => import('../views/admin/OTAConfig.vue'),
            meta: { title: 'ota_config' }
          },
          {
            path: 'mqtt-config',
            name: 'MQTTConfig',
            component: () => import('../views/admin/MQTTConfig.vue'),
            meta: { title: 'mqtt_config' }
          },
          {
            path: 'udp-config',
            name: 'UDPConfig',
            component: () => import('../views/admin/UDPConfig.vue'),
            meta: { title: 'udp_config' }
          },
          {
            path: 'mqtt-server-config',
            name: 'MQTTServerConfig',
            component: () => import('../views/admin/MQTTServerConfig.vue'),
            meta: { title: 'mqtt_server_config_management' }
          },
          {
            path: 'mcp-config',
            name: 'MCPConfig',
            component: () => import('../views/admin/MCPConfig.vue'),
            meta: { title: 'mcp_config_management' }
          },
          {
            path: 'mcp-market',
            name: 'MCPMarket',
            component: () => import('../views/admin/MCPMarket.vue'),
            meta: { title: 'mcp_market' }
          },
          {
            path: 'memory-config',
            name: 'MemoryConfig',
            component: () => import('../views/admin/MemoryConfig.vue'),
            meta: { title: 'memory_config' }
          },
          {
            path: 'knowledge-search-config',
            name: 'KnowledgeSearchConfig',
            component: () => import('../views/admin/KnowledgeSearchConfig.vue'),
            meta: { title: 'knowledge_search_config' }
          },
          {
            path: 'chat-settings',
            name: 'ChatSettings',
            component: () => import('../views/admin/ChatSettings.vue'),
            meta: { title: 'chat_settings' }
          },
          {
            path: 'vision-config',
            name: 'VisionConfig',
            component: () => import('../views/admin/VisionConfig.vue'),
            meta: { title: 'vision_config' }
          },
          {
            path: 'pool-stats',
            name: 'PoolStats',
            component: () => import('../views/admin/PoolStats.vue'),
            meta: { title: 'pool_stats' }
          },
          {
            path: 'global-roles',
            name: 'GlobalRoles',
            component: () => import('../views/admin/GlobalRoles.vue'),
            meta: { title: 'global_roles' }
          },
          {
            path: 'users',
            name: 'Users',
            component: () => import('../views/admin/Users.vue'),
            meta: { title: 'user_management' }
          },
          {
            path: 'devices',
            name: 'AdminDevices',
            component: () => import('../views/admin/Devices.vue'),
            meta: { title: 'device_management' }
          },
          {
            path: 'agents',
            name: 'AdminAgents',
            component: () => import('../views/admin/Agents.vue'),
            meta: { title: 'agent_management' }
          }
        ]
      },
      // User routes
      {
        path: '/console',
        redirect: '/agents',
        meta: { title: 'agent_console' }
      },
      {
        path: '/agents',
        name: 'Agents',
        component: () => import('../views/user/Agents.vue'),
        meta: { title: 'my_agents' }
      },
      {
        path: '/user/agents',
        name: 'UserAgents',
        component: () => import('../views/user/Agents.vue'),
        meta: { title: 'my_agents' }
      },
      {
        path: '/agents/:id/edit',
        name: 'AgentEdit',
        component: () => import('../views/user/AgentEdit.vue'),
        meta: { title: 'edit_agent' }
      },
      {
        path: '/user/agents/:id/edit',
        name: 'UserAgentEdit',
        component: () => import('../views/user/AgentEdit.vue'),
        meta: { title: 'edit_agent' }
      },
      {
        path: '/user/agents/:id/devices',
        name: 'AgentDevices',
        component: () => import('../views/user/AgentDevices.vue'),
        meta: { title: 'agent_device_management' }
      },
      {
        path: '/user/devices',
        name: 'UserDevices',
        component: () => import('../views/user/AgentDevices.vue'),
        meta: { title: 'device_list' }
      },
      {
        path: '/speakers',
        name: 'Speakers',
        component: () => import('../views/user/Speakers.vue'),
        meta: { title: 'voiceprint_management' }
      },
      {
        path: '/user/speakers',
        name: 'UserSpeakers',
        component: () => import('../views/user/Speakers.vue'),
        meta: { title: 'voiceprint_management' }
      },
      {
        path: '/voice-clones',
        name: 'VoiceClones',
        component: () => import('../views/user/VoiceClones.vue'),
        meta: { title: 'voice_clone' }
      },
      {
        path: '/more',
        name: 'MobileMore',
        component: () => import('../views/mobile/MobileMore.vue'),
        meta: { title: 'more_features' }
      },
      {
        path: '/user/agents/:id/history',
        name: 'AgentHistory',
        component: () => import('../views/user/AgentHistory.vue'),
        meta: { title: 'chat_history' }
      },

      {
        path: '/user/api-tokens',
        name: 'UserAPITokens',
        component: () => import('../views/user/APITokens.vue'),
        meta: { title: 'api_token_management' }
      },
      {
        path: '/user/knowledge-bases',
        name: 'UserKnowledgeBases',
        component: () => import('../views/user/KnowledgeBases.vue'),
        meta: { title: 'my_knowledge_bases' }
      },
      {
        path: 'user/roles',
        name: 'UserRoles',
        component: () => import('../views/user/Roles.vue'),
        meta: { title: 'my_roles' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // Allow setup page without auth
  if (to.path === '/setup') {
    next()
    return
  }

  // Already logged in: redirect from login page based on role (admin goes to wizard on first login)
  if (to.path === '/login' && authStore.isAuthenticated) {
    if (authStore.user?.role === 'admin') {
      if (!localStorage.getItem('admin_first_login_done')) {
        next('/admin/config-wizard')
      } else {
        next('/dashboard')
      }
    } else {
      next('/agents')
    }
    return
  }

  // Route requires authentication
  if (to.meta.requiresAuth) {
    if (!authStore.isAuthenticated) {
      // No token — redirect to login
      next('/login')
      return
    }

    // Token present but no user info — validate token
    if (!authStore.user && !authStore.isValidating) {
      try {
        await authStore.getProfile()
      } catch (error) {
        // 401: token invalid — redirect to login
        if (error.response?.status === 401) {
          next('/login')
          return
        }
        // Network error (backend unreachable) — allow access with error shown
        if (error.code === 'ERR_NETWORK' || error.message?.includes('Failed to fetch') || error.message?.includes('ERR_CONNECTION_REFUSED')) {
          // On network error without local user info, redirect to login
          if (!authStore.user) {
            next('/login')
            return
          }
          // Fall through to final next()
        } else {
          // Other errors — allow access (backend may be temporarily unavailable)
          // Fall through to final next()
        }
      }
    }

    // Wait for in-progress validation (max 2 seconds)
    if (authStore.isValidating) {
      let waitCount = 0
      while (authStore.isValidating && waitCount < 20) {
        await new Promise(resolve => setTimeout(resolve, 100))
        waitCount++
      }
    }
  }

  // Root path: redirect based on role (admin goes to wizard on first login)
  if (to.path === '/' && authStore.isAuthenticated) {
    if (authStore.user?.role === 'admin') {
      if (!localStorage.getItem('admin_first_login_done')) {
        next('/admin/config-wizard')
      } else {
        next('/dashboard')
      }
    } else {
      next('/agents')
    }
    return
  }

  // Non-admin accessing admin page — redirect to agents
  if (to.meta.requiresAdmin && authStore.user?.role !== 'admin') {
    next('/agents')
    return
  }

  next()
})

export default router
