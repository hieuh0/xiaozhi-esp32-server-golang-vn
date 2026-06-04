---
phase: 3
title: "Replace Templates"
status: done
priority: P1
effort: "2h"
dependencies: [1, 2]
---

# Phase 3: Replace Templates

## Overview

Dùng workflow song song để thay thế tất cả hardcoded Chinese text trong 54 Vue files bằng `t('key')` calls. Mỗi agent xử lý 1 file độc lập. Phase này cần locale files từ Phase 2 để tra key mapping.

## Architecture

**Lookup map từ Phase 2:**
```js
// Được build từ zh.js: { '登录': 'login', '配置向导': 'config_wizard', ... }
const textToKey = invertLocale(zh)
```

**Replacement patterns:**

| Context | Before | After |
|---------|--------|-------|
| Template text node | `>登录<` | `>{{ t('login') }}<` |
| el-button / span content | `<span>退出登录</span>` | `<span>{{ t('logout') }}</span>` |
| Attribute placeholder | `placeholder="请输入"` | `:placeholder="t('please_input')"` |
| Attribute label | `label="用户名"` | `:label="t('username')"` |
| Script ElMessage | `ElMessage('登录成功')` | `ElMessage(t('login_success'))` |
| Script confirm | `'确认删除?'` | `t('confirm_delete')` |

**Setup trong script section:**
```js
// Thêm vào <script setup> nếu chưa có:
import { useLocale } from '../composables/useLocale'  // (đường dẫn tương đối)
const { t } = useLocale()
```

## Related Code Files

- Read + Write: tất cả 54 `manager/frontend/src/**/*.vue`
- Read: `manager/frontend/src/locales/zh.js` (để build reverse map)

## Implementation Steps

1. Build reverse map từ `zh.js`: `{ '中文text': 'key_name' }`
2. Workflow song song — 1 agent per Vue file:
   a. Đọc file content
   b. Trong `<template>`: tìm Chinese text, thay bằng `{{ t('key') }}`
   c. Trong `<template>`: tìm Chinese attrs, thay bằng `:attr="t('key')"`
   d. Trong `<script setup>`: tìm Chinese strings, thay bằng `t('key')`
   e. Thêm `import { useLocale }` nếu chưa có + `const { t } = useLocale()`
   f. Ghi file đã modified
3. Agent kiểm tra sau replace: grep Chinese chars còn sót, log ra `plans/reports/unmatched-strings-i18n-report.txt` để fix thủ công

## Import Path Rules

Đường dẫn import phụ thuộc vị trí file:
- `src/views/*.vue` → `'../composables/useLocale'`
- `src/views/admin/*.vue` → `'../../composables/useLocale'`
- `src/views/admin/forms/*.vue` → `'../../../composables/useLocale'`
- `src/views/user/*.vue` → `'../../composables/useLocale'`
- `src/views/mobile/*.vue` → `'../../composables/useLocale'`
- `src/components/*.vue` → `'../composables/useLocale'`
- `src/components/common/*.vue` → `'../../composables/useLocale'`
- `src/components/user/*.vue` → `'../../composables/useLocale'`
- `src/App.vue` → `'./composables/useLocale'`

## Success Criteria

- [ ] `grep -rh "[一-鿿]" src/` trả về < 50 matches (cho phép một số edge cases)
- [ ] Mọi file có `t(` đều có `import { useLocale }` và `const { t } = useLocale()`
- [ ] Không có `t(` gọi với key không tồn tại trong zh.js (warn log)
- [ ] `vite build` không có compile errors

## Risk Assessment

- **Trung bình-cao** — Replace tự động có thể sinh lỗi syntax nếu Chinese text nằm trong expression phức tạp
- **Mitigation:** Sau replace, chạy `vite build` kiểm tra; các lỗi compile sẽ chỉ đúng file/dòng cần fix thủ công
- **Fallback:** Strings không có trong locale map → giữ nguyên text gốc (không replace), log ra `plans/reports/unmatched-strings-i18n-report.txt`
- **Scope:** Chỉ xử lý `<script setup>` — files dùng Options API (nếu có) giữ nguyên, fix thủ công sau
