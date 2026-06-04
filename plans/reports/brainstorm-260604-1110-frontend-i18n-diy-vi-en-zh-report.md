# Brainstorm: DIY i18n cho Frontend (VI/EN/ZH)

## Vấn đề
- 54 file Vue, ~22.757 ký tự tiếng Trung hardcoded trong templates
- Không có vue-i18n, không có locale files
- Yêu cầu: hỗ trợ VI/EN/ZH, không dùng thư viện vue-i18n

## Quyết định đã chốt

| Hạng mục | Quyết định |
|----------|-----------|
| Kiến trúc | DIY `useLocale` composable + Pinia store |
| Ngôn ngữ mặc định | Tiếng Việt (`vi`) |
| Persist | `localStorage` key `lang` |
| Switcher UI | AppHeader dropdown (Element Plus) |
| Ngôn ngữ hỗ trợ | zh / vi / en |

## Kiến trúc

```
src/
  locales/
    zh.js        # Extracted từ codebase hiện tại
    vi.js        # Dịch sang tiếng Việt
    en.js        # Dịch sang tiếng Anh
  stores/
    locale.js    # Pinia store: currentLang + setLang + localStorage
  composables/
    useLocale.js # t(key) + currentLang + setLang
```

## Core implementation (~30 dòng)

```js
// stores/locale.js
export const useLocaleStore = defineStore('locale', {
  state: () => ({ lang: localStorage.getItem('lang') || 'vi' }),
  actions: { setLang(l) { this.lang = l; localStorage.setItem('lang', l) } }
})

// composables/useLocale.js
const maps = { zh, vi, en }
export function useLocale() {
  const store = useLocaleStore()
  const t = (key) => maps[store.lang]?.[key] ?? maps.zh[key] ?? key
  return { t, lang: computed(() => store.lang), setLang: store.setLang }
}
```

## Phạm vi

- **Trong scope:** 54 .vue files, 3 ngôn ngữ, AppHeader switcher, localStorage
- **Ngoài scope:** .js utils files, dynamic strings từ server/API

## Phases triển khai

1. Tạo infrastructure (stores/locale.js, composables/useLocale.js, locales/zh.js skeleton)
2. Workflow extract + translate: scan 54 files → sinh keys → tạo zh.js/vi.js/en.js
3. Workflow replace: thay hardcoded text bằng `{{ t('key') }}` trong templates
4. Thêm language switcher vào AppHeader.vue
5. Test: kiểm tra switching hoạt động, fallback keys, localStorage persist
