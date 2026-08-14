import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountStatusIndicator from '@/custom/admin/account/WegooAccountStatusIndicator.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

function makeAccount(overrides: Partial<Account>): Account {
  return {
    id: 1,
    name: 'account',
    platform: 'antigravity',
    type: 'oauth',
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-03-15T00:00:00Z',
    updated_at: '2026-03-15T00:00:00Z',
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
  }
}

describe('Wegoo account status priority', () => {
  it('shows insufficient balance before the generic paused state', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          status: 'active',
          schedulable: false,
          error_message: 'Manual disabled 2026-06-22: upstream 403 insufficient balance',
        }),
      },
      global: { stubs: { Icon: true } },
    })

    expect(wrapper.text()).toContain('admin.accounts.status.insufficientBalance')
    expect(wrapper.text()).not.toContain('admin.accounts.status.paused')
  })
})
