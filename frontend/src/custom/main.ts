import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from '@/custom/App.vue'
import router from '@/custom/router'
import i18n, { initI18n } from '@/i18n'
import { useAppStore } from '@/stores/app'
import { DEFAULT_SITE_NAME } from '@/utils/branding'
import { clearAppReloadMarker, recoverFromChunkLoadError } from '@/custom/utils/chunkRecovery'
import { initTheme } from '@/custom/utils/theme'
import '@/custom/styles/style.css'

if (typeof window !== 'undefined') {
  window.addEventListener('unhandledrejection', (event) => {
    if (recoverFromChunkLoadError(event.reason, 'unhandledrejection')) {
      event.preventDefault()
    }
  })
  window.addEventListener('error', (event) => {
    recoverFromChunkLoadError(event.error || event.message, 'window.error')
  })
}

function showBootstrapError(error: unknown) {
  console.error('Failed to bootstrap app:', error)
  const appRoot = document.getElementById('app')
  if (!appRoot) {
    return
  }

  appRoot.innerHTML = `
    <main style="min-height:100vh;display:grid;place-items:center;padding:24px;box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f8fafc;color:#0f172a;">
      <section style="max-width:520px;width:100%;padding:28px;border:1px solid #fecaca;border-radius:20px;background:#fff;box-shadow:0 20px 60px rgba(15,23,42,.08);">
        <h1 style="margin:0 0 12px;font-size:24px;">页面加载失败</h1>
        <p style="margin:0 0 12px;line-height:1.7;color:#475569;">页面脚本启动失败，通常是缓存异常、网络中断或浏览器扩展拦截导致。请强制刷新页面，或清除浏览器缓存后重试。</p>
        <p style="margin:0;"><a href="/home" style="color:#1d4ed8;font-weight:700;margin-right:12px;">返回首页</a><a href="/login" style="color:#1d4ed8;font-weight:700;">重新登录</a></p>
      </section>
    </main>
  `
}

async function bootstrap() {
  // Apply theme class globally before app mount to keep all routes consistent.
  initTheme()

  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)

  // Initialize settings from injected config BEFORE mounting (prevents flash)
  // This must happen after pinia is installed but before router and i18n
  const appStore = useAppStore()
  appStore.initFromInjectedConfig()

  // Set document title immediately after config is loaded
  document.title = `${appStore.siteName || DEFAULT_SITE_NAME} - AI API Gateway`

  await initI18n()

  app.use(router)
  app.use(i18n)

  // 等待路由器完成初始导航后再挂载，避免竞态条件导致的空白渲染
  await router.isReady()
  app.mount('#app')
  clearAppReloadMarker()
}

bootstrap().catch((error) => {
  if (!recoverFromChunkLoadError(error, 'bootstrap')) {
    showBootstrapError(error)
  }
})
