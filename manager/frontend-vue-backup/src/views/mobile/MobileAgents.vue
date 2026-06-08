<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Bot, ChevronRight, Plus } from '@lucide/vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'

const { t } = useLocale()
const router = useRouter()

const agents = ref([])
const loading = ref(false)

const loadAgents = async () => {
  loading.value = true
  try {
    const res = await api.get('/user/agents')
    agents.value = res.data.data || []
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('load_agent_list_failed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadAgents)
</script>

<template>
  <div class="p-4 pb-8">
    <!-- Toolbar -->
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-base font-bold text-[var(--color-text)]">{{ t('my_agents') }}</h2>
      <Button size="sm" @click="router.push('/agents')">
        <Plus class="w-4 h-4 mr-1" />{{ t('add_agent') }}
      </Button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="space-y-3">
      <div v-for="i in 3" :key="i" class="h-[72px] rounded-2xl bg-[var(--color-surface-muted)] animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="agents.length === 0" class="py-16 flex flex-col items-center gap-4 text-[var(--color-text-secondary)]">
      <Bot class="w-14 h-14 opacity-25" />
      <p class="text-sm">{{ t('create_first_agent') }}</p>
      <Button @click="router.push('/agents')">{{ t('add_agent') }}</Button>
    </div>

    <!-- Agent cards -->
    <div v-else class="space-y-2.5">
      <button
        v-for="agent in agents"
        :key="agent.id"
        type="button"
        @click="router.push(`/agents/${agent.id}/devices`)"
        class="flex items-center w-full gap-3 p-4 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] hover:bg-[var(--color-surface-muted)] active:scale-[0.99] transition-all text-left"
      >
        <div class="flex items-center justify-center w-11 h-11 rounded-xl bg-blue-50 dark:bg-blue-900/30 text-[var(--color-primary)] shrink-0">
          <Bot class="w-5 h-5" />
        </div>
        <div class="flex-1 min-w-0">
          <div class="text-sm font-semibold text-[var(--color-text)] truncate">{{ agent.name || t('unnamed_agent') }}</div>
          <div class="text-xs text-[var(--color-text-secondary)] mt-0.5">
            {{ t('device') }}: {{ agent.device_count ?? 0 }} &nbsp;·&nbsp; {{ t('online') }}: {{ agent.online_device_count ?? 0 }}
          </div>
        </div>
        <ChevronRight class="w-4 h-4 text-[var(--color-text-tertiary)] shrink-0" />
      </button>
    </div>
  </div>
</template>
