const CODE = /^[2-9A-HJKMNP-Z]{10}$/

/** Accepts a bare invite code or a full invite link and returns the code. */
export function parseInviteInput(input: string): string | null {
  const s = input.trim()
  if (!s) return null
  const fromLink = /\/t\/([^/?#\s]+)/i.exec(s)
  const code = (fromLink ? fromLink[1] : s).toUpperCase()
  return CODE.test(code) ? code : null
}

export function inviteLink(code: string): string {
  return `${window.location.origin}/t/${code}`
}
