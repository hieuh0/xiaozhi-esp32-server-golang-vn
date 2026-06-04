<template>
  <div class="simple-login-page">
    <div class="simple-login-card">
      <div class="simple-login-header">
        <h1>{{ t('simple_login_test') }}简单登录测试</h1>
        <p>用于快速验证认证链路和路由跳转。</p>
      </div>

      <div class="simple-login-form">
        <label class="simple-login-label" for="simple-login-username">{{ t('username') }}用户名</label>
        <input id="simple-login-username" v-model="username" type="text" class="simple-login-input" />

        <label class="simple-login-label" for="simple-login-password">{{ t('password') }}密码</label>
        <input id="simple-login-password" v-model="password" type="password" class="simple-login-input" />

        <button @click="login" class="simple-login-button">
          {{ t('login') }}  登录
        </button>
      </div>

      <div class="debug-info">
        <h3>{{ t('debug_info') }}调试信息</h3>
        <p>认证状态: {{ authStore.isAuthenticated }}</p>
        <p>用户信息: {{ JSON.stringify(authStore.user) }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useLocale } from '../composables/useLocale'
const { t } = useLocale()

const router = useRouter()
const authStore = useAuthStore()

const username = ref('admin')
const password = ref('password')

const login = async () => {
  try {
    const result = await authStore.login({
      username: username.value,
      password: password.value
    })
    
    if (result.success) {
      alert('登录成功!')
      if (authStore.user?.role === 'admin') {
        router.push('/dashboard')
      } else {
        router.push('/agents')
      }
    } else {
      alert('登录失败: ' + result.message)
    }
  } catch (error) {
    alert('登录错误: ' + error.message)
  }
}
</script>

<style scoped>
.simple-login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 20px;
}

.simple-login-card {
  width: min(100%, 460px);
  padding: 28px;
}

.simple-login-header {
  margin-bottom: 24px;
}

.simple-login-header h1 {
  margin: 0;
  color: var(--apple-text);
  font-size: 30px;
  letter-spacing: -0.04em;
}

.simple-login-header p {
  margin: 10px 0 0;
  color: var(--apple-text-secondary);
  line-height: 1.6;
}

.simple-login-form {
  display: grid;
  gap: 12px;
}

.simple-login-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--apple-text);
}

.simple-login-input {
  width: 100%;
  min-height: 48px;
  padding: 0 16px;
  border: 1px solid var(--apple-line);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.96);
  color: var(--apple-text);
  outline: none;
  box-sizing: border-box;
}

.simple-login-input:focus {
  border-color: rgba(0, 122, 255, 0.34);
  box-shadow: 0 0 0 4px rgba(0, 122, 255, 0.08);
}

.simple-login-button {
  min-height: 48px;
  margin-top: 8px;
  border: 0;
  border-radius: 16px;
  background: linear-gradient(180deg, #2e90ff 0%, #007aff 100%);
  color: #ffffff;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  box-shadow: 0 16px 32px rgba(0, 122, 255, 0.18);
}

.simple-login-button:hover {
  transform: translateY(-1px);
}

.debug-info {
  margin-top: 20px;
}

.debug-info h3 {
  margin: 0 0 10px;
  color: var(--apple-text);
}

.debug-info p {
  margin: 6px 0;
  color: var(--apple-text-secondary);
  word-break: break-word;
}
</style>
