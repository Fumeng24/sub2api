import { describe, expect, it } from 'vitest'

import {
  buildModelsListConfig,
  createModelsListState,
  setModelsListCandidates,
} from '@/custom/admin/groups/groupsModelsList'

describe('Wegoo group model list policy', () => {
  it('drops saved models that group accounts no longer support', () => {
    const state = createModelsListState({
      enabled: true,
      models: ['gpt-5.5', 'stale-pro-model'],
    })

    setModelsListCandidates(state, ['gpt-5.5', 'gpt-5.4'])

    expect(state.items).toEqual([
      { id: 'gpt-5.5', selected: true },
      { id: 'gpt-5.4', selected: false },
    ])
    expect(buildModelsListConfig(state)).toEqual({
      enabled: true,
      models: ['gpt-5.5'],
    })
  })

  it('persists an explicit empty list after account candidates load empty', () => {
    const state = createModelsListState({
      enabled: true,
      models: ['stale-pro-model'],
    })

    setModelsListCandidates(state, [])

    expect(state.items).toEqual([])
    expect(buildModelsListConfig(state)).toEqual({
      enabled: true,
      models: [],
    })
  })
})
