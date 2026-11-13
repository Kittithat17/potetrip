import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Download, QrCode } from 'lucide-react'
import { useId, useState } from 'react'
import type { Transfer } from '../../../api/types'
import { Dialog } from '../../../components/ui/Dialog'
import { Button, Spinner } from '../../../components/ui/ui'
import { useI18n } from '../../../i18n/context'
import { formatShortDate } from '../../../lib/datetime'
import { HOME_CURRENCY, useHomeApprox } from '../../../lib/homeApprox'
import { exponent, formatMoney } from '../../../lib/money'
import { blobToDataUrl, formatPromptPay, promptPayQr, qrCardImage, saveImage } from './promptpay'

type Props = {
  transfer: Transfer | null
  receiverName: string
  promptpay: string
  currency: string // trip base currency
  onClose: () => void
}

/**
 * QR the payer scans with any Thai banking app. PromptPay only moves baht, so a
 * trip in another currency gets an estimate at the latest reference rate.
 */
export function PromptPayDialog({ transfer, receiverName, promptpay, currency, onClose }: Props) {
  const { t, locale } = useI18n()
  const titleId = useId()
  const home = useHomeApprox(currency)
  const isBaht = currency === HOME_CURRENCY
  const units = transfer ? transfer.amount_base / 10 ** exponent(currency) : 0
  const baht = isBaht ? units : home.rate ? Math.round(units * home.rate * 100) / 100 : undefined

  const qr = useQuery({
    queryKey: ['promptpay-qr', promptpay, baht],
    queryFn: () => promptPayQr(promptpay, baht),
    enabled: !!transfer,
    staleTime: Infinity,
  })

  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  // The finished card, shown full size so a long-press saves it to the gallery.
  const [card, setCard] = useState<string | null>(null)
  const amountText = baht !== undefined ? formatMoney(Math.round(baht * 100), HOME_CURRENCY) : ''
  const toText = t('settle.qrTo', { name: receiverName }) + ' · ' + formatPromptPay(promptpay)

  const onSave = async () => {
    if (!qr.data) return
    setSaving(true)
    try {
      const blob = await qrCardImage(qr.data, {
        amount: amountText || t('settle.qrAnyAmount'),
        to: toText,
        note: !isBaht && transfer ? t('settle.qrConverted', { amount: formatMoney(transfer.amount_base, currency) }) : undefined,
      })
      const safeName = receiverName.replace(/[^\p{L}\p{M}\p{N}]+/gu, '-').replace(/^-|-$/g, '') || 'qr'
      const result = await saveImage(blob, `promptpay-${safeName}${baht !== undefined ? '-' + baht.toFixed(2) : ''}.png`)
      if (result === 'long-press') {
        setCard(await blobToDataUrl(blob))
      } else if (result !== 'cancelled') {
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      }
    } finally {
      setSaving(false)
    }
  }

  const close = () => {
    setSaved(false)
    setCard(null)
    onClose()
  }

  return (
    <Dialog open={!!transfer} onClose={close} labelledBy={titleId}>
      {card ? (
        <div className="space-y-3 text-center">
          <h2 id={titleId} className="text-lg font-bold">
            {t('settle.qrLongPressTitle')}
          </h2>
          <p className="text-sm text-slate-600">{t('settle.qrLongPress')}</p>
          {/* No pointer handlers here: the browser's own long-press menu must open. */}
          <img src={card} alt={t('settle.qrAlt', { name: receiverName })} className="mx-auto max-h-[60vh] w-auto rounded-xl ring-1 ring-slate-200" />
          <Button variant="secondary" className="w-full" onClick={() => setCard(null)} data-autofocus>
            <ArrowLeft className="size-4" /> {t('common.back')}
          </Button>
        </div>
      ) : (
      <div className="space-y-4 text-center">
        <div className="flex items-center gap-3 text-left">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-brand-50 text-brand-600">
            <QrCode className="size-5" />
          </span>
          <div>
            <h2 id={titleId} className="text-lg font-bold">
              {t('settle.qrTitle', { name: receiverName })}
            </h2>
            <p className="text-sm text-slate-600">{t('settle.qrHint')}</p>
          </div>
        </div>

        <div className="mx-auto flex aspect-square w-60 items-center justify-center rounded-2xl bg-white p-2 ring-1 ring-slate-200">
          {qr.data ? <img src={qr.data} alt={t('settle.qrAlt', { name: receiverName })} className="size-full" /> : <Spinner />}
        </div>

        <div>
          {baht !== undefined ? (
            <p className="text-3xl font-bold tabular-nums">{amountText}</p>
          ) : (
            <p className="text-sm text-amber-800">{t('settle.qrNoAmount')}</p>
          )}
          {!isBaht && transfer && (
            <p className="text-xs text-slate-500">
              {t('settle.qrConverted', { amount: formatMoney(transfer.amount_base, currency) })}
              {home.rateDate && ` · ${formatShortDate(home.rateDate, locale)}`}
            </p>
          )}
          <p className="mt-1 text-sm text-slate-600">
            {t('settle.qrTo', { name: receiverName })} · <span className="font-mono tabular-nums">{formatPromptPay(promptpay)}</span>
          </p>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <Button variant="secondary" onClick={close}>
            {t('common.close')}
          </Button>
          <Button onClick={onSave} disabled={!qr.data || saving} data-autofocus>
            <Download className="size-4" />
            {saved ? t('settle.qrSaved') : t('settle.qrSave')}
          </Button>
        </div>
      </div>
      )}
    </Dialog>
  )
}
