import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

function injectPublicSettings(): Plugin {
  return {
    name: 'inject-public-settings-for-visual-check',
    transformIndexHtml: {
      order: 'pre',
      async handler(html) {
        try {
          const response = await fetch('http://127.0.0.1:8080/api/v1/settings/public', {
            signal: AbortSignal.timeout(2000),
          })
          if (response.ok) {
            const body = await response.json() as { code?: number; data?: Record<string, unknown> }
            if (body.code === 0 && body.data) {
              const encoded = JSON.stringify(body.data).replace(/</g, '\\u003c')
              const siteName = typeof body.data.site_name === 'string' ? body.data.site_name.trim() : ''
              const title = siteName ? `${siteName} - AI API Gateway` : "Wegoo's API - AI API Gateway"
              return html
                .replace(/<title>[^<]*<\/title>/i, `<title>${title}</title>`)
                .replace('</head>', `<script>window.__APP_CONFIG__=${encoded};</script>\n</head>`)
            }
          }
        } catch {
          // The app falls back to its normal async settings request.
        }
        return html
      },
    },
  }
}

export default defineConfig({
  plugins: [vue(), injectPublicSettings()],
  resolve: {
    alias: [
      { find: /^@\/types\/payment$/, replacement: resolve(__dirname, 'src/custom/types/payment.ts') },
      { find: /^@\/types$/, replacement: resolve(__dirname, 'src/custom/types/index-fork.ts') },
      { find: '@', replacement: resolve(__dirname, 'src') },
      { find: 'vue-i18n', replacement: 'vue-i18n/dist/vue-i18n.runtime.esm-bundler.js' },
    ],
  },
  define: {
    __INTLIFY_JIT_COMPILATION__: true,
  },
  server: {
    host: '127.0.0.1',
    port: 4173,
    proxy: {
      '/api': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '/v1': { target: 'http://127.0.0.1:8080', changeOrigin: true },
      '^/setup/(status|test-db|test-redis|install)': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
