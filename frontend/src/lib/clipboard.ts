/**
 * Copies text, returning whether it worked. The Clipboard API only exists on
 * secure pages (https or localhost); on plain http, e.g. a phone testing over
 * Wi-Fi, fall back to selecting the text and running the legacy copy command,
 * which browsers (iOS Safari included) still honour during a tap.
 * `container` should be inside any open dialog so its focus trap isn't upset.
 */
export async function copyText(text: string, container: HTMLElement = document.body): Promise<boolean> {
  if (window.isSecureContext && navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      /* fall through to the legacy path */
    }
  }
  const span = document.createElement('span')
  span.textContent = text
  Object.assign(span.style, {
    position: 'fixed',
    top: '0',
    left: '0',
    whiteSpace: 'pre',
    userSelect: 'text',
    webkitUserSelect: 'text',
    opacity: '0',
    pointerEvents: 'none',
  })
  container.appendChild(span)
  const selection = window.getSelection()
  let ok = false
  try {
    const range = document.createRange()
    range.selectNodeContents(span)
    selection?.removeAllRanges()
    selection?.addRange(range)
    ok = document.execCommand('copy')
  } catch {
    ok = false
  } finally {
    selection?.removeAllRanges()
    span.remove()
  }
  return ok
}
