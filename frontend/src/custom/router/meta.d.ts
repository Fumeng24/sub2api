import 'vue-router'
import type { RegisteredFeatureFlag } from '@/utils/featureFlags'

declare module 'vue-router' {
  interface RouteMeta {
    requiresSupport?: boolean

    seoTitle?: string
    description?: string
    robots?: string
    canonicalPath?: string
    ogType?: string
    ogImage?: string

    requiresPurchaseAvailable?: boolean
    featureFlag?: RegisteredFeatureFlag
    requiresSubscriptionGroups?: boolean
  }
}
