import { safeGetStorageItem, safeRemoveStorageItem, safeSetStorageItem } from '@/custom/utils/browserStorage'

const RECOVERY_KEY = 'wegoo_chunk_recovery_v2'
const RECOVERY_WINDOW_MS = 60_000
const MAX_RECOVERY_ATTEMPTS = 3

interface RecoveryState {
  buildKey: string
  attempts: number
  updatedAt: number
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return `${error.name} ${error.message}`.trim()
  }
  if (typeof error === 'string') {
    return error
  }
  if (error && typeof error === 'object') {
    const record = error as Record<string, unknown>
    return `${String(record.name || '')} ${String(record.message || '')}`.trim()
  }
  return ''
}

export function isChunkLoadError(error: unknown): boolean {
  const message = errorMessage(error)
  if (!message) return false
  return [
    'Failed to fetch dynamically imported module',
    'Importing a module script failed',
    'error loading dynamically imported module',
    'Loading chunk',
    'Loading CSS chunk',
    'ChunkLoadError',
    'CSS_CHUNK_LOAD_FAILED',
  ].some((needle) => message.toLowerCase().includes(needle.toLowerCase()))
}

function currentBuildKey(): string {
  if (typeof document === 'undefined') {
    return 'unknown'
  }

  const scripts = Array.from(document.scripts)
  const entryScript = scripts.find((script) => /\/assets\/index-[^/]+\.js(?:\?|$)/.test(script.src))
  const moduleScript = scripts.find((script) => script.type === 'module' && script.src)
  return entryScript?.src || moduleScript?.src || 'unknown'
}

function readRecoveryState(): RecoveryState | null {
  const raw = safeGetStorageItem('sessionStorage', RECOVERY_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Partial<RecoveryState>
    if (
      typeof parsed.buildKey === 'string' &&
      typeof parsed.attempts === 'number' &&
      typeof parsed.updatedAt === 'number'
    ) {
      return parsed as RecoveryState
    }
  } catch {
    // Ignore malformed state from older builds.
  }
  return null
}

function writeRecoveryState(state: RecoveryState): void {
  safeSetStorageItem('sessionStorage', RECOVERY_KEY, JSON.stringify(state))
}

function cacheBustedURL(attempts: number): string {
  const url = new URL(window.location.href)
  url.searchParams.set('_app_reload', `${Date.now()}-${attempts}`)
  return url.toString()
}

function showRecoveryFallback(): void {
  const root = document.getElementById('app')
  if (!root) return

  root.innerHTML = `
    <main style="min-height:100vh;display:grid;place-items:center;padding:24px;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f8fafc;color:#0f172a;">
      <section style="max-width:520px;width:100%;padding:28px;border:1px solid #bfdbfe;border-radius:20px;background:#fff;box-shadow:0 20px 60px rgba(15,23,42,.08);">
        <h1 style="margin:0 0 12px;font-size:24px;">页面资源已更新</h1>
        <p style="margin:0 0 18px;line-height:1.7;color:#475569;">浏览器还在使用旧页面资源，导致页面无法继续加载。点击下面按钮会强制重新获取最新页面，不需要关闭浏览器。</p>
        <button id="wegoo-reload-latest" type="button" style="border:0;border-radius:999px;background:#1d4ed8;color:#fff;font-weight:700;padding:10px 18px;cursor:pointer;">重新加载最新版本</button>
      </section>
    </main>
  `

  document.getElementById('wegoo-reload-latest')?.addEventListener('click', () => {
    safeRemoveStorageItem('sessionStorage', RECOVERY_KEY)
    window.location.replace(cacheBustedURL(0))
  })
}

export function recoverFromChunkLoadError(error: unknown, source = 'app'): boolean {
  if (!isChunkLoadError(error) || typeof window === 'undefined') {
    return false
  }

  const now = Date.now()
  const buildKey = currentBuildKey()
  const previous = readRecoveryState()
  const attempts =
    previous && previous.buildKey === buildKey && now - previous.updatedAt < RECOVERY_WINDOW_MS
      ? previous.attempts + 1
      : 1

  if (attempts <= MAX_RECOVERY_ATTEMPTS) {
    writeRecoveryState({ buildKey, attempts, updatedAt: now })
    console.warn(`[${source}] Frontend chunk is stale, reloading latest app version...`)
    window.location.replace(cacheBustedURL(attempts))
    return true
  }

  console.error(`[${source}] Frontend chunk recovery failed after ${attempts - 1} attempts.`, error)
  showRecoveryFallback()
  return true
}

export function clearAppReloadMarker(): void {
  if (typeof window === 'undefined') {
    return
  }

  safeRemoveStorageItem('sessionStorage', RECOVERY_KEY)

  const url = new URL(window.location.href)
  if (!url.searchParams.has('_app_reload')) {
    return
  }
  url.searchParams.delete('_app_reload')
  window.history.replaceState(window.history.state, document.title, `${url.pathname}${url.search}${url.hash}`)
}
