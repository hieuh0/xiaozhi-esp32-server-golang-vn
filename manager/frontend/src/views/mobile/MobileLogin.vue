<template>
  <div class="mobile-login-container">
    <div class="mobile-login-header">
      <img class="mobile-login-logo" :src="appLogo" :alt="t('xiaozhi_management_system')" />
      <h1>{{ t('xiaozhi_management_system') }}</h1>
      <p>{{ t('ai_voice_assistant_platform') }}</p>
    </div>
    
    <van-tabs v-model:active="activeTab" class="mobile-login-tabs">
      <van-tab :title="t('login')" name="login">
        <van-form @submit="handleLogin" class="mobile-login-form">
          <van-cell-group inset>
            <van-field
              v-model="loginForm.username"
              name="username"
              :label="t('username')"
              :placeholder="t('enter_username')"
              :rules="[{ required: true, message: t('enter_username') }]"
            />
            <van-field
              v-model="loginForm.password"
              type="password"
              name="password"
              :label="t('password')"
              :placeholder="t('enter_password')"
              :rules="[{ required: true, message: t('enter_password') }]"
            />
            <div v-if="loginCaptchaEnabled" class="mobile-captcha-panel">
              <div class="mobile-captcha-copy">
                <span>{{ t('captcha') }}</span>
                <strong>{{ loginCaptchaPrompt || t('generating_questions') }}</strong>
                <p>{{ t('arithmetic_captcha_hint') }}</p>
              </div>
              <van-button
                size="small"
                plain
                type="primary"
                native-type="button"
                :loading="loginCaptchaLoading"
                @click="refreshLoginCaptcha"
              >
                {{ t('refresh_captcha') }}</van-button>
            </div>
            <van-field
              v-if="loginCaptchaEnabled"
              v-model="loginForm.captchaAnswer"
              name="captchaAnswer"
              :label="t('calc_result')"
              :placeholder="t('enter_calc_result')"
              input-align="left"
              :rules="[{ required: true, message: t('enter_calc_result') }]"
            />
          </van-cell-group>
          
          <div class="mobile-login-actions">
            <van-button
              round
              block
              type="primary"
              native-type="submit"
              :loading="loading"
              :disabled="loginCaptchaEnabled && (loginCaptchaLoading || !loginForm.captchaId)"
              loading-:text="t('logging_in')"
              class="mobile-login-button"
            >
              {{ t('login') }}
            </van-button>
          </div>
        </van-form>
      </van-tab>
      
      <van-tab :title="t('register')" name="register">
        <van-form @submit="handleRegister" class="mobile-login-form">
          <van-cell-group inset>
            <van-field
              v-model="registerForm.username"
              name="username"
              :label="t('username')"
              :placeholder="t('enter_username')"
              :rules="[{ required: true, message: t('enter_username') }]"
            />
            <van-field
              v-model="registerForm.email"
              name="email"
              :label="t('email')"
              :placeholder="t('enter_email')"
              :rules="[
                { required: true, message: t('enter_email') },
                { pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/, message: t('enter_valid_email') }
              ]"
            />
            <van-field
              v-model="registerForm.password"
              type="password"
              name="password"
              :label="t('password')"
              :placeholder="t('enter_password_min6')"
              :rules="[
                { required: true, message: t('enter_password') },
                { pattern: /^.{6,}$/, message: t('password_min_length') }
              ]"
            />
            <van-field
              v-model="registerForm.confirmPassword"
              type="password"
              name="confirmPassword"
              :label="t('confirm_password')"
              :placeholder="t('confirm_password_prompt')"
              :rules="[
                { required: true, message: t('confirm_password_prompt') },
                { validator: validateConfirmPassword }
              ]"
            />
            <div class="mobile-captcha-panel">
              <div class="mobile-captcha-copy">
                <span>{{ t('captcha') }}</span>
                <strong>{{ registerCaptchaPrompt || t('generating_questions') }}</strong>
                <p>{{ t('captcha_math_hint') }}</p>
              </div>
              <van-button
                size="small"
                plain
                type="primary"
                native-type="button"
                :loading="registerCaptchaLoading"
                @click="refreshRegisterCaptcha"
              >
                {{ t('refresh_captcha') }}</van-button>
            </div>
            <van-field
              v-model="registerForm.captchaAnswer"
              name="captchaAnswer"
              :label="t('calc_result')"
              :placeholder="t('enter_calc_result')"
              input-align="left"
              :rules="[{ required: true, message: t('enter_calc_result') }]"
            />
          </van-cell-group>
          
          <div class="mobile-login-actions">
            <van-button
              round
              block
              type="primary"
              native-type="submit"
              :loading="loading"
              :disabled="registerCaptchaLoading || !registerForm.captchaId"
              loading-:text="t('registering')"
              class="mobile-login-button"
            >
              {{ t('register') }}
            </van-button>
          </div>
        </van-form>
      </van-tab>
    </van-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { showSuccessToast, showFailToast } from 'vant'
import { useAuthStore } from '../../stores/auth'
import api from '../../utils/api'
import { getPostLoginRedirectPath } from '../../utils/authRedirect'
import { checkNeedsSetup } from '../../utils/setupStatus'
import appLogo from '@/assets/brand/app-logo.webp'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const router = useRouter()
const authStore = useAuthStore()

const activeTab = ref('login')
const loading = ref(false)
const loginCaptchaPrompt = ref('')
const registerCaptchaPrompt = ref('')
const loginCaptchaLoading = ref(false)
const registerCaptchaLoading = ref(false)
const loginCaptchaEnabled = ref(true)

const loginForm = reactive({
  username: '',
  password: '',
  captchaId: '',
  captchaAnswer: ''
})

const registerForm = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  captchaId: '',
  captchaAnswer: ''
})

// Custom validator: confirm password
const validateConfirmPassword = (val) => {
  if (val !== registerForm.password) {
    return t('password_mismatch')
  }
  return true
}

const fetchCaptcha = async (form, promptRef, loadingRef) => {
  loadingRef.value = true
  try {
    const { data } = await api.get('/captcha/challenge', { silentError: true })
    form.captchaId = data.captchaId
    form.captchaAnswer = ''
    promptRef.value = data.prompt
  } catch (error) {
    form.captchaId = ''
    form.captchaAnswer = ''
    promptRef.value = t('question_load_failed')
  } finally {
    loadingRef.value = false
  }
}

const clearLoginCaptcha = () => {
  loginForm.captchaId = ''
  loginForm.captchaAnswer = ''
  loginCaptchaPrompt.value = ''
}

const loadLoginCaptchaStatus = async () => {
  try {
    const { data } = await api.get('/captcha/status', { silentError: true })
    loginCaptchaEnabled.value = data?.enabled !== false
  } catch (error) {
    loginCaptchaEnabled.value = true
  }

  if (!loginCaptchaEnabled.value) {
    clearLoginCaptcha()
  }
}

const refreshLoginCaptcha = async () => {
  if (!loginCaptchaEnabled.value) {
    clearLoginCaptcha()
    return
  }
  await fetchCaptcha(loginForm, loginCaptchaPrompt, loginCaptchaLoading)
}

const refreshRegisterCaptcha = async () => {
  await fetchCaptcha(registerForm, registerCaptchaPrompt, registerCaptchaLoading)
}

const handleLogin = async () => {
  if (loginCaptchaEnabled.value && !loginForm.captchaId) {
    showFailToast(t('captcha_load_failed'))
    await refreshLoginCaptcha()
    return
  }

  loading.value = true
  const credentials = {
    username: loginForm.username,
    password: loginForm.password
  }
  if (loginCaptchaEnabled.value) {
    credentials.captchaId = loginForm.captchaId
    credentials.captchaAnswer = loginForm.captchaAnswer.trim()
  }
  const result = await authStore.login(credentials)
  loading.value = false
  
  if (result.success) {
    showSuccessToast(t('login_success'))
    router.push(getPostLoginRedirectPath(authStore.user))
  } else {
    showFailToast(result.message || t('login_failed'))
    if (loginCaptchaEnabled.value) {
      await refreshLoginCaptcha()
    }
  }
}

const handleRegister = async () => {
  if (!registerForm.captchaId) {
    showFailToast(t('captcha_load_failed'))
    await refreshRegisterCaptcha()
    return
  }

  loading.value = true
  const result = await authStore.register({
    username: registerForm.username,
    email: registerForm.email,
    password: registerForm.password,
    captchaId: registerForm.captchaId,
    captchaAnswer: registerForm.captchaAnswer.trim()
  })
  loading.value = false
  
  if (result.success) {
    showSuccessToast(t('register_success_login'))
    activeTab.value = 'login'
    // Reset registration form
    Object.assign(registerForm, {
      username: '',
      email: '',
      password: '',
      confirmPassword: '',
      captchaId: '',
      captchaAnswer: ''
    })
    await Promise.all([
      loginCaptchaEnabled.value ? refreshLoginCaptcha() : Promise.resolve(),
      refreshRegisterCaptcha()
    ])
  } else {
    showFailToast(result.message || t('register_failed'))
    await refreshRegisterCaptcha()
  }
}

// Check system status; redirect to setup page if not initialized
const checkSystemStatus = async () => {
  try {
    if (await checkNeedsSetup()) {
      router.push('/setup')
    }
  } catch (error) {
    console.error(t('check_system_failed'), error)
  }
}

onMounted(async () => {
  checkSystemStatus()
  await loadLoginCaptchaStatus()
  Promise.allSettled([
    loginCaptchaEnabled.value ? refreshLoginCaptcha() : Promise.resolve(),
    refreshRegisterCaptcha()
  ])
})
</script>

<style scoped>
.mobile-login-container {
  min-height: 100vh;
  padding: 32px 16px 96px;
  display: flex;
  flex-direction: column;
}

.mobile-login-header {
  text-align: left;
  color: var(--apple-text);
  margin-bottom: 24px;
}

.mobile-login-logo {
  width: 72px;
  height: 72px;
  border-radius: 24px;
  object-fit: cover;
  display: block;
  margin-bottom: 18px;
  box-shadow: 0 16px 32px rgba(0, 122, 255, 0.18);
}

.mobile-login-header h1 {
  font-size: 32px;
  line-height: 1.08;
  letter-spacing: -0.04em;
  font-weight: 700;
  margin-bottom: 8px;
}

.mobile-login-header p {
  font-size: 14px;
  color: var(--apple-text-secondary);
  line-height: 1.7;
}

.mobile-login-tabs {
  flex: 1;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.9);
  box-shadow: var(--apple-shadow-lg);
  overflow: hidden;
}

.mobile-login-form {
  padding: 20px 0 10px;
}

.mobile-login-actions {
  padding: 20px 16px;
}

.mobile-login-button {
  height: 44px;
  font-size: 16px;
  font-weight: 500;
}

.mobile-captcha-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 12px 16px 6px;
  padding: 14px 16px;
  border-radius: 18px;
  border: 1px solid var(--apple-border);
  background: rgba(247, 248, 250, 0.92);
}

.mobile-captcha-copy {
  min-width: 0;
}

.mobile-captcha-copy span {
  display: inline-block;
  margin-bottom: 6px;
  color: var(--apple-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.mobile-captcha-copy strong {
  display: block;
  color: var(--apple-text);
  font-size: 18px;
  letter-spacing: -0.02em;
}

.mobile-captcha-copy p {
  margin: 6px 0 0;
  color: var(--apple-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

:deep(.van-tabs__nav) {
  background: transparent;
}

:deep(.van-tabs__line) {
  background: var(--apple-primary);
}
</style>
