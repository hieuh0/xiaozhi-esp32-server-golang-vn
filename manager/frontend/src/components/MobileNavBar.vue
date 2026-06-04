<template>
  <van-nav-bar
    :title="title"
    :left-arrow="showBack"
    :left-text="leftText"
    :right-text="rightText"
    @click-left="handleLeftClick"
    @click-right="handleRightClick"
    fixed
    placeholder
    safe-area-inset-top
    class="mobile-nav-bar"
  >
    <template #right v-if="$slots.right">
      <slot name="right"></slot>
    </template>
  </van-nav-bar>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useLocale } from '../composables/useLocale'
const { t } = useLocale()

const props = defineProps({
  title: {
    type: String,
    default: '小智管理系统' /* static: cannot use t() inside defineProps */
  },
  showBack: {
    type: Boolean,
    default: true
  },
  leftText: {
    type: String,
    default: ''
  },
  rightText: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['click-left', 'click-right'])

const router = useRouter()

const handleLeftClick = () => {
  if (props.showBack) {
    router.back()
  }
  emit('click-left')
}

const handleRightClick = () => {
  emit('click-right')
}
</script>

<style scoped>
.mobile-nav-bar {
  background: transparent;
  color: var(--apple-text);
}

:deep(.van-nav-bar) {
  margin: 12px 12px 0;
  border-radius: 22px;
  border: 1px solid rgba(255, 255, 255, 0.84);
  box-shadow: var(--apple-shadow-md);
  overflow: hidden;
}

:deep(.van-nav-bar__title) {
  color: var(--apple-text);
  font-weight: 700;
}

:deep(.van-nav-bar__arrow) {
  color: var(--apple-text);
}

:deep(.van-nav-bar__text) {
  color: var(--apple-text);
}
</style>
