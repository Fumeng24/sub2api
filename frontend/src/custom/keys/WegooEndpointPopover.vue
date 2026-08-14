<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { CustomEndpoint } from '@/types'

const props = defineProps<{
  apiBaseUrl: string
  customEndpoints: CustomEndpoint[]
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const copiedEndpoint = ref<string | null>(null)

let copiedResetTimer: number | undefined

const allEndpoints = computed(() => {
  const items: Array<{ name: string; endpoint: string; description: string; isDefault: boolean }> = []
  if (props.apiBaseUrl) {
    items.push({
      name: t('keys.endpoints.title'),
      endpoint: props.apiBaseUrl,
      description: '',
      isDefault: true,
    })
  }
  for (const ep of props.customEndpoints) {
    items.push({ ...ep, isDefault: false })
  }
  return items
})

async function copy(url: string) {
  const success = await copyToClipboard(url, t('keys.endpoints.copied'))
  if (!success) return

  copiedEndpoint.value = url
  if (copiedResetTimer !== undefined) {
    window.clearTimeout(copiedResetTimer)
  }
  copiedResetTimer = window.setTimeout(() => {
    if (copiedEndpoint.value === url) {
      copiedEndpoint.value = null
    }
  }, 1800)
}

function tooltipHint(endpoint: string): string {
  return copiedEndpoint.value === endpoint
    ? t('keys.endpoints.copiedHint')
    : t('keys.endpoints.clickToCopy')
}

function speedTestUrl(endpoint: string): string {
  return `https://www.tcptest.cn/http/${encodeURIComponent(endpoint)}`
}

onBeforeUnmount(() => {
  if (copiedResetTimer !== undefined) {
    window.clearTimeout(copiedResetTimer)
  }
})
</script>

<template>
  <div v-if="allEndpoints.length > 0" class="flex min-w-0 flex-wrap gap-2">
    <div
      v-for="(item, index) in allEndpoints"
      :key="index"
      class="group/endpoint relative flex w-full max-w-full min-w-0 items-start gap-2 rounded-lg border border-[color:var(--apple-border-soft)] bg-[var(--apple-surface)] px-3 py-2.5 text-xs shadow-sm transition-colors hover:border-[color:var(--apple-border)] hover:bg-[var(--apple-surface-elevated)] sm:w-auto sm:items-center sm:px-2.5 sm:py-1.5"
    >
      <div
        class="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 w-max max-w-[min(24rem,calc(100vw-1rem))] -translate-x-1/2 translate-y-1 rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] px-3 py-2.5 text-left opacity-0 shadow-[0_14px_36px_-20px_rgba(15,23,42,0.35)] ring-1 ring-[color:var(--apple-border-soft)] transition-all duration-150 group-hover/endpoint:translate-y-0 group-hover/endpoint:opacity-100 group-focus-within/endpoint:translate-y-0 group-focus-within/endpoint:opacity-100"
      >
        <p
          v-if="item.description"
          class="max-w-[24rem] break-words text-xs leading-5 text-[var(--apple-muted)]"
        >
          {{ item.description }}
        </p>
        <p class="mt-1.5 max-w-full break-all font-mono text-[11px] leading-4 text-[var(--apple-muted-2)]">
          {{ item.endpoint }}
        </p>
        <p
          class="flex items-center gap-1.5 text-[11px] leading-4 text-[var(--apple-blue)]"
          :class="item.description ? 'mt-1.5' : ''"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-[var(--apple-blue)]"></span>
          {{ tooltipHint(item.endpoint) }}
        </p>
        <div class="absolute left-1/2 top-full h-3 w-3 -translate-x-1/2 -translate-y-1/2 rotate-45 border-b border-r border-[color:var(--apple-border)] bg-[var(--apple-surface)]"></div>
      </div>

      <div class="mt-0.5 flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-md bg-[var(--apple-surface-elevated)] text-[var(--apple-muted-2)] ring-1 ring-[color:var(--apple-border-soft)] sm:mt-0">
        <Icon name="server" size="xs" />
      </div>

      <div class="flex min-w-0 flex-1 flex-col gap-1 sm:flex-row sm:items-center sm:gap-1.5">
        <div class="flex min-w-0 items-center gap-1.5">
          <span class="min-w-0 max-w-[8rem] truncate font-medium text-[var(--apple-text)]">{{ item.name }}</span>
          <span
            v-if="item.isDefault"
            class="flex-shrink-0 rounded-md bg-[var(--apple-surface-elevated)] px-1.5 py-px text-[10px] font-medium leading-tight text-[var(--apple-muted)] ring-1 ring-[color:var(--apple-border-soft)]"
          >{{ t('keys.endpoints.default') }}</span>
        </div>

        <span class="hidden h-3 w-px flex-shrink-0 bg-[var(--apple-border-soft)] sm:inline"></span>

        <code
          class="block min-w-0 max-w-full cursor-pointer break-all font-mono text-[11px] leading-4 text-[var(--apple-muted)] decoration-[var(--apple-muted-2)] decoration-dashed underline-offset-2 hover:text-[var(--apple-blue)] hover:underline focus:text-[var(--apple-blue)] focus:underline focus:outline-none sm:max-w-[28rem] sm:text-xs"
          data-testid="endpoint-url"
          role="button"
          tabindex="0"
          @click="copy(item.endpoint)"
          @keydown.enter.prevent="copy(item.endpoint)"
          @keydown.space.prevent="copy(item.endpoint)"
        >{{ item.endpoint }}</code>
      </div>

      <div class="flex flex-shrink-0 items-center gap-1">
        <button
          type="button"
          class="flex-shrink-0 rounded-md p-1 transition-colors"
          :class="copiedEndpoint === item.endpoint
            ? 'text-[var(--apple-success)]'
            : 'text-[var(--apple-muted-2)] hover:bg-[var(--apple-hover)] hover:text-[var(--apple-blue)]'"
          :aria-label="tooltipHint(item.endpoint)"
          @click="copy(item.endpoint)"
        >
          <Icon
            :name="copiedEndpoint === item.endpoint ? 'check' : 'copy'"
            size="xs"
            :stroke-width="copiedEndpoint === item.endpoint ? 2.2 : 2"
          />
        </button>

        <a
          :href="speedTestUrl(item.endpoint)"
          target="_blank"
          rel="noopener noreferrer"
          class="flex-shrink-0 rounded-md p-1 text-[var(--apple-muted-2)] transition-colors hover:bg-[var(--apple-hover)] hover:text-[var(--apple-warning)]"
          :title="t('keys.endpoints.speedTest')"
        >
          <Icon name="bolt" size="xs" :stroke-width="2" />
        </a>
      </div>
    </div>
  </div>
</template>
