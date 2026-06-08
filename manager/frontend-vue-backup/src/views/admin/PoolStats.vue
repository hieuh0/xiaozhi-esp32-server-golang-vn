<script setup>
import { h, ref, computed, onMounted, onUnmounted } from 'vue'
import { RefreshCw } from '@lucide/vue'
import api from '@/utils/api'
import { ElMessage } from 'element-plus'
import { useLocale } from '../../composables/useLocale'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import DataTable from '@/components/ui/data-table.vue'

const { t } = useLocale()

const latestStats = ref(null)
const summary = ref({
  total_records: 0,
  oldest_timestamp: null,
  newest_timestamp: null
})

const tableData = computed(() => {
  if (!latestStats.value?.stats) return []
  return Object.entries(latestStats.value.stats)
    .filter(([, v]) => v && typeof v === 'object')
    .map(([poolKey, s]) => ({
      poolKey,
      total: s.total_resources || 0,
      available: s.available_resources || 0,
      inUse: s.in_use_resources || 0,
      maxSize: s.max_size || 0,
      minSize: s.min_size || 0,
      maxIdle: s.max_idle || 0,
      isClosed: s.is_closed || false
    }))
})

const columns = computed(() => [
  { accessorKey: 'poolKey', header: t('pool_key_col') },
  { accessorKey: 'total', header: t('total_resources') },
  { accessorKey: 'available', header: t('available_resources') },
  { accessorKey: 'inUse', header: t('in_use') },
  { accessorKey: 'maxSize', header: t('max_capacity') },
  { accessorKey: 'minSize', header: t('min_capacity') },
  { accessorKey: 'maxIdle', header: t('max_idle') },
  {
    accessorKey: 'isClosed',
    header: t('status'),
    cell: ({ row }) => h(Badge, {
      variant: row.original.isClosed ? 'destructive' : 'secondary'
    }, () => row.original.isClosed ? t('closed') : t('running'))
  }
])

const formatTime = (ts) => {
  if (!ts) return '—'
  return new Date(ts).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit'
  })
}

const loadSummary = async () => {
  try {
    const res = await api.get('/admin/pool/stats/summary')
    summary.value = res.data?.data || {}
  } catch (error) {
    console.error(t('load_stats_summary_failed'), error)
  }
}

const loadStats = async () => {
  try {
    const res = await api.get('/admin/pool/stats?type=latest')
    latestStats.value = res.data?.data || res.data || null
  } catch (error) {
    console.error(t('load_stats_failed_v2'), error)
    ElMessage.error(t('load_stats_failed'))
  }
}

const refreshStats = () => {
  loadSummary()
  loadStats()
  ElMessage.success(t('refresh_success'))
}

let refreshTimer = null
onMounted(() => {
  loadSummary()
  loadStats()
  refreshTimer = setInterval(loadStats, 30000)
})
onUnmounted(() => clearInterval(refreshTimer))
</script>

<template>
  <Card>
    <CardHeader class="flex-row items-center justify-between pb-4">
      <h3 class="text-lg font-semibold text-[var(--color-text)]">{{ t('resource_pool_stats') }}</h3>
      <Button variant="outline" size="sm" @click="refreshStats">
        <RefreshCw class="w-4 h-4 mr-1.5" />{{ t('refresh') }}
      </Button>
    </CardHeader>
    <CardContent class="space-y-6">

      <!-- Summary stats -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="text-center">
          <div class="text-2xl font-bold text-[var(--color-text)]">{{ summary.total_records || 0 }}</div>
          <div class="text-sm text-[var(--color-text-secondary)] mt-1">{{ t('total_records') }}</div>
        </div>
        <div class="text-center">
          <div class="text-2xl font-bold text-[var(--color-text)]">{{ t('latest_only') }}</div>
          <div class="text-sm text-[var(--color-text-secondary)] mt-1">{{ t('storage_mode') }}</div>
        </div>
        <div class="text-center">
          <div class="text-sm font-semibold text-[var(--color-text)] break-all">{{ formatTime(summary.oldest_timestamp) }}</div>
          <div class="text-sm text-[var(--color-text-secondary)] mt-1">{{ t('earliest_time') }}</div>
        </div>
        <div class="text-center">
          <div class="text-sm font-semibold text-[var(--color-text)] break-all">{{ formatTime(summary.newest_timestamp) }}</div>
          <div class="text-sm text-[var(--color-text-secondary)] mt-1">{{ t('latest_time') }}</div>
        </div>
      </div>

      <!-- Divider + timestamp -->
      <div v-if="latestStats" class="border-t border-[var(--color-line)] pt-4">
        <p class="text-xs text-[var(--color-text-secondary)] mb-4">
          {{ t('latest_stats_title', { time: formatTime(latestStats.timestamp) }) }}
        </p>
        <DataTable :columns="columns" :data="tableData" />
      </div>

      <!-- Empty state -->
      <div v-else class="py-12 text-center text-[var(--color-text-secondary)]">
        {{ t('no_statistics') }}
      </div>
    </CardContent>
  </Card>
</template>
