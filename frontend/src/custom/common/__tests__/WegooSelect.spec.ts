import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import WegooSelect from '../WegooSelect.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

describe('WegooSelect touch interaction', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 768 })
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn((query: string) => ({
        matches: query === '(pointer: coarse)',
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn()
      }))
    })
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      x: 16,
      y: 700,
      top: 700,
      right: 374,
      bottom: 744,
      left: 16,
      width: 358,
      height: 44,
      toJSON: () => ({})
    } as DOMRect)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('keeps a long mobile dropdown scrollable and its options selectable', async () => {
    const options = Array.from({ length: 12 }, (_, index) => ({
      value: `group-${index + 1}`,
      label: `Group ${index + 1}`
    }))
    const wrapper = mount(WegooSelect, {
      attachTo: document.body,
      props: {
        modelValue: null,
        options,
        searchable: true
      },
      global: {
        stubs: { Icon: { template: '<span />' } }
      }
    })

    await wrapper.get('.select-trigger').trigger('click')
    await nextTick()
    await nextTick()

    const dropdown = document.body.querySelector<HTMLElement>('.select-dropdown-portal')
    const search = document.body.querySelector<HTMLInputElement>('.select-search-input')
    const optionList = document.body.querySelector<HTMLElement>('.select-options')
    const optionNodes = document.body.querySelectorAll<HTMLElement>('.select-option')

    expect(dropdown).not.toBeNull()
    expect(dropdown?.style.maxHeight).toBe('320px')
    expect(dropdown?.style.bottom).not.toBe('')
    expect(document.activeElement).not.toBe(search)
    expect(optionNodes).toHaveLength(options.length)

    optionList!.scrollTop = 120
    optionNodes[optionNodes.length - 1].dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()

    expect(wrapper.emitted('update:modelValue')).toEqual([['group-12']])
    expect(wrapper.get('.select-trigger').attributes('aria-expanded')).toBe('false')
    wrapper.unmount()
  })
})
