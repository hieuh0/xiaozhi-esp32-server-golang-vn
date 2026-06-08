<template>
  <div class="space-y-5">
    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('domain_or_ip') }}</label>
      <Input v-model="model.host" :placeholder="t('ip_example')" />
    </div>

    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('port') }}</label>
      <Input
        type="number" :min="1" :max="65535"
        :value="model.port"
        @input="model.port = Number($event.target.value)"
      />
    </div>

    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('protocol') }}</label>
      <div class="flex gap-6">
        <label class="flex items-center gap-2 cursor-pointer">
          <input type="radio" v-model="model.protocol" value="http" class="accent-[var(--color-primary)]" />
          <span class="text-sm text-[var(--color-text)]">HTTP</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer">
          <input type="radio" v-model="model.protocol" value="https" class="accent-[var(--color-primary)]" />
          <span class="text-sm text-[var(--color-text)]">HTTPS</span>
        </label>
      </div>
    </div>

    <div class="grid gap-1.5">
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('signature_key') }}</label>
      <Input v-model="model.signature_key" :placeholder="t('shared_with_mqtt_auth')" />
    </div>

    <div class="flex items-center gap-3">
      <Switch :model-value="!!model.enableMqttUdp" @update:model-value="v => model.enableMqttUdp = v" />
      <label class="text-sm font-medium text-[var(--color-text)]">{{ t('enable_mqtt_udp') }}</label>
      <span class="text-xs text-[var(--color-text-secondary)]">{{ t('enable_mqtt_hint') }}</span>
    </div>

    <template v-if="model.enableMqttUdp">
      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('mqtt_server_port') }} <span class="text-red-500">*</span></label>
        <Input
          type="number" :min="1" :max="65535" :placeholder="t('mqtt_port_hint')"
          :value="model.mqttServerPort"
          @input="model.mqttServerPort = Number($event.target.value)"
        />
        <p class="text-xs text-[var(--color-text-secondary)]">{{ t('mqtt_ip_tls_hint') }}</p>
      </div>

      <div class="grid gap-1.5">
        <label class="text-sm font-medium text-[var(--color-text)]">{{ t('udp_port') }} <span class="text-red-500">*</span></label>
        <Input
          type="number" :min="1" :max="65535" :placeholder="t('port_example')"
          :value="model.udpPort"
          @input="model.udpPort = Number($event.target.value)"
        />
        <p class="text-xs text-[var(--color-text-secondary)]">{{ t('external_ip_hint') }}</p>
      </div>
    </template>
  </div>
</template>

<script setup>
import { useLocale } from '../../../composables/useLocale'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

const { t } = useLocale()

defineProps({
  model: { type: Object, required: true }
})
</script>
