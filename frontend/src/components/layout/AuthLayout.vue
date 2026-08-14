<template>
  <div class="auth-shell">
    <section class="auth-story" aria-labelledby="auth-story-title">
      <div class="auth-story__pattern" aria-hidden="true">
        <span></span><span></span><span></span><span></span><span></span>
      </div>

      <header class="auth-story__brand">
        <router-link to="/home">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </router-link>
        <em>AI GATEWAY</em>
      </header>

      <div class="auth-story__content">
        <p class="auth-story__kicker">{{ t('auth.gateway.kicker') }}</p>
        <h1 id="auth-story-title">{{ t('auth.gateway.title') }}</h1>
        <p class="auth-story__lead">{{ t('auth.gateway.subtitle') }}</p>

        <div class="auth-signal" :aria-label="t('auth.gateway.routeLabel')">
          <div class="auth-signal__node">
            <span><Icon name="terminal" size="md" /></span>
            <strong>{{ t('auth.gateway.route.application') }}</strong>
            <small>SDK / CLI</small>
          </div>
          <i aria-hidden="true"></i>
          <div class="auth-signal__node auth-signal__node--brand">
            <span><img :src="siteLogo || '/logo.svg'" alt="" /></span>
            <strong>{{ t('auth.gateway.route.gateway') }}</strong>
            <small>One Key</small>
          </div>
          <i aria-hidden="true"></i>
          <div class="auth-signal__node">
            <span><Icon name="cube" size="md" /></span>
            <strong>{{ t('auth.gateway.route.models') }}</strong>
            <small>GPT / Claude / Gemini</small>
          </div>
        </div>

        <div class="auth-story__note">
          <span><Icon name="checkCircle" size="sm" /></span>
          <p>{{ t('auth.gateway.note') }}</p>
        </div>
      </div>

      <footer>
        <span>{{ t('auth.gateway.footer') }}</span>
        <span>{{ currentYear }}</span>
      </footer>
    </section>

    <section class="auth-form-panel">
      <header class="auth-form-panel__topbar">
        <router-link to="/home" class="auth-mobile-brand">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </router-link>
        <LocaleSwitcher />
      </header>

      <div class="auth-form-panel__body">
        <div class="auth-mobile-greeting">
          <p>{{ t('auth.gateway.kicker') }}</p>
          <strong>{{ t('auth.gateway.mobileGreeting') }}</strong>
        </div>

        <div class="auth-form-content">
          <slot />
        </div>

        <div class="auth-form-footer">
          <slot name="footer" />
        </div>

        <p class="auth-copyright">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { DEFAULT_SITE_NAME } from '@/utils/branding'
import { sanitizeUrl } from '@/utils/url'

defineOptions({ name: 'AuthLayout' })

const appStore = useAppStore()
const { t } = useI18n()

const siteName = computed(() => appStore.siteName || DEFAULT_SITE_NAME)
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell,
.auth-shell * {
  box-sizing: border-box;
  letter-spacing: 0;
}

.auth-shell {
  display: grid;
  min-height: 100dvh;
  grid-template-columns: minmax(0, 1fr) minmax(440px, 520px);
  overflow-x: hidden;
  background: #ffffff;
  color: #121923;
}

.auth-story {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 100dvh;
  flex-direction: column;
  overflow: hidden;
  background: #101822;
  color: #ffffff;
}

.auth-story__pattern {
  pointer-events: none;
  position: absolute;
  inset: 0;
  overflow: hidden;
  opacity: 0.6;
}

.auth-story__pattern span {
  position: absolute;
  display: block;
  height: 1px;
  background: rgba(255, 255, 255, 0.1);
  transform-origin: left center;
}

.auth-story__pattern span:nth-child(1) {
  top: 19%;
  left: 48%;
  width: 52%;
  transform: rotate(28deg);
}

.auth-story__pattern span:nth-child(2) {
  top: 31%;
  left: 61%;
  width: 48%;
  background: rgba(53, 197, 182, 0.3);
  transform: rotate(-32deg);
}

.auth-story__pattern span:nth-child(3) {
  bottom: 28%;
  left: 54%;
  width: 54%;
  transform: rotate(21deg);
}

.auth-story__pattern span:nth-child(4) {
  bottom: 15%;
  left: 70%;
  width: 36%;
  background: rgba(223, 116, 79, 0.28);
  transform: rotate(-48deg);
}

.auth-story__pattern span:nth-child(5) {
  top: 12%;
  right: 8%;
  width: 1px;
  height: 76%;
  background: rgba(255, 255, 255, 0.07);
}

.auth-story__brand,
.auth-story__content,
.auth-story footer {
  position: relative;
  z-index: 1;
  width: min(100% - 80px, 760px);
  margin-right: auto;
  margin-left: auto;
}

.auth-story__brand {
  display: flex;
  min-height: 88px;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.auth-story__brand a,
.auth-mobile-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 11px;
  color: inherit;
  text-decoration: none;
}

.auth-story__brand img,
.auth-mobile-brand img {
  width: 38px;
  height: 38px;
  flex: none;
  border-radius: 8px;
  object-fit: contain;
}

.auth-story__brand a span,
.auth-mobile-brand span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 750;
}

.auth-story__brand em {
  color: rgba(255, 255, 255, 0.42);
  font-size: 10px;
  font-style: normal;
  font-weight: 750;
}

.auth-story__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  padding: 54px 0 72px;
}

.auth-story__kicker,
.auth-mobile-greeting p {
  margin: 0;
  color: #5fd3c8;
  font-size: 11px;
  font-weight: 800;
  line-height: 1.45;
  text-transform: uppercase;
}

.auth-story h1 {
  max-width: 650px;
  margin: 18px 0 0;
  white-space: pre-line;
  font-size: 52px;
  font-weight: 660;
  line-height: 1.08;
  overflow-wrap: anywhere;
}

.auth-story__lead {
  max-width: 610px;
  margin: 24px 0 0;
  color: rgba(255, 255, 255, 0.62);
  font-size: 15px;
  line-height: 1.8;
}

.auth-signal {
  display: grid;
  max-width: 680px;
  grid-template-columns: minmax(0, 1fr) 52px minmax(0, 1fr) 52px minmax(0, 1fr);
  align-items: center;
  margin-top: 48px;
}

.auth-signal__node {
  display: flex;
  min-width: 0;
  min-height: 128px;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 7px;
  border-top: 1px solid rgba(255, 255, 255, 0.18);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 18px 4px;
}

.auth-signal__node > span {
  display: inline-flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.07);
  color: #87a9ff;
}

.auth-signal__node--brand > span {
  background: transparent;
}

.auth-signal__node img {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  object-fit: contain;
}

.auth-signal__node strong {
  max-width: 100%;
  overflow: hidden;
  color: #ffffff;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.auth-signal__node small {
  color: rgba(255, 255, 255, 0.38);
  font-size: 9px;
  line-height: 1.5;
}

.auth-signal > i {
  position: relative;
  display: block;
  width: 100%;
  height: 1px;
  background: rgba(95, 211, 200, 0.42);
}

.auth-signal > i::after {
  position: absolute;
  top: -3px;
  right: 0;
  width: 7px;
  height: 7px;
  border-top: 1px solid #5fd3c8;
  border-right: 1px solid #5fd3c8;
  content: '';
  transform: rotate(45deg);
}

.auth-story__note {
  display: flex;
  max-width: 680px;
  align-items: flex-start;
  gap: 12px;
  margin-top: 30px;
  border-left: 2px solid #df744f;
  padding: 5px 0 5px 14px;
}

.auth-story__note > span {
  display: inline-flex;
  margin-top: 2px;
  color: #5fd3c8;
}

.auth-story__note p {
  margin: 0;
  color: rgba(255, 255, 255, 0.54);
  font-size: 11px;
  line-height: 1.65;
}

.auth-story footer {
  display: flex;
  min-height: 64px;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.34);
  font-size: 10px;
}

.auth-form-panel {
  display: flex;
  min-width: 0;
  min-height: 100dvh;
  flex-direction: column;
  overflow-y: auto;
  background: #ffffff;
}

.auth-form-panel__topbar {
  display: flex;
  min-height: 72px;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding: 0 30px;
}

.auth-mobile-brand,
.auth-mobile-greeting {
  display: none;
}

.auth-form-panel__body {
  display: flex;
  width: min(100% - 64px, 390px);
  flex: 1;
  flex-direction: column;
  justify-content: center;
  margin: 0 auto;
  padding: 54px 0 42px;
}

.auth-form-content {
  width: 100%;
}

.auth-form-footer {
  margin-top: 30px;
  color: #6b7684;
  font-size: 13px;
  text-align: center;
}

.auth-copyright {
  margin: 34px 0 0;
  color: #a0a8b2;
  font-size: 10px;
  line-height: 1.6;
  text-align: center;
}

:deep(.input),
:deep(.btn) {
  border-radius: 8px;
}

:deep(.input) {
  min-height: 48px;
  border-color: #d9dee4;
  background: #ffffff;
  color: #121923;
  font-size: 14px;
}

:deep(.input::placeholder) {
  color: #a0a8b2;
}

:deep(.input:focus) {
  border-color: #0f9f91;
  box-shadow: 0 0 0 3px rgba(15, 159, 145, 0.12);
}

:deep(.input-label) {
  margin-bottom: 7px;
  color: #394554;
  font-size: 12px;
  font-weight: 650;
}

:deep(.btn) {
  min-height: 48px;
  font-weight: 700;
}

:deep(.btn-primary) {
  border-color: #0f9f91;
  background: #0f9f91;
  color: #ffffff;
  box-shadow: 0 10px 28px rgba(15, 159, 145, 0.2);
}

:deep(.btn-primary:hover:not(:disabled)) {
  border-color: #087d73;
  background: #087d73;
}

:deep(.btn-secondary) {
  border-color: #d9dee4;
  background: #ffffff;
  color: #263342;
}

:deep(a) {
  color: #087d73;
}

:deep(a:hover) {
  color: #065f58;
}

:global(.dark) .auth-shell,
:global(.dark) .auth-form-panel {
  background: #0d131a;
  color: #f5f7fa;
}

:global(.dark) .auth-form-footer {
  color: #9ca7b3;
}

:global(.dark) .auth-copyright {
  color: #687482;
}

:global(.dark) .auth-shell :deep(.input) {
  border-color: #303a46;
  background: #121a23;
  color: #f5f7fa;
}

:global(.dark) .auth-shell :deep(.input-label) {
  color: #c7d0da;
}

:global(.dark) .auth-shell :deep(.btn-secondary) {
  border-color: #303a46;
  background: #121a23;
  color: #e6ebf0;
}

@media (max-width: 1100px) {
  .auth-shell {
    grid-template-columns: minmax(0, 1fr) minmax(420px, 480px);
  }

  .auth-story h1 {
    font-size: 44px;
  }

  .auth-story__brand,
  .auth-story__content,
  .auth-story footer {
    width: min(100% - 56px, 760px);
  }

  .auth-signal {
    grid-template-columns: minmax(0, 1fr) 34px minmax(0, 1fr) 34px minmax(0, 1fr);
  }
}

@media (max-width: 900px) {
  .auth-shell {
    display: block;
  }

  .auth-story {
    display: none;
  }

  .auth-form-panel__topbar {
    justify-content: space-between;
    border-bottom: 1px solid #e5e9ed;
    padding: 0 22px;
  }

  :global(.dark) .auth-form-panel__topbar {
    border-color: #27313c;
  }

  .auth-mobile-brand {
    display: inline-flex;
  }

  .auth-form-panel__body {
    width: min(100% - 40px, 410px);
    padding-top: 52px;
  }

  .auth-mobile-greeting {
    display: block;
    margin-bottom: 28px;
  }

  .auth-mobile-greeting strong {
    display: block;
    margin-top: 8px;
    color: #121923;
    font-size: 18px;
    line-height: 1.4;
  }

  :global(.dark) .auth-mobile-greeting strong {
    color: #f5f7fa;
  }
}

@media (max-width: 480px) {
  .auth-form-panel__topbar {
    min-height: 64px;
    padding: 0 16px;
  }

  .auth-mobile-brand span {
    max-width: 150px;
  }

  .auth-form-panel__body {
    width: min(100% - 32px, 410px);
    justify-content: flex-start;
    padding: 42px 0 32px;
  }
}
</style>
