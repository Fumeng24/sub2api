import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { createPinia } from 'pinia'
import AvailableChannelsTable from '@/custom/channels/WegooAvailableChannelsTable.vue'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN } from '@/constants/channel'
import type { UserAvailableChannel } from '@/api/channels'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const rows: UserAvailableChannel[] = [
  {
    name: 'Gateway Core',
    description: 'Public model catalog',
    platforms: [
      {
        platform: 'openai',
        endpoints: ['Responses', 'Images'],
        supported_endpoint_types: ['responses'],
        supported_models: [],
        groups: [
          {
            id: 7,
            name: 'Codex Pro',
            platform: 'openai',
            subscription_type: 'standard',
            rate_multiplier: 0.15,
            discounted_rate_multiplier: null,
            is_exclusive: false,
            supported_models: [
              {
                name: 'gpt-5.5-codex',
                platform: 'openai',
                pricing: {
                  billing_mode: BILLING_MODE_TOKEN,
                  input_price: 0.000001,
                  output_price: 0.000008,
                  cache_write_price: 0.0000005,
                  cache_read_price: 0.0000001,
                  image_output_price: null,
                  per_request_price: null,
                  intervals: [],
                },
              },
            ],
          },
        ],
      },
    ],
  },
]

function mountTable(inputRows: UserAvailableChannel[] = rows) {
  return mount(AvailableChannelsTable, {
    props: {
      rows: inputRows,
      loading: false,
      pricingKeyPrefix: 'availableChannels.pricing',
      noPricingLabel: 'Backend configured',
      noModelsLabel: 'No models',
      emptyLabel: 'Empty',
      userGroupRates: {},
    },
    global: {
      plugins: [createPinia()],
      stubs: {
        Icon: true,
        PlatformIcon: true,
        GroupBadge: true,
        SupportedModelChip: true,
      },
    },
  })
}

describe('AvailableChannelsTable', () => {
  it('renders the selected group model pricing matrix', () => {
    const wrapper = mountTable()

    expect(wrapper.text()).toContain('Codex Pro')
    expect(wrapper.text()).toContain('0.15x')

    const modelTable = wrapper.get('[data-testid="available-channels-model-table"]')
    expect(modelTable.text()).toContain('Gateway Core')
    expect(modelTable.text()).toContain('Responses')
    expect(modelTable.text()).toContain('Images')
    expect(modelTable.text()).toContain('gpt-5.5-codex')
    expect(modelTable.text()).toContain('availableChannels.pricing.billingModeToken')
    expect(modelTable.text()).toContain('¥0.15')
    expect(modelTable.text()).toContain('¥1.2')
    expect(modelTable.text()).toContain('¥0.075')
    expect(modelTable.text()).toContain('¥0.015')

    const mobileCatalog = wrapper.get('[data-testid="available-channels-mobile"]')
    expect(mobileCatalog.text()).toContain('OpenAI')
    expect(mobileCatalog.text()).toContain('gpt-5.5-codex')
    expect(mobileCatalog.text()).toContain('¥0.15')
    expect(mobileCatalog.text()).toContain('¥1.2')
    expect(mobileCatalog.text()).toContain('¥0.075')
    expect(mobileCatalog.text()).toContain('¥0.015')
  })

  it('orders service tiers by vendor and then rate', () => {
    const sortedRows: UserAvailableChannel[] = [{
      ...rows[0],
      platforms: [
        {
          ...rows[0].platforms[0],
          platform: 'gemini',
          groups: [
            {
              ...rows[0].platforms[0].groups[0],
              id: 30,
              name: 'Gemini Plus',
              platform: 'gemini',
              rate_multiplier: 0.4,
            },
            {
              ...rows[0].platforms[0].groups[0],
              id: 31,
              name: 'Gemini生图[1k]',
              platform: 'gemini',
              rate_multiplier: 0.1,
              supported_models: [{
                ...rows[0].platforms[0].groups[0].supported_models![0],
                name: 'imagen-4',
                platform: 'gemini',
                pricing: {
                  billing_mode: BILLING_MODE_IMAGE,
                  input_price: null,
                  output_price: null,
                  cache_write_price: null,
                  cache_read_price: null,
                  image_output_price: 0.1,
                  per_request_price: null,
                  intervals: [],
                },
              }],
            },
          ],
        },
        {
          ...rows[0].platforms[0],
          groups: [
            {
              ...rows[0].platforms[0].groups[0],
              id: 20,
              name: 'Codex Pro',
              platform: 'openai',
              rate_multiplier: 0.3,
            },
            {
              ...rows[0].platforms[0].groups[0],
              id: 21,
              name: 'Codex Lite',
              platform: 'openai',
              rate_multiplier: 0.1,
            },
            {
              ...rows[0].platforms[0].groups[0],
              id: 22,
              name: 'GPT生图[1.5k]',
              platform: 'openai',
              rate_multiplier: 0.03,
              supported_models: [{
                ...rows[0].platforms[0].groups[0].supported_models![0],
                name: 'gpt-image-1',
                platform: 'openai',
                pricing: {
                  billing_mode: BILLING_MODE_IMAGE,
                  input_price: null,
                  output_price: null,
                  cache_write_price: null,
                  cache_read_price: null,
                  image_output_price: 0.03,
                  per_request_price: null,
                  intervals: [],
                },
              }],
            },
          ],
        },
        {
          ...rows[0].platforms[0],
          platform: 'anthropic',
          groups: [{
            ...rows[0].platforms[0].groups[0],
            id: 10,
            name: 'Claude Plus',
            platform: 'anthropic',
            rate_multiplier: 0.3,
          }],
        },
      ],
    }]

    const wrapper = mountTable(sortedRows)

    expect(wrapper.findAll('.available-group-tab__name').map((node) => node.text())).toEqual([
      'Claude Plus',
      'Codex Lite',
      'Codex Pro',
      'Gemini Plus',
    ])
  })

  it('keeps normal Gemini tiers when they contain image models', () => {
    const mixedRows: UserAvailableChannel[] = [{
      ...rows[0],
      platforms: [{
        ...rows[0].platforms[0],
        platform: 'gemini',
        groups: [{
          ...rows[0].platforms[0].groups[0],
          id: 30,
          name: 'Gemini Plus',
          platform: 'gemini',
          supported_models: [
            {
              ...rows[0].platforms[0].groups[0].supported_models![0],
              name: 'gemini-2.5-pro',
              platform: 'gemini',
            },
            {
              ...rows[0].platforms[0].groups[0].supported_models![0],
              name: 'imagen-4',
              platform: 'gemini',
              pricing: {
                billing_mode: BILLING_MODE_IMAGE,
                input_price: null,
                output_price: null,
                cache_write_price: null,
                cache_read_price: null,
                image_output_price: 0.1,
                per_request_price: null,
                intervals: [],
              },
            },
          ],
        }],
      }],
    }]

    const wrapper = mountTable(mixedRows)

    expect(wrapper.text()).toContain('Gemini Plus')
    expect(wrapper.text()).toContain('gemini-2.5-pro')
    expect(wrapper.text()).not.toContain('imagen-4')
  })
})
