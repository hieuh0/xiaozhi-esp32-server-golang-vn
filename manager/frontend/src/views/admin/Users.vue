<template>
  <div class="config-page">
    <div class="page-actions">
      <el-input
        v-model="searchKeyword"
        :placeholder="t('search_user')"
        style="width: 200px"
        prefix-icon="Search"
        clearable
      />
      <el-button type="primary" @click="openAddDialog">
        <el-icon><Plus /></el-icon>
        {{ t('add_user') }}</el-button>
    </div>

    <!-- 用户列表表格 -->
    <el-table :data="filteredUserList" v-loading="tableLoading" style="width: 100%">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="username" :label="t('username')" width="150" />
      <el-table-column prop="email" :label="t('email')" width="200" />
      <el-table-column prop="role" :label="t('role')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'">
            {{ row.role === 'admin' ? t('admin') : t('normal_user') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('created_at')" width="180">
        <template #default="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('actions')" width="360">
        <template #default="{ row }">
          <el-button size="small" @click="openEditDialog(row)">{{ t('edit') }}</el-button>
          <el-button size="small" type="success" @click="openQuotaDialog(row)" :disabled="row.role === 'admin'">{{ t('voice_clone_quota') }}</el-button>
          <el-button size="small" type="warning" @click="openResetPasswordDialog(row)">
            {{ t('reset_password_title') }}
          </el-button>
          <el-button 
            size="small" 
            type="danger" 
            @click="handleDeleteUser(row)"
            :disabled="row.role === 'admin'"
          >
            {{ t('delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑用户对话框 -->
    <el-dialog 
      v-model="userDialogVisible" 
      :title="isEditMode ? t('edit_user') : t('add_user')"
      width="500px"
      @close="resetUserForm"
    >
      <el-form 
        ref="userFormRef" 
        :model="userForm" 
        :rules="userFormRules" 
        label-width="80px"
      >
        <el-form-item :label="t('username')" prop="username">
          <el-input 
            v-model="userForm.username" 
            :disabled="isEditMode"
            :placeholder="t('enter_username')"
          />
        </el-form-item>
        
        <el-form-item :label="t('email')" prop="email">
          <el-input v-model="userForm.email" :placeholder="t('enter_email')" />
        </el-form-item>
        
        <el-form-item v-if="!isEditMode" :label="t('password')" prop="password">
          <el-input 
            v-model="userForm.password" 
            type="password" 
            :placeholder="t('enter_password_min6')"
            show-password
          />
        </el-form-item>
        
        <el-form-item :label="t('role')" prop="role">
          <el-select v-model="userForm.role" :placeholder="t('select_role')" style="width: 100%">
            <el-option :label="t('normal_user')" value="user" />
            <el-option :label="t('admin')" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="userDialogVisible = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" @click="handleUserSubmit" :loading="userSubmitLoading">
          {{ isEditMode ? t('save') : t('add') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog 
      v-model="resetPasswordDialogVisible" 
      :title="t('reset_password_title')"
      width="400px"
      @close="resetPasswordForm"
    >
      <el-form 
        ref="passwordFormRef" 
        :model="passwordForm" 
        :rules="passwordFormRules" 
        label-width="80px"
      >
        <el-form-item :label="t('user')">
          <el-input v-model="currentUser.username" disabled />
        </el-form-item>
        
        <el-form-item :label="t('new_password')" prop="newPassword">
          <el-input 
            v-model="passwordForm.newPassword" 
            type="password" 
            :placeholder="t('new_password_ph')"
            show-password
          />
        </el-form-item>
        
        <el-form-item :label="t('confirm_password')" prop="confirmPassword">
          <el-input 
            v-model="passwordForm.confirmPassword" 
            type="password" 
            :placeholder="t('confirm_password_ph')"
            show-password
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="resetPasswordDialogVisible = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" @click="handleResetPassword" :loading="resetPasswordLoading">{{ t('confirm_reset') }}</el-button>
      </template>
    </el-dialog>

    <!-- 声音复刻额度对话框 -->
    <el-dialog
      v-model="quotaDialogVisible"
      :title="t('voice_clone_quota_title', { name: quotaUser.username || '' })"
      width="900px"
      @close="resetQuotaDialog"
    >
      <div class="quota-hint">{{ t('quota_hint') }}</div>
      <el-table :data="quotaRows" v-loading="quotaLoading" style="margin-top: 12px">
        <el-table-column prop="tts_config_name" :label="t('tts_config_name_col')" min-width="180" />
        <el-table-column prop="tts_config_id" label="TTS Config ID" min-width="180" />
        <el-table-column prop="provider" label="Provider" width="120" />
        <el-table-column :label="t('used_count_col')" width="100">
          <template #default="{ row }">{{ row.used_count }}</template>
        </el-table-column>
        <el-table-column :label="t('remaining_count_col')" width="100">
          <template #default="{ row }">{{ row.remaining_count < 0 ? t('unlimited') : row.remaining_count }}</template>
        </el-table-column>
        <el-table-column :label="t('max_count_col')" width="180">
          <template #default="{ row }">
            <el-input-number v-model="row.max_count" :min="-1" :step="1" :precision="0" controls-position="right" style="width: 140px" />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="quotaDialogVisible = false">{{ t('cancel') }}</el-button>
        <el-button type="primary" :loading="quotaSaving" @click="saveQuotaSettings">{{ t('save_quota') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

// 数据状态
const userList = ref([])
const tableLoading = ref(false)
const userDialogVisible = ref(false)
const resetPasswordDialogVisible = ref(false)
const userSubmitLoading = ref(false)
const resetPasswordLoading = ref(false)
const quotaDialogVisible = ref(false)
const quotaLoading = ref(false)
const quotaSaving = ref(false)
const quotaRows = ref([])
const quotaUser = ref({})
const quotaOriginalMaxMap = ref({})
const isEditMode = ref(false)
const currentUser = ref({})
const searchKeyword = ref('')

// 计算属性
const filteredUserList = computed(() => {
  if (!searchKeyword.value) {
    return userList.value
  }
  return userList.value.filter(user => 
    user.username.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    user.email.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

// 表单引用
const userFormRef = ref()
const passwordFormRef = ref()

// 用户表单数据
const userForm = reactive({
  username: '',
  email: '',
  password: '',
  role: ''
})

// 密码表单数据
const passwordForm = reactive({
  newPassword: '',
  confirmPassword: ''
})

// 用户表单验证规则
const userFormRules = {
  username: [
    { required: true, message: t('enter_username'), trigger: 'blur' }
  ],
  email: [
    { required: true, message: t('enter_email'), trigger: 'blur' },
    { type: 'email', message: t('enter_valid_email'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('enter_password'), trigger: 'blur' },
    { min: 6, message: t('password_min_length'), trigger: 'blur' }
  ],
  role: [
    { required: true, message: t('select_role'), trigger: 'change' }
  ]
}

// 密码表单验证规则
const passwordFormRules = {
  newPassword: [
    { required: true, message: t('enter_new_password'), trigger: 'blur' },
    { min: 6, message: t('password_min_length'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('confirm_password_prompt'), trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== passwordForm.newPassword) {
          callback(new Error(t('password_mismatch')))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// 加载用户列表
const loadUserList = async () => {
  tableLoading.value = true
  try {
    const response = await api.get('/admin/users')
    userList.value = response.data.data || []
  } catch (error) {
    ElMessage.error(t('load_user_list_failed'))
  } finally {
    tableLoading.value = false
  }
}

// 打开添加用户对话框
const openAddDialog = () => {
  isEditMode.value = false
  userDialogVisible.value = true
}

// 打开编辑用户对话框
const openEditDialog = (user) => {
  isEditMode.value = true
  currentUser.value = user
  userForm.username = user.username
  userForm.email = user.email
  userForm.role = user.role
  userDialogVisible.value = true
}

// 重置用户表单
const resetUserForm = () => {
  userForm.username = ''
  userForm.email = ''
  userForm.password = ''
  userForm.role = ''
  currentUser.value = {}
  if (userFormRef.value) {
    userFormRef.value.resetFields()
  }
}

// 处理用户提交
const handleUserSubmit = async () => {
  if (!userFormRef.value) return
  
  try {
    await userFormRef.value.validate()
    userSubmitLoading.value = true
    
    if (isEditMode.value) {
      // 编辑用户
      await api.put(`/admin/users/${currentUser.value.id}`, {
        email: userForm.email,
        role: userForm.role
      })
      ElMessage.success(t('user_update_success'))
    } else {
      // 添加用户
      await api.post('/admin/users', {
        username: userForm.username,
        email: userForm.email,
        password: userForm.password,
        role: userForm.role
      })
      ElMessage.success(t('user_add_success'))
    }
    
    userDialogVisible.value = false
    loadUserList()
  } catch (error) {
    ElMessage.error(isEditMode.value ? t('update_user_failed') : t('add_user_failed'))
  } finally {
    userSubmitLoading.value = false
  }
}

// 删除用户
const handleDeleteUser = async (user) => {
  try {
    await ElMessageBox.confirm(
      t('confirm_delete_user', { name: user.username }),
      t('delete_confirm'),
      {
        confirmButtonText: t('confirm'),
        cancelButtonText: t('cancel'),
        type: 'warning'
      }
    )
    
    await api.delete(`/admin/users/${user.id}`)
    ElMessage.success(t('user_delete_success'))
    loadUserList()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('delete_user_failed'))
    }
  }
}

// 打开重置密码对话框
const openResetPasswordDialog = (user) => {
  currentUser.value = user
  resetPasswordDialogVisible.value = true
}

// 打开复刻额度设置
const openQuotaDialog = async (user) => {
  quotaUser.value = user
  quotaDialogVisible.value = true
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
  } catch (error) {
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
    if (!item.tts_config_id) {
      ElMessage.error(t('invalid_tts_config_id'))
      return
    }
    if (!Number.isInteger(item.max_count) || item.max_count < -1) {
      ElMessage.error(t('max_count_integer_error'))
      return
    }
  }

  const items = normalizedItems.filter((item) => quotaOriginalMaxMap.value[item.tts_config_id] !== item.max_count)
  if (items.length === 0) {
    ElMessage.info(t('quota_unchanged'))
    return
  }

  quotaSaving.value = true
  try {
    await api.put(`/admin/users/${quotaUser.value.id}/voice-clone-quotas`, { items })
    ElMessage.success(t('clone_quota_save_success'))
    await loadQuotaSettings(quotaUser.value.id)
  } catch (error) {
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

// 重置密码表单
const resetPasswordForm = () => {
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  if (passwordFormRef.value) {
    passwordFormRef.value.resetFields()
  }
}

// 处理重置密码
const handleResetPassword = async () => {
  if (!passwordFormRef.value) return
  
  try {
    await passwordFormRef.value.validate()
    
    await ElMessageBox.confirm(
      t('confirm_reset_password', { name: currentUser.value.username }),
      t('reset_password_confirm'),
      {
        confirmButtonText: t('confirm'),
        cancelButtonText: t('cancel'),
        type: 'warning'
      }
    )
    
    resetPasswordLoading.value = true
    
    await api.post(`/admin/users/${currentUser.value.id}/reset-password`, {
      new_password: passwordForm.newPassword
    })
    
    ElMessage.success(t('password_reset_success'))
    resetPasswordDialogVisible.value = false
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('reset_password_failed'))
    }
  } finally {
    resetPasswordLoading.value = false
  }
}

// 格式化日期时间
const formatDateTime = (dateString) => {
  if (!dateString) return '--'
  return new Date(dateString).toLocaleString('zh-CN')
}

// 组件挂载时加载数据
onMounted(() => {
  loadUserList()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.quota-hint {
  color: #666;
  font-size: 13px;
}
</style>
