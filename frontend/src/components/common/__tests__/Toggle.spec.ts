import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Toggle from '../Toggle.vue'

describe('Toggle', () => {
  it('exposes switch semantics and emits the requested next value', async () => {
    const wrapper = mount(Toggle, {
      props: { modelValue: false },
    })
    const button = wrapper.get('button')

    expect(button.attributes('role')).toBe('switch')
    expect(button.attributes('aria-checked')).toBe('false')

    await button.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])
  })

  it('does not emit while disabled', async () => {
    const wrapper = mount(Toggle, {
      props: { modelValue: true, disabled: true },
    })
    const button = wrapper.get('button')

    expect(button.attributes('disabled')).toBeDefined()
    expect(button.attributes('aria-disabled')).toBe('true')

    await button.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})
