import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'
import en from '@/custom/i18n/locales/en'
import zh from '@/custom/i18n/locales/zh'

const source = readFileSync(resolve(__dirname, '../WegooUpstreamsView.vue'), 'utf8')

function readLocaleValue(locale: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((value, key) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[key]
  }, locale)
}

describe('WegooUpstreamsView', () => {
  it('has Chinese and English text for every static upstream translation key', () => {
    const keys = new Set(
      [...source.matchAll(/t\('([^']*admin\.upstreams\.[^']+)'/g)].map((match) => match[1]),
    )

    expect(keys.size).toBeGreaterThan(60)
    for (const key of keys) {
      expect(readLocaleValue(zh as Record<string, unknown>, key), `missing zh key ${key}`).toBeTruthy()
      expect(readLocaleValue(en as Record<string, unknown>, key), `missing en key ${key}`).toBeTruthy()
    }
  })

  it('keeps generated and historical account workflows distinct', () => {
    expect(source).toContain("result = await adminAPI.upstreams.probe(result.id)")
    expect(source).toContain('rate_multiplier: group?.rate_multiplier')
    expect(source).toContain('admin.upstreams.detail.bindPreservesAccount')
    expect(source).toContain('adminAPI.upstreams.bindAccounts(selectedUpstream.value.id, selectedCandidateIdsArray.value)')
    expect(source).not.toContain('allowRebind')
  })

  it('polls persisted upstream state without turning each UI refresh into a remote probe', () => {
    expect(source).toContain("defaultEnabled: true")
    expect(source).toContain("onRefresh: () => loadUpstreams(true)")
    expect(source).toContain("admin.upstreams.refreshStale")
  })

  it('shows redacted management and account-level probe failure reasons', () => {
    expect(source).toContain('item.metadata?.management_hint')
    expect(source).toContain('item.metadata?.account_billing')
    expect(source).toContain('failureScopeLabel(selectedUpstream, reason)')
    expect(source).toContain("t('admin.upstreams.failureReasonsTitle')")
    expect(source).toContain("code === 'ok' && !hasRate")
  })

  it('shows local groups instead of protocol model counts in the upstream list', () => {
    expect(source).toContain("t('admin.upstreams.columns.localGroups')")
    expect(source).toContain('item.local_groups || []')
    expect(source).toContain("t('admin.upstreams.notBound')")
    expect(source).not.toContain("t('admin.upstreams.columns.capabilities')")
    expect(source).not.toContain('protocol.models.length')
  })

  it('does not present optional protocol diagnostics as a healthy upstream failure', () => {
    expect(source).toContain("if (item.status !== 'healthy')")
  })

  it('manages bound account groups through the remote transaction endpoint', () => {
    expect(source).toContain('adminAPI.upstreams.changeAccountUpstreamGroup(upstreamID, account.id')
    expect(source).toContain("selectedUpstream.value?.kind === 'sub2api' && target.id == null")
    expect(source).toContain('groupCatalogueIsStale()')
    expect(source).toContain('accountGroupChanging.add(account.id)')
    expect(source).toContain('accountGroupChanging.delete(account.id)')
  })

  it('previews automatic names before applying them', () => {
    expect(source).toContain('adminAPI.upstreams.previewAccountRenames()')
    expect(source).toContain('adminAPI.upstreams.applyAccountRenames()')
    expect(source).toContain("'upstream group is not currently verified': 'groupNotVerified'")
  })

  it('keeps catalogue models unselected until a real request succeeds', () => {
    expect(source).toContain('currentModels.value = [...result.available_models]')
    expect(source).toContain('generationForm.models = []')
    expect(source).toContain(':disabled="!modelProbeResults[model]?.success"')
    expect(source).toContain('if (!generationForm.models.includes(model)) generationForm.models.push(model)')
    expect(source).toContain('generationForm.models = generationForm.models.filter(item => item !== model)')
  })

  it('only bulk-selects models with a current successful verification', () => {
    expect(source).toContain('const verifiedModels = computed')
    expect(source).toContain("modelProbeResults[model]?.success")
    expect(source).toContain('generationForm.models = allModelsSelected.value ? [] : [...verifiedModels.value]')
    expect(source).toContain('generationForm.models = currentModels.value.filter(model => modelProbeResults[model]?.success)')
  })
})
