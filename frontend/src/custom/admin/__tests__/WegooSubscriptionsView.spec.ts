import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const srcRoot = resolve(__dirname, '../../..')
const readSource = (file: string) => readFileSync(resolve(srcRoot, file), 'utf8')

describe('WegooSubscriptionsView overlay', () => {
  it('keeps site subscription controls and route wiring outside the official view', () => {
    const view = readSource('custom/admin/WegooSubscriptionsView.vue')
    const router = readSource('custom/router/index.ts')

    expect(view).toContain("import { resetSubscriptionWithCost } from '@/custom/api/admin/subscriptions'")
    expect(view).toContain("import '@/custom/admin/adminApple.css'")
    expect(view).toContain('function canResetWithCost(')
    expect(view).toContain('async function confirmResetWithCost()')
    expect(router).toContain(
      "component: () => import('@/custom/admin/WegooSubscriptionsView.vue')",
    )
  })
})
