<template>
  <header class="flex items-center justify-between gap-4 px-5 h-16 shrink-0 border-b border-[var(--color-line)] bg-[var(--color-surface-strong)]">
    <!-- Left: hamburger (mobile) + title + breadcrumb -->
    <div class="flex items-center gap-3 min-w-0">
      <Button variant="ghost" size="icon" class="lg:hidden" aria-label="Toggle menu" @click="emit('toggle-sidebar')">
        <Menu class="size-5" />
      </Button>
      <div class="min-w-0">
        <p v-if="eyebrow" class="text-[10px] font-bold uppercase tracking-widest text-[var(--color-primary)]">
          {{ eyebrow }}
        </p>
        <h1 class="text-lg font-semibold tracking-tight text-[var(--color-text)] leading-tight truncate">
          {{ title }}
        </h1>
        <AppBreadcrumb v-if="isAdmin" />
      </div>
    </div>

    <!-- Right: actions -->
    <div class="flex items-center gap-2 shrink-0">
      <slot name="actions" />

      <!-- Admin shortcuts -->
      <template v-if="showAdminShortcuts">
        <RouterLink to="/admin/config-wizard" custom v-slot="{ navigate, isActive }">
          <Button
            variant="ghost"
            size="sm"
            :class="isActive ? 'bg-[var(--color-primary-soft)] text-[var(--color-primary)]' : ''"
            @click="navigate"
          >
            <Wand2 class="size-4 mr-1" />{{ t('config_wizard') }}
          </Button>
        </RouterLink>
        <RouterLink to="/admin/ota-config" custom v-slot="{ navigate, isActive }">
          <Button
            variant="ghost"
            size="sm"
            :class="isActive ? 'bg-[var(--color-primary-soft)] text-[var(--color-primary)]' : ''"
            @click="navigate"
          >
            <Upload class="size-4 mr-1" />{{ t('ota_config') }}
          </Button>
        </RouterLink>
      </template>

      <!-- Theme toggle -->
      <Button variant="ghost" size="icon" :aria-label="themeAriaLabel" @click="themeStore.nextMode()">
        <Sun v-if="themeStore.mode === 'light'" class="size-4" />
        <Moon v-else-if="themeStore.mode === 'dark'" class="size-4" />
        <Monitor v-else class="size-4" />
      </Button>

      <!-- Language switcher -->
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="ghost" size="sm" class="gap-1">
            {{ langLabel }}<ChevronDown class="size-3" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem @click="setLang('vi')">🇻🇳 Tiếng Việt</DropdownMenuItem>
          <DropdownMenuItem @click="setLang('en')">🇬🇧 English</DropdownMenuItem>
          <DropdownMenuItem @click="setLang('zh')">🇨🇳 Chinese</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <!-- Profile dropdown -->
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <button
            type="button"
            class="flex items-center gap-2 rounded-full border border-[var(--color-line)] bg-[var(--color-surface)] px-2.5 py-1.5 text-sm transition-colors hover:bg-[var(--color-surface-muted)]"
          >
            <span class="flex size-7 items-center justify-center rounded-full bg-[var(--color-primary-soft)] text-xs font-bold text-[var(--color-primary)]">
              {{ initial }}
            </span>
            <span class="hidden sm:flex flex-col items-start leading-tight">
              <strong class="text-xs font-semibold text-[var(--color-text)]">{{ username }}</strong>
              <small class="text-[10px] text-[var(--color-text-secondary)]">{{ roleLabel }}</small>
            </span>
            <ChevronDown class="size-3 text-[var(--color-text-tertiary)]" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem v-if="!isAdmin" @click="emit('command', 'api-tokens')">
            API Token
          </DropdownMenuItem>
          <DropdownMenuItem class="text-[var(--color-danger)]" @click="emit('command', 'logout')">
            {{ t('logout') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'
import { Menu, Sun, Moon, Monitor, ChevronDown, Wand2, Upload } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem
} from '@/components/ui/dropdown-menu'
import AppBreadcrumb from './AppBreadcrumb.vue'
import { useLocale } from '../composables/useLocale'
import { useThemeStore } from '../stores/theme'

const { t, lang, setLang } = useLocale()
const themeStore = useThemeStore()

const props = defineProps({
  title: { type: String, default: '' },
  eyebrow: { type: String, default: '' },
  username: { type: String, default: '' },
  roleLabel: { type: String, default: '' },
  initial: { type: String, default: 'U' },
  isAdmin: { type: Boolean, default: false },
  showAdminShortcuts: { type: Boolean, default: false }
})
const emit = defineEmits(['command', 'toggle-sidebar'])

const themeAriaLabel = computed(() => ({
  dark: t('switch_auto_mode'),
  auto: t('switch_light_mode'),
  light: t('switch_dark_mode')
})[themeStore.mode])

const langLabel = computed(() => ({ vi: '🇻🇳 VI', en: '🇬🇧 EN', zh: '🇨🇳 ZH' })[lang.value] ?? 'VI')
</script>
