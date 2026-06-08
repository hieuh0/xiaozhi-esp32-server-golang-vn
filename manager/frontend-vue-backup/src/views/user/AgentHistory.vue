<script setup>
import { ref, reactive, onMounted, onBeforeUnmount, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Download, Play, Pause, MoreHorizontal, ChevronLeft, ChevronRight } from '@lucide/vue'
import api from '../../utils/api'
import { useLocale } from '../../composables/useLocale'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select'
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from '@/components/ui/dropdown-menu'

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

const filters = reactive({
  role: '',
  device_id: '',
  start_date: '',
  end_date: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 50
})

const totalPages = computed(() => Math.ceil(total.value / pagination.pageSize))

const audioRefs = ref({})
const playingAudioId = ref(null)
const chatMessagesRef = ref(null)
const audioBlobUrls = ref({})

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

const loadDevices = async () => {
  try {
    const response = await api.get(`/user/agents/${agentId.value}/devices`)
    devices.value = response.data.data || []
  } catch (error) {
    console.error(t('load_device_list_failed_v2'), error)
  }
}

const loadMessages = async () => {
  if (!agentId.value) return
  loading.value = true
  try {
    const params = { page: pagination.page, page_size: pagination.pageSize }
    if (filters.role) params.role = filters.role
    if (filters.device_id) params.device_id = filters.device_id
    if (filters.start_date) params.start_date = filters.start_date
    if (filters.end_date) params.end_date = filters.end_date

    const response = await api.get(`/user/history/agents/${agentId.value}/messages`, { params })
    const data = response.data.data || []
    messages.value = [...data].reverse()
    total.value = response.data.total || 0
    await preloadAudioMessages()
    await nextTick()
    scrollToBottom()
  } catch (error) {
    ElMessage.error(t('load_messages_failed_prefix') + (error.response?.data?.error || error.message))
    console.error(t('load_message_list_failed'), error)
    messages.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { pagination.page = 1; loadMessages() }

const handleReset = () => {
  filters.role = ''
  filters.device_id = ''
  filters.start_date = ''
  filters.end_date = ''
  pagination.page = 1
  loadMessages()
}

const handlePageChange = (page) => { pagination.page = page; loadMessages() }

const handleSizeChange = (size) => {
  pagination.pageSize = Number(size)
  pagination.page = 1
  loadMessages()
}

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

const handleExport = async () => {
  exporting.value = true
  try {
    const params = { agent_id: agentId.value }
    if (filters.role) params.role = filters.role
    if (filters.device_id) params.device_id = filters.device_id
    if (filters.start_date) params.start_date = filters.start_date
    if (filters.end_date) params.end_date = filters.end_date

    const response = await api.get('/user/history/export', { params, responseType: 'blob' })
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

const formatTimeShort = (dateString) => {
  const date = new Date(dateString)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const msgDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  if (msgDate.getTime() === today.getTime()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  if (msgDate.getTime() === yesterday.getTime()) {
    return t('yesterday_prefix', { time: date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) })
  }
  if (date.getFullYear() === now.getFullYear()) {
    return date.toLocaleString(undefined, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleString(undefined, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const shouldShowTime = (message, index) => {
  if (index === 0) return true
  const currentTime = new Date(message.created_at).getTime()
  const prevTime = new Date(messages.value[index - 1].created_at).getTime()
  return (currentTime - prevTime) > 5 * 60 * 1000
}

const scrollToBottom = () => {
  if (chatMessagesRef.value) {
    nextTick(() => { chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight })
  }
}

const getAudioUrl = async (messageId) => {
  if (audioBlobUrls.value[messageId]) return audioBlobUrls.value[messageId]
  try {
    const response = await api.get(`/user/history/messages/${messageId}/audio`, { responseType: 'blob' })
    const blobUrl = URL.createObjectURL(response.data)
    audioBlobUrls.value[messageId] = blobUrl
    return blobUrl
  } catch (error) {
    console.warn(t('load_audio_failed'), messageId, error)
    return null
  }
}

const preloadAudioMessages = async () => {
  const audioMessages = messages.value.filter(msg => msg.audio_path)
  const promises = audioMessages.slice(0, 10).map(msg => getAudioUrl(msg.id).catch(err => {
    console.warn(t('preload_audio_failed'), msg.id, err)
    return null
  }))
  await Promise.all(promises)
}

const handleAudioEnded = (messageId) => { playingAudioId.value = null }

const handleAudioError = async (messageId) => {
  console.warn(t('audio_load_failed'), messageId)
  try {
    const url = await getAudioUrl(messageId)
    if (url) {
      const audio = audioRefs.value[messageId]
      if (audio) audio.load()
    }
  } catch (error) {
    console.warn(t('audio_reload_failed'), messageId, error)
  }
}

const toggleAudio = async (messageId) => {
  const audio = audioRefs.value[messageId]
  if (!audio) return
  if (!audioBlobUrls.value[messageId]) {
    const url = await getAudioUrl(messageId)
    if (!url) { console.warn(t('audio_load_play_failed'), messageId); return }
    await new Promise((resolve) => { audio.onloadeddata = resolve; audio.load() })
  }
  if (playingAudioId.value && playingAudioId.value !== messageId) {
    const otherAudio = audioRefs.value[playingAudioId.value]
    if (otherAudio) { otherAudio.pause(); otherAudio.currentTime = 0 }
  }
  if (playingAudioId.value === messageId) {
    audio.pause()
    playingAudioId.value = null
  } else {
    try {
      await audio.play()
      playingAudioId.value = messageId
    } catch (error) {
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
    await Promise.all([loadAgent(), loadDevices(), loadMessages()])
  } catch (error) {
    console.error(t('init_failed'), error)
  }
})

onBeforeUnmount(() => {
  Object.values(audioBlobUrls.value).forEach(url => { if (url) URL.revokeObjectURL(url) })
  audioBlobUrls.value = {}
})
</script>

<template>
  <div class="min-h-full py-2 pb-6">
    <!-- Header -->
    <div class="max-w-[1120px] mx-auto mb-3 flex items-center justify-between gap-4">
      <div class="flex items-center gap-3 min-w-0">
        <Button variant="ghost" size="sm" @click="$router.back()">
          <ArrowLeft class="w-4 h-4 mr-1" />{{ t('back') }}
        </Button>
        <div class="grid gap-0.5 min-w-0">
          <span class="text-xs font-semibold text-[var(--color-text-secondary)]">{{ t('current_agent') }}</span>
          <strong class="text-base text-[var(--color-text)] truncate">{{ agentName || t('unnamed_agent') }}</strong>
          <p v-if="total > 0" class="m-0 text-sm text-[var(--color-text-secondary)]">{{ t('messages_total_count', { count: total }) }}</p>
        </div>
      </div>
      <Button :disabled="exporting" @click="handleExport">
        <Download class="w-4 h-4 mr-1.5" />{{ t('export_records') }}
      </Button>
    </div>

    <!-- Filter panel -->
    <div class="max-w-[1120px] mx-auto mb-3 p-4 border border-[var(--color-line)] rounded-xl bg-[var(--color-surface)]">
      <div class="flex flex-wrap items-end gap-3">
        <div class="grid gap-1">
          <label class="text-xs font-medium text-[var(--color-text-secondary)]">{{ t('role') }}</label>
          <Select v-model="filters.role">
            <SelectTrigger class="w-[120px]"><SelectValue :placeholder="t('all')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="">{{ t('all') }}</SelectItem>
              <SelectItem value="user">{{ t('user') }}</SelectItem>
              <SelectItem value="assistant">{{ t('robot') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="grid gap-1">
          <label class="text-xs font-medium text-[var(--color-text-secondary)]">{{ t('device') }}</label>
          <Select v-model="filters.device_id">
            <SelectTrigger class="w-[160px]"><SelectValue :placeholder="t('all')" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="">{{ t('all') }}</SelectItem>
              <SelectItem
                v-for="device in devices"
                :key="device.id"
                :value="device.device_name || device.device_code"
              >{{ device.device_name || device.device_code }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="grid gap-1">
          <label class="text-xs font-medium text-[var(--color-text-secondary)]">{{ t('start_date') }}</label>
          <Input type="date" v-model="filters.start_date" class="w-[150px]" />
        </div>
        <div class="grid gap-1">
          <label class="text-xs font-medium text-[var(--color-text-secondary)]">{{ t('end_date') }}</label>
          <Input type="date" v-model="filters.end_date" class="w-[150px]" />
        </div>
        <div class="flex gap-2 items-end">
          <Button size="sm" @click="handleSearch">{{ t('query') }}</Button>
          <Button size="sm" variant="outline" @click="handleReset">{{ t('reset') }}</Button>
        </div>
      </div>
    </div>

    <!-- Messages card -->
    <div :class="['max-w-[1120px] mx-auto border border-[var(--color-line)] rounded-xl bg-[var(--color-surface)] overflow-hidden', loading && 'opacity-60 pointer-events-none']">
      <!-- Empty state -->
      <div v-if="!loading && messages.length === 0" class="py-16 text-center text-sm text-[var(--color-text-secondary)]">
        {{ t('no_chat_history') }}
      </div>

      <!-- Chat area -->
      <div v-else class="bg-[var(--color-surface-muted)] min-h-[400px]">
        <div
          ref="chatMessagesRef"
          class="p-5 max-h-[70vh] overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-[var(--color-line)]"
        >
          <div v-for="(message, index) in messages" :key="message.id" class="mb-4 flex flex-col">
            <!-- Time divider -->
            <div v-if="shouldShowTime(message, index)" class="text-center my-3 text-xs text-[var(--color-text-tertiary)]">
              {{ formatTimeShort(message.created_at) }}
            </div>

            <!-- Assistant (left) -->
            <div v-if="message.role === 'assistant'" class="flex justify-start">
              <div class="max-w-[75%] px-3.5 py-2.5 rounded-2xl rounded-tl-lg bg-[var(--color-surface)] shadow-sm">
                <div class="flex flex-col gap-2">
                  <div v-if="message.content" class="text-sm text-[var(--color-text)] whitespace-pre-wrap break-words leading-relaxed">{{ message.content }}</div>
                  <div v-if="message.audio_path" class="flex items-center">
                    <audio
                      :ref="el => audioRefs[message.id] = el"
                      :src="audioBlobUrls[message.id]"
                      @ended="handleAudioEnded(message.id)"
                      @error="handleAudioError(message.id)"
                    />
                    <Button variant="ghost" size="icon" class="h-7 w-7" :aria-label="playingAudioId === message.id ? t('pause') : t('play')" @click="toggleAudio(message.id)">
                      <Pause v-if="playingAudioId === message.id" class="w-3.5 h-3.5" />
                      <Play v-else class="w-3.5 h-3.5" />
                    </Button>
                  </div>
                  <div class="flex items-center gap-1.5 opacity-60 hover:opacity-100 transition-opacity">
                    <span class="text-[11px] text-[var(--color-text-tertiary)]">{{ formatTimeShort(message.created_at) }}</span>
                    <DropdownMenu>
                      <DropdownMenuTrigger as-child>
                        <button type="button" :aria-label="t('more_actions')" class="p-0.5 rounded hover:bg-[var(--color-surface-muted)] text-[var(--color-text-tertiary)] hover:text-[var(--color-text)] transition-colors">
                          <MoreHorizontal class="w-3.5 h-3.5" />
                        </button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent>
                        <DropdownMenuItem @click="handleDelete(message.id)">{{ t('delete') }}</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              </div>
            </div>

            <!-- User (right) -->
            <div v-else class="flex justify-end">
              <div class="max-w-[75%] px-3.5 py-2.5 rounded-2xl rounded-tr-lg bg-blue-50 border border-blue-100 dark:bg-blue-900/20 dark:border-blue-800 shadow-sm">
                <div class="flex flex-col gap-2">
                  <div v-if="message.content" class="text-sm text-[var(--color-text)] whitespace-pre-wrap break-words leading-relaxed">{{ message.content }}</div>
                  <div v-if="message.audio_path" class="flex items-center">
                    <audio
                      :ref="el => audioRefs[message.id] = el"
                      :src="audioBlobUrls[message.id]"
                      @ended="handleAudioEnded(message.id)"
                      @error="handleAudioError(message.id)"
                    />
                    <Button variant="ghost" size="icon" class="h-7 w-7" :aria-label="playingAudioId === message.id ? t('pause') : t('play')" @click="toggleAudio(message.id)">
                      <Pause v-if="playingAudioId === message.id" class="w-3.5 h-3.5" />
                      <Play v-else class="w-3.5 h-3.5" />
                    </Button>
                  </div>
                  <div class="flex items-center justify-end gap-1.5 opacity-60 hover:opacity-100 transition-opacity">
                    <DropdownMenu>
                      <DropdownMenuTrigger as-child>
                        <button type="button" class="p-0.5 rounded hover:bg-blue-100 dark:hover:bg-blue-800/40 text-[var(--color-text-tertiary)] hover:text-[var(--color-text)] transition-colors">
                          <MoreHorizontal class="w-3.5 h-3.5" />
                        </button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent>
                        <DropdownMenuItem @click="handleDelete(message.id)">{{ t('delete') }}</DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <span class="text-[11px] text-[var(--color-text-tertiary)]">{{ formatTimeShort(message.created_at) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Pagination -->
        <div v-if="total > 0" class="flex items-center justify-center gap-3 py-3 px-5 border-t border-[var(--color-line)] bg-[var(--color-surface)]">
          <Button variant="outline" size="icon" class="h-8 w-8" aria-label="Previous page" :disabled="pagination.page <= 1" @click="handlePageChange(pagination.page - 1)">
            <ChevronLeft class="w-4 h-4" />
          </Button>
          <span class="text-sm text-[var(--color-text-secondary)] min-w-[80px] text-center">{{ pagination.page }} / {{ totalPages }}</span>
          <Button variant="outline" size="icon" class="h-8 w-8" aria-label="Next page" :disabled="pagination.page >= totalPages" @click="handlePageChange(pagination.page + 1)">
            <ChevronRight class="w-4 h-4" />
          </Button>
          <Select :model-value="String(pagination.pageSize)" @update:model-value="handleSizeChange">
            <SelectTrigger class="w-[80px] h-8"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="20">20</SelectItem>
              <SelectItem value="50">50</SelectItem>
              <SelectItem value="100">100</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>
  </div>
</template>
