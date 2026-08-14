import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import StatusView from '../StatusView.vue'

const { getPublicChannelMonitors } = vi.hoisted(() => ({
  getPublicChannelMonitors: vi.fn(),
}))

vi.mock('@/custom/api/publicGateway', () => ({
  default: {
    getPublicChannelMonitors,
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
    }),
  }
})

vi.mock('@/composables/useChannelMonitorFormat', () => ({
  useChannelMonitorFormat: () => ({
    providerLabel: (value: string) => value,
    statusLabel: (value: string) => value,
  }),
}))

function monitor(overrides: Record<string, unknown>) {
  return {
    id: 1,
    name: 'OpenAI Pro',
    provider: 'openai',
    group_name: 'pro',
    primary_model: 'gpt-5.5',
    primary_status: 'operational',
    primary_latency_ms: 120,
    endpoint_ping_ms: 80,
    availability_7d: 100,
    availability_15d: 100,
    availability_30d: 100,
    timeline: [],
    extra_models: [],
    updated_at: '2026-06-28T00:00:00Z',
    ...overrides,
  }
}

function mountView() {
  return mount(StatusView, {
    global: {
      stubs: {
        PublicGatewayHeader: true,
        MonitorCardGrid: { template: '<div data-test="monitor-card-grid" />' },
        Icon: true,
      },
    },
  })
}

describe('public StatusView overall status', () => {
  beforeEach(() => {
    getPublicChannelMonitors.mockReset()
  })

  it('keeps the global status degraded when filters hide degraded services', async () => {
    getPublicChannelMonitors.mockResolvedValue({
      last_updated_at: '2026-06-28T01:23:45Z',
      trend_period: '7d',
      items: [
        monitor({ id: 1, provider: 'openai', primary_status: 'operational' }),
        monitor({ id: 2, name: 'Claude Plus', provider: 'anthropic', primary_status: 'failed' }),
      ],
    })

    const wrapper = mountView()
    await flushPromises()

    ;(wrapper.vm as any).providerFilter = 'openai'
    await nextTick()

    expect(wrapper.get('[data-test="public-overall-status"]').text()).toBe('全站部分波动')
    wrapper.unmount()
  })

  it('uses the backend monitor checked time instead of local request time', async () => {
    getPublicChannelMonitors.mockResolvedValue({
      last_updated_at: '2026-06-28T01:23:45Z',
      trend_period: '7d',
      items: [
        monitor({
          id: 1,
          primary_status: 'operational',
          timeline: [{ status: 'operational', latency_ms: 120, ping_latency_ms: 80, checked_at: '2026-06-28T00:00:00Z' }],
        }),
      ],
    })

    const wrapper = mountView()
    await flushPromises()

    expect((wrapper.vm as any).lastUpdatedAt.toISOString()).toBe('2026-06-28T01:23:45.000Z')
    expect(wrapper.get('[data-test="public-last-updated"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('keeps the backend channel monitor order', async () => {
    getPublicChannelMonitors.mockResolvedValue({
      last_updated_at: '2026-06-28T01:23:45Z',
      trend_period: '7d',
      items: [
        monitor({ id: 10, name: 'Backend first', primary_status: 'failed', primary_latency_ms: 900 }),
        monitor({ id: 20, name: 'Backend second', primary_status: 'operational', primary_latency_ms: 80 }),
      ],
    })

    const wrapper = mountView()
    await flushPromises()

    expect((wrapper.vm as any).sortedItems.map((item: { id: number }) => item.id)).toEqual([10, 20])
    wrapper.unmount()
  })
})
