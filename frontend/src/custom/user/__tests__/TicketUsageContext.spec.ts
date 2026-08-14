import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import enLocale from '@/custom/i18n/locales/en'
import zhLocale from '@/custom/i18n/locales/zh'

const usageView = readFileSync(resolve(__dirname, '../WegooUsageView.vue'), 'utf8')
const ticketsView = readFileSync(resolve(__dirname, '../TicketsView.vue'), 'utf8')
const userErrorRequestsTable = readFileSync(resolve(__dirname, '../WegooUserErrorRequestsTable.vue'), 'utf8')
const contextLink = readFileSync(resolve(__dirname, '../../tickets/TicketContextLink.vue'), 'utf8')

function localeValue(locale: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((current, key) => {
    if (current === null || typeof current !== 'object' || Array.isArray(current)) return undefined
    return (current as Record<string, unknown>)[key]
  }, locale)
}

function expectLocalizedString(path: string): void {
  expect(localeValue(zhLocale, path)).toEqual(expect.any(String))
  expect(localeValue(enLocale, path)).toEqual(expect.any(String))
}

describe('ticket usage context handoff', () => {
  it('passes redacted issue-record diagnostics into ticket creation', () => {
    expect(usageView).toContain("context_type: 'request'")
    expect(usageView).toContain('status_code: String(row.status_code')
    expect(usageView).toContain('category: row.category')
    expect(usageView).toContain('platform: row.platform')
    expect(usageView).toContain('error_message: row.message')
    expect(usageView).toContain('api_key_name: row.key_name')
    expect(userErrorRequestsTable).toContain("(e: 'createTicket', v: UserErrorRequest): void")
    expect(userErrorRequestsTable).toContain("emit('createTicket', row)")
  })

  it('persists route diagnostics in ticket context data after templates load', () => {
    expect(ticketsView).toContain('ROUTE_CONTEXT_QUERY_KEYS')
    expect(ticketsView).toContain('applyCreateContextFromQuery()')
    expect(ticketsView).toContain('for (const [key, value] of Object.entries(createContextData))')
    expect(ticketsView).toContain("contextType === 'usage'")
    expect(ticketsView).toContain("contextType === 'request'")
    expect(ticketsView).toContain('tickets.form.usageBodyPrefill')
    expect(ticketsView).toContain('tickets.form.requestBodyPrefill')
  })

  it('maps issue-record categories and status codes to actionable ticket guidance', () => {
    expect(ticketsView).toContain('resolveRequestAdviceKey(')
    expect(ticketsView).toContain("statusCode === 429")
    expect(ticketsView).toContain("statusCode >= 500")
    expect(ticketsView).toContain('requestAdviceText')
    expect(ticketsView).toContain('out.recommended_action = requestAdviceText.value')
    for (const key of [
      'auth',
      'rate_limit',
      'quota',
      'invalid_request',
      'service_unavailable',
      'upstream',
      'internal',
      'cyber',
      'default',
    ]) {
      expectLocalizedString(`tickets.form.requestAdvice.${key}`)
    }
    expectLocalizedString('tickets.form.requestAdviceTitle')
    expectLocalizedString('tickets.contextFields.recommended_action')
  })

  it('shows context-aware attachment hints for request, order, and invoice handoff', () => {
    expect(ticketsView).toContain('contextAttachmentHint')
    expect(ticketsView).toContain("createForm.context_type === 'request'")
    expect(ticketsView).toContain("createForm.context_type === 'order'")
    expect(ticketsView).toContain("createForm.context_type === 'invoice'")
    for (const key of ['request', 'order', 'invoice']) {
      expectLocalizedString(`tickets.form.attachmentHint.${key}`)
    }
  })

  it('links common ticket context types back to their source pages', () => {
    expect(contextLink).toContain("case 'usage':")
    expect(contextLink).toContain("case 'request':")
    expect(contextLink).toContain("case 'request_id':")
    expect(contextLink).toContain("case 'order':")
    expect(contextLink).toContain("case 'invoice':")
    expect(contextLink).toContain('/admin/invoices?search=')
    expect(contextLink).toContain('/orders?search=')
  })

  it('defines user-visible context labels in both locales', () => {
    for (const key of [
      'request_id',
      'model',
      'api_key_id',
      'group_name',
      'actual_cost',
      'duration_ms',
      'status_code',
      'category',
      'platform',
      'error_message',
      'api_key_name',
    ]) {
      expectLocalizedString(`tickets.contextFields.${key}`)
    }
    expectLocalizedString('tickets.form.usageBodyPrefill')
    expectLocalizedString('tickets.form.requestBodyPrefill')
  })
})
