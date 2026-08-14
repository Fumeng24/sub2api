import type { UserMonitorView } from '@/api/channelMonitor'
import { STATUS_OPERATIONAL } from '@/constants/channelMonitor'
import type { OverallStatus } from '@/custom/user/monitor/WegooMonitorHero.vue'

export function resolveOverallMonitorStatus(items: UserMonitorView[]): OverallStatus {
  if (items.length === 0) return 'degraded'
  return items.every((item) => item.primary_status === STATUS_OPERATIONAL)
    ? 'operational'
    : 'degraded'
}
