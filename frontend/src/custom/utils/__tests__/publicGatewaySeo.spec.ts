import { readFileSync, readdirSync, statSync } from 'node:fs'
import { dirname, extname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { transformWegooIndexHtml } from '@/custom/vite/wegooIndexHtml'

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../../../..')
const thisTestFile = fileURLToPath(import.meta.url)
const frontendRoot = join(repoRoot, 'frontend')
const publicRoot = join(frontendRoot, 'public')
const srcRoot = join(frontendRoot, 'src')

function readFrontend(relativePath: string): string {
  return readFileSync(join(frontendRoot, relativePath), 'utf8')
}

function readRenderedIndexHtml(): string {
  return transformWegooIndexHtml(readFrontend('index.html'))
}

function collectTextFiles(root: string, out: string[] = []): string[] {
  for (const entry of readdirSync(root)) {
    const fullPath = join(root, entry)
    const stat = statSync(fullPath)
    if (stat.isDirectory()) {
      if (entry === 'node_modules' || entry === 'dist') continue
      collectTextFiles(fullPath, out)
      continue
    }

    if (['.vue', '.ts', '.js', '.html', '.css', '.txt', '.xml'].includes(extname(fullPath))) {
      out.push(fullPath)
    }
  }
  return out
}

function parseXml(source: string, label: string): Document {
  const doc = new DOMParser().parseFromString(source, 'application/xml')
  const parserError = doc.querySelector('parsererror')
  expect(parserError?.textContent ?? '').toBe('')
  expect(doc.documentElement.nodeName, `${label} must have a root element`).not.toBe('')
  return doc
}

function textValues(doc: Document, selector: string): string[] {
  return [...doc.querySelectorAll(selector)]
    .map((node) => node.textContent?.trim() ?? '')
    .filter(Boolean)
}

function routerCanonicalPaths(): string[] {
  const routerSource = readFrontend('src/custom/router/index.ts')
  return [...routerSource.matchAll(/canonicalPath:\s*['"]([^'"]+)['"]/g)]
    .map((match) => match[1])
    .filter((path) => path.startsWith('/'))
}

describe('public gateway SEO and brand isolation', () => {
  it('does not ship reference-site brand, domain, or primary palette in frontend implementation files', () => {
    const referenceDomain = ['buzz', 'ai.cc'].join('')
    const referenceBrand = ['buzz', 'ai'].join('')
    const referenceShortBrand = ['b', 'uzz'].join('')
    const referenceGold = ['#', 'D4A85C'].join('')
    const referenceGoldLight = ['#', 'F0DCA0'].join('')
    const forbidden = [
      referenceDomain,
      referenceBrand,
      referenceShortBrand,
      referenceGold,
      referenceGoldLight,
    ].map((term) => term.toLowerCase())

    const checkedFiles = [
      'index.html',
      ...collectTextFiles(publicRoot).map((path) => path.slice(frontendRoot.length + 1)),
      ...collectTextFiles(srcRoot)
        .filter((path) => path !== thisTestFile)
        .map((path) => path.slice(frontendRoot.length + 1)),
    ]

    const matches: string[] = []
    for (const relativePath of checkedFiles) {
      const content = readFrontend(relativePath).toLowerCase()
      for (const term of forbidden) {
        if (content.includes(term)) {
          matches.push(`${relativePath}: ${term}`)
        }
      }
    }

    expect(matches).toEqual([])
  })

  it('keeps the HTML fallback and structured data needed by crawlers', () => {
    const html = readRenderedIndexHtml()

    expect(html).toContain('<link rel="canonical" href="https://ai.wegoo.site/home" />')
    expect(html).toContain('href="https://ai.wegoo.site/llms.txt"')
    expect(html).toContain('href="https://ai.wegoo.site/feed.xml"')
    expect(html).toContain('"@type": "Organization"')
    expect(html).toContain('"@type": "WebApplication"')
    expect(html).toContain('"@type": "FAQPage"')
    expect(html).toContain('<h1>Wegoo AI - AI Gateway for Developers</h1>')
    expect(html).toContain('<a href="/pricing">')
    expect(html).toContain('<a href="/docs">')
    expect(html).toContain('<a href="/status">')
    expect(html).toContain('<a href="/enterprise">')
    expect(html).toContain('<a href="/llms.txt">')
  })

  it('keeps index.html structured data parseable and complete for the gateway homepage', () => {
    const html = readRenderedIndexHtml()
    const jsonLdMatch = html.match(/<script type="application\/ld\+json">\s*([\s\S]*?)\s*<\/script>/)

    expect(jsonLdMatch).not.toBeNull()
    const data = JSON.parse(jsonLdMatch?.[1] ?? '{}') as {
      '@graph'?: Array<Record<string, unknown>>
    }
    const graph = data['@graph'] ?? []
    const types = new Set(graph.map((item) => item['@type']))
    const faq = graph.find((item) => item['@type'] === 'FAQPage') as {
      mainEntity?: Array<Record<string, unknown>>
    } | undefined
    const app = graph.find((item) => item['@type'] === 'WebApplication') as {
      featureList?: unknown
    } | undefined

    expect([...types]).toEqual(expect.arrayContaining(['Organization', 'WebSite', 'WebApplication', 'FAQPage']))
    expect(Array.isArray(app?.featureList) ? app.featureList.length : 0).toBeGreaterThanOrEqual(4)
    expect(Array.isArray(faq?.mainEntity) ? faq.mainEntity.length : 0).toBeGreaterThanOrEqual(3)
  })

  it('keeps public crawler resources aligned with gateway public routes and APIs', () => {
    const robots = readFrontend('public/robots.txt')
    const sitemap = readFrontend('public/sitemap.xml')
    const llms = readFrontend('public/llms.txt')
    const feed = readFrontend('public/feed.xml')

    for (const path of ['/home', '/pricing', '/docs', '/status', '/enterprise', '/key-usage']) {
      expect(sitemap).toContain(`https://ai.wegoo.site${path}`)
      expect(llms).toContain(`https://ai.wegoo.site${path}`)
    }

    for (const path of [
      '/api/v1/public/model-pricing',
      '/api/v1/public/channels/available',
      '/api/v1/public/channel-monitors',
    ]) {
      expect(robots).toContain(`Allow: ${path}`)
      expect(llms).toContain(`https://ai.wegoo.site${path}`)
    }

    for (const privatePath of ['/admin', '/dashboard', '/keys', '/purchase', '/payment', '/api']) {
      expect(robots).toContain(`Disallow: ${privatePath}`)
    }

    expect(robots).toContain('Sitemap: https://ai.wegoo.site/sitemap.xml')
    expect(sitemap).toContain('https://ai.wegoo.site/llms.txt')
    expect(sitemap).toContain('https://ai.wegoo.site/feed.xml')
    expect(feed).toContain('<link>https://ai.wegoo.site/home</link>')
    expect(feed).toContain('<link>https://ai.wegoo.site/pricing</link>')
    expect(feed).toContain('<link>https://ai.wegoo.site/llms.txt</link>')
  })

  it('keeps sitemap and feed XML parseable and aligned with public canonical routes', () => {
    const sitemapDoc = parseXml(readFrontend('public/sitemap.xml'), 'sitemap.xml')
    const feedDoc = parseXml(readFrontend('public/feed.xml'), 'feed.xml')
    const sitemapUrls = textValues(sitemapDoc, 'loc')
    const feedLinks = textValues(feedDoc, 'channel > link, item > link')

    for (const path of routerCanonicalPaths()) {
      expect(sitemapUrls).toContain(`https://ai.wegoo.site${path}`)
    }

    for (const privatePath of ['/login', '/setup', '/dashboard', '/keys', '/purchase', '/payment']) {
      expect(sitemapUrls).not.toContain(`https://ai.wegoo.site${privatePath}`)
    }

    expect(feedDoc.querySelector('rss')?.getAttribute('version')).toBe('2.0')
    expect(feedDoc.querySelector('channel > title')?.textContent).toContain('Wegoo AI')
    expect(feedDoc.querySelector('channel > item')).not.toBeNull()
    expect(feedLinks).toEqual(expect.arrayContaining([
      'https://ai.wegoo.site/home',
      'https://ai.wegoo.site/pricing',
      'https://ai.wegoo.site/llms.txt',
    ]))
  })

  it('keeps robots allowlist and llms public fact sources in sync', () => {
    const robots = readFrontend('public/robots.txt')
    const llms = readFrontend('public/llms.txt')
    const allowLines = robots
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter((line) => line.startsWith('Allow: /api/v1/public/'))
      .map((line) => line.slice('Allow: '.length))

    expect(allowLines.length).toBeGreaterThanOrEqual(3)
    for (const allowedPath of allowLines) {
      expect(llms).toContain(`https://ai.wegoo.site${allowedPath}`)
    }
    expect(robots).toContain('Disallow: /api')
  })

  it('keeps public model endpoint capabilities driven by backend response fields', () => {
    const modelsView = readFrontend('src/custom/public/ModelsView.vue')
    const channelTable = readFrontend('src/custom/channels/WegooAvailableChannelsTable.vue')
    const channelTypes = readFrontend('src/custom/api/channels.d.ts')

    expect(channelTypes).toContain('endpoints?: string[]')
    expect(channelTypes).toContain('supported_endpoint_types?: string[]')
    expect(modelsView).toContain('sectionEndpointValues(section)')
    expect(modelsView).toContain('section.endpoints && section.endpoints.length > 0')
    expect(modelsView).toContain('section.supported_endpoint_types && section.supported_endpoint_types.length > 0')
    expect(channelTable).toContain('sectionEndpoints(section)')
    expect(channelTable).toContain('section.endpoints && section.endpoints.length > 0')
    expect(channelTable).toContain('section.supported_endpoint_types && section.supported_endpoint_types.length > 0')
  })

  it('keeps public docs aligned with user-visible issue-record troubleshooting categories', () => {
    const docsView = readFrontend('src/custom/public/DocsView.vue')

    expect(docsView).toContain('troubleshootingRows')
    expect(docsView).toContain('/usage?tab=errors')
    for (const category of [
      'auth',
      'rate_limit',
      'quota',
      'invalid_request',
      'service_unavailable',
      'cyber',
    ]) {
      expect(docsView).toContain(`category: '${category}'`)
    }
    for (const status of ['401 / 403', '429', '400 / 422', '5xx / overload']) {
      expect(docsView).toContain(status)
    }
    expect(docsView).toContain('Do not upload full secret keys')
    expect(docsView).toContain('不要上传完整密钥')
  })
})
