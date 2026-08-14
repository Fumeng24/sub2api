import { describe, expect, it } from 'vitest'
import type { UserMonitorView } from '@/api/channelMonitor'
import { resolveOverallMonitorStatus } from '@/custom/user/channelStatus'

function monitor(primaryStatus: UserMonitorView['primary_status']): UserMonitorView {
  return {
    id: 1,
    name: 'Smoke monitor',
    provider: 'openai',
    group_name: 'Codex Pro',
    primary_model: 'gpt-5.6',
    primary_status: primaryStatus,
    primary_latency_ms: 800,
    primary_ping_latency_ms: 100,
    availability_7d: 99.9,
    extra_models: [],
    timeline: [],
  }
}

describe('resolveOverallMonitorStatus', () => {
  it('does not report an empty monitor set as operational', () => {
    expect(resolveOverallMonitorStatus([])).toBe('degraded')
  })

  it('reports operational only when every monitor is operational', () => {
    expect(resolveOverallMonitorStatus([monitor('operational'), monitor('operational')]))
      .toBe('operational')
  })

  it('reports degraded when any monitor is not operational', () => {
    expect(resolveOverallMonitorStatus([monitor('operational'), monitor('failed')]))
      .toBe('degraded')
  })
})
