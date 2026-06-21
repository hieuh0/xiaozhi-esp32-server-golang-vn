# Frontend Code Standards

**xiaozhi-esp32-server-golang** — Manager Dashboard (React + TypeScript)

---

## 1. File Organization

```
manager/frontend/src/
├── routes/                           # TanStack Router file-based routes
│   ├── _auth/
│   │   ├── _layout.tsx               # Protected layout
│   │   ├── dashboard.tsx
│   │   ├── admin/
│   │   │   ├── asr-config.tsx
│   │   │   └── llm-config.tsx
│   │   └── user/
│   │       └── devices.tsx
│   ├── index.tsx                     # Public home
│   ├── login.tsx
│   └── register.tsx
├── components/
│   ├── admin/
│   │   ├── asr-config-form.tsx
│   │   ├── tts-config-form.tsx
│   │   └── provider-fields.tsx
│   ├── layout/
│   │   ├── app-header.tsx
│   │   ├── sidebar.tsx
│   │   └── app-layout.tsx
│   ├── ui/
│   │   ├── button.tsx
│   │   ├── input.tsx
│   │   └── form.tsx
│   └── charts/
│       └── latency-chart.tsx
├── hooks/
│   ├── useDeviceQuery.ts
│   └── useConfigMutation.ts
├── services/
│   ├── devices.ts
│   ├── config.ts
│   └── auth.ts
├── i18n/
│   ├── en.ts
│   ├── vi.ts
│   └── zh.ts
├── types/
│   └── index.ts
├── lib/
│   ├── utils.ts
│   └── api-client.ts
├── styles/
│   └── globals.css
└── main.tsx
```

---

## 2. Naming Conventions

**Files**
- Components: `kebab-case.tsx` (e.g., `asr-config-form.tsx`)
- Hooks: `camelCase.ts` (e.g., `useDeviceQuery.ts`)
- Utils: `kebab-case.ts` (e.g., `format-utils.ts`)
- Types: `index.ts` or `{name}.types.ts`

**Variables & Functions**
- Component: PascalCase (e.g., `ASRConfigForm`)
- Hooks: `useXxx` (e.g., `useDeviceQuery`)
- Constants: UPPER_SNAKE_CASE (e.g., `API_BASE_URL`)
- Functions: camelCase (e.g., `formatLatency()`)

**Types**
- Interfaces: PascalCase (e.g., `Device`, `ConfigForm`)
- Enums: PascalCase (e.g., `ProviderType`, `AuthRole`)
- Unions: `T | U` (e.g., `"success" | "error"`)

---

## 3. Component Structure

```tsx
import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { useDeviceQuery } from "@/hooks/useDeviceQuery";

interface DeviceListProps {
  filter?: string;
  onSelect?: (deviceId: string) => void;
}

/**
 * DeviceList renders a table of connected devices.
 */
export function DeviceList({ filter, onSelect }: DeviceListProps) {
  const { data: devices, isLoading } = useDeviceQuery(filter);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const handleSelect = useCallback(
    (id: string) => {
      setSelectedId(id);
      onSelect?.(id);
    },
    [onSelect]
  );

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="space-y-4">
      {devices?.map((device) => (
        <DeviceRow
          key={device.id}
          device={device}
          isSelected={selectedId === device.id}
          onSelect={handleSelect}
        />
      ))}
    </div>
  );
}

function DeviceRow({
  device,
  isSelected,
  onSelect,
}: {
  device: Device;
  isSelected: boolean;
  onSelect: (id: string) => void;
}) {
  return (
    <div
      className={`p-4 rounded border cursor-pointer ${
        isSelected ? "border-blue-500 bg-blue-50" : "border-gray-300"
      }`}
      onClick={() => onSelect(device.id)}
    >
      <h3>{device.name}</h3>
      <p className="text-sm text-gray-600">{device.id}</p>
    </div>
  );
}
```

---

## 4. Forms (React Hook Form + Zod)

```tsx
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Button } from "@/components/ui/button";

const schema = z.object({
  provider: z.enum(["funasr", "doubao", "xunfei"]),
  funasr: z.object({
    ws_url: z.string().url(),
    language: z.enum(["zh", "en", "vi"]),
  }).optional(),
});

type FormData = z.infer<typeof schema>;

export function ASRConfigForm({ onSubmit }: { onSubmit: (data: FormData) => Promise<void> }) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  const provider = watch("provider");

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <select {...register("provider")}>
        <option value="funasr">FunASR</option>
        <option value="doubao">Doubao</option>
        <option value="xunfei">Xunfei</option>
      </select>
      {errors.provider && <span>{errors.provider.message}</span>}

      {provider === "funasr" && (
        <>
          <input {...register("funasr.ws_url")} placeholder="ws://..." />
          {errors.funasr?.ws_url && <span>{errors.funasr.ws_url.message}</span>}
        </>
      )}

      <Button type="submit" disabled={isSubmitting}>
        {isSubmitting ? "Saving..." : "Save"}
      </Button>
    </form>
  );
}
```

---

## 5. API Client (Axios + React Query)

```tsx
// lib/api-client.ts
import axios from "axios";

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080",
  timeout: 10000,
});

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem("auth_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export default apiClient;

// services/devices.ts
import apiClient from "@/lib/api-client";
import { Device } from "@/types";

export async function fetchDevices(filter?: string): Promise<Device[]> {
  const { data } = await apiClient.get<Device[]>("/api/devices", {
    params: { filter },
  });
  return data;
}

// hooks/useDeviceQuery.ts
import { useQuery } from "@tanstack/react-query";
import { fetchDevices } from "@/services/devices";

export function useDeviceQuery(filter?: string) {
  return useQuery({
    queryKey: ["devices", filter],
    queryFn: () => fetchDevices(filter),
    staleTime: 1000 * 60 * 5,
  });
}
```

---

## 6. Styling (Tailwind CSS)

```tsx
// Always use Tailwind classes; avoid inline styles
export function DeviceCard({ device }: { device: Device }) {
  return (
    <div className="p-4 rounded-lg border border-gray-200 shadow-sm hover:shadow-md transition-shadow">
      <h3 className="text-lg font-semibold text-gray-900">{device.name}</h3>
      <p className="mt-2 text-sm text-gray-600">{device.description}</p>
      <div className="mt-4 flex gap-2">
        <button className="px-3 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">
          Configure
        </button>
        <button className="px-3 py-2 border border-gray-300 text-gray-700 rounded text-sm hover:bg-gray-50">
          Remove
        </button>
      </div>
    </div>
  );
}
```

**shadcn/ui components**: Always use pre-built components:

```tsx
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";

export function MyForm() {
  return (
    <Card>
      <Input placeholder="Enter name" />
      <Button variant="primary">Submit</Button>
    </Card>
  );
}
```

---

## 7. Internationalization (i18n)

```ts
// i18n/en.ts
export const en = {
  common: {
    save: "Save",
    cancel: "Cancel",
    delete: "Delete",
  },
  devices: {
    title: "Devices",
    addNew: "Add Device",
    status: {
      online: "Online",
      offline: "Offline",
    },
  },
};

// i18n/vi.ts
export const vi = {
  common: {
    save: "Lưu",
    cancel: "Hủy",
    delete: "Xóa",
  },
  devices: {
    title: "Thiết Bị",
    addNew: "Thêm Thiết Bị",
    status: {
      online: "Trực Tuyến",
      offline: "Ngoại Tuyến",
    },
  },
};

// Usage
import { useTranslation } from "react-i18next";

export function DeviceTitle() {
  const { t } = useTranslation();
  return <h1>{t("devices.title")}</h1>;
}
```

---

## 8. Type Safety

```tsx
// types/index.ts
export interface Device {
  id: string;
  name: string;
  status: "online" | "offline" | "idle";
  lastSeen: string;
  config?: DeviceConfig;
}

export interface DeviceConfig {
  asr: ASRConfig;
  tts: TTSConfig;
  llm: LLMConfig;
}

// Always type component props
type Props = {
  devices: Device[];
  onDeviceSelect: (device: Device) => void;
  isLoading?: boolean;
};

export function DeviceList({ devices, onDeviceSelect, isLoading }: Props) {
  // Component body
}
```

---

## 9. Testing (Vitest + React Testing Library)

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { DeviceList } from "./device-list";

describe("DeviceList", () => {
  it("renders device list", () => {
    const devices = [{ id: "1", name: "Device 1" }];
    render(<DeviceList devices={devices} onDeviceSelect={vi.fn()} />);
    expect(screen.getByText("Device 1")).toBeInTheDocument();
  });

  it("calls onSelect when device clicked", () => {
    const onSelect = vi.fn();
    const devices = [{ id: "1", name: "Device 1" }];
    render(<DeviceList devices={devices} onDeviceSelect={onSelect} />);
    fireEvent.click(screen.getByText("Device 1"));
    expect(onSelect).toHaveBeenCalled();
  });
});
```

---

## 10. Best Practices

**Do**:
- Use functional components + hooks (no class components)
- Memoize expensive computations with `useMemo`
- Use `React.memo()` for expensive list items
- Lazy-load routes for large applications
- Keep components under 300 lines
- Extract nested components to separate files

**Don't**:
- Avoid inline functions in props (hurts performance)
- Don't mutate state directly
- Avoid multiple `useState` for related state (use `useReducer` instead)
- Don't fetch data in component body (use React Query or custom hooks)
- Avoid deeply nested JSX (extract subcomponents)

---

## 11. Common Patterns

**Custom Hook for Data Fetching**:

```tsx
function useFetchDevices(filter?: string) {
  return useQuery({
    queryKey: ["devices", filter],
    queryFn: () => fetchDevices(filter),
    staleTime: 5 * 60 * 1000,
    retry: 2,
  });
}
```

**Mutation with Optimistic Update**:

```tsx
const { mutate: saveDevice } = useMutation({
  mutationFn: (device: Device) => saveDeviceAPI(device),
  onMutate: async (newDevice) => {
    // Optimistically update UI
    queryClient.setQueryData(["devices"], (old: Device[]) => [
      ...old.filter(d => d.id !== newDevice.id),
      newDevice,
    ]);
  },
  onError: (error, newDevice, context) => {
    // Revert on error
    toast.error("Failed to save device");
  },
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["devices"] });
    toast.success("Device saved");
  },
});
```

---

## 12. Performance Tips

- Use `React.memo()` for components that receive same props often
- Lazy-load routes: `const Page = lazy(() => import('./pages/device'))`
- Use `virtualizer` for long lists (100+ items)
- Code-split CSS with Tailwind's `@apply`
- Monitor bundle size with `npm run build -- --analyze`

---

## 13. Code Review Checklist

- [ ] Components are under 300 lines
- [ ] No console warnings or errors
- [ ] All TypeScript types are strict
- [ ] Props are properly typed
- [ ] No inline functions in JSX props
- [ ] Accessibility: ARIA labels, semantic HTML
- [ ] Dark mode support (if applicable)
- [ ] Mobile responsive (test on small screens)
- [ ] Form validation matches backend
- [ ] Error states handled
- [ ] Loading states visible
- [ ] Empty states shown
