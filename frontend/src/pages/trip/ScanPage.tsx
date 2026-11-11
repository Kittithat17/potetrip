import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { ApiError } from '../../api/client'
import { uploadSlip } from '../../api/endpoints'
import type { SlipOutcome } from '../../api/types'
import { Button, Card, Page, PageHeader } from '../../components/ui'
import { MAX_UPLOAD_BYTES, mapLimit, resizeImage } from '../../lib/image'
import { formatMoney } from '../../lib/money'
import { useTrip } from './context'
import { ExpenseForm, type ExpenseDraft } from './ExpenseFormPage'

type Status = 'waiting' | 'resizing' | 'reading' | 'parsed' | 'failed'
type Item = { key: string; file: File; preview: string; status: Status; outcome?: SlipOutcome; error?: string }
type Decision = 'saved' | 'skipped'

const UPLOAD_CONCURRENCY = 3

export function ScanPage() {
  const { trip, me, base } = useTrip()
  const navigate = useNavigate()
  const [items, setItems] = useState<Item[]>([])
  const [stage, setStage] = useState<'pick' | 'review' | 'done'>('pick')
  const [index, setIndex] = useState(0)
  const [decisions, setDecisions] = useState<Record<string, Decision>>({})
  const inputRef = useRef<HTMLInputElement>(null)

  // Object URLs hold the image in memory; release them when leaving the page.
  const previews = useRef<string[]>([])
  useEffect(() => () => previews.current.forEach((u) => URL.revokeObjectURL(u)), [])

  const update = (key: string, patch: Partial<Item>) =>
    setItems((prev) => prev.map((it) => (it.key === key ? { ...it, ...patch } : it)))

  const onPick = async (e: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? [])
    e.target.value = ''
    if (!files.length) return
    const added: Item[] = files.map((file, i) => {
      const preview = URL.createObjectURL(file)
      previews.current.push(preview)
      return { key: `${Date.now()}-${i}-${file.name}`, file, preview, status: 'waiting' }
    })
    setItems((prev) => [...prev, ...added])

    await mapLimit(added, UPLOAD_CONCURRENCY, async (it) => {
      update(it.key, { status: 'resizing' })
      const blob = await resizeImage(it.file)
      if (blob.size > MAX_UPLOAD_BYTES) {
        update(it.key, { status: 'failed', error: 'ไฟล์ใหญ่เกิน 8MB' })
        return
      }
      update(it.key, { status: 'reading' })
      try {
        const name = blob === it.file ? it.file.name : it.file.name.replace(/\.\w+$/, '') + '.jpg'
        const outcome = await uploadSlip(trip.id, me.id, blob, name)
        update(it.key, { status: outcome.status, outcome, error: outcome.error })
      } catch (err) {
        update(it.key, { status: 'failed', error: err instanceof ApiError ? err.message : 'อัปโหลดไม่สำเร็จ' })
      }
    })
  }

  const busy = items.some((it) => it.status === 'waiting' || it.status === 'resizing' || it.status === 'reading')
  const reviewable = items.filter((it) => it.status === 'parsed' || it.status === 'failed')

  const decide = (key: string, d: Decision) => {
    setDecisions((prev) => ({ ...prev, [key]: d }))
    if (index + 1 < reviewable.length) setIndex(index + 1)
    else setStage('done')
    window.scrollTo(0, 0)
  }

  if (stage === 'review') {
    const it = reviewable[index]
    return (
      <ExpenseForm
        key={it.key}
        heading={`ตรวจสอบสลิป ${index + 1}/${reviewable.length}`}
        draft={draftFrom(it.outcome, trip.base_currency)}
        slipId={it.outcome?.slip_id ?? undefined}
        top={<SlipSummary item={it} baseCurrency={trip.base_currency} />}
        onSaved={() => decide(it.key, 'saved')}
        extraActions={
          <Button type="button" variant="ghost" className="w-full" onClick={() => decide(it.key, 'skipped')}>
            ข้าม ไม่บันทึกรายการนี้
          </Button>
        }
      />
    )
  }

  if (stage === 'done') {
    const saved = Object.values(decisions).filter((d) => d === 'saved').length
    return (
      <Page>
        <PageHeader title="สแกนสลิป" back={base} />
        <Card className="space-y-4 text-center">
          <p className="text-4xl">✅</p>
          <p className="text-lg font-medium">บันทึกแล้ว {saved} จาก {reviewable.length} รายการ</p>
          <Button className="w-full" onClick={() => navigate(base)}>
            กลับหน้าทริป
          </Button>
        </Card>
      </Page>
    )
  }

  return (
    <Page>
      <PageHeader title="สแกนสลิป" back={base} />
      <div className="space-y-4">
        <Card className="space-y-3 text-center">
          <p className="text-slate-600">เลือกรูปสลิปโอนเงินหรือใบเสร็จได้ทีละหลายรูป AI จะอ่านยอดเงินและวันที่ให้ คุณตรวจและแก้ได้ก่อนบันทึกทุกครั้ง</p>
          <input ref={inputRef} type="file" accept="image/*" multiple onChange={onPick} className="hidden" />
          <Button className="w-full" onClick={() => inputRef.current?.click()}>
            📷 เลือกรูป
          </Button>
          <p className="text-xs text-slate-500">รูปจะถูกส่งไปให้ AI อ่านแล้วลบทิ้งทันที ระบบไม่เก็บรูปสลิปไว้</p>
        </Card>

        {items.length > 0 && (
          <Card className="px-2">
            <ul className="divide-y divide-slate-100">
              {items.map((it) => (
                <li key={it.key} className="flex items-center gap-3 px-2 py-2">
                  <img src={it.preview} alt="" className="size-14 shrink-0 rounded-lg bg-slate-100 object-cover" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm">{it.outcome?.suggestion?.title ?? it.file.name}</p>
                    <StatusLine item={it} baseCurrency={trip.base_currency} />
                  </div>
                </li>
              ))}
            </ul>
          </Card>
        )}

        {items.length > 0 && (
          <Button className="w-full" disabled={busy || reviewable.length === 0} onClick={() => (setIndex(0), setStage('review'))}>
            {busy ? 'กำลังอ่านสลิป…' : `ตรวจสอบ ${reviewable.length} รายการ`}
          </Button>
        )}
        <p className="text-center text-sm">
          <Link to={`${base}/expenses/new`} className="text-teal-700 underline">
            หรือกรอกรายจ่ายเอง
          </Link>
        </p>
      </div>
    </Page>
  )
}

function StatusLine({ item, baseCurrency }: { item: Item; baseCurrency: string }) {
  const sug = item.outcome?.suggestion
  switch (item.status) {
    case 'waiting':
      return <p className="text-xs text-slate-500">รอคิว…</p>
    case 'resizing':
      return <p className="text-xs text-slate-500">กำลังย่อรูป…</p>
    case 'reading':
      return (
        <p className="flex items-center gap-1 text-xs text-teal-700">
          <span className="size-3 animate-spin rounded-full border-2 border-teal-600 border-t-transparent" /> AI กำลังอ่าน…
        </p>
      )
    case 'failed':
      return <p className="text-xs text-red-700">{item.error ?? 'อ่านไม่สำเร็จ'} · กรอกเองได้ตอนตรวจสอบ</p>
    case 'parsed':
      return (
        <p className="text-xs text-emerald-700">
          อ่านได้ ✓{' '}
          {sug?.amount != null && sug.currency ? formatMoney(sug.amount, sug.currency) : 'ไม่พบยอดเงิน'}
          {sug?.currency && sug.currency !== baseCurrency && <span className="text-amber-700"> · สกุล {sug.currency}</span>}
          {sug?.low_confidence && <span className="text-amber-700"> · ไม่ค่อยแน่ใจ</span>}
        </p>
      )
  }
}

function SlipSummary({ item, baseCurrency }: { item: Item; baseCurrency: string }) {
  const r = item.outcome?.result
  const sug = item.outcome?.suggestion
  const foreign = sug?.currency && sug.currency !== baseCurrency
  return (
    <div className="mb-4 space-y-2">
      <div className="flex gap-3 rounded-2xl bg-white p-3 ring-1 ring-slate-200">
        <a href={item.preview} target="_blank" rel="noreferrer" className="shrink-0">
          <img src={item.preview} alt="รูปสลิป" className="h-28 w-20 rounded-lg bg-slate-100 object-cover" />
        </a>
        <dl className="min-w-0 flex-1 space-y-0.5 text-xs text-slate-600">
          {r ? (
            <>
              <Row k="ประเภท" v={{ transfer_slip: 'สลิปโอนเงิน', receipt: 'ใบเสร็จ', unknown: 'ไม่แน่ใจ' }[r.kind]} />
              <Row k="ยอด" v={r.amount && `${r.amount} ${r.currency}`} />
              <Row k="ร้าน" v={r.merchant} />
              <Row k="จาก" v={r.from_name} />
              <Row k="ถึง" v={r.to_name} />
              <Row k="เลขอ้างอิง" v={r.reference} />
              <Row k="ความมั่นใจ" v={`${Math.round(r.confidence * 100)}%`} />
            </>
          ) : (
            <p className="text-red-700">{item.error ?? 'AI อ่านสลิปนี้ไม่ได้'} กรอกข้อมูลเองได้เลย</p>
          )}
        </dl>
      </div>
      {foreign && (
        <p className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">
          สลิปนี้เป็นสกุล {sug.currency} แต่ตอนนี้ยังรองรับเฉพาะ {baseCurrency} กรุณากรอกยอดเป็น {baseCurrency} เอง
        </p>
      )}
      {sug?.low_confidence && !foreign && (
        <p className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">AI ไม่ค่อยมั่นใจกับสลิปนี้ ตรวจยอดเงินและวันที่ให้ดีก่อนบันทึก</p>
      )}
    </div>
  )
}

function Row({ k, v }: { k: string; v?: string }) {
  if (!v) return null
  return (
    <div className="flex gap-2">
      <dt className="shrink-0 text-slate-400">{k}</dt>
      <dd className="truncate">{v}</dd>
    </div>
  )
}

function draftFrom(o: SlipOutcome | undefined, baseCurrency: string): ExpenseDraft {
  const s = o?.suggestion
  if (!s) return {}
  return {
    title: s.title,
    amount: s.currency === baseCurrency ? s.amount : null,
    spentAt: s.spent_at,
    category: s.category,
  }
}
