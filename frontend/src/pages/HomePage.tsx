import { useMutation } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { addMember, createTrip } from '../api/endpoints'
import { HealthBadge } from '../components/HealthBadge'
import { Avatar, Button, Card, ErrorText, Field, inputClass, Page, TextInput } from '../components/ui'
import { setMyMemberId } from '../lib/identity'
import { parseInviteInput } from '../lib/invite'
import { forgetTrip, getRecentTrips, type RecentTrip } from '../lib/recentTrips'
import { CURRENCIES } from '../lib/money'

export function HomePage() {
  const navigate = useNavigate()
  const [tripName, setTripName] = useState('')
  const [currency, setCurrency] = useState('THB')
  const [myName, setMyName] = useState('')
  const [invite, setInvite] = useState('')
  const [inviteError, setInviteError] = useState('')
  const [recent, setRecent] = useState<RecentTrip[]>(getRecentTrips)

  const create = useMutation({
    mutationFn: async () => {
      const trip = await createTrip({ name: tripName, base_currency: currency })
      const me = await addMember(trip.id, { name: myName })
      setMyMemberId(trip.id, me.id)
      return trip
    },
    onSuccess: (trip) => navigate(`/t/${trip.invite_code}`),
  })

  const onCreate = (e: FormEvent) => {
    e.preventDefault()
    create.mutate()
  }

  const onJoin = (e: FormEvent) => {
    e.preventDefault()
    const code = parseInviteInput(invite)
    if (!code) {
      setInviteError('รหัสหรือลิงก์เชิญไม่ถูกต้อง')
      return
    }
    navigate(`/t/${code}`)
  }

  return (
    <Page>
      <header className="pt-10 pb-6">
        <h1 className="text-3xl font-bold text-teal-800">Travel Split</h1>
        <p className="mt-1 text-slate-600">หารค่าใช้จ่ายทริปกับเพื่อน ไม่ต้องจดเอง</p>
      </header>

      <div className="space-y-4">
        {recent.length > 0 && (
          <Card className="px-2">
            <h2 className="mb-1 px-2 text-lg font-bold">ทริปของฉัน</h2>
            <ul className="divide-y divide-slate-100">
              {recent.map((t) => (
                <li key={t.id} className="flex items-center">
                  <Link to={`/t/${t.code}`} className="flex min-w-0 flex-1 items-center gap-3 rounded-xl px-2 py-3 hover:bg-slate-50">
                    <Avatar name={t.name} />
                    <span className="min-w-0">
                      <span className="block truncate font-medium">{t.name}</span>
                      <span className="block truncate text-xs text-slate-500">
                        {t.myName ? `คุณคือ ${t.myName} · ` : ''}
                        {t.code}
                      </span>
                    </span>
                  </Link>
                  <button
                    onClick={() => {
                      if (!confirm(`ซ่อน "${t.name}" จากรายการนี้? (ข้อมูลทริปไม่ถูกลบ เข้าใหม่ได้ด้วยลิงก์เชิญ)`)) return
                      forgetTrip(t.id)
                      setRecent(getRecentTrips())
                    }}
                    className="rounded-lg px-3 py-2 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
                    aria-label={`ซ่อน ${t.name}`}
                  >
                    ✕
                  </button>
                </li>
              ))}
            </ul>
          </Card>
        )}

        <Card>
          <h2 className="mb-3 text-lg font-bold">สร้างทริปใหม่</h2>
          <form onSubmit={onCreate} className="space-y-3">
            <Field label="ชื่อทริป">
              <TextInput value={tripName} onChange={(e) => setTripName(e.target.value)} placeholder="เช่น โตเกียว พ.ย. 69" required maxLength={100} />
            </Field>
            <Field label="ชื่อของคุณ">
              <TextInput value={myName} onChange={(e) => setMyName(e.target.value)} placeholder="ชื่อเล่น" required maxLength={50} />
            </Field>
            <Field label="สกุลเงินหลัก" hint="ใช้สรุปยอดว่าใครต้องโอนให้ใคร">
              <select value={currency} onChange={(e) => setCurrency(e.target.value)} className={inputClass}>
                {CURRENCIES.map((c) => (
                  <option key={c}>{c}</option>
                ))}
              </select>
            </Field>
            <ErrorText error={create.error} />
            <Button type="submit" className="w-full" disabled={create.isPending}>
              {create.isPending ? 'กำลังสร้าง…' : 'สร้างทริป'}
            </Button>
          </form>
        </Card>

        <Card>
          <h2 className="mb-3 text-lg font-bold">เข้าทริปของเพื่อน</h2>
          <form onSubmit={onJoin} className="space-y-3">
            <Field label="ลิงก์หรือรหัสเชิญ">
              <TextInput
                value={invite}
                onChange={(e) => {
                  setInvite(e.target.value)
                  setInviteError('')
                }}
                placeholder="วางลิงก์ หรือพิมพ์รหัส 10 ตัว"
                autoCapitalize="characters"
                required
              />
            </Field>
            {inviteError && <ErrorText error={inviteError} />}
            <Button type="submit" variant="secondary" className="w-full">
              เข้าทริป
            </Button>
          </form>
        </Card>

        <div className="text-center">
          <HealthBadge />
        </div>
      </div>
    </Page>
  )
}
