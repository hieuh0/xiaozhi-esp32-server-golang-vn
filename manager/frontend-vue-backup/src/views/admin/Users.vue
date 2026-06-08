<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, MoreHorizontal } from '@lucide/vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { useFormatDate } from '../../composables/use-format-date'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import NumberInput from '@/components/ui/number-input.vue'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const { t } = useLocale()
const { formatDate } = useFormatDate()

const userList = ref([])
const tableLoading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const userDialogOpen = ref(false)
const resetPasswordDialogOpen = ref(false)
const quotaDialogOpen = ref(false)
const userSubmitLoading = ref(false)
const resetPasswordLoading = ref(false)
const quotaLoading = ref(false)
const quotaSaving = ref(false)
const quotaRows = ref([])
const quotaUser = ref({})
const quotaOriginalMaxMap = ref({})
const isEditMode = ref(false)
const currentUser = ref({})
const searchKeyword = ref('')
const showPassword = ref(false)

const filteredUserList = computed(() => {
  if (!searchKeyword.value) return userList.value
  const kw = searchKeyword.value.toLowerCase()
  return userList.value.filter(u =>
    u.username.toLowerCase().includes(kw) || (u.email || '').toLowerCase().includes(kw)
  )
})

const userForm = reactive({ username: '', email: '', password: '', role: '' })
const passwordForm = reactive({ newPassword: '', confirmPassword: '' })

const validateUserForm = () => {
  if (!isEditMode.value && !String(userForm.username || '').trim()) {
    ElMessage.error(t('enter_username')); return false
  }
  if (!String(userForm.email || '').trim()) {
    ElMessage.error(t('enter_email')); return false
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(userForm.email)) {
    ElMessage.error(t('enter_valid_email')); return false
  }
  if (!isEditMode.value && !String(userForm.password || '').trim()) {
    ElMessage.error(t('enter_password')); return false
  }
  if (!isEditMode.value && userForm.password.length < 6) {
    ElMessage.error(t('password_min_length')); return false
  }
  if (!userForm.role) {
    ElMessage.error(t('select_role')); return false
  }
  return true
}

const validatePasswordForm = () => {
  if (!String(passwordForm.newPassword || '').trim()) {
    ElMessage.error(t('enter_new_password')); return false
  }
  if (passwordForm.newPassword.length < 6) {
    ElMessage.error(t('password_min_length')); return false
  }
  if (!passwordForm.confirmPassword) {
    ElMessage.error(t('confirm_password_prompt')); return false
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    ElMessage.error(t('password_mismatch')); return false
  }
  return true
}

const loadUserList = async () => {
  tableLoading.value = true
  try {
    const response = await api.get('/admin/users', { params: { page: page.value, page_size: pageSize.value } })
    userList.value = response.data.data || []
    total.value = response.data.total || 0
  } catch {
    ElMessage.error(t('load_user_list_failed'))
  } finally {
    tableLoading.value = false
  }
}

const openAddDialog = () => {
  isEditMode.value = false
  userForm.username = ''
  userForm.email = ''
  userForm.password = ''
  userForm.role = ''
  currentUser.value = {}
  showPassword.value = false
  userDialogOpen.value = true
}

const openEditDialog = (user) => {
  isEditMode.value = true
  currentUser.value = user
  userForm.username = user.username
  userForm.email = user.email
  userForm.role = user.role
  userDialogOpen.value = true
}

const resetUserForm = () => {
  userForm.username = ''
  userForm.email = ''
  userForm.password = ''
  userForm.role = ''
  currentUser.value = {}
  showPassword.value = false
}

const handleUserSubmit = async () => {
  if (!validateUserForm()) return
  userSubmitLoading.value = true
  try {
    if (isEditMode.value) {
      await api.put(`/admin/users/${currentUser.value.id}`, { email: userForm.email, role: userForm.role })
      ElMessage.success(t('user_update_success'))
    } else {
      await api.post('/admin/users', {
        username: userForm.username,
        email: userForm.email,
        password: userForm.password,
        role: userForm.role
      })
      ElMessage.success(t('user_add_success'))
    }
    userDialogOpen.value = false
    loadUserList()
  } catch {
    ElMessage.error(isEditMode.value ? t('update_user_failed') : t('add_user_failed'))
  } finally {
    userSubmitLoading.value = false
  }
}

const handleDeleteUser = async (user) => {
  try {
    await ElMessageBox.confirm(
      t('confirm_delete_user', { name: user.username }),
      t('delete_confirm'),
      { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' }
    )
    await api.delete(`/admin/users/${user.id}`)
    ElMessage.success(t('user_delete_success'))
    loadUserList()
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('delete_user_failed'))
  }
}

const openResetPasswordDialog = (user) => {
  currentUser.value = user
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  resetPasswordDialogOpen.value = true
}

const handleResetPassword = async () => {
  if (!validatePasswordForm()) return
  try {
    await ElMessageBox.confirm(
      t('confirm_reset_password', { name: currentUser.value.username }),
      t('reset_password_confirm'),
      { confirmButtonText: t('confirm'), cancelButtonText: t('cancel'), type: 'warning' }
    )
    resetPasswordLoading.value = true
    await api.post(`/admin/users/${currentUser.value.id}/reset-password`, { new_password: passwordForm.newPassword })
    ElMessage.success(t('password_reset_success'))
    resetPasswordDialogOpen.value = false
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(t('reset_password_failed'))
  } finally {
    resetPasswordLoading.value = false
  }
}

const openQuotaDialog = async (user) => {
  quotaUser.value = user
  quotaDialogOpen.value = true
  await loadQuotaSettings(user.id)
}

const loadQuotaSettings = async (userID) => {
  quotaLoading.value = true
  try {
    const response = await api.get(`/admin/users/${userID}/voice-clone-quotas`)
    const quotas = response.data?.data?.quotas || []
    quotaRows.value = quotas.map((item) => ({
      ...item,
      max_count: Number.isFinite(Number(item.max_count)) ? Number(item.max_count) : -1,
      used_count: Number(item.used_count || 0),
      remaining_count: Number.isFinite(Number(item.remaining_count)) ? Number(item.remaining_count) : -1
    }))
    quotaOriginalMaxMap.value = quotaRows.value.reduce((acc, row) => {
      acc[row.tts_config_id] = Number(row.max_count)
      return acc
    }, {})
  } catch {
    ElMessage.error(t('load_clone_quota_failed'))
    quotaRows.value = []
    quotaOriginalMaxMap.value = {}
  } finally {
    quotaLoading.value = false
  }
}

const saveQuotaSettings = async () => {
  if (!quotaUser.value?.id) return
  const normalizedItems = quotaRows.value.map((row) => ({
    tts_config_id: row.tts_config_id,
    max_count: Number(row.max_count)
  }))
  for (const item of normalizedItems) {
    if (!item.tts_config_id) { ElMessage.error(t('invalid_tts_config_id')); return }
    if (!Number.isInteger(item.max_count) || item.max_count < -1) { ElMessage.error(t('max_count_integer_error')); return }
  }
  const items = normalizedItems.filter((item) => quotaOriginalMaxMap.value[item.tts_config_id] !== item.max_count)
  if (items.length === 0) { ElMessage.info(t('quota_unchanged')); return }

  quotaSaving.value = true
  try {
    await api.put(`/admin/users/${quotaUser.value.id}/voice-clone-quotas`, { items })
    ElMessage.success(t('clone_quota_save_success'))
    await loadQuotaSettings(quotaUser.value.id)
  } catch {
    ElMessage.error(t('save_clone_quota_failed'))
  } finally {
    quotaSaving.value = false
  }
}

const resetQuotaDialog = () => {
  quotaRows.value = []
  quotaUser.value = {}
  quotaOriginalMaxMap.value = {}
}

const totalPages = computed(() => Math.ceil(total.value / pageSize.value) || 1)

onMounted(() => { loadUserList() })
</script>

<template>
  <div class="grid gap-4 px-6 pb-8">
    <!-- Toolbar -->
    <div class="flex items-center justify-between gap-4">
      <Input v-model="searchKeyword" :placeholder="t('search_user')" class="max-w-xs" />
      <Button @click="openAddDialog">
        <Plus class="w-4 h-4 mr-1.5" />{{ t('add_user') }}
      </Button>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead class="w-16">ID</TableHead>
            <TableHead>{{ t('username') }}</TableHead>
            <TableHead>{{ t('email') }}</TableHead>
            <TableHead class="w-28">{{ t('role') }}</TableHead>
            <TableHead class="w-44">{{ t('created_at') }}</TableHead>
            <TableHead class="w-16 text-center">{{ t('actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-if="tableLoading">
            <TableCell colspan="6" class="py-10 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
          </TableRow>
          <template v-else>
            <TableEmpty v-if="!filteredUserList.length" />
            <TableRow v-for="row in filteredUserList" :key="row.id">
              <TableCell class="text-[var(--color-text-secondary)] text-xs font-mono">{{ row.id }}</TableCell>
              <TableCell class="font-semibold">{{ row.username }}</TableCell>
              <TableCell class="text-[var(--color-text-secondary)]">{{ row.email }}</TableCell>
              <TableCell>
                <span :class="row.role === 'admin'
                  ? 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800'
                  : 'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800'">
                  {{ row.role === 'admin' ? t('admin') : t('normal_user') }}
                </span>
              </TableCell>
              <TableCell class="text-[var(--color-text-secondary)] text-sm">{{ formatDate(row.created_at) }}</TableCell>
              <TableCell class="text-center">
                <DropdownMenu>
                  <DropdownMenuTrigger as-child>
                    <Button variant="ghost" size="icon" class="h-8 w-8" :aria-label="t('more_actions')">
                      <MoreHorizontal class="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem @click="openEditDialog(row)">{{ t('edit') }}</DropdownMenuItem>
                    <DropdownMenuItem :disabled="row.role === 'admin'" @click="row.role !== 'admin' && openQuotaDialog(row)">{{ t('voice_clone_quota') }}</DropdownMenuItem>
                    <DropdownMenuItem @click="openResetPasswordDialog(row)">{{ t('reset_password_title') }}</DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem :disabled="row.role === 'admin'" class="text-destructive" @click="row.role !== 'admin' && handleDeleteUser(row)">{{ t('delete') }}</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
      <span>{{ total }} {{ t('total_items') }}</span>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--; loadUserList()">{{ t('prev') }}</Button>
        <span>{{ page }} / {{ totalPages }}</span>
        <Button variant="outline" size="sm" :disabled="page >= totalPages" @click="page++; loadUserList()">{{ t('next') }}</Button>
      </div>
    </div>

    <!-- Add/Edit user dialog -->
    <Dialog v-model:open="userDialogOpen" @update:open="v => { if (!v) resetUserForm() }">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ isEditMode ? t('edit_user') : t('add_user') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('username') }}</label>
            <Input v-model="userForm.username" :disabled="isEditMode" :placeholder="t('enter_username')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('email') }}</label>
            <Input v-model="userForm.email" type="email" autocomplete="email" :placeholder="t('enter_email')" />
          </div>
          <div v-if="!isEditMode" class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('password') }}</label>
            <div class="relative">
              <Input v-model="userForm.password" :type="showPassword ? 'text' : 'password'" :placeholder="t('enter_password_min6')" class="pr-16" />
              <button type="button" class="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors" @click="showPassword = !showPassword">
                {{ showPassword ? t('hide') : t('show') }}
              </button>
            </div>
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('role') }}</label>
            <Select :model-value="userForm.role" @update:model-value="v => userForm.role = v">
              <SelectTrigger><SelectValue :placeholder="t('select_role')" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="user">{{ t('normal_user') }}</SelectItem>
                <SelectItem value="admin">{{ t('admin') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="userDialogOpen = false">{{ t('cancel') }}</Button>
          <Button :disabled="userSubmitLoading" @click="handleUserSubmit">{{ isEditMode ? t('save') : t('add') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Reset password dialog -->
    <Dialog v-model:open="resetPasswordDialogOpen">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('reset_password_title') }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-2">
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('user') }}</label>
            <Input :model-value="currentUser.username" disabled />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('new_password') }}</label>
            <Input v-model="passwordForm.newPassword" type="password" :placeholder="t('new_password_ph')" />
          </div>
          <div class="grid gap-1.5">
            <label class="text-sm font-semibold text-[var(--color-text)]">{{ t('confirm_password') }}</label>
            <Input v-model="passwordForm.confirmPassword" type="password" :placeholder="t('confirm_password_ph')" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="resetPasswordDialogOpen = false">{{ t('cancel') }}</Button>
          <Button :disabled="resetPasswordLoading" @click="handleResetPassword">{{ t('confirm_reset') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Voice clone quota dialog -->
    <Dialog v-model:open="quotaDialogOpen" @update:open="v => { if (!v) resetQuotaDialog() }">
      <DialogContent class="max-w-4xl">
        <DialogHeader>
          <DialogTitle>{{ t('voice_clone_quota_title', { name: quotaUser.username || '' }) }}</DialogTitle>
        </DialogHeader>
        <p class="text-sm text-[var(--color-text-secondary)]">{{ t('quota_hint') }}</p>
        <div class="rounded-xl border border-[var(--color-line)] overflow-hidden mt-2">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('tts_config_name_col') }}</TableHead>
                <TableHead>TTS Config ID</TableHead>
                <TableHead class="w-28">Provider</TableHead>
                <TableHead class="w-24">{{ t('used_count_col') }}</TableHead>
                <TableHead class="w-32">{{ t('remaining_count_col') }}</TableHead>
                <TableHead class="w-40">{{ t('max_count_col') }}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="quotaLoading">
                <TableCell colspan="6" class="py-8 text-center text-sm text-[var(--color-text-secondary)]">Loading...</TableCell>
              </TableRow>
              <template v-else>
                <TableEmpty v-if="!quotaRows.length" />
                <TableRow v-for="row in quotaRows" :key="row.tts_config_id">
                  <TableCell class="font-medium">{{ row.tts_config_name }}</TableCell>
                  <TableCell class="font-mono text-xs text-[var(--color-text-secondary)]">{{ row.tts_config_id }}</TableCell>
                  <TableCell>{{ row.provider }}</TableCell>
                  <TableCell>{{ row.used_count }}</TableCell>
                  <TableCell>{{ row.remaining_count < 0 ? t('unlimited') : row.remaining_count }}</TableCell>
                  <TableCell>
                    <NumberInput v-model="row.max_count" :min="-1" :step="1" :precision="0" class="w-32" />
                  </TableCell>
                </TableRow>
              </template>
            </TableBody>
          </Table>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="quotaDialogOpen = false">{{ t('cancel') }}</Button>
          <Button :disabled="quotaSaving" @click="saveQuotaSettings">{{ t('save_quota') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
