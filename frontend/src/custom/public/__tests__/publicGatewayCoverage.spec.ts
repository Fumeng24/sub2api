import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const srcRoot = resolve(__dirname, '../../..')

const readSource = (file: string) => readFileSync(resolve(srcRoot, file), 'utf8')

const publicShellViews = [
  'custom/public/ModelsView.vue',
  'custom/public/DocsView.vue',
  'custom/public/StatusView.vue',
  'custom/public/EnterpriseView.vue',
]

describe('public Gateway visual coverage', () => {
  it('keeps the homepage in the developer gateway shell', () => {
    const source = readSource('custom/home/WegooHomeView.vue')

    expect(source).toContain('gateway-home')
    expect(source).toContain('gateway-nav')
    expect(source).toContain('gateway-hero')
    expect(source).toContain('gateway-code-window')
    expect(source).toContain('AI Gateway · Multi-model API')
    expect(source).toContain("import DOMPurify from 'dompurify'")
    expect(source).toContain('v-html="sanitizedHomeContent"')
    expect(source).toContain('sandbox="allow-scripts allow-same-origin allow-forms allow-popups')
  })

  it.each(publicShellViews)('%s keeps the public Gateway shell contract', (file) => {
    const source = readSource(file)

    expect(source).toContain('public-gateway-shell')
    expect(source).toContain('<PublicGatewayHeader />')
    expect(source).toContain('public-gateway-hero')
    expect(source).toContain('public-gateway-panel')
    expect(source).toContain("import './publicGateway.css'")
    expect(source).toContain("from './PublicGatewayHeader.vue'")
  })

  it('keeps the public header focused on the gateway primary routes', () => {
    const source = readSource('custom/public/PublicGatewayHeader.vue')

    for (const path of ['/home', '/pricing', '/docs', '/status', '/enterprise']) {
      expect(source).toMatch(new RegExp(`to=["']${path}["']`))
    }
    expect(source).toContain('public-gateway-nav')
    expect(source).toContain('public-gateway-primary')
    expect(source).toContain('public-gateway-secondary')
    expect(source).toContain("isDark ? 'sun' : 'moon'")
  })

  it('registers every public Gateway page in the active router', () => {
    const source = readSource('router/index.ts')

    const routeComponents = {
      '/pricing': 'ModelsView.vue',
      '/docs': 'DocsView.vue',
      '/status': 'StatusView.vue',
      '/enterprise': 'EnterpriseView.vue',
    }

    for (const [path, component] of Object.entries(routeComponents)) {
      expect(source).toContain(`path: '${path}'`)
      expect(source).toContain(`component: () => import('@/custom/public/${component}')`)
    }
  })

  it('uses the official console palette without legacy marketing gradients', () => {
    const source = readSource('custom/public/publicGateway.css')

    expect(source).toContain('--gw-bg: #f9fafb')
    expect(source).toContain('.dark .public-gateway-shell')
    expect(source).toContain('border-radius: 8px')
    expect(source).not.toContain('radial-gradient')
    expect(source).not.toContain('linear-gradient')
    expect(source).not.toContain('border-radius: 999px')
    expect(source).not.toMatch(/\n\s*\.public-gateway-primary,\n\s*\.public-gateway-secondary \{\n\s*display: none;/)
  })

  it('keeps public-page dependencies in the production Tailwind scan', () => {
    const source = readSource('../tailwind.config.js')

    expect(source).toContain("'./src/custom/public/**/*.vue'")
    expect(source).toContain("'./src/custom/channels/WegooAvailableChannelsTable.vue'")
    expect(source).toContain("'./src/custom/user/monitor/**/*.vue'")
    expect(source).toContain("'./src/custom/common/WegooEmptyState.vue'")
  })
})
