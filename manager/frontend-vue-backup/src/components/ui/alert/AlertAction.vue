<script setup>
import { cn } from "@/lib/utils";

const props = defineProps({
  class: { type: null, required: false },
});
</script>

<template>
  <div
    data-slot="alert-action"
    :class="cn('absolute top-2.5 right-3', props.class)"
  >
    <slot />
  </div>
</template>
