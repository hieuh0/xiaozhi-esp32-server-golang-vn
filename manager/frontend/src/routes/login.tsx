import { useState, useEffect } from 'react'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import appLogo from '@/assets/brand/app-logo.webp'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'
import { useCaptcha } from '@/hooks/use-captcha'
import { getPostLoginRedirectPath } from '@/utils/auth-redirect'
import api from '@/utils/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

function CaptchaBlock({ captcha, onRefresh }: { captcha: ReturnType<typeof useCaptcha>; onRefresh: () => void }) {
  const { t } = useLocale()
  return (
    <>
      <div className="flex items-center justify-between gap-4 p-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-2)]">
        <div className="min-w-0">
          <span className="text-[11px] font-semibold text-[var(--color-text-secondary)] block mb-1">{t('captcha')}</span>
          <strong className="text-lg tracking-tight text-[var(--color-text)] block">{captcha.prompt || t('generating_questions')}</strong>
          <p className="text-[13px] text-[var(--color-text-secondary)] mt-1 leading-relaxed">{t('arithmetic_captcha_hint')}</p>
        </div>
        <Button type="button" variant="ghost" size="sm" disabled={captcha.loading} onClick={onRefresh}>{t('refresh_captcha')}</Button>
      </div>
      <div>
        <label className="text-sm font-medium text-[var(--color-text)] block mb-1.5">{t('calc_result')}</label>
        <Input inputMode="numeric" value={captcha.answer} onChange={(e) => captcha.setAnswer(e.target.value)} placeholder={t('enter_calc_result')} />
      </div>
    </>
  )
}

function LoginPage() {
  const { t } = useLocale()
  const router = useRouter()
  const { login, register, user, isAdmin } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [captchaEnabled, setCaptchaEnabled] = useState(true)
  const loginCaptcha = useCaptcha()
  const registerCaptcha = useCaptcha()

  const loginSchema = z.object({
    username: z.string().min(1, t('enter_username')),
    password: z.string().min(1, t('enter_password')),
  })
  const registerSchema = z.object({
    username: z.string().min(1, t('enter_username')),
    email: z.string().min(1, t('enter_email')).email(t('enter_valid_email')),
    password: z.string().min(6, t('password_min_length')),
    confirmPassword: z.string().min(1, t('confirm_password_prompt')),
  }).refine((d) => d.password === d.confirmPassword, { message: t('password_mismatch'), path: ['confirmPassword'] })

  const loginForm = useForm<z.infer<typeof loginSchema>>({ resolver: zodResolver(loginSchema) })
  const registerForm = useForm<z.infer<typeof registerSchema>>({ resolver: zodResolver(registerSchema) })

  useEffect(() => {
    if (user) { router.navigate({ to: getPostLoginRedirectPath(isAdmin ? 'admin' : 'user') as '/' }); return }
    const init = async () => {
      try {
        const { data } = await api.get<{ enabled: boolean }>('/captcha/status')
        setCaptchaEnabled(data?.enabled !== false)
      } catch { /* default true */ }
    }
    init().then(() => {
      if (captchaEnabled) loginCaptcha.fetch()
      else loginCaptcha.clear()
      registerCaptcha.fetch()
    })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const handleLogin = loginForm.handleSubmit(async (values) => {
    if (captchaEnabled && !loginCaptcha.id) { toast.error(t('captcha_load_failed')); await loginCaptcha.fetch(); return }
    setLoading(true)
    const result = await login({
      username: values.username, password: values.password,
      ...(captchaEnabled && { captchaId: loginCaptcha.id, captchaAnswer: loginCaptcha.answer.trim() }),
    })
    setLoading(false)
    if (result.success) {
      toast.success(t('login_success'))
      localStorage.setItem('admin_first_login_done', '1')
      router.navigate({ to: getPostLoginRedirectPath(result.user?.role ?? '') as '/' })
    } else {
      toast.error(result.message)
      if (captchaEnabled) await loginCaptcha.fetch()
    }
  })

  const handleRegister = registerForm.handleSubmit(async (values) => {
    if (!registerCaptcha.id) { toast.error(t('captcha_load_failed')); await registerCaptcha.fetch(); return }
    setLoading(true)
    const result = await register({
      username: values.username, email: values.email, password: values.password,
      captchaId: registerCaptcha.id, captchaAnswer: registerCaptcha.answer.trim(),
    })
    setLoading(false)
    if (result.success) {
      toast.success(t('register_success_login'))
      registerForm.reset()
      await registerCaptcha.fetch()
      if (captchaEnabled) await loginCaptcha.fetch()
    } else {
      toast.error(result.message)
      await registerCaptcha.fetch()
    }
  })

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)] p-6">
      <div className="w-full max-w-5xl grid lg:grid-cols-[1fr_420px] gap-6 items-center">
        <section className="hidden lg:block px-4 py-6">
          <div className="inline-flex items-center gap-3 mb-6 px-3 py-2 rounded-2xl bg-[var(--color-surface-1)] border border-[var(--color-line)]">
            <img src={appLogo} alt={t('xiaozhi_management_system')} className="w-14 h-14 rounded-xl object-cover" />
            <div>
              <strong className="block text-[var(--color-text)] text-lg leading-snug">{t('xiaozhi_management_system')}</strong>
              <span className="block text-[var(--color-text-secondary)] text-[13px] mt-0.5">{t('ai_platform_title')}</span>
            </div>
          </div>
          <p className="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-2">XIAOZHI CONTROL CENTER</p>
          <h1 className="text-5xl font-bold tracking-tight leading-tight text-[var(--color-text)] mb-4">{t('xiaozhi_tagline')}</h1>
          <p className="text-[var(--color-text-secondary)] leading-relaxed max-w-lg">{t('platform_desc')}</p>
          <div className="flex flex-wrap gap-2 mt-5">
            <Badge>{t('agent_orchestration')}</Badge>
            {[t('device_access'), t('voiceprint_knowledge'), 'MCP / OpenClaw', t('mcp_remote_call'), t('proactive_voice_push')].map((b) => (
              <Badge key={b} variant="outline">{b}</Badge>
            ))}
          </div>
        </section>

        <Card className="w-full">
          <CardHeader className="pb-2">
            <p className="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-1">WELCOME BACK</p>
            <h2 className="text-2xl font-bold tracking-tight text-[var(--color-text)]">{t('login_or_create')}</h2>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="login" className="mt-2">
              <TabsList className="w-full">
                <TabsTrigger value="login" className="flex-1">{t('login')}</TabsTrigger>
                <TabsTrigger value="register" className="flex-1">{t('register')}</TabsTrigger>
              </TabsList>
              <TabsContent value="login">
                <Form {...loginForm}>
                  <form onSubmit={handleLogin} className="space-y-4 mt-4">
                    <FormField control={loginForm.control} name="username" render={({ field }) => (
                      <FormItem><FormLabel>{t('username')}</FormLabel><FormControl><Input {...field} placeholder={t('enter_username')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    <FormField control={loginForm.control} name="password" render={({ field }) => (
                      <FormItem><FormLabel>{t('password')}</FormLabel><FormControl><Input {...field} type="password" autoComplete="current-password" placeholder={t('enter_password')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    {captchaEnabled && <CaptchaBlock captcha={loginCaptcha} onRefresh={loginCaptcha.fetch} />}
                    <Button type="submit" className="w-full" disabled={loading || (captchaEnabled && (loginCaptcha.loading || !loginCaptcha.id))}>
                      {loading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}{t('login')}
                    </Button>
                  </form>
                </Form>
              </TabsContent>
              <TabsContent value="register">
                <Form {...registerForm}>
                  <form onSubmit={handleRegister} className="space-y-4 mt-4">
                    <FormField control={registerForm.control} name="username" render={({ field }) => (
                      <FormItem><FormLabel>{t('username')}</FormLabel><FormControl><Input {...field} placeholder={t('enter_username')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    <FormField control={registerForm.control} name="email" render={({ field }) => (
                      <FormItem><FormLabel>{t('email')}</FormLabel><FormControl><Input {...field} type="email" autoComplete="email" placeholder={t('enter_email')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    <FormField control={registerForm.control} name="password" render={({ field }) => (
                      <FormItem><FormLabel>{t('password')}</FormLabel><FormControl><Input {...field} type="password" autoComplete="new-password" placeholder={t('enter_password_min6')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    <FormField control={registerForm.control} name="confirmPassword" render={({ field }) => (
                      <FormItem><FormLabel>{t('confirm_password')}</FormLabel><FormControl><Input {...field} type="password" autoComplete="new-password" placeholder={t('confirm_password_prompt')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    <CaptchaBlock captcha={registerCaptcha} onRefresh={registerCaptcha.fetch} />
                    <Button type="submit" className="w-full" disabled={loading || registerCaptcha.loading || !registerCaptcha.id}>
                      {loading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}{t('register')}
                    </Button>
                  </form>
                </Form>
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/login')({
  component: LoginPage,
})
