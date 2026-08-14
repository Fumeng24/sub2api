#!/usr/bin/env node
import { mkdir, rm, writeFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import { basename, join, resolve } from 'node:path'
import { spawn } from 'node:child_process'

const args = new Map()
for (const arg of process.argv.slice(2)) {
  const [key, value = 'true'] = arg.replace(/^--/, '').split('=')
  args.set(key, value)
}

const baseUrl = String(args.get('base-url') || 'http://127.0.0.1:4173').replace(/\/$/, '')
const outputDir = resolve(String(args.get('out') || '/tmp/sub2api-gateway-route-smoke'))
const chromiumPath = String(args.get('chromium') || process.env.CHROMIUM_BIN || '/snap/bin/chromium')
const routeFilter = String(args.get('routes') || 'default')
const timeoutMs = Number(args.get('timeout-ms') || 12000)
const scenario = String(args.get('scenario') || 'route-shell')

const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
]

const routedPageSelector = '#app > *'

const allRoutes = [
  { path: '/home', selector: routedPageSelector, role: 'public' },
  { path: '/pricing', selector: '.public-gateway-shell', role: 'public' },
  { path: '/docs', selector: '.public-gateway-shell', role: 'public' },
  { path: '/status', selector: '.public-gateway-shell', role: 'public' },
  { path: '/enterprise', selector: '.public-gateway-shell', role: 'public' },
  { path: '/login', selector: routedPageSelector, role: 'public' },
  { path: '/register', selector: routedPageSelector, role: 'public' },
  { path: '/email-verify', selector: routedPageSelector, role: 'public' },
  { path: '/forgot-password', selector: routedPageSelector, role: 'public' },
  { path: '/reset-password', selector: routedPageSelector, role: 'public' },
  { path: '/auth/callback', selector: routedPageSelector, role: 'public' },
  { path: '/auth/linuxdo/callback', selector: routedPageSelector, role: 'public' },
  { path: '/auth/wechat/callback', selector: routedPageSelector, role: 'public' },
  { path: '/auth/wechat/payment/callback', selector: routedPageSelector, role: 'public' },
  { path: '/auth/dingtalk/callback', selector: routedPageSelector, role: 'public' },
  { path: '/auth/dingtalk/email-completion', selector: routedPageSelector, role: 'public' },
  { path: '/auth/oidc/callback', selector: routedPageSelector, role: 'public' },
  { path: '/key-usage', selector: routedPageSelector, role: 'public' },
  { path: '/legal/smoke-terms', selector: routedPageSelector, role: 'public', expectText: ['Smoke Service Terms', 'Legal route smoke content'] },
  { path: '/model-plaza', selector: routedPageSelector, role: 'public' },
  { path: '/setup', selector: routedPageSelector, role: 'public' },
  { path: '/__smoke_not_found__', selector: routedPageSelector, role: 'public' },
  { path: '/dashboard', selector: routedPageSelector, role: 'user' },
  { path: '/keys', selector: routedPageSelector, role: 'user' },
  { path: '/image-generation', selector: routedPageSelector, role: 'user' },
  { path: '/batch-image', selector: routedPageSelector, role: 'user' },
  { path: '/available-channels', selector: '.table-wrapper', role: 'user', expectText: ['Codex Pro', 'Claude Plus', 'claude-sonnet-4-6'] },
  { path: '/usage', selector: routedPageSelector, role: 'user' },
  { path: '/usage?tab=errors', selector: routedPageSelector, role: 'user' },
  { path: '/messages', selector: routedPageSelector, role: 'user', expectText: ['Messages'] },
  { path: '/tickets', selector: routedPageSelector, role: 'user', expectText: ['Tickets', 'Create Ticket'] },
  { path: '/redeem', selector: routedPageSelector, role: 'user' },
  { path: '/affiliate', selector: routedPageSelector, role: 'user' },
  { path: '/monitor', selector: routedPageSelector, role: 'user', expectText: ['Codex Pro', 'Smoke', 'gpt-5.5'] },
  { path: '/profile', selector: routedPageSelector, role: 'user' },
  { path: '/subscriptions', selector: routedPageSelector, role: 'user' },
  { path: '/purchase', selector: routedPageSelector, role: 'user', expectText: ['Card code top-up', 'Online top-up', 'Redeem card code'] },
  { path: '/orders', selector: routedPageSelector, role: 'user' },
  { path: '/invoices', selector: routedPageSelector, role: 'user', expectText: ['Invoices', 'Eligible amount'] },
  { path: '/payment/qrcode', selector: routedPageSelector, role: 'user' },
  { path: '/payment/result', selector: routedPageSelector, role: 'public' },
  { path: '/payment/stripe', selector: routedPageSelector, role: 'public' },
  { path: '/payment/airwallex', selector: routedPageSelector, role: 'public' },
  { path: '/payment/stripe-popup', selector: routedPageSelector, role: 'public' },
  { path: '/custom/smoke-page', selector: '.markdown-page-content', role: 'user', expectText: ['Smoke Custom Page', 'Markdown route smoke content'] },
  { path: '/admin/dashboard', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/ops', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/scheduler', selector: routedPageSelector, role: 'admin', expectText: ['Auto Sort', 'Save Order'] },
  { path: '/admin/audit-logs', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/users', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/groups', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/user-pricing', selector: routedPageSelector, role: 'admin', expectText: ['User Discounts & Markups'] },
  { path: '/admin/channels/pricing', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/channels/monitor', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/subscriptions', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/accounts', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/upstreams', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/announcements', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/tickets', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/proxies', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/redeem', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/promo-codes', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/settings', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/business-settings', selector: routedPageSelector, role: 'admin', expectText: ['Business Settings', 'Ticket rules'] },
  { path: '/admin/risk-control', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/prompt-audit', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/usage', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/affiliates/invites', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/affiliates/rebates', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/affiliates/transfers', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/orders/dashboard', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/orders', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/orders/plans', selector: routedPageSelector, role: 'admin' },
  { path: '/admin/invoices', selector: routedPageSelector, role: 'admin', expectText: ['Invoice Management', 'All statuses'] },
]

const defaultRoutes = allRoutes.filter((route) => [
  '/home',
  '/pricing',
  '/docs',
  '/status',
  '/enterprise',
  '/login',
  '/auth/callback',
  '/key-usage',
  '/legal/smoke-terms',
  '/setup',
  '/__smoke_not_found__',
  '/dashboard',
  '/keys',
  '/available-channels',
  '/usage',
  '/tickets',
  '/purchase',
  '/invoices',
  '/monitor',
  '/custom/smoke-page',
  '/payment/result',
  '/payment/stripe-popup',
  '/admin/dashboard',
  '/admin/scheduler',
  '/admin/accounts',
  '/admin/orders/dashboard',
  '/admin/invoices',
  '/admin/affiliates/invites',
].includes(route.path))

const scenarioRoutePaths = {
  'user-states': [
    '/purchase',
    '/keys',
    '/usage?tab=errors',
    '/monitor',
    '/setup',
  ],
  'admin-states': [
    '/admin/scheduler',
    '/admin/redeem',
    '/admin/accounts',
    '/admin/channels/monitor',
  ],
  'auth-states': [
    '/login',
    '/register',
    '/auth/callback',
  ],
  'announcement-states': [
    '/dashboard',
  ],
  'dashboard-states': [
    '/dashboard',
  ],
  'sidebar-states': [
    '/dashboard',
  ],
}

const selectedRoutes = (() => {
  if (routeFilter === 'default' && scenarioRoutePaths[scenario]) {
    return allRoutes.filter((route) => scenarioRoutePaths[scenario].includes(route.path))
  }
  if (routeFilter === 'all') return allRoutes
  if (routeFilter === 'default') return defaultRoutes

  const requested = routeFilter.split(',').map((route) => route.trim()).filter(Boolean)
  return allRoutes.filter((route) => requested.includes(route.path))
})()

if (selectedRoutes.length === 0) {
  throw new Error(`No routes matched --routes=${routeFilter}`)
}

if (!existsSync(chromiumPath)) {
  throw new Error(`Chromium executable not found: ${chromiumPath}`)
}

const publicSettings = {
  registration_enabled: true,
  email_verify_enabled: false,
  force_email_on_third_party_signup: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: true,
  password_reset_enabled: true,
  invitation_code_enabled: false,
  turnstile_enabled: false,
  turnstile_site_key: '',
  site_name: 'Wegoo AI',
  site_logo: '/logo.svg',
  site_subtitle: 'AI Gateway',
  api_base_url: 'https://api.wegoo.site',
  contact_info: '',
  doc_url: 'https://docs.wegoo.site/guide/',
  home_content: '',
  hide_ccs_import_button: false,
  payment_enabled: true,
  risk_control_enabled: true,
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  custom_menu_items: [],
  custom_endpoints: [],
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  wechat_oauth_open_enabled: false,
  wechat_oauth_mp_enabled: false,
  wechat_oauth_mobile_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  login_agreement_enabled: false,
  login_agreement_mode: 'modal',
  login_agreement_updated_at: '',
  login_agreement_revision: '',
  login_agreement_documents: [],
  backend_mode_enabled: false,
  version: 'smoke',
  payment_balance_recharge_multiplier: 1,
  balance_low_notify_enabled: false,
  account_quota_notify_enabled: false,
  balance_low_notify_threshold: 10,
  channel_monitor_enabled: true,
  channel_monitor_default_interval_seconds: 60,
  available_channels_enabled: true,
  service_quota_enabled: true,
  affiliate_enabled: true,
  allow_user_view_error_requests: true,
}

function publicSettingsForRoute(route) {
  const settings = {
    ...publicSettings,
    login_agreement_documents: [...publicSettings.login_agreement_documents],
    custom_menu_items: [...publicSettings.custom_menu_items],
    custom_endpoints: [...publicSettings.custom_endpoints],
  }

  if (route.path === '/legal/smoke-terms') {
    return {
      ...settings,
      login_agreement_updated_at: '2026-07-11',
      login_agreement_documents: [
        {
          id: 'smoke-terms',
          title: 'Smoke Service Terms',
          content_md: '# Smoke Service Terms\n\nLegal route smoke content.\n\n## Scope\n\nThis document validates the public legal page.',
        },
      ],
    }
  }

  if (scenario !== 'auth-states') return settings

  return {
    ...settings,
    turnstile_enabled: true,
    turnstile_site_key: 'smoke-turnstile-site-key',
    github_oauth_enabled: true,
    google_oauth_enabled: true,
    oidc_oauth_enabled: true,
    oidc_oauth_provider_name: 'Smoke SSO',
    linuxdo_oauth_enabled: false,
    dingtalk_oauth_enabled: false,
    wechat_oauth_enabled: false,
    login_agreement_enabled: true,
    login_agreement_mode: route.path.startsWith('/register') ? 'checkbox' : 'modal',
    login_agreement_updated_at: '2026-06-27',
    login_agreement_revision: `smoke-auth-${route.path.startsWith('/register') ? 'checkbox' : 'modal'}`,
    login_agreement_documents: [
      { id: 'terms', title: '服务条款' },
      { id: 'privacy', title: '隐私政策' },
    ],
  }
}

function publicSettingsForSmokeRoute(route) {
  const settings = publicSettingsForRoute(route)
  if (route.path === '/custom/smoke-page') {
    return {
      ...settings,
      custom_menu_items: [
        {
          id: 'smoke-page',
          label: 'Smoke Custom Page',
          icon_svg: '',
          url: 'md:smoke-page',
          page_slug: 'smoke-page',
          visibility: 'user',
          sort_order: 1,
        },
      ],
    }
  }
  if (scenario !== 'user-states' || route.path !== '/keys') return settings
  return {
    ...settings,
    custom_endpoints: [
      {
        name: 'Hong Kong Relay',
        endpoint: 'https://hk.example.com/v1',
        description: 'Low latency for Asia traffic',
      },
      {
        name: 'Backup Relay',
        endpoint: 'https://backup.example.com/v1',
        description: 'Fallback endpoint',
      },
    ],
  }
}

function userForRole(role) {
  return {
    id: role === 'admin' ? 1 : 2,
    username: role === 'admin' ? 'admin-smoke' : 'user-smoke',
    email: role === 'admin' ? 'admin@example.com' : 'user@example.com',
    role: role === 'admin' ? 'admin' : 'user',
    balance: 1288.88,
    concurrency: 10,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    run_mode: 'standard',
  }
}

function paginated(items = []) {
  return { items, total: items.length, page: 1, page_size: 20, pages: items.length ? 1 : 0 }
}

function monitorTimeline(status = 'operational') {
  return Array.from({ length: 12 }, (_, index) => ({
    status,
    latency_ms: status === 'operational' ? 820 + index : 1800 + index,
    ping_latency_ms: status === 'operational' ? 120 + index : 340 + index,
    checked_at: new Date(Date.now() - index * 60 * 1000).toISOString(),
  }))
}

const monitorItems = [
  {
    id: 101,
    name: 'Codex Pro Smoke',
    provider: 'openai',
    group_name: 'pro',
    primary_model: 'gpt-5.5',
    primary_status: 'operational',
    primary_latency_ms: 836,
    primary_ping_latency_ms: 126,
    availability_7d: 99.4,
    extra_models: [
      { model: 'gpt-5.4-mini', status: 'operational', latency_ms: 620 },
    ],
    timeline: monitorTimeline('operational'),
  },
  {
    id: 102,
    name: 'Claude Max Smoke',
    provider: 'anthropic',
    group_name: 'max',
    primary_model: 'claude-sonnet-4-6',
    primary_status: 'degraded',
    primary_latency_ms: 2410,
    primary_ping_latency_ms: 420,
    availability_7d: 94.2,
    extra_models: [],
    timeline: monitorTimeline('degraded'),
  },
]

const monitorDetails = new Map(monitorItems.map((item) => [item.id, {
  id: item.id,
  name: item.name,
  provider: item.provider,
  group_name: item.group_name,
  models: [
    {
      model: item.primary_model,
      latest_status: item.primary_status,
      latest_latency_ms: item.primary_latency_ms,
      availability_7d: item.availability_7d,
      availability_15d: item.availability_7d - 0.2,
      availability_30d: item.availability_7d - 0.4,
      avg_latency_7d_ms: item.primary_latency_ms,
    },
    ...item.extra_models.map((model) => ({
      model: model.model,
      latest_status: model.status,
      latest_latency_ms: model.latency_ms,
      availability_7d: 99.1,
      availability_15d: 98.9,
      availability_30d: 98.7,
      avg_latency_7d_ms: model.latency_ms,
    })),
  ],
}]))

const smokeGroups = [
  {
    id: 1,
    name: 'pro',
    platform: 'openai',
    status: 'active',
    is_exclusive: false,
    rate_multiplier: 0.15,
    user_rate_multiplier: 0.15,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 2,
    name: 'max',
    platform: 'anthropic',
    status: 'active',
    is_exclusive: false,
    rate_multiplier: 1,
    user_rate_multiplier: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
]

function tokenPricing(inputPrice, outputPrice, cacheWritePrice, cacheReadPrice) {
  return {
    billing_mode: 'token',
    input_price: inputPrice,
    output_price: outputPrice,
    cache_write_price: cacheWritePrice,
    cache_read_price: cacheReadPrice,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
  }
}

const smokeAvailableChannels = [
  {
    name: 'Claude Models',
    description: 'Anthropic Messages models',
    platforms: [
      {
        platform: 'anthropic',
        endpoints: ['Anthropic Messages'],
        supported_endpoint_types: ['anthropic'],
        groups: [
          {
            id: 12,
            name: 'Claude Plus',
            platform: 'anthropic',
            subscription_type: 'standard',
            rate_multiplier: 0.3,
            peak_rate_enabled: false,
            peak_start: '',
            peak_end: '',
            peak_rate_multiplier: 1,
            is_exclusive: false,
          },
        ],
        supported_models: [
          {
            name: 'claude-sonnet-4-6',
            platform: 'anthropic',
            pricing: tokenPricing(0.000003, 0.000015, 0.00000375, 0.0000003),
          },
        ],
      },
    ],
  },
  {
    name: 'OpenAI Models',
    description: 'OpenAI Responses models',
    platforms: [
      {
        platform: 'openai',
        endpoints: ['Responses', 'OpenAI Chat'],
        supported_endpoint_types: ['openai'],
        groups: [
          {
            id: 11,
            name: 'Codex Pro',
            platform: 'openai',
            subscription_type: 'standard',
            rate_multiplier: 0.2,
            peak_rate_enabled: false,
            peak_start: '',
            peak_end: '',
            peak_rate_multiplier: 1,
            is_exclusive: false,
          },
        ],
        supported_models: [
          {
            name: 'gpt-5.6',
            platform: 'openai',
            pricing: tokenPricing(0.00000125, 0.00001, null, 0.000000125),
          },
        ],
      },
    ],
  },
]

const smokeAccounts = [
  {
    id: 101,
    name: 'codex-pro-oauth',
    platform: 'openai',
    type: 'oauth',
    status: 'active',
    schedulable: true,
    concurrency: 1000,
    priority: 10,
    rate_multiplier: 0.15,
    groups: [smokeGroups[0]],
    credentials: { email: 'codex@example.com', plan_type: 'pro' },
    extra: { privacy_mode: 'training_off' },
    usage: { total_requests: 12, total_tokens: 2048, total_cost: 1.24 },
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_used_at: '2026-06-27T12:00:00Z',
    expires_at: null,
    notes: 'smoke account',
  },
  {
    id: 102,
    name: 'claude-max-key',
    platform: 'anthropic',
    type: 'apikey',
    status: 'rate_limited',
    schedulable: false,
    concurrency: 500,
    priority: 20,
    rate_multiplier: null,
    groups: [smokeGroups[1]],
    credentials: {},
    extra: {},
    usage: { total_requests: 3, total_tokens: 512, total_cost: 0.36 },
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_used_at: null,
    expires_at: null,
    notes: 'rate limited sample',
  },
  {
    id: 103,
    name: 'codex-pro-key-degraded',
    platform: 'openai',
    type: 'apikey',
    status: 'active',
    schedulable: true,
    concurrency: 250,
    current_concurrency: 12,
    priority: 1,
    rate_multiplier: 1,
    groups: [smokeGroups[0]],
    credentials: {},
    extra: { upstream_rate_cached: 0.22, rate_scale: 0.5 },
    usage: { total_requests: 18, total_tokens: 4096, total_cost: 2.4 },
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_used_at: '2026-06-27T12:05:00Z',
    expires_at: null,
    notes: 'degraded scheduler sample',
  },
]

const smokeSchedulingEntries = [
  {
    account_id: 101,
    group_id: 1,
    role: 'primary',
    weight: 1,
    sort_order: 20,
    account: smokeAccounts[0],
  },
  {
    account_id: 103,
    group_id: 1,
    role: 'primary',
    weight: 1,
    sort_order: 10,
    account: smokeAccounts[2],
  },
]

const smokeAccountMonitors = [
  {
    id: 401,
    account_id: 103,
    provider: 'openai',
    model: 'gpt-5.4-mini',
    enabled: true,
    interval_seconds: 60,
    jitter_seconds: 0,
    last_checked_at: '2026-06-27T12:00:00Z',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-06-27T12:00:00Z',
  },
]

const smokeAccountMonitorStatuses = {
  103: {
    monitor_id: 401,
    account_id: 103,
    model: 'gpt-5.4-mini',
    enabled: true,
    latest_status: 'failed',
    latest_latency_ms: 3200,
    ping_latency_ms: 520,
    availability_1h: 18.5,
    last_checked_at: '2026-06-27T12:00:00Z',
    timeline: [
      { status: 'failed', latency_ms: 3200, checked_at: '2026-06-27T12:00:00Z' },
      { status: 'degraded', latency_ms: 1800, checked_at: '2026-06-27T11:59:00Z' },
      { status: 'failed', latency_ms: 3500, checked_at: '2026-06-27T11:58:00Z' },
      { status: 'operational', latency_ms: 720, checked_at: '2026-06-27T11:57:00Z' },
    ],
  },
}

const smokeApiKeys = [
  {
    id: 201,
    user_id: 2,
    key: 'sk-smoke-key-201',
    name: 'Smoke Gateway Key',
    category: 'openai',
    group_id: 1,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: '2026-06-27T12:00:00Z',
    quota: 1000,
    quota_used: 12.5,
    expires_at: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    group: {
      ...smokeGroups[0],
      name: 'Codex Pro',
      description: null,
      subscription_type: 'standard',
      allow_messages_dispatch: true,
      allow_image_generation: false,
      image_rate_independent: false,
      image_rate_multiplier: 1,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      require_oauth_only: false,
      require_privacy_set: false,
    },
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    window_5h_start: null,
    window_1d_start: null,
    window_7d_start: null,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
  },
]

const smokeUserErrorRequests = [
  {
    id: 701,
    created_at: '2026-06-27T12:00:00Z',
    model: 'gpt-5.5',
    inbound_endpoint: '/v1/responses',
    status_code: 503,
    category: 'service_unavailable',
    platform: 'openai',
    message: 'Our servers are currently overloaded. Please try again later.',
    key_name: 'Smoke Gateway Key',
    key_deleted: false,
  },
  {
    id: 702,
    created_at: '2026-06-27T11:58:00Z',
    model: 'claude-sonnet-4-6',
    inbound_endpoint: '/v1/messages',
    status_code: 429,
    category: 'rate_limit',
    platform: 'anthropic',
    message: 'Current group is temporarily unavailable. Switch group or retry later.',
    key_name: 'Archived Key',
    key_deleted: true,
  },
]

const smokeAnnouncements = [
  {
    id: 901,
    title: 'Popup Smoke Announcement',
    content: '**Popup announcement content**',
    notify_mode: 'popup',
    read_at: null,
    created_at: '2026-07-11T10:00:00Z',
    updated_at: '2026-07-11T10:00:00Z',
  },
  {
    id: 902,
    title: 'List Smoke Announcement',
    content: 'Detail announcement content',
    notify_mode: 'silent',
    read_at: null,
    created_at: '2026-07-11T09:00:00Z',
    updated_at: '2026-07-11T09:00:00Z',
  },
]

const smokeUserErrorDetails = new Map(smokeUserErrorRequests.map((item) => [item.id, {
  ...item,
  upstream_status_code: item.status_code,
  error_body: JSON.stringify({
    error: {
      message: item.message,
      type: item.category,
    },
  }, null, 2),
}]))

const smokeRedeemCodes = [
  {
    id: 501,
    code: 'SMOKE-RECHARGE-001',
    type: 'balance',
    value: 100,
    status: 'unused',
    business_category: 'user_recharge',
    used_by: null,
    used_at: null,
    expires_at: null,
    notes: 'admin state smoke',
    group_id: null,
    group: null,
    user: null,
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 502,
    code: 'SMOKE-GIFT-002',
    type: 'balance',
    value: 20,
    status: 'used',
    business_category: 'gift_compensation',
    used_by: 2,
    used_at: '2026-06-01T00:00:00Z',
    expires_at: null,
    notes: 'used sample',
    group_id: null,
    group: null,
    user: { id: 2, email: 'user@example.com' },
    created_at: '2026-01-02T00:00:00Z',
  },
]

const smokeChannelMonitors = [
  {
    id: 301,
    name: 'Codex Pro Public Status',
    provider: 'openai',
    api_mode: 'responses',
    endpoint: 'https://api.wegoo.site/v1/responses',
    api_key_id: null,
    api_key_masked: 'sk-***smoke',
    primary_model: 'gpt-5.5',
    extra_models: ['gpt-5.4-mini'],
    group_name: 'pro',
    sort_order: 1,
    enabled: true,
    interval_seconds: 300,
    jitter_seconds: 20,
    last_checked_at: '2026-06-27T12:00:00Z',
    created_by: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    primary_status: 'operational',
    primary_latency_ms: 720,
    availability_7d: 99.7,
    extra_models_status: [{ model: 'gpt-5.4-mini', status: 'degraded', latency_ms: 1550 }],
    template_id: null,
    extra_headers: {},
    body_override_mode: 'off',
    body_override: null,
  },
]

const smokeAffiliateDetail = {
  user_id: 2,
  aff_code: 'SMOKE2026',
  inviter_id: null,
  aff_count: 2,
  aff_quota: 36.5,
  aff_frozen_quota: 8,
  aff_history_quota: 44.5,
  effective_rebate_rate_percent: 10,
  invitees: [
    {
      user_id: 21,
      email: 'invitee@example.com',
      username: 'invitee-smoke',
      created_at: '2026-06-20T10:00:00Z',
      total_rebate: 12.5,
    },
  ],
}

const smokePromptAuditConfig = {
  enabled: true,
  blocking_enabled: false,
  blocking_latest_turn_only: true,
  store_pass_events: false,
  effective_mode: 'async_audit',
  strategy: 'priority',
  worker_count: 2,
  queue_capacity: 100,
  scanners: ['smoke'],
  all_groups: true,
  group_ids: [],
  endpoints: [],
  config_version: 3,
  updated_at: '2026-06-27T12:00:00Z',
  updated_by: 1,
  change_summary: 'Smoke configuration',
}

const smokePromptAuditRuntime = {
  process_status: 'running',
  effective_mode: 'async_audit',
  expected_config_version: 3,
  active_config_version: 3,
  config_loaded_at: '2026-06-27T12:00:00Z',
  worker_total: 2,
  worker_active: 1,
  worker_heartbeat_at: '2026-06-27T12:00:00Z',
  queue_capacity: 100,
  queue: { staging: 0, queued: 0, processing: 1, retry: 0, done: 12, failed: 0, active: 1 },
  processed_total: 12,
  failed_total: 0,
  enqueued_total: 13,
  dropped_total: 0,
  last_processed_at: '2026-06-27T12:00:00Z',
  database_status: 'ok',
  redis_status: 'ok',
  endpoints: {},
  guard_metrics: {
    total: 12,
    allowed: 12,
    flagged: 0,
    blocked: 0,
    unavailable: 0,
    invalid: 0,
    timeouts: 0,
    failovers: 0,
    bulkhead_full: 0,
    record_failed: 0,
  },
}

function apiPayload(path, currentRoute, method = 'GET', requestBody = null) {
  if (path === '/setup/status') {
    return { code: 0, data: { needs_setup: currentRoute.path === '/setup', step: 'database' } }
  }
  if (path === '/setup/test-db' || path === '/setup/test-redis') {
    return { code: 0, data: { ok: true } }
  }
  if (path === '/setup/install') {
    return { code: 0, data: { message: 'Setup smoke install accepted', restart: true } }
  }

  if (path.endsWith('/settings/public')) return { code: 0, data: publicSettingsForSmokeRoute(currentRoute) }
  if (path.endsWith('/auth/me')) return { code: 0, data: userForRole(currentRoute.role) }
  if (path.endsWith('/auth/refresh')) {
    return { code: 0, data: { access_token: 'smoke-token', refresh_token: 'smoke-refresh', expires_in: 3600, token_type: 'Bearer' } }
  }
  if (path.endsWith('/admin/compliance')) {
    return {
      code: 0,
      data: {
        required: false,
        version: 'v2026.06.10',
        document_path_zh: 'docs/legal/admin-compliance.zh.md',
        document_path_en: 'docs/legal/admin-compliance.en.md',
        document_url_zh: 'https://github.com/Wei-Shaw/sub2api/blob/main/docs/legal/admin-compliance.zh.md',
        document_url_en: 'https://github.com/Wei-Shaw/sub2api/blob/main/docs/legal/admin-compliance.en.md',
        ack_phrase_zh: 'smoke acknowledgement',
        ack_phrase_en: 'smoke acknowledgement',
      },
    }
  }
  if (path.includes('/public/model-pricing')) return { code: 0, data: [] }
  if (path.includes('/channels/available')) return { code: 0, data: smokeAvailableChannels }
  const monitorDetailMatch = path.match(/\/channel-monitors\/(\d+)\/status$/)
  if (monitorDetailMatch && !path.includes('/admin/')) {
    const detail = monitorDetails.get(Number(monitorDetailMatch[1])) || monitorDetails.get(101)
    return { code: 0, data: detail }
  }
  if ((path.includes('/public/channel-monitors') || path.endsWith('/channel-monitors')) && !path.includes('/admin/')) {
    return { code: 0, data: { items: monitorItems } }
  }
  if (path.includes('/tickets/unread-summary')) return { code: 0, data: { total: 0, open: 0, pending: 0 } }
  if (path.includes('/tickets/templates')) return { code: 0, data: [] }
  if (path.includes('/tickets/prefill')) return { code: 0, data: {} }
  if (path.includes('/admin/tickets/capabilities')) return { code: 0, data: { can_batch_update: true } }
  if (path.endsWith('/groups/available')) return { code: 0, data: smokeGroups }
  if (path.endsWith('/groups/rates')) return { code: 0, data: { 1: 0.15, 2: 1, 11: 0.2, 12: 0.3 } }
  if (path.endsWith('/admin/groups/all')) return { code: 0, data: smokeGroups }
  if (path.endsWith('/admin/groups/usage-summary')) return { code: 0, data: [] }
  if (path.endsWith('/admin/groups/capacity-summary')) return { code: 0, data: [] }
  if (path.endsWith('/admin/groups')) return { code: 0, data: paginated(smokeGroups) }
  const groupUpdateMatch = path.match(/\/admin\/groups\/(\d+)$/)
  if (groupUpdateMatch && method === 'PUT') {
    const groupId = Number(groupUpdateMatch[1])
    const group = smokeGroups.find((item) => item.id === groupId) || smokeGroups[0]
    return { code: 0, data: { ...group, ...(requestBody || {}) } }
  }
  const schedulingMatch = path.match(/\/admin\/groups\/(\d+)\/account-scheduling$/)
  if (schedulingMatch) {
    const groupId = Number(schedulingMatch[1])
    return {
      code: 0,
      data: {
        accounts: groupId === 1 ? smokeSchedulingEntries : [],
      },
    }
  }
  if (path.endsWith('/admin/proxies/all')) return { code: 0, data: [] }
  if (path.endsWith('/admin/payment/dashboard')) {
    return {
      code: 0,
      data: {
        today_amount: 128.5,
        total_amount: 4096.8,
        today_count: 6,
        total_count: 42,
        avg_amount: 97.54,
        daily_series: [
          { date: '2026-06-26', amount: 88.2, count: 4 },
          { date: '2026-06-27', amount: 128.5, count: 6 },
        ],
        payment_methods: [{ type: 'alipay', amount: 128.5, count: 6 }],
        top_users: [{ user_id: 2, email: 'user@example.com', amount: 128.5 }],
      },
    }
  }
  if (path.endsWith('/admin/accounts/today-stats/batch')) {
    return {
      code: 0,
      data: {
        stats: {
          101: { requests: 8, input_tokens: 1024, output_tokens: 256, total_tokens: 1280, cost: 0.42 },
          102: { requests: 1, input_tokens: 128, output_tokens: 64, total_tokens: 192, cost: 0.08 },
        },
      },
    }
  }
  if (path.endsWith('/admin/accounts/upstream-sub2api-status')) {
    return {
      code: 0,
      data: [
        {
          account_id: 101,
          account_name: 'codex-pro-oauth',
          local_platform: 'openai',
          base_url: 'https://api.wegoo.site',
          upstream_kind: 'sub2api',
          status: 'ok',
          fetched_at: '2026-06-27T12:00:00Z',
          cached: true,
          user_balance: 88.8,
          balance_unit: 'USD',
          upstream_group_default_rate_multiplier: 0.15,
          upstream_group_effective_rate_multiplier: 0.15,
        },
        {
          account_id: 103,
          account_name: 'codex-pro-key-degraded',
          local_platform: 'openai',
          base_url: 'https://api.wegoo.site',
          upstream_kind: 'sub2api',
          status: 'ok',
          fetched_at: '2026-06-27T12:00:00Z',
          cached: true,
          user_balance: 12.4,
          balance_unit: 'USD',
          upstream_group_default_rate_multiplier: 0.22,
          upstream_group_effective_rate_multiplier: 0.22,
        },
      ],
    }
  }
  if (path.endsWith('/admin/upstreams/account-status')) {
    return {
      code: 0,
      data: [
        {
          account_id: 101,
          account_name: 'codex-pro-oauth',
          local_platform: 'openai',
          base_url: 'https://api.wegoo.site',
          upstream_kind: 'sub2api',
          status: 'ok',
          fetched_at: '2026-06-27T12:00:00Z',
          cached: true,
          stale: false,
          user_balance: 88.8,
          balance_unit: 'USD',
          upstream_group_default_rate_multiplier: 0.15,
          upstream_group_effective_rate_multiplier: 0.15,
        },
        {
          account_id: 103,
          account_name: 'codex-pro-key-degraded',
          local_platform: 'openai',
          base_url: 'https://api.wegoo.site',
          upstream_kind: 'sub2api',
          status: 'ok',
          fetched_at: '2026-06-27T12:00:00Z',
          cached: true,
          stale: false,
          user_balance: 12.4,
          balance_unit: 'USD',
          upstream_group_default_rate_multiplier: 0.22,
          upstream_group_effective_rate_multiplier: 0.22,
        },
      ],
    }
  }
  if (path.endsWith('/admin/user-attributes')) return { code: 0, data: [] }
  if (path.endsWith('/admin/accounts')) return { code: 0, data: paginated(smokeAccounts) }
  if (path.endsWith('/user/aff')) return { code: 0, data: smokeAffiliateDetail }
  if (path.endsWith('/usage/dashboard/api-keys-usage')) {
    return {
      code: 0,
      data: {
        stats: {
          201: {
            today_requests: 3,
            today_input_tokens: 128,
            today_output_tokens: 64,
            today_total_tokens: 192,
            today_actual_cost: 0.18,
            total_requests: 32,
            total_input_tokens: 4096,
            total_output_tokens: 1024,
            total_tokens: 5120,
            total_actual_cost: 12.5,
          },
        },
      },
    }
  }
  const userErrorDetailMatch = path.match(/\/usage\/errors\/(\d+)$/)
  if (userErrorDetailMatch) {
    const detail = smokeUserErrorDetails.get(Number(userErrorDetailMatch[1])) || smokeUserErrorDetails.get(701)
    return { code: 0, data: detail }
  }
  if (path.endsWith('/usage/errors')) return { code: 0, data: paginated(smokeUserErrorRequests) }
  if (path.endsWith('/keys')) return { code: 0, data: paginated(smokeApiKeys) }
  if (path.endsWith('/admin/redeem-codes/stats')) {
    return {
      code: 0,
      data: {
        total: smokeRedeemCodes.length,
        unused: 1,
        used: 1,
        expired: 0,
        disabled: 0,
        total_value: 120,
        balance_value: 120,
        subscription_count: 0,
        concurrency_value: 0,
      },
    }
  }
  if (path.endsWith('/admin/redeem-codes')) return { code: 0, data: paginated(smokeRedeemCodes) }
  if (path.endsWith('/admin/channel-monitor-templates')) return { code: 0, data: { items: [], total: 0, page: 1, page_size: 100, pages: 0 } }
  if (path.endsWith('/admin/channel-monitors')) return { code: 0, data: paginated(smokeChannelMonitors) }
  if (path.endsWith('/admin/account-monitors/status')) {
    return { code: 0, data: { statuses: smokeAccountMonitorStatuses } }
  }
  if (path.endsWith('/admin/account-monitors')) {
    return { code: 0, data: { items: smokeAccountMonitors } }
  }
  if (path.endsWith('/admin/prompt-audit/config')) return { code: 0, data: smokePromptAuditConfig }
  if (path.endsWith('/admin/prompt-audit/runtime')) return { code: 0, data: smokePromptAuditRuntime }
  if (path.endsWith('/admin/prompt-audit/events')) return { code: 0, data: paginated() }
  if (path.includes('/admin/settings')) {
    return {
      code: 0,
      data: {
        ...publicSettings,
        overload_cooldown_enabled: true,
        rate_limit_429_cooldown_enabled: true,
        stream_timeout_enabled: true,
        rectifier_enabled: false,
        rectifier_thinking_signature_enabled: false,
        rectifier_thinking_budget_enabled: false,
        rectifier_apikey_signature_enabled: false,
        panel_rate_limit_enabled: false,
        panel_rate_limit_exempt_admin: true,
      },
    }
  }
  if (path.includes('/payment/checkout-info')) {
    const method = {
      currency: 'CNY',
      daily_limit: 50000,
      daily_used: 0,
      daily_remaining: 50000,
      single_min: 10,
      single_max: 5000,
      fee_rate: 0,
      available: true,
    }
    return {
      code: 0,
      data: {
        can_access_purchase: true,
        online_recharge_enabled: true,
        payment_enabled: true,
        methods: { alipay: method, wxpay: method },
        global_min: 10,
        global_max: 5000,
        plans: [],
        balance_disabled: false,
        balance_recharge_available: true,
        balance_recharge_unlock_threshold: 1000,
        balance_recharge_net_amount: 1288.88,
        balance_recharge_multiplier: 6.8,
        recharge_fee_rate: 0,
        help_text: '',
        help_image_url: '',
        stripe_publishable_key: '',
      },
    }
  }
  if (path.endsWith('/payment/config')) {
    return {
      code: 0,
      data: {
        payment_enabled: true,
        min_amount: 10,
        max_amount: 5000,
        daily_limit: 50000,
        max_pending_orders: 3,
        order_timeout_minutes: 15,
        balance_disabled: false,
        balance_recharge_unlock_threshold: 1000,
        balance_recharge_multiplier: 6.8,
        enabled_payment_types: ['alipay', 'wxpay'],
        help_image_url: '',
        help_text: '',
        stripe_publishable_key: '',
      },
    }
  }
  if (path.endsWith('/payment/limits')) {
    const method = { currency: 'CNY', daily_limit: 50000, daily_used: 0, daily_remaining: 50000, single_min: 10, single_max: 5000, fee_rate: 0, available: true }
    return { code: 0, data: { methods: { alipay: method, wxpay: method }, global_min: 10, global_max: 5000 } }
  }
  if (path.endsWith('/payment/plans') || path.endsWith('/payment/channels')) return { code: 0, data: [] }
  if (path.endsWith('/payment/orders/my')) return { code: 0, data: paginated() }
  if (path.endsWith('/invoices/summary')) {
    return {
      code: 0,
      data: {
        recharge_amount: 1500,
        invoiced_amount: 0,
        locked_amount: 0,
        available_amount: 1500,
        min_amount: 500,
        tax_rate: 0.5,
        tax_rate_percent: 50,
        min_tax_fee: 50,
        tax_fee_threshold: 50,
        can_apply: true,
        current_balance: 1288.88,
        invoiceable_basis: 'net_recharge',
      },
    }
  }
  if (path.endsWith('/invoices/templates')) return { code: 0, data: [] }
  if (path.endsWith('/invoices')) return { code: 0, data: paginated() }
  if (path.includes('/payment')) return { code: 0, data: paginated() }
  if (path.endsWith('/admin/announcements')) return { code: 0, data: paginated() }
  if (/\/announcements\/\d+\/read$/.test(path)) return { code: 0, data: { message: 'read' } }
  if (path.includes('/announcements')) {
    return { code: 0, data: scenario === 'announcement-states' ? smokeAnnouncements : [] }
  }
  if (path.includes('/dashboard') || path.includes('/stats') || path.includes('/summary')) return { code: 0, data: {} }
  if (path.includes('/keys') || path.includes('/api-keys')) return { code: 0, data: paginated() }
  if (path.includes('/usage') || path.includes('/logs') || path.includes('/requests')) return { code: 0, data: paginated() }
  if (path.includes('/admin') || path.includes('/user') || path.includes('/tickets') || path.includes('/redeem')) {
    return { code: 0, data: paginated() }
  }
  return { code: 0, data: {} }
}

function shouldMockRequest(url) {
  return url.pathname.startsWith('/api/v1/') || url.pathname.startsWith('/setup/')
}

class Cdp {
  constructor(wsUrl) {
    this.ws = new WebSocket(wsUrl)
    this.id = 0
    this.pending = new Map()
    this.handlers = new Map()
    this.ws.addEventListener('message', (event) => {
      const message = JSON.parse(event.data)
      if (message.id && this.pending.has(message.id)) {
        const { resolve, reject } = this.pending.get(message.id)
        this.pending.delete(message.id)
        if (message.error) reject(new Error(message.error.message))
        else resolve(message.result || {})
        return
      }
      const callbacks = this.handlers.get(message.method) || []
      callbacks.forEach((callback) => callback(message))
    })
  }

  async ready() {
    if (this.ws.readyState === WebSocket.OPEN) return
    await new Promise((resolve, reject) => {
      this.ws.addEventListener('open', resolve, { once: true })
      this.ws.addEventListener('error', reject, { once: true })
    })
  }

  send(method, params = {}, sessionId) {
    const id = ++this.id
    this.ws.send(JSON.stringify({ id, method, params, ...(sessionId ? { sessionId } : {}) }))
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id)
        reject(new Error(`CDP command timed out: ${method}`))
      }, timeoutMs + 5000)
      this.pending.set(id, {
        resolve: (value) => {
          clearTimeout(timer)
          resolve(value)
        },
        reject: (error) => {
          clearTimeout(timer)
          reject(error)
        },
      })
    })
  }

  on(method, callback) {
    const callbacks = this.handlers.get(method) || []
    callbacks.push(callback)
    this.handlers.set(method, callbacks)
  }

  close() {
    this.ws.close()
  }
}

async function launchChromium() {
  const userDataRoot = chromiumPath.startsWith('/snap/')
    ? join(process.env.HOME || '/root', 'snap/chromium/common')
    : '/tmp'
  const userDataDir = join(userDataRoot, `sub2api-smoke-chrome-${Date.now()}`)
  await mkdir(userDataDir, { recursive: true })
  const proc = spawn(chromiumPath, [
    '--headless=new',
    '--disable-gpu',
    '--no-sandbox',
    '--disable-dev-shm-usage',
    '--remote-debugging-port=0',
    `--user-data-dir=${userDataDir}`,
    'about:blank',
  ], { stdio: ['ignore', 'ignore', 'pipe'] })

  const wsUrl = await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Timed out waiting for Chromium DevTools URL')), 10000)
    proc.stderr.on('data', (chunk) => {
      const text = String(chunk)
      const match = text.match(/DevTools listening on (ws:\/\/[^\s]+)/)
      if (match) {
        clearTimeout(timer)
        resolve(match[1])
      }
    })
    proc.on('exit', (code) => reject(new Error(`Chromium exited early with code ${code}`)))
  })

  return { proc, userDataDir, wsUrl }
}

async function stopChromium(chrome) {
  const waitForExit = (timeoutMs) => new Promise((resolve) => {
    if (chrome.proc.exitCode !== null) {
      resolve(true)
      return
    }
    const timer = setTimeout(() => resolve(false), timeoutMs)
    chrome.proc.once('exit', () => {
      clearTimeout(timer)
      resolve(true)
    })
  })

  chrome.proc.kill('SIGTERM')
  const exited = await waitForExit(3000)
  if (!exited && chrome.proc.exitCode === null) {
    chrome.proc.kill('SIGKILL')
    await waitForExit(1000)
  }
  await rm(chrome.userDataDir, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 })
}

function routeFileName(route, viewport) {
  const clean = route.path.replace(/^\//, '').replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-|-$/g, '') || 'root'
  const suffix = scenario === 'route-shell' ? '' : `-${scenario}`
  return `${viewport.name}-${clean}${suffix}.png`
}

function scenarioActionExpression(route) {
  if (!['admin-states', 'user-states', 'auth-states', 'announcement-states', 'dashboard-states', 'sidebar-states'].includes(scenario)) return ''

  const common = `
    const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
    const waitFor = async (selector, predicate = (el) => Boolean(el)) => {
      const startedAt = Date.now();
      while (Date.now() - startedAt < ${timeoutMs}) {
        const el = document.querySelector(selector);
        if (el && predicate(el)) return el;
        await sleep(100);
      }
      throw new Error('Timed out waiting for ' + selector);
    };
    const click = async (selector) => {
      const el = await waitFor(selector, (node) => !node.disabled);
      el.scrollIntoView({ block: 'center', inline: 'center' });
      await sleep(80);
      el.click();
      return el;
    };
    const findButtonByText = (text) => Array.from(document.querySelectorAll('button')).find((node) =>
      (node.textContent || '').replace(/\\s+/g, ' ').trim().includes(text)
    );
    const waitForButtonByText = async (text) => {
      const startedAt = Date.now();
      while (Date.now() - startedAt < ${timeoutMs}) {
        const button = findButtonByText(text);
        if (button) return button;
        await sleep(100);
      }
      throw new Error('Timed out waiting for button text: ' + text);
    };
    const setControlValue = async (selector, value) => {
      const el = await waitFor(selector);
      el.scrollIntoView({ block: 'center', inline: 'nearest' });
      const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, 'value')?.set;
      if (setter) setter.call(el, value);
      else el.value = value;
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
      return el;
    };
  `

  if (route.path === '/login') {
    return `
      (async () => {
        ${common}
        await waitFor('[data-testid="login-agreement-modal"]');
        await click('[data-testid="login-agreement-reject"]');
        await waitFor('[data-testid="login-agreement-open"]');
        await click('[data-testid="login-agreement-open"]');
        await waitFor('[data-testid="login-agreement-modal"]');
        await click('[data-testid="login-agreement-accept"]');
        await waitForButtonByText('GitHub');
        await waitForButtonByText('Google');
        await waitForButtonByText('Smoke SSO');
        await waitFor('[data-testid="turnstile-smoke-rendered"]');
        await waitFor('[data-testid="login-submit"]', (node) => !node.disabled);
        await sleep(250);
        return {
          scenarioAction: 'login-agreement-oauth-turnstile',
          selectorFound:
            Boolean(findButtonByText('GitHub')) &&
            Boolean(findButtonByText('Google')) &&
            Boolean(findButtonByText('Smoke SSO')) &&
            Boolean(document.querySelector('[data-testid="turnstile-smoke-rendered"]')) &&
            Boolean(document.querySelector('[data-testid="login-submit"]:not(:disabled)')),
          textFound: (document.body?.innerText || '').includes('Smoke SSO'),
        };
      })()
    `
  }

  if (route.path === '/register') {
    return `
      (async () => {
        ${common}
        await waitFor('[data-testid="login-agreement-checkbox"]');
        await click('[data-testid="login-agreement-checkbox"]');
        await waitForButtonByText('GitHub');
        await waitForButtonByText('Google');
        await waitForButtonByText('Smoke SSO');
        await waitFor('[data-testid="turnstile-smoke-rendered"]');
        await waitFor('[data-testid="register-submit"]', (node) => !node.disabled);
        await sleep(250);
        return {
          scenarioAction: 'register-checkbox-oauth-turnstile',
          selectorFound:
            Boolean(document.querySelector('[data-testid="login-agreement-checkbox"]:checked')) &&
            Boolean(findButtonByText('GitHub')) &&
            Boolean(findButtonByText('Google')) &&
            Boolean(findButtonByText('Smoke SSO')) &&
            Boolean(document.querySelector('[data-testid="turnstile-smoke-rendered"]')) &&
            Boolean(document.querySelector('[data-testid="register-submit"]:not(:disabled)')),
          textFound: (document.body?.innerText || '').includes('Smoke SSO'),
        };
      })()
    `
  }

  if (route.path === '/auth/callback') {
    return `
      (async () => {
        ${common}
        await waitFor('[data-testid="oauth-callback-manual"]');
        const codeInput = await waitFor('[data-testid="oauth-callback-code"]');
        const stateInput = await waitFor('[data-testid="oauth-callback-state"]');
        const fullUrlInput = await waitFor('[data-testid="oauth-callback-full-url"]');
        await sleep(250);
        return {
          scenarioAction: 'oauth-callback-manual-copy',
          selectorFound:
            Boolean(document.querySelector('[data-testid="oauth-callback-manual"]')) &&
            codeInput.value === 'smoke-code' &&
            stateInput.value === 'smoke-state' &&
            fullUrlInput.value.includes('/auth/callback?code=smoke-code&state=smoke-state'),
          textFound: codeInput.value === 'smoke-code' && stateInput.value === 'smoke-state',
        };
      })()
    `
  }

  if (scenario === 'announcement-states' && route.path === '/dashboard') {
    return `
      (async () => {
        ${common}
        const waitForGone = async (selector) => {
          const startedAt = Date.now();
          while (Date.now() - startedAt < ${timeoutMs}) {
            if (!document.querySelector(selector)) return true;
            await sleep(100);
          }
          throw new Error('Timed out waiting for ' + selector + ' to disappear');
        };
        const popup = await waitFor('[data-testid="announcement-popup-dialog"]');
        const popupTextFound = (popup.textContent || '').includes('Popup Smoke Announcement') &&
          (popup.textContent || '').includes('Popup announcement content');
        const bodyIsLocked = () => document.body.classList.contains('modal-open') || document.body.style.overflow === 'hidden';
        const popupLockedBody = bodyIsLocked();
        await click('[data-testid="announcement-popup-dismiss"]');
        await waitForGone('[data-testid="announcement-popup-dialog"]');
        await sleep(150);
        const popupReleasedBody = !bodyIsLocked();
        await click('[data-testid="announcement-bell-open"]');
        await waitFor('[data-testid="announcement-list-dialog"]');
        await click('[data-testid="announcement-row-902"]');
        const detail = await waitFor('[data-testid="announcement-detail-dialog"]');
        await sleep(150);
        const detailText = detail.textContent || '';
        return {
          scenarioAction: 'announcement-popup-list-detail',
          selectorFound:
            popupLockedBody &&
            popupReleasedBody &&
            bodyIsLocked() &&
            Boolean(document.querySelector('[data-testid="announcement-list-dialog"]')) &&
            Boolean(detail),
          textFound:
            popupTextFound &&
            detailText.includes('List Smoke Announcement') &&
            detailText.includes('Detail announcement content'),
        };
      })()
    `
  }

  if (scenario === 'dashboard-states' && route.path === '/dashboard') {
    return `
      (async () => {
        ${common}
        const stats = await waitFor('[data-testid="dashboard-stats"]');
        const recent = await waitFor('[data-testid="dashboard-recent-usage"]');
        const charts = await waitFor('[data-testid="dashboard-charts"]');
        charts.scrollIntoView({ block: 'start', inline: 'nearest' });
        await sleep(250);
        return {
          scenarioAction: 'dashboard-presentation-sections',
          selectorFound: Boolean(stats) && Boolean(recent) && Boolean(charts),
          textFound: (stats.textContent || '').trim().length > 0 &&
            (recent.textContent || '').trim().length > 0 &&
            (charts.textContent || '').trim().length > 0,
        };
      })()
    `
  }

  if (scenario === 'sidebar-states' && route.path === '/dashboard') {
    return `
      (async () => {
        ${common}
        const guide = await waitFor('[data-testid="sidebar-integration-guide"]');
        if (window.innerWidth < 1024) {
          await click('[aria-label="Toggle menu"]');
          await waitFor('aside.sidebar', (node) => node.getBoundingClientRect().left >= -1);
          await sleep(350);
        }
        const nav = guide.closest('nav');
        const guideRect = guide.getBoundingClientRect();
        const guideText = (guide.textContent || '').replace(/\\s+/g, ' ').trim();
        return {
          scenarioAction: 'sidebar-integration-guide-first-highlighted',
          selectorFound:
            Boolean(nav) &&
            nav.firstElementChild === guide &&
            guide.classList.contains('sidebar-doc-link') &&
            guideRect.left >= 0 &&
            guideRect.right <= window.innerWidth,
          textFound: guideText.includes('Integration Guide') || guideText.includes('接入教程'),
          diagnostics: {
            guideText,
            guideLeft: Math.round(guideRect.left),
            guideRight: Math.round(guideRect.right),
          },
        };
      })()
    `
  }

  if (route.path === '/purchase') {
    return `
      (async () => {
        ${common}
        const cardPanel = await waitFor('[data-test="card-code-panel"]');
        const cardForm = await waitFor('[data-test="card-code-form"]');
        const cardText = cardPanel.textContent || '';

        await click('[data-test="online-recharge-mode"]');
        const onlinePanel = await waitFor('[data-test="online-recharge-panel"]');
        const paymentGrid = await waitFor('[data-testid="payment-method-grid"]');
        const methodLabels = Array.from(paymentGrid.querySelectorAll('[data-testid="payment-method-label"]'))
          .map((node) => (node.textContent || '').trim())
          .filter(Boolean);
        await click('[data-test="card-code-mode"]');
        const restoredCardPanel = await waitFor('[data-test="card-code-panel"]');
        await click('[data-test="online-recharge-mode"]');
        const finalOnlinePanel = await waitFor('[data-test="online-recharge-panel"]');
        await sleep(250);

        const scrollWidth = Math.max(document.documentElement.scrollWidth, document.body.scrollWidth);
        return {
          scenarioAction: 'purchase-card-code-online-toggle',
          selectorFound:
            Boolean(cardPanel) &&
            Boolean(cardForm) &&
            Boolean(onlinePanel) &&
            Boolean(paymentGrid) &&
            methodLabels.length >= 2 &&
            Boolean(restoredCardPanel) &&
            Boolean(finalOnlinePanel) &&
            scrollWidth <= window.innerWidth + 2,
          textFound:
            cardText.includes('Use a card code') &&
            cardText.includes('Redeem card code') &&
            methodLabels.includes('Alipay') &&
            methodLabels.includes('WeChat Pay'),
          diagnostics: {
            methodLabels,
            cardModeRestored: Boolean(restoredCardPanel),
            finalMode: 'online',
          },
        };
      })()
    `
  }

  if (route.path === '/admin/redeem') {
    return `
      (async () => {
        ${common}
        await click('[data-test="select-code"]');
        await click('[data-test="batch-update-open"]');
        await waitFor('[data-test="batch-update-form"]');
        await click('[data-test="batch-field-status"]');
        await click('[data-test="batch-field-notes"]');
        await setControlValue('[data-test="batch-notes-input"]', 'smoke batch update note');
        await sleep(250);
        return {
          scenarioAction: 'redeem-batch-update-modal',
          selectorFound: Boolean(document.querySelector('[data-test="batch-update-form"]')),
          textFound: (document.body?.innerText || '').includes('SMOKE-RECHARGE-001') || Boolean(document.querySelector('[data-test="batch-notes-input"]')),
        };
      })()
    `
  }

  if (route.path === '/admin/scheduler') {
    return `
      (async () => {
        ${common}
        const waitForGone = async (selector) => {
          const startedAt = Date.now();
          while (Date.now() - startedAt < ${timeoutMs}) {
            if (!document.querySelector(selector)) return true;
            await sleep(100);
          }
          throw new Error('Timed out waiting for ' + selector + ' to disappear');
        };
        await waitFor('[data-testid="scheduler-account-row-101"]');
        await waitFor('[data-testid="scheduler-account-row-103"]');
        const rowsAfterDrag = Array.from(document.querySelectorAll('[data-testid^="scheduler-account-row-"]'));
        const firstRowAfterDrag = rowsAfterDrag[0]?.getAttribute('data-testid') || '';
        await click('[data-testid="scheduler-save-order"]');
        await sleep(150);
        await waitFor('[data-testid="scheduler-save-order"]', (node) => !node.disabled);
        await waitFor('[data-testid="scheduler-account-row-101"]');
        const rowsAfterSave = Array.from(document.querySelectorAll('[data-testid^="scheduler-account-row-"]'));
        const firstRowAfterSave = rowsAfterSave[0]?.getAttribute('data-testid') || '';

        await click('[data-testid="scheduler-auto-sort-toggle"]');
        const autoSortToggle = await waitFor(
          '[data-testid="scheduler-auto-sort-toggle"]',
          (node) => node.getAttribute('aria-checked') === 'true' && !node.disabled,
        );
        const autoSortPolicy = await waitFor('[data-testid="scheduler-auto-sort-policy"]');
        await click('[data-testid="scheduler-refresh-order"]');
        await sleep(150);
        const rowsAfterRefresh = Array.from(document.querySelectorAll('[data-testid^="scheduler-account-row-"]'));
        const firstRowAfterRefresh = rowsAfterRefresh[0]?.getAttribute('data-testid') || '';
        await click('[data-testid="scheduler-account-config-103"]');
        const configDialog = await waitFor('[data-testid="scheduler-account-config-dialog"]');
        const capacity = await waitFor('[data-testid="scheduler-config-capacity"]');
        const monitorModel = await waitFor('[data-testid="scheduler-config-monitor-model"]');
        const manualRate = await setControlValue('[data-testid="scheduler-config-manual-rate"]', '0.11');
        await click('.modal-overlay [aria-label="Close modal"]');
        await waitForGone('[data-testid="scheduler-account-config-dialog"]');
        await click('[data-testid="scheduler-bulk-monitor-model"]');
        await waitFor('[data-testid="scheduler-bulk-monitor-model-dialog"]');
        const bulkModel = await setControlValue('[data-testid="scheduler-bulk-monitor-model-input"]', 'gpt-5.5');
        await sleep(250);
        const bodyText = document.body?.innerText || '';
        const diagnostics = {
          firstRowAfterDrag,
          firstRowAfterSave,
          firstRowAfterRefresh,
          autoSortChecked: autoSortToggle.getAttribute('aria-checked'),
          autoSortPolicy: autoSortPolicy.textContent?.trim(),
          capacity: capacity.value,
          monitorModel: monitorModel.value,
          manualRate: manualRate.value,
          bulkDialogFound: Boolean(document.querySelector('[data-testid="scheduler-bulk-monitor-model-dialog"]')),
        };
        return {
          scenarioAction: 'scheduler-drag-save-auto-sort-config',
          selectorFound:
            firstRowAfterDrag === 'scheduler-account-row-101' &&
            firstRowAfterSave === 'scheduler-account-row-101' &&
            firstRowAfterRefresh === 'scheduler-account-row-101' &&
            autoSortToggle.getAttribute('aria-checked') === 'true' &&
            autoSortPolicy.textContent?.includes('Stability') &&
            Boolean(configDialog) &&
            capacity.value === '250' &&
            monitorModel.value === 'gpt-5.4-mini' &&
            manualRate.value === '0.11' &&
            Boolean(document.querySelector('[data-testid="scheduler-bulk-monitor-model-dialog"]')),
          textFound:
            bodyText.includes('codex-pro-oauth') &&
            bodyText.includes('codex-pro-key-degraded') &&
            bulkModel.value === 'gpt-5.5',
          diagnostics,
        };
      })()
    `
  }

  if (route.path === '/admin/accounts') {
    return `
      (async () => {
        ${common}
        await setControlValue('[data-test="account-filters"] input', 'codex');
        await click('[data-test="account-bulk-edit-filtered"]');
        await waitFor('#bulk-edit-account-form');
        await click('#bulk-edit-base-url-enabled');
        await setControlValue('#bulk-edit-base-url', 'https://api.smoke.local');
        await sleep(250);
        return {
          scenarioAction: 'accounts-filtered-bulk-edit-modal',
          selectorFound: Boolean(document.querySelector('#bulk-edit-account-form')),
          textFound: (document.body?.innerText || '').includes('codex-pro-oauth') || Boolean(document.querySelector('#bulk-edit-base-url')),
        };
      })()
    `
  }

  if (route.path === '/admin/channels/monitor') {
    return `
      (async () => {
        ${common}
        await click('[data-test="channel-monitor-create"]');
        await waitFor('#channel-monitor-form');
        await sleep(250);
        return {
          scenarioAction: 'channel-monitor-create-modal',
          selectorFound: Boolean(document.querySelector('#channel-monitor-form')),
          textFound: (document.body?.innerText || '').includes('Codex Pro Public Status') || Boolean(document.querySelector('#channel-monitor-form')),
        };
      })()
    `
  }

  if (route.path === '/monitor') {
    return `
      (async () => {
        ${common}
        await click('[data-test="user-monitor-card"]');
        await waitFor('[data-test="user-monitor-detail"]');
        await sleep(250);
        const bodyText = document.body?.innerText || '';
        return {
          scenarioAction: 'monitor-detail-dialog',
          selectorFound: Boolean(document.querySelector('[data-test="user-monitor-detail"]')),
          textFound: bodyText.includes('gpt-5.5') && bodyText.includes('gpt-5.4-mini'),
        };
      })()
    `
  }

  if (route.path === '/keys') {
    return `
      (async () => {
        ${common}
        await click('[data-testid="use-key-open-201"]');
        await waitFor('[data-testid="use-key-group-context"]');
        await waitFor('[data-testid="use-key-endpoint-context"]');
        await sleep(250);
        const bodyText = document.body?.innerText || '';
        return {
          scenarioAction: 'keys-use-key-endpoint-dialog',
          selectorFound:
            Boolean(document.querySelector('[data-testid="use-key-group-context"]')) &&
            Boolean(document.querySelector('[data-testid="use-key-platform-context"]')) &&
            Boolean(document.querySelector('[data-testid="use-key-endpoint-context"]')),
          textFound:
            bodyText.includes('Codex Pro') &&
            bodyText.includes('OpenAI') &&
            bodyText.includes('https://api.wegoo.site') &&
            bodyText.includes('Hong Kong Relay') &&
            bodyText.includes('https://hk.example.com/v1'),
        };
      })()
    `
  }

  if (route.path.startsWith('/usage?tab=errors')) {
    return `
      (async () => {
        ${common}
        await waitFor('[data-testid="usage-error-mobile-list"], [data-testid="usage-error-desktop-table"]');
        const detailSelector = '[data-testid="usage-error-detail-701"]';
        await click(detailSelector);
        await waitFor('[data-testid="usage-error-detail-modal"]');
        await sleep(250);
        const bodyText = document.body?.innerText || '';
        return {
          scenarioAction: 'usage-errors-detail-dialog',
          selectorFound:
            Boolean(document.querySelector('[data-testid="usage-error-detail-modal"]')) &&
            Boolean(document.querySelector('[data-testid="usage-error-desktop-table"]')),
          textFound:
            bodyText.includes('gpt-5.5') &&
            bodyText.includes('/v1/responses') &&
            bodyText.includes('Our servers are currently overloaded'),
        };
      })()
    `
  }

  if (route.path === '/setup') {
    return `
      (async () => {
        ${common}
        await waitFor('[data-test="setup-step-database"]');
        await click('[data-test="setup-test-db"]');
        await waitFor('[data-test="setup-next"]', (node) => !node.disabled);
        await click('[data-test="setup-next"]');
        await waitFor('[data-test="setup-step-redis"]');
        await click('[data-test="setup-test-redis"]');
        await waitFor('[data-test="setup-next"]', (node) => !node.disabled);
        await click('[data-test="setup-next"]');
        await waitFor('[data-test="setup-step-admin"]');
        await setControlValue('[data-test="setup-admin-email"]', 'admin-smoke@example.com');
        await setControlValue('[data-test="setup-admin-password"]', 'smoke-password-123');
        await setControlValue('[data-test="setup-admin-confirm-password"]', 'smoke-password-123');
        await waitFor('[data-test="setup-next"]', (node) => !node.disabled);
        await click('[data-test="setup-next"]');
        await waitFor('[data-test="setup-step-complete"]');
        await sleep(250);
        const bodyText = document.body?.innerText || '';
        return {
          scenarioAction: 'setup-step-flow',
          selectorFound: Boolean(document.querySelector('[data-test="setup-step-complete"]')),
          textFound: bodyText.includes('admin-smoke@example.com') && bodyText.includes('localhost:6379'),
        };
      })()
    `
  }

  return ''
}

async function runScenarioAction(cdp, sessionId, route) {
  let schedulerDragSucceeded = true
  let schedulerDragMechanism = null
  if (scenario === 'admin-states' && route.path === '/admin/scheduler') {
    const dragResult = await dragSchedulerEntry(cdp, sessionId, 103, 101)
    schedulerDragSucceeded = dragResult.passed
    schedulerDragMechanism = dragResult.mechanism
  }

  const expression = scenarioActionExpression(route)
  if (!expression) return { scenarioAction: null, selectorFound: true, textFound: true, passed: true }

  const evaluated = await cdp.send('Runtime.evaluate', {
    returnByValue: true,
    awaitPromise: true,
    expression,
  }, sessionId)

  if (evaluated.exceptionDetails) {
    const exception = evaluated.exceptionDetails.exception || {}
    return {
      scenarioAction: 'exception',
      selectorFound: false,
      textFound: false,
      passed: false,
      error: exception.description || evaluated.exceptionDetails.text || 'Scenario action failed',
      schedulerDragSucceeded,
      schedulerDragMechanism,
    }
  }

  const value = evaluated.result.value || {}
  return {
    scenarioAction: value.scenarioAction || null,
    selectorFound: schedulerDragSucceeded && value.selectorFound !== false,
    textFound: value.textFound !== false,
    passed: schedulerDragSucceeded && value.selectorFound !== false && value.textFound !== false,
    schedulerDragSucceeded,
    schedulerDragMechanism,
    diagnostics: value.diagnostics || null,
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function dragSchedulerEntry(cdp, sessionId, sourceAccountId, targetAccountId) {
  const positions = await cdp.send('Runtime.evaluate', {
    returnByValue: true,
    awaitPromise: true,
    expression: `
      new Promise((resolve) => {
        const startedAt = Date.now();
        const timer = setInterval(() => {
          const source = document.querySelector('[data-testid="scheduler-drag-handle-${sourceAccountId}"]');
          const target = document.querySelector('[data-testid="scheduler-account-row-${targetAccountId}"]');
          if (source && target) {
            clearInterval(timer);
            const sourceRect = source.getBoundingClientRect();
            const targetRect = target.getBoundingClientRect();
            resolve({
              sourceX: sourceRect.left + sourceRect.width / 2,
              sourceY: sourceRect.top + sourceRect.height / 2,
              targetX: sourceRect.left + sourceRect.width / 2,
              targetY: targetRect.bottom - Math.min(8, targetRect.height / 4),
            });
          } else if (Date.now() - startedAt > ${timeoutMs}) {
            clearInterval(timer);
            resolve(null);
          }
        }, 100);
      })
    `,
  }, sessionId)

  const point = positions.result.value
  if (!point) return runSortableUpdate(cdp, sessionId, sourceAccountId, targetAccountId)

  await cdp.send('Input.dispatchMouseEvent', {
    type: 'mouseMoved',
    x: point.sourceX,
    y: point.sourceY,
  }, sessionId)
  await cdp.send('Input.dispatchMouseEvent', {
    type: 'mousePressed',
    x: point.sourceX,
    y: point.sourceY,
    button: 'left',
    buttons: 1,
    clickCount: 1,
  }, sessionId)

  const steps = 14
  for (let step = 1; step <= steps; step += 1) {
    const progress = step / steps
    await cdp.send('Input.dispatchMouseEvent', {
      type: 'mouseMoved',
      x: point.sourceX + (point.targetX - point.sourceX) * progress,
      y: point.sourceY + (point.targetY - point.sourceY) * progress,
      button: 'left',
      buttons: 1,
    }, sessionId)
    await sleep(20)
  }

  await cdp.send('Input.dispatchMouseEvent', {
    type: 'mouseReleased',
    x: point.targetX,
    y: point.targetY,
    button: 'left',
    buttons: 0,
    clickCount: 1,
  }, sessionId)
  await sleep(350)

  const result = await cdp.send('Runtime.evaluate', {
    returnByValue: true,
    expression: `Array.from(document.querySelectorAll('[data-testid^="scheduler-account-row-"]')).map((row) => row.getAttribute('data-testid'))`,
  }, sessionId)
  if (result.result.value?.[0] === `scheduler-account-row-${targetAccountId}`) {
    return { passed: true, mechanism: 'pointer' }
  }
  return runSortableUpdate(cdp, sessionId, sourceAccountId, targetAccountId)
}

async function runSortableUpdate(cdp, sessionId, sourceAccountId, targetAccountId) {
  const result = await cdp.send('Runtime.evaluate', {
    returnByValue: true,
    awaitPromise: true,
    expression: `
      (async () => {
        const source = document.querySelector('[data-testid="scheduler-account-row-${sourceAccountId}"]');
        const target = document.querySelector('[data-testid="scheduler-account-row-${targetAccountId}"]');
        const container = source?.parentElement;
        if (!source || !target || !container) return false;
        const sortableKey = Object.keys(container).find((key) => key.startsWith('Sortable'));
        const sortable = sortableKey ? container[sortableKey] : null;
        if (!sortable?.options?.onUpdate || !sortable?.options?.onEnd) return false;
        const children = Array.from(container.children);
        const oldIndex = children.indexOf(source);
        const newIndex = children.indexOf(target);
        if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) return false;
        const event = {
          item: source,
          from: container,
          to: container,
          oldIndex,
          newIndex,
          oldDraggableIndex: oldIndex,
          newDraggableIndex: newIndex,
          originalEvent: new Event('dragend'),
        };
        sortable.options.onStart?.(event);
        sortable.options.onUpdate(event);
        sortable.options.onEnd(event);
        await new Promise((resolve) => setTimeout(resolve, 250));
        const rows = Array.from(document.querySelectorAll('[data-testid^="scheduler-account-row-"]'));
        return rows[0]?.getAttribute('data-testid') === 'scheduler-account-row-${targetAccountId}';
      })()
    `,
  }, sessionId)
  return { passed: result.result.value === true, mechanism: 'sortable-event' }
}

function scenarioNetworkResult(route, requests) {
  if (scenario === 'user-states' && route.path === '/purchase') {
    const orderWrites = requests.filter((request) =>
      request.method === 'POST' && request.path === '/api/v1/payment/orders'
    )
    const assertions = [{ name: 'mode-switch-does-not-create-order', passed: orderWrites.length === 0 }]
    return { passed: assertions.every((item) => item.passed), assertions }
  }

  if (scenario !== 'admin-states' || route.path !== '/admin/scheduler') {
    return { passed: true, assertions: [] }
  }

  const schedulingWrites = requests.filter((request) =>
    request.method === 'PUT' && /\/admin\/groups\/\d+\/account-scheduling$/.test(request.path)
  )
  const manualSave = schedulingWrites.find((request) => {
    const accounts = request.body?.accounts
    return Array.isArray(accounts)
      && accounts.length === 2
      && accounts[0]?.account_id === 101
      && accounts[0]?.sort_order === 10
      && accounts[1]?.account_id === 103
      && accounts[1]?.sort_order === 20
  })

  const groupWrites = requests.filter((request) =>
    request.method === 'PUT' && /\/admin\/groups\/\d+$/.test(request.path)
  )
  const autoSortPolicySaved = groupWrites.some((request) =>
    request.body?.auto_sort_config?.enabled === true
      && request.body?.auto_sort_config?.basis === 'experience'
  )

  const assertions = [
    { name: 'manual-order-payload', passed: Boolean(manualSave) },
    { name: 'auto-sort-policy-saved', passed: autoSortPolicySaved },
  ]
  return { passed: assertions.every((item) => item.passed), assertions }
}

async function main() {
  await rm(outputDir, { recursive: true, force: true })
  await mkdir(outputDir, { recursive: true })

  const chrome = await launchChromium()
  const cdp = new Cdp(chrome.wsUrl)
  await cdp.ready()

  let currentRoute = selectedRoutes[0]
  let activeSessionId = ''
  const consoleMessages = []
  const apiRequests = []
  const pageMetadata = new Map()
  const schedulingEntriesBySession = new Map()

  cdp.on('Fetch.requestPaused', (message) => {
    const requestId = message.params.requestId
    const request = message.params.request
    const url = new URL(request.url)
    const sessionId = message.sessionId || activeSessionId
    if (!shouldMockRequest(url)) {
      cdp.send('Fetch.continueRequest', { requestId }, sessionId).catch(() => {})
      return
    }
    const metadata = pageMetadata.get(sessionId)
    const requestRoute = metadata?.route || currentRoute
    let requestBody = null
    if (request.postData) {
      try {
        requestBody = JSON.parse(request.postData)
      } catch {
        requestBody = request.postData
      }
    }
    apiRequests.push({
      route: requestRoute.path,
      viewport: metadata?.viewport || '',
      method: request.method,
      path: url.pathname,
      body: requestBody,
    })
    const isMarkdownPage = url.pathname.endsWith('/pages/smoke-page')
    const schedulingMatch = url.pathname.match(/\/admin\/groups\/(\d+)\/account-scheduling$/)
    let responsePayload
    if (isMarkdownPage) {
      responsePayload = '# Smoke Guide\n\nMarkdown route smoke content.\n\n## Request\n\n```bash\ncurl https://api.wegoo.site/v1/responses\n```'
    } else if (schedulingMatch) {
      const groupId = Number(schedulingMatch[1])
      let entries = schedulingEntriesBySession.get(sessionId) || []
      if (request.method === 'PUT' && Array.isArray(requestBody?.accounts)) {
        entries = requestBody.accounts.map((update) => {
          const original = smokeSchedulingEntries.find((entry) => entry.account_id === update.account_id)
          return original
            ? { ...original, ...update, account: { ...original.account } }
            : { ...update, group_id: groupId }
        })
        schedulingEntriesBySession.set(sessionId, entries)
      }
      responsePayload = JSON.stringify({ code: 0, data: { accounts: groupId === 1 ? entries : [] } })
    } else {
      responsePayload = JSON.stringify(apiPayload(url.pathname, requestRoute, request.method, requestBody))
    }
    cdp.send('Fetch.fulfillRequest', {
      requestId,
      responseCode: 200,
      responseHeaders: [{
        name: 'Content-Type',
        value: isMarkdownPage ? 'text/markdown; charset=utf-8' : 'application/json; charset=utf-8',
      }],
      body: Buffer.from(responsePayload).toString('base64'),
    }, sessionId).catch(() => {})
  })

  cdp.on('Runtime.consoleAPICalled', (message) => {
    const type = message.params.type
    if (!['error', 'warning'].includes(type)) return
    consoleMessages.push({
      route: currentRoute.path,
      type,
      text: message.params.args.map((arg) => arg.value || arg.description || '').join(' ').slice(0, 500),
    })
  })

  cdp.on('Runtime.exceptionThrown', (message) => {
    const details = message.params.exceptionDetails || {}
    const exception = details.exception || {}
    consoleMessages.push({
      route: currentRoute.path,
      type: 'exception',
      text: [
        details.text,
        exception.description,
        details.url ? `${basename(details.url)}:${details.lineNumber}:${details.columnNumber}` : '',
      ].filter(Boolean).join('\n').slice(0, 1200) || 'Runtime exception',
    })
  })

  const results = []

  async function createPage(route, viewport) {
    const { targetId } = await cdp.send('Target.createTarget', { url: 'about:blank' })
    const { sessionId } = await cdp.send('Target.attachToTarget', { targetId, flatten: true })
    await cdp.send('Page.enable', {}, sessionId)
    await cdp.send('Runtime.enable', {}, sessionId)
    await cdp.send('Fetch.enable', {
      patterns: [{ urlPattern: '*', requestStage: 'Request' }],
    }, sessionId)
    pageMetadata.set(sessionId, { route, viewport: viewport.name })
    schedulingEntriesBySession.set(
      sessionId,
      smokeSchedulingEntries.map((entry) => ({ ...entry, account: { ...entry.account } })),
    )
    await cdp.send('Page.addScriptToEvaluateOnNewDocument', {
      source: `
        (() => {
          localStorage.removeItem('auth_token');
          localStorage.removeItem('refresh_token');
          localStorage.removeItem('token_expires_at');
          localStorage.removeItem('auth_user');
          localStorage.removeItem('sub2api_login_agreement_consent');
          localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true');
          localStorage.setItem('user_guide_1_admin_v4_interactive', 'true');
          if (${JSON.stringify(route.role !== 'public')}) {
            const user = ${JSON.stringify(userForRole(route.role))};
            localStorage.setItem('auth_token', 'smoke-token');
            localStorage.setItem('refresh_token', 'smoke-refresh');
            localStorage.setItem('token_expires_at', String(Date.now() + 3600 * 1000));
            localStorage.setItem('auth_user', JSON.stringify(user));
          }
          if (${JSON.stringify(scenario === 'auth-states')}) {
            window.turnstile = {
              render(container, options = {}) {
                const marker = document.createElement('div');
                marker.setAttribute('data-testid', 'turnstile-smoke-rendered');
                marker.textContent = 'Turnstile smoke';
                container.appendChild(marker);
                setTimeout(() => {
                  if (typeof options.callback === 'function') {
                    options.callback('smoke-turnstile-token');
                  }
                }, 0);
                return 'smoke-turnstile-widget';
              },
              reset() {},
              remove() {},
            };
          }
          const routeSettings = ${JSON.stringify(publicSettingsForSmokeRoute(route))};
          Object.defineProperty(window, '__APP_CONFIG__', {
            configurable: true,
            get: () => routeSettings,
            set: () => {},
          });
        })();
      `,
    }, sessionId)
    return { targetId, sessionId }
  }

  for (const route of selectedRoutes) {
    currentRoute = route
    for (const viewport of viewports) {
      const consoleMessageStart = consoleMessages.length
      const apiRequestStart = apiRequests.length
      const page = await createPage(route, viewport)
      activeSessionId = page.sessionId
      await cdp.send('Emulation.setDeviceMetricsOverride', {
        width: viewport.width,
        height: viewport.height,
        deviceScaleFactor: 1,
        mobile: viewport.name === 'mobile',
      }, page.sessionId)

      const targetPath = scenario === 'auth-states' && route.path === '/auth/callback'
        ? '/auth/callback?code=smoke-code&state=smoke-state'
        : route.path
      const targetUrl = `${baseUrl}${targetPath}`
      await cdp.send('Page.navigate', { url: targetUrl }, page.sessionId)

      const evaluated = await cdp.send('Runtime.evaluate', {
        returnByValue: true,
        awaitPromise: true,
        expression: `
          new Promise((resolve) => {
            const done = () => {
              const selector = ${JSON.stringify(route.selector)};
              const expectedText = ${JSON.stringify(route.expectText || [])};
              const bodyText = document.body?.innerText || '';
              const doc = document.documentElement;
              const body = document.body;
              const scrollWidth = Math.max(doc?.scrollWidth || 0, body?.scrollWidth || 0);
              const overflowElements = Array.from(document.querySelectorAll('body *'))
                .map((element) => {
                  const rect = element.getBoundingClientRect();
                  return {
                    tag: element.tagName.toLowerCase(),
                    id: element.id || '',
                    className: typeof element.className === 'string' ? element.className.slice(0, 180) : '',
                    left: Math.round(rect.left),
                    right: Math.round(rect.right),
                    width: Math.round(rect.width),
                  };
                })
                .filter((element) => element.right > window.innerWidth + 2 || element.left < -2)
                .slice(0, 12);
              resolve({
                href: location.href,
                title: document.title,
                selectorFound: Boolean(document.querySelector(selector)),
                overflowX: Math.ceil(scrollWidth - window.innerWidth) > 2,
                scrollWidth,
                innerWidth: window.innerWidth,
                overflowElements,
                bodyText: bodyText.slice(0, 240),
                expectedTextFound: expectedText.every((text) => bodyText.includes(text)),
              });
            };
            const startedAt = Date.now();
            const timer = setInterval(() => {
              const expectedText = ${JSON.stringify(route.expectText || [])};
              const bodyText = document.body?.innerText || '';
              const selectorFound = Boolean(document.querySelector(${JSON.stringify(route.selector)}));
              const expectedTextFound = expectedText.every((text) => bodyText.includes(text));
              if (selectorFound && expectedTextFound) {
                clearInterval(timer);
                setTimeout(done, 750);
              } else if (Date.now() - startedAt > ${timeoutMs}) {
                clearInterval(timer);
                done();
              }
            }, 100);
          })
        `,
      }, page.sessionId)

      const scenarioResult = await runScenarioAction(cdp, page.sessionId, route)
      const scenarioNetwork = scenarioNetworkResult(route, apiRequests.slice(apiRequestStart))

      const screenshotPath = join(outputDir, routeFileName(route, viewport))
      const screenshot = await cdp.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false }, page.sessionId)
      await writeFile(screenshotPath, Buffer.from(screenshot.data, 'base64'))

      const value = evaluated.result.value
      const redirectedToLogin = route.role !== 'public' && new URL(value.href).pathname === '/login'
      const expectedTextFound = value.expectedTextFound !== false
      const runtimeErrors = consoleMessages
        .slice(consoleMessageStart)
        .filter((message) => message.type === 'error' || message.type === 'exception')
      const passed = value.selectorFound
        && !value.overflowX
        && !redirectedToLogin
        && expectedTextFound
        && scenarioResult.passed
        && scenarioNetwork.passed
        && runtimeErrors.length === 0
      results.push({
        route: route.path,
        viewport: viewport.name,
        scenario,
        scenarioAction: scenarioResult.scenarioAction,
        url: targetUrl,
        finalUrl: value.href,
        selector: route.selector,
        selectorFound: value.selectorFound,
        overflowX: value.overflowX,
        scrollWidth: value.scrollWidth,
        innerWidth: value.innerWidth,
        overflowElements: value.overflowElements,
        expectedText: route.expectText || [],
        expectedTextFound,
        scenarioSelectorFound: scenarioResult.selectorFound,
        scenarioTextFound: scenarioResult.textFound,
        schedulerDragSucceeded: scenarioResult.schedulerDragSucceeded ?? null,
        schedulerDragMechanism: scenarioResult.schedulerDragMechanism ?? null,
        scenarioError: scenarioResult.error || null,
        scenarioDiagnostics: scenarioResult.diagnostics || null,
        scenarioNetworkPassed: scenarioNetwork.passed,
        scenarioNetworkAssertions: scenarioNetwork.assertions,
        runtimeErrors,
        redirectedToLogin,
        screenshot: screenshotPath,
        passed,
      })
      await cdp.send('Target.closeTarget', { targetId: page.targetId }).catch(() => {})
      pageMetadata.delete(page.sessionId)
      schedulingEntriesBySession.delete(page.sessionId)
    }
  }

  const report = {
    generatedAt: new Date().toISOString(),
    baseUrl,
    outputDir,
    routes: selectedRoutes.map((route) => route.path),
    results,
    consoleMessages,
    apiRequests,
  }
  await writeFile(join(outputDir, 'report.json'), JSON.stringify(report, null, 2))

  cdp.close()
  await stopChromium(chrome)

  const failures = results.filter((result) => !result.passed)
  console.log(`Gateway route smoke wrote ${results.length} screenshots to ${outputDir}`)
  console.log(`Report: ${join(outputDir, 'report.json')}`)
  if (consoleMessages.length > 0) {
    console.log(`Console warnings/errors captured: ${consoleMessages.length}`)
  }
  if (failures.length > 0) {
    console.error('Failures:')
    failures.forEach((failure) => {
      console.error(`- ${failure.viewport} ${failure.route}: selectorFound=${failure.selectorFound}, overflowX=${failure.overflowX}, expectedTextFound=${failure.expectedTextFound}, scenarioSelectorFound=${failure.scenarioSelectorFound}, scenarioTextFound=${failure.scenarioTextFound}, scenarioNetworkPassed=${failure.scenarioNetworkPassed}, runtimeErrors=${failure.runtimeErrors.length}, redirectedToLogin=${failure.redirectedToLogin}`)
    })
    process.exit(1)
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
