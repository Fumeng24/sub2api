import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const srcRoot = resolve(__dirname, '../..')

type RouteComponent = {
  path: string
  component: string
}

const readSource = (file: string) => readFileSync(resolve(srcRoot, file), 'utf8')
const readRouteSource = (file: string) =>
  readSource(file.startsWith('custom/') ? file : `views/${file}`)

function routeComponents(): RouteComponent[] {
  const lines = readSource('custom/router/index.ts').split(/\r?\n/)
  const routes: RouteComponent[] = []
  let currentPath = ''
  let redirected = false

  for (const line of lines) {
    const pathMatch = line.match(/path:\s*'([^']+)'/)
    if (pathMatch) {
      currentPath = pathMatch[1]
      redirected = false
    }

    if (/redirect:\s*/.test(line)) {
      currentPath = ''
      redirected = true
    }

    const componentMatch = line.match(
      /component:\s*\(\)\s*=>\s*import\('@\/(views|custom)\/([^']+)'\)/,
    )
    if (componentMatch && currentPath && !redirected) {
      routes.push({
        path: currentPath,
        component:
          componentMatch[1] === 'views'
            ? componentMatch[2]
            : `custom/${componentMatch[2]}`,
      })
      currentPath = ''
    }
  }

  return routes
}

const authLayoutViews = [
  'custom/auth/WegooLoginView.vue',
  'custom/auth/WegooRegisterView.vue',
  'custom/auth/WegooEmailVerifyView.vue',
  'custom/auth/WegooLinuxDoCallbackView.vue',
  'custom/auth/WegooWechatCallbackView.vue',
  'custom/auth/WegooDingTalkCallbackView.vue',
  'custom/auth/WegooDingTalkEmailCompletionView.vue',
  'custom/auth/WegooOidcCallbackView.vue',
  'custom/auth/WegooForgotPasswordView.vue',
  'custom/auth/WegooResetPasswordView.vue',
]

const callbackShellViews = [
  'custom/auth/WegooOAuthCallbackView.vue',
  'custom/auth/WegooWechatPaymentCallbackView.vue',
]

const publicGatewayViews = [
  'custom/public/ModelsView.vue',
  'custom/public/DocsView.vue',
  'custom/public/StatusView.vue',
  'custom/public/EnterpriseView.vue',
]

const gatewayUserAppLayoutViews = [
  'custom/user/WegooDashboardView.vue',
  'custom/user/WegooKeysView.vue',
  'custom/user/WegooBatchImageGuideView.vue',
  'custom/user/WegooUsageView.vue',
  'custom/user/WegooRedeemView.vue',
  'custom/user/WegooAffiliateView.vue',
  'custom/user/WegooAvailableChannelsView.vue',
  'custom/user/WegooProfileView.vue',
  'custom/user/WegooSubscriptionsView.vue',
  'custom/user/WegooPaymentView.vue',
  'custom/user/WegooUserOrdersView.vue',
  'custom/user/WegooPaymentQRCodeView.vue',
  'custom/user/WegooStripePaymentView.vue',
  'custom/user/WegooAirwallexPaymentView.vue',
  'custom/user/WegooCustomPageView.vue',
  'custom/user/WegooChannelStatusView.vue',
]

const officialUserAppLayoutViews = [
  'custom/user/ImageGenerationView.vue',
  'custom/user/WegooMessagesView.vue',
  'custom/user/TicketsView.vue',
  'custom/user/InvoicesView.vue',
]

const userAppLayoutViews = [
  ...gatewayUserAppLayoutViews,
  ...officialUserAppLayoutViews,
]

const standaloneGatewayViews = [
  'custom/home/WegooHomeView.vue',
  'ModelPlazaView.vue',
  'custom/public/WegooKeyUsageView.vue',
  'custom/public/WegooLegalDocumentView.vue',
  'custom/public/WegooNotFoundView.vue',
  'custom/setup/WegooSetupWizardView.vue',
  'custom/user/WegooPaymentResultView.vue',
  'custom/user/WegooStripePopupView.vue',
]

const adminGatewayViews = [
  'custom/admin/WegooDashboardView.vue',
  'custom/admin/ops/WegooOpsDashboard.vue',
  'custom/admin/SchedulerView.vue',
  'custom/admin/WegooUsersView.vue',
  'custom/admin/WegooGroupsView.vue',
  'custom/admin/WegooUserPricingView.vue',
  'custom/admin/WegooChannelsView.vue',
  'custom/admin/WegooChannelMonitorView.vue',
  'custom/admin/WegooSubscriptionsView.vue',
  'custom/admin/WegooAccountsView.vue',
  'custom/admin/WegooUpstreamsView.vue',
  'custom/admin/WegooAnnouncementsView.vue',
  'custom/admin/TicketsView.vue',
  'custom/admin/WegooProxiesView.vue',
  'custom/admin/WegooRedeemView.vue',
  'custom/admin/WegooPromoCodesView.vue',
  'custom/admin/WegooSettingsView.vue',
  'custom/admin/WegooRiskControlView.vue',
  'custom/admin/WegooUsageView.vue',
  'custom/admin/affiliates/WegooAdminAffiliateInvitesView.vue',
  'custom/admin/affiliates/WegooAdminAffiliateRebatesView.vue',
  'custom/admin/affiliates/WegooAdminAffiliateTransfersView.vue',
  'custom/admin/orders/WegooAdminPaymentDashboardView.vue',
  'custom/admin/orders/WegooAdminOrdersView.vue',
  'custom/admin/orders/WegooAdminPaymentPlansView.vue',
  'custom/admin/InvoicesView.vue',
]

describe('route-level Gateway coverage', () => {
  it('keeps every component route classified for visual audit', () => {
    const routedComponents = [...new Set(routeComponents().map((route) => route.component))].sort()
    const classifiedComponents = [
      ...authLayoutViews,
      ...callbackShellViews,
      ...publicGatewayViews,
      ...userAppLayoutViews,
      ...standaloneGatewayViews,
      ...adminGatewayViews,
    ].sort()

    expect(classifiedComponents).toEqual(routedComponents)
  })

  it.each(authLayoutViews)('%s uses the Gateway auth shell', (file) => {
    const source = readRouteSource(file)

    expect(source).toContain('<AuthLayout>')
    expect(source).toContain("import AuthLayout from '@/custom/layout/WegooAuthLayout.vue'")
  })

  it('routes shared console layout through the site overlay', () => {
    const appLayout = readSource('custom/layout/WegooAppLayout.vue')

    expect(appLayout).toContain("import AppHeader from '@/custom/layout/WegooAppHeader.vue'")
    expect(appLayout).toContain("import AppSidebar from '@/custom/layout/WegooAppSidebar.vue'")
    expect(readSource('custom/layout/WegooAppHeader.vue')).toContain(
      "import AnnouncementBell from '@/custom/common/WegooAnnouncementBell.vue'",
    )
    expect(readSource('custom/App.vue')).toContain(
      "import AnnouncementPopup from '@/custom/common/WegooAnnouncementPopup.vue'",
    )
    expect(readSource('custom/layout/WegooAuthLayout.vue')).toContain('gateway-auth-shell')
  })

  it('keeps dashboard presentation in the site overlay', () => {
    const dashboard = readSource('custom/user/WegooDashboardView.vue')

    expect(dashboard).toContain(
      "import UserDashboardStats from '@/custom/user/dashboard/WegooUserDashboardStats.vue'",
    )
    expect(dashboard).toContain(
      "import UserDashboardCharts from '@/custom/user/dashboard/WegooUserDashboardCharts.vue'",
    )
    expect(dashboard).toContain(
      "import UserDashboardRecentUsage from '@/custom/user/dashboard/WegooUserDashboardRecentUsage.vue'",
    )
  })

  it('keeps channel monitor presentation in the site overlay', () => {
    const publicStatus = readSource('custom/public/StatusView.vue')
    const userStatusView = readSource('custom/user/WegooChannelStatusView.vue')
    const userStatus = readSource('custom/user/WegooChannelStatusWorkspace.vue')
    const cardGrid = readSource('custom/user/monitor/WegooMonitorCardGrid.vue')
    const card = readSource('custom/user/monitor/WegooMonitorCard.vue')

    expect(publicStatus).toContain(
      "import MonitorCardGrid from '@/custom/user/monitor/WegooMonitorCardGrid.vue'",
    )
    expect(userStatusView).toContain(
      "import WegooChannelStatusWorkspace from '@/custom/user/WegooChannelStatusWorkspace.vue'",
    )
    expect(userStatus).toContain(
      "import MonitorCardGrid from '@/custom/user/monitor/WegooMonitorCardGrid.vue'",
    )
    expect(cardGrid).toContain("import MonitorCard from './WegooMonitorCard.vue'")
    expect(card).toContain("import MonitorTimeline from './WegooMonitorTimeline.vue'")
    expect(card).toContain(
      "import MonitorAvailabilityRow from './WegooMonitorAvailabilityRow.vue'",
    )
  })

  it('keeps profile overview presentation in the site overlay', () => {
    const profile = readSource('custom/user/WegooProfileView.vue')
    const profileInfo = readSource('custom/user/profile/WegooProfileInfoCard.vue')

    expect(profile).toContain(
      "import WegooProfileHero from '@/custom/user/profile/WegooProfileHero.vue'",
    )
    expect(profile).toContain(
      "import ProfileInfoCard from '@/custom/user/profile/WegooProfileInfoCard.vue'",
    )
    expect(profile).toContain(
      "import ProfileBalanceNotifyCard from '@/custom/user/profile/WegooProfileBalanceNotifyCard.vue'",
    )
    expect(profile).toContain(
      "import ProfilePasswordForm from '@/custom/user/profile/WegooProfilePasswordForm.vue'",
    )
    expect(profile).toContain(
      "import ProfileTotpCard from '@/custom/user/profile/WegooProfileTotpCard.vue'",
    )
    expect(profileInfo).toContain("import ProfileAvatarCard from './WegooProfileAvatarCard.vue'")
    expect(profileInfo).toContain("import ProfileEditForm from './WegooProfileEditForm.vue'")
    expect(profileInfo).toContain(
      "import ProfileIdentityBindingsSection from './WegooProfileIdentityBindingsSection.vue'",
    )
    const totpCard = readSource('custom/user/profile/WegooProfileTotpCard.vue')
    expect(totpCard).toContain("import TotpSetupModal from './WegooTotpSetupModal.vue'")
    expect(totpCard).toContain("import TotpDisableDialog from './WegooTotpDisableDialog.vue'")
  })

  it.each(callbackShellViews)('%s uses the Gateway callback shell', (file) => {
    const source = readRouteSource(file)

    expect(source).toContain('gateway-callback-shell')
    expect(source).toContain('gateway-callback-card')
  })

  it.each(publicGatewayViews)('%s uses the public Gateway shell', (file) => {
    const source = readRouteSource(file)

    expect(source).toContain('public-gateway-shell')
    expect(source).toContain('<PublicGatewayHeader />')
  })

  it.each(gatewayUserAppLayoutViews)('%s uses the shared Gateway console layout', (file) => {
    const source = readRouteSource(file)

    expect(source).toContain('AppLayout')
    expect(source).toContain("from '@/custom/layout/WegooAppLayout.vue'")
  })

  it.each(officialUserAppLayoutViews)('%s uses the official console layout', (file) => {
    const source = readRouteSource(file)

    expect(source).toContain('AppLayout')
    expect(source).toContain("from '@/components/layout/AppLayout.vue'")
    expect(source).not.toContain("from '@/custom/layout/WegooAppLayout.vue'")
  })

  it('keeps standalone routes in explicit Gateway shells', () => {
    expect(readSource('router/index.ts')).toContain(
      "component: () => import('@/custom/home/WegooHomeView.vue')",
    )
    expect(readSource('custom/home/WegooHomeView.vue')).toContain('gateway-home')
    expect(readSource('custom/public/WegooKeyUsageView.vue')).toContain('gateway-key-usage')
    expect(readSource('custom/public/WegooLegalDocumentView.vue')).toContain('legal-gateway-shell')
    expect(readSource('custom/setup/WegooSetupWizardView.vue')).toContain('setup-gateway-shell')
    expect(readSource('custom/user/WegooPaymentResultView.vue')).toContain('gateway-payment-result')
    expect(readSource('custom/user/WegooStripePopupView.vue')).toContain('gateway-payment-popup')
    const notFoundSource = readSource('custom/public/WegooNotFoundView.vue')
    expect(notFoundSource).toContain('gateway-not-found')
    expect(notFoundSource).toContain('gateway-not-found__panel')
    expect(notFoundSource).toContain('AI Gateway · Route check')
    expect(notFoundSource).toContain("to=\"/home\"")
    expect(notFoundSource).not.toContain('var(--apple-bg)')
  })

  it('keeps /auth as a login route compatibility alias', () => {
    const source = readSource('custom/router/index.ts')

    expect(source).toContain("path: '/login'")
    expect(source).toContain("alias: '/auth'")
    expect(source).toContain("'/login', '/auth'")
    expect(source).toContain("to.path === '/auth'")
  })
})
