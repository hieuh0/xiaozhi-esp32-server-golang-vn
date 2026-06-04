<template>
  <div class="agent-history-page">
    <div class="page-header">
      <div class="header-left">
        <el-button 
          @click="$router.back()" 
          :icon="ArrowLeft" 
          circle 
          size="large"
        />
        <div class="header-context">
          <span class="context-label">{{ t('current_agent') }}当前智能体</span>
          <strong class="context-value">{{ agentName || t('unnamed_agent') }}</strong>
          <p class="context-meta" v-if="total > 0">共 {{ total }} 条消息</p>
        </div>
      </div>
      <div class="header-right">
        <el-button @click="handleExport" :loading="exporting">
          <el-icon><Download /></el-icon>
          {{ t('export_records') }}</el-button>
      </div>
    </div>

    <!-- 筛选面板 -->
    <el-card class="filter-card" shadow="never">
      <el-form :model="filters" inline>
        <el-form-item :label="t('role')">
          <el-select v-model="filters.role" :placeholder="t('all')" clearable style="width: 120px">
            <el-option :label="t('all')" value="" />
            <el-option :label="t('user')" value="user" />
            <el-option :label="t('robot')" value="assistant" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('device')">
          <el-select v-model="filters.device_id" :placeholder="t('all')" clearable style="width: 150px">
            <el-option :label="t('all')" value="" />
            <el-option 
              v-for="device in devices" 
              :key="device.id" 
              :label="device.device_name || device.device_code" 
              :value="device.device_name"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('start_date')">
          <el-date-picker
            v-model="filters.start_date"
            type="date"
            :placeholder="t('select_date')"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            style="width: 150px"
            clearable
          />
        </el-form-item>
        <el-form-item :label="t('end_date')">
          <el-date-picker
            v-model="filters.end_date"
            type="date"
            :placeholder="t('select_date')"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            style="width: 150px"
            clearable
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">{{ t('query') }}查询</el-button>
          <el-button @click="handleReset">{{ t('reset') }}重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 消息列表 - 微信风格 -->
    <el-card class="messages-card" shadow="never" v-loading="loading">
      <div v-if="messages.length === 0" class="empty-state">
        <el-empty :description="t('no_chat_history')" />
      </div>
      <div v-else class="chat-container">
        <div class="chat-messages" ref="chatMessagesRef">
          <div 
            v-for="(message, index) in messages" 
            :key="message.id" 
            class="message-wrapper"
            :class="{ 'message-right': message.role === 'user', 'message-left': message.role === 'assistant' }"
          >
            <!-- 时间戳（如果与上一条消息时间间隔超过5分钟，显示时间） -->
            <div v-if="shouldShowTime(message, index)" class="message-time-divider">
              {{ formatTimeShort(message.created_at) }}
            </div>
            
            <div class="message-bubble-wrapper">
              <!-- 左侧：机器人消息 -->
              <template v-if="message.role === 'assistant'">
                <div class="message-bubble message-bubble-left">
                  <div class="message-content-wrapper">
                    <!-- 文本内容 -->
                    <div v-if="message.content" class="message-text">{{ message.content }}</div>
                    <!-- 音频播放器 -->
                    <div v-if="message.audio_path" class="audio-bubble">
                      <audio
                        :ref="el => audioRefs[message.id] = el"
                        :src="audioBlobUrls[message.id]"
                        @ended="handleAudioEnded(message.id)"
                        @error="handleAudioError(message.id)"
                      />
                      <el-button 
                        :icon="playingAudioId === message.id ? VideoPause : VideoPlay"
                        circle
                        size="small"
                        @click="toggleAudio(message.id)"
                        class="audio-play-btn-simple"
                      />
                    </div>
                    <div class="message-meta">
                      <span class="message-time-small">{{ formatTimeShort(message.created_at) }}</span>
                      <el-dropdown trigger="click" @command="handleMessageAction">
                        <el-icon class="message-more"><MoreFilled /></el-icon>
                        <template #dropdown>
                          <el-dropdown-menu>
                            <el-dropdown-item :command="{action: 'delete', id: message.id}">{{ t('delete') }}删除</el-dropdown-item>
                          </el-dropdown-menu>
                        </template>
                      </el-dropdown>
                    </div>
                  </div>
                </div>
              </template>
              
              <!-- 右侧：用户消息 -->
              <template v-else>
                <div class="message-bubble message-bubble-right">
                  <div class="message-content-wrapper">
                    <!-- 文本内容 -->
                    <div v-if="message.content" class="message-text">{{ message.content }}</div>
                    <!-- 音频播放器 -->
                    <div v-if="message.audio_path" class="audio-bubble">
                      <audio
                        :ref="el => audioRefs[message.id] = el"
                        :src="audioBlobUrls[message.id]"
                        @ended="handleAudioEnded(message.id)"
                        @error="handleAudioError(message.id)"
                      />
                      <el-button 
                        :icon="playingAudioId === message.id ? VideoPause : VideoPlay"
                        circle
                        size="small"
                        @click="toggleAudio(message.id)"
                        class="audio-play-btn-simple"
                      />
                    </div>
                    <div class="message-meta">
                      <el-dropdown trigger="click" @command="handleMessageAction">
                        <el-icon class="message-more"><MoreFilled /></el-icon>
                        <template #dropdown>
                          <el-dropdown-menu>
                            <el-dropdown-item :command="{action: 'delete', id: message.id}">删除</el-dropdown-item>
                          </el-dropdown-menu>
                        </template>
                      </el-dropdown>
                      <span class="message-time-small">{{ formatTimeShort(message.created_at) }}</span>
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>

        <!-- 分页 -->
        <div class="pagination" v-if="total > 0">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="total"
            :page-sizes="[20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Download, User, Service, VideoPlay, VideoPause, MoreFilled } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
const { t } = useLocale()

const route = useRoute()
const router = useRouter()

const agentId = computed(() => {
  const id = route.params.id
  return id ? String(id) : null
})
const agentName = ref('')
const loading = ref(false)
const exporting = ref(false)
const messages = ref([])
const total = ref(0)
const devices = ref([])
const deletingId = ref(null)

// 筛选条件
const filters = reactive({
  role: '',
  device_id: '',
  start_date: '',
  end_date: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 50
})

// 计算总页数
const totalPages = computed(() => {
  return Math.ceil(total.value / pagination.pageSize)
})

// 音频播放相关
const audioRefs = ref({})
const playingAudioId = ref(null)
const chatMessagesRef = ref(null)
const audioBlobUrls = ref({}) // 存储音频 Blob URL

// 加载智能体信息
const loadAgent = async () => {
  if (!agentId.value) {
    ElMessage.error(t('agent_id_invalid'))
    router.back()
    return
  }
  try {
    const response = await api.get(`/user/agents/${agentId.value}`)
    agentName.value = response.data.data?.name || t('agent')
  } catch (error) {
    console.error(t('load_agent_info_failed_v2'), error)
    ElMessage.error(t('load_agent_info_failed'))
  }
}

// 加载设备列表
const loadDevices = async () => {
  try {
    const response = await api.get(`/user/agents/${agentId.value}/devices`)
    devices.value = response.data.data || []
  } catch (error) {
    console.error(t('load_device_list_failed_v2'), error)
  }
}

// 加载消息列表
const loadMessages = async () => {
  if (!agentId.value) {
    return
  }
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize
    }
    if (filters.role) params.role = filters.role
    if (filters.device_id) params.device_id = filters.device_id
    if (filters.start_date) params.start_date = filters.start_date
    if (filters.end_date) params.end_date = filters.end_date

    const response = await api.get(`/user/history/agents/${agentId.value}/messages`, { params })
    // 后端返回的是按时间倒序（最新在前），需要反转数组使最新的在底部
    const data = response.data.data || []
    messages.value = [...data].reverse() // 反转数组，最新的在底部
    total.value = response.data.total || 0
    
    // 预加载有音频的消息
    await preloadAudioMessages()
    
    // 加载完成后滚动到底部（显示最新消息）
    await nextTick()
    scrollToBottom()
  } catch (error) {
    ElMessage.error('加载消息列表失败: ' + (error.response?.data?.error || error.message))
    console.error(t('load_message_list_failed'), error)
    messages.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 查询
const handleSearch = () => {
  pagination.page = 1
  loadMessages()
}

// 重置筛选
const handleReset = () => {
  filters.role = ''
  filters.device_id = ''
  filters.start_date = ''
  filters.end_date = ''
  pagination.page = 1
  loadMessages()
}

// 分页变化
const handlePageChange = (page) => {
  pagination.page = page
  loadMessages()
}

const handleSizeChange = (size) => {
  pagination.pageSize = size
  pagination.page = 1
  loadMessages()
}

// 删除消息
const handleDelete = async (messageId) => {
  try {
    await ElMessageBox.confirm(t('confirm_delete_message'), t('hint'), {
      confirmButtonText: t('confirm'),
      cancelButtonText: t('cancel'),
      type: 'warning'
    })
    
    deletingId.value = messageId
    await api.delete(`/user/history/messages/${messageId}`)
    ElMessage.success(t('delete_success'))
    loadMessages()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('delete_failed'))
      console.error(t('delete_message_failed'), error)
    }
  } finally {
    deletingId.value = null
  }
}

// 导出记录
const handleExport = async () => {
  exporting.value = true
  try {
    const params = {
      agent_id: agentId.value
    }
    if (filters.role) params.role = filters.role
    if (filters.device_id) params.device_id = filters.device_id
    if (filters.start_date) params.start_date = filters.start_date
    if (filters.end_date) params.end_date = filters.end_date

    const response = await api.get('/user/history/export', { 
      params,
      responseType: 'blob'
    })
    
    // 创建下载链接
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `chat_history_${new Date().toISOString().slice(0, 10)}.json`)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
    
    ElMessage.success(t('export_success'))
  } catch (error) {
    ElMessage.error(t('export_failed'))
    console.error(t('export_failed_v2'), error)
  } finally {
    exporting.value = false
  }
}

// 格式化时间（完整）
const formatTime = (dateString) => {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 格式化时间（简短，用于消息气泡）
const formatTimeShort = (dateString) => {
  const date = new Date(dateString)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const msgDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  
  // 如果是今天，只显示时间
  if (msgDate.getTime() === today.getTime()) {
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  
  // 如果是昨天
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  if (msgDate.getTime() === yesterday.getTime()) {
    return '昨天 ' + date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  
  // 如果是今年，显示月日和时间
  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1}月${date.getDate()}日 ${date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    })}`
  }
  
  // 其他情况显示完整日期和时间
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 判断是否显示时间分隔线
const shouldShowTime = (message, index) => {
  if (index === 0) return true
  const currentTime = new Date(message.created_at).getTime()
  const prevTime = new Date(messages.value[index - 1].created_at).getTime()
  // 如果与上一条消息间隔超过5分钟，显示时间
  return (currentTime - prevTime) > 5 * 60 * 1000
}

// 处理消息操作
const handleMessageAction = (command) => {
  if (command.action === 'delete') {
    handleDelete(command.id)
  }
}

// 滚动到底部
const scrollToBottom = () => {
  if (chatMessagesRef.value) {
    nextTick(() => {
      chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight
    })
  }
}

// 获取音频URL（使用 Blob URL 以支持认证）
const getAudioUrl = async (messageId) => {
  // 如果已有 Blob URL，直接返回
  if (audioBlobUrls.value[messageId]) {
    return audioBlobUrls.value[messageId]
  }
  
  try {
    // 使用 axios 获取音频数据（会自动携带认证 token）
    const response = await api.get(`/user/history/messages/${messageId}/audio`, {
      responseType: 'blob' // 重要：指定响应类型为 blob
    })
    
    // 创建 Blob URL
    const blobUrl = URL.createObjectURL(response.data)
    audioBlobUrls.value[messageId] = blobUrl
    
    return blobUrl
  } catch (error) {
    // 静默处理，只记录日志，不显示错误提示
    console.warn(t('load_audio_failed'), messageId, error)
    return null
  }
}


// 预加载音频消息
const preloadAudioMessages = async () => {
  const audioMessages = messages.value.filter(msg => msg.audio_path)
  // 并发预加载，但限制并发数
  const promises = audioMessages.slice(0, 10).map(msg => getAudioUrl(msg.id).catch(err => {
    console.warn(t('preload_audio_failed'), msg.id, err)
    return null
  }))
  await Promise.all(promises)
}

// 音频播放结束
const handleAudioEnded = (messageId) => {
  playingAudioId.value = null
}

// 音频加载错误处理
const handleAudioError = async (messageId) => {
  // 静默处理，只记录日志，不显示错误提示
  console.warn(t('audio_load_failed'), messageId)
  // 尝试重新加载
  try {
    const url = await getAudioUrl(messageId)
    if (url) {
      const audio = audioRefs.value[messageId]
      if (audio) {
        audio.load() // 重新加载音频
      }
    }
  } catch (error) {
    // 静默处理，只记录日志
    console.warn(t('audio_reload_failed'), messageId, error)
  }
}

// 切换音频播放
const toggleAudio = async (messageId) => {
  const audio = audioRefs.value[messageId]
  if (!audio) return

  // 如果还没有加载音频，先加载
  if (!audioBlobUrls.value[messageId]) {
    const url = await getAudioUrl(messageId)
    if (!url) {
      // 静默处理，只记录日志，不显示错误提示
      console.warn(t('audio_load_play_failed'), messageId)
      return
    }
    // 等待音频元素加载
    await new Promise((resolve) => {
      audio.onloadeddata = resolve
      audio.load()
    })
  }

  // 停止其他音频
  if (playingAudioId.value && playingAudioId.value !== messageId) {
    const otherAudio = audioRefs.value[playingAudioId.value]
    if (otherAudio) {
      otherAudio.pause()
      otherAudio.currentTime = 0
    }
  }

  if (playingAudioId.value === messageId) {
    // 暂停当前音频
    audio.pause()
    playingAudioId.value = null
  } else {
    // 播放音频
    try {
      await audio.play()
      playingAudioId.value = messageId
    } catch (error) {
      // 静默处理，只记录日志，不显示错误提示
      console.warn(t('play_audio_failed'), messageId, error)
    }
  }
}


onMounted(async () => {
  if (!agentId.value) {
    ElMessage.error(t('agent_id_invalid'))
    router.push('/user/agents')
    return
  }
  try {
    await Promise.all([
      loadAgent(),
      loadDevices(),
      loadMessages()
    ])
  } catch (error) {
    console.error(t('init_failed'), error)
  }
})

// 组件卸载时清理 Blob URL，避免内存泄漏
onBeforeUnmount(() => {
  Object.values(audioBlobUrls.value).forEach(url => {
    if (url) {
      URL.revokeObjectURL(url)
    }
  })
  audioBlobUrls.value = {}
})
</script>

<style scoped>
.agent-history-page {
  padding: 0;
  background: transparent;
  min-height: 100%;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  border-radius: var(--apple-radius-lg);
  box-shadow: var(--apple-shadow-md);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-context {
  display: grid;
  gap: 4px;
}

.context-label {
  color: var(--apple-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.context-value {
  color: var(--apple-text);
  font-size: 16px;
  line-height: 1.3;
}

.context-meta {
  margin: 0;
  color: var(--apple-text-secondary);
  font-size: 14px;
}

.filter-card {
  margin-bottom: 20px;
}

.messages-card {
  min-height: 400px;
}

.empty-state {
  padding: 60px 0;
  text-align: center;
}

.chat-container {
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  min-height: 500px;
  border-radius: 22px;
  overflow: hidden;
}

.chat-messages {
  padding: 20px;
  max-height: 70vh;
  overflow-y: auto;
}

.message-wrapper {
  display: flex;
  flex-direction: column;
  margin-bottom: 16px;
}

.message-time-divider {
  text-align: center;
  margin: 16px 0;
  font-size: 12px;
  color: var(--apple-text-tertiary);
}

.message-bubble-wrapper {
  display: flex;
  align-items: flex-start;
  max-width: 75%;
}

.message-right {
  margin-left: auto;
  justify-content: flex-end;
  width: 100%;
  display: flex;
}

.message-left {
  margin-right: auto;
  justify-content: flex-start;
  width: 100%;
  display: flex;
}

/* 消息气泡 */
.message-bubble {
  position: relative;
  padding: 10px 14px;
  border-radius: 18px;
  word-wrap: break-word;
  word-break: break-word;
  box-shadow: 0 8px 16px rgba(15, 23, 42, 0.05);
  max-width: 100%;
}

.message-bubble-left {
  background: rgba(255, 255, 255, 0.94);
  border-top-left-radius: 8px;
}

.message-bubble-right {
  background: rgba(0, 122, 255, 0.12);
  border: 1px solid rgba(0, 122, 255, 0.16);
  border-top-right-radius: 8px;
  margin-left: auto;
}

.message-content-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.message-text {
  color: var(--apple-text);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14px;
}

.message-bubble-right .message-text {
  color: var(--apple-text);
}

/* 音频气泡 */
.audio-bubble {
  margin: 4px 0;
  display: flex;
  align-items: center;
}

.audio-play-btn-simple {
  flex-shrink: 0;
}

/* 消息元信息 */
.message-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
  opacity: 0.7;
}

.message-meta:hover {
  opacity: 1;
}

.message-time-small {
  font-size: 11px;
  color: var(--apple-text-tertiary);
}

.message-bubble-right .message-time-small {
  color: var(--apple-primary-pressed);
}

.message-more {
  font-size: 14px;
  color: var(--apple-text-tertiary);
  cursor: pointer;
  padding: 2px;
  border-radius: 8px;
  transition: all 0.2s;
}

.message-more:hover {
  background: rgba(0, 122, 255, 0.08);
  color: var(--apple-primary);
}

.message-bubble-right .message-more {
  color: var(--apple-primary-pressed);
}

.message-bubble-right .message-more:hover {
  background: rgba(0, 122, 255, 0.12);
}

/* 分页 */
.pagination {
  margin-top: 20px;
  padding: 20px;
  display: flex;
  justify-content: center;
  background: rgba(255, 255, 255, 0.88);
  border-top: 1px solid rgba(229, 229, 234, 0.72);
}

/* 滚动条样式 */
.chat-messages::-webkit-scrollbar {
  width: 6px;
}

.chat-messages::-webkit-scrollbar-track {
  background: rgba(229, 229, 234, 0.52);
  border-radius: 3px;
}

.chat-messages::-webkit-scrollbar-thumb {
  background: rgba(142, 142, 147, 0.58);
  border-radius: 3px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: rgba(110, 110, 115, 0.68);
}

/* Element Plus 组件样式覆盖 */
:deep(.el-slider__runway) {
  margin: 0;
  height: 4px;
}

:deep(.el-slider__bar) {
  height: 4px;
}

:deep(.el-slider__button) {
  width: 12px;
  height: 12px;
  border: 2px solid var(--apple-primary);
}

:deep(.el-slider__button-wrapper) {
  width: 24px;
  height: 24px;
  top: -10px;
}

:deep(.el-dropdown-menu__item) {
  padding: 8px 20px;
}
</style>


