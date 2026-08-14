const activeModalLocks = new Set<symbol>()

function syncBodyLock() {
  if (typeof document === 'undefined') return
  document.body.classList.toggle('modal-open', activeModalLocks.size > 0)
}

export function setBodyModalLock(token: symbol, locked: boolean) {
  if (locked) {
    activeModalLocks.add(token)
  } else {
    activeModalLocks.delete(token)
  }
  syncBodyLock()
}

export function releaseBodyModalLock(token: symbol) {
  activeModalLocks.delete(token)
  syncBodyLock()
}
