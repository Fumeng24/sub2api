import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/api/auth', () => ({
  prepareOAuthBindAccessTokenCookie: vi.fn(),
  resolveWeChatOAuthStartStrict: (settings: {
    wechat_oauth_open_enabled?: boolean
    wechat_oauth_mp_enabled?: boolean
    wechat_oauth_mobile_enabled?: boolean
  } | null | undefined) => ({
    mode: settings?.wechat_oauth_open_enabled ? 'open' : settings?.wechat_oauth_mp_enabled ? 'mp' : null,
    openEnabled: settings?.wechat_oauth_open_enabled === true,
    mpEnabled: settings?.wechat_oauth_mp_enabled === true,
    mobileEnabled: settings?.wechat_oauth_mobile_enabled === true,
    isWeChatBrowser: false,
    unavailableReason: settings?.wechat_oauth_open_enabled || settings?.wechat_oauth_mp_enabled ? null : 'not_configured',
  }),
}))

describe('user api oauth binding urls', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/api/v1')
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('builds third-party bind urls against the bind start endpoint', async () => {
    const { buildOAuthBindingStartURL } = await import('@/api/user')

    expect(buildOAuthBindingStartURL('linuxdo', { redirectTo: '/settings/profile' })).toBe(
      'https://api.example.com/api/v1/auth/oauth/linuxdo/bind/start?redirect=%2Fsettings%2Fprofile&intent=bind_current_user'
    )
    expect(
      buildOAuthBindingStartURL('wechat', {
        redirectTo: '/settings/profile',
        wechatOAuthSettings: {
          wechat_oauth_open_enabled: true,
          wechat_oauth_mp_enabled: false,
          wechat_oauth_mobile_enabled: false
        }
      })
    ).toBe(
      'https://api.example.com/api/v1/auth/oauth/wechat/bind/start?redirect=%2Fsettings%2Fprofile&intent=bind_current_user&mode=open'
    )
  }, 30000)
})
