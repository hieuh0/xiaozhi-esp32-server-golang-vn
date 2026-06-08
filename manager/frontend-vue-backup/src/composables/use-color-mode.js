import { useColorMode } from '@vueuse/core'
import { useThemeStore } from '../stores/theme'

/**
 * Wraps VueUse useColorMode with the project's 3-state cycle (dark/light/auto).
 * Keeps VueUse and pinia theme store in sync.
 */
export function useAppColorMode() {
  const themeStore = useThemeStore()
  const colorMode = useColorMode({
    attribute: 'class',
    modes: { dark: 'dark', light: 'light' }
  })

  function setMode(mode) {
    themeStore.setMode(mode)
    colorMode.value = mode === 'auto' ? 'auto' : mode
  }

  function nextMode() {
    themeStore.nextMode()
  }

  return { mode: () => themeStore.mode, setMode, nextMode }
}
