import { Loader2 } from 'lucide-react'
import { Button } from './button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from './dialog'
import { useLocale } from '@/hooks/use-locale'

interface ConfirmDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  description: string
  isLoading?: boolean
  confirmLabel?: string
  cancelLabel?: string
  confirmVariant?: 'default' | 'destructive'
}

export function ConfirmDialog({
  open, onClose, onConfirm, title, description, isLoading,
  confirmLabel, cancelLabel, confirmVariant = 'destructive',
}: ConfirmDialogProps) {
  const { t } = useLocale()
  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-[var(--color-text-secondary)]">{description}</p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isLoading}>
            {cancelLabel ?? t('cancel')}
          </Button>
          <Button variant={confirmVariant} onClick={onConfirm} disabled={isLoading}>
            {isLoading && <Loader2 className="w-4 h-4 mr-1.5 animate-spin" />}
            {confirmLabel ?? t('confirm_delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
