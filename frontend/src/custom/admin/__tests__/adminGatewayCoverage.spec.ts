import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const adminViewsRoot = resolve(__dirname, '../../../views/admin')
const customAdminViewsRoot = resolve(__dirname, '..')

type AdminGatewayView = {
  file: string
  stylesheetImport: string
  custom?: boolean
}

const gatewayViews: AdminGatewayView[] = [
  { file: 'WegooDashboardView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooUsersView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooAccountsView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'orders/WegooAdminPaymentDashboardView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'orders/WegooAdminOrdersView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'orders/WegooAdminPaymentPlansView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooRedeemView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooSettingsView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooSubscriptionsView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooGroupsView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooChannelsView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooChannelMonitorView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooUsageView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooRiskControlView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooProxiesView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'ops/WegooOpsDashboard.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooAnnouncementsView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooPromoCodesView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'WegooBackupView.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true },
  { file: 'affiliates/WegooAdminAffiliateRecordsTable.vue', stylesheetImport: '@/custom/admin/adminApple.css', custom: true }
]

const officialAdminViews = [
  'WegooUpstreamsView.vue',
  'WegooUserPricingView.vue',
  'SchedulerView.vue',
  'InvoicesView.vue',
  'TicketsView.vue',
]

const delegatedAdminViews = [
  { file: 'affiliates/WegooAdminAffiliateInvitesView.vue', type: 'invites', custom: true },
  { file: 'affiliates/WegooAdminAffiliateRebatesView.vue', type: 'rebates', custom: true },
  { file: 'affiliates/WegooAdminAffiliateTransfersView.vue', type: 'transfers', custom: true }
]

const readAdminView = (file: string, custom = false) =>
  readFileSync(resolve(custom ? customAdminViewsRoot : adminViewsRoot, file), 'utf8')

describe('admin Gateway visual coverage', () => {
  it.each(gatewayViews)('$file keeps admin chrome compact', ({ file, stylesheetImport, custom }) => {
    const source = readAdminView(file, custom)

    expect(source).not.toContain('admin-gateway-hero')
    expect(source).not.toContain('admin-gateway-summary-card')
    expect(source).toMatch(new RegExp(`import\\s+['"]${stylesheetImport.replace('.', '\\.')}['"]`))
  })

  it.each(officialAdminViews)('%s uses the official admin chrome', (file) => {
    const source = readAdminView(file, true)

    expect(source).toContain("from '@/components/layout/AppLayout.vue'")
    expect(source).not.toContain('adminApple.css')
  })

  it('uses the shared official Toggle for scheduler state controls', () => {
    const source = readAdminView('SchedulerView.vue', true)

    expect(source).toContain("import Toggle from '@/components/common/Toggle.vue'")
    expect(source).toContain(':data-testid="`scheduler-schedulable-toggle-${entry.account_id}`"')
    expect(source).toContain(':data-testid="`scheduler-monitor-toggle-${entry.account_id}`"')
    expect(source).not.toContain('admin-apple-page')
    expect(source).not.toContain('admin-gateway-panel')
    expect(source).not.toContain('inline-flex h-4 w-7')
  })

  it.each(delegatedAdminViews)('$file delegates to the covered affiliate records table', ({ file, type, custom }) => {
    const source = readAdminView(file, custom)

    expect(source).toContain('AdminAffiliateRecordsTable')
    expect(source).toContain(`type="${type}"`)
    expect(source).toContain("from '@/custom/admin/affiliates/WegooAdminAffiliateRecordsTable.vue'")
  })

  it('keeps the order page on the shared table and inline detail dialog', () => {
    const source = readAdminView('orders/AdminOrdersView.vue')

    expect(source).toContain("import OrderTable from '@/components/payment/OrderTable.vue'")
    expect(source).toContain('<BaseDialog :show="showDetailDialog"')
    expect(source).not.toContain('AdminOrderTable')
    expect(source).not.toContain('AdminOrderDetail')

    const customSource = readAdminView('orders/WegooAdminOrdersView.vue', true)
    expect(customSource).toContain("import OrderTable from '@/custom/payment/WegooOrderTable.vue'")
    expect(customSource).toContain('<BaseDialog :show="showDetailDialog"')
    expect(customSource).toContain(':require-force="refundRequireForce"')
    expect(customSource).toContain(':warning="refundWarning"')
    expect(customSource).toContain('if (res.data.require_force)')
  })

  it('keeps Composite reasoning policy wired into the custom group page', () => {
    const source = readAdminView('WegooGroupsView.vue', true)

    expect(source).toContain('supportsReasoningEffortPolicyPlatform,')
    expect(source.match(/supportsReasoningEffortPolicyPlatform\((?:create|edit)Form\.platform\)/g)).toHaveLength(4)
  })
})
