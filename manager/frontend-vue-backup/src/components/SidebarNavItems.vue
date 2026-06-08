<template>
  <nav :class="$attrs.class">
    <template v-for="item in items" :key="item.label">
      <!-- Leaf item -->
      <RouterLink
        v-if="!item.children"
        :to="item.path"
        class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors"
        :class="isActive(item.path)
          ? 'bg-[var(--color-primary-soft)] text-[var(--color-primary)] font-semibold'
          : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-strong)] hover:text-[var(--color-text)]'"
        @click="emit('navigate')"
      >
        <component :is="item.icon" v-if="item.icon" class="size-4 shrink-0" />
        <span class="truncate">{{ item.label }}</span>
      </RouterLink>

      <!-- Group item -->
      <div v-else class="flex flex-col">
        <button
          type="button"
          class="flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors"
          :class="isGroupActive(item)
            ? 'text-[var(--color-text)]'
            : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-strong)] hover:text-[var(--color-text)]'"
          @click="toggleGroup(item.label)"
        >
          <span class="flex items-center gap-2.5">
            <component :is="item.icon" v-if="item.icon" class="size-4 shrink-0" />
            <span class="truncate">{{ item.label }}</span>
          </span>
          <ChevronDown
            class="size-3.5 shrink-0 transition-transform"
            :class="expanded[item.label] ? 'rotate-180' : ''"
          />
        </button>
        <div
          v-if="expanded[item.label]"
          class="ml-4 mt-0.5 flex flex-col gap-0.5 border-l border-[var(--color-line)] pl-3"
        >
          <RouterLink
            v-for="child in item.children"
            :key="child.path"
            :to="child.path"
            class="rounded-md px-2 py-1.5 text-xs transition-colors truncate"
            :class="isActive(child.path)
              ? 'text-[var(--color-primary)] font-semibold'
              : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'"
            @click="emit('navigate')"
          >
            {{ child.label }}
          </RouterLink>
        </div>
      </div>
    </template>
  </nav>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { ChevronDown } from '@lucide/vue'

defineOptions({ inheritAttrs: false })

defineProps({
  items: { type: Array, default: () => [] }
})
const emit = defineEmits(['navigate'])

const route = useRoute()
const expanded = ref({})

function toggleGroup(label) {
  expanded.value[label] = !expanded.value[label]
}

function isActive(path) {
  return path && (route.path === path || route.path.startsWith(path + '/'))
}

function isGroupActive(item) {
  return item.children?.some(c => isActive(c.path)) ?? false
}
</script>
