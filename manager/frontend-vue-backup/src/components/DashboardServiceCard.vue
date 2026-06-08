<script setup>
import { ref, onMounted } from 'vue'
import { Copy } from '@lucide/vue'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'
import { useLocale } from '@/composables/useLocale'
import { useAuthStore } from '@/stores/auth'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

const { t } = useLocale()
const authStore = useAuthStore()

const addressLoading = ref(false)
const serviceAddress = ref({ otaUrl: '', wsUrl: '', mqttEndpoint: '', udpAddress: '' })
const otaTestLoading = ref(false)
const otaTestResult = ref(null)

async function loadServiceAddress() {
  addressLoading.value = true
  serviceAddress.value = { otaUrl: '', wsUrl: '', mqttEndpoint: '', udpAddress: '' }
  try {
    const [otaRes, udpRes] = await Promise.all([
      api.get('/admin/ota-configs'),
      api.get('/admin/udp-configs')
    ])
    const otaList = otaRes.data?.data || []
    const config = otaList.find(c => c.is_default) || otaList[0]
    if (config?.json_data) {
      const data = JSON.parse(config.json_data || '{}')
      let envData = data.external || {}
      if (!envData.websocket?.url && !envData.ota_url) envData = data.test || {}
      let otaUrl = envData.ota_url || ''
      if (!otaUrl && envData.websocket?.url) {
        const m = envData.websocket.url.match(/^(wss?):\/\/([^:/]+)(?::(\d+))?/)
        if (m) {
          const proto = m[1] === 'wss' ? 'https' : 'http'
          const port = m[3] || (m[1] === 'wss' ? '443' : '80')
          otaUrl = `${proto}://${m[2]}:${port}/xiaozhi/ota/`
        }
      }
      serviceAddress.value.otaUrl = otaUrl
      serviceAddress.value.wsUrl = envData.websocket?.url || ''
      if (envData.mqtt?.enable && envData.mqtt?.endpoint) {
        serviceAddress.value.mqttEndpoint = envData.mqtt.endpoint
      }
    }
    const udpList = udpRes.data?.data || []
    const udpConfig = udpList.find(c => c.is_default) || udpList[0]
    if (udpConfig?.json_data) {
      const d = JSON.parse(udpConfig.json_data || '{}')
      if (d.external_host && d.external_port != null) {
        serviceAddress.value.udpAddress = `${d.external_host}:${d.external_port}`
      }
    }
  } catch (err) {
    console.error(t('load_service_addr_failed'), err)
  } finally {
    addressLoading.value = false
  }
}

function copyAddress(text) {
  if (!text) return
  navigator.clipboard.writeText(text)
    .then(() => ElMessage.success(t('copied_to_clipboard')))
    .catch(() => ElMessage.error(t('copy_failed')))
}

function formatOtaResponseDisplay(str) {
  if (!str) return ''
  try { return JSON.stringify(JSON.parse(String(str).trim()), null, 2) } catch { return String(str).trim() }
}

async function runOtaTest() {
  otaTestLoading.value = true
  otaTestResult.value = null
  try {
    const res = await api.post('/admin/configs/test', { types: ['ota'] }, { timeout: 30000 })
    const data = res.data?.data ?? res.data
    const ota = data?.ota
    if (ota && typeof ota === 'object') {
      const entry = Object.entries(ota).find(([k]) => !k.startsWith('_'))
      if (entry) {
        const [, value] = entry
        let txt = ''
        if (value.websocket) {
          const ws = value.websocket
          txt += `WebSocket: ${ws.ok ? '✓' : '✗'} ${ws.message}${ws.first_packet_ms != null ? ` (${ws.first_packet_ms}ms)` : ''}\n`
        }
        if (value.mqtt_udp) {
          const m = value.mqtt_udp
          txt += `MQTT UDP: ${m.ok ? '✓' : '✗'} ${m.message}${m.first_packet_ms != null ? ` (${m.first_packet_ms}ms)` : ''}\n`
        }
        if (value.ota_response !== undefined && value.ota_response !== '') {
          txt += `\n--- ${t('ota_return_label')} ---\n${formatOtaResponseDisplay(value.ota_response)}`
        }
        otaTestResult.value = txt.trim() || t('detail_not_available')
        ElMessage[value.ok ? 'success' : 'warning'](value.message || (value.ok ? t('ota_test_passed') : t('ota_test_failed')))
      } else {
        otaTestResult.value = t('ota_test_no_result')
      }
    } else {
      otaTestResult.value = typeof data === 'string' ? data : JSON.stringify(data || {}, null, 2)
    }
  } catch (error) {
    const msg = (error.response?.data && typeof error.response.data === 'object')
      ? JSON.stringify(error.response.data, null, 2)
      : (error.response?.data?.message || error.message || t('request_failed'))
    otaTestResult.value = msg
    ElMessage.error(t('ota_test_request_failed'))
  } finally {
    otaTestLoading.value = false
  }
}

onMounted(loadServiceAddress)
</script>

<template>
  <Card>
    <CardHeader class="flex-row items-start justify-between pb-3">
      <div>
        <p class="text-[11px] font-bold tracking-widest text-[var(--color-text-tertiary)] uppercase mb-1">SERVICE ADDRESS</p>
        <h3 class="text-lg font-semibold text-[var(--color-text)]">{{ t('service_address') }}</h3>
      </div>
      <Button variant="outline" size="sm" :disabled="otaTestLoading" @click="runOtaTest">
        {{ t('ota_test') }}
      </Button>
    </CardHeader>
    <CardContent>
      <div v-if="addressLoading" class="py-6 text-center text-sm text-[var(--color-text-secondary)]">
        {{ t('loading') || 'Loading…' }}
      </div>
      <template v-else-if="serviceAddress.otaUrl || serviceAddress.wsUrl">
        <div class="flex flex-col gap-2.5">
          <div v-for="row in [
            { label: 'OTA', value: serviceAddress.otaUrl },
            { label: 'WS',  value: serviceAddress.wsUrl },
            { label: 'MQTT', value: serviceAddress.mqttEndpoint },
            { label: 'UDP', value: serviceAddress.udpAddress },
          ]" :key="row.label" v-show="row.value"
            class="grid grid-cols-[64px_1fr_auto] items-center gap-2.5 px-4 py-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)]"
          >
            <span class="text-[11px] font-bold tracking-widest text-[var(--color-text-tertiary)]">{{ row.label }}</span>
            <span class="text-sm font-medium text-[var(--color-text)] truncate" :title="row.value">{{ row.value }}</span>
            <Button variant="ghost" size="icon" class="w-7 h-7" :aria-label="t('copy')" @click="copyAddress(row.value)">
              <Copy class="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
        <div v-if="otaTestResult !== null" class="mt-4 flex flex-col gap-2">
          <Badge variant="secondary">{{ t('ota_return_chip') }}</Badge>
          <pre class="text-xs leading-relaxed p-3 rounded-lg bg-[var(--color-surface-muted)] border border-[var(--color-line)] max-h-44 overflow-auto whitespace-pre-wrap break-words">{{ otaTestResult }}</pre>
        </div>
      </template>
      <div v-else class="text-sm text-[var(--color-text-secondary)]">{{ t('no_ota_config') }}</div>
    </CardContent>
  </Card>
</template>
