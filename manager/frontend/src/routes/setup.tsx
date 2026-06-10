import { useState, useEffect } from 'react'
import { createFileRoute, Link, useRouter } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { CheckCircle, Loader2 } from 'lucide-react'
import api from '@/utils/api'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

interface AdminInfo { username: string; email: string }

function SetupPage() {
  const { t } = useLocale()
  const router = useRouter()
  const [checking, setChecking] = useState(true)
  const [needsSetup, setNeedsSetup] = useState(false)
  const [initialized, setInitialized] = useState(false)
  const [adminInfo, setAdminInfo] = useState<AdminInfo | null>(null)
  const [errorMessage, setErrorMessage] = useState('')

  const schema = z.object({
    admin_username: z.string().min(3, t('enter_admin_username')).max(50),
    admin_email: z.string().min(1, t('enter_admin_email')).email(t('enter_valid_email')),
    admin_password: z.string().min(6, t('enter_admin_password_min6')).max(100),
    confirmPassword: z.string().min(1, t('enter_password_again')),
  }).refine((d) => d.admin_password === d.confirmPassword, {
    message: t('passwords_inconsistent'), path: ['confirmPassword'],
  })

  const form = useForm<z.infer<typeof schema>>({ resolver: zodResolver(schema) })

  useEffect(() => {
    api.get<{ needs_setup: boolean }>('/setup/status')
      .then(({ data }) => {
        if (data.needs_setup) setNeedsSetup(true)
        else router.navigate({ to: '/login' })
      })
      .catch(() => setErrorMessage(t('check_system_refresh')))
      .finally(() => setChecking(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const onSubmit = form.handleSubmit(async (values) => {
    setErrorMessage('')
    try {
      const { data } = await api.post<{ admin: AdminInfo }>('/setup/initialize', {
        admin_username: values.admin_username,
        admin_email: values.admin_email,
        admin_password: values.admin_password,
      })
      setAdminInfo(data.admin)
      setInitialized(true)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setErrorMessage(e.response?.data?.error || t('system_init_retry'))
    }
  })

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)] p-6">
      <div className="w-full max-w-5xl grid lg:grid-cols-[1fr_460px] gap-6 items-center">
        <section className="hidden lg:block px-4 py-6">
          <p className="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-2">FIRST RUN EXPERIENCE</p>
          <h1 className="text-5xl font-bold font-display tracking-tight leading-tight text-[var(--color-text)] mb-4">{t('lighter_system_init')}</h1>
          <p className="text-[var(--color-text-secondary)] leading-relaxed max-w-lg text-[15px]">{t('setup_admin_hint')}</p>
        </section>

        <Card className="w-full">
          <CardHeader className="pb-2">
            <p className="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-1">SYSTEM SETUP</p>
            <h2 className="text-2xl font-bold font-display tracking-tight text-[var(--color-text)]">{t('system_init')}</h2>
            <p className="text-[var(--color-text-secondary)] text-[15px] leading-relaxed">{t('welcome_initial_setup')}</p>
          </CardHeader>
          <CardContent>
            {checking ? (
              <div className="flex flex-col items-center py-12 gap-4">
                <Loader2 className="w-8 h-8 animate-spin text-[var(--color-primary)]" />
                <p className="text-[var(--color-text-secondary)]">{t('checking_system_status')}</p>
              </div>
            ) : initialized ? (
              <div className="flex flex-col items-center py-10 text-center gap-4">
                <CheckCircle className="w-16 h-16 text-[var(--color-primary)]" />
                <h3 className="text-xl font-semibold text-[var(--color-text)]">{t('init_success')}</h3>
                <p className="text-[var(--color-text-secondary)] leading-relaxed">{t('system_init_admin_created')}</p>
                <div className="w-full bg-[var(--color-surface-2)] rounded-lg border border-[var(--color-line)] p-4 text-left space-y-1">
                  <p className="text-[var(--color-text)] text-sm"><strong>{t('username_label')}</strong> {adminInfo?.username}</p>
                  <p className="text-[var(--color-text)] text-sm"><strong>{t('email_label')}</strong> {adminInfo?.email}</p>
                </div>
                <Button asChild className="mt-2"><Link to="/login">{t('go_to_login')}</Link></Button>
              </div>
            ) : !needsSetup ? (
              <div className="flex flex-col items-center py-10 text-center gap-4">
                <CheckCircle className="w-16 h-16 text-[var(--color-primary)]" />
                <h3 className="text-xl font-semibold text-[var(--color-text)]">{t('system_initialized')}</h3>
                <p className="text-[var(--color-text-secondary)] leading-relaxed">{t('system_init_login_prompt')}</p>
                <Button asChild><Link to="/login">{t('go_to_login')}</Link></Button>
              </div>
            ) : (
              <div>
                <h3 className="text-lg font-semibold text-[var(--color-text)] mb-1">{t('create_admin_account')}</h3>
                <p className="text-[var(--color-text-secondary)] text-sm mb-6">{t('set_admin_account_info')}</p>
                <Form {...form}>
                  <form onSubmit={onSubmit} className="space-y-4">
                    <FormField control={form.control} name="admin_username" render={({ field }) => (
                      <FormItem><FormLabel>{t('admin_username')}</FormLabel><FormControl><Input {...field} placeholder={t('enter_admin_username')} autoComplete="username" /></FormControl><FormMessage /></FormItem>
                    )} />
                    <FormField control={form.control} name="admin_email" render={({ field }) => (
                      <FormItem><FormLabel>{t('admin_email')}</FormLabel><FormControl><Input {...field} type="email" autoComplete="email" placeholder={t('enter_admin_email')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    <FormField control={form.control} name="admin_password" render={({ field }) => (
                      <FormItem><FormLabel>{t('admin_password')}</FormLabel><FormControl><Input {...field} type="password" autoComplete="new-password" placeholder={t('enter_admin_password_min6')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    <FormField control={form.control} name="confirmPassword" render={({ field }) => (
                      <FormItem><FormLabel>{t('confirm_password')}</FormLabel><FormControl><Input {...field} type="password" autoComplete="new-password" placeholder={t('enter_password_again')} /></FormControl><FormMessage /></FormItem>
                    )} />
                    {errorMessage && (
                      <div className="text-sm text-[var(--color-danger)] bg-[var(--color-danger)]/10 border border-[var(--color-danger)]/20 rounded-lg px-3 py-2">{errorMessage}</div>
                    )}
                    <Button type="submit" className="w-full" disabled={form.formState.isSubmitting}>
                      {form.formState.isSubmitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                      {form.formState.isSubmitting ? t('initializing') : t('start_init')}
                    </Button>
                  </form>
                </Form>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/setup')({
  component: SetupPage,
})
