<template>
  <div class="public-gateway-shell">
    <PublicGatewayHeader />

    <main class="public-gateway-container pb-16">
      <section class="public-gateway-hero">
        <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.42fr)] lg:items-end">
          <div class="min-w-0">
            <p class="public-gateway-kicker">{{ copy.kicker }}</p>
            <h1 class="public-gateway-title">{{ copy.title }}</h1>
            <p class="public-gateway-lead">{{ copy.lead }}</p>
            <div class="mt-6 flex flex-wrap gap-3">
              <router-link to="/tickets" class="public-gateway-primary">{{ copy.contact }}</router-link>
              <router-link to="/payment" class="public-gateway-secondary">{{ copy.wallet }}</router-link>
              <router-link to="/invoices" class="public-gateway-secondary">{{ copy.invoices }}</router-link>
            </div>
          </div>

          <aside class="public-gateway-panel p-4">
            <p class="text-xs font-semibold uppercase text-[var(--gw-gold)]">{{ copy.handoffKicker }}</p>
            <div class="mt-4 grid gap-3">
              <div v-for="item in handoffSteps" :key="item.title" class="flex gap-3">
                <span class="mt-0.5 inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-[color:var(--gw-border)] bg-[var(--gw-bg-soft)] text-[var(--gw-accent)]">
                  <Icon :name="item.icon" size="sm" />
                </span>
                <div class="min-w-0">
                  <h2 class="text-sm font-semibold text-[var(--gw-text)]">{{ item.title }}</h2>
                  <p class="mt-1 text-xs leading-5 text-[var(--gw-text-3)]">{{ item.description }}</p>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <article v-for="item in cards" :key="item.title" class="public-gateway-card p-5">
          <Icon :name="item.icon" size="lg" class="text-[var(--gw-accent)]" />
          <h2 class="mt-4 text-lg font-semibold text-[var(--gw-text)]">{{ item.title }}</h2>
          <p class="mt-2 text-sm leading-6 text-[var(--gw-text-2)]">{{ item.description }}</p>
        </article>
      </section>

      <section class="mt-5 grid gap-5 lg:grid-cols-[minmax(0,0.72fr)_minmax(320px,0.42fr)]">
        <article class="public-gateway-panel p-5">
          <div>
            <p class="text-xs font-semibold uppercase text-[var(--gw-gold)]">{{ copy.invoiceKicker }}</p>
            <h2 class="mt-2 text-2xl font-semibold text-[var(--gw-text)]">{{ copy.invoiceTitle }}</h2>
            <p class="mt-2 text-sm leading-6 text-[var(--gw-text-2)]">{{ copy.invoiceLead }}</p>
          </div>
          <div class="mt-5 overflow-hidden rounded-lg border border-[color:var(--gw-border)]">
            <div
              v-for="item in invoiceRules"
              :key="item.source"
              class="grid gap-2 border-b border-[color:var(--gw-border-soft)] p-3 last:border-b-0 sm:grid-cols-[minmax(0,0.35fr)_auto_minmax(0,1fr)] sm:items-center"
            >
              <strong class="min-w-0 text-sm text-[var(--gw-text)]">{{ item.source }}</strong>
              <span
                :class="[
                  'inline-flex w-fit items-center rounded-full px-2.5 py-1 text-xs font-semibold',
                  item.invoiceable ? 'bg-[var(--gw-accent-soft)] text-[var(--gw-accent)]' : 'bg-[var(--gw-gold-soft)] text-[var(--gw-gold)]'
                ]"
              >
                {{ item.invoiceable ? copy.invoiceable : copy.notInvoiceable }}
              </span>
              <p class="min-w-0 text-sm leading-6 text-[var(--gw-text-2)]">{{ item.note }}</p>
            </div>
          </div>
        </article>

        <aside class="public-gateway-panel p-5">
          <p class="text-xs font-semibold uppercase text-[var(--gw-accent)]">{{ copy.supportKicker }}</p>
          <h2 class="mt-2 text-2xl font-semibold text-[var(--gw-text)]">{{ copy.supportTitle }}</h2>
          <p class="mt-2 text-sm leading-6 text-[var(--gw-text-2)]">{{ copy.supportLead }}</p>
          <div class="mt-5 grid gap-2">
            <router-link to="/tickets" class="public-gateway-secondary rounded-lg">{{ copy.openTicket }}</router-link>
            <router-link to="/orders" class="public-gateway-secondary rounded-lg">{{ copy.orders }}</router-link>
            <router-link to="/key-usage" class="public-gateway-secondary rounded-lg">{{ copy.keyUsage }}</router-link>
          </div>
        </aside>
      </section>

      <section class="public-gateway-panel mt-5 p-5">
        <div class="grid gap-5 lg:grid-cols-[minmax(0,0.38fr)_minmax(0,1fr)] lg:items-start">
          <div>
            <p class="text-xs font-semibold uppercase text-[var(--gw-accent)]">{{ copy.workflowKicker }}</p>
            <h2 class="mt-2 text-2xl font-semibold text-[var(--gw-text)]">{{ copy.workflowTitle }}</h2>
            <p class="mt-2 text-sm leading-6 text-[var(--gw-text-2)]">{{ copy.workflowLead }}</p>
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <div v-for="item in workflow" :key="item.title" class="public-gateway-stat">
              <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-[color:var(--gw-border)] text-[var(--gw-gold)]">
                <Icon :name="item.icon" size="sm" />
              </span>
              <h3 class="mt-3 text-sm font-semibold text-[var(--gw-text)]">{{ item.title }}</h3>
              <p class="mt-1 whitespace-normal text-sm leading-6 text-[var(--gw-text-2)]">{{ item.description }}</p>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PublicGatewayHeader from './PublicGatewayHeader.vue'
import './publicGateway.css'

type IconName = InstanceType<typeof Icon>['$props']['name']

const { locale } = useI18n()

const copy = computed(() => locale.value.startsWith('zh')
  ? {
    kicker: 'Enterprise · Billing Support',
    title: '大额充值、发票和专属支持',
    lead: '企业入口用于大额充值、对账、发票和专属支持。相关余额、退款和可开票范围会以订单和账单记录为准。',
    contact: '提交工单',
    wallet: '进入钱包',
    invoices: '发票中心',
    handoffKicker: 'Support Handoff',
    invoiceKicker: 'Invoice Policy',
    invoiceTitle: '开票基于可核对的付费记录',
    invoiceLead: '用户充值、卡密充值和人工收款可进入开票口径；赠送补偿、邀请返佣和赠送扣减不参与开票。',
    invoiceable: '可开票',
    notInvoiceable: '不可开票',
    supportKicker: 'Operations',
    supportTitle: '把订单、发票和请求上下文带进工单',
    supportLead: '企业问题通常需要财务和技术同时排查，入口会引导用户带上订单号、发票申请、API Key 用量或请求 ID。',
    openTicket: '打开工单',
    orders: '查看订单',
    keyUsage: '查询 Key 用量',
    workflowKicker: 'Workflow',
    workflowTitle: '企业流程统一留痕',
    workflowLead: '人工确认、入账、发票审核和售后处理都会留有记录，方便后续核对。',
  }
  : {
    kicker: 'Enterprise · Billing Support',
    title: 'Large top-ups, invoices, and dedicated support',
    lead: 'Enterprise routes handle large top-ups, reconciliation, invoices, and dedicated support. Balance sources, refunds, and invoice eligibility follow order and billing records.',
    contact: 'Open Ticket',
    wallet: 'Open Wallet',
    invoices: 'Invoice Center',
    handoffKicker: 'Support Handoff',
    invoiceKicker: 'Invoice Policy',
    invoiceTitle: 'Invoices follow verifiable payment records',
    invoiceLead: 'User payments, redeem-code top-ups, and manual collection can be invoiceable. Gifts, referral bonuses, and gift deductions are not invoiceable.',
    invoiceable: 'Invoiceable',
    notInvoiceable: 'Not invoiceable',
    supportKicker: 'Operations',
    supportTitle: 'Bring orders, invoices, and request context into tickets',
    supportLead: 'Enterprise issues often need both billing and technical context. The flow asks users to include order numbers, invoice requests, API key usage, or request IDs.',
    openTicket: 'Open Ticket',
    orders: 'View Orders',
    keyUsage: 'Check Key Usage',
    workflowKicker: 'Workflow',
    workflowTitle: 'Enterprise handling stays traceable',
    workflowLead: 'Manual confirmation, crediting, and invoice review stay recorded for later reconciliation.',
  })

const cards = computed<Array<{ title: string; description: string; icon: IconName }>>(() => locale.value.startsWith('zh')
  ? [
    { title: '人工收款', description: '适合大额充值、对公转账和需要线下确认的付款。', icon: 'creditCard' },
    { title: '发票协作', description: '订单和发票记录可统一查看。', icon: 'document' },
    { title: '专属支持', description: '工单可关联订单、发票、请求 ID 和账户上下文。', icon: 'chat' },
    { title: '透明账单', description: '充值、退款、赠送、返佣和扣减按业务分类展示。', icon: 'dollar' },
  ]
  : [
    { title: 'Manual Collection', description: 'For large payments, wire transfers, and offline confirmation.', icon: 'creditCard' },
    { title: 'Invoice Workflow', description: 'Orders and invoice records stay easy to review.', icon: 'document' },
    { title: 'Dedicated Support', description: 'Tickets can reference orders, invoices, request IDs, and account context.', icon: 'chat' },
    { title: 'Clear Billing Records', description: 'Top-ups, refunds, gifts, rebates, and deductions are classified.', icon: 'dollar' },
  ])

const invoiceRules = computed(() => locale.value.startsWith('zh')
  ? [
    { source: '在线支付充值', invoiceable: true, note: '支付宝、微信、Stripe 等用户自主充值可进入可开票订单范围。' },
    { source: '卡密兑换', invoiceable: true, note: '卡密作为用户付费充值来源，可以进入开票订单范围。' },
    { source: '管理员人工收款', invoiceable: true, note: '线下收款后由管理员按人工收款分类入账，和订单/发票统计保持一致。' },
    { source: '邀请返佣 / 赠送补偿', invoiceable: false, note: '这类余额属于赠送来源，不参与净充值和可开票订单。' },
  ]
  : [
    { source: 'Online top-ups', invoiceable: true, note: 'Alipay, WeChat, Stripe, and other user payments can be included in invoiceable orders.' },
    { source: 'Redeem-code top-ups', invoiceable: true, note: 'Redeem codes represent paid top-ups and can enter the invoice order scope.' },
    { source: 'Manual collection', invoiceable: true, note: 'Offline payments should be credited by admins as manual collection to keep orders and invoice totals aligned.' },
    { source: 'Referral / gift balance', invoiceable: false, note: 'Gift-like balance sources are not part of net recharge or invoiceable orders.' },
  ])

const handoffSteps = computed<Array<{ title: string; description: string; icon: IconName }>>(() => locale.value.startsWith('zh')
  ? [
    { title: '先看钱包', description: '充值能力和余额来源仍以钱包页为入口。', icon: 'creditCard' },
    { title: '再选发票', description: '可开票订单在发票中心勾选和提交。', icon: 'document' },
    { title: '最后交给工单', description: '复杂对账和专属支持通过工单留痕。', icon: 'chat' },
  ]
  : [
    { title: 'Start in wallet', description: 'Top-up capability and balance sources still start in the wallet.', icon: 'creditCard' },
    { title: 'Select invoices', description: 'Invoiceable orders are selected and submitted in the invoice center.', icon: 'document' },
    { title: 'Escalate by ticket', description: 'Complex reconciliation and dedicated support should leave a ticket trail.', icon: 'chat' },
  ])

const workflow = computed<Array<{ title: string; description: string; icon: IconName }>>(() => locale.value.startsWith('zh')
  ? [
    { title: '确认付款', description: '用户提交金额、付款方式、订单或转账备注，避免人工口头确认。', icon: 'clipboard' },
    { title: '分类入账', description: '管理员按用户充值、人工收款、退款或赠送分类处理余额。', icon: 'database' },
    { title: '审核开票', description: '发票只读取可开票来源，退款和不可开票余额会被排除。', icon: 'shield' },
  ]
  : [
    { title: 'Confirm payment', description: 'Users provide amount, method, order number, or transfer memo instead of relying on chat notes.', icon: 'clipboard' },
    { title: 'Record balance changes', description: 'Admins classify balance changes as recharge, manual collection, refund, or gift.', icon: 'database' },
    { title: 'Review invoices', description: 'Invoices use only invoiceable sources; refunds and non-invoiceable balance are excluded.', icon: 'shield' },
  ])
</script>
