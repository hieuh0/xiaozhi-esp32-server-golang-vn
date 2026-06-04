---
phase: 1
title: "Infrastructure"
status: done
priority: P1
effort: "1h"
dependencies: []
---

# Phase 1: Infrastructure

## Overview

Tạo toàn bộ infrastructure i18n: Pinia store, `useLocale` composable, và 3 skeleton locale files (zh/vi/en). Sau phase này, bất kỳ Vue file nào đều có thể import `useLocale` và gọi `t('key')`.

## Architecture

```
src/
  locales/
    zh.js          # { key: '中文文本', ... }
    vi.js          # { key: 'Tiếng Việt', ... }
    en.js          # { key: 'English text', ... }
  stores/
    locale.js      # Pinia store: currentLang + setLang + localStorage
  composables/
    useLocale.js   # t(key), lang ref, setLang fn
```

**Pattern theo auth.js** (Composition API, localStorage trực tiếp trong store):

```js
// stores/locale.js
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useLocaleStore = defineStore('locale', () => {
  const lang = ref(localStorage.getItem('lang') || 'vi')
  const setLang = (l) => {
    lang.value = l
    localStorage.setItem('lang', l)
  }
  return { lang, setLang }
})
```

```js
// composables/useLocale.js
import { computed } from 'vue'
import { useLocaleStore } from '../stores/locale'
import zh from '../locales/zh.js'
import vi from '../locales/vi.js'
import en from '../locales/en.js'

const maps = { zh, vi, en }

export function useLocale() {
  const store = useLocaleStore()
  // Pinia setup stores unwrap refs automatically — store.lang is already a string
  // Fallback chain: currentLang → zh → key itself
  const t = (key) => maps[store.lang]?.[key] ?? maps.zh[key] ?? key
  return {
    t,
    lang: computed(() => store.lang),
    setLang: store.setLang,
  }
}
```

```js
// locales/zh.js — skeleton, Phase 2 sẽ điền đầy đủ
export default {}

// locales/vi.js — skeleton
export default {}

// locales/en.js — skeleton
export default {}
```

## Related Code Files

- Create: `manager/frontend/src/stores/locale.js`
- Create: `manager/frontend/src/composables/useLocale.js`
- Create: `manager/frontend/src/locales/zh.js`
- Create: `manager/frontend/src/locales/vi.js`
- Create: `manager/frontend/src/locales/en.js`
- Read: `manager/frontend/src/stores/auth.js` (tham khảo pattern Composition API)

## Implementation Steps

1. Tạo `src/stores/locale.js` theo Composition API pattern (giống auth.js)
2. Tạo `src/composables/useLocale.js` với `t(key)` + fallback chain
3. Tạo `src/locales/zh.js`, `vi.js`, `en.js` với `export default {}`
4. Không cần register store vào main.js — Pinia tự-register khi `useLocaleStore()` được gọi lần đầu

## Success Criteria

- [ ] `src/stores/locale.js` tồn tại, export `useLocaleStore`
- [ ] `src/composables/useLocale.js` tồn tại, export `useLocale`
- [ ] 3 locale files tồn tại, export `default {}`
- [ ] Import `useLocale` trong bất kỳ Vue file nào không throw lỗi

## Risk Assessment

- **Thấp** — không modify file hiện có, chỉ tạo mới
