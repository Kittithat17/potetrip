import { act, fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { I18nProvider } from '../../i18n/I18nProvider'
import { useConfirm } from './confirm'
import { ConfirmProvider } from './ConfirmProvider'

beforeEach(() => vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'requestAnimationFrame', 'cancelAnimationFrame'] }))
afterEach(() => vi.useRealTimers())

function Probe() {
  const confirm = useConfirm()
  const [result, setResult] = useState('none')
  return (
    <>
      <button onClick={async () => setResult(String(await confirm({ title: 'Delete it?', message: 'Gone for good', confirmLabel: 'Delete' })))}>
        open
      </button>
      <output>{result}</output>
    </>
  )
}

function setup() {
  render(
    <I18nProvider initialLang="en">
      <ConfirmProvider>
        <Probe />
      </ConfirmProvider>
    </I18nProvider>,
  )
  const open = () => {
    fireEvent.click(screen.getByText('open'))
    act(() => vi.advanceTimersByTime(50)) // let the enter animation start
  }
  const settleAnimations = () => act(() => vi.advanceTimersByTime(300))
  return { open, settleAnimations }
}

it('resolves true on confirm and closes after the exit animation', async () => {
  const { open, settleAnimations } = setup()
  open()
  const dialog = screen.getByRole('alertdialog', { name: 'Delete it?' })
  expect(dialog).toHaveAccessibleDescription('Gone for good')
  expect(document.body.style.overflow).toBe('hidden')

  await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Delete' })))
  expect(screen.getByText('true')).toBeInTheDocument()
  // Still on screen while it animates out, then removed.
  expect(screen.queryByRole('alertdialog')).toBeInTheDocument()
  settleAnimations()
  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  expect(document.body.style.overflow).toBe('')
})

it('resolves false on Cancel and on Escape', async () => {
  const { open, settleAnimations } = setup()
  open()
  await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Cancel' })))
  expect(screen.getByText('false')).toBeInTheDocument()
  settleAnimations()

  open()
  expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus() // safe default for destructive actions
  await act(async () => fireEvent.keyDown(document, { key: 'Escape' }))
  settleAnimations()
  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  expect(screen.getByText('false')).toBeInTheDocument()
})
