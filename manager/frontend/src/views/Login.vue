<template>
  <div class="login-container">
    <div class="login-shell">
      <section class="login-hero">
        <div class="login-brand">
          <img class="login-brand-logo" :src="appLogo" alt="小智管理系统" />
          <div>
            <strong>{{ t('xiaozhi_management_system') }}小智管理系统</strong>
            <span>{{ t('ai_platform_title') }}AI 服务与设备管理平台</span>
          </div>
        </div>
        <p class="login-eyebrow">XIAOZHI CONTROL CENTER</p>
        <h1>{{ t('xiaozhi_tagline') }}小智 AI Go 到起飞。</h1>
        <p>
          {{ t('platform_desc') }}</p>
        <div class="login-meta">
          <span class="apple-chip is-primary">{{ t('agent_orchestration') }}智能体编排</span>
          <span class="apple-chip">{{ t('device_access') }}设备接入</span>
          <span class="apple-chip">{{ t('voiceprint_knowledge') }}声纹与知识库</span>
          <span class="apple-chip">MCP / OpenClaw</span>
          <span class="apple-chip">{{ t('mcp_remote_call') }}MCP 远程调用</span>
          <span class="apple-chip">{{ t('proactive_voice_push') }}主动语音下发</span>
        </div>
      </section>

      <el-card class="login-card">
        <template #header>
          <div class="card-header">
            <div>
              <p class="card-eyebrow">WELCOME BACK</p>
              <h2>{{ t('login_or_create') }}登录或创建账户</h2>
            </div>
          </div>
        </template>

        <el-tabs v-model="activeTab" class="login-tabs">
          <el-tab-pane :label="t('login')" name="login">
            <el-form
              ref="loginFormRef"
              :model="loginForm"
              :rules="loginRules"
              label-position="top"
            >
              <el-form-item :label="t('username')" prop="username">
                <el-input v-model="loginForm.username" :placeholder="t('enter_username')" />
              </el-form-item>
              <el-form-item :label="t('password')" prop="password">
                <el-input
                  v-model="loginForm.password"
                  type="password"
                  :placeholder="t('enter_password')"
                  @keyup.enter="handleLogin"
                />
              </el-form-item>
              <div v-if="loginCaptchaEnabled" class="captcha-strip">
                <div class="captcha-copy">
                  <span class="captcha-label">人机验证</span>
                  <strong>{{ loginCaptchaPrompt || t('generating_questions') }}</strong>
                  <p>简单算术题，防止脚本批量登录。</p>
                </div>
                <el-button
                  link
                  type="primary"
                  :loading="loginCaptchaLoading"
                  @click="refreshLoginCaptcha"
                >
                  {{ t('refresh_captcha') }}
                </el-button>
              </div>
              <el-form-item v-if="loginCaptchaEnabled" :label="t('calc_result')" prop="captchaAnswer">
                <el-input
                  v-model="loginForm.captchaAnswer"
                  inputmode="numeric"
                  :placeholder="t('enter_calc_result')"
                  @keyup.enter="handleLogin"
                />
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="loading"
                  :disabled="loginCaptchaEnabled && (loginCaptchaLoading || !loginForm.captchaId)"
                  @click="handleLogin"
                  style="width: 100%"
                >
                  {{ t('login') }}
                </el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane :label="t('register')" name="register">
            <el-form
              ref="registerFormRef"
              :model="registerForm"
              :rules="registerRules"
              label-position="top"
            >
              <el-form-item :label="t('username')" prop="username">
                <el-input v-model="registerForm.username" :placeholder="t('enter_username')" />
              </el-form-item>
              <el-form-item :label="t('email')" prop="email">
                <el-input v-model="registerForm.email" :placeholder="t('enter_email')" />
              </el-form-item>
              <el-form-item :label="t('password')" prop="password">
                <el-input
                  v-model="registerForm.password"
                  type="password"
                  :placeholder="t('enter_password')"
                />
              </el-form-item>
              <el-form-item :label="t('confirm_password')" prop="confirmPassword">
                <el-input
                  v-model="registerForm.confirmPassword"
                  type="password"
                  :placeholder="t('confirm_password_prompt')"
                  @keyup.enter="handleRegister"
                />
              </el-form-item>
              <div class="captcha-strip">
                <div class="captcha-copy">
                  <span class="captcha-label">人机验证</span>
                  <strong>{{ registerCaptchaPrompt || t('generating_questions') }}</strong>
                  <p>完成简单算式后再提交注册。</p>
                </div>
                <el-button
                  link
                  type="primary"
                  :loading="registerCaptchaLoading"
                  @click="refreshRegisterCaptcha"
                >
                  {{ t('refresh_captcha') }}
                </el-button>
              </div>
              <el-form-item :label="t('calc_result')" prop="captchaAnswer">
                <el-input
                  v-model="registerForm.captchaAnswer"
                  inputmode="numeric"
                  :placeholder="t('enter_calc_result')"
                  @keyup.enter="handleRegister"
                />
              </el-form-item>
              <el-form-item>
                <el-button
                  type="primary"
                  :loading="loading"
                  :disabled="registerCaptchaLoading || !registerForm.captchaId"
                  @click="handleRegister"
                  style="width: 100%"
                >
                  {{ t('register') }}
                </el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import api from '../utils/api'
import { getPostLoginRedirectPath } from '../utils/authRedirect'
import { checkNeedsSetup } from '../utils/setupStatus'
import appLogo from '@/assets/brand/app-logo.webp'
import { useLocale } from '../composables/useLocale'
const { t } = useLocale()

const router = useRouter()
const authStore = useAuthStore()

const activeTab = ref('login')
const loading = ref(false)
const loginFormRef = ref()
const registerFormRef = ref()
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

const loginRules = {
  username: [{ required: true, message: t('enter_username'), trigger: 'blur' }],
  password: [{ required: true, message: t('enter_password'), trigger: 'blur' }],
  captchaAnswer: [
    {
      validator: (rule, value, callback) => {
        if (!loginCaptchaEnabled.value || String(value || '').trim()) {
          callback()
          return
        }
        callback(new Error(t('enter_calc_result')))
      },
      trigger: 'blur'
    }
  ]
}

const registerRules = {
  username: [{ required: true, message: t('enter_username'), trigger: 'blur' }],
  email: [
    { required: true, message: t('enter_email'), trigger: 'blur' },
    { type: 'email', message: t('enter_valid_email'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('enter_password'), trigger: 'blur' },
    { min: 6, message: t('password_min_length'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('confirm_password_prompt'), trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== registerForm.password) {
          callback(new Error(t('password_mismatch')))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ],
  captchaAnswer: [
    { required: true, message: t('enter_calc_result'), trigger: 'blur' }
  ]
}

const fetchCaptcha = async (form, promptRef, loadingRef, formRef) => {
  loadingRef.value = true
  try {
    const { data } = await api.get('/captcha/challenge', { silentError: true })
    form.captchaId = data.captchaId
    form.captchaAnswer = ''
    promptRef.value = data.prompt
    formRef?.value?.clearValidate?.(['captchaAnswer'])
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
  loginFormRef.value?.clearValidate?.(['captchaAnswer'])
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
  await fetchCaptcha(loginForm, loginCaptchaPrompt, loginCaptchaLoading, loginFormRef)
}

const refreshRegisterCaptcha = async () => {
  await fetchCaptcha(registerForm, registerCaptchaPrompt, registerCaptchaLoading, registerFormRef)
}

const handleLogin = async () => {
  if (!loginFormRef.value) return

  try {
    await loginFormRef.value.validate()
  } catch {
    return
  }

  if (loginCaptchaEnabled.value && !loginForm.captchaId) {
    ElMessage.error(t('captcha_load_failed'))
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
    ElMessage.success(t('login_success'))
    router.push(getPostLoginRedirectPath(authStore.user))
  } else {
    ElMessage.error(result.message)
    if (loginCaptchaEnabled.value) {
      await refreshLoginCaptcha()
    }
  }
}

const handleRegister = async () => {
  if (!registerFormRef.value) return

  try {
    await registerFormRef.value.validate()
  } catch {
    return
  }

  if (!registerForm.captchaId) {
    ElMessage.error(t('captcha_load_failed'))
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
    ElMessage.success(t('register_success_login'))
    activeTab.value = 'login'
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
    ElMessage.error(result.message)
    await refreshRegisterCaptcha()
  }
}

// 检查系统状态，如果未初始化则跳转到引导页面
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
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.login-shell {
  width: min(1120px, 100%);
  display: grid;
  grid-template-columns: minmax(0, 1fr) 420px;
  gap: 24px;
  align-items: center;
}

.login-hero {
  padding: 28px;
}

.login-brand {
  display: inline-flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 24px;
  padding: 10px 14px 10px 10px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.74);
  border: 1px solid rgba(255, 255, 255, 0.86);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.08);
}

.login-brand-logo {
  width: 58px;
  height: 58px;
  border-radius: 20px;
  object-fit: cover;
  box-shadow: 0 12px 24px rgba(0, 122, 255, 0.18);
}

.login-brand strong,
.login-brand span {
  display: block;
}

.login-brand strong {
  color: var(--apple-text);
  font-size: 18px;
  line-height: 1.25;
}

.login-brand span {
  margin-top: 3px;
  color: var(--apple-text-secondary);
  font-size: 13px;
}

.login-eyebrow,
.card-eyebrow {
  margin: 0 0 8px;
  color: var(--apple-primary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.login-hero h1 {
  margin: 0;
  font-size: 48px;
  line-height: 1.02;
  letter-spacing: -0.05em;
}

.login-hero p {
  margin: 16px 0 0;
  max-width: 520px;
  color: var(--apple-text-secondary);
  line-height: 1.8;
}

.login-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 22px;
}

.login-card {
  border-radius: 30px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  box-shadow: var(--apple-shadow-lg);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h2 {
  margin: 0;
  color: var(--apple-text);
  font-size: 28px;
  letter-spacing: -0.03em;
}

.login-tabs {
  margin-top: 8px;
}

.captcha-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
  padding: 14px 16px;
  border-radius: 18px;
  border: 1px solid var(--apple-border);
  background: rgba(247, 248, 250, 0.92);
}

.captcha-copy {
  min-width: 0;
}

.captcha-copy strong {
  display: block;
  color: var(--apple-text);
  font-size: 18px;
  letter-spacing: -0.02em;
}

.captcha-copy p {
  margin: 6px 0 0;
  color: var(--apple-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.captcha-label {
  display: inline-block;
  margin-bottom: 6px;
  color: var(--apple-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

@media (max-width: 960px) {
  .login-shell {
    grid-template-columns: 1fr;
  }

  .login-hero {
    padding: 8px 0;
  }

  .login-hero h1 {
    font-size: 38px;
  }

  .captcha-strip {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
