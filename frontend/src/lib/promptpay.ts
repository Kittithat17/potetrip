import generatePayload from 'promptpay-qr'
import QRCode from 'qrcode'

/**
 * Builds a PromptPay QR image (data URL) for a phone number or ID.
 * `baht` fixes the amount in the payer's banking app; omit it to let them type it.
 */
export async function promptPayQr(id: string, baht?: number): Promise<string> {
  const payload = generatePayload(id, baht && baht > 0 ? { amount: Math.round(baht * 100) / 100 } : {})
  return QRCode.toDataURL(payload, { margin: 1, width: 560, errorCorrectionLevel: 'M' })
}

/** Shows a PromptPay ID the way Thai banking apps do: 081-234-5678 or 1-2345-67890-12-3. */
export function formatPromptPay(id: string): string {
  if (id.length === 10) return `${id.slice(0, 3)}-${id.slice(3, 6)}-${id.slice(6)}`
  if (id.length === 13) return `${id[0]}-${id.slice(1, 5)}-${id.slice(5, 10)}-${id.slice(10, 12)}-${id[12]}`
  return id
}

/**
 * Draws a shareable card (QR + amount + receiver) so a saved image still says
 * who to pay and how much. Returns a PNG blob.
 */
export async function qrCardImage(qrDataUrl: string, lines: { amount: string; to: string; note?: string }): Promise<Blob> {
  const img = new Image()
  img.src = qrDataUrl
  await img.decode()

  const W = 720
  const pad = 56
  const qrSize = W - pad * 2
  const H = pad + 64 + qrSize + 40 + 72 + 48 + (lines.note ? 40 : 0) + pad
  const canvas = document.createElement('canvas')
  canvas.width = W
  canvas.height = H
  const ctx = canvas.getContext('2d')!
  const font = (weight: number, size: number) =>
    `${weight} ${size}px "Noto Sans Thai", system-ui, -apple-system, "Segoe UI", sans-serif`

  // Web fonts load lazily; drawing before they arrive can mis-measure text
  // (iOS Safari then misplaces centred lines), so wait for each face first.
  await Promise.all([font(700, 40), font(700, 60), font(400, 30), font(400, 24)].map((f) => document.fonts?.load(f).catch(() => undefined)))

  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, W, H)
  ctx.textBaseline = 'top'
  ctx.textAlign = 'left'

  // Centres by measuring instead of textAlign=center, which iOS Safari gets
  // wrong here, and shrinks lines that would run past the edges.
  const line = (text: string, weight: number, size: number, y: number, color: string) => {
    let s = size
    ctx.font = font(weight, s)
    while (ctx.measureText(text).width > W - pad * 2 && s > 14) {
      s -= 2
      ctx.font = font(weight, s)
    }
    ctx.fillStyle = color
    ctx.fillText(text, Math.max(pad, (W - ctx.measureText(text).width) / 2), y + (size - s) / 2)
  }

  let y = pad
  line('PromptPay', 700, 40, y, '#1e3a8a')
  y += 64
  ctx.drawImage(img, pad, y, qrSize, qrSize)
  y += qrSize + 40
  line(lines.amount, 700, 60, y, '#0f172a')
  y += 72
  line(lines.to, 400, 30, y, '#475569')
  if (lines.note) {
    y += 48
    line(lines.note, 400, 24, y, '#94a3b8')
  }

  return new Promise((resolve, reject) => canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('toBlob failed'))), 'image/png'))
}

export type SaveResult = 'shared' | 'cancelled' | 'downloaded' | 'long-press'

/**
 * Saves an image where people expect it:
 * - phones with a share sheet for files (https only): the sheet, whose
 *   "Save Image" puts it in the photo gallery;
 * - other phones (e.g. plain http while testing): nothing is saved here; the
 *   caller shows the image so a long-press saves it to the gallery;
 * - computers: a normal file download.
 */
export async function saveImage(blob: Blob, filename: string): Promise<SaveResult> {
  const file = new File([blob], filename, { type: blob.type })
  // Desktop browsers can share files too, but there people expect a download.
  const touch = window.matchMedia?.('(pointer: coarse)').matches
  if (touch && navigator.canShare?.({ files: [file] })) {
    try {
      await navigator.share({ files: [file] })
      return 'shared'
    } catch {
      return 'cancelled'
    }
  }
  if (touch) return 'long-press'
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 10_000)
  return 'downloaded'
}

/** Blob -> data URL; long-press "Save to Photos" is most reliable on data URLs. */
export function blobToDataUrl(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader()
    r.onload = () => resolve(String(r.result))
    r.onerror = () => reject(r.error)
    r.readAsDataURL(blob)
  })
}
