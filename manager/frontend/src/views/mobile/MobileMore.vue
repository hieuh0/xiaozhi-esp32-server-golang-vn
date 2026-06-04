<template>
  <div class="mobile-more-page">
    <van-cell-group inset :title="t('common_functions')">
      <van-cell
        v-for="item in commonItems"
        :key="item.path"
        :title="item.title"
        :label="item.desc"
        is-link
        @click="go(item.path)"
      />
    </van-cell-group>

    <template v-if="authStore.isAdmin">
      <van-cell-group inset :title="t('service_config')">
        <van-cell
          v-for="item in serviceItems"
          :key="item.path"
          :title="item.title"
          is-link
          @click="go(item.path)"
        />
      </van-cell-group>

      <van-cell-group inset :title="t('ai_config')">
        <van-cell
          v-for="item in aiItems"
          :key="item.path"
          :title="item.title"
          is-link
          @click="go(item.path)"
        />
      </van-cell-group>

      <van-cell-group inset :title="t('system_management')">
        <van-cell
          v-for="item in systemItems"
          :key="item.path"
          :title="item.title"
          is-link
          @click="go(item.path)"
        />
      </van-cell-group>
    </template>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const router = useRouter()
const authStore = useAuthStore()

const commonItems = computed(() => {
  if (authStore.isAdmin) {
    return [
      { title: t('config_wizard'), desc: '首次部署推荐从这里开始', path: '/admin/config-wizard' },
      { title: t('resource_pool_stats'), desc: '查看系统资源池使用情况', path: '/admin/pool-stats' }
    ]
  }

  return [
    { title: t('my_roles'), desc: '管理个人角色模板', path: '/user/roles' },
    { title: t('voice_clone'), desc: '管理声音复刻任务', path: '/voice-clones' },
    { title: t('my_knowledge_base'), desc: '管理知识库文档', path: '/user/knowledge-bases' }
  ]
})

const serviceItems = [
  { title: t('ota_config'), path: '/admin/ota-config' },
  { title: t('mqtt_config'), path: '/admin/mqtt-config' },
  { title: t('mqtt_server_config'), path: '/admin/mqtt-server-config' },
  { title: t('udp_config'), path: '/admin/udp-config' },
  { title: t('mcp_config'), path: '/admin/mcp-config' },
  { title: t('mcp_market'), path: '/admin/mcp-market' },
  { title: t('voiceprint_recognition_config'), path: '/admin/speaker-config' },
  { title: t('chat_settings'), path: '/admin/chat-settings' }
]

const aiItems = [
  { title: t('vad_config'), path: '/admin/vad-config' },
  { title: t('asr_config'), path: '/admin/asr-config' },
  { title: t('llm_config'), path: '/admin/llm-config' },
  { title: t('tts_config'), path: '/admin/tts-config' },
  { title: t('vision_config'), path: '/admin/vision-config' },
  { title: t('memory_config'), path: '/admin/memory-config' },
  { title: t('knowledge_retrieval_config'), path: '/admin/knowledge-search-config' }
]

const systemItems = [
  { title: t('global_role'), path: '/admin/global-roles' },
  { title: t('user_management'), path: '/admin/users' },
  { title: t('device_management'), path: '/admin/devices' },
  { title: t('agent_management'), path: '/admin/agents' }
]

const go = (path) => {
  router.push(path)
}
</script>

<style scoped>
.mobile-more-page {
  padding: 12px 0 96px;
}

:deep(.van-cell-group) {
  margin-bottom: 14px;
  border-radius: 20px;
  overflow: hidden;
}

:deep(.van-cell-group__title) {
  padding: 0 18px 10px;
  font-weight: 700;
  color: var(--apple-text);
}

:deep(.van-cell) {
  min-height: 62px;
}

:deep(.van-cell__label) {
  margin-top: 6px;
  color: var(--apple-text-secondary);
}
</style>
