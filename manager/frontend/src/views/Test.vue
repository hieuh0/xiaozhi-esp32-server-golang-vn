<template>
  <div class="test-page">
    <h1>{{ t('test_page') }}测试页面</h1>
    <p>{{ t('routing_works') }}</p>
    <div class="debug-info">
      <h3>{{ t('debug_info_label') }}调试信息：</h3>
      <p>当前路径: {{ $route.path }}</p>
      <p>用户认证状态: {{ authStore.isAuthenticated }}</p>
      <p>用户信息: {{ authStore.user }}</p>
      <p>是否管理员: {{ authStore.isAdmin }}</p>
    </div>
    <div class="actions">
      <el-button @click="$router.push('/login')">{{ t('go_to_login_page') }}去登录页</el-button>
      <el-button @click="$router.push('/dashboard')">{{ t('go_to_dashboard') }}去仪表板</el-button>
      <el-button @click="$router.push('/agents')">{{ t('go_to_agent_list') }}去智能体列表</el-button>
    </div>
  </div>
</template>

<script setup>
import { useAuthStore } from '../stores/auth'
import { useLocale } from '../composables/useLocale'
const { t } = useLocale()

const authStore = useAuthStore()
</script>

<style scoped>
.test-page {
  padding: 24px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  box-shadow: var(--apple-shadow-md);
}

.test-page h1 {
  margin: 0 0 12px;
  color: var(--apple-text);
}

.test-page > p {
  margin: 0;
  color: var(--apple-text-secondary);
}

.debug-info {
  margin: 20px 0;
  padding: 16px 18px;
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  border-radius: 18px;
  color: var(--apple-text-secondary);
}

.debug-info h3 {
  margin: 20px 0;
  color: var(--apple-text);
}

.debug-info p {
  margin: 8px 0;
}

.actions {
  margin-top: 20px;
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.actions .el-button {
  margin-right: 0;
}
</style>
