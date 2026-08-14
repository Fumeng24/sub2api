import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ChannelStatusView from '@/custom/user/WegooChannelStatusView.vue'

const {
  listChannelMonitorViews,
  fetchChannelMonitorDetail,
  showError,
  useAutoRefresh,
  autoRefreshState,
} = vi.hoisted(() => {
  const state = {
    enabled: { value: false },
    intervalSeconds: { value: 60 },
    countdown: { value: 60 },
    intervals: [30, 60, 120] as const,
    setEnabled: vi.fn((value: boolean) => {
      state.enabled.value = value
    }),
    setInterval: vi.fn((value: number) => {
      state.intervalSeconds.value = value
    }),
    start: vi.fn(),
    stop: vi.fn(),
  }

  return {
    listChannelMonitorViews: vi.fn(),
    fetchChannelMonitorDetail: vi.fn(),
    showError: vi.fn(),
    useAutoRefresh: vi.fn(() => state),
    autoRefreshState: state,
  }
})

vi.mock('@/api/channelMonitor', () => ({
  list: listChannelMonitorViews,
  status: fetchChannelMonitorDetail,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: { channel_monitor_enabled: true },
    showError,
  }),
}))

vi.mock('@/composables/useAutoRefresh', () => ({
  useAutoRefresh,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'channelStatus.gateway.totalServices') return '监控服务'
        if (key === 'channelStatus.gateway.operationalServices') return '正常服务'
        if (key === 'channelStatus.gateway.degradedServices') return '异常服务'
        if (key === 'channelStatus.gateway.avgLatency') return '平均延迟'
        if (key === 'channelStatus.windowTab.7d') return '7 天'
        if (params?.window) return `${key}:${params.window}`
        return key
      },
    }),
  }
})

function mountView() {
  return mount(ChannelStatusView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        WegooChannelStatusWorkspace: {
          props: ['overallStatus'],
          template: '<div data-test="overall-status">{{ overallStatus }}</div>',
        },
        MonitorDetailDialog: true,
      },
    },
  })
}

describe('ChannelStatusView monitor status semantics', () => {
  beforeEach(() => {
    listChannelMonitorViews.mockReset()
    fetchChannelMonitorDetail.mockReset()
    showError.mockReset()
    useAutoRefresh.mockClear()
    autoRefreshState.enabled.value = false
    autoRefreshState.intervalSeconds.value = 60
    autoRefreshState.countdown.value = 60
    autoRefreshState.setEnabled.mockClear()
    autoRefreshState.setInterval.mockClear()
    autoRefreshState.start.mockClear()
    autoRefreshState.stop.mockClear()
  })

  it('does not report operational when monitor data is empty', async () => {
    listChannelMonitorViews.mockResolvedValue({ items: [] })

    const wrapper = mountView()

    await flushPromises()

    const overallStatus = wrapper.get('[data-test="overall-status"]')
    expect(overallStatus.text()).toBe('degraded')
    expect(overallStatus.text()).not.toBe('operational')
    wrapper.unmount()
  })
})
