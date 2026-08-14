import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import zh from '@/custom/i18n/locales/zh'
import en from '@/custom/i18n/locales/en'

const componentPath = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../../custom/layout/WegooAppSidebar.vue',
)
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../styles/style.css')
const styleSource = readFileSync(stylePath, 'utf8')
const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../../router/index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })

  it('only renders deployment version controls for administrators', () => {
    expect(componentSource).toContain('<VersionBadge v-if="isAdmin" :version="siteVersion" />')
  })
})

describe('AppSidebar feature-gated navigation', () => {
  it('hides user subscription entries unless subscription groups exist', () => {
    expect(componentSource).toContain('const flagUserSubscriptions = () => subscriptionCapabilityStore.hasSubscriptionGroups')
    expect(componentSource).toContain("{ path: '/subscriptions', label: t('nav.mySubscriptions'), icon: CreditCardIcon, hideInSimpleMode: true, featureFlag: flagUserSubscriptions }")
  })

  it('hides the balance recharge entry unless payment and checkout access are available', () => {
    expect(componentSource).toContain('const flagPurchase = () => flagPayment() && paymentCheckoutStore.canAccessPurchase')
    expect(componentSource).toContain("{ path: '/purchase', label: t('nav.buySubscription'), icon: RechargeSubscriptionIcon, hideInSimpleMode: true, featureFlag: flagPurchase }")
  })

  it('keeps payment orders behind the payment feature flag', () => {
    expect(componentSource).toContain("{ path: '/orders', label: t('nav.myOrders'), icon: OrderListIcon, hideInSimpleMode: true, featureFlag: flagPayment }")
  })

  it('loads public settings before enforcing payment-gated routes', () => {
    expect(routerSource).toContain('if ((to.meta.requiresPayment || to.meta.requiresRiskControl) && !appStore.publicSettingsLoaded) {')
    expect(routerSource).toContain('appStore.cachedPublicSettings?.payment_enabled === false')
  })

  it('guards direct access with the same feature capabilities as sidebar entries', () => {
    expect(routerSource).toContain("import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'")
    expect(routerSource).toContain('if (to.meta.featureFlag) {')
    expect(routerSource).toContain('if (!isFeatureFlagEnabled(FeatureFlags[to.meta.featureFlag])) {')
    expect(routerSource).toContain("featureFlag: 'availableChannels'")
    expect(routerSource).toContain("featureFlag: 'channelMonitor'")
    expect(routerSource).toContain("featureFlag: 'affiliate'")
  })

  it('guards direct subscription access with backend subscription group capability', () => {
    expect(routerSource).toContain("import { useSubscriptionCapabilityStore } from '@/custom/stores/subscriptionCapability'")
    expect(routerSource).toContain('requiresSubscriptionGroups: true')
    expect(routerSource).toContain('const subscriptionCapabilityStore = useSubscriptionCapabilityStore()')
    expect(routerSource).toContain('subscriptionCapabilityStore.fetchSubscriptionCapability().catch(() => false)')
  })
})

describe('AppSidebar i18n navigation labels', () => {
  it('has Chinese and English labels for every nav key used by the sidebar and route titles', () => {
    const navKeyPattern = /(?:t\(|titleKey:\s*)['"]nav\.([A-Za-z0-9_]+)['"]/g
    const navKeys = new Set<string>()

    for (const source of [componentSource, routerSource]) {
      for (const match of source.matchAll(navKeyPattern)) {
        navKeys.add(match[1])
      }
    }

    const zhNav = zh.nav as Record<string, unknown>
    const enNav = en.nav as Record<string, unknown>
    const missingZh = [...navKeys].filter((key) => typeof zhNav[key] !== 'string' || zhNav[key] === '')
    const missingEn = [...navKeys].filter((key) => typeof enNav[key] !== 'string' || enNav[key] === '')

    expect(missingZh).toEqual([])
    expect(missingEn).toEqual([])
    expect(navKeys).toContain('schedulerManagement')
  })
})
