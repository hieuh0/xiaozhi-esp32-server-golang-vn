<script setup>
import { computed } from 'vue'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

const props = defineProps({
  modelValue: { type: Number, default: 0 },
  min: { type: Number, default: -Infinity },
  max: { type: Number, default: Infinity },
  step: { type: Number, default: 1 },
  disabled: Boolean
})
const emit = defineEmits(['update:modelValue'])

const clamp = (v) => Math.min(props.max, Math.max(props.min, v))

const onInput = (e) => {
  const v = parseFloat(e.target.value)
  if (!isNaN(v)) emit('update:modelValue', clamp(v))
}
const decrement = () => emit('update:modelValue', clamp(props.modelValue - props.step))
const increment = () => emit('update:modelValue', clamp(props.modelValue + props.step))

const atMin = computed(() => props.modelValue <= props.min)
const atMax = computed(() => props.modelValue >= props.max)
</script>

<template>
  <div class="flex items-center gap-0">
    <Button
      type="button"
      variant="outline"
      size="icon"
      class="h-9 w-9 rounded-r-none border-r-0"
      :disabled="disabled || atMin"
      @click="decrement"
    >
      <span class="text-base leading-none">−</span>
    </Button>
    <Input
      type="number"
      :value="modelValue"
      :min="min"
      :max="max"
      :step="step"
      :disabled="disabled"
      class="w-20 rounded-none text-center [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
      @input="onInput"
    />
    <Button
      type="button"
      variant="outline"
      size="icon"
      class="h-9 w-9 rounded-l-none border-l-0"
      :disabled="disabled || atMax"
      @click="increment"
    >
      <span class="text-base leading-none">+</span>
    </Button>
  </div>
</template>
