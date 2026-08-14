type StorageKind = 'localStorage' | 'sessionStorage'
type SafeStorageFacade = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

function getStorage(kind: StorageKind): Storage | null {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const storage = window[kind]
    const probeKey = `__wegoo_storage_probe_${kind}__`
    storage.setItem(probeKey, '1')
    storage.removeItem(probeKey)
    return storage
  } catch {
    return null
  }
}

export function safeGetStorageItem(kind: StorageKind, key: string): string | null {
  try {
    return getStorage(kind)?.getItem(key) ?? null
  } catch {
    return null
  }
}

export function safeSetStorageItem(kind: StorageKind, key: string, value: string): boolean {
  try {
    const storage = getStorage(kind)
    if (!storage) {
      return false
    }
    storage.setItem(key, value)
    return true
  } catch {
    return false
  }
}

export function safeRemoveStorageItem(kind: StorageKind, key: string): void {
  try {
    getStorage(kind)?.removeItem(key)
  } catch {
    // Email webviews may expose Storage but throw when it is touched.
  }
}

export function createSafeStorageFacade(kind: StorageKind): SafeStorageFacade {
  return {
    getItem: (key) => safeGetStorageItem(kind, key),
    setItem: (key, value) => {
      safeSetStorageItem(kind, key, value)
    },
    removeItem: (key) => {
      safeRemoveStorageItem(kind, key)
    },
  }
}
