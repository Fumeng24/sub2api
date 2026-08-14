import type { UserSupportedModel } from '@/api/channels'

declare module '@/api/channels' {
  interface UserAvailableGroup {
    group_rate_discount_multiplier?: number | null
    discounted_rate_multiplier?: number | null
    group_rate_discount_name?: string | null
    group_rate_discount_schedule_mode?: string | null
    group_rate_discount_start_at?: string | null
    group_rate_discount_end_at?: string | null
    group_rate_discount_weekdays?: number[] | null
    group_rate_discount_daily_start_time?: string | null
    group_rate_discount_daily_end_time?: string | null
    group_rate_discount_timezone?: string | null
    supported_models?: UserSupportedModel[] | null
  }

  interface UserChannelPlatformSection {
    endpoints?: string[] | null
    supported_endpoint_types?: string[] | null
  }
}

export {}
