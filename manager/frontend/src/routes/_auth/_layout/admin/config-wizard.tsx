import { useEffect } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Check, Copy } from 'lucide-react'
import { useLocale } from '@/hooks/use-locale'
import { useConfigWizard, type GenericConfigForm } from '@/hooks/use-config-wizard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const STEPS = [
  { title: 'OTA', sub: 'Service Address' },
  { title: 'VAD', sub: 'Voice Activity' },
  { title: 'ASR', sub: 'Speech Recognition' },
  { title: 'LLM', sub: 'Language Model' },
  { title: 'TTS', sub: 'Text to Speech' },
]

function Stepper({ current }: { current: number }) {
  return (
    <div className="flex items-center justify-center gap-1 mb-6 flex-wrap">
      {STEPS.map((step, i) => (
        <div key={i} className="flex items-center gap-1">
          <div className="flex items-center gap-1.5">
            <div className={['w-7 h-7 rounded-full flex items-center justify-center text-xs font-semibold border-2 transition-colors',
              i < current ? 'bg-[var(--color-primary)] border-[var(--color-primary)] text-white'
                : i === current ? 'border-[var(--color-primary)] text-[var(--color-primary)] bg-transparent'
                : 'border-[var(--color-line)] text-[var(--color-text-tertiary)]'].join(' ')}>
              {i < current ? <Check className="w-3.5 h-3.5" /> : <span>{i + 1}</span>}
            </div>
            <div className="hidden sm:flex flex-col leading-none">
              <span className={`text-xs font-semibold ${i <= current ? 'text-[var(--color-text)]' : 'text-[var(--color-text-tertiary)]'}`}>{step.title}</span>
              <span className="text-[10px] text-[var(--color-text-tertiary)] mt-0.5">{step.sub}</span>
            </div>
          </div>
          {i < STEPS.length - 1 && <div className={`h-0.5 w-8 mx-1 transition-colors ${i < current ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-line)]'}`} />}
        </div>
      ))}
    </div>
  )
}

function GenericStepForm({ title, form, onChange }: { title: string; form: GenericConfigForm; onChange: (p: Partial<GenericConfigForm>) => void }) {
  const { t } = useLocale()
  return (
    <div className="grid gap-4">
      <p className="text-base font-semibold text-[var(--color-text)]">{title}</p>
      <div className="grid grid-cols-2 gap-4">
        <div className="grid gap-1.5">
          <label className="text-sm font-medium">{t('config_name')}</label>
          <Input value={form.name} onChange={e => onChange({ name: e.target.value })} />
        </div>
        <div className="grid gap-1.5">
          <label className="text-sm font-medium">{t('config_id')}</label>
          <Input value={form.config_id} onChange={e => onChange({ config_id: e.target.value })} />
        </div>
      </div>
      <div className="grid gap-1.5">
        <label className="text-sm font-medium">{t('provider')}</label>
        <Input value={form.provider} onChange={e => onChange({ provider: e.target.value })} />
      </div>
      <div className="grid gap-1.5">
        <label className="text-sm font-medium">JSON {t('config')}</label>
        <Textarea value={form.json_data} onChange={e => onChange({ json_data: e.target.value })} rows={8} className="font-mono text-xs resize-y" placeholder="{}" />
      </div>
    </div>
  )
}

function ConfigWizardPage() {
  const { t } = useLocale()
  const navigate = useNavigate()
  const wiz = useConfigWizard()

  useEffect(() => { wiz.initialize() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const { currentStep, saving, testing, otaTestResult, otaForm, setOta } = wiz

  return (
    <div className="px-6 pb-8 max-w-[860px] mx-auto">
      <Stepper current={currentStep} />

      <div className="rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-6">
        {/* Step 0: OTA */}
        {currentStep === 0 && (
          <div className="grid gap-4">
            <div>
              <p className="text-base font-semibold text-[var(--color-text)] mb-1">{t('ota_config')}</p>
              <p className="text-sm text-[var(--color-text-secondary)]">{t('domain_ip_hint')}</p>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="grid gap-1.5 col-span-2">
                <label className="text-sm font-medium">{t('host')}</label>
                <Input value={otaForm.host} onChange={e => setOta({ host: e.target.value })} placeholder="example.com or 192.168.1.1" />
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium">{t('port')}</label>
                <Input type="number" value={otaForm.port} onChange={e => setOta({ port: Number(e.target.value) })} min={1} max={65535} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="grid gap-1.5">
                <label className="text-sm font-medium">{t('protocol')}</label>
                <Select value={otaForm.protocol} onValueChange={v => setOta({ protocol: v as 'http' | 'https' })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="https">HTTPS</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-1.5">
                <label className="text-sm font-medium">{t('signature_key')}</label>
                <Input value={otaForm.signature_key} onChange={e => setOta({ signature_key: e.target.value })} />
              </div>
            </div>
            <div className="flex items-center gap-3 pt-1">
              <Switch checked={otaForm.enableMqttUdp} onCheckedChange={v => setOta({ enableMqttUdp: v })} />
              <span className="text-sm font-medium">{t('enable_mqtt_udp')}</span>
            </div>
            {otaForm.enableMqttUdp && (
              <div className="grid grid-cols-2 gap-3 pl-2 border-l-2 border-[var(--color-line)]">
                <div className="grid gap-1.5">
                  <label className="text-sm font-medium">{t('mqtt_server_port')}</label>
                  <Input type="number" value={otaForm.mqttServerPort} onChange={e => setOta({ mqttServerPort: Number(e.target.value) })} min={1} max={65535} />
                </div>
                <div className="grid gap-1.5">
                  <label className="text-sm font-medium">{t('udp_port')}</label>
                  <Input type="number" value={otaForm.udpPort} onChange={e => setOta({ udpPort: Number(e.target.value) })} min={1} max={65535} />
                </div>
              </div>
            )}
          </div>
        )}

        {currentStep === 1 && <GenericStepForm title={t('vad_config')} form={wiz.vadForm} onChange={wiz.setVad} />}
        {currentStep === 2 && <GenericStepForm title={t('asr_config')} form={wiz.asrForm} onChange={wiz.setAsr} />}
        {currentStep === 3 && <GenericStepForm title={t('llm_config')} form={wiz.llmForm} onChange={wiz.setLlm} />}
        {currentStep === 4 && <GenericStepForm title={t('tts_config')} form={wiz.ttsForm} onChange={wiz.setTts} />}

        {/* Step 5: Done */}
        {currentStep === 5 && (
          <div className="grid gap-4">
            <div>
              <p className="text-base font-semibold text-[var(--color-text)] mb-1">{t('config_done_title')}</p>
              <p className="text-sm text-[var(--color-text-secondary)]">{t('config_done_hint')}</p>
            </div>
            {[
              { label: t('ota_addr_label'), value: wiz.finalOtaUrl },
              { label: t('ws_addr_label'), value: wiz.finalWsUrl },
              ...(wiz.finalMqttEndpoint ? [{ label: t('mqtt_endpoint_label'), value: wiz.finalMqttEndpoint }] : []),
              ...(wiz.finalUdpEndpoint ? [{ label: t('udp_info_label'), value: wiz.finalUdpEndpoint }] : []),
            ].map(({ label, value }) => (
              <div key={label} className="grid gap-1.5">
                <label className="text-sm text-[var(--color-text-secondary)]">{label}</label>
                <div className="flex gap-2">
                  <Input value={value} readOnly className="flex-1 font-mono text-xs" />
                  <Button variant="outline" size="sm" onClick={() => wiz.copyToClipboard(value)}>
                    <Copy className="w-3.5 h-3.5 mr-1" />{t('copy')}
                  </Button>
                </div>
              </div>
            ))}
            <div className="border-t border-[var(--color-line)] pt-4 grid gap-3">
              <Button variant="outline" disabled={testing} onClick={wiz.runOtaTest} className="w-fit">
                {testing ? '...' : t('ota_test')}
              </Button>
              {otaTestResult !== null && (
                <pre className="p-3 rounded-xl border border-[var(--color-line)] bg-[var(--color-surface-muted)] text-xs leading-relaxed whitespace-pre-wrap break-all max-h-[280px] overflow-auto">{otaTestResult}</pre>
              )}
            </div>
          </div>
        )}

        {/* Navigation */}
        <div className="flex gap-2 mt-6 pt-4 border-t border-[var(--color-line)]">
          {currentStep > 0 && currentStep < 5 && (
            <Button variant="outline" onClick={wiz.prevStep}>{t('prev_step')}</Button>
          )}
          {currentStep < 5 && (
            <Button variant="outline" onClick={wiz.skipStep}>{t('skip_step')}</Button>
          )}
          {currentStep < 5 ? (
            <Button disabled={saving} onClick={wiz.saveAndNext}>
              {saving ? '...' : currentStep === 4 ? t('save_and_finish') : t('save_and_next')}
            </Button>
          ) : (
            <>
              <Button onClick={() => navigate({ to: '/dashboard' })}>{t('back_to_home')}</Button>
              <Button variant="outline" onClick={() => wiz.initialize()}>{t('reconfigure')}</Button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_auth/_layout/admin/config-wizard')({
  component: ConfigWizardPage,
})
