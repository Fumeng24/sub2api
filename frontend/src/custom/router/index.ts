/**
 * Vue Router configuration for Sub2API frontend
 * Defines all application routes with lazy loading and navigation guards
 */

import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { useAdminComplianceStore } from '@/stores/adminCompliance'
import { usePaymentCheckoutStore } from '@/custom/stores/paymentCheckout'
import { useSubscriptionCapabilityStore } from '@/custom/stores/subscriptionCapability'
import { useNavigationLoadingState } from '@/composables/useNavigationLoading'
import { useRoutePrefetch } from '@/composables/useRoutePrefetch'
import { getSetupStatus } from '@/api/setup'
import { resolveCompletedSetupRedirectPath } from '@/router/setupRedirect'
import { resolveRouteDocumentTitle } from '@/router/title'
import { applyRouteSeo } from '@/custom/utils/seo'
import { recoverFromChunkLoadError } from '@/custom/utils/chunkRecovery'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

/**
 * Route definitions with lazy loading
 */
const routes: RouteRecordRaw[] = [
  // ==================== Setup Routes ====================
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/custom/setup/WegooSetupWizardView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Setup',
      robots: 'noindex,nofollow'
    }
  },

  // ==================== Public Routes ====================
  {
    path: '/home',
    name: 'Home',
    component: () => import('@/custom/home/WegooHomeView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Home',
      seoTitle: 'AI 模型服务 - GPT Claude Gemini Codex API',
      description:
        'Wegoo AI 提供 GPT/Codex、Claude、Gemini 与 AI 生图服务，模型目录、访问凭证、用量记录、余额和服务状态集中呈现。',
      canonicalPath: '/home'
    }
  },
  {
    path: '/login',
    name: 'Login',
    alias: '/auth',
    component: () => import('@/custom/auth/WegooLoginView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Login',
      titleKey: 'common.login',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/custom/auth/WegooRegisterView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Register',
      titleKey: 'auth.createAccount',
      seoTitle: '注册 Wegoo AI，使用 AI 模型服务',
      description:
        'Wegoo AI 支持 ChatGPT API、Claude API、Gemini API、Codex API 和 AI 生图，访问凭证、服务档位和用量记录集中呈现。',
      canonicalPath: '/register'
    }
  },
  {
    path: '/email-verify',
    name: 'EmailVerify',
    component: () => import('@/custom/auth/WegooEmailVerifyView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Verify Email',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/auth/callback',
    name: 'OAuthCallback',
    alias: '/auth/oauth/callback',
    component: () => import('@/custom/auth/WegooOAuthCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'OAuth Callback',
      titleKey: 'auth.oauthCallbackPageTitle',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/auth/linuxdo/callback',
    name: 'LinuxDoOAuthCallback',
    component: () => import('@/custom/auth/WegooLinuxDoCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'LinuxDo OAuth Callback',
      titleKey: 'auth.linuxdoCallbackPageTitle',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/auth/wechat/callback',
    name: 'WeChatOAuthCallback',
    component: () => import('@/custom/auth/WegooWechatCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'WeChat OAuth Callback',
      titleKey: 'auth.wechatCallbackPageTitle',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/auth/wechat/payment/callback',
    name: 'WeChatPaymentOAuthCallback',
    component: () => import('@/custom/auth/WegooWechatPaymentCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'WeChat Payment Callback',
      titleKey: 'auth.wechatPaymentCallbackPageTitle',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/auth/dingtalk/callback',
    name: 'DingTalkOAuthCallback',
    component: () => import('@/custom/auth/WegooDingTalkCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'DingTalk OAuth Callback',
      titleKey: 'auth.dingtalkCallbackPageTitle',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/auth/dingtalk/email-completion',
    name: 'dingtalk-email-completion',
    component: () => import('@/custom/auth/WegooDingTalkEmailCompletionView.vue'),
    meta: {
      requiresAuth: false,
      title: 'DingTalk Email Completion',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/auth/oidc/callback',
    name: 'OIDCOAuthCallback',
    component: () => import('@/custom/auth/WegooOidcCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'OIDC OAuth Callback',
      titleKey: 'auth.oidcCallbackPageTitle',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/custom/auth/WegooForgotPasswordView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Forgot Password',
      titleKey: 'auth.forgotPasswordTitle',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/reset-password',
    name: 'ResetPassword',
    component: () => import('@/custom/auth/WegooResetPasswordView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Reset Password',
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/key-usage',
    name: 'KeyUsage',
    component: () => import('@/custom/public/WegooKeyUsageView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Key Usage',
      seoTitle: 'AI API Key 用量和余额查询 - Wegoo AI',
      description:
        '快速查询 API Key 余额、Token 消耗和调用记录，适合开发者、高频用户和团队核对 GPT、Claude、Gemini API 使用成本。',
      canonicalPath: '/key-usage'
    }
  },
  {
    path: '/pricing',
    name: 'PublicModels',
    component: () => import('@/custom/public/ModelsView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Models',
      seoTitle: 'AI 模型价格和公开分组 - Wegoo AI',
      description:
        '查看 Wegoo AI 公开模型、标准分组、端点类型和价格字段。公开页使用后端只读事实源，完整授权分组以控制台为准。',
      canonicalPath: '/pricing'
    }
  },
  {
    path: '/docs',
    name: 'PublicDocs',
    component: () => import('@/custom/public/DocsView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Docs',
      seoTitle: 'Wegoo AI API 接入文档',
      description:
        '快速接入 Wegoo AI Gateway，创建 API Key，替换 Base URL，并使用 OpenAI SDK、Anthropic SDK、Codex CLI 或 Claude Code 调用模型。',
      canonicalPath: '/docs'
    }
  },
  {
    path: '/status',
    name: 'PublicStatus',
    component: () => import('@/custom/public/StatusView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Status',
      seoTitle: 'AI 模型服务状态 - Wegoo AI',
      description:
        '查看 Wegoo AI 用户可见服务状态、模型族可用性、延迟和近期状态趋势。公开页不暴露真实上游账号。',
      canonicalPath: '/status'
    }
  },
  {
    path: '/enterprise',
    name: 'Enterprise',
    component: () => import('@/custom/public/EnterpriseView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Enterprise',
      seoTitle: '企业充值、发票和专属支持 - Wegoo AI',
      description:
        'Wegoo AI 企业服务承接大额充值、人工收款、发票协作、对账和专属支持，余额来源与可开票口径以后台记录为准。',
      canonicalPath: '/enterprise'
    }
  },
  {
    path: '/legal/:documentId',
    name: 'LegalDocument',
    component: () => import('@/custom/public/WegooLegalDocumentView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Legal Document',
      seoTitle: 'Wegoo AI 服务条款与政策',
      description: '查看 Wegoo AI 的服务条款、隐私政策、服务地区声明及相关使用规则。'
    }
  },
  {
    path: '/model-plaza',
    name: 'ModelPlaza',
    component: () => import('@/views/ModelPlazaView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Model Plaza',
      titleKey: 'modelPlaza.title'
    }
  },

  // ==================== User Routes ====================
  {
    path: '/',
    redirect: '/home'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/custom/user/WegooDashboardView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Dashboard',
      titleKey: 'dashboard.title',
      descriptionKey: 'dashboard.welcomeMessage'
    }
  },
  {
    path: '/keys',
    name: 'Keys',
    component: () => import('@/custom/user/WegooKeysView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'API Keys',
      titleKey: 'keys.title',
      descriptionKey: 'keys.description'
    }
  },
  {
    path: '/image-generation',
    name: 'ImageGeneration',
    component: () => import('@/custom/user/ImageGenerationView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Image Generation',
      titleKey: 'imageGeneration.title',
      descriptionKey: 'imageGeneration.description'
    }
  },
  {
    path: '/batch-image',
    name: 'BatchImageGuide',
    alias: '/docs/batch-image',
    component: () => import('@/custom/user/WegooBatchImageGuideView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Batch Image Guide',
      titleKey: 'batchImageGuide.title',
      descriptionKey: 'batchImageGuide.description'
    }
  },
  {
    path: '/usage',
    name: 'Usage',
    component: () => import('@/custom/user/WegooUsageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Usage Records',
      titleKey: 'usage.title',
      descriptionKey: 'usage.description'
    }
  },
  {
    path: '/messages',
    alias: '/announcements',
    name: 'Messages',
    component: () => import('@/custom/user/WegooMessagesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Messages',
      titleKey: 'announcements.title',
      descriptionKey: 'announcements.description'
    }
  },
  {
    path: '/tickets',
    name: 'Tickets',
    component: () => import('@/custom/user/TicketsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Tickets',
      titleKey: 'tickets.title',
      descriptionKey: 'tickets.description'
    }
  },
  {
    path: '/redeem',
    name: 'Redeem',
    component: () => import('@/custom/user/WegooRedeemView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Redeem Code',
      titleKey: 'redeem.title',
      descriptionKey: 'redeem.description'
    }
  },
  {
    path: '/affiliate',
    name: 'Affiliate',
    component: () => import('@/custom/user/WegooAffiliateView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Affiliate',
      titleKey: 'affiliate.title',
      descriptionKey: 'affiliate.description',
      featureFlag: 'affiliate'
    }
  },
  {
    path: '/available-channels',
    name: 'UserAvailableChannels',
    component: () => import('@/custom/user/WegooAvailableChannelsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Available Channels',
      titleKey: 'availableChannels.title',
      descriptionKey: 'availableChannels.description',
      featureFlag: 'availableChannels'
    }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: () => import('@/custom/user/WegooProfileView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Profile',
      titleKey: 'profile.title',
      descriptionKey: 'profile.description'
    }
  },
  {
    path: '/subscriptions',
    name: 'Subscriptions',
    component: () => import('@/custom/user/WegooSubscriptionsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'My Subscriptions',
      titleKey: 'userSubscriptions.title',
      descriptionKey: 'userSubscriptions.description',
      requiresSubscriptionGroups: true
    }
  },
  {
    path: '/payment',
    redirect: (to) => ({
      path: '/purchase',
      query: to.query,
      hash: to.hash
    })
  },
  {
    path: '/purchase',
    name: 'PurchaseSubscription',
    component: () => import('@/custom/user/WegooPaymentView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Purchase Subscription',
      titleKey: 'nav.buySubscription',
      descriptionKey: 'purchase.description',
      requiresPayment: true,
      requiresPurchaseAvailable: true
    }
  },
  {
    path: '/orders',
    name: 'OrderList',
    component: () => import('@/custom/user/WegooUserOrdersView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'My Orders',
      titleKey: 'nav.myOrders',
      requiresPayment: true
    }
  },
  {
    path: '/invoices',
    name: 'Invoices',
    component: () => import('@/custom/user/InvoicesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Invoices',
      titleKey: 'nav.invoices'
    }
  },
  {
    path: '/payment/qrcode',
    name: 'PaymentQRCode',
    component: () => import('@/custom/user/WegooPaymentQRCodeView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Payment',
      titleKey: 'payment.qr.scanToPay',
      requiresPayment: true
    }
  },
  {
    path: '/payment/result',
    name: 'PaymentResult',
    component: () => import('@/custom/user/WegooPaymentResultView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Payment Result',
      titleKey: 'payment.result.success',
      requiresPayment: false,
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/payment/stripe',
    name: 'StripePayment',
    component: () => import('@/custom/user/WegooStripePaymentView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Stripe Payment',
      titleKey: 'payment.stripePay',
      requiresPayment: false,
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/payment/airwallex',
    name: 'AirwallexPayment',
    component: () => import('@/custom/user/WegooAirwallexPaymentView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Airwallex Payment',
      titleKey: 'payment.airwallexPay',
      requiresPayment: false,
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/payment/stripe-popup',
    name: 'StripePopup',
    component: () => import('@/custom/user/WegooStripePopupView.vue'),
    meta: {
      requiresAuth: false,
      requiresAdmin: false,
      title: 'Payment',
      requiresPayment: false,
      robots: 'noindex,nofollow'
    }
  },
  {
    path: '/custom/:id',
    name: 'CustomPage',
    component: () => import('@/custom/user/WegooCustomPageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Custom Page',
      titleKey: 'customPage.title',
    }
  },

  // ==================== Admin Routes ====================
  {
    path: '/admin',
    redirect: '/admin/dashboard'
  },
  {
    path: '/admin/dashboard',
    name: 'AdminDashboard',
    component: () => import('@/custom/admin/WegooDashboardView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Admin Dashboard',
      titleKey: 'admin.dashboard.title',
      descriptionKey: 'admin.dashboard.description'
    }
  },
  {
    path: '/admin/ops',
    name: 'AdminOps',
    component: () => import('@/custom/admin/ops/WegooOpsDashboard.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Ops Monitoring',
      titleKey: 'admin.ops.title',
      descriptionKey: 'admin.ops.description'
    }
  },
  {
    path: '/admin/scheduler',
    name: 'AdminScheduler',
    component: () => import('@/custom/admin/SchedulerView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Scheduler Management',
      titleKey: 'admin.scheduler.title',
      descriptionKey: 'admin.scheduler.description'
    }
  },
  {
    path: '/admin/users',
    name: 'AdminUsers',
    component: () => import('@/custom/admin/WegooUsersView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'User Management',
      titleKey: 'admin.users.title',
      descriptionKey: 'admin.users.description'
    }
  },
  {
    path: '/admin/groups',
    name: 'AdminGroups',
    component: () => import('@/custom/admin/WegooGroupsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Group Management',
      titleKey: 'admin.groups.title',
      descriptionKey: 'admin.groups.description'
    }
  },
  {
    path: '/admin/user-pricing',
    name: 'AdminUserPricing',
    component: () => import('@/custom/admin/WegooUserPricingView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'User Pricing',
      titleKey: 'admin.userPricing.title',
      descriptionKey: 'admin.userPricing.description'
    }
  },
  {
    path: '/admin/channels',
    redirect: '/admin/channels/pricing'
  },
  {
    path: '/admin/channels/pricing',
    name: 'AdminChannels',
    component: () => import('@/custom/admin/WegooChannelsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Channel Management',
      titleKey: 'admin.channels.title',
      descriptionKey: 'admin.channels.description'
    }
  },
  {
    path: '/admin/channels/monitor',
    name: 'AdminChannelMonitor',
    component: () => import('@/custom/admin/WegooChannelMonitorView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Channel Monitor',
      titleKey: 'admin.channelMonitor.title',
      descriptionKey: 'admin.channelMonitor.description',
      featureFlag: 'channelMonitor'
    }
  },
  {
    path: '/monitor',
    name: 'ChannelStatus',
    component: () => import('@/custom/user/WegooChannelStatusView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Channel Status',
      titleKey: 'nav.channelStatus',
      featureFlag: 'channelMonitor'
    }
  },
  {
    path: '/admin/subscriptions',
    name: 'AdminSubscriptions',
    component: () => import('@/custom/admin/WegooSubscriptionsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Subscription Management',
      titleKey: 'admin.subscriptions.title',
      descriptionKey: 'admin.subscriptions.description'
    }
  },
  {
    path: '/admin/accounts',
    name: 'AdminAccounts',
    component: () => import('@/custom/admin/WegooAccountsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Account Management',
      titleKey: 'admin.accounts.title',
      descriptionKey: 'admin.accounts.description'
    }
  },
  {
    path: '/admin/upstreams',
    name: 'AdminUpstreams',
    component: () => import('@/custom/admin/WegooUpstreamsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Upstream Management',
      titleKey: 'admin.upstreams.title',
      descriptionKey: 'admin.upstreams.description'
    }
  },
  {
    path: '/admin/announcements',
    name: 'AdminAnnouncements',
    component: () => import('@/custom/admin/WegooAnnouncementsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Announcements',
      titleKey: 'admin.announcements.title',
      descriptionKey: 'admin.announcements.description'
    }
  },
  {
    path: '/admin/tickets',
    name: 'AdminTickets',
    component: () => import('@/custom/admin/TicketsView.vue'),
    meta: {
      requiresAuth: true,
      requiresSupport: true,
      title: 'Ticket Management',
      titleKey: 'admin.tickets.title',
      descriptionKey: 'admin.tickets.description'
    }
  },
  {
    path: '/admin/proxies',
    name: 'AdminProxies',
    component: () => import('@/custom/admin/WegooProxiesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Proxy Management',
      titleKey: 'admin.proxies.title',
      descriptionKey: 'admin.proxies.description'
    }
  },
  {
    path: '/admin/redeem',
    name: 'AdminRedeem',
    component: () => import('@/custom/admin/WegooRedeemView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Redeem Code Management',
      titleKey: 'admin.redeem.title',
      descriptionKey: 'admin.redeem.description'
    }
  },
  {
    path: '/admin/promo-codes',
    name: 'AdminPromoCodes',
    component: () => import('@/custom/admin/WegooPromoCodesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Promo Code Management',
      titleKey: 'admin.promo.title',
      descriptionKey: 'admin.promo.description'
    }
  },
  {
    path: '/admin/settings',
    name: 'AdminSettings',
    component: () => import('@/custom/admin/WegooSettingsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'System Settings',
      titleKey: 'admin.settings.title',
      descriptionKey: 'admin.settings.description'
    }
  },
  {
    path: '/admin/risk-control',
    name: 'AdminRiskControl',
    component: () => import('@/custom/admin/WegooRiskControlView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Risk Control',
      titleKey: 'admin.riskControl.title',
      descriptionKey: 'admin.riskControl.description',
      requiresRiskControl: true
    }
  },
  {
    path: '/admin/usage',
    name: 'AdminUsage',
    component: () => import('@/custom/admin/WegooUsageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Usage Records',
      titleKey: 'admin.usage.title',
      descriptionKey: 'admin.usage.description'
    }
  },
  {
    path: '/admin/affiliates',
    redirect: '/admin/affiliates/invites'
  },
  {
    path: '/admin/affiliates/invites',
    name: 'AdminAffiliateInvites',
    component: () => import('@/custom/admin/affiliates/WegooAdminAffiliateInvitesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Affiliate Invite Records',
      titleKey: 'nav.affiliateInviteRecords',
      descriptionKey: 'admin.affiliates.invitesDescription',
      featureFlag: 'affiliate'
    }
  },
  {
    path: '/admin/affiliates/rebates',
    name: 'AdminAffiliateRebates',
    component: () => import('@/custom/admin/affiliates/WegooAdminAffiliateRebatesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Affiliate Rebate Records',
      titleKey: 'nav.affiliateRebateRecords',
      descriptionKey: 'admin.affiliates.rebatesDescription',
      featureFlag: 'affiliate'
    }
  },
  {
    path: '/admin/affiliates/transfers',
    name: 'AdminAffiliateTransfers',
    component: () => import('@/custom/admin/affiliates/WegooAdminAffiliateTransfersView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Affiliate Transfer Records',
      titleKey: 'nav.affiliateTransferRecords',
      descriptionKey: 'admin.affiliates.transfersDescription',
      featureFlag: 'affiliate'
    }
  },


  // ==================== Payment Admin Routes ====================
  {
    path: '/admin/orders/dashboard',
    name: 'AdminPaymentDashboard',
    component: () => import('@/custom/admin/orders/WegooAdminPaymentDashboardView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Payment Dashboard',
      titleKey: 'nav.paymentDashboard',
      requiresPayment: true
    }
  },
  {
    path: '/admin/orders',
    name: 'AdminOrders',
    component: () => import('@/custom/admin/orders/WegooAdminOrdersView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Order Management',
      titleKey: 'nav.orderManagement',
      requiresPayment: true
    }
  },
  {
    path: '/admin/orders/plans',
    name: 'AdminPaymentPlans',
    component: () => import('@/custom/admin/orders/WegooAdminPaymentPlansView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Subscription Plans',
      titleKey: 'nav.paymentPlans',
      requiresPayment: true
    }
  },
  {
    path: '/admin/invoices',
    name: 'AdminInvoices',
    component: () => import('@/custom/admin/InvoicesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Invoice Management',
      titleKey: 'nav.invoiceManagement'
    }
  },

  // ==================== 404 Not Found ====================
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/custom/public/WegooNotFoundView.vue'),
    meta: {
      requiresAuth: false,
      title: '404 Not Found',
      robots: 'noindex,nofollow'
    }
  }
]

/**
 * Create router instance
 */
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    // Scroll to saved position when using browser back/forward
    if (savedPosition) {
      return savedPosition
    }
    // Scroll to top for new routes
    return { top: 0 }
  }
})

/**
 * Navigation guard: Authentication check
 */
let authInitialized = false

// 初始化导航加载状态和预加载
const navigationLoading = useNavigationLoadingState()
// 延迟初始化预加载，传入 router 实例
let routePrefetch: ReturnType<typeof useRoutePrefetch> | null = null
const BACKEND_MODE_ALLOWED_PATHS = ['/login', '/auth', '/key-usage', '/setup', '/payment/result', '/payment/airwallex', '/legal']
const BACKEND_MODE_CALLBACK_PATHS = [
  '/auth/callback',
  '/auth/linuxdo/callback',
  '/auth/dingtalk/callback',
  '/auth/dingtalk/email-completion',
  '/auth/oidc/callback',
  '/auth/wechat/callback',
  '/auth/wechat/payment/callback',
]
const BACKEND_MODE_PENDING_AUTH_PATHS = ['/register', '/email-verify']

function isBackendModePublicRouteAllowed(path: string, hasPendingAuthSession: boolean): boolean {
  if (BACKEND_MODE_ALLOWED_PATHS.some((allowedPath) => path === allowedPath || path.startsWith(allowedPath))) {
    return true
  }

  if (BACKEND_MODE_CALLBACK_PATHS.some((callbackPath) => path === callbackPath)) {
    return true
  }

  if (hasPendingAuthSession && BACKEND_MODE_PENDING_AUTH_PATHS.some((allowedPath) => path === allowedPath)) {
    return true
  }

  return false
}

router.beforeEach(async (to, _from, next) => {
  // 开始导航加载状态
  navigationLoading.startNavigation()

  const authStore = useAuthStore()

  // Restore auth state from localStorage on first navigation (page refresh)
  if (!authInitialized) {
    authStore.checkAuth()
    authInitialized = true
  }

  // Set page title and route-level SEO tags.
  const appStore = useAppStore()
  const adminSettingsStore = useAdminSettingsStore()
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  const documentTitle = resolveRouteDocumentTitle(to, appStore.siteName, customMenuItems)
  applyRouteSeo(to, { siteName: appStore.siteName, title: documentTitle })

  // Check if route requires authentication
  const requiresAuth = to.meta.requiresAuth !== false // Default to true
  const requiresAdmin = to.meta.requiresAdmin === true
  const requiresSupport = to.meta.requiresSupport === true

  if (to.path === '/setup') {
    try {
      const status = await getSetupStatus()
      if (!status.needs_setup) {
        next(resolveCompletedSetupRedirectPath(authStore.isAuthenticated, authStore.isAdmin))
        return
      }
    } catch {
      // If setup status cannot be determined, keep the setup page reachable.
    }
  }

  // If route doesn't require auth, allow access
  if (!requiresAuth) {
    // If already authenticated and trying to access login/register, redirect to appropriate dashboard
    if (authStore.isAuthenticated && (to.path === '/login' || to.path === '/auth' || to.path === '/register')) {
      // In backend mode, non-admin users should NOT be redirected away from login
      // (they are blocked from all protected routes, so redirecting would cause a loop)
      if (appStore.backendModeEnabled && !authStore.isAdmin && !authStore.isSupport) {
        next()
        return
      }
      // Admin users go to admin dashboard, regular/support users go to user dashboard
      next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
      return
    }
    // Model Plaza:公开路由但受「启用开关 + 可选强制登录」双重控制(后端同口径 fail-closed)
    if (to.path === '/model-plaza') {
      if (!appStore.publicSettingsLoaded) {
        try {
          await appStore.fetchPublicSettings()
        } catch (error) {
          console.warn('Failed to load public settings in route guard', error)
        }
      }
      const plazaSettings = appStore.cachedPublicSettings
      // 仅在设置成功加载且明确为 false 时拦截(瞬时加载失败视为未知,由后端 404 兜底)
      if (appStore.publicSettingsLoaded && plazaSettings?.model_plaza_enabled === false) {
        next(
          authStore.isAuthenticated
            ? authStore.isAdmin
              ? '/admin/dashboard'
              : '/dashboard'
            : '/home'
        )
        return
      }
      if (plazaSettings?.model_plaza_require_auth === true && !authStore.isAuthenticated) {
        next({ path: '/login', query: { redirect: to.fullPath } })
        return
      }
      // Backend mode:登录的非管理员也不可见(匿名由下方公共拦截处理,广场不在白名单)
      if (appStore.backendModeEnabled && authStore.isAuthenticated && !authStore.isAdmin) {
        next('/login')
        return
      }
    }
    // Backend mode: block public pages for unauthenticated users (except login, key-usage, setup)
    if (appStore.backendModeEnabled && !authStore.isAuthenticated) {
      const isAllowed = isBackendModePublicRouteAllowed(to.path, authStore.hasPendingAuthSession)
      if (!isAllowed) {
        next('/login')
        return
      }
    }
    next()
    return
  }

  // Route requires authentication
  if (!authStore.isAuthenticated) {
    // Not authenticated, redirect to login
    next({
      path: '/login',
      query: { redirect: to.fullPath } // Save intended destination
    })
    return
  }

  // Check admin requirement
  if (requiresAdmin && !authStore.isAdmin) {
    // User is authenticated but not admin, redirect to user dashboard
    next(authStore.isSupport ? '/admin/tickets' : '/dashboard')
    return
  }

  if (requiresSupport && !authStore.canAccessTicketAdmin) {
    next('/dashboard')
    return
  }

  if (to.meta.featureFlag) {
    if (!appStore.cachedPublicSettings) {
      await appStore.fetchPublicSettings().catch(() => null)
    }
    if (!isFeatureFlagEnabled(FeatureFlags[to.meta.featureFlag])) {
      next(authStore.isAdmin ? '/admin/settings' : '/dashboard')
      return
    }
  }

  if (to.meta.requiresSubscriptionGroups) {
    const subscriptionCapabilityStore = useSubscriptionCapabilityStore()
    let hasSubscriptionGroups = subscriptionCapabilityStore.hasSubscriptionGroups
    if (!hasSubscriptionGroups || !subscriptionCapabilityStore.capabilityLoaded) {
      hasSubscriptionGroups = await subscriptionCapabilityStore.fetchSubscriptionCapability().catch(() => false)
    }
    if (!hasSubscriptionGroups) {
      next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
      return
    }
  }

  if (requiresAdmin && authStore.isAdmin) {
    const adminComplianceStore = useAdminComplianceStore()
    if (!adminComplianceStore.initialized) {
      try {
        await adminComplianceStore.fetchStatus()
      } catch (error) {
        const err = error as { status?: number; code?: string; metadata?: Record<string, string> }
        if (err.status === 423 && err.code === 'ADMIN_COMPLIANCE_ACK_REQUIRED') {
          adminComplianceStore.requireAcknowledgement(err.metadata)
        }
      }
    }
  }

  // 公共设置可能尚未加载（App.vue 的 onMounted 异步拉取晚于首次导航，且纯静态部署
  // 无 __APP_CONFIG__ 注入）。此时 cachedPublicSettings 为空会把 payment/risk_control
  // 误判为“未启用”而错误拦截，故这里先确保设置加载完成。
  if ((to.meta.requiresPayment || to.meta.requiresRiskControl) && !appStore.publicSettingsLoaded) {
    try {
      await appStore.fetchPublicSettings()
    } catch (error) {
      console.warn('Failed to load public settings in route guard', error)
    }
  }

  // Only an explicit value from successfully loaded settings can disable a route.
  // A transient settings failure is unknown state, not a confirmed feature toggle.
  if (
    to.meta.requiresPayment &&
    appStore.publicSettingsLoaded &&
    appStore.cachedPublicSettings?.payment_enabled === false
  ) {
    next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
    return
  }

  if (to.meta.requiresPurchaseAvailable) {
    const paymentCheckoutStore = usePaymentCheckoutStore()
    const checkout = await paymentCheckoutStore.fetchCheckoutInfo()
    if (!checkout || !paymentCheckoutStore.canAccessPurchase) {
      next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
      return
    }
  }

  if (
    to.meta.requiresRiskControl &&
    appStore.publicSettingsLoaded &&
    appStore.cachedPublicSettings?.risk_control_enabled === false
  ) {
    next(authStore.isAdmin ? '/admin/settings' : '/dashboard')
    return
  }

  // 简易模式下限制访问某些页面
  if (authStore.isSimpleMode) {
    const restrictedPaths = [
      '/admin/groups',
      '/admin/subscriptions',
      '/admin/redeem',
      '/subscriptions',
      '/redeem'
    ]

    if (restrictedPaths.some((path) => to.path.startsWith(path))) {
      // 简易模式下访问受限页面,重定向到仪表板
      next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
      return
    }
  }

  // Backend mode: admin/support get protected access, non-privileged users are blocked.
  if (appStore.backendModeEnabled) {
    if (authStore.isAuthenticated && (authStore.isAdmin || authStore.isSupport)) {
      next()
      return
    }
    const isAllowed = isBackendModePublicRouteAllowed(to.path, authStore.hasPendingAuthSession)
    if (!isAllowed) {
      next('/login')
      return
    }
  }

  // All checks passed, allow navigation
  next()
})

/**
 * Navigation guard: End loading and trigger prefetch
 */
router.afterEach((to) => {
  // 结束导航加载状态
  navigationLoading.endNavigation()

  // 懒初始化预加载（首次导航时创建，传入 router 实例）
  if (!routePrefetch) {
    routePrefetch = useRoutePrefetch(router)
  }
  // 触发路由预加载（在浏览器空闲时执行）
  routePrefetch.triggerPrefetch(to)
})

/**
 * Navigation guard: Error handling
 * Handles dynamic import failures caused by deployment updates
 */
router.onError((error) => {
  console.error('Router error:', error)
  recoverFromChunkLoadError(error, 'router')
})

export default router
