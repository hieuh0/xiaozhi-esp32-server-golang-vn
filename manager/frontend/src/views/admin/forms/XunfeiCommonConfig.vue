<template>
  <el-form-item :label="t('app_id')" :prop="`${prefix}.app_id`">
    <el-input v-model="modelValue[`${prefix}`].app_id" :placeholder="t('enter_xunfei_app_id')" />
  </el-form-item>
  <el-form-item label="API Key" :prop="`${prefix}.api_key`">
    <el-input v-model="modelValue[`${prefix}`].api_key" :placeholder="t('enter_api_key')" type="password" show-password />
  </el-form-item>
  <el-form-item label="API Secret" :prop="`${prefix}.api_secret`">
    <el-input v-model="modelValue[`${prefix}`].api_secret" :placeholder="t('enter_api_secret')" type="password" show-password />
  </el-form-item>
  <el-form-item label="WebSocket URL" :prop="`${prefix}.ws_url`">
    <el-input v-model="modelValue[`${prefix}`].ws_url" :placeholder="defaultWsUrl" />
  </el-form-item>
  <el-form-item :label="t('frame_duration')" :prop="`${prefix}.frame_duration`">
    <el-input-number v-model="modelValue[`${prefix}`].frame_duration" :min="1" :max="1000" style="width: 100%" />
  </el-form-item>
  <el-form-item :label="t('connection_timeout')" :prop="`${prefix}.connect_timeout`">
    <el-input-number v-model="modelValue[`${prefix}`].connect_timeout" :min="1" :max="60" style="width: 100%" />
  </el-form-item>
  <el-form-item :label="t('read_timeout')" :prop="`${prefix}.read_timeout`">
    <el-input-number v-model="modelValue[`${prefix}`].read_timeout" :min="1" :max="300" style="width: 100%" />
  </el-form-item>
</template>

<script setup>
import { useLocale } from '../../../composables/useLocale'
const { t } = useLocale()
defineProps({
  modelValue: {
    type: Object,
    required: true
  },
  prefix: {
    type: String,
    required: true
  },
  defaultWsUrl: {
    type: String,
    default: ''
  }
})
</script>