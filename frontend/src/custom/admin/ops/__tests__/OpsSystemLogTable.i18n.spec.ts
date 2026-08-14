import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OpsSystemLogTable from '@/views/admin/ops/components/OpsSystemLogTable.vue'

const { listSystemLogs, getSystemLogSinkHealth, getRuntimeLogConfig } = vi.hoisted(() => ({
  listSystemLogs: vi.fn(),
  getSystemLogSinkHealth: vi.fn(),
  getRuntimeLogConfig: vi.fn(),
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listSystemLogs,
    getSystemLogSinkHealth,
    getRuntimeLogConfig,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

const messages: Record<string, string> = {
  'admin.ops.systemLogs.title': 'System Logs',
  'admin.ops.systemLogs.description': 'Search, filter, and clean runtime logs.',
  'admin.ops.systemLogs.all': 'All',
  'admin.ops.systemLogs.queue': 'Queue',
  'admin.ops.systemLogs.written': 'Written',
  'admin.ops.systemLogs.dropped': 'Dropped',
  'admin.ops.systemLogs.failed': 'Failed',
  'admin.ops.systemLogs.runtimeConfig': 'Runtime log configuration',
  'admin.ops.systemLogs.empty': 'No system logs',
  'admin.ops.systemLogs.search': 'Search',
  'common.loading': 'Loading',
  'common.reset': 'Reset',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

describe('OpsSystemLogTable i18n', () => {
  it('renders the official English labels instead of fixed Chinese copy', async () => {
    listSystemLogs.mockResolvedValue({ items: [], total: 0 })
    getSystemLogSinkHealth.mockResolvedValue({
      queue_depth: 0,
      queue_capacity: 100,
      dropped_count: 0,
      write_failed_count: 0,
      written_count: 12,
      avg_write_delay_ms: 0,
    })
    getRuntimeLogConfig.mockResolvedValue({
      level: 'info',
      enable_sampling: false,
      sampling_initial: 100,
      sampling_thereafter: 100,
      caller: true,
      stacktrace_level: 'error',
      retention_days: 30,
    })

    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: true,
          Pagination: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('System Logs')
    expect(wrapper.text()).toContain('Runtime log configuration')
    expect(wrapper.text()).toContain('No system logs')
    expect(wrapper.text()).not.toContain('系统日志')
  })
})
