import { useMutation } from '@tanstack/react-query'
import { ChevronLeft } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { createTrip } from '../api/endpoints'
import { AppFrame, BlueHeader, Sheet } from '../components/shell'
import { Button, ErrorText, Field, inputClass, TextInput } from '../components/ui'
import { useI18n } from '../i18n/context'
import { currencyName } from '../lib/currencyInfo'
import { setIdentity } from '../lib/identity'
import { CURRENCIES } from '../lib/money'

export function NewTripPage() {
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const [tripName, setTripName] = useState('')
  const [currency, setCurrency] = useState('THB')
  const [myName, setMyName] = useState('')

  const create = useMutation({
    mutationFn: async () => {
      const { trip, member, token } = await createTrip({ name: tripName, base_currency: currency, creator_name: myName })
      setIdentity(trip.id, { token, memberId: member.id })
      return trip
    },
    onSuccess: (trip) => navigate(`/t/${trip.invite_code}`, { replace: true }),
  })

  const onCreate = (e: FormEvent) => {
    e.preventDefault()
    create.mutate()
  }

  return (
    <AppFrame>
      <BlueHeader className="pb-16">
        <div className="flex items-center gap-1">
          <Link to="/" aria-label={t('common.back')} className="-ml-2 rounded-full p-1 hover:bg-white/15">
            <ChevronLeft className="size-7" />
          </Link>
          <h1 className="text-xl font-bold">{t('newTrip.title')}</h1>
        </div>
        <p className="mt-1 pl-8 text-sm text-brand-100">{t('newTrip.subtitle')}</p>
      </BlueHeader>
      <main className="mx-auto -mt-10 max-w-md px-4">
        <Sheet>
          <form onSubmit={onCreate} className="space-y-4">
            <Field label={t('newTrip.tripName')}>
              <TextInput value={tripName} onChange={(e) => setTripName(e.target.value)} placeholder={t('newTrip.tripNamePlaceholder')} required maxLength={100} />
            </Field>
            <Field label={t('newTrip.yourName')}>
              <TextInput value={myName} onChange={(e) => setMyName(e.target.value)} placeholder={t('newTrip.yourNamePlaceholder')} required maxLength={50} />
            </Field>
            <Field label={t('newTrip.currency')} hint={t('newTrip.currencyHint')}>
              <select value={currency} onChange={(e) => setCurrency(e.target.value)} className={inputClass}>
                {CURRENCIES.map((c) => (
                  <option key={c} value={c}>
                    {c} · {currencyName(c, locale)}
                  </option>
                ))}
              </select>
            </Field>
            <ErrorText error={create.error} />
            <Button type="submit" className="w-full" disabled={create.isPending}>
              {create.isPending ? t('newTrip.creating') : t('newTrip.create')}
            </Button>
          </form>
        </Sheet>
      </main>
    </AppFrame>
  )
}
