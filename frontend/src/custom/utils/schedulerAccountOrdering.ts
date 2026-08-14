import type { AccountSchedulingEntry } from '@/types'
import type { AccountMonitorStatus } from '@/custom/api/admin/accountMonitor'

export const SCHEDULER_RANK_HEALTHY = 0
export const SCHEDULER_RANK_PROBATION = 1
export const SCHEDULER_RANK_DEGRADED = 2
export const SCHEDULER_RANK_FAILED = 3
export const SCHEDULER_RANK_UNSCHEDULABLE = 4

function isFutureTime(value: string | null | undefined, nowMs: number): boolean {
  if (!value) return false
  const ts = new Date(value).getTime()
  return Number.isFinite(ts) && ts > nowMs
}

function isRuntimeUnavailable(entry: AccountSchedulingEntry, nowMs: number): boolean {
  if (entry.block_reason) return true
  const account = entry.account
  if (!account) return true
  if (account.status !== 'active') return true
  if (account.schedulable === false) return true
  if (isFutureTime(account.temp_unschedulable_until, nowMs)) return true
  if (isFutureTime(account.rate_limit_reset_at, nowMs)) return true
  if (isFutureTime(account.overload_until, nowMs)) return true
  return false
}

export function schedulerAccountHealthRank(
  entry: AccountSchedulingEntry,
  status?: AccountMonitorStatus,
  nowMs = Date.now(),
): number {
  if (isRuntimeUnavailable(entry, nowMs)) return SCHEDULER_RANK_UNSCHEDULABLE

  switch (status?.latest_status) {
    case 'operational':
      return SCHEDULER_RANK_HEALTHY
    // A missing or not-yet-classified probe is not proof of health. Keep it in
    // probation so a fresh account can collect samples without displacing a
    // proven stable account. The backend experience sorter applies the same
    // policy when it has real user-request statistics.
    case '' :
    case undefined:
      return SCHEDULER_RANK_PROBATION
    case 'degraded':
      return SCHEDULER_RANK_DEGRADED
    case 'failed':
    case 'error':
    default:
      return SCHEDULER_RANK_FAILED
  }
}

export function compareSchedulerEntries(
  a: AccountSchedulingEntry,
  b: AccountSchedulingEntry,
  statusForEntry: (entry: AccountSchedulingEntry) => AccountMonitorStatus | undefined = () => undefined,
  nowMs = Date.now(),
): number {
  const rankA = schedulerAccountHealthRank(a, statusForEntry(a), nowMs)
  const rankB = schedulerAccountHealthRank(b, statusForEntry(b), nowMs)
  if (rankA !== rankB) return rankA - rankB

  // Stability and model success are the canonical policy. Those metrics are
  // intentionally calculated by the backend from real requests and persisted
  // as the group's sort_order. The UI must not invent a second score from
  // stale monitor/rate data; use the persisted order as the deterministic
  // tie-breaker while the backend refreshes the metrics.
  return comparePersistedSchedulerEntries(a, b)
}

/**
 * Compare entries by the order persisted on the selected group.
 * Account.priority is a legacy global field and can differ for the same
 * account in another group, so it must not override group sort_order here.
 */
export function comparePersistedSchedulerEntries(
  a: AccountSchedulingEntry,
  b: AccountSchedulingEntry,
): number {
  // Zero is an explicit and valid persisted position. Only nullish/blank or
  // non-finite values are considered absent; truthiness/`> 0` would silently
  // discard the first account returned by account_groups.
  const persistedOrder = (value: unknown): number => {
    if (value === null || value === undefined || value === '') return Number.POSITIVE_INFINITY
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : Number.POSITIVE_INFINITY
  }
  const orderA = persistedOrder(a.sort_order)
  const orderB = persistedOrder(b.sort_order)
  if (orderA !== orderB) return orderA - orderB

  // Role is metadata for failover semantics, not a ranking criterion. A
  // persisted sort_order must remain visible even when a backup is deliberately
  // placed ahead of a primary by the canonical backend policy.
  return Number(a.account_id) - Number(b.account_id)
}

export function compareSchedulerAvailability(
  a: AccountSchedulingEntry,
  b: AccountSchedulingEntry,
  statusForEntry: (entry: AccountSchedulingEntry) => AccountMonitorStatus | undefined,
  nowMs = Date.now(),
): number {
  // Kept as a compatibility alias for callers outside the scheduler page.
  // Availability alone is not a supported ordering policy anymore: the
  // backend's stability/model-success score is authoritative for every group.
  return compareSchedulerEntries(a, b, statusForEntry, nowMs)
}
