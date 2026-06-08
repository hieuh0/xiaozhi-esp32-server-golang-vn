<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  multiple: { type: Boolean, default: false },
  accept: { type: String, default: '*' },
  maxSize: { type: Number, default: 10 * 1024 * 1024 } // 10MB
})
const emit = defineEmits(['update:modelValue'])

const dragging = ref(false)
const error = ref('')
const inputRef = ref(null)

const formatSize = (bytes) => {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1048576).toFixed(1) + ' MB'
}

function processFiles(fileList) {
  error.value = ''
  const files = Array.from(fileList)
  const invalid = files.find(f => f.size > props.maxSize)
  if (invalid) {
    error.value = `"${invalid.name}" exceeds max size of ${formatSize(props.maxSize)}`
    return
  }
  const next = props.multiple ? [...props.modelValue, ...files] : files.slice(0, 1)
  emit('update:modelValue', next)
}

function onDrop(e) {
  dragging.value = false
  processFiles(e.dataTransfer.files)
}

function onInputChange(e) {
  processFiles(e.target.files)
  e.target.value = '' // reset so same file can be re-selected
}

function remove(index) {
  const next = [...props.modelValue]
  next.splice(index, 1)
  emit('update:modelValue', next)
}

const acceptLabel = computed(() => {
  if (!props.accept || props.accept === '*') return 'Any file type'
  return props.accept.replace(/,/g, ', ')
})
</script>

<template>
  <div class="flex flex-col gap-3">
    <!-- Drop zone -->
    <div
      class="flex flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed p-8 transition-colors cursor-pointer"
      :class="dragging
        ? 'border-[var(--color-primary)] bg-[var(--color-primary-soft)]'
        : 'border-[var(--color-line)] hover:border-[var(--color-primary)]'"
      @dragover.prevent="dragging = true"
      @dragleave="dragging = false"
      @drop.prevent="onDrop"
      @click="inputRef?.click()"
    >
      <span class="text-3xl">📁</span>
      <p class="text-sm font-medium text-[var(--color-text)]">
        Drag & drop files here, or <span class="text-[var(--color-primary)] underline">browse</span>
      </p>
      <p class="text-xs text-[var(--color-text-tertiary)]">
        {{ acceptLabel }} · Max {{ formatSize(maxSize) }}
      </p>
      <input
        ref="inputRef"
        type="file"
        class="hidden"
        :multiple="multiple"
        :accept="accept"
        @change="onInputChange"
      />
    </div>

    <!-- Error -->
    <p v-if="error" class="text-sm text-[var(--color-danger)]">{{ error }}</p>

    <!-- File list -->
    <ul v-if="modelValue.length" class="flex flex-col gap-1">
      <li
        v-for="(file, i) in modelValue"
        :key="i"
        class="flex items-center justify-between rounded-lg border border-[var(--color-line)] bg-[var(--color-surface)] px-3 py-2 text-sm"
      >
        <span class="truncate text-[var(--color-text)]">{{ file.name }}</span>
        <span class="ml-4 shrink-0 text-[var(--color-text-tertiary)]">{{ formatSize(file.size) }}</span>
        <button
          type="button"
          class="ml-3 shrink-0 text-[var(--color-danger)] hover:opacity-80"
          @click="remove(i)"
        >
          ✕
        </button>
      </li>
    </ul>
  </div>
</template>
