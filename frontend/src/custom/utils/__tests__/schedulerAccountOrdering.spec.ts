import { describe, expect, it } from 'vitest'

import type { AccountSchedulingEntry } from '@/types'
import type { AccountMonitorStatus } from '@/custom/api/admin/accountMonitor'
import {
  comparePersistedSchedulerEntries,
  compareSchedulerAvailability,
  compareSchedulerEntries,
} from '../schedulerAccountOrdering'

function entry(
  id: number,
  overrides: Partial<NonNullable<AccountSchedulingEntry['account']>> = {},
): AccountSchedulingEntry {
  return {
    account_id: id,
    group_id: 1,
    role: 'primary',
    weight: 1,
    sort_order: 0,
    account: {
      id,
      name: `account-${id}`,
      platform: 'openai',
      type: 'apikey',
      credentials: {},
      extra: {},
      proxy_id: null,
      concurrency: 1,
      priority: 10,
      status: 'active',
      error_message: null,
      last_used_at: null,
      expires_at: null,
      auto_pause_on_expired: false,
      created_at: '2026-06-28T00:00:00Z',
      updated_at: '2026-06-28T00:00:00Z',
      schedulable: true,
      rate_limited_at: null,
      rate_limit_reset_at: null,
      overload_until: null,
      temp_unschedulable_until: null,
      temp_unschedulable_reason: null,
      session_window_start: null,
      session_window_end: null,
      session_window_status: null,
      ...overrides,
    },
  } as AccountSchedulingEntry
}

function status(accountId: number, latestStatus: AccountMonitorStatus['latest_status'], availability = 100): AccountMonitorStatus {
  return {
    monitor_id: accountId,
    account_id: accountId,
    model: 'gpt-5.4-mini',
    enabled: true,
    latest_status: latestStatus,
    latest_latency_ms: 120,
    ping_latency_ms: 80,
    availability_1h: availability,
    last_checked_at: '2026-06-28T00:00:00Z',
    timeline: [],
  }
}

describe('scheduler account ordering', () => {
  it('does not prefer oauth over an equally healthy account', () => {
    const now = Date.parse('2026-06-28T00:00:00Z')
    const healthyApiKey = entry(1, { type: 'apikey' })
    const blockedOAuth = entry(2, {
      type: 'oauth',
      temp_unschedulable_until: '2026-06-28T00:05:00Z',
    })
    const healthyOAuth = entry(3, { type: 'oauth' })

    const sorted = [blockedOAuth, healthyApiKey, healthyOAuth]
      .sort((a, b) => compareSchedulerEntries(a, b, () => undefined, now))
      .map((item) => item.account_id)

    expect(sorted).toEqual([1, 3, 2])
  })

  it('keeps failed monitored accounts below healthy accounts in the canonical comparator', () => {
    const now = Date.parse('2026-06-28T00:00:00Z')
    const healthy = entry(1, { type: 'apikey' })
    const failedOAuth = entry(2, { type: 'oauth' })
    const degraded = entry(3, { type: 'apikey' })
    const statuses = new Map<number, AccountMonitorStatus>([
      [1, status(1, 'operational', 92)],
      [2, status(2, 'failed', 100)],
      [3, status(3, 'degraded', 99)],
    ])

    const sorted = [failedOAuth, degraded, healthy]
      .sort((a, b) => compareSchedulerAvailability(a, b, (item) => statuses.get(item.account_id), now))
      .map((item) => item.account_id)

    expect(sorted).toEqual([1, 3, 2])
  })

  it('uses backend block_reason before local account field inference', () => {
    const now = Date.parse('2026-06-28T00:00:00Z')
    const backendBlocked = { ...entry(1), block_reason: 'quota_exceeded' }
    const healthy = entry(2)

    const sorted = [backendBlocked, healthy]
      .sort((a, b) => compareSchedulerEntries(a, b, () => undefined, now))
      .map((item) => item.account_id)

    expect(sorted).toEqual([2, 1])
  })

  it('keeps the selected group sort order ahead of global account priority', () => {
    const first = entry(1, { priority: 99 })
    first.sort_order = 20
    const second = entry(2, { priority: 1 })
    second.sort_order = 10

    const sorted = [first, second]
      .sort(comparePersistedSchedulerEntries)
      .map((item) => item.account_id)

    expect(sorted).toEqual([2, 1])
  })

  it('keeps backup entries after primary entries when group order ties', () => {
    const backup = entry(1)
    backup.role = 'backup'
    backup.sort_order = 10
    const primary = entry(2)
    primary.sort_order = 10

    expect([backup, primary].sort(comparePersistedSchedulerEntries).map((item) => item.account_id))
      .toEqual([1, 2])
  })

  it('preserves an explicit group order across primary and backup roles', () => {
    const backup = entry(1)
    backup.role = 'backup'
    backup.sort_order = 10
    const primary = entry(2)
    primary.sort_order = 20

    expect([primary, backup].sort(comparePersistedSchedulerEntries).map((item) => item.account_id))
      .toEqual([1, 2])
  })

  it('treats zero as an explicit persisted position and only uses account id as tie-breaker', () => {
    const first = entry(20)
    first.sort_order = 0
    const second = entry(10)
    second.sort_order = 1

    expect([second, first].sort(comparePersistedSchedulerEntries).map((item) => item.account_id))
      .toEqual([20, 10])

    const tiedHigh = entry(30)
    tiedHigh.sort_order = 5
    const tiedLow = entry(5)
    tiedLow.sort_order = 5
    expect([tiedHigh, tiedLow].sort(comparePersistedSchedulerEntries).map((item) => item.account_id))
      .toEqual([5, 30])
  })

  it('treats a missing monitor result as probation, not healthy', () => {
    const now = Date.parse('2026-06-28T00:00:00Z')
    const fresh = entry(1)
    const proven = entry(2)
    const statuses = new Map<number, AccountMonitorStatus>([
      [2, status(2, 'operational', 100)],
    ])

    const sorted = [fresh, proven]
      .sort((a, b) => compareSchedulerEntries(a, b, (item) => statuses.get(item.account_id), now))
      .map((item) => item.account_id)

    expect(sorted).toEqual([2, 1])
  })
})
