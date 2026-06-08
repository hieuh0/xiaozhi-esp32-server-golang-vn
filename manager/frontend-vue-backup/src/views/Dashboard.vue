<script setup>
import { ref, computed, onMounted } from 'vue'
import { User, Monitor, Wifi, Settings, Download, Upload, Cpu, Wand2 } from '@lucide/vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import api from '@/utils/api'
import { useLocale } from '../composables/useLocale'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import DashboardServiceCard from '@/components/DashboardServiceCard.vue'

const { t } = useLocale()
const authStore = useAuthStore()

const stats = ref({ totalUsers: 0, totalDevices: 0, totalAgents: 0, onlineDevices: 0 })
const programStartedAt = ref('—')
const fileInput = ref(null)

const metricCards = computed(() => [
  {
    icon: User, iconClass: 'text-[var(--color-primary)] bg-[var(--color-primary-soft)]',
    trend: authStore.isAdmin ? t('global_user') : t('linked_account'),
    value: authStore.isAdmin ? stats.value.totalUsers : 1,
    label: authStore.isAdmin ? t('total_users') : t('current_logged_account')
  },
  {
    icon: Monitor, iconClass: 'text-green-700 bg-green-50 dark:text-green-400 dark:bg-green-900/30',
    trend: t('online_devices'),
    value: stats.value.totalDevices,
    label: authStore.isAdmin ? t('total_devices') : t('my_devices')
  },
  {
    icon: Cpu, iconClass: 'text-amber-700 bg-amber-50 dark:text-amber-400 dark:bg-amber-900/30',
    trend: t('active'),
    value: stats.value.totalAgents,
    label: authStore.isAdmin ? t('agent_count') : t('my_agents')
  },
  {
    icon: Wifi, iconClass: 'text-red-700 bg-red-50 dark:text-red-400 dark:bg-red-900/30',
    trend: t('realtime_monitoring'),
    value: stats.value.onlineDevices,
    label: t('online_devices')
  },
])

const loadStats = async () => {
  try {
    const res = await api.get('/dashboard/stats')
    stats.value = {
      totalUsers: res.data.totalUsers || 0,
      totalDevices: res.data.totalDevices || 0,
      totalAgents: res.data.totalAgents || 0,
      onlineDevices: res.data.onlineDevices || 0
    }
    programStartedAt.value = res.data?.programStartedAt
      ? new Date(res.data.programStartedAt).toLocaleString('zh-CN')
      : '—'
  } catch (error) {
    console.error(t('load_stats_failed_v2'), error)
  }
}

const exportConfig = async () => {
  try {
    const res = await fetch('/api/admin/configs/export', {
      headers: { Authorization: `Bearer ${authStore.token}` }
    })
    if (res.ok) {
      const url = URL.createObjectURL(await res.blob())
      const a = Object.assign(document.createElement('a'), { href: url, download: 'config.yaml' })
      document.body.appendChild(a)
      a.click()
      URL.revokeObjectURL(url)
      document.body.removeChild(a)
      ElMessage.success(t('config_export_success'))
    } else {
      ElMessage.error(t('config_export_failed'))
    }
  } catch {
    ElMessage.error(t('config_export_failed'))
  }
}

const handleFileChange = async (e) => {
  const file = e.target.files[0]
  if (!file) return
  const ext = file.name.toLowerCase().substring(file.name.lastIndexOf('.'))
  if (!['.yaml', '.yml', '.json'].includes(ext)) {
    ElMessage.error(t('select_yaml_or_json'))
    return
  }
  const fd = new FormData()
  fd.append('file', file)
  try {
    const res = await fetch('/api/admin/configs/import', {
      method: 'POST', headers: { Authorization: `Bearer ${authStore.token}` }, body: fd
    })
    if (res.ok) {
      ElMessage.success(t('config_import_success'))
    } else {
      const err = await res.json()
      ElMessage.error(err.error || t('config_import_failed'))
    }
  } catch {
    ElMessage.error(t('config_import_failed'))
  }
  e.target.value = ''
}

onMounted(loadStats)
</script>

<template>
  <div class="grid gap-5">
    <!-- Stats grid -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <div v-for="m in metricCards" :key="m.label"
        class="p-5 rounded-xl bg-[var(--color-surface)] border border-[var(--color-line)]"
      >
        <div class="flex items-center justify-between mb-4">
          <span class="w-10 h-10 rounded-2xl inline-flex items-center justify-center" :class="m.iconClass">
            <component :is="m.icon" class="w-5 h-5" />
          </span>
          <span class="text-xs font-semibold text-[var(--color-text-secondary)]">{{ m.trend }}</span>
        </div>
        <strong class="block text-[34px] font-bold tracking-tight leading-none text-[var(--color-text)]">{{ m.value }}</strong>
        <p class="mt-2.5 text-sm text-[var(--color-text-secondary)]">{{ m.label }}</p>
      </div>
    </div>

    <!-- Main content grid -->
    <div class="grid lg:grid-cols-[1.3fr_360px] gap-4 items-start">
      <div class="grid gap-4">
        <!-- Service address (admin) -->
        <DashboardServiceCard v-if="authStore.isAdmin" />

        <!-- Config management (admin) -->
        <Card v-if="authStore.isAdmin">
          <CardHeader class="pb-3">
            <p class="text-[11px] font-bold tracking-widest text-[var(--color-text-tertiary)] uppercase mb-1">CONFIGURATION</p>
            <h3 class="text-lg font-semibold text-[var(--color-text)]">{{ t('config_management') }}</h3>
          </CardHeader>
          <CardContent class="grid gap-3">
            <button class="flex items-center gap-3.5 w-full p-4 rounded-xl border border-[var(--color-primary)]/20 bg-[var(--color-primary-soft)] hover:-translate-y-px hover:shadow-sm transition-all text-left" @click="$router.push('/admin/config-wizard')">
              <span class="w-10 h-10 rounded-2xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><Wand2 class="w-5 h-5" /></span>
              <span class="flex flex-col gap-0.5"><strong class="text-[15px] text-[var(--color-text)]">{{ t('config_wizard') }}</strong><small class="text-sm text-[var(--color-text-secondary)]">{{ t('from_wizard_desc') }}</small></span>
            </button>
            <button class="flex items-center gap-3.5 w-full p-4 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:-translate-y-px hover:shadow-sm transition-all text-left" @click="exportConfig">
              <span class="w-10 h-10 rounded-2xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><Download class="w-5 h-5" /></span>
              <span class="flex flex-col gap-0.5"><strong class="text-[15px] text-[var(--color-text)]">{{ t('export_config_title') }}</strong><small class="text-sm text-[var(--color-text-secondary)]">{{ t('export_config_desc') }}</small></span>
            </button>
            <button class="flex items-center gap-3.5 w-full p-4 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:-translate-y-px hover:shadow-sm transition-all text-left" @click="fileInput.click()">
              <span class="w-10 h-10 rounded-2xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><Upload class="w-5 h-5" /></span>
              <span class="flex flex-col gap-0.5"><strong class="text-[15px] text-[var(--color-text)]">{{ t('import_config_title') }}</strong><small class="text-sm text-[var(--color-text-secondary)]">{{ t('import_config_desc') }}</small></span>
            </button>
            <input ref="fileInput" type="file" accept=".yaml,.yml,.json" class="hidden" @change="handleFileChange" />
          </CardContent>
        </Card>
      </div>

      <!-- Sidebar -->
      <div class="grid gap-4">
        <!-- System info -->
        <Card>
          <CardHeader class="pb-3">
            <p class="text-[11px] font-bold tracking-widest text-[var(--color-text-tertiary)] uppercase mb-1">SYSTEM</p>
            <h3 class="text-lg font-semibold text-[var(--color-text)]">{{ t('system_info') }}</h3>
          </CardHeader>
          <CardContent>
            <dl class="divide-y divide-[var(--color-line)]">
              <div v-for="row in [
                { label: t('system_version'), value: 'v1.0.0', plain: true },
                { label: t('program_start_time'), value: programStartedAt, plain: true },
                { label: t('current_user_label'), value: authStore.user?.username || '—', plain: true },
                { label: t('user_role_label'), value: null, badge: true },
              ]" :key="row.label" class="flex items-center justify-between gap-3 py-3">
                <dt class="text-sm text-[var(--color-text-secondary)]">{{ row.label }}</dt>
                <dd v-if="row.plain" class="text-sm font-semibold text-[var(--color-text)]">{{ row.value }}</dd>
                <dd v-else>
                  <Badge :variant="authStore.isAdmin ? 'destructive' : 'secondary'">
                    {{ authStore.isAdmin ? t('admin') : t('normal_user') }}
                  </Badge>
                </dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <!-- Quick actions -->
        <Card>
          <CardHeader class="pb-3">
            <p class="text-[11px] font-bold tracking-widest text-[var(--color-text-tertiary)] uppercase mb-1">SHORTCUTS</p>
            <h3 class="text-lg font-semibold text-[var(--color-text)]">{{ t('quick_actions') }}</h3>
          </CardHeader>
          <CardContent class="grid gap-3">
            <template v-if="authStore.isAdmin">
              <button v-for="qa in [
                { icon: User, label: t('user_management'), desc: t('view_account_desc'), route: '/admin/users' },
                { icon: Settings, label: t('llm_config'), desc: t('llm_config_desc'), route: '/admin/llm-config' },
                { icon: Cpu, label: t('vad_config'), desc: t('vad_config_desc'), route: '/admin/vad-config' },
              ]" :key="qa.route"
                class="flex items-center gap-3 w-full p-3 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:-translate-y-px hover:shadow-sm transition-all text-left"
                @click="$router.push(qa.route)"
              >
                <span class="w-9 h-9 rounded-xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><component :is="qa.icon" class="w-4 h-4" /></span>
                <span class="flex flex-col gap-0.5 min-w-0"><strong class="text-sm text-[var(--color-text)]">{{ qa.label }}</strong><small class="text-xs text-[var(--color-text-secondary)] truncate">{{ qa.desc }}</small></span>
              </button>
            </template>
            <template v-else>
              <button class="flex items-center gap-3 w-full p-3 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:-translate-y-px transition-all text-left" @click="$router.push('/agents')">
                <span class="w-9 h-9 rounded-xl inline-flex items-center justify-center bg-[var(--color-primary-soft)] text-[var(--color-primary)] shrink-0"><Monitor class="w-4 h-4" /></span>
                <span class="flex flex-col gap-0.5"><strong class="text-sm text-[var(--color-text)]">{{ t('agent_management') }}</strong><small class="text-xs text-[var(--color-text-secondary)]">{{ t('agent_mgmt_desc') }}</small></span>
              </button>
              <p class="text-sm text-[var(--color-text-secondary)]">{{ t('normal_user_quick_hint') }}</p>
            </template>
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>
