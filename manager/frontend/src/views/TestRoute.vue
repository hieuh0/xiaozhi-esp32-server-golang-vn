<template>
  <div class="test-route">
    <h1>{{ t('test_route_page') }}</h1>
    <p>{{ t('routing_works_hint') }}</p>
    <div class="route-state">{{ message }}</div>
    <el-button @click="testFunction">{{ t('test_button') }}</el-button>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useLocale } from '../composables/useLocale'
const { t } = useLocale()

const message = ref('Hello World')

const testFunction = () => {
  message.value = t('button_clicked')
}
</script>

<style scoped>
.test-route {
  padding: 24px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.9);
  box-shadow: var(--apple-shadow-md);
}

.test-route h1 {
  margin: 0 0 12px;
  color: var(--apple-text);
}

.test-route p {
  margin: 0 0 16px;
  color: var(--apple-text-secondary);
}

.route-state {
  margin-bottom: 16px;
  padding: 12px 14px;
  border-radius: 16px;
  background: rgba(248, 250, 252, 0.92);
  border: 1px solid rgba(229, 229, 234, 0.72);
  color: var(--apple-text);
}
</style>
