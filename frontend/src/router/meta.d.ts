/**
 * Type definitions for Vue Router meta fields
 * Extends the RouteMeta interface with custom properties
 */

import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    /**
     * Whether this route requires authentication
     * @default true
     */
    requiresAuth?: boolean

    /**
     * Whether this route requires admin role
     * @default false
     */
    requiresAdmin?: boolean

    /**
     * Whether this route requires support/admin ticket access
     * @default false
     */
    requiresSupport?: boolean

    /**
     * Page title for this route
     */
    title?: string

    /**
     * Search/share title. Falls back to title/titleKey when omitted.
     */
    seoTitle?: string

    /**
     * Search/share description. Falls back to descriptionKey/default site description when omitted.
     */
    description?: string

    /**
     * Search engine robots directive, for example "index,follow" or "noindex,nofollow".
     */
    robots?: string

    /**
     * Canonical path without host. Useful for aliases and redirects.
     */
    canonicalPath?: string

    /**
     * Open Graph type for social previews.
     * @default website
     */
    ogType?: string

    /**
     * Open Graph/Twitter preview image.
     */
    ogImage?: string

    /**
     * Optional breadcrumb items for navigation
     */
    breadcrumbs?: Array<{
      label: string
      to?: string
    }>

    /**
     * Icon name for this route (for sidebar navigation)
     */
    icon?: string

    /**
     * Whether to hide this route from navigation menu
     * @default false
     */
    hideInMenu?: boolean

    /**
     * Whether this route requires internal payment system to be enabled
     * @default false
     */
    requiresPayment?: boolean

    /**
     * 是否要求风控中心功能开关已启用
     * @default false
     */
    requiresRiskControl?: boolean

    /**
     * i18n key for the page title
     */
    titleKey?: string

    /**
     * i18n key for the page description
     */
    descriptionKey?: string
  }
}
