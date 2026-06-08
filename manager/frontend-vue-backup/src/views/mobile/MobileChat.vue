<script setup>
import { MessageCircle } from '@lucide/vue'
import { useLocale } from '../../composables/useLocale'

const { t } = useLocale()
</script>

<template>
  <div class="flex flex-col items-center justify-center h-full py-16 gap-4 text-[var(--color-text-secondary)]">
    <MessageCircle class="w-14 h-14 opacity-25" />
    <p class="text-sm">{{ t('no_chat_history_select') }}</p>
  </div>
</template>
