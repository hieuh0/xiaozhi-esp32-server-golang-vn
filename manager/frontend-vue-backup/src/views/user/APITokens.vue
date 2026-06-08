<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@lucide/vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableEmpty } from '@/components/ui/table'

const { t } = useLocale()
const { formatDate: formatTime } = useFormatDate()

const loading = ref(false)
const creating = ref(false)
const tokens = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const showCreate = ref(false)
const showPlainToken = ref(false)
const latestToken = ref('')

const form = reactive({ name: '', expires_in_days: 0 })

const badgeClass = (active) => active
  ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800'
  : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800/40 dark:text-gray-400 dark:border-gray-700'

const totalPages = () => Math.ceil(total.value / pageSize.value) || 1

const loadTokens = async () => {
  loading.value = true
  try {
    const res = await api.get('/user/api-tokens', { params: { page: page.value, page_size: pageSize.value } })
    tokens.value = res.data.data || []
    total.value = res.data.total || 0
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
  if (!form.name.trim()) { ElMessage.error(t('enter_token_name')); return }
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
    confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning'
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

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <router-link to="/openapi-docs" class="text-sm text-[var(--color-primary)] hover:underline">{{ t('view_public_openapi') }}</router-link>
      <Button @click="openCreateDialog">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('create_token') }}
      </Button>
    </div>

    <!-- Info -->
    <div class="rounded-lg border border-blue-200 bg-blue-50 text-blue-800 p-3 text-sm dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-300">
      {{ t('call_method_hint') }}
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
      <div class="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('name') }}</TableHead>
              <TableHead>{{ t('prefix_col') }}</TableHead>
              <TableHead class="w-24">{{ t('status') }}</TableHead>
              <TableHead>{{ t('last_used_col') }}</TableHead>
              <TableHead>{{ t('expire_time_col') }}</TableHead>
              <TableHead>{{ t('created_at') }}</TableHead>
              <TableHead class="w-20 text-center">{{ t('actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="loading">
              <TableCell colspan="7" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
            </TableRow>
            <template v-else>
              <TableEmpty v-if="!tokens.length" />
              <TableRow v-for="row in tokens" :key="row.id">
                <TableCell class="font-medium">{{ row.name }}</TableCell>
                <TableCell class="font-mono text-xs">{{ row.token_prefix }}</TableCell>
                <TableCell><span :class="badgeClass(row.is_active)">{{ row.is_active ? t('available') : t('revoked') }}</span></TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ formatTime(row.last_used_at) }}</TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ formatTime(row.expires_at) }}</TableCell>
                <TableCell class="text-sm text-[var(--color-text-secondary)]">{{ formatTime(row.created_at) }}</TableCell>
                <TableCell class="text-center">
                  <Button variant="ghost" size="sm" :disabled="!row.is_active" class="text-destructive hover:text-destructive" @click="handleRevoke(row)">
                    {{ t('revoke') }}
                  </Button>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
      <span>{{ total }} {{ t('total_items') }}</span>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadTokens()">{{ t('prev') }}</Button>
        <span>{{ page }} / {{ totalPages() }}</span>
        <Button variant="outline" size="sm" :disabled="page >= totalPages()" @click="page++; loadTokens()">{{ t('next') }}</Button>
      </div>
    </div>

    <!-- Create dialog -->
    <Dialog v-model:open="showCreate">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('create_api_token_title') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('token_name_label') }}</label>
            <Input v-model="form.name" maxlength="100" :placeholder="t('token_name_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-medium text-[var(--color-text)]">{{ t('valid_days_label') }}</label>
            <Input v-model.number="form.expires_in_days" type="number" min="0" max="3650" />
            <p class="text-xs text-[var(--color-text-tertiary)]">{{ t('valid_forever_hint') }}</p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showCreate = false">{{ t('cancel') }}</Button>
          <Button :disabled="creating" @click="handleCreate">{{ t('create') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Plain token dialog -->
    <Dialog v-model:open="showPlainToken">
      <DialogContent class="max-w-xl">
        <DialogHeader>
          <DialogTitle>{{ t('save_token_now') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-3 py-2">
          <div class="rounded-lg border border-yellow-200 bg-yellow-50 text-yellow-800 p-3 text-sm dark:bg-yellow-900/20 dark:border-yellow-800 dark:text-yellow-300">
            {{ t('plain_token_hint') }}
          </div>
          <textarea
            :value="latestToken"
            readonly
            rows="3"
            class="dark:bg-input/30 border-input rounded-md border bg-transparent px-2.5 py-2 text-sm font-mono w-full resize-none focus-visible:outline-none"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showPlainToken = false">{{ t('close') }}</Button>
          <Button @click="copyToken">{{ t('copy_token') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
