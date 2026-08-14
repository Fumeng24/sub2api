/**
 * Local type overlays that extend the upstream public frontend contract.
 * Keep upstream-owned declarations in `src/types/index.ts` and expose local
 * additions through the thin hooks defined here.
 */

export type UserRole = 'admin' | 'support' | 'user'

export interface AdminUserCustom {
  group_discounts?: Record<number, number>
}

export interface UserAffiliateDetailCustom {
  bind_bonus_amount: number
  can_bind_inviter: boolean
  can_claim_bind_bonus: boolean
  bind_bonus_claimed_at?: string
  rebate_duration_days: number
}

export interface PublicSettingsCustom {
  payment_balance_recharge_multiplier?: number
  group_rate_discount?: ActiveGroupRateDiscount | null
  upcoming_group_rate_discount?: ActiveGroupRateDiscount | null
}

export interface GroupRateDiscountSettings {
  enabled: boolean
  name: string
  discount_multiplier: number
  schedule_mode: 'once' | 'weekly' | string
  start_at: string
  end_at: string
  weekdays: number[]
  daily_start_time: string
  daily_end_time: string
  group_ids: number[]
}

export interface ActiveGroupRateDiscount {
  name: string
  discount_multiplier: number
  schedule_mode: 'once' | 'weekly' | string
  start_at: string
  end_at: string
  weekdays: number[]
  daily_start_time: string
  daily_end_time: string
  timezone?: string
  group_ids: number[]
}

// ==================== Support Ticket Types ====================

export type TicketStatus = 'open' | 'pending' | 'resolved' | 'closed'
export type TicketPriority = 'low' | 'normal' | 'high' | 'urgent'
export type TicketCategory = 'general' | 'billing' | 'usage' | 'technical' | 'account'
export type TicketMessageSenderType = 'user' | 'admin' | 'system'
export type TicketMessageVisibility = 'public' | 'internal'
export type TicketTemplateFieldType =
  | 'text'
  | 'textarea'
  | 'select'
  | 'group_select'
  | 'recent_orders'
  | 'amount'
  | 'image'
  | 'attachments'

export interface TicketAttachment {
  name: string
  url: string
  content_type?: string
  size?: number
}

export interface TicketMessage {
  id: number
  ticket_id: number
  sender_type: TicketMessageSenderType
  sender_id?: number | null
  sender_name: string
  visibility: TicketMessageVisibility
  body: string
  attachments?: TicketAttachment[]
  edited_at?: string | null
  created_at: string
  updated_at: string
}

export interface Ticket {
  id: number
  ticket_no: string
  user_id: number
  user_email: string
  user_name: string
  subject: string
  category: TicketCategory
  priority: TicketPriority
  status: TicketStatus
  source: string
  template_key: string
  context_type: string
  context_id: string
  context_data?: Record<string, unknown>
  assignee_id?: number | null
  escalated_at?: string | null
  escalated_by?: number | null
  escalation_reason?: string
  last_message_at: string
  last_user_message_at?: string | null
  last_admin_message_at?: string | null
  resolved_at?: string | null
  closed_at?: string | null
  unread_count: number
  messages?: TicketMessage[]
  created_at: string
  updated_at: string
}

export interface TicketUnreadSummary {
  total: number
  open: number
  pending: number
  resolved: number
  closed: number
}

export interface TicketStats {
  total: number
  open: number
  pending: number
  resolved: number
  closed: number
  unassigned: number
  assigned_to_me: number
  handled_by_me: number
  escalated: number
  sla_overdue: number
  unread: number
}

export interface TicketSupportPermissions {
  can_view_all: boolean
  can_view_escalated: boolean
  can_internal_note: boolean
  can_close: boolean
  can_transfer: boolean
  can_batch_update: boolean
  can_update_priority: boolean
  can_update_category: boolean
  can_reply_unassigned: boolean
  can_reply_assigned_to_self: boolean
  can_escalate: boolean
}

export interface TicketTemplateOption {
  value: string
  label: string
}

export interface TicketTemplateField {
  key: string
  label: string
  type: TicketTemplateFieldType | string
  required?: boolean
  min_length?: number
  max_length?: number
  min_value?: number
  options?: TicketTemplateOption[]
  description?: string
  placeholder?: string
}

export interface TicketTemplate {
  key: string
  name: string
  description?: string
  category: TicketCategory | string
  priority: TicketPriority | string
  subject_template?: string
  body_min_length?: number
  requires_super_admin: boolean
  auto_assign_super_admin: boolean
  context_type?: string
  fields?: TicketTemplateField[]
}

export interface TicketSLASettings {
  enabled: boolean
  first_response_minutes: number
  reminder_before_minutes: number
  auto_escalate_after_minutes: number
  reminder_notifications: boolean
  auto_escalate_notifications: boolean
  auto_close_resolved_days: number
  worker_interval_seconds: number
}

export interface TicketSystemSettings {
  templates: TicketTemplate[]
  support_permissions: TicketSupportPermissions
  sla: TicketSLASettings
}

export interface TicketAdminCapabilities {
  role: UserRole | string
  is_super_admin: boolean
  support_permissions: TicketSupportPermissions
  can_view_all: boolean
  can_view_escalated: boolean
  can_internal_note: boolean
  can_close: boolean
  can_transfer: boolean
  can_batch_update: boolean
  can_update_priority: boolean
  can_update_category: boolean
  can_reply_unassigned: boolean
  can_reply_assigned_to_self: boolean
  can_escalate: boolean
  can_adjust_balance: boolean
}

export type TicketPrefillGroup = Pick<
  import('../../types').Group,
  'id' | 'name' | 'rate_multiplier' | 'platform' | 'status'
>

export interface TicketPrefillOrder {
  id: number
  order_no?: string
  amount: number
  pay_amount?: number
  currency?: string
  status: string
  order_type?: string
  payment_type?: string
  out_trade_no?: string
  created_at: string
}

export interface TicketPrefillData {
  groups?: TicketPrefillGroup[]
  recent_orders?: TicketPrefillOrder[]
}

export interface CreateTicketRequest {
  subject: string
  body: string
  category?: TicketCategory | string
  priority?: TicketPriority | string
  template_key?: string
  context_type?: string
  context_id?: string
  context_data?: Record<string, unknown>
  attachments?: TicketAttachment[]
}

export interface AddTicketMessageRequest {
  body: string
  internal?: boolean
  attachments?: TicketAttachment[]
}

export interface UpdateTicketRequest {
  status?: TicketStatus | string
  priority?: TicketPriority | string
  category?: TicketCategory | string
  assignee_id?: number
}

// ==================== Group and Scheduling Extensions ====================

export interface GroupCustom<TModelsListConfig = unknown> {
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
  force_openai_priority?: boolean
  openai_stable_low_ttft?: boolean
  models_list_config?: TModelsListConfig
}

export interface GroupAutoSortConfig {
  enabled?: boolean
  basis?: 'rate' | 'experience' | 'availability' | 'latency'
}

export interface AdminGroupCustom {
  auto_sort_config?: GroupAutoSortConfig
}

export type AccountSchedulingRole = 'primary' | 'backup'

export interface AccountSchedulingEntry {
  account_id: number
  group_id: number
  role: AccountSchedulingRole
  weight: number
  sort_order: number
  scheduling_configured: boolean
  account?: import('../../types').Account
  state?: string
  block_reason?: string
  group_reserve?: boolean
  group_reserve_until?: string | null
  group_reserve_reason?: string
  recent_user_avg_first_token_ms?: number | null
  recent_user_first_token_sample_count?: number
}

export interface AccountSchedulingConfig {
  accounts: AccountSchedulingEntry[]
}

export interface UpdateAccountSchedulingRequest {
  accounts: Array<{
    account_id: number
    role: AccountSchedulingRole
    weight: number
    sort_order: number
    scheduling_configured?: boolean
  }>
}

export type ApiKeyCategory = 'openai' | 'anthropic' | 'other'

export interface ApiKeyCustom {
  category: ApiKeyCategory
}

export interface CreateApiKeyRequestCustom {
  category?: ApiKeyCategory
}

export interface UpdateApiKeyRequestCustom {
  category?: ApiKeyCategory
}

export interface GroupMutationCustom {
  force_openai_priority?: boolean
  openai_stable_low_ttft?: boolean
}

export interface UpdateGroupRequestCustom extends GroupMutationCustom {
  auto_sort_config?: GroupAutoSortConfig
}

export interface GroupRateChangeNotificationRequest {
  new_rate_multiplier: number
  window_minutes?: number
  effective_at?: string
  message?: string
}

export interface GroupRateChangeNotificationUser {
  user_id: number
  email: string
  username: string
  request_count: number
  actual_cost: number
  last_used_at: string
}

export interface GroupRateChangeNotificationPreview {
  group_id: number
  group_name: string
  old_rate_multiplier: number
  new_rate_multiplier: number
  window_minutes: number
  effective_at: string
  message?: string
  user_count: number
  skipped_count: number
  users: GroupRateChangeNotificationUser[]
}

export interface GroupRateChangeNotificationSendResult {
  group_id: number
  user_count: number
  sent: number
  skipped: number
  failed: number
  last_error?: string
  effective_at: string
}

// ==================== Account, Usage, and User Extensions ====================

export interface AccountExtraCustom {
  model_rate_limits?: Record<string, { rate_limited_at: string; rate_limit_reset_at: string; reason?: string }>
}

export interface AccountCustom {
  temp_unschedulable_status_code?: number | null
  upstream_id?: number | null
}

export type RedeemCodeTypeCustom =
  | 'balance'
  | 'concurrency'
  | 'subscription'
  | 'invitation'
  | 'admin_balance'
  | 'admin_concurrency'
  | 'affiliate_balance'

export type BalanceBusinessCategory =
  | ''
  | 'recharge'
  | 'manual_collection'
  | 'manual_refund'
  | 'gift_compensation'
  | 'gift_reversal'
  | 'system_service_fee'
  | 'affiliate_reward'

export interface RedeemCodeCustom {
  business_category: BalanceBusinessCategory
}

export interface BatchUpdateRedeemCodeFieldsCustom {
  business_category?: BalanceBusinessCategory
}

export interface UpdateUserRequestCustom {
  group_discounts?: Record<number, number | null>
}

export interface UserSubscriptionCustom {
  auto_reset_daily: boolean
}
