<template>
  <div class="api-tokens-page">
    <div class="page-actions">
      <router-link class="doc-link" to="/openapi-docs">{{ t('view_public_openapi') }}查看公开 OpenAPI 接口说明</router-link>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon>
        {{ t('create_token') }}</el-button>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      <template #title>
        支持两种调用方式：Authorization: Bearer &lt;token&gt; 或 X-API-Token: &lt;token&gt;
      </template>
    </el-alert>

    <el-card class="table-card" shadow="never">
      <el-table :data="tokens" v-loading="loading" empty-text="暂无 Token，请先创建">
        <el-table-column prop="name" :label="t('name')" min-width="180" />
        <el-table-column prop="token_prefix" label="前缀" min-width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'info'">{{ row.is_active ? t('available') : t('revoked') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后使用" min-width="170">
          <template #default="{ row }">{{ formatTime(row.last_used_at) }}</template>
        </el-table-column>
        <el-table-column label="过期时间" min-width="170">
          <template #default="{ row }">{{ formatTime(row.expires_at) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              type="danger"
              :disabled="!row.is_active"
              @click="handleRevoke(row)"
            >
              吊销
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="showCreate" title="创建 API Token" width="480px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="Token 名称" prop="name">
          <el-input v-model="form.name" maxlength="100" placeholder="例如：生产环境调用" />
        </el-form-item>
        <el-form-item label="有效天数">
          <el-input-number v-model="form.expires_in_days" :min="0" :max="3650" />
          <div class="form-tip">0 表示永不过期</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showPlainToken" title="请立即保存 Token" width="640px">
      <el-alert type="warning" :closable="false" show-icon>
        明文 Token 后续无法再次查看，请立即复制并安全保存。
      </el-alert>
      <el-input class="token-input" v-model="latestToken" type="textarea" :rows="3" readonly />
      <template #footer>
        <el-button @click="showPlainToken = false">关闭</el-button>
        <el-button type="primary" @click="copyToken">复制 Token</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const loading = ref(false)
const creating = ref(false)
const tokens = ref([])
const showCreate = ref(false)
const showPlainToken = ref(false)
const latestToken = ref('')
const formRef = ref()

const form = reactive({
  name: '',
  expires_in_days: 0
})

const rules = {
  name: [{ required: true, message: t('enter_token_name'), trigger: 'blur' }]
}

const formatTime = (val) => {
  if (!val) return '-'
  return new Date(val).toLocaleString()
}

const loadTokens = async () => {
  loading.value = true
  try {
    const res = await api.get('/user/api-tokens')
    tokens.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

const openCreateDialog = () => {
  form.name = ''
  form.expires_in_days = 0
  showCreate.value = true
}

const handleCreate = async () => {
  if (!formRef.value) return
  await formRef.value.validate()

  creating.value = true
  try {
    const res = await api.post('/user/api-tokens', form)
    latestToken.value = res.data?.data?.token || ''
    showCreate.value = false
    showPlainToken.value = true
    ElMessage.success(t('token_created'))
    await loadTokens()
  } finally {
    creating.value = false
  }
}

const handleRevoke = async (row) => {
  await ElMessageBox.confirm(`确定吊销 Token「${row.name}」吗？`, t('hint'), {
    confirmButtonText: t('confirm'),
    cancelButtonText: t('cancel'),
    type: 'warning'
  })
  await api.delete(`/user/api-tokens/${row.id}`)
  ElMessage.success(t('token_revoked'))
  await loadTokens()
}

const copyToken = async () => {
  if (!latestToken.value) return
  await navigator.clipboard.writeText(latestToken.value)
  ElMessage.success(t('token_copied'))
}

onMounted(loadTokens)
</script>

<style scoped>
.api-tokens-page { padding: 8px; }
.page-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.doc-link {
  display: inline-block;
  color: var(--apple-primary);
  text-decoration: none;
  font-size: 13px;
}
.doc-link:hover { text-decoration: underline; }
.table-card { margin-top: 12px; }
.form-tip { color: #909399; font-size: 12px; margin-top: 6px; }
.token-input { margin-top: 12px; }
</style>
