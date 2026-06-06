<template>
  <el-form :model="model" label-width="140px" class="wizard-form">
    <el-form-item :label="t('domain_or_ip')" prop="host">
      <el-input v-model="model.host" :placeholder="t('ip_example')" clearable />
    </el-form-item>
    <el-form-item :label="t('port')" prop="port">
      <el-input-number v-model="model.port" :min="1" :max="65535" style="width: 100%" />
    </el-form-item>
    <el-form-item :label="t('protocol')" prop="protocol">
      <el-radio-group v-model="model.protocol">
        <el-radio value="http">HTTP</el-radio>
        <el-radio value="https">HTTPS</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item :label="t('signature_key')" prop="signature_key">
      <el-input v-model="model.signature_key" :placeholder="t('shared_with_mqtt_auth')" clearable />
    </el-form-item>
    <el-form-item :label="t('enable_mqtt_udp')" prop="enableMqttUdp">
      <el-switch v-model="model.enableMqttUdp" />
      <span class="form-hint">{{ t('enable_mqtt_hint') }}</span>
    </el-form-item>
    <template v-if="model.enableMqttUdp">
      <el-form-item :label="t('mqtt_server_port')" prop="mqttServerPort" required>
        <el-input-number v-model="model.mqttServerPort" :min="1" :max="65535" style="width: 100%" :placeholder="t('mqtt_port_hint')" />
        <span class="form-hint">{{ t('mqtt_ip_tls_hint') }}</span>
      </el-form-item>
      <el-form-item :label="t('udp_port')" prop="udpPort" required>
        <el-input-number v-model="model.udpPort" :min="1" :max="65535" style="width: 100%" :placeholder="t('port_example')" />
        <span class="form-hint">{{ t('external_ip_hint') }}</span>
      </el-form-item>
    </template>
  </el-form>
</template>

<script setup>
import { useLocale } from '../../../composables/useLocale'

const { t } = useLocale()

defineProps({
  model: { type: Object, required: true }
})
</script>

<style scoped>
.form-hint { display: block; color: #909399; font-size: 12px; margin-top: 4px; line-height: 1.4; }
.wizard-form { margin-bottom: 24px; }
</style>
