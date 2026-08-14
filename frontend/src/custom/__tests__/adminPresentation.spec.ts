import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const srcRoot = resolve(__dirname, '../..')
const readSource = (file: string) => readFileSync(resolve(srcRoot, file), 'utf8')

describe('admin presentation overlays', () => {
  it('keeps announcement and payment presentation outside official leaves', () => {
    expect(readSource('custom/admin/WegooAnnouncementsView.vue')).toContain(
      "import AnnouncementTargetingEditor from '@/custom/admin/announcements/WegooAnnouncementTargetingEditor.vue'",
    )
    expect(readSource('custom/admin/orders/WegooAdminPaymentDashboardView.vue')).toContain(
      "import OrderStatsCards from '@/custom/admin/payment/WegooOrderStatsCards.vue'",
    )
  })

  it('keeps ops chart presentation outside official leaves', () => {
    const dashboard = readSource('custom/admin/ops/WegooOpsDashboard.vue')

    expect(dashboard).toContain(
      "import OpsDashboardSkeleton from '@/custom/admin/ops/WegooOpsDashboardSkeleton.vue'",
    )
    expect(dashboard).toContain(
      "import OpsLatencyChart from '@/custom/admin/ops/WegooOpsLatencyChart.vue'",
    )
    expect(dashboard).toContain(
      "import OpsErrorDistributionChart from '@/custom/admin/ops/WegooOpsErrorDistributionChart.vue'",
    )
    expect(dashboard).toContain(
      "import OpsErrorTrendChart from '@/custom/admin/ops/WegooOpsErrorTrendChart.vue'",
    )
    expect(dashboard).toContain(
      "import OpsThroughputTrendChart from '@/custom/admin/ops/WegooOpsThroughputTrendChart.vue'",
    )
    expect(dashboard).toContain(
      "import OpsSwitchRateTrendChart from '@/custom/admin/ops/WegooOpsSwitchRateTrendChart.vue'",
    )
  })

  it('keeps admin user presentation outside official user leaves', () => {
    const users = readSource('custom/admin/WegooUsersView.vue')

    expect(users).toContain(
      "import UserAttributesConfigModal from '@/custom/admin/user/WegooUserAttributesConfigModal.vue'",
    )
    expect(users).toContain(
      "import UserPlatformQuotaCell from '@/custom/admin/user/WegooUserPlatformQuotaCell.vue'",
    )
  })

  it('keeps usage summary presentation outside the official stats component', () => {
    expect(readSource('custom/user/WegooUsageView.vue')).toContain(
      "import UsageStatsCards from '@/custom/admin/usage/WegooUsageStatsCards.vue'",
    )
    expect(readSource('custom/admin/WegooUsageView.vue')).toContain(
      "import UsageStatsCards from '@/custom/admin/usage/WegooUsageStatsCards.vue'",
    )
  })
})
