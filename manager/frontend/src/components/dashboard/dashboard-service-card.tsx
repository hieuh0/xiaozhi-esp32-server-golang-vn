import { Suspense, useState } from 'react'
import { Copy, Loader2 } from 'lucide-react'
import { useSuspenseQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { dashboardApi } from '@/features/dashboard/api/dashboard-api'
import { useLocale } from '@/hooks/use-locale'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

function ServiceCardContent() {
  const { t } = useLocale()
  const [otaTestLoading, setOtaTestLoading] = useState(false)
  const [otaTestResult, setOtaTestResult] = useState<string | null>(null)
  const { data } = useSuspenseQuery({
    queryKey: ['service-address'],
    queryFn: dashboardApi.getServiceAddress,
    staleTime: 60_000,
  })

  const rows = [
    { label: 'OTA', value: data.otaUrl },
    { label: 'WS', value: data.wsUrl },
    { label: 'MQTT', value: data.mqttEndpoint },
    { label: 'UDP', value: data.udpAddress },
  ].filter((r) => r.value)

  const copyAddress = (text: string) => {
    navigator.clipboard.writeText(text)
      .then(() => toast.success(t('copied_to_clipboard')))
      .catch(() => toast.error(t('copy_failed')))
  }

  const runOtaTest = async () => {
    setOtaTestLoading(true)
    setOtaTestResult(null)
    try {
      const res = await dashboardApi.runOtaTest()
      setOtaTestResult(res.text || t('detail_not_available'))
      toast[res.ok ? 'success' : 'warning'](res.message || (res.ok ? t('ota_test_passed') : t('ota_test_failed')))
    } catch {
      toast.error(t('ota_test_request_failed'))
    } finally {
      setOtaTestLoading(false)
    }
  }

  return (
    <>
      {rows.length === 0 ? (
        <p className="text-sm text-[var(--color-text-secondary)]">{t('no_ota_config')}</p>
      ) : (
        <div className="flex flex-col gap-2.5">
          {rows.map((row) => (
            <div key={row.label} className="grid grid-cols-[64px_1fr_auto] items-center gap-2.5 px-4 py-3 rounded-lg bg-[var(--color-surface-2)] border border-[var(--color-line)]">
              <span className="text-[11px] font-bold tracking-widest text-[var(--color-text-tertiary)]">{row.label}</span>
              <span className="text-sm font-medium text-[var(--color-text)] truncate" title={row.value}>{row.value}</span>
              <Button variant="ghost" size="icon" className="w-7 h-7" aria-label={t('copy')} onClick={() => copyAddress(row.value)}>
                <Copy className="w-3.5 h-3.5" />
              </Button>
            </div>
          ))}
        </div>
      )}
      {otaTestResult !== null && (
        <div className="mt-4 flex flex-col gap-2">
          <Badge variant="secondary">{t('ota_return_chip')}</Badge>
          <pre className="text-xs leading-relaxed p-3 rounded-lg bg-[var(--color-surface-2)] border border-[var(--color-line)] max-h-44 overflow-auto whitespace-pre-wrap break-words">{otaTestResult}</pre>
        </div>
      )}
      <div className="mt-3 flex justify-end">
        <Button variant="outline" size="sm" disabled={otaTestLoading} onClick={runOtaTest}>
          {otaTestLoading && <Loader2 className="w-4 h-4 mr-1.5 animate-spin" />}{t('ota_test')}
        </Button>
      </div>
    </>
  )
}

export function DashboardServiceCard() {
  const { t } = useLocale()
  return (
    <Card>
      <CardHeader className="pb-3">
        <p className="text-[11px] font-bold tracking-widest text-[var(--color-text-tertiary)] uppercase mb-1">SERVICE ADDRESS</p>
        <h3 className="text-lg font-semibold text-[var(--color-text)]">{t('service_address')}</h3>
      </CardHeader>
      <CardContent>
        <Suspense fallback={<div className="flex flex-col gap-2.5">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}</div>}>
          <ServiceCardContent />
        </Suspense>
      </CardContent>
    </Card>
  )
}
