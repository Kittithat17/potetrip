import { Copy, Link2 } from 'lucide-react'
import { useId, useRef, useState } from 'react'
import { Dialog } from '../../components/Dialog'
import { Button } from '../../components/ui'
import { useI18n } from '../../i18n/context'
import { inviteLink } from '../../lib/invite'
import { useTrip } from './context'

type Props = { open: boolean; onClose: () => void; onCopied: () => void }

/** Our own invite sheet, used where the native share sheet isn't available. */
export function InviteDialog({ open, onClose, onCopied }: Props) {
  const { t } = useI18n()
  const { trip } = useTrip()
  const url = inviteLink(trip.invite_code)
  const inputRef = useRef<HTMLInputElement>(null)
  const [state, setState] = useState<'idle' | 'copied' | 'manual'>('idle')
  const titleId = useId()

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url)
      setState('copied')
      onCopied()
      setTimeout(onClose, 600)
    } catch {
      // The Clipboard API needs HTTPS; select the text so a long-press copies it.
      inputRef.current?.select()
      setState('manual')
    }
  }

  const close = () => {
    setState('idle')
    onClose()
  }

  return (
    <Dialog open={open} onClose={close} labelledBy={titleId}>
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-brand-50 text-brand-600">
            <Link2 className="size-5" />
          </span>
          <div>
            <h2 id={titleId} className="text-lg font-bold">
              {t('trip.inviteTitle')}
            </h2>
            <p className="text-sm text-slate-600">{t('trip.inviteHint')}</p>
          </div>
        </div>

        <input
          ref={inputRef}
          readOnly
          value={url}
          onFocus={(e) => e.currentTarget.select()}
          aria-label={t('trip.inviteTitle')}
          className="block w-full truncate rounded-xl bg-slate-100 px-3 py-3 font-mono text-sm text-slate-700 outline-none focus:ring-2 focus:ring-brand-500"
        />
        <p className="text-center text-xs text-slate-500">
          {t('trip.inviteCode')}: <span className="font-mono font-semibold tracking-widest text-slate-700">{trip.invite_code}</span>
        </p>
        {state === 'manual' && <p className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">{t('trip.copyManual')}</p>}

        <div className="grid grid-cols-2 gap-2">
          <Button variant="secondary" onClick={close}>
            {t('common.close')}
          </Button>
          <Button onClick={copy} data-autofocus>
            <Copy className="size-4" />
            {state === 'copied' ? t('common.copied') : t('common.copy')}
          </Button>
        </div>
      </div>
    </Dialog>
  )
}
