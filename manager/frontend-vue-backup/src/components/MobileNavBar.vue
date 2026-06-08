<script setup>
import { ChevronLeft } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { useLocale } from '../composables/useLocale'

const { t } = useLocale()
const router = useRouter()

const props = defineProps({
  title: { type: String, default: '' },
  showBack: { type: Boolean, default: true },
  leftText: { type: String, default: '' },
  rightText: { type: String, default: '' }
})

const emit = defineEmits(['click-left', 'click-right'])

const handleLeftClick = () => {
  if (props.showBack) router.back()
  emit('click-left')
}
const handleRightClick = () => emit('click-right')
</script>

<template>
  <header class="flex items-center h-12 px-3 mx-3 mt-3 rounded-2xl border border-[var(--color-line)] bg-[var(--color-surface)] sticky top-3 z-10 shadow-sm">
    <button
      v-if="showBack"
      type="button"
      @click="handleLeftClick"
      class="flex items-center justify-center min-w-[44px] h-full -ml-1 mr-1 rounded-xl text-[var(--color-text)]"
    >
      <ChevronLeft class="w-5 h-5" />
    </button>
    <h1 class="flex-1 text-base font-bold text-[var(--color-text)] truncate">
      {{ title || t('xiaozhi_management_system') }}
    </h1>
    <slot name="right" />
  </header>
</template>
