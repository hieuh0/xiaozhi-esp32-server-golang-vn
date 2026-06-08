<template>
  <nav v-if="crumbs.length > 1" aria-label="breadcrumb">
    <ol class="flex items-center flex-wrap gap-1 mt-0.5">
      <li v-for="(crumb, i) in crumbs" :key="i" class="flex items-center gap-1">
        <RouterLink
          v-if="i < crumbs.length - 1"
          :to="crumb.path"
          class="text-xs text-[var(--color-primary)] opacity-75 hover:opacity-100 transition-opacity"
        >
          {{ crumb.label }}
        </RouterLink>
        <span v-else aria-current="page" class="text-xs text-[var(--color-text-secondary)]">
          {{ crumb.label }}
        </span>
        <span v-if="i < crumbs.length - 1" aria-hidden="true" class="text-xs text-[var(--color-text-tertiary)]">
          /
        </span>
      </li>
    </ol>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useLocale } from '../composables/useLocale'

const route = useRoute()
const { t } = useLocale()

const crumbs = computed(() => {
  const meta = route.meta?.breadcrumb
  if (!Array.isArray(meta) || meta.length === 0) return []
  return meta.map(crumb => ({ label: t(crumb.labelKey), path: crumb.path || '' }))
})
</script>
