import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { isMobile } from '../utils/device'

// 根据设备类型动态加载组件
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
      // 管理员路由
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
      // 用户路由
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

  // 如果访问引导页面，直接通过
  if (to.path === '/setup') {
    next()
    return
  }

  // 如果访问登录页且已登录，根据角色跳转（管理员首次未完成向导则去配置向导）
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

  // 如果需要认证
  if (to.meta.requiresAuth) {
    if (!authStore.isAuthenticated) {
      // 没有token，跳转到登录页
      next('/login')
      return
    }

    // 有token但没有用户信息，尝试验证token有效性
    if (!authStore.user && !authStore.isValidating) {
      try {
        await authStore.getProfile()
      } catch (error) {
        // 如果是401错误（token无效），跳转到登录页
        if (error.response?.status === 401) {
          next('/login')
          return
        }
        // 如果是网络错误（后端连接失败），允许继续访问（但会显示错误）
        if (error.code === 'ERR_NETWORK' || error.message?.includes('Failed to fetch') || error.message?.includes('ERR_CONNECTION_REFUSED')) {
          // 网络错误时，如果本地有用户信息，允许继续访问
          if (!authStore.user) {
            next('/login')
            return
          }
          // 注意：这里不调用 next()，让代码继续执行到最后的 next()
        } else {
          // 其他错误，允许继续访问（可能是后端暂时不可用）
          // 注意：这里不调用 next()，让代码继续执行到最后的 next()
        }
      }
    }

    // 如果正在验证中，等待验证完成（最多等待2秒）
    if (authStore.isValidating) {
      let waitCount = 0
      while (authStore.isValidating && waitCount < 20) {
        await new Promise(resolve => setTimeout(resolve, 100))
        waitCount++
      }
    }
  }

  // 如果访问根路径，根据角色跳转（管理员首次未完成向导则去配置向导）
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

  // 如果普通用户访问管理员页面，跳转到智能体工作台
  if (to.meta.requiresAdmin && authStore.user?.role !== 'admin') {
    next('/agents')
    return
  }

  next()
})

export default router
