import { useState, useEffect } from 'react'
import { createFileRoute, useRouter } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Loader2, Zap } from 'lucide-react'
import { toast } from 'sonner'
import appLogo from '@/assets/brand/app-logo.svg'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/hooks/use-locale'
import { useCaptcha } from '@/hooks/use-captcha'
import { useThemeStore } from '@/stores/theme'
import { getPostLoginRedirectPath } from '@/utils/auth-redirect'
import api from '@/utils/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

const LANG_LABELS: Record<string, string> = { vi: '🇻🇳 VI', en: '🇬🇧 EN', zh: '🇨🇳 ZH' }

function CaptchaBlock({ captcha, onRefresh }: { captcha: ReturnType<typeof useCaptcha>; onRefresh: () => void }) {
  const { t } = useLocale()
  return (
    <>
      <div className="flex items-center justify-between gap-4 p-3 rounded-lg border border-[var(--color-line)] bg-[var(--color-surface-muted)]">
        <div className="min-w-0">
          <span className="text-[11px] font-bold text-[var(--color-text-secondary)] block mb-1 uppercase tracking-wider font-mono">{t('captcha')}</span>
          <strong className="text-lg tracking-tight text-[var(--color-text)] block">{captcha.prompt || t('generating_questions')}</strong>
          <p className="text-[13px] text-[var(--color-text-secondary)] mt-1 leading-relaxed">{t('arithmetic_captcha_hint')}</p>
        </div>
        <Button type="button" variant="ghost" size="sm" disabled={captcha.loading} onClick={onRefresh}>{t('refresh_captcha')}</Button>
      </div>
      <div>
        <label className="text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)] block mb-1.5 font-mono">{t('calc_result')}</label>
        <Input inputMode="numeric" value={captcha.answer} onChange={(e) => captcha.setAnswer(e.target.value)} placeholder={t('enter_calc_result')} />
      </div>
    </>
  )
}

function LoginPage() {
  const { t, lang, setLang } = useLocale()
  const router = useRouter()
  const { login, register, user, isAdmin } = useAuthStore()
  const { nextMode } = useThemeStore()
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
    <div
      className="min-h-[100dvh] flex flex-col items-center justify-center p-6 relative overflow-hidden"
      style={{ background: 'var(--color-bg)' }}
    >
      {/* Ambient background glow */}
      <div
        className="absolute inset-0 pointer-events-none"
        style={{
          background: 'radial-gradient(ellipse 80% 60% at 70% 0%, color-mix(in srgb, var(--color-surface-2) 60%, transparent) 0%, transparent 70%)',
        }}
      />
      <div
        className="absolute inset-0 pointer-events-none"
        style={{
          background: 'radial-gradient(ellipse 50% 40% at 20% 100%, color-mix(in srgb, var(--color-primary-soft) 40%, transparent) 0%, transparent 60%)',
        }}
      />

      {/* Top bar — logo */}
      <header className="fixed top-0 left-0 right-0 flex items-center justify-between px-6 h-14 z-10">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg flex items-center justify-center bg-[var(--color-primary)] shadow-[var(--shadow-primary-glow)]">
            <Zap className="w-4 h-4" style={{ color: 'var(--primary-foreground)' }} />
          </div>
          <span className="font-semibold font-display text-sm text-[var(--color-text)]">Xiaozhi AI Platform</span>
        </div>
        <button
          type="button"
          onClick={nextMode}
          className="text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors px-2 py-1 rounded"
        >
          {LANG_LABELS[lang] ?? '🇻🇳 VI'}
        </button>
      </header>

      {/* Main grid */}
      <div className="relative z-10 w-full max-w-5xl grid lg:grid-cols-[1fr_420px] gap-8 items-center">

        {/* Left — branding panel (desktop only) */}
        <section className="hidden lg:flex flex-col gap-6 px-4 py-6">
          <div className="inline-flex items-center gap-3 px-3 py-2.5 rounded-2xl bg-[var(--color-surface-1)] border border-[var(--color-line)] shadow-[var(--shadow-card)] w-fit">
            <img src={appLogo} alt={t('xiaozhi_management_system')} className="w-12 h-12 rounded-xl object-cover" />
            <div>
              <strong className="block text-[var(--color-text)] text-base leading-snug font-display">{t('xiaozhi_management_system')}</strong>
              <span className="block text-[var(--color-text-secondary)] text-xs mt-0.5">{t('ai_platform_title')}</span>
            </div>
          </div>

          <div>
            <p className="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-2 font-mono">XIAOZHI CONTROL CENTER</p>
            <h1 className="text-5xl font-bold font-display tracking-tight leading-tight text-[var(--color-text)] mb-4">
              {t('xiaozhi_tagline')}
            </h1>
            <p className="text-[var(--color-text-secondary)] leading-relaxed max-w-lg text-[15px]">
              {t('platform_desc')}
            </p>
          </div>

          <div className="flex flex-wrap gap-2">
            <Badge className="bg-[var(--color-primary-soft)] text-[var(--color-primary)] border-[var(--color-primary)]/30 border">
              {t('agent_orchestration')}
            </Badge>
            {[t('device_access'), t('voiceprint_knowledge'), 'MCP / OpenClaw', t('mcp_remote_call'), t('proactive_voice_push')].map((b) => (
              <Badge key={b} variant="outline" className="border-[var(--color-line)] text-[var(--color-text-secondary)]">{b}</Badge>
            ))}
          </div>

          {/* Decorative stat chips */}
          <div className="flex gap-3 mt-2">
            {[
              { label: 'Agents', value: '∞' },
              { label: 'Devices', value: 'IoT' },
              { label: 'Voice AI', value: 'Live' },
            ].map((chip) => (
              <div key={chip.label} className="px-3 py-2 rounded-xl bg-[var(--color-surface-1)] border border-[var(--color-line)] shadow-[var(--shadow-card)]">
                <p className="text-xs font-mono uppercase tracking-wider text-[var(--color-text-tertiary)]">{chip.label}</p>
                <p className="text-lg font-bold font-display text-[var(--color-primary)]">{chip.value}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Right — login card */}
        <div
          className="w-full rounded-2xl border border-[var(--color-line)] p-8 relative overflow-hidden transition-shadow duration-300"
          style={{
            background: 'var(--card)',
            boxShadow: '0 0 0 1px var(--color-line), var(--shadow-card), 0 0 24px color-mix(in srgb, var(--color-primary) 6%, transparent)',
          }}
        >
          {/* Top highlight line */}
          <div
            className="absolute top-0 left-0 right-0 h-px"
            style={{ background: 'linear-gradient(90deg, transparent 0%, var(--color-primary) 50%, transparent 100%)', opacity: 0.3 }}
          />

          <div className="mb-6">
            <p className="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-1 font-mono">WELCOME BACK</p>
            <h2 className="text-2xl font-bold font-display tracking-tight text-[var(--color-text)]">{t('login_or_create')}</h2>
          </div>

          <Tabs defaultValue="login" className="mt-2">
            <TabsList className="w-full bg-[var(--color-surface-muted)] border border-[var(--color-line)]">
              <TabsTrigger value="login" className="flex-1">{t('login')}</TabsTrigger>
              <TabsTrigger value="register" className="flex-1">{t('register')}</TabsTrigger>
            </TabsList>

            <TabsContent value="login">
              <Form {...loginForm}>
                <form onSubmit={handleLogin} className="space-y-4 mt-4">
                  <FormField control={loginForm.control} name="username" render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)] font-mono">{t('username')}</FormLabel>
                      <FormControl><Input {...field} placeholder={t('enter_username')} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                  <FormField control={loginForm.control} name="password" render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)] font-mono">{t('password')}</FormLabel>
                      <FormControl><Input {...field} type="password" autoComplete="current-password" placeholder={t('enter_password')} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                  {captchaEnabled && <CaptchaBlock captcha={loginCaptcha} onRefresh={loginCaptcha.fetch} />}
                  <Button
                    type="submit"
                    className="w-full shadow-[var(--shadow-primary-glow)] active:scale-[0.98] transition-all"
                    disabled={loading || (captchaEnabled && (loginCaptcha.loading || !loginCaptcha.id))}
                  >
                    {loading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                    {t('login')}
                  </Button>
                </form>
              </Form>
            </TabsContent>

            <TabsContent value="register">
              <Form {...registerForm}>
                <form onSubmit={handleRegister} className="space-y-4 mt-4">
                  <FormField control={registerForm.control} name="username" render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)] font-mono">{t('username')}</FormLabel>
                      <FormControl><Input {...field} placeholder={t('enter_username')} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                  <FormField control={registerForm.control} name="email" render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)] font-mono">{t('email')}</FormLabel>
                      <FormControl><Input {...field} type="email" autoComplete="email" placeholder={t('enter_email')} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                  <FormField control={registerForm.control} name="password" render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)] font-mono">{t('password')}</FormLabel>
                      <FormControl><Input {...field} type="password" autoComplete="new-password" placeholder={t('enter_password_min6')} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                  <FormField control={registerForm.control} name="confirmPassword" render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs font-bold uppercase tracking-wider text-[var(--color-text-secondary)] font-mono">{t('confirm_password')}</FormLabel>
                      <FormControl><Input {...field} type="password" autoComplete="new-password" placeholder={t('confirm_password_prompt')} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                  <CaptchaBlock captcha={registerCaptcha} onRefresh={registerCaptcha.fetch} />
                  <Button
                    type="submit"
                    className="w-full shadow-[var(--shadow-primary-glow)] active:scale-[0.98] transition-all"
                    disabled={loading || registerCaptcha.loading || !registerCaptcha.id}
                  >
                    {loading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                    {t('register')}
                  </Button>
                </form>
              </Form>
            </TabsContent>
          </Tabs>
        </div>
      </div>

      {/* Footer */}
      <footer className="fixed bottom-0 left-0 right-0 flex items-center justify-between px-6 py-3 z-10">
        <span className="text-xs font-mono text-[var(--color-text-disabled)]">© 2024 Xiaozhi AI Platform</span>
        <div className="flex items-center gap-1">
          {Object.entries(LANG_LABELS).map(([code, label]) => (
            <button
              key={code}
              type="button"
              onClick={() => setLang(code as 'vi' | 'en' | 'zh')}
              className={`text-xs px-2 py-1 rounded transition-colors ${
                lang === code
                  ? 'font-bold text-[var(--color-primary)]'
                  : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </footer>
    </div>
  )
}

export const Route = createFileRoute('/login')({
  component: LoginPage,
})
