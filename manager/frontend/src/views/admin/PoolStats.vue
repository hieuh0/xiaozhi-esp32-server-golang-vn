<template>
  <div class="pool-stats">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ t('resource_pool_stats') }}</span>
          <div class="header-actions">
            <el-button type="primary" size="small" @click="refreshStats">
              <el-icon><Refresh /></el-icon>
              {{ t('refresh') }}
            </el-button>
            <el-select v-model="viewType" size="small" style="width: 120px; margin-left: 10px;" disabled>
              <el-option :label="t('latest_data')" value="latest" />
            </el-select>
          </div>
        </div>
      </template>

      <!-- 统计摘要 -->
      <el-row :gutter="20" style="margin-bottom: 20px;">
        <el-col :span="6">
          <el-statistic :title="t('total_records')" :value="summary.total_records || 0" />
        </el-col>
        <el-col :span="6">
          <div class="stat-item">
            <div class="stat-title">{{ t('storage_mode') }}</div>
            <div class="stat-value">{{ t('latest_only') }}</div>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="stat-item">
            <div class="stat-title">{{ t('earliest_time') }}</div>
            <div class="stat-value">{{ formatTime(summary.oldest_timestamp) }}</div>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="stat-item">
            <div class="stat-title">{{ t('latest_time') }}</div>
            <div class="stat-value">{{ formatTime(summary.newest_timestamp) }}</div>
          </div>
        </el-col>
      </el-row>

      <!-- 最新统计数据 -->
      <div v-if="viewType === 'latest' && latestStats">
        <el-divider>{{ t('latest_stats_title', { time: formatTime(latestStats.timestamp) }) }}</el-divider>
        <el-table :data="formatStatsData(latestStats.stats)" border stripe style="width: 100%" v-if="latestStats.stats">
          <el-table-column prop="poolKey" :label="t('pool_key_col')" width="200" />
          <el-table-column prop="total" :label="t('total_resources')" width="120" />
          <el-table-column prop="available" :label="t('available_resources')" width="120" />
          <el-table-column prop="inUse" :label="t('in_use')" width="120" />
          <el-table-column prop="maxSize" :label="t('max_capacity')" width="120" />
          <el-table-column prop="minSize" :label="t('min_capacity')" width="120" />
          <el-table-column prop="maxIdle" :label="t('max_idle')" width="120" />
          <el-table-column prop="isClosed" :label="t('status')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.isClosed ? 'danger' : 'success'">
                {{ row.isClosed ? t('closed') : t('running') }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 空状态 -->
      <el-empty v-if="!latestStats" :description="t('no_statistics')" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import api from '@/utils/api'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const viewType = ref('latest')
const latestStats = ref(null)
const summary = ref({
  total_records: 0,
  storage_duration: t('save_latest_only'),
  oldest_timestamp: null,
  newest_timestamp: null
})

let refreshTimer = null

onMounted(() => {
  loadSummary()
  loadStats()
  // 每30秒自动刷新
  refreshTimer = setInterval(() => {
    loadStats()
  }, 30000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})

// 加载统计摘要
const loadSummary = async () => {
  try {
    const response = await api.get('/admin/pool/stats/summary')
    // 后端返回格式: { data: { data: {...} } }
    summary.value = response.data?.data || {}
  } catch (error) {
    console.error(t('load_stats_summary_failed'), error)
  }
}

// 加载统计数据
const loadStats = async () => {
  try {
    const response = await api.get('/admin/pool/stats?type=latest')
    console.log(t('latest_stats_response'), response)
    // 后端返回格式: { data: { timestamp: "...", stats: {...} } }
    // axios 会自动解析，所以 response.data 就是后端返回的 { data: {...} }
    // 需要再取一层 data
    latestStats.value = response.data?.data || response.data || null
    console.log(t('parsed_latest_data'), latestStats.value)
  } catch (error) {
    console.error(t('load_stats_failed_v2'), error)
    ElMessage.error(t('load_stats_failed'))
  }
}

// 刷新统计数据
const refreshStats = () => {
  loadSummary()
  loadStats()
  ElMessage.success(t('refresh_success'))
}

// 格式化统计数据
const formatStatsData = (stats) => {
  if (!stats || typeof stats !== 'object') {
    return []
  }

  const result = []
  for (const [poolKey, poolStats] of Object.entries(stats)) {
    if (poolStats && typeof poolStats === 'object') {
      result.push({
        poolKey,
        total: poolStats.total_resources || 0,
        available: poolStats.available_resources || 0,
        inUse: poolStats.in_use_resources || 0,
        maxSize: poolStats.max_size || 0,
        minSize: poolStats.min_size || 0,
        maxIdle: poolStats.max_idle || 0,
        isClosed: poolStats.is_closed || false
      })
    }
  }
  return result
}

// 格式化时间
const formatTime = (timestamp) => {
  if (!timestamp) {
    return '-'
  }
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

</script>

<style scoped>
.pool-stats {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
}

.el-statistic {
  text-align: center;
}

.el-timeline {
  padding-left: 20px;
}

.stat-item {
  text-align: center;
  padding: 10px;
}

.stat-title {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}
</style>
