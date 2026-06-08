<script setup>
import 'vanilla-colorful/hex-color-picker.js'

const props = defineProps({
  modelValue: { type: String, default: '#000000' }
})
const emit = defineEmits(['update:modelValue'])
</script>

<template>
  <hex-color-picker
    :color="modelValue"
    @color-changed="emit('update:modelValue', $event.detail.value)"
  />
</template>
