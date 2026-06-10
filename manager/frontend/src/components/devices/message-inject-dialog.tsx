import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { devicesApi } from '@/features/devices/api/devices-api'
import { getDeviceDisplayName, isDeviceOnline } from '@/features/devices/types'
import type { Device } from '@/features/devices/types'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

interface MessageInjectDialogProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  devices: Device[]
  defaultDeviceId?: string
  lockDevice?: boolean
  onSuccess?: () => void
}

export function MessageInjectDialog({ open, onOpenChange, devices, defaultDeviceId, lockDevice, onSuccess }: MessageInjectDialogProps) {
  const { t } = useLocale()

  const [deviceId, setDeviceId] = useState(defaultDeviceId || '')
  const [message, setMessage] = useState('')
  const [skipLlm, setSkipLlm] = useState(false)
  const [autoListen, setAutoListen] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open) {
      setDeviceId(defaultDeviceId || '')
      setMessage('')
      setSkipLlm(false)
      setAutoListen(true)
    }
  }, [open, defaultDeviceId])

  const handleSubmit = async () => {
    if (!deviceId) { toast.error(t('select_device')); return }
    if (!message.trim()) { toast.error(t('enter_push_content')); return }
    setSubmitting(true)
    try {
      await devicesApi.injectMessage({ device_id: deviceId, message, skip_llm: skipLlm, auto_listen: autoListen })
      toast.success(t('voice_push_success'))
      onSuccess?.()
      onOpenChange(false)
    } catch (e) { toast.error((e as Error).message || t('voice_push_failed')) }
    finally { setSubmitting(false) }
  }

  const deviceSelectDisabled = lockDevice && !!defaultDeviceId

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{t('voice_push')}</DialogTitle>
        </DialogHeader>

        <div className="grid gap-4 py-2">
          <div className="grid gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text)]">{t('select_device_prompt')}</label>
            <Select value={deviceId} onValueChange={setDeviceId} disabled={deviceSelectDisabled}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t('select_push_voice_device')} />
              </SelectTrigger>
              <SelectContent>
                {devices.map((d) => {
                  const online = isDeviceOnline(d.last_active_at)
                  return (
                    <SelectItem key={d.id} value={d.device_name || ''}>
                      <div className="flex items-center justify-between gap-3 w-full">
                        <span className="font-medium truncate">{getDeviceDisplayName(d)}</span>
                        <span className={cn('text-[10px] px-1.5 py-0.5 rounded-full border font-medium', online ? 'status-success' : 'status-danger')}>
                          {online ? t('online') : t('offline')}
                        </span>
                      </div>
                    </SelectItem>
                  )
                })}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text)]">{t('push_content')}</label>
            <Textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={t('enter_broadcast_content')}
              maxLength={500}
              rows={4}
            />
            <p className="text-xs text-[var(--color-text-tertiary)] text-right">{message.length}/500</p>
          </div>

          <div className="flex items-center justify-between gap-4 px-4 py-3 rounded-xl bg-[var(--color-surface-2)] border border-[var(--color-line)]">
            <div>
              <p className="text-sm font-semibold text-[var(--color-text)]">{t('direct_broadcast')} — {skipLlm ? t('enable') : t('close')}</p>
              <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{skipLlm ? t('msg_direct_tts') : t('msg_via_llm')}</p>
            </div>
            <Switch checked={skipLlm} onCheckedChange={setSkipLlm} />
          </div>

          <div className="flex items-center justify-between gap-4 px-4 py-3 rounded-xl bg-[var(--color-surface-2)] border border-[var(--color-line)]">
            <div>
              <p className="text-sm font-semibold text-[var(--color-text)]">{t('switch_to_idle')} — {!autoListen ? t('enable') : t('close')}</p>
              <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">{!autoListen ? t('broadcast_return_idle') : t('broadcast_continue_listen')}</p>
            </div>
            <Switch checked={!autoListen} onCheckedChange={(v) => setAutoListen(!v)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t('cancel')}</Button>
          <Button disabled={submitting} onClick={handleSubmit}>
            {submitting ? t('pushing') : t('voice_push')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
