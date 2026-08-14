import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(__dirname, '../WegooKeysView.vue'),
  'utf8',
)

describe('KeysView gateway access flow', () => {
  it('opens the access details dialog immediately after creating a key', () => {
    expect(source).toContain('const createdKey = await keysAPI.create')
    expect(source).toContain('selectedKey.value = createdKey')
    expect(source).toContain('showUseKeyModal.value = true')
    expect(source).toContain('if (!showUseKeyModal.value) {\n    selectedKey.value = null\n  }')
  })

  it('keeps the create group selector wired to group_id', () => {
    expect(source).toContain('v-model="formData.group_id"')
    expect(source).toContain(':options="groupOptions"')
    expect(source).toContain('const rawGroupOptions = computed<GroupOption[]>')
    expect(source).toContain('group_id: formData.value.group_id')
  })
})
