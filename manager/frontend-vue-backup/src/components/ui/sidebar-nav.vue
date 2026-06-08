<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'

const props = defineProps({
  /**
   * Array of nav items: { label: string, icon?: Component, path?: string, children?: NavItem[] }
   */
  items: { type: Array, default: () => [] },
  /** Controlled open state for mobile sheet (optional) */
  open: { type: Boolean, default: false }
})
const emit = defineEmits(['update:open'])

const route = useRoute()
const expanded = ref({}) // { [label]: boolean } tracks open groups

function toggleGroup(label) {
  expanded.value[label] = !expanded.value[label]
}

function isActive(path) {
  if (!path) return false
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(item) {
  return item.children?.some(c => isActive(c.path)) ?? false
}
</script>

<template>
  <!-- Desktop sidebar (rendered inline, hidden on mobile) -->
  <nav class="hidden lg:flex flex-col gap-1 w-56 shrink-0">
    <template v-for="item in items" :key="item.label">
      <!-- Leaf item -->
      <RouterLink
        v-if="!item.children"
        :to="item.path"
        class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors"
        :class="isActive(item.path)
          ? 'bg-[var(--color-primary)] text-white'
          : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text)]'"
      >
        <component :is="item.icon" v-if="item.icon" class="size-4 shrink-0" />
        {{ item.label }}
      </RouterLink>

      <!-- Group item -->
      <div v-else>
        <button
          type="button"
          class="flex w-full items-center justify-between gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors"
          :class="isGroupActive(item)
            ? 'text-[var(--color-text)]'
            : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text)]'"
          @click="toggleGroup(item.label)"
        >
          <span class="flex items-center gap-2.5">
            <component :is="item.icon" v-if="item.icon" class="size-4 shrink-0" />
            {{ item.label }}
          </span>
          <span class="text-xs transition-transform" :class="expanded[item.label] ? 'rotate-180' : ''">▾</span>
        </button>
        <div v-if="expanded[item.label]" class="ml-4 flex flex-col gap-0.5 border-l border-[var(--color-line)] pl-3">
          <RouterLink
            v-for="child in item.children"
            :key="child.path"
            :to="child.path"
            class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
            :class="isActive(child.path)
              ? 'text-[var(--color-primary)] font-medium'
              : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'"
          >
            <component :is="child.icon" v-if="child.icon" class="size-3.5 shrink-0" />
            {{ child.label }}
          </RouterLink>
        </div>
      </div>
    </template>
  </nav>

  <!-- Mobile sheet -->
  <Sheet :open="open" @update:open="emit('update:open', $event)">
    <SheetContent side="left" class="w-64 p-4">
      <nav class="flex flex-col gap-1 pt-4">
        <template v-for="item in items" :key="item.label">
          <RouterLink
            v-if="!item.children"
            :to="item.path"
            class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors"
            :class="isActive(item.path)
              ? 'bg-[var(--color-primary)] text-white'
              : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)]'"
            @click="emit('update:open', false)"
          >
            <component :is="item.icon" v-if="item.icon" class="size-4 shrink-0" />
            {{ item.label }}
          </RouterLink>
          <div v-else>
            <button
              type="button"
              class="flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm font-medium text-[var(--color-text-secondary)]"
              @click="toggleGroup(item.label)"
            >
              <span class="flex items-center gap-2.5">
                <component :is="item.icon" v-if="item.icon" class="size-4 shrink-0" />
                {{ item.label }}
              </span>
              <span class="text-xs transition-transform" :class="expanded[item.label] ? 'rotate-180' : ''">▾</span>
            </button>
            <div v-if="expanded[item.label]" class="ml-4 flex flex-col gap-0.5 border-l border-[var(--color-line)] pl-3">
              <RouterLink
                v-for="child in item.children"
                :key="child.path"
                :to="child.path"
                class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
                @click="emit('update:open', false)"
              >
                {{ child.label }}
              </RouterLink>
            </div>
          </div>
        </template>
      </nav>
    </SheetContent>
  </Sheet>
</template>
