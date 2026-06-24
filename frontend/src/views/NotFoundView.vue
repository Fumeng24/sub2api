<template>
  <div
    class="flex min-h-screen items-center justify-center bg-[var(--apple-bg)] px-4 py-12 text-[var(--apple-text)]"
  >
    <div class="w-full max-w-lg text-center">
      <div class="mb-7 border-b border-[color:var(--apple-border-soft)] pb-7">
        <p class="text-8xl font-semibold leading-none text-[var(--apple-muted-2)] sm:text-[10rem]">
          404
        </p>
        <div
          class="mx-auto mt-5 flex h-12 w-12 items-center justify-center rounded-lg border border-[color:var(--apple-border)] bg-[var(--apple-surface)] text-[var(--apple-blue)]"
        >
          <Icon name="exclamationCircle" size="lg" :stroke-width="1.75" />
        </div>
      </div>

      <div class="mb-8">
        <h1 class="mb-3 text-2xl font-semibold tracking-normal text-[var(--apple-text)]">
          {{ t('errors.pageNotFound') }}
        </h1>
        <p class="mx-auto max-w-sm text-sm leading-6 text-[var(--apple-muted)]">
          {{ t('errors.pageNotFoundDescription') }}
        </p>
      </div>

      <div class="flex flex-col justify-center gap-3 sm:flex-row">
        <button @click="goBack" class="btn btn-secondary">
          <Icon name="arrowLeft" size="md" class="mr-2" />
          {{ t('errors.goBack') }}
        </button>
        <router-link to="/" class="btn btn-primary">
          <Icon name="home" size="md" class="mr-2" />
          {{ t('errors.goHome') }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const router = useRouter()

function goBack(): void {
  router.back()
}
</script>
