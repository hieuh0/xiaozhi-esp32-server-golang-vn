<script setup>
import { cn } from "@/lib/utils";

const props = defineProps({
  class: { type: null, required: false },
});
</script>

<template>
  <li
    data-slot="breadcrumb-item"
    :class="cn('gap-1.5 inline-flex items-center', props.class)"
  >
    <slot />
  </li>
</template>
