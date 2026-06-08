<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { CheckCircle, Loader2 } from '@lucide/vue'
import api from '@/utils/api'
import { useLocale } from '@/composables/useLocale'
import { Form, FormField, FormControl, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

const { t } = useLocale()
const router = useRouter()

const checking = ref(true)
const needsSetup = ref(false)
const initialized = ref(false)
const initializing = ref(false)
const errorMessage = ref('')
const adminInfo = ref({})

const schema = toTypedSchema(z.object({
  admin_username: z.string().min(3, t('enter_admin_username')).max(50),
  admin_email: z.string().min(1, t('enter_admin_email')).email(t('enter_valid_email')),
  admin_password: z.string().min(6, t('enter_admin_password_min6')).max(100),
  confirmPassword: z.string().min(1, t('enter_password_again')),
}).refine(d => d.admin_password === d.confirmPassword, {
  message: t('passwords_inconsistent'),
  path: ['confirmPassword'],
}))

const checkSetupStatus = async () => {
  try {
    checking.value = true
    const response = await api.get('/setup/status')
    if (response.data.needs_setup) {
      needsSetup.value = true
    } else {
      router.push('/login')
    }
  } catch (error) {
    console.error(t('check_system_failed'), error)
    errorMessage.value = t('check_system_refresh')
  } finally {
    checking.value = false
  }
}

const initializeSystem = async (values) => {
  try {
    initializing.value = true
    errorMessage.value = ''
    const response = await api.post('/setup/initialize', {
      admin_username: values.admin_username,
      admin_email: values.admin_email,
      admin_password: values.admin_password,
    })
    adminInfo.value = response.data.admin
    initialized.value = true
  } catch (error) {
    console.error(t('system_init_failed'), error)
    errorMessage.value = error.response?.data?.error || t('system_init_retry')
  } finally {
    initializing.value = false
  }
}

onMounted(checkSetupStatus)
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-[var(--color-bg)] p-6">
    <div class="w-full max-w-5xl grid lg:grid-cols-[1fr_460px] gap-6 items-center">

      <!-- Intro -->
      <section class="hidden lg:block px-4 py-6">
        <p class="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-2">FIRST RUN EXPERIENCE</p>
        <h1 class="text-5xl font-bold tracking-tight leading-tight text-[var(--color-text)] mb-4">{{ t('lighter_system_init') }}</h1>
        <p class="text-[var(--color-text-secondary)] leading-relaxed max-w-lg text-[15px]">{{ t('setup_admin_hint') }}</p>
      </section>

      <!-- Setup card -->
      <Card class="w-full">
        <CardHeader class="pb-2">
          <p class="text-[11px] font-bold tracking-widest text-[var(--color-primary)] uppercase mb-1">SYSTEM SETUP</p>
          <h2 class="text-2xl font-bold tracking-tight text-[var(--color-text)]">{{ t('system_init') }}</h2>
          <p class="text-[var(--color-text-secondary)] text-[15px] leading-relaxed">{{ t('welcome_initial_setup') }}</p>
        </CardHeader>
        <CardContent>

          <!-- Checking spinner -->
          <div v-if="checking" class="flex flex-col items-center py-12 gap-4">
            <Loader2 class="w-8 h-8 animate-spin text-[var(--color-primary)]" />
            <p class="text-[var(--color-text-secondary)]">{{ t('checking_system_status') }}</p>
          </div>

          <!-- Success after form submit -->
          <div v-else-if="initialized" class="flex flex-col items-center py-10 text-center gap-4">
            <CheckCircle class="w-16 h-16 text-[var(--color-primary)]" />
            <h3 class="text-xl font-semibold text-[var(--color-text)]">{{ t('init_success') }}</h3>
            <p class="text-[var(--color-text-secondary)] leading-relaxed">{{ t('system_init_admin_created') }}</p>
            <div class="w-full bg-[var(--color-surface-muted)] rounded-lg border border-[var(--color-line)] p-4 text-left space-y-1">
              <p class="text-[var(--color-text)] text-sm"><strong>{{ t('username_label') }}</strong> {{ adminInfo.username }}</p>
              <p class="text-[var(--color-text)] text-sm"><strong>{{ t('email_label') }}</strong> {{ adminInfo.email }}</p>
            </div>
            <Button as-child class="mt-2">
              <router-link to="/login">{{ t('go_to_login') }}</router-link>
            </Button>
          </div>

          <!-- Already initialized (edge case) -->
          <div v-else-if="!needsSetup" class="flex flex-col items-center py-10 text-center gap-4">
            <CheckCircle class="w-16 h-16 text-[var(--color-primary)]" />
            <h3 class="text-xl font-semibold text-[var(--color-text)]">{{ t('system_initialized') }}</h3>
            <p class="text-[var(--color-text-secondary)] leading-relaxed">{{ t('system_init_login_prompt') }}</p>
            <Button as-child>
              <router-link to="/login">{{ t('go_to_login') }}</router-link>
            </Button>
          </div>

          <!-- Setup form -->
          <div v-else>
            <h3 class="text-lg font-semibold text-[var(--color-text)] mb-1">{{ t('create_admin_account') }}</h3>
            <p class="text-[var(--color-text-secondary)] text-sm mb-6">{{ t('set_admin_account_info') }}</p>

            <Form :validation-schema="schema" @submit="initializeSystem" class="space-y-4">
              <FormField name="admin_username" v-slot="{ componentField }">
                <FormItem>
                  <FormLabel>{{ t('admin_username') }}</FormLabel>
                  <FormControl>
                    <Input v-bind="componentField" :placeholder="t('enter_admin_username')" autocomplete="username" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField name="admin_email" v-slot="{ componentField }">
                <FormItem>
                  <FormLabel>{{ t('admin_email') }}</FormLabel>
                  <FormControl>
                    <Input v-bind="componentField" type="email" autocomplete="email" :placeholder="t('enter_admin_email')" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField name="admin_password" v-slot="{ componentField }">
                <FormItem>
                  <FormLabel>{{ t('admin_password') }}</FormLabel>
                  <FormControl>
                    <Input v-bind="componentField" type="password" autocomplete="new-password" :placeholder="t('enter_admin_password_min6')" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField name="confirmPassword" v-slot="{ componentField }">
                <FormItem>
                  <FormLabel>{{ t('confirm_password') }}</FormLabel>
                  <FormControl>
                    <Input v-bind="componentField" type="password" autocomplete="new-password" :placeholder="t('enter_password_again')" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <div v-if="errorMessage" class="text-sm text-[var(--color-danger)] bg-[var(--color-danger)]/10 border border-[var(--color-danger)]/20 rounded-lg px-3 py-2">
                {{ errorMessage }}
              </div>

              <Button type="submit" class="w-full" :disabled="initializing">
                <Loader2 v-if="initializing" class="w-4 h-4 mr-2 animate-spin" />
                {{ initializing ? t('initializing') : t('start_init') }}
              </Button>
            </Form>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
