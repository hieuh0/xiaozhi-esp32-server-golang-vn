<template>
  <div class="api-tokens-page">
    <div class="page-actions">
      <router-link class="doc-link" to="/openapi-docs">{{ t('view_public_openapi') }}</router-link>
      <el-button type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon>
        {{ t('create_token') }}</el-button>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      <template #title>{{ t('call_method_hint') }}</template>
    </el-alert>

    <el-card class="table-card" shadow="never">
      <el-table :data="tokens" v-loading="loading" :empty-text="t('no_tokens')">
        <el-table-column prop="name" :label="t('name')" min-width="180" />
        <el-table-column prop="token_prefix" :label="t('prefix_col')" min-width="140" />
        <el-table-column :label="t('status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'info'">{{ row.is_active ? t('available') : t('revoked') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('last_used_col')" min-width="170">
          <template #default="{ row }">{{ formatTime(row.last_used_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('expire_time_col')" min-width="170">
          <template #default="{ row }">{{ formatTime(row.expires_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('created_at')" min-width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('actions')" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              type="danger"
              :disabled="!row.is_active"
              @click="handleRevoke(row)"
            >{{ t('revoke') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="showCreate" :title="t('create_api_token_title')" width="480px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item :label="t('token_name_label')" prop="name">
          <el-input v-model="form.name" maxlength="100" :placeholder="t('token_name_ph')" />
        </el-form-item>
        <el-form-item :label="t('valid_days_label')">
          <el-input-number v-model="form.expires_in_days" :min="0" :max="3650" />
          <div class="form-tip">{{ t('valid_forever_hint') }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">{{ t('create') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showPlainToken" :title="t('save_token_now')" width="640px">
      <el-alert type="warning" :closable="false" show-icon>
        {{ t('plain_token_hint') }}
      </el-alert>
      <el-input class="token-input" v-model="latestToken" type="textarea" :rows="3" readonly />
      <template #footer>
        <el-button @click="showPlainToken = false">{{ t('close') }}</el-button>
        <el-button type="primary" @click="copyToken">{{ t('copy_token') }}</el-button>
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
  await ElMessageBox.confirm(t('confirm_revoke_token', { name: row.name }), t('hint'), {
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
