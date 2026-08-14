import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { getAvailableModels } from '@/custom/api/admin/accounts'

describe('admin accounts api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('normalizes wrapped account model payloads into selectable models', async () => {
    get.mockResolvedValue({
      data: {
        data: {
          models: [
            { id: 'gpt-5.5', displayName: 'GPT-5.5' },
            { value: 'gpt-5.4', label: 'GPT-5.4' },
            'gpt-5.4',
          ],
        },
      },
    })

    await expect(getAvailableModels(42)).resolves.toEqual([
      {
        id: 'gpt-5.5',
        type: 'model',
        display_name: 'GPT-5.5',
        created_at: '',
      },
      {
        id: 'gpt-5.4',
        type: 'model',
        display_name: 'GPT-5.4',
        created_at: '',
      },
    ])
    expect(get).toHaveBeenCalledWith('/admin/accounts/42/models')
  })
})
