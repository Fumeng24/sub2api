import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

const upstreamDashboardAPI = vi.hoisted(() => ({ source: 'upstream' }))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: upstreamDashboardAPI,
    accounts: { source: 'upstream-accounts' },
    channelMonitor: { source: 'upstream-channel-monitor' },
    settings: { source: 'upstream-settings' },
    users: { source: 'upstream-users' },
  },
}))

import adminAPI, {
  accountsAPI,
  channelMonitorAPI,
  settingsAPI,
  upstreamsAPI,
  usersAPI,
} from '@/custom/api/admin'
import { appendAuthSourceDefaultsToUpdateRequest } from '@/custom/api/admin/settings'

describe('custom admin API extensions', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: {} })
    post.mockReset().mockResolvedValue({ data: {} })
    put.mockReset().mockResolvedValue({ data: {} })
  })

  it('overlays custom modules while preserving untouched upstream modules', () => {
    expect(adminAPI.dashboard).toBe(upstreamDashboardAPI)
    expect(adminAPI.accounts).toBe(accountsAPI)
    expect(adminAPI.channelMonitor).toBe(channelMonitorAPI)
    expect(adminAPI.settings).toBe(settingsAPI)
    expect(adminAPI.upstreams).toBe(upstreamsAPI)
    expect(adminAPI.users).toBe(usersAPI)
  })

  it('uses custom account management endpoints', async () => {
    await accountsAPI.copyAccount(12, { name: 'copy' })
    await accountsAPI.updateSchedulerConfig(12, { concurrency: 4 })
    await accountsAPI.bulkTestModels({ account_ids: [12], model_ids: ['gpt-5.6'] })

    expect(post).toHaveBeenNthCalledWith(1, '/admin/accounts/12/copy', { name: 'copy' })
    expect(put).toHaveBeenCalledWith('/admin/accounts/12/scheduler-config', { concurrency: 4 })
    expect(post).toHaveBeenNthCalledWith(
      2,
      '/admin/accounts/bulk-test-models',
      { account_ids: [12], model_ids: ['gpt-5.6'] },
      { timeout: 300000 },
    )
  })

  it('sends the balance business category through the custom users API', async () => {
    await usersAPI.updateBalance(7, 20, 'add', 'manual credit', 'gift_compensation')

    expect(post).toHaveBeenCalledWith('/admin/users/7/balance', {
      balance: 20,
      operation: 'add',
      notes: 'manual credit',
      business_category: 'gift_compensation',
    })
  })

  it('fills missing auth-source defaults before building a settings update', () => {
    const payload = appendAuthSourceDefaultsToUpdateRequest({}, {} as never) as Record<string, unknown>

    expect(payload.auth_source_default_email_balance).toBe(0)
    expect(payload.auth_source_default_email_concurrency).toBe(5)
    expect(payload.auth_source_default_email_subscriptions).toEqual([])
    expect(payload.auth_source_default_email_platform_quotas).toEqual({
      anthropic: { daily: null, weekly: null, monthly: null },
      openai: { daily: null, weekly: null, monthly: null },
      gemini: { daily: null, weekly: null, monthly: null },
      antigravity: { daily: null, weekly: null, monthly: null },
      grok: { daily: null, weekly: null, monthly: null },
    })
  })
})
