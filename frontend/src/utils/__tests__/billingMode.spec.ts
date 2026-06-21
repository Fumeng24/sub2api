import { describe, expect, it } from 'vitest'

import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_TOKEN,
  getDisplayBillingMode,
  isImageUsage,
} from '../billingMode'

describe('billingMode helpers', () => {
  it('treats historical image rows without billing_mode as image usage', () => {
    const row = {
      billing_mode: null,
      image_count: 2,
    }

    expect(getDisplayBillingMode(row)).toBe(BILLING_MODE_IMAGE)
    expect(isImageUsage(row)).toBe(true)
  })

  it('keeps explicit token rows as token even when image_count is present', () => {
    const row = {
      billing_mode: BILLING_MODE_TOKEN,
      image_count: 2,
    }

    expect(getDisplayBillingMode(row)).toBe(BILLING_MODE_TOKEN)
    expect(isImageUsage(row)).toBe(false)
  })

  it('does not infer image mode when there are no images', () => {
    const row = {
      billing_mode: null,
      image_count: 0,
    }

    expect(getDisplayBillingMode(row)).toBeNull()
    expect(isImageUsage(row)).toBe(false)
  })
})
