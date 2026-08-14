<template>
  <BaseDialog
    :show="show"
    :title="t('keys.useKeyModal.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="use-key-modal-content space-y-4">
      <div
        v-if="!platform"
        class="flex items-start gap-3 rounded-lg border border-[color:color-mix(in_srgb,var(--apple-warning)_28%,var(--apple-border))] bg-[color-mix(in_srgb,var(--apple-warning)_8%,var(--apple-surface))] p-4"
      >
        <Icon name="exclamationTriangle" size="md" class="mt-0.5 flex-shrink-0 text-[var(--apple-warning)]" />
        <div class="min-w-0">
          <p class="text-sm font-medium text-[var(--apple-text)]">
            {{ t('keys.useKeyModal.noGroupTitle') }}
          </p>
          <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
            {{ t('keys.useKeyModal.noGroupDescription') }}
          </p>
        </div>
      </div>

      <template v-else>
        <section class="rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] p-3 shadow-sm sm:p-4">
          <div class="flex items-start gap-3">
            <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] text-[var(--apple-muted)]">
              <Icon name="key" size="sm" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-[var(--apple-text)]">
                {{ t('keys.useKeyModal.title') }}
              </p>
              <p class="mt-1 text-sm leading-6 text-[var(--apple-muted)]">
                {{ platformDescription }}
              </p>
            </div>
          </div>

          <div class="mt-4 grid gap-2 sm:grid-cols-2">
            <div
              v-if="groupName"
              class="min-w-0 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-3 py-2"
              data-testid="use-key-group-context"
            >
              <p class="text-[11px] font-semibold uppercase tracking-wide text-[var(--apple-muted-2)]">
                {{ t('keys.useKeyModal.connection.group') }}
              </p>
              <p class="mt-1 truncate text-sm font-medium text-[var(--apple-text)]">
                {{ groupName }}
              </p>
              <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
                {{ t('keys.useKeyModal.connection.groupHint') }}
              </p>
            </div>

            <div
              class="min-w-0 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-3 py-2"
              data-testid="use-key-platform-context"
            >
              <p class="text-[11px] font-semibold uppercase tracking-wide text-[var(--apple-muted-2)]">
                {{ t('keys.useKeyModal.connection.platform') }}
              </p>
              <p class="mt-1 truncate text-sm font-medium text-[var(--apple-text)]">
                {{ platformLabel }}
              </p>
              <p class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
                {{ t('keys.useKeyModal.connection.platformHint') }}
              </p>
            </div>
          </div>

          <div
            v-if="endpointOptions.length"
            class="mt-3 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-3 py-2"
            data-testid="use-key-endpoint-context"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <p class="text-[11px] font-semibold uppercase tracking-wide text-[var(--apple-muted-2)]">
                {{ t('keys.useKeyModal.connection.endpoints') }}
              </p>
              <span class="rounded-md bg-[var(--apple-surface)] px-2 py-0.5 text-[11px] text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border-soft)]">
                {{ t('keys.useKeyModal.connection.endpointCount', { count: endpointOptions.length }) }}
              </span>
            </div>
            <div class="mt-2 grid gap-2">
              <div
                v-for="endpoint in endpointOptions"
                :key="endpoint.endpoint"
                class="min-w-0 rounded-md bg-[var(--apple-surface)] px-2.5 py-2 ring-1 ring-[color:var(--apple-border-soft)]"
              >
                <div class="flex min-w-0 flex-wrap items-center gap-1.5">
                  <span class="min-w-0 truncate text-xs font-medium text-[var(--apple-text)]">
                    {{ endpoint.name }}
                  </span>
                  <span
                    v-if="endpoint.isDefault"
                    class="rounded bg-[var(--apple-surface-elevated)] px-1.5 py-px text-[10px] text-[var(--apple-muted)]"
                  >
                    {{ t('keys.useKeyModal.connection.defaultEndpoint') }}
                  </span>
                </div>
                <p class="mt-1 break-all font-mono text-xs leading-5 text-[var(--apple-muted)]">
                  {{ endpoint.endpoint }}
                </p>
                <p v-if="endpoint.description" class="mt-1 text-xs leading-5 text-[var(--apple-muted)]">
                  {{ endpoint.description }}
                </p>
              </div>
            </div>
          </div>

          <nav
            v-if="clientTabs.length"
            class="mt-4 grid min-w-0 grid-cols-2 gap-1 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-1 sm:flex sm:flex-wrap"
            aria-label="Client"
          >
            <button
              v-for="tab in clientTabs"
              :key="tab.id"
              type="button"
              @click="activeClientTab = tab.id"
              :class="[
                'flex min-w-0 items-center justify-center gap-1.5 rounded-md px-2.5 py-2 text-center text-xs font-medium leading-4 transition-colors sm:w-auto sm:text-sm',
                activeClientTab === tab.id
                  ? 'bg-[var(--apple-surface)] text-[var(--apple-text)] shadow-sm ring-1 ring-[color:var(--apple-border-soft)]'
                  : 'text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'
              ]"
            >
              <Icon :name="tab.icon" size="sm" class="flex-shrink-0" />
              <span class="min-w-0 break-words">{{ tab.label }}</span>
            </button>
          </nav>
        </section>

        <nav
          v-if="showShellTabs"
          class="grid min-w-0 grid-cols-2 gap-1 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-1 sm:flex sm:flex-wrap"
          aria-label="Tabs"
        >
          <button
            v-for="tab in currentTabs"
            :key="tab.id"
            type="button"
            @click="activeTab = tab.id"
            :class="[
              'flex min-w-0 items-center justify-center gap-1.5 rounded-md px-2.5 py-2 text-center text-xs font-medium leading-4 transition-colors sm:w-auto sm:text-sm',
              activeTab === tab.id
                ? 'bg-[var(--apple-surface)] text-[var(--apple-text)] shadow-sm ring-1 ring-[color:var(--apple-border-soft)]'
                : 'text-[var(--apple-muted)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-text)]'
            ]"
          >
            <Icon :name="tab.icon" size="sm" class="flex-shrink-0" />
            <span class="min-w-0 break-words">{{ tab.label }}</span>
          </button>
        </nav>

        <div class="space-y-3">
          <div
            v-for="(file, index) in currentFiles"
            :key="index"
            class="relative"
          >
            <p
              v-if="file.hint"
              class="mb-2 flex items-center gap-1.5 rounded-md border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] px-2.5 py-1.5 text-xs leading-5 text-[var(--apple-muted)]"
            >
              <Icon name="infoCircle" size="xs" class="flex-shrink-0 text-[var(--apple-muted-2)]" />
              {{ file.hint }}
            </p>
            <div class="overflow-hidden rounded-lg border border-slate-800 bg-slate-950 shadow-sm">
              <div class="flex min-w-0 items-center justify-between gap-3 border-b border-white/10 bg-slate-900/90 px-3 py-2.5 sm:px-4">
                <span class="min-w-0 truncate font-mono text-xs text-slate-300">{{ file.path }}</span>
                <button
                  type="button"
                  @click="copyContent(file.content, index)"
                  class="flex flex-shrink-0 items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
                  :class="copiedIndex === index
                    ? 'bg-emerald-500/15 text-emerald-300'
                    : 'bg-white/10 text-slate-300 hover:bg-white/15 hover:text-white'"
                >
                  <Icon
                    :name="copiedIndex === index ? 'check' : 'copy'"
                    size="xs"
                    :stroke-width="copiedIndex === index ? 2 : 1.8"
                  />
                  {{ copiedIndex === index ? t('keys.useKeyModal.copied') : t('keys.useKeyModal.copy') }}
                </button>
              </div>
              <pre class="max-w-full overflow-x-auto p-3 font-mono text-xs leading-5 text-slate-100 sm:p-4 sm:text-sm"><code v-if="file.highlighted" v-html="file.highlighted"></code><code v-else v-text="file.content"></code></pre>
            </div>
          </div>
        </div>

        <div
          v-if="showPlatformNote"
          class="use-key-platform-note flex items-start gap-2.5 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface-elevated)] p-3"
        >
          <Icon name="infoCircle" size="sm" class="mt-0.5 flex-shrink-0 text-[var(--apple-muted-2)]" />
          <p class="text-sm leading-6 text-[var(--apple-muted)]">
            {{ platformNote }}
          </p>
        </div>
      </template>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button
          @click="emit('close')"
          class="btn btn-secondary"
        >
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/custom/common/WegooBaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { CustomEndpoint, GroupPlatform } from '@/types'
import {
  CLAUDE_CC_SWITCH_OPUS_MODEL,
  CLAUDE_CC_SWITCH_SONNET_MODEL,
  GEMINI_CC_SWITCH_MODEL,
  GROK_CC_SWITCH_MODEL,
  OPENAI_CC_SWITCH_CODEX_MODEL
} from '@/custom/keys/ccswitchImport'

interface Props {
  show: boolean
  apiKey: string
  baseUrl: string
  platform: GroupPlatform | null
  allowMessagesDispatch?: boolean
  groupName?: string
  customEndpoints?: CustomEndpoint[]
}

interface Emits {
  (e: 'close'): void
}

interface TabConfig {
  id: string
  label: string
  icon: 'terminal' | 'sparkles' | 'cpu' | 'bolt'
}

interface FileConfig {
  path: string
  content: string
  hint?: string  // Optional hint message for this file
  highlighted?: string
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const { copyToClipboard: clipboardCopy } = useClipboard()

const copiedIndex = ref<number | null>(null)
const activeTab = ref<string>('unix')
const activeClientTab = ref<string>('claude')

const endpointOptions = computed(() => {
  const items: Array<{ name: string; endpoint: string; description: string; isDefault: boolean }> = []
  if (props.baseUrl) {
    items.push({
      name: t('keys.useKeyModal.connection.primaryEndpoint'),
      endpoint: props.baseUrl,
      description: '',
      isDefault: true
    })
  }

  for (const endpoint of props.customEndpoints || []) {
    if (!endpoint.endpoint) continue
    items.push({
      name: endpoint.name || t('keys.useKeyModal.connection.customEndpoint'),
      endpoint: endpoint.endpoint,
      description: endpoint.description || '',
      isDefault: false
    })
  }
  return items
})

const groupName = computed(() => props.groupName?.trim() || '')

const platformLabel = computed(() => {
  switch (props.platform) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    case 'antigravity':
      return 'Antigravity'
    case 'grok':
      return 'Grok'
    default:
      return t('common.unknown')
  }
})

// Reset tabs when platform changes
const defaultClientTab = computed(() => {
  switch (props.platform) {
    case 'openai':
      return 'codex'
    case 'gemini':
      return 'gemini'
    case 'antigravity':
      return 'claude'
    case 'grok':
      return 'grok'
    default:
      return 'claude'
  }
})

watch(() => props.platform, () => {
  activeTab.value = 'unix'
  activeClientTab.value = defaultClientTab.value
}, { immediate: true })

// Reset shell tab when client changes
watch(activeClientTab, () => {
  activeTab.value = 'unix'
})

const clientTabs = computed((): TabConfig[] => {
  if (!props.platform) return []
  switch (props.platform) {
    case 'openai': {
      const tabs: TabConfig[] = [
        { id: 'openai-sdk', label: t('keys.useKeyModal.cliTabs.openaiSdk'), icon: 'terminal' },
        { id: 'anthropic-sdk', label: t('keys.useKeyModal.cliTabs.anthropicSdk'), icon: 'terminal' },
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: 'terminal' },
        { id: 'codex-ws', label: t('keys.useKeyModal.cliTabs.codexCliWs'), icon: 'terminal' },
      ]
      if (props.allowMessagesDispatch) {
        tabs.push({ id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: 'terminal' })
      }
      tabs.push({ id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: 'terminal' })
      return tabs
    }
    case 'gemini':
      return [
        { id: 'openai-sdk', label: t('keys.useKeyModal.cliTabs.openaiSdk'), icon: 'terminal' },
        { id: 'gemini', label: t('keys.useKeyModal.cliTabs.geminiCli'), icon: 'sparkles' },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: 'terminal' }
      ]
    case 'antigravity':
      return [
        { id: 'openai-sdk', label: t('keys.useKeyModal.cliTabs.openaiSdk'), icon: 'terminal' },
        { id: 'anthropic-sdk', label: t('keys.useKeyModal.cliTabs.anthropicSdk'), icon: 'terminal' },
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: 'terminal' },
        { id: 'gemini', label: t('keys.useKeyModal.cliTabs.geminiCli'), icon: 'sparkles' },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: 'terminal' }
      ]
    case 'grok':
      return [
        { id: 'grok', label: t('keys.useKeyModal.cliTabs.grokCli'), icon: 'terminal' },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: 'terminal' }
      ]
    default:
      return [
        { id: 'openai-sdk', label: t('keys.useKeyModal.cliTabs.openaiSdk'), icon: 'terminal' },
        { id: 'anthropic-sdk', label: t('keys.useKeyModal.cliTabs.anthropicSdk'), icon: 'terminal' },
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: 'terminal' },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: 'terminal' }
      ]
  }
})

// Shell tabs (3 types for environment variable based configs)
const shellTabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: 'terminal' },
  { id: 'cmd', label: 'Windows CMD', icon: 'cpu' },
  { id: 'powershell', label: 'PowerShell', icon: 'bolt' }
]

// OpenAI tabs (2 OS types)
const openaiTabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: 'terminal' },
  { id: 'windows', label: 'Windows', icon: 'cpu' }
]

const showShellTabs = computed(() => !['opencode', 'openai-sdk', 'anthropic-sdk'].includes(activeClientTab.value))

const currentTabs = computed(() => {
  if (!showShellTabs.value) return []
  if (activeClientTab.value === 'codex' || activeClientTab.value === 'codex-ws' || activeClientTab.value === 'grok') {
    return openaiTabs
  }
  return shellTabs
})

const platformDescription = computed(() => {
  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.description')
      }
      if (activeClientTab.value.endsWith('-sdk')) {
        return t('keys.useKeyModal.sdk.description')
      }
      return t('keys.useKeyModal.openai.description')
    case 'gemini':
      return t('keys.useKeyModal.gemini.description')
    case 'antigravity':
      return t('keys.useKeyModal.antigravity.description')
    case 'grok':
      return t('keys.useKeyModal.grok.description')
    default:
      return t('keys.useKeyModal.description')
  }
})

const platformNote = computed(() => {
  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.note')
      }
      if (activeClientTab.value.endsWith('-sdk')) {
        return t('keys.useKeyModal.sdk.note')
      }
      return activeTab.value === 'windows'
        ? t('keys.useKeyModal.openai.noteWindows')
        : t('keys.useKeyModal.openai.note')
    case 'gemini':
      return t('keys.useKeyModal.gemini.note')
    case 'antigravity':
      return activeClientTab.value === 'claude'
        ? t('keys.useKeyModal.antigravity.claudeNote')
        : t('keys.useKeyModal.antigravity.geminiNote')
    case 'grok':
      return activeTab.value === 'windows'
        ? t('keys.useKeyModal.grok.noteWindows')
        : t('keys.useKeyModal.grok.note')
    default:
      return t('keys.useKeyModal.note')
  }
})

const showPlatformNote = computed(() => activeClientTab.value !== 'opencode')

const escapeHtml = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const wrapToken = (className: string, value: string) =>
  `<span class="${className}">${escapeHtml(value)}</span>`

const keyword = (value: string) => wrapToken('text-emerald-300', value)
const variable = (value: string) => wrapToken('text-sky-200', value)
const operator = (value: string) => wrapToken('text-slate-400', value)
const string = (value: string) => wrapToken('text-amber-200', value)
const comment = (value: string) => wrapToken('text-slate-500', value)

// Syntax highlighting helpers
// Generate file configs based on platform and active tab
const currentFiles = computed((): FileConfig[] => {
  const baseUrl = props.baseUrl || window.location.origin
  const apiKey = props.apiKey
  const baseRoot = baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  const ensureV1 = (value: string) => {
    const trimmed = value.replace(/\/+$/, '')
    return trimmed.endsWith('/v1') ? trimmed : `${trimmed}/v1`
  }
  const apiBase = ensureV1(baseRoot)
  const antigravityBase = ensureV1(`${baseRoot}/antigravity`)
  const antigravityGeminiBase = (() => {
    const trimmed = `${baseRoot}/antigravity`.replace(/\/+$/, '')
    return trimmed.endsWith('/v1beta') ? trimmed : `${trimmed}/v1beta`
  })()
  const geminiBase = (() => {
    const trimmed = baseRoot.replace(/\/+$/, '')
    return trimmed.endsWith('/v1beta') ? trimmed : `${trimmed}/v1beta`
  })()

  if (activeClientTab.value === 'openai-sdk') {
    return [generateOpenAISdkFile(apiBase, apiKey)]
  }
  if (activeClientTab.value === 'anthropic-sdk') {
    return [generateAnthropicSdkFile(props.platform === 'antigravity' ? antigravityBase : baseRoot, apiKey)]
  }

  if (activeClientTab.value === 'opencode') {
    switch (props.platform) {
      case 'anthropic':
        return [generateOpenCodeConfig('anthropic', apiBase, apiKey)]
      case 'openai':
        return [generateOpenCodeConfig('openai', apiBase, apiKey)]
      case 'gemini':
        return [generateOpenCodeConfig('gemini', geminiBase, apiKey)]
      case 'antigravity':
        return [
          generateOpenCodeConfig('antigravity-claude', antigravityBase, apiKey, 'opencode.json (Claude)'),
          generateOpenCodeConfig('antigravity-gemini', antigravityGeminiBase, apiKey, 'opencode.json (Gemini)')
        ]
      case 'grok':
        return [generateOpenCodeConfig('grok', apiBase, apiKey)]
      default:
        return [generateOpenCodeConfig('openai', apiBase, apiKey)]
    }
  }

  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return generateAnthropicFiles(baseUrl, apiKey)
      }
      if (activeClientTab.value === 'codex-ws') {
        return generateOpenAIWsFiles(baseUrl, apiKey)
      }
      return generateOpenAIFiles(baseUrl, apiKey)
    case 'gemini':
      return [generateGeminiCliContent(geminiBase, apiKey)]
    case 'antigravity':
      if (activeClientTab.value === 'gemini') {
        return [generateGeminiCliContent(antigravityGeminiBase, apiKey)]
      }
      return generateAnthropicFiles(`${baseUrl}/antigravity`, apiKey)
    case 'grok':
      return generateGrokFiles(apiBase, apiKey)
    default:
      return generateAnthropicFiles(baseUrl, apiKey)
  }
})

function generateOpenAISdkFile(baseUrl: string, apiKey: string): FileConfig {
  const content = `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${apiKey}",
  baseURL: "${baseUrl}"
});

const response = await client.chat.completions.create({
  model: "${OPENAI_CC_SWITCH_CODEX_MODEL}",
  messages: [{ role: "user", content: "Hello from Wegoo AI" }]
});

console.log(response.choices[0]?.message?.content);`

  return {
    path: 'openai-sdk.ts',
    content,
    hint: t('keys.useKeyModal.sdk.openaiHint')
  }
}

function generateAnthropicSdkFile(baseUrl: string, apiKey: string): FileConfig {
  const content = `import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic({
  apiKey: "${apiKey}",
  baseURL: "${baseUrl}"
});

const message = await client.messages.create({
  model: "${CLAUDE_CC_SWITCH_SONNET_MODEL}",
  max_tokens: 1024,
  messages: [{ role: "user", content: "Hello from Wegoo AI" }]
});

console.log(message.content);`

  return {
    path: 'anthropic-sdk.ts',
    content,
    hint: t('keys.useKeyModal.sdk.anthropicHint')
  }
}

function generateAnthropicFiles(baseUrl: string, apiKey: string): FileConfig[] {
  let path: string
  let content: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = `export ANTHROPIC_BASE_URL="${baseUrl}"
export ANTHROPIC_AUTH_TOKEN="${apiKey}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
export CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      break
    case 'cmd':
      path = 'Command Prompt'
      content = `set ANTHROPIC_BASE_URL=${baseUrl}
set ANTHROPIC_AUTH_TOKEN=${apiKey}
set CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
set CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      break
    case 'powershell':
      path = 'PowerShell'
      content = `$env:ANTHROPIC_BASE_URL="${baseUrl}"
$env:ANTHROPIC_AUTH_TOKEN="${apiKey}"
$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
$env:CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      break
    default:
      path = 'Terminal'
      content = ''
  }

  const vscodeSettingsPath = activeTab.value === 'unix'
    ? '~/.claude/settings.json'
    : '%userprofile%\\.claude\\settings.json'

  const vscodeContent = `{
  "env": {
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "ANTHROPIC_AUTH_TOKEN": "${apiKey}",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0"
  }
}`

  return [
    { path, content },
    { path: vscodeSettingsPath, content: vscodeContent, hint: 'VSCode Claude Code' }
  ]
}

function generateGeminiCliContent(baseUrl: string, apiKey: string): FileConfig {
  const model = GEMINI_CC_SWITCH_MODEL
  const modelComment = t('keys.useKeyModal.gemini.modelComment')
  let path: string
  let content: string
  let highlighted: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = `export GOOGLE_GEMINI_BASE_URL="${baseUrl}"
export GEMINI_API_KEY="${apiKey}"
export GEMINI_MODEL="${model}"  # ${modelComment}`
      highlighted = `${keyword('export')} ${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(`"${baseUrl}"`)}
${keyword('export')} ${variable('GEMINI_API_KEY')}${operator('=')}${string(`"${apiKey}"`)}
${keyword('export')} ${variable('GEMINI_MODEL')}${operator('=')}${string(`"${model}"`)}  ${comment(`# ${modelComment}`)}`
      break
    case 'cmd':
      path = 'Command Prompt'
      content = `set GOOGLE_GEMINI_BASE_URL=${baseUrl}
set GEMINI_API_KEY=${apiKey}
set GEMINI_MODEL=${model}`
      highlighted = `${keyword('set')} ${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(baseUrl)}
${keyword('set')} ${variable('GEMINI_API_KEY')}${operator('=')}${string(apiKey)}
${keyword('set')} ${variable('GEMINI_MODEL')}${operator('=')}${string(model)}
${comment(`REM ${modelComment}`)}`
      break
    case 'powershell':
      path = 'PowerShell'
      content = `$env:GOOGLE_GEMINI_BASE_URL="${baseUrl}"
$env:GEMINI_API_KEY="${apiKey}"
$env:GEMINI_MODEL="${model}"  # ${modelComment}`
      highlighted = `${keyword('$env:')}${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(`"${baseUrl}"`)}
${keyword('$env:')}${variable('GEMINI_API_KEY')}${operator('=')}${string(`"${apiKey}"`)}
${keyword('$env:')}${variable('GEMINI_MODEL')}${operator('=')}${string(`"${model}"`)}  ${comment(`# ${modelComment}`)}`
      break
    default:
      path = 'Terminal'
      content = ''
      highlighted = ''
  }

  return { path, content, highlighted }
}

function generateOpenAIFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'
  const contextWindow = 1050000
  const autoCompactLimit = 880000

  // config.toml content
  const configContent = `model_provider = "OpenAI"
model = "${OPENAI_CC_SWITCH_CODEX_MODEL}"
review_model = "${OPENAI_CC_SWITCH_CODEX_MODEL}"
model_reasoning_effort = "xhigh"
model_context_window = ${contextWindow}
model_auto_compact_token_limit = ${autoCompactLimit}
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${baseUrl}"
wire_api = "responses"
requires_openai_auth = true

[features]
goals = true`

  // auth.json content
  const authContent = `{
  "OPENAI_API_KEY": "${apiKey}"
}`

  return [
    {
      path: `${configDir}/config.toml`,
      content: configContent,
      hint: t('keys.useKeyModal.openai.configTomlHint')
    },
    {
      path: `${configDir}/auth.json`,
      content: authContent
    }
  ]
}

function generateGrokFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.grok' : '~/.grok'
  const content = `[models]
default = "sub2api-grok"
web_search = "sub2api-grok"

[model."sub2api-grok"]
model = "${GROK_CC_SWITCH_MODEL}"
base_url = "${baseUrl}"
name = "Grok 4.5 via Sub2API"
api_key = "${apiKey}"
api_backend = "responses"
context_window = 1000000
supports_backend_search = true`
  return [{ path: `${configDir}/config.toml`, content, hint: t('keys.useKeyModal.grok.configTomlHint') }]
}

function generateOpenAIWsFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'
  const contextWindow = 1050000
  const autoCompactLimit = 880000

  // config.toml content with WebSocket v2
  const configContent = `model_provider = "OpenAI"
model = "${OPENAI_CC_SWITCH_CODEX_MODEL}"
review_model = "${OPENAI_CC_SWITCH_CODEX_MODEL}"
model_reasoning_effort = "xhigh"
model_context_window = ${contextWindow}
model_auto_compact_token_limit = ${autoCompactLimit}
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${baseUrl}"
wire_api = "responses"
supports_websockets = true
requires_openai_auth = true

[features]
responses_websockets_v2 = true
goals = true`

  // auth.json content
  const authContent = `{
  "OPENAI_API_KEY": "${apiKey}"
}`

  return [
    {
      path: `${configDir}/config.toml`,
      content: configContent,
      hint: t('keys.useKeyModal.openai.configTomlHint')
    },
    {
      path: `${configDir}/auth.json`,
      content: authContent
    }
  ]
}

function generateOpenCodeConfig(platform: string, baseUrl: string, apiKey: string, pathLabel?: string): FileConfig {
  const provider: Record<string, any> = {
    [platform]: {
      options: {
        baseURL: baseUrl,
        apiKey
      }
    }
  }
  const openaiModels = {
    'gpt-5.6': {
      name: 'GPT-5.6 (Sol)',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-sol': {
      name: 'GPT-5.6 Sol',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-terra': {
      name: 'GPT-5.6 Terra',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-luna': {
      name: 'GPT-5.6 Luna',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.5': {
      name: 'GPT-5.5',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.4': {
      name: 'GPT-5.4',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.4-mini': {
      name: 'GPT-5.4 Mini',
      limit: {
        context: 400000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.3-codex-spark': {
      name: 'GPT-5.3 Codex Spark',
      limit: {
        context: 128000,
        output: 32000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'codex-mini-latest': {
      name: 'Codex Mini',
      limit: {
        context: 200000,
        output: 100000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {}
      }
    }
  }
  const geminiModels = {
    'gemini-2.0-flash': {
      name: 'Gemini 2.0 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-2.5-flash': {
      name: 'Gemini 2.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-2.5-pro': {
      name: 'Gemini 2.5 Pro',
      limit: {
        context: 2097152,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.5-flash': {
      name: 'Gemini 3.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-3-flash-preview': {
      name: 'Gemini 3 Flash Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-3-pro-preview': {
      name: 'Gemini 3 Pro Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-preview': {
      name: 'Gemini 3.1 Pro Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }

  const antigravityGeminiModels = {
    'gemini-2.5-flash': {
      name: 'Gemini 2.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'disable'
        }
      }
    },
    'gemini-2.5-flash-lite': {
      name: 'Gemini 2.5 Flash Lite',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-2.5-flash-thinking': {
      name: 'Gemini 2.5 Flash (Thinking)',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3-flash': {
      name: 'Gemini 3 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-low': {
      name: 'Gemini 3.1 Pro Low',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-high': {
      name: 'Gemini 3.1 Pro High',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-2.5-flash-image': {
      name: 'Gemini 2.5 Flash Image',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image'],
        output: ['image']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-flash-image': {
      name: 'Gemini 3.1 Flash Image',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image'],
        output: ['image']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }
  const claudeModels = {
    'claude-fable-5': {
      name: 'Claude Fable 5',
      limit: {
        context: 1048576,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          type: 'adaptive'
        }
      }
    },
    [CLAUDE_CC_SWITCH_SONNET_MODEL]: {
      name: 'Claude Sonnet 5',
      limit: {
        context: 1048576,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          type: 'adaptive'
        }
      }
    },
    [CLAUDE_CC_SWITCH_OPUS_MODEL]: {
      name: 'Claude Opus 5',
      limit: {
        context: 1048576,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          type: 'adaptive'
        }
      }
    },
    'claude-opus-4-6-thinking': {
      name: 'Claude 4.6 Opus (Thinking)',
      limit: {
        context: 200000,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'claude-sonnet-4-6': {
      name: 'Claude 4.6 Sonnet',
      limit: {
        context: 200000,
        output: 64000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }
  const grokModels = {
    'grok-4.5': { name: 'Grok 4.5', limit: { context: 1000000, output: 128000 } },
    'grok-4.3': { name: 'Grok 4.3', limit: { context: 1000000, output: 128000 } },
    'grok-build-0.1': { name: 'Grok Build 0.1', limit: { context: 256000, output: 128000 } },
    'grok-composer-2.5-fast': { name: 'Grok Composer 2.5 Fast', limit: { context: 500000, output: 128000 } }
  }

  if (platform === 'gemini') {
    provider[platform].npm = '@ai-sdk/google'
    provider[platform].models = geminiModels
  } else if (platform === 'anthropic') {
    provider[platform].npm = '@ai-sdk/anthropic'
    provider[platform].name = 'Tomato Claude'
    provider[platform].models = claudeModels
  } else if (platform === 'antigravity-claude') {
    provider[platform].npm = '@ai-sdk/anthropic'
    provider[platform].name = 'Antigravity (Claude)'
    provider[platform].models = claudeModels
  } else if (platform === 'antigravity-gemini') {
    provider[platform].npm = '@ai-sdk/google'
    provider[platform].name = 'Antigravity (Gemini)'
    provider[platform].models = antigravityGeminiModels
  } else if (platform === 'openai') {
    provider[platform].npm = '@ai-sdk/openai'
    provider[platform].name = 'Tomato Codex / GPT'
    provider[platform].models = openaiModels
  } else if (platform === 'grok') {
    provider[platform].npm = '@ai-sdk/openai'
    provider[platform].name = 'Grok via Sub2API'
    provider[platform].models = grokModels
  }

  const agent =
    platform === 'openai'
      ? {
          build: {
            model: `openai/${OPENAI_CC_SWITCH_CODEX_MODEL}`,
            variant: 'xhigh',
            options: {
              store: false
            }
          },
          plan: {
            model: `openai/${OPENAI_CC_SWITCH_CODEX_MODEL}`,
            variant: 'xhigh',
            options: {
              store: false
            }
          }
        }
      : undefined

  const content = JSON.stringify(
    {
      $schema: 'https://opencode.ai/config.json',
      provider,
      ...(agent ? { agent } : {})
    },
    null,
    2
  )

  return {
    path: pathLabel ?? 'opencode.json',
    content,
    hint: t('keys.useKeyModal.opencode.hint')
  }
}

const copyContent = async (content: string, index: number) => {
  const success = await clipboardCopy(content, t('keys.copied'))
  if (success) {
    copiedIndex.value = index
    setTimeout(() => {
      copiedIndex.value = null
    }, 2000)
  }
}
</script>
