import { Link2, Share2 } from 'lucide-react'
import { CopyLinkDialog } from '../../../components/ui/CopyLinkDialog'
import { Button } from '../../../components/ui/ui'
import { useI18n } from '../../../i18n/context'
import { inviteLink } from '../../../lib/invite'
import { useTrip } from '../layout/context'

type Props = { open: boolean; onClose: () => void; onCopied?: () => void }

/** Our own invite sheet, used where the native share sheet isn't available. */
export function InviteDialog({ open, onClose, onCopied }: Props) {
  const { t } = useI18n()
  const { trip } = useTrip()
  // The phone's own share sheet (LINE, Messages, ...), available on https.
  const shareVia = async () => {
    try {
      await navigator.share({ title: trip.name, text: t('trip.shareText', { name: trip.name }), url: inviteLink(trip.invite_code) })
    } catch {
      /* closed the share sheet */
    }
  }
  return (
    <CopyLinkDialog
      open={open}
      onClose={onClose}
      onCopied={onCopied}
      icon={Link2}
      title={t('trip.inviteTitle')}
      hint={t('trip.inviteHint')}
      url={inviteLink(trip.invite_code)}
    >
      <p className="text-center text-xs text-slate-500">
        {t('trip.inviteCode')}: <span className="font-mono font-semibold tracking-widest text-slate-700">{trip.invite_code}</span>
      </p>
      {typeof navigator.share === 'function' && (
        <Button variant="secondary" className="w-full" onClick={shareVia}>
          <Share2 className="size-4" /> {t('trip.shareVia')}
        </Button>
      )}
    </CopyLinkDialog>
  )
}
