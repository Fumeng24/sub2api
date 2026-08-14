import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { listUsers, getUser } = vi.hoisted(() => ({
  listUsers: vi.fn(),
  getUser: vi.fn(),
}))

vi.mock('@/custom/api/admin', () => ({
  adminAPI: {
    users: {
      list: listUsers,
      getById: getUser,
    },
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, values?: Record<string, unknown>) =>
        values ? `${key}:${JSON.stringify(values)}` : key,
    }),
  }
})

import WegooAnnouncementUserSelector from '../WegooAnnouncementUserSelector.vue'

describe('WegooAnnouncementUserSelector', () => {
  beforeEach(() => {
    listUsers.mockReset()
    getUser.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('resolves existing recipients and removes them', async () => {
    getUser.mockResolvedValue({ id: 42, email: 'alice@example.com', username: 'alice' })

    const wrapper = mount(WegooAnnouncementUserSelector, {
      props: { modelValue: [42] },
      global: { stubs: { Icon: true } },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('alice@example.com')
    await wrapper.get('button[aria-label="admin.announcements.form.removeUser"]').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([[[]]])
  })

  it('searches users and emits the selected user id', async () => {
    listUsers.mockResolvedValue({
      items: [{ id: 7, email: 'bob@example.com', username: 'bob' }],
      total: 1,
    })

    vi.useFakeTimers()
    const wrapper = mount(WegooAnnouncementUserSelector, {
      props: { modelValue: [] },
      global: { stubs: { Icon: true } },
    })

    await wrapper.get('input[type="search"]').setValue('bob')
    vi.advanceTimersByTime(250)
    await flushPromises()

    expect(listUsers).toHaveBeenCalledWith(1, 20, {
      search: 'bob',
      sort_by: 'id',
      sort_order: 'asc',
    })
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[[7]]])
  })
})
