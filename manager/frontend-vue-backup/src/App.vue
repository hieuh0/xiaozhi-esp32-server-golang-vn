<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script setup>
import { onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import api from '@/utils/api'
import { useLocale } from '@/composables/useLocale'

const router = useRouter()
const route = useRoute()
const { t, lang } = useLocale()

watch([() => route.meta?.title, lang], ([titleKey]) => {
  const appTitle = t('xiaozhi_system_title')
  document.title = titleKey ? `${t(titleKey)} - ${appTitle}` : appTitle
}, { immediate: true })

const checkSystemStatus = async () => {
  try {
    const response = await api.get('/setup/status')
    if (response.data.needs_setup && router.currentRoute.value.path !== '/setup') {
      router.push('/setup')
    }
  } catch (error) {
    console.error(t('check_system_failed'), error)
  }
}

onMounted(checkSystemStatus)
</script>

