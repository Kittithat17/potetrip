import generatePayload from 'promptpay-qr'
import { expect, it } from 'vitest'
import { formatPromptPay, promptPayQr } from './promptpay'

it('encodes the phone number and amount into the payload', () => {
  const p = generatePayload('0812345678', { amount: 351.75 })
  expect(p).toContain('0066812345678') // phone in international form
  expect(p).toContain('5406351.75') // tag 54, length 6, amount
  expect(p).toContain('5303764') // THB
})

it('renders a QR image', async () => {
  expect(await promptPayQr('0812345678', 100)).toMatch(/^data:image\/png;base64,/)
})

it('formats IDs', () => {
  expect(formatPromptPay('0812345678')).toBe('081-234-5678')
  expect(formatPromptPay('1234567890123')).toBe('1-2345-67890-12-3')
})
