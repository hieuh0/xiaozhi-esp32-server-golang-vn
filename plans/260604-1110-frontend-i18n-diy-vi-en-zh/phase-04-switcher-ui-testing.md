---
phase: 4
title: "Switcher UI & Testing"
status: done
priority: P2
effort: "1h"
dependencies: [1, 2, 3]
---

# Phase 4: Switcher UI & Testing

## Overview

Thêm language switcher vào AppHeader, verify switching hoạt động đúng, và kiểm tra các edge cases: localStorage persist, reload giữ nguyên ngôn ngữ, fallback key missing.

## Architecture

**AppHeader language switcher (Element Plus `el-dropdown`):**

```html
<!-- Thêm vào toolbar section trong AppHeader.vue -->
<el-dropdown @command="setLang" trigger="click">
  <el-button text>
    <span>{{ langLabel }}</span>
    <el-icon><ArrowDown /></el-icon>
  </el-button>
  <template #dropdown>
    <el-dropdown-menu>
      <el-dropdown-item command="vi" :class="{ active: lang === 'vi' }">
        🇻🇳 Tiếng Việt
      </el-dropdown-item>
      <el-dropdown-item command="en" :class="{ active: lang === 'en' }">
        🇬🇧 English
      </el-dropdown-item>
      <el-dropdown-item command="zh" :class="{ active: lang === 'zh' }">
        🇨🇳 中文
      </el-dropdown-item>
    </el-dropdown-menu>
  </template>
</el-dropdown>
```

```js
// Trong <script setup> của AppHeader.vue
import { useLocale } from '../composables/useLocale'
const { t, lang, setLang } = useLocale()

const langLabel = computed(() => ({
  vi: '🇻🇳 VI', en: '🇬🇧 EN', zh: '🇨🇳 ZH'
})[lang.value] ?? 'VI')
```

## Related Code Files

- Modify: `manager/frontend/src/components/AppHeader.vue`

## Implementation Steps

1. Mở `AppHeader.vue`, tìm vị trí phù hợp trong header toolbar (cạnh logout button)
2. Thêm `el-dropdown` language switcher như mẫu trên
3. Import `useLocale`, destructure `{ t, lang, setLang }`
4. Thêm computed `langLabel` để hiển thị flag + code ngắn
5. Manual test:
   - Mở app, kiểm tra ngôn ngữ mặc định là VI
   - Switch sang EN → tất cả text chuyển sang tiếng Anh
   - Switch sang ZH → text về tiếng Trung gốc
   - Reload page → ngôn ngữ giữ nguyên (localStorage persist)
6. Kiểm tra mobile: `MobileNavBar.vue` và `MobileTabBar.vue` cũng cần `t()` calls nếu có text

## Success Criteria

- [ ] Language switcher hiển thị trong AppHeader
- [ ] Click VI/EN/ZH → toàn bộ UI text đổi ngay lập tức (reactive)
- [ ] Reload page → giữ nguyên ngôn ngữ đã chọn
- [ ] Fallback: key missing trong vi.js/en.js → hiển thị text từ zh.js (không crash)
- [ ] `vite build` thành công, không warnings liên quan i18n

## Risk Assessment

- **Thấp** — AppHeader đã có slot toolbar, chỉ thêm component
- **Edge case:** MobileLayout có navbar riêng — kiểm tra text ở đây có được dịch không
