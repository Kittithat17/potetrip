import {
  browserSupportsWebAuthn,
  startAuthentication,
  startRegistration,
  type PublicKeyCredentialCreationOptionsJSON,
  type PublicKeyCredentialRequestOptionsJSON,
} from '@simplewebauthn/browser'
import * as api from '../../api/endpoints'

// Passkeys sign in with the device's Face ID, fingerprint or PIN. Nothing
// goes through Google, so they work where Google is blocked (mainland China).
// WebAuthn needs a secure page: https, or http://localhost in development.

export const passkeysSupported = () => window.isSecureContext && browserSupportsWebAuthn()

export async function signInWithPasskey() {
  const options = (await api.passkeyLoginBegin()) as PublicKeyCredentialRequestOptionsJSON
  return api.passkeyLoginFinish(await startAuthentication({ optionsJSON: options }))
}

export async function signUpWithPasskey(name: string) {
  const options = (await api.passkeySignupBegin(name)) as PublicKeyCredentialCreationOptionsJSON
  return api.passkeySignupFinish(await startRegistration({ optionsJSON: options }))
}

/** Adds a passkey on this device to the signed-in account. */
export async function addPasskey() {
  const options = (await api.passkeyRegisterBegin()) as PublicKeyCredentialCreationOptionsJSON
  return api.passkeyRegisterFinish(await startRegistration({ optionsJSON: options }))
}

/** The person closed the Face ID / passkey prompt (or it timed out). */
export function isPasskeyCancel(err: unknown): boolean {
  return err instanceof Error && (err.name === 'NotAllowedError' || err.name === 'AbortError' || ('code' in err && err.code === 'ERROR_CEREMONY_ABORTED'))
}
