---
title: "Frontend DIY i18n: VI/EN/ZH Language Switching"
description: "Add Vietnamese/English/Chinese language switching to Vue 3 frontend using DIY composable, no vue-i18n"
status: done
priority: P2
branch: "main"
tags: [frontend, i18n, vue3, pinia]
blockedBy: []
blocks: []
created: "2026-06-04T06:31:02.895Z"
createdBy: "ck:plan"
source: skill
---

# Frontend DIY i18n: VI/EN/ZH Language Switching

## Overview

Thêm hỗ trợ đa ngôn ngữ (VI/EN/ZH) cho 54 Vue files trong `manager/frontend/src/`. Không dùng vue-i18n — xây DIY `useLocale` composable + Pinia store. Ngôn ngữ mặc định: Tiếng Việt. Persist bằng localStorage. Switcher tại AppHeader.

**Scope:** 54 `.vue` files, ~22.757 ký tự tiếng Trung hardcoded.
**Constraint:** Zero npm packages mới. Chỉ Vue 3 + Pinia đã có sẵn.

## Phases

| Phase | Name | Status | Effort |
|-------|------|--------|--------|
| 1 | [Infrastructure](./phase-01-infrastructure.md) | Done | 1h |
| 2 | [Extract & Translate Strings](./phase-02-extract-translate-strings.md) | Done | 2h |
| 3 | [Replace Templates](./phase-03-replace-templates.md) | Done | 2h |
| 4 | [Switcher UI & Testing](./phase-04-switcher-ui-testing.md) | Done | 1h |

## Key Files

- `manager/frontend/src/stores/locale.js` — create
- `manager/frontend/src/composables/useLocale.js` — create
- `manager/frontend/src/locales/zh.js` — create
- `manager/frontend/src/locales/vi.js` — create
- `manager/frontend/src/locales/en.js` — create
- `manager/frontend/src/components/AppHeader.vue` — modify (add switcher)
- 53 other `.vue` files — modify (replace hardcoded text with `t('key')`)

## Dependencies

None — standalone feature, no blockers.

## Validation Log

### Session 1 — 2026-06-04
**Trigger:** `/ck:plan validate` before implementation
**Questions asked:** 4

#### Questions & Answers

1. **[Architecture]** Phase 1 composable bug: `store.lang.value` → Pinia setup stores tự unwrap refs
   - Options: `store.lang` trực tiếp | `storeToRefs(store)`
   - **Answer:** `store.lang` trực tiếp
   - **Rationale:** Pinia 2.x setup stores unwrap refs automatically — `.value` sẽ là `undefined`

2. **[Scope]** Options API files có cần xử lý không?
   - Options: Bỏ qua, chỉ `<script setup>` | Xử lý cả Options API
   - **Answer:** Bỏ qua — chỉ `<script setup>`
   - **Rationale:** Kiểm tra nhanh cho thấy tất cả files dùng script setup; exceptions sẽ fix thủ công

3. **[Scope]** Unmatched strings log đặt ở đâu?
   - Options: `plans/reports/` | `manager/frontend/` root
   - **Answer:** `plans/reports/unmatched-strings-i18n-report.txt`
   - **Rationale:** Không commit artifact tạm vào src/

4. **[Assumptions]** AI translation có cần manual review?
   - Options: Không cần | Review vi.js thủ công
   - **Answer:** Không cần — AI translate đủ tốt cho admin UI nội bộ

#### Confirmed Decisions
- Pinia pattern: `store.lang` (không `.value`) — verified against Pinia 2.x docs
- Scope: chỉ `<script setup>` files
- Unmatched log: `plans/reports/unmatched-strings-i18n-report.txt`
- Translation: auto only, no manual review

#### Action Items
- [x] Phase 1: sửa `store.lang.value` → `store.lang` trong composable code
- [x] Phase 3: cập nhật log path sang `plans/reports/`
- [x] Phase 3: thêm note về Options API scope
- [x] Phase 2: thêm note về translation quality decision

#### Impact on Phases
- Phase 1: composable code fixed (critical bug)
- Phase 2: minor note thêm vào Risk Assessment
- Phase 3: log path + scope clarification

### Verification Results
- Claims checked: 12
- Verified: 10 | Failed: 1 | Unverified: 1
- Tier: Standard (4 phases)
- Failures: Phase 1 composable `store.lang.value` — FIXED post-interview

### Whole-Plan Consistency Sweep
- plan.md ↔ phase files: consistent ✓
- No stale terms or renamed APIs ✓
- No duplicate embedded drafts ✓
- Pinia pattern fix propagated to phase-01 ✓
- Log path consistent across phase-03 and plan.md ✓
- **Result: 0 unresolved contradictions — eligible for /ck:cook**
