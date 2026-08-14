import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { MonitorTimelinePoint } from '@/api/channelMonitor'
import MonitorTimeline from '@/custom/user/monitor/WegooMonitorTimeline.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/custom/composables/useChannelMonitorFormat', () => ({
  useChannelMonitorFormat: () => ({
    statusLabel: (status: string) => status,
    formatLatency: (latency: number | null) => latency == null ? '-' : String(latency),
    formatRelativeTime: (checkedAt: string) => checkedAt,
  }),
}))

function point(
  status: MonitorTimelinePoint['status'],
  latency: number,
  checkedAt: string,
): MonitorTimelinePoint {
  return {
    status,
    latency_ms: latency,
    ping_latency_ms: 20,
    checked_at: checkedAt,
  }
}

describe('WegooMonitorTimeline', () => {
  it('pads on the left and keeps the newest sample on the right', () => {
    const wrapper = mount(MonitorTimeline, {
      props: {
        buckets: [
          point('operational', 120, 'newest'),
          point('failed', 900, 'older'),
        ],
        countdownSeconds: 12,
        length: 4,
      },
    })

    const bars = wrapper.findAll('[data-test="monitor-timeline-bar"]')
    expect(bars).toHaveLength(4)
    expect(bars[0].classes()).toContain('monitor-timeline-bar--empty')
    expect(bars[1].classes()).toContain('monitor-timeline-bar--empty')
    expect(bars[2].classes()).toContain('monitor-timeline-bar--failed')
    expect(bars[2].attributes('title')).toBe('older · failed · 900ms')
    expect(bars[3].classes()).toContain('monitor-timeline-bar--operational')
    expect(bars[3].attributes('title')).toBe('newest · operational · 120ms')
  })

  it('replaces timeline bars with the maintenance state', () => {
    const wrapper = mount(MonitorTimeline, {
      props: {
        buckets: [point('operational', 120, 'newest')],
        countdownSeconds: 12,
        maintenance: true,
      },
    })

    expect(wrapper.find('[data-test="monitor-timeline"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('monitorCommon.maintenancePaused')
  })
})
