import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { isMobile } from '../utils/device'
import { getPostLoginRedirectPath } from '../utils/authRedirect'

// Dynamically load login component based on device type
const getLoginComponent = () => {
  return isMobile()
    ? import('../views/mobile/MobileLogin.vue')
    : import('../views/Login.vue')
}

const getAgentsComponent = () => {
  return isMobile()
    ? import('../views/mobile/MobileAgents.vue')
    : import('../views/user/Agents.vue')
}

const getUserDevicesComponent = () => {
  return isMobile()
    ? import('../views/mobile/MobileDevices.vue')
    : import('../views/user/AgentDevices.vue')
}

const routes = [
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('../views/Setup.vue')
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
            path: 'config-overview',
            name: 'AdminConfigOverview',
            component: () => import('../views/admin/AdminConfigOverview.vue'),
            meta: { title: 'config_overview', requiresAdmin: true }
          },
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
            meta: {
              title: 'vad_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'vad_config' }
              ]
            }
          },
          {
            path: 'asr-config',
            name: 'ASRConfig',
            component: () => import('../views/admin/ASRConfig.vue'),
            meta: {
              title: 'asr_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'asr_config' }
              ]
            }
          },
          {
            path: 'llm-config',
            name: 'LLMConfig',
            component: () => import('../views/admin/LLMConfig.vue'),
            meta: {
              title: 'llm_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'llm_config' }
              ]
            }
          },
          {
            path: 'tts-config',
            name: 'TTSConfig',
            component: () => import('../views/admin/TTSConfig.vue'),
            meta: {
              title: 'tts_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'tts_config' }
              ]
            }
          },
          {
            path: 'speaker-config',
            name: 'SpeakerConfig',
            component: () => import('../views/admin/SpeakerConfig.vue'),
            meta: {
              title: 'speaker_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'speaker_config' }
              ]
            }
          },
          {
            path: 'ota-config',
            name: 'OTAConfig',
            component: () => import('../views/admin/OTAConfig.vue'),
            meta: {
              title: 'ota_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'ota_config' }
              ]
            }
          },
          {
            path: 'mqtt-config',
            name: 'MQTTConfig',
            component: () => import('../views/admin/MQTTConfig.vue'),
            meta: {
              title: 'mqtt_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'mqtt_config' }
              ]
            }
          },
          {
            path: 'udp-config',
            name: 'UDPConfig',
            component: () => import('../views/admin/UDPConfig.vue'),
            meta: {
              title: 'udp_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'udp_config' }
              ]
            }
          },
          {
            path: 'mqtt-server-config',
            name: 'MQTTServerConfig',
            component: () => import('../views/admin/MQTTServerConfig.vue'),
            meta: {
              title: 'mqtt_server_config_management',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'mqtt_server_config_management' }
              ]
            }
          },
          {
            path: 'mcp-config',
            name: 'MCPConfig',
            component: () => import('../views/admin/MCPConfig.vue'),
            meta: {
              title: 'mcp_config_management',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'mcp_config_management' }
              ]
            }
          },
          {
            path: 'mcp-market',
            name: 'MCPMarket',
            component: () => import('../views/admin/MCPMarket.vue'),
            meta: {
              title: 'mcp_market',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'mcp_market' }
              ]
            }
          },
          {
            path: 'memory-config',
            name: 'MemoryConfig',
            component: () => import('../views/admin/MemoryConfig.vue'),
            meta: {
              title: 'memory_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'memory_config' }
              ]
            }
          },
          {
            path: 'knowledge-search-config',
            name: 'KnowledgeSearchConfig',
            component: () => import('../views/admin/KnowledgeSearchConfig.vue'),
            meta: {
              title: 'knowledge_search_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'knowledge_search_config' }
              ]
            }
          },
          {
            path: 'chat-settings',
            name: 'ChatSettings',
            component: () => import('../views/admin/ChatSettings.vue'),
            meta: {
              title: 'chat_settings',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'chat_settings' }
              ]
            }
          },
          {
            path: 'vision-config',
            name: 'VisionConfig',
            component: () => import('../views/admin/VisionConfig.vue'),
            meta: {
              title: 'vision_config',
              breadcrumb: [
                { labelKey: 'dashboard', path: '/dashboard' },
                { labelKey: 'config_overview', path: '/admin/config-overview' },
                { labelKey: 'vision_config' }
              ]
            }
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
        component: getAgentsComponent,
        meta: { title: 'my_agents' }
      },
      {
        path: '/user/agents',
        redirect: '/agents'
      },
      {
        path: '/agents/:id/edit',
        name: 'AgentEdit',
        component: () => import('../views/user/AgentEdit.vue'),
        meta: { title: 'edit_agent' }
      },
      {
        path: '/user/agents/:id/edit',
        redirect: to => `/agents/${to.params.id}/edit`
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
        component: getUserDevicesComponent,
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
        redirect: '/speakers'
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

  // Already logged in: redirect from login page based on role
  if (to.path === '/login' && authStore.isAuthenticated) {
    next(getPostLoginRedirectPath(authStore.user))
    return
  }

  // Route requires authentication
  if (to.meta.requiresAuth) {
    if (!authStore.isAuthenticated) {
      next('/login')
      return
    }

    // Token present but no user info — validate token (getProfile deduplicates concurrent calls)
    if (!authStore.user) {
      try {
        await authStore.getProfile()
      } catch (error) {
        if (error.response?.status === 401 || !authStore.user) {
          next('/login')
          return
        }
      }
    }
  }

  // Root path: redirect based on role
  if (to.path === '/' && authStore.isAuthenticated) {
    next(getPostLoginRedirectPath(authStore.user))
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
