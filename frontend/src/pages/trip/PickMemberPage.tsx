import { useState, type FormEvent } from 'react'
import { useAddMember } from '../../api/hooks'
import type { Member, Trip } from '../../api/types'
import { Avatar, Button, Card, ErrorText, Field, Page, PageHeader, TextInput } from '../../components/ui'

type Props = { trip: Trip; members: Member[]; onPicked: (memberId: string) => void }

export function PickMemberPage({ trip, members, onPicked }: Props) {
  const [name, setName] = useState('')
  const [promptpay, setPromptpay] = useState('')
  const add = useAddMember(trip.id)

  const onAdd = (e: FormEvent) => {
    e.preventDefault()
    add.mutate({ name, promptpay: promptpay || null }, { onSuccess: (m) => onPicked(m.id) })
  }

  return (
    <Page>
      <PageHeader title={trip.name} back="/" />
      <div className="space-y-4">
        <p className="text-slate-600">คุณคือใครในทริปนี้?</p>

        {members.length > 0 && (
          <ul className="space-y-2">
            {members.map((m) => (
              <li key={m.id}>
                <button
                  onClick={() => onPicked(m.id)}
                  className="flex w-full items-center gap-3 rounded-2xl bg-white p-3 text-left shadow-sm ring-1 ring-slate-200 hover:ring-teal-600"
                >
                  <Avatar name={m.name} />
                  <span className="font-medium">{m.name}</span>
                </button>
              </li>
            ))}
          </ul>
        )}

        <Card>
          <h2 className="mb-3 font-bold">{members.length ? 'ไม่มีชื่อคุณ? เพิ่มตัวเอง' : 'เพิ่มชื่อของคุณ'}</h2>
          <form onSubmit={onAdd} className="space-y-3">
            <Field label="ชื่อ">
              <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="ชื่อเล่น" required maxLength={50} />
            </Field>
            <Field label="พร้อมเพย์ (ไม่บังคับ)" hint="เบอร์โทรหรือเลขบัตร ไว้ให้เพื่อนสแกนโอนคืน">
              <TextInput value={promptpay} onChange={(e) => setPromptpay(e.target.value)} inputMode="numeric" placeholder="08x-xxx-xxxx" />
            </Field>
            <ErrorText error={add.error} />
            <Button type="submit" className="w-full" disabled={add.isPending}>
              {add.isPending ? 'กำลังเพิ่ม…' : 'เข้าร่วมทริป'}
            </Button>
          </form>
        </Card>
      </div>
    </Page>
  )
}
