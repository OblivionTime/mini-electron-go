// Minimal Wails-like bridge over MiniBlink's window.mbQuery / window.__mbEmit.
// JS -> Go: invoke("Struct.Method", ...args) => Promise
// Go -> JS: on("event", handler)
let seq = 0

export function invoke(method, ...args) {
  return new Promise((resolve, reject) => {
    if (typeof window.mbQuery !== 'function') {
      reject(new Error('window.mbQuery not found (not running inside MiniBlink?)'))
      return
    }
    const id = ++seq
    let settled = false
    window.mbQuery(id, JSON.stringify({ id, method, args }), (_customMsg, response) => {
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
