---
phase: 2
title: "Extract & Translate Strings"
status: done
priority: P1
effort: "2h"
dependencies: [1]
---

# Phase 2: Extract & Translate Strings

## Overview

Dùng workflow tự động để scan toàn bộ 54 `.vue` files, extract tất cả text tiếng Trung, sinh key tự động, rồi tạo 3 locale files đầy đủ (zh/vi/en). Đây là bước nặng nhất — xử lý ~22.757 ký tự phân tán trong 54 files.

## Architecture

**Workflow 2 phase:**

```
Phase A (parallel): 54 agents, mỗi agent đọc 1 file Vue
  → Extract tất cả Chinese text từ template + script
  → Trả về: { file, strings: [{text: '登录', key: 'login'}] }

Phase B (1 agent): Tổng hợp
  → Dedup strings trùng nhau (cùng text → cùng key)
  → Sinh zh.js, vi.js, en.js đầy đủ
  → Ghi vào src/locales/
```

**Key naming convention (tự động):**
- Dùng text tiếng Trung làm seed để sinh key tiếng Anh ngắn gọn
- Ưu tiên semantic: `'登录'` → `login`, `'配置向导'` → `config_wizard`
- Với text dài: lấy 3-4 từ đầu, snake_case
- Với text trùng: dùng cùng key (dedup đảm bảo DRY)

**Locale file format:**
```js
// zh.js
export default {
  login: '登录',
  config_wizard: '配置向导',
  device_management: '设备管理',
  // ...
}
```

## Related Code Files

- Read: tất cả 54 `manager/frontend/src/**/*.vue`
- Write: `manager/frontend/src/locales/zh.js`
- Write: `manager/frontend/src/locales/vi.js`
- Write: `manager/frontend/src/locales/en.js`

## Implementation Steps

1. Chạy workflow scan song song 54 Vue files:
   - Mỗi agent: đọc file, extract text tiếng Trung trong `<template>` và string literals trong `<script>`
   - Trả về danh sách `{text, context}` per file
2. Agent tổng hợp:
   - Gom tất cả strings từ 54 agents
   - Dedup theo text: cùng text tiếng Trung → cùng key
   - Sinh key gợi nhớ từ text (ưu tiên semantic, fallback snake_case)
3. Agent dịch tạo locale files:
   - `zh.js`: `{ key: text_gốc }` (baseline)
   - `vi.js`: `{ key: bản_dịch_tiếng_việt }` (AI translate)
   - `en.js`: `{ key: bản_dịch_tiếng_anh }` (AI translate)
4. Ghi 3 files vào `src/locales/`

## Key Extraction Rules

Chỉ extract text:
- Trong `<template>`: text nodes (`>text<`), placeholder/label attrs (`placeholder="..."`, `label="..."`, `title="..."`)
- Trong `<script>`: string literals là message/label (e.g. ElMessage, toast, confirm dialogs)

**Không extract:**
- Config keys, API paths, variable names
- Strings trong comments
- HTML attributes không phải human-readable (class, id, ref, v-model...)

## Success Criteria

- [ ] `src/locales/zh.js` có ≥ 200 keys (estimate từ 22k chars)
- [ ] `src/locales/vi.js` có cùng số keys như zh.js
- [ ] `src/locales/en.js` có cùng số keys như zh.js
- [ ] Không có key trùng lặp trong cùng locale file
- [ ] Mọi key đều là snake_case hợp lệ

## Risk Assessment

- **Trung bình** — AI extraction có thể miss một số strings hoặc extract nhầm config values
- **Mitigation:** Phase 3 kiểm tra lại khi replace — nếu string không có trong locale, giữ nguyên để xử lý thủ công sau
- **Translation quality:** AI translate đủ tốt cho admin UI (nội bộ); không cần manual review trước khi ship
