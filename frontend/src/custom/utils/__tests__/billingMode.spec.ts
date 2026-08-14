import { describe, expect, it } from 'vitest'

import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
  BILLING_MODE_VIDEO,
  getDisplayBillingMode,
  getBillingModeLabel,
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

  it('keeps explicit video rows as video even when the legacy image counter is present', () => {
    const row = {
      billing_mode: BILLING_MODE_VIDEO,
      image_count: 1,
    }

    expect(getDisplayBillingMode(row)).toBe(BILLING_MODE_VIDEO)
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

  it('uses the admin usage namespace by default for billing labels', () => {
    const t = (key: string) => key

    expect(getBillingModeLabel(BILLING_MODE_PER_REQUEST, t)).toBe('admin.usage.billingModePerRequest')
    expect(getBillingModeLabel(BILLING_MODE_IMAGE, t)).toBe('admin.usage.billingModeImage')
    expect(getBillingModeLabel(BILLING_MODE_VIDEO, t)).toBe('admin.usage.billingModeVideo')
    expect(getBillingModeLabel(BILLING_MODE_TOKEN, t)).toBe('admin.usage.billingModeToken')
  })

  it('allows user-facing billing labels to use the usage namespace', () => {
    const t = (key: string) => key

    expect(getBillingModeLabel(BILLING_MODE_PER_REQUEST, t, 'usage')).toBe('usage.billingModePerRequest')
    expect(getBillingModeLabel(BILLING_MODE_IMAGE, t, 'usage')).toBe('usage.billingModeImage')
    expect(getBillingModeLabel(BILLING_MODE_VIDEO, t, 'usage')).toBe('usage.billingModeVideo')
    expect(getBillingModeLabel(BILLING_MODE_TOKEN, t, 'usage')).toBe('usage.billingModeToken')
  })
})
