<template>
  <div class="public-gateway-shell">
    <PublicGatewayHeader />

    <main class="public-gateway-container pb-16">
      <section class="public-gateway-hero">
        <p class="public-gateway-kicker">{{ copy.kicker }}</p>
        <h1 class="public-gateway-title">{{ copy.title }}</h1>
        <p class="public-gateway-lead">{{ copy.lead }}</p>
        <div class="mt-6 flex flex-wrap gap-3">
          <router-link to="/register" class="public-gateway-primary">{{ copy.start }}</router-link>
          <router-link to="/pricing" class="public-gateway-secondary">{{ copy.models }}</router-link>
        </div>
      </section>

      <section class="grid min-w-0 gap-4 lg:grid-cols-[minmax(0,0.85fr)_minmax(320px,0.45fr)]">
        <div class="min-w-0 space-y-4">
          <article
            v-for="section in sections"
            :key="section.title"
            class="public-gateway-card p-5"
          >
            <p class="text-xs font-semibold uppercase text-[var(--gw-accent)]">{{ section.kicker }}</p>
            <h2 class="mt-2 text-xl font-semibold text-[var(--gw-text)]">{{ section.title }}</h2>
            <p class="mt-2 text-sm leading-6 text-[var(--gw-text-2)]">{{ section.description }}</p>
            <div class="mt-4 flex flex-wrap gap-2">
              <span v-for="item in section.items" :key="item" class="public-gateway-chip">
                {{ item }}
              </span>
            </div>
          </article>

          <article class="public-gateway-panel min-w-0 p-5">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <p class="gateway-kicker">{{ copy.examplesKicker }}</p>
                <h2 class="mt-2 text-xl font-semibold text-[var(--gw-text)]">{{ copy.examplesTitle }}</h2>
                <p class="mt-2 text-sm leading-6 text-[var(--gw-text-2)]">{{ copy.examplesLead }}</p>
              </div>
              <router-link to="/key-usage" class="public-gateway-secondary rounded-lg">
                {{ copy.keyUsage }}
              </router-link>
            </div>

            <div class="mt-5 flex flex-wrap gap-2">
              <button
                v-for="sample in codeSamples"
                :key="sample.id"
                type="button"
                class="public-gateway-chip"
                :class="{ 'border-[color:var(--gw-accent)] text-[var(--gw-accent)]': activeSampleId === sample.id }"
                @click="activeSampleId = sample.id"
              >
                {{ sample.label }}
              </button>
            </div>

            <div class="mt-4 min-w-0">
              <div class="mb-3 flex items-center justify-between border-b border-[color:var(--gw-border)] pb-3">
                <span class="min-w-0 truncate text-xs font-semibold text-[var(--gw-text-3)]">{{ activeSample.filename }}</span>
                <button class="inline-flex items-center gap-1 text-xs font-semibold text-[var(--gw-accent)]" type="button" @click="copyCode(activeSample.code)">
                  <Icon :name="copied ? 'check' : 'copy'" size="xs" />
                  {{ copied ? copy.copied : copy.copy }}
                </button>
              </div>
              <pre class="public-gateway-code"><code>{{ activeSample.code }}</code></pre>
            </div>
          </article>

          <article class="public-gateway-panel p-5">
            <p class="gateway-kicker">{{ copy.errorsKicker }}</p>
            <h2 class="mt-2 text-xl font-semibold text-[var(--gw-text)]">{{ copy.errorsTitle }}</h2>
            <p class="mt-2 text-sm leading-6 text-[var(--gw-text-2)]">{{ copy.errorsLead }}</p>
            <div class="mt-5 grid gap-3 md:grid-cols-2">
              <div v-for="item in errorGuides" :key="item.title" class="public-gateway-stat">
                <div class="flex items-start gap-3">
                  <span class="mt-0.5 inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-[color:var(--gw-border)] text-[var(--gw-accent)]">
                    <Icon :name="item.icon" size="sm" />
                  </span>
                  <div class="min-w-0">
                    <h3 class="text-sm font-semibold text-[var(--gw-text)]">{{ item.title }}</h3>
                    <p class="mt-1 whitespace-normal text-sm leading-6 text-[var(--gw-text-2)]">{{ item.description }}</p>
                  </div>
                </div>
              </div>
            </div>

            <div class="mt-6 min-w-0">
              <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
                <div>
                  <h3 class="text-base font-semibold text-[var(--gw-text)]">{{ copy.troubleshootingTitle }}</h3>
                  <p class="mt-1 text-sm leading-6 text-[var(--gw-text-2)]">{{ copy.troubleshootingLead }}</p>
                </div>
                <router-link to="/usage?tab=errors" class="public-gateway-secondary rounded-lg">
                  {{ copy.issueRecords }}
                </router-link>
              </div>

              <div class="mt-4 grid min-w-0 gap-2">
                <div
                  v-for="row in troubleshootingRows"
                  :key="row.id"
                  class="public-gateway-stat min-w-0 p-4"
                >
                  <div class="grid min-w-0 gap-3 lg:grid-cols-[minmax(150px,0.52fr)_minmax(0,1fr)_minmax(0,1fr)] lg:items-start">
                    <div class="min-w-0">
                      <p class="text-xs font-semibold uppercase text-[var(--gw-accent)]">{{ row.status }}</p>
                      <p class="mt-1 truncate text-sm font-semibold text-[var(--gw-text)]">{{ row.category }}</p>
                    </div>
                    <div class="min-w-0">
                      <p class="text-xs font-semibold uppercase text-[var(--gw-text-3)]">{{ copy.firstAction }}</p>
                      <p class="mt-1 text-sm leading-6 text-[var(--gw-text-2)]">{{ row.action }}</p>
                    </div>
                    <div class="min-w-0">
                      <p class="text-xs font-semibold uppercase text-[var(--gw-text-3)]">{{ copy.ticketEvidence }}</p>
                      <p class="mt-1 text-sm leading-6 text-[var(--gw-text-2)]">{{ row.evidence }}</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </article>
        </div>

        <aside class="public-gateway-panel min-w-0 p-4 lg:sticky lg:top-24 lg:self-start">
          <div class="mb-3 flex items-center justify-between border-b border-[color:var(--gw-border)] pb-3">
            <span class="text-xs font-semibold text-[var(--gw-text-3)]">{{ quickstartSample.filename }}</span>
            <button class="inline-flex items-center gap-1 text-xs font-semibold text-[var(--gw-accent)]" type="button" @click="copyCode(quickstartSample.code)">
              <Icon :name="copied ? 'check' : 'copy'" size="xs" />
              {{ copied ? copy.copied : copy.copy }}
            </button>
          </div>
          <pre class="public-gateway-code"><code>{{ quickstartSample.code }}</code></pre>
          <div class="mt-4 grid gap-2">
            <router-link to="/keys" class="public-gateway-secondary rounded-lg">{{ copy.createKey }}</router-link>
            <router-link to="/key-usage" class="public-gateway-secondary rounded-lg">{{ copy.keyUsage }}</router-link>
            <router-link to="/status" class="public-gateway-secondary rounded-lg">{{ copy.status }}</router-link>
          </div>
        </aside>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import Icon from '@/components/icons/Icon.vue'
import PublicGatewayHeader from './PublicGatewayHeader.vue'
import { resolvePublicApiBaseUrls } from '@/custom/utils/publicApiBaseUrl'
import './publicGateway.css'

const { locale } = useI18n()
const appStore = useAppStore()
const { copied, copyToClipboard } = useClipboard()
const activeSampleId = ref('openai')

const baseUrls = computed(() => resolvePublicApiBaseUrls(appStore.apiBaseUrl))
const quickstartSample = computed(() => ({
  filename: 'quickstart.sh',
  code: `export WEGOO_API_KEY=sk-...

curl ${baseUrls.value.v1}/chat/completions \\
  -H "Authorization: Bearer $WEGOO_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.4",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'`
}))

const codeSamples = computed(() => [
  {
    id: 'openai',
    label: 'OpenAI SDK',
    filename: 'openai.ts',
    code: `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.WEGOO_API_KEY,
  baseURL: "${baseUrls.value.v1}"
});

const stream = await client.chat.completions.create({
  model: "gpt-5.4",
  messages: [{ role: "user", content: "Hello" }],
  stream: true
});`
  },
  {
    id: 'anthropic',
    label: 'Anthropic SDK',
    filename: 'anthropic.ts',
    code: `import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic({
  apiKey: process.env.WEGOO_API_KEY,
  baseURL: "${baseUrls.value.root}"
});

const message = await client.messages.create({
  model: "claude-sonnet-4",
  max_tokens: 1024,
  messages: [{ role: "user", content: "Hello" }]
});`
  },
  {
    id: 'codex',
    label: 'Codex CLI',
    filename: 'codex.env',
    code: `export OPENAI_API_KEY="$WEGOO_API_KEY"
export OPENAI_BASE_URL="${baseUrls.value.v1}"

codex --model gpt-5.4`
  },
  {
    id: 'curl',
    label: 'curl',
    filename: quickstartSample.value.filename,
    code: quickstartSample.value.code
  }
])

const activeSample = computed(() => codeSamples.value.find((sample) => sample.id === activeSampleId.value) ?? codeSamples.value[0])

const copy = computed(() => locale.value.startsWith('zh')
  ? {
    kicker: 'Developer Docs · Quick Start',
    title: '几分钟内接入 API 网关',
    lead: '这里提供公开快速开始入口。登录控制台后可以创建 Key、查看模型价格、复制 SDK 示例并核对每次请求的扣费记录。',
    start: '创建账户',
    models: '查看模型价格',
    copy: '复制',
    copied: '已复制',
    createKey: '进入 Key 管理',
    keyUsage: '查询 Key 用量',
    status: '查看服务状态',
    examplesKicker: 'SDK Examples',
    examplesTitle: '复制后只需要替换 Key',
    examplesLead: '公开文档页直接给出 OpenAI、Anthropic、Codex CLI 和 curl 示例。实际可用模型、分组和价格以控制台展示为准。',
    errorsKicker: 'Error Handling',
    errorsTitle: '错误先按可操作原因理解',
    errorsLead: '常见失败会归类到余额、分组、模型和服务状态，方便快速定位处理方式。',
    troubleshootingTitle: '提交工单前先按分类处理',
    troubleshootingLead: '这些分类和用量页的问题记录一致；如果仍失败，再从问题记录创建工单，系统会带上请求上下文。',
    issueRecords: '查看问题记录',
    firstAction: '先处理',
    ticketEvidence: '工单补充',
  }
  : {
    kicker: 'Developer Docs · Quick Start',
    title: 'Connect to the gateway in minutes',
    lead: 'This public quick start points developers to keys, model pricing, SDK examples, and usage records in the console.',
    start: 'Create Account',
    models: 'View Models',
    copy: 'Copy',
    copied: 'Copied',
    createKey: 'Open Key Management',
    keyUsage: 'Check Key Usage',
    status: 'View Service Status',
    examplesKicker: 'SDK Examples',
    examplesTitle: 'Copy and replace the key',
    examplesLead: 'The public docs include OpenAI, Anthropic, Codex CLI, and curl examples. Available models, groups, and prices are shown in the console.',
    errorsKicker: 'Error Handling',
    errorsTitle: 'Read errors as actionable causes',
    errorsLead: 'Common failures are grouped by balance, group availability, model support, and service status for quicker action.',
    troubleshootingTitle: 'Handle by category before opening a ticket',
    troubleshootingLead: 'These categories match the records on the usage page. If the request still fails, create a ticket from the matching record so request context is attached.',
    issueRecords: 'View records',
    firstAction: 'First action',
    ticketEvidence: 'Ticket evidence',
  })

const sections = computed(() => locale.value.startsWith('zh')
  ? [
    {
      kicker: 'Step 01',
      title: '创建 API Key',
      description: '注册或登录控制台，在 API 密钥页创建调用凭证。Key 创建成功后完整密钥只展示一次。',
      items: ['额度限制', '分组路由', 'IP 白名单', '过期时间'],
    },
    {
      kicker: 'Step 02',
      title: '替换 Base URL',
      description: '保持 OpenAI SDK、Anthropic SDK、Codex CLI 或 Claude Code 的调用方式，只替换网关地址和 Authorization。',
      items: ['OpenAI SDK', 'Anthropic SDK', 'Codex CLI', 'Claude Code'],
    },
    {
      kicker: 'Step 03',
      title: '核对用量和错误',
      description: '每次请求会记录模型、token、缓存、费用、延迟和错误分类，便于核对扣费和排查问题。',
      items: ['Token 明细', 'Cache Read/Write', 'TTFT', '错误分类'],
    },
  ]
  : [
    {
      kicker: 'Step 01',
      title: 'Create an API key',
      description: 'Sign in to the console and create a gateway credential. The full key is shown once after creation.',
      items: ['Limits', 'Routing', 'IP allowlist', 'Expiration'],
    },
    {
      kicker: 'Step 02',
      title: 'Replace the Base URL',
      description: 'Keep OpenAI SDK, Anthropic SDK, Codex CLI, or Claude Code usage patterns and change the endpoint plus authorization.',
      items: ['OpenAI SDK', 'Anthropic SDK', 'Codex CLI', 'Claude Code'],
    },
    {
      kicker: 'Step 03',
      title: 'Audit usage and errors',
      description: 'Requests expose model, tokens, cache, cost, latency, and classified errors for billing and troubleshooting.',
      items: ['Token detail', 'Cache Read/Write', 'TTFT', 'Error classes'],
    },
  ])

const errorGuides = computed(() => locale.value.startsWith('zh')
  ? [
    {
      icon: 'creditCard' as const,
      title: '余额不足',
      description: '充值或兑换卡密后重试。余额、订单和发票入口在控制台保持同一账务口径。'
    },
    {
      icon: 'server' as const,
      title: '当前分组不可用',
      description: '切换分组或稍后重试。状态页会展示相关模型服务的可用性和近期表现。'
    },
    {
      icon: 'ban' as const,
      title: '模型不支持',
      description: '到模型价格页确认该分组支持的模型、端点和缓存计费方式。'
    },
    {
      icon: 'refresh' as const,
      title: '服务繁忙',
      description: '平台会按服务策略自动重试；如果仍未恢复，会返回可读的错误提示。'
    }
  ]
  : [
    {
      icon: 'creditCard' as const,
      title: 'Insufficient balance',
      description: 'Top up or redeem a card code, then retry. Wallet, orders, and invoices use the same billing records.'
    },
    {
      icon: 'server' as const,
      title: 'Group unavailable',
      description: 'Switch groups or retry later. The status page shows model-service availability and recent performance.'
    },
    {
      icon: 'ban' as const,
      title: 'Model unsupported',
      description: 'Use the model catalog to verify group support, endpoint type, and cache pricing.'
    },
    {
      icon: 'refresh' as const,
      title: 'Service busy',
      description: 'The platform retries according to service policy and returns readable errors if the issue persists.'
    }
  ])

const troubleshootingRows = computed(() => locale.value.startsWith('zh')
  ? [
    {
      id: 'auth',
      status: '401 / 403',
      category: 'auth',
      action: '确认 API Key、分组权限和客户端 Base URL 是否匹配；凭证重置后要换新 Key。',
      evidence: '提供请求时间、Key 名称、分组和客户端配置截图，不要上传完整密钥。'
    },
    {
      id: 'rate-limit',
      status: '429',
      category: 'rate_limit',
      action: '降低并发或稍后重试；容量紧张时切换到更空闲的分组。',
      evidence: '提供请求时间、模型、并发量、客户端提示和可复现的分组。'
    },
    {
      id: 'quota',
      status: '402 / quota',
      category: 'quota',
      action: '检查余额、卡密兑换记录、分组额度或订阅状态，确认充值已到账。',
      evidence: '提供订单号、兑换码批次或充值时间；余额到账异常可直接关联订单。'
    },
    {
      id: 'invalid-request',
      status: '400 / 422',
      category: 'invalid_request',
      action: '检查模型名、endpoint、请求参数和客户端版本；模型能力以模型页展示为准。',
      evidence: '提供脱敏后的请求片段、模型名、endpoint 和同请求是否在其他分组可用。'
    },
    {
      id: 'unavailable',
      status: '5xx / overload',
      category: 'service_unavailable',
      action: '这是服务暂不可用或负载较高；稍后重试或切换分组，平台会按策略自动重试。',
      evidence: '提供请求时间、模型、分组和客户端最终看到的错误文案。'
    },
    {
      id: 'safety',
      status: 'policy',
      category: 'cyber',
      action: '检查提示词、附件和调用用途是否触发安全策略，必要时调整业务输入。',
      evidence: '提供脱敏后的提示词片段、附件类型和业务场景说明。'
    },
  ]
  : [
    {
      id: 'auth',
      status: '401 / 403',
      category: 'auth',
      action: 'Confirm the API key, group permission, and client Base URL. Use a new key after credential reset.',
      evidence: 'Include request time, key name, group, and client configuration screenshot. Do not upload full secret keys.'
    },
    {
      id: 'rate-limit',
      status: '429',
      category: 'rate_limit',
      action: 'Reduce concurrency or retry later. Switch to a less saturated group when capacity is tight.',
      evidence: 'Include request time, model, concurrency, client message, and the reproducible group.'
    },
    {
      id: 'quota',
      status: '402 / quota',
      category: 'quota',
      action: 'Check balance, card-code redemption, group quota, or subscription status, and confirm the top-up landed.',
      evidence: 'Include order number, card batch, or recharge time. Link the order directly for crediting issues.'
    },
    {
      id: 'invalid-request',
      status: '400 / 422',
      category: 'invalid_request',
      action: 'Check model name, endpoint, request parameters, and client version. Model capability comes from the model page.',
      evidence: 'Include a redacted request snippet, model, endpoint, and whether the same request works in another group.'
    },
    {
      id: 'unavailable',
      status: '5xx / overload',
      category: 'service_unavailable',
      action: 'The service is temporarily unavailable or overloaded. Retry later or switch groups; the platform retries by policy.',
      evidence: 'Include request time, model, group, and the final client-visible error message.'
    },
    {
      id: 'safety',
      status: 'policy',
      category: 'cyber',
      action: 'Review whether the prompt, attachments, or use case triggered safety policy, then adjust the input.',
      evidence: 'Include a redacted prompt snippet, attachment type, and business scenario.'
    },
  ])

function copyCode(code: string) {
  copyToClipboard(code, copy.value.copied)
}
</script>
