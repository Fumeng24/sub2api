import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
    put,
    delete: del,
  },
}))

import upstreamsAPI from '@/custom/api/admin/upstreams'

describe('admin upstreams api', () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20 } })
    post.mockReset().mockResolvedValue({ data: {} })
    put.mockReset().mockResolvedValue({ data: {} })
    del.mockReset().mockResolvedValue({ data: { deleted: true, unbound_account_count: 0 } })
  })

  it('lists upstreams and unbound account candidates with explicit pagination', async () => {
    await upstreamsAPI.list(2, 50, 'primary')
    await upstreamsAPI.listBindCandidates(7, 1, 100, 'legacy')

    expect(get).toHaveBeenNthCalledWith(1, '/admin/upstreams', {
      params: { page: 2, page_size: 50, search: 'primary' },
    })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/upstreams/7/bind-candidates', {
      params: { page: 1, page_size: 100, search: 'legacy' },
    })
  })

  it('does not allow rebinding an existing account by default', async () => {
    await upstreamsAPI.bindAccounts(7, [11, 12])

    expect(post).toHaveBeenCalledWith('/admin/upstreams/7/bind', {
      account_ids: [11, 12],
      allow_rebind: false,
    })
  })

  it('keeps generation preview separate from the confirmed write', async () => {
    const specs = [{
      platform: 'openai' as const,
      upstream_group_name: 'vip',
      models: ['gpt-test'],
      local_group_ids: [3],
      rate_multiplier: 0.25,
    }]
    post
      .mockResolvedValueOnce({ data: { valid: true, creates: 1, skips: 0, items: [] } })
      .mockResolvedValueOnce({ data: { results: [{ index: 0, success: true, skipped: false, account_id: 22 }] } })

    await upstreamsAPI.previewGeneratedAccounts(7, specs)
    await upstreamsAPI.generateAccounts(7, specs)

    expect(post).toHaveBeenNthCalledWith(1, '/admin/upstreams/7/accounts/preview', { accounts: specs })
    expect(post).toHaveBeenNthCalledWith(2, '/admin/upstreams/7/accounts/generate', { accounts: specs })
  })

  it('tests a selected model through the upstream endpoint', async () => {
    post.mockResolvedValueOnce({ data: {
      success: true,
      platform: 'openai',
      group_name: 'vip',
      model: 'gpt-test',
      latency_ms: 42,
      status: 'ok',
    } })

    await upstreamsAPI.testModel(7, { platform: 'openai', group_name: 'vip', model: 'gpt-test' })

    expect(post).toHaveBeenCalledWith('/admin/upstreams/7/model-test', {
      platform: 'openai',
      group_name: 'vip',
      model: 'gpt-test',
    }, { timeout: 120_000 })
  })

  it('discovers and batch checks one upstream group with one request', async () => {
    post.mockResolvedValueOnce({ data: {
      success: false,
      platform: 'openai',
      group_name: 'vip',
      status: 'partial',
      latency_ms: 42,
      available_models: ['gpt-test'],
      results: [],
    } })

    await upstreamsAPI.probeModels(7, {
      platform: 'openai',
      group_name: 'vip',
      models: ['gpt-test', 'gpt-missing'],
    })

    expect(post).toHaveBeenCalledWith('/admin/upstreams/7/models/probe', {
      platform: 'openai',
      group_name: 'vip',
      models: ['gpt-test', 'gpt-missing'],
    }, { timeout: 120_000 })
  })

  it('extends model probe timeout by backend concurrency waves', async () => {
    await upstreamsAPI.probeModels(7, {
      platform: 'openai',
      group_name: 'vip',
      models: ['gpt-1', 'gpt-2', 'gpt-3', 'gpt-4'],
    })

    expect(post).toHaveBeenCalledWith('/admin/upstreams/7/models/probe', expect.any(Object), {
      timeout: 210_000,
    })
  })

  it('deletes an account when unbinding by default and can explicitly preserve it', async () => {
    await upstreamsAPI.unbindAccounts(7, [11])
    await upstreamsAPI.unbindAccounts(7, [12], false)

    expect(post).toHaveBeenNthCalledWith(1, '/admin/upstreams/7/unbind', {
      account_ids: [11],
      delete_accounts: true,
    })
    expect(post).toHaveBeenNthCalledWith(2, '/admin/upstreams/7/unbind', {
      account_ids: [12],
      delete_accounts: false,
    })
  })

  it('changes a bound account upstream group through the managed transaction endpoint', async () => {
    put.mockResolvedValueOnce({ data: { account: { id: 11 }, models: ['gpt-test'] } })

    await upstreamsAPI.changeAccountUpstreamGroup(7, 11, { group_name: 'vip', group_id: 3 })

    expect(put).toHaveBeenCalledWith('/admin/upstreams/7/accounts/11/upstream-group', {
      group_name: 'vip',
      group_id: 3,
    })
  })

  it('previews automatic account names before applying them', async () => {
    post
      .mockResolvedValueOnce({ data: { renames: 1, skips: 1, items: [] } })
      .mockResolvedValueOnce({ data: { renamed: 1, skipped: 1, failed: 0, items: [] } })

    await upstreamsAPI.previewAccountRenames()
    await upstreamsAPI.applyAccountRenames()

    expect(post).toHaveBeenNthCalledWith(1, '/admin/upstreams/accounts/rename-preview')
    expect(post).toHaveBeenNthCalledWith(2, '/admin/upstreams/accounts/rename-apply')
  })

  it('reads persisted managed rates and refreshes only on an explicit action', async () => {
    get.mockResolvedValueOnce({ data: [{ account_id: 11, status: 'ok', cached: true }] })
    post.mockResolvedValueOnce({ data: [{ account_id: 11, status: 'ok', cached: true }] })

    await upstreamsAPI.getManagedAccountStatuses([11, 12])
    await upstreamsAPI.refreshManagedAccountStatuses([11])

    expect(get).toHaveBeenCalledWith('/admin/upstreams/account-status', {
      params: { account_ids: '11,12' },
    })
    expect(post).toHaveBeenCalledWith('/admin/upstreams/account-status/refresh', {
      account_ids: [11],
    })
  })

  it('batches managed account status requests at the server limit', async () => {
    const ids = Array.from({ length: 501 }, (_, index) => index + 1)
    get
      .mockResolvedValueOnce({ data: [{ account_id: 1, status: 'ok', cached: true }] })
      .mockResolvedValueOnce({ data: [{ account_id: 501, status: 'ok', cached: true }] })
    post
      .mockResolvedValueOnce({ data: [{ account_id: 1, status: 'ok', cached: true }] })
      .mockResolvedValueOnce({ data: [{ account_id: 501, status: 'ok', cached: true }] })

    const cached = await upstreamsAPI.getManagedAccountStatuses(ids)
    const refreshed = await upstreamsAPI.refreshManagedAccountStatuses(ids)

    expect(cached.map(item => item.account_id)).toEqual([1, 501])
    expect(refreshed.map(item => item.account_id)).toEqual([1, 501])
    expect(get).toHaveBeenNthCalledWith(1, '/admin/upstreams/account-status', {
      params: { account_ids: ids.slice(0, 500).join(',') },
    })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/upstreams/account-status', {
      params: { account_ids: '501' },
    })
    expect(post).toHaveBeenNthCalledWith(1, '/admin/upstreams/account-status/refresh', {
      account_ids: ids.slice(0, 500),
    })
    expect(post).toHaveBeenNthCalledWith(2, '/admin/upstreams/account-status/refresh', {
      account_ids: [501],
    })
  })
})
