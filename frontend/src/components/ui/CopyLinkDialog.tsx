import type { LucideIcon } from 'lucide-react'
import { Copy } from 'lucide-react'
import { useId, useRef, useState, type ReactNode } from 'react'
import { useI18n } from '../../i18n/context'
import { copyText } from '../../lib/clipboard'
import { Dialog } from './Dialog'
import { Button } from './ui'

type Props = {
  open: boolean
  onClose: () => void
  onCopied?: () => void
  icon: LucideIcon
  title: string
  hint: string
  url: string
  /** Extra content under the link (e.g. the invite code or a warning). */
  children?: ReactNode
  tone?: 'brand' | 'amber'
}

/** A sheet that shows a link with a copy button, falling back to manual copy. */
export function CopyLinkDialog({ open, onClose, onCopied, icon: Icon, title, hint, url, children, tone = 'brand' }: Props) {
  const { t } = useI18n()
  const inputRef = useRef<HTMLInputElement>(null)
  const [state, setState] = useState<'idle' | 'copied' | 'manual'>('idle')
  const titleId = useId()

  const close = () => {
    setState('idle')
    onClose()
  }

  const copy = async () => {
    if (await copyText(url, inputRef.current?.parentElement ?? undefined)) {
      setState('copied')
      onCopied?.()
      setTimeout(close, 600)
      return
    }
    // Both copy paths were blocked; select the text so a long-press copies it.
    inputRef.current?.select()
    setState('manual')
  }

  const badge = tone === 'amber' ? 'bg-amber-50 text-amber-600' : 'bg-brand-50 text-brand-600'

  return (
    <Dialog open={open} onClose={close} labelledBy={titleId}>
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <span className={`flex size-10 shrink-0 items-center justify-center rounded-full ${badge}`}>
            <Icon className="size-5" />
          </span>
          <div>
            <h2 id={titleId} className="text-lg font-bold">
              {title}
            </h2>
            <p className="text-sm text-slate-600">{hint}</p>
          </div>
        </div>

        <input
          ref={inputRef}
          readOnly
          value={url}
          onFocus={(e) => e.currentTarget.select()}
          aria-label={title}
          className="block w-full truncate rounded-xl bg-slate-100 px-3 py-3 font-mono text-sm text-slate-700 outline-none focus:ring-2 focus:ring-brand-500"
        />
        {children}
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
