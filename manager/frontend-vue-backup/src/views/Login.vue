<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { Loader2 } from '@lucide/vue'
import { useAuthStore } from '../stores/auth'
import api from '../utils/api'
import { getPostLoginRedirectPath } from '../utils/authRedirect'
import { checkNeedsSetup } from '../utils/setupStatus'
import appLogo from '@/assets/brand/app-logo.webp'
import { useLocale } from '../composables/useLocale'
import { Form, FormField, FormControl, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

const { t } = useLocale()
const router = useRouter()
const authStore = useAuthStore()

const activeTab = ref('login')
const loading = ref(false)

// Captcha state — managed outside vee-validate (conditional, API-driven)
const loginCaptchaPrompt = ref('')
const loginCaptchaLoading = ref(false)
const loginCaptchaEnabled = ref(true)
const loginCaptchaId = ref('')
const loginCaptchaAnswer = ref('')
const registerCaptchaPrompt = ref('')
const registerCaptchaLoading = ref(false)
const registerCaptchaId = ref('')
const registerCaptchaAnswer = ref('')

const loginSchema = toTypedSchema(z.object({
  username: z.string().min(1, t('enter_username')),
  password: z.string().min(1, t('enter_password')),
}))

const registerSchema = toTypedSchema(z.object({
  username: z.string().min(1, t('enter_username')),
  email: z.string().min(1, t('enter_email')).email(t('enter_valid_email')),
  password: z.string().min(6, t('password_min_length')),
  confirmPassword: z.string().min(1, t('confirm_password_prompt')),
}).refine(d => d.password === d.confirmPassword, {
  message: t('password_mismatch'),
  path: ['confirmPassword'],
}))

const fetchCaptcha = async (idRef, answerRef, promptRef, loadingRef) => {
  loadingRef.value = true
  try {
    const { data } = await api.get('/captcha/challenge', { silentError: true })
    idRef.value = data.captchaId
    answerRef.value = ''
    promptRef.value = data.prompt
  } catch {
    idRef.value = ''
    answerRef.value = ''
    promptRef.value = t('question_load_failed')
  } finally {
    loadingRef.value = false
  }
}

const clearLoginCaptcha = () => {
  loginCaptchaId.value = ''
  loginCaptchaAnswer.value = ''
  loginCaptchaPrompt.value = ''
}

const loadLoginCaptchaStatus = async () => {
  try {
    const { data } = await api.get('/captcha/status', { silentError: true })
    loginCaptchaEnabled.value = data?.enabled !== false
  } catch {
    loginCaptchaEnabled.value = true
  }
  if (!loginCaptchaEnabled.value) clearLoginCaptcha()
}

const refreshLoginCaptcha = async () => {
  if (!loginCaptchaEnabled.value) { clearLoginCaptcha(); return }
  await fetchCaptcha(loginCaptchaId, loginCaptchaAnswer, loginCaptchaPrompt, loginCaptchaLoading)
}
const refreshRegisterCaptcha = () =>
  fetchCaptcha(registerCaptchaId, registerCaptchaAnswer, registerCaptchaPrompt, registerCaptchaLoading)

const handleLogin = async (values) => {
  if (loginCaptchaEnabled.value && !loginCaptchaId.value) {
    ElMessage.error(t('captcha_load_failed'))
    await refreshLoginCaptcha()
    return
  }
  loading.value = true
  const credentials = { username: values.username, password: values.password }
  if (loginCaptchaEnabled.value) {
    credentials.captchaId = loginCaptchaId.value
    credentials.captchaAnswer = loginCaptchaAnswer.value.trim()
  }
  const result = await authStore.login(credentials)
  loading.value = false
  if (result.success) {
    ElMessage.success(t('login_success'))
    router.push(getPostLoginRedirectPath(authStore.user))
  } else {
    ElMessage.error(result.message)
    if (loginCaptchaEnabled.value) await refreshLoginCaptcha()
  }
}

const handleRegister = async (values) => {
  if (!registerCaptchaId.value) {
    ElMessage.error(t('captcha_load_failed'))
    await refreshRegisterCaptcha()
    return
  }
  loading.value = true
  const result = await authStore.register({
    username: values.username,
    email: values.email,
    password: values.password,
    captchaId: registerCaptchaId.value,
    captchaAnswer: registerCaptchaAnswer.value.trim(),
  })
  loading.value = false
  if (result.success) {
    ElMessage.success(t('register_success_login'))
    activeTab.value = 'login'
    registerCaptchaId.value = ''
    registerCaptchaAnswer.value = ''
    await Promise.all([
      loginCaptchaEnabled.value ? refreshLoginCaptcha() : Promise.resolve(),
      refreshRegisterCaptcha(),
    ])
  } else {
    ElMessage.error(result.message)
    await refreshRegisterCaptcha()
  }
}

onMounted(async () => {
  try {
    if (await checkNeedsSetup()) router.push('/setup')
  } catch (error) {
    console.error(t('check_system_failed'), error)
  }
  await loadLoginCaptchaStatus()
  Promise.allSettled([
    loginCaptchaEnabled.value ? refreshLoginCaptcha() : Promise.resolve(),
    refreshRegisterCaptcha(),
  ])
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-[var(--color-bg)] p-6">
    <div class="w-full max-w-5xl grid lg:grid-cols-[1fr_420px] gap-6 items-center">

      <!-- Hero -->
      <section class="hidden lg:block px-4 py-6">
        <div class="inline-flex items-center gap-3 mb-6 px-3 py-2 rounded-2xl bg-[var(--color-surface)] border border-[var(--color-line)]">
          <img :src="appLogo" :alt="t('xiaozhi_management_system')" class="w-14 h-14 rounded-xl object-cover" />
          <div>
            <strong class="block text-[var(--color-text)] text-lg leading-snug">{{ t('xiaozhi_management_system') }}</strong>
            <span class="block text-[var(--color-text-secondary)] text-[13px] mt-0.5">{{ t('ai_platform_title') }}</span>
          </div>
        </div>
        <p class="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-2">XIAOZHI CONTROL CENTER</p>
        <h1 class="text-5xl font-bold tracking-tight leading-tight text-[var(--color-text)] mb-4">{{ t('xiaozhi_tagline') }}</h1>
        <p class="text-[var(--color-text-secondary)] leading-relaxed max-w-lg">{{ t('platform_desc') }}</p>
        <div class="flex flex-wrap gap-2 mt-5">
          <Badge>{{ t('agent_orchestration') }}</Badge>
          <Badge variant="outline">{{ t('device_access') }}</Badge>
          <Badge variant="outline">{{ t('voiceprint_knowledge') }}</Badge>
          <Badge variant="outline">MCP / OpenClaw</Badge>
          <Badge variant="outline">{{ t('mcp_remote_call') }}</Badge>
          <Badge variant="outline">{{ t('proactive_voice_push') }}</Badge>
        </div>
      </section>

      <!-- Auth card -->
      <Card class="w-full">
        <CardHeader class="pb-2">
          <p class="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-1">WELCOME BACK</p>
          <h2 class="text-2xl font-bold tracking-tight text-[var(--color-text)]">{{ t('login_or_create') }}</h2>
        </CardHeader>
        <CardContent>
          <Tabs v-model="activeTab" class="mt-2">
            <TabsList class="w-full">
              <TabsTrigger value="login" class="flex-1">{{ t('login') }}</TabsTrigger>
              <TabsTrigger value="register" class="flex-1">{{ t('register') }}</TabsTrigger>
            </TabsList>

            <TabsContent value="login">
              <Form :validation-schema="loginSchema" @submit="handleLogin" class="space-y-4 mt-4">
                <FormField name="username" v-slot="{ componentField }">
                  <FormItem>
                    <FormLabel>{{ t('username') }}</FormLabel>
                    <FormControl><Input v-bind="componentField" :placeholder="t('enter_username')" /></FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField name="password" v-slot="{ componentField }">
                  <FormItem>
                    <FormLabel>{{ t('password') }}</FormLabel>
                    <FormControl>
                      <Input v-bind="componentField" type="password" autocomplete="current-password" :placeholder="t('enter_password')" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <template v-if="loginCaptchaEnabled">
                  <div class="flex items-center justify-between gap-4 p-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-muted)]">
                    <div class="min-w-0">
                      <span class="text-[11px] font-semibold text-[var(--color-text-secondary)] block mb-1">{{ t('captcha') }}</span>
                      <strong class="text-lg tracking-tight text-[var(--color-text)] block">{{ loginCaptchaPrompt || t('generating_questions') }}</strong>
                      <p class="text-[13px] text-[var(--color-text-secondary)] mt-1 leading-relaxed">{{ t('arithmetic_captcha_hint') }}</p>
                    </div>
                    <Button type="button" variant="ghost" size="sm" :disabled="loginCaptchaLoading" @click="refreshLoginCaptcha">
                      {{ t('refresh_captcha') }}
                    </Button>
                  </div>
                  <div>
                    <label class="text-sm font-medium text-[var(--color-text)] block mb-1.5">{{ t('calc_result') }}</label>
                    <Input v-model="loginCaptchaAnswer" inputmode="numeric" :placeholder="t('enter_calc_result')" />
                  </div>
                </template>
                <Button type="submit" class="w-full" :disabled="loading || (loginCaptchaEnabled && (loginCaptchaLoading || !loginCaptchaId))">
                  <Loader2 v-if="loading" class="w-4 h-4 mr-2 animate-spin" />
                  {{ t('login') }}
                </Button>
              </Form>
            </TabsContent>

            <TabsContent value="register">
              <Form :validation-schema="registerSchema" @submit="handleRegister" class="space-y-4 mt-4">
                <FormField name="username" v-slot="{ componentField }">
                  <FormItem>
                    <FormLabel>{{ t('username') }}</FormLabel>
                    <FormControl><Input v-bind="componentField" :placeholder="t('enter_username')" /></FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField name="email" v-slot="{ componentField }">
                  <FormItem>
                    <FormLabel>{{ t('email') }}</FormLabel>
                    <FormControl>
                      <Input v-bind="componentField" type="email" autocomplete="email" :placeholder="t('enter_email')" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField name="password" v-slot="{ componentField }">
                  <FormItem>
                    <FormLabel>{{ t('password') }}</FormLabel>
                    <FormControl>
                      <Input v-bind="componentField" type="password" autocomplete="new-password" :placeholder="t('enter_password_min6')" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField name="confirmPassword" v-slot="{ componentField }">
                  <FormItem>
                    <FormLabel>{{ t('confirm_password') }}</FormLabel>
                    <FormControl>
                      <Input v-bind="componentField" type="password" autocomplete="new-password" :placeholder="t('confirm_password_prompt')" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <div class="flex items-center justify-between gap-4 p-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-muted)]">
                  <div class="min-w-0">
                    <span class="text-[11px] font-semibold text-[var(--color-text-secondary)] block mb-1">{{ t('captcha') }}</span>
                    <strong class="text-lg tracking-tight text-[var(--color-text)] block">{{ registerCaptchaPrompt || t('generating_questions') }}</strong>
                    <p class="text-[13px] text-[var(--color-text-secondary)] mt-1 leading-relaxed">{{ t('captcha_math_hint') }}</p>
                  </div>
                  <Button type="button" variant="ghost" size="sm" :disabled="registerCaptchaLoading" @click="refreshRegisterCaptcha">
                    {{ t('refresh_captcha') }}
                  </Button>
                </div>
                <div>
                  <label class="text-sm font-medium text-[var(--color-text)] block mb-1.5">{{ t('calc_result') }}</label>
                  <Input v-model="registerCaptchaAnswer" inputmode="numeric" :placeholder="t('enter_calc_result')" />
                </div>
                <Button type="submit" class="w-full" :disabled="loading || registerCaptchaLoading || !registerCaptchaId">
                  <Loader2 v-if="loading" class="w-4 h-4 mr-2 animate-spin" />
                  {{ t('register') }}
                </Button>
              </Form>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
