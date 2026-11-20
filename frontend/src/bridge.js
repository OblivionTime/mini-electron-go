// Minimal Wails-like bridge over MiniBlink's window.mbQuery / window.__mbEmit.
// JS -> Go: invoke("Struct.Method", ...args) => Promise
// Go -> JS: on("event", handler)
let seq = 0
const ready = new Promise((resolve) => {
  if (window.__mbToken) resolve(window.__mbToken)
  else window.addEventListener('mb-ready', () => resolve(window.__mbToken), { once: true })
})

export async function invoke(method, ...args) {
  if (typeof window.mbQuery !== 'function') {
    throw new Error('window.mbQuery not found (not running inside MiniBlink?)')
  }
  const token = await Promise.race([ready, new Promise((resolve) => setTimeout(resolve, 5000))])
  if (!token) throw new Error('bridge unavailable for this document')
  return new Promise((resolve, reject) => {
    const id = ++seq
    let settled = false
    window.mbQuery(id, JSON.stringify({ id, method, args, token }), (_customMsg, response) => {
      if (settled) return
      settled = true
      let res
      try {
        res = JSON.parse(response)
      } catch (e) {
        reject(new Error('invalid response: ' + response))
        return
      }
      if (res.error) reject(new Error(res.error))
      else resolve(res.result)
    })
  })
}

// Global entry point used by the auto-generated binding wrappers.
window.__mbInvoke = invoke

const listeners = new Map()

window.__mbEmit = (event, data) => {
  const set = listeners.get(event)
  if (!set) return
  for (const fn of set) {
    try {
      fn(data)
    } catch (e) {
      console.error('[mb-emit]', event, e)
    }
  }
}

export function on(event, fn) {
  if (!listeners.has(event)) listeners.set(event, new Set())
  listeners.get(event).add(fn)
  return () => listeners.get(event)?.delete(fn)
}
