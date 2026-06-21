<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-hidden bg-gray-50 text-gray-950 dark:bg-dark-950 dark:text-white"
  >
    <!-- Header -->
    <header class="relative z-20 border-b border-gray-200/70 bg-white/90 px-4 py-4 backdrop-blur dark:border-dark-800/70 dark:bg-dark-950/80 sm:px-6">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <!-- Logo -->
        <div class="flex items-center">
          <div class="h-9 w-9 overflow-hidden rounded-lg bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:ring-dark-800">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            class="rounded-md p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
            @click="handleDocsLinkClick"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-md p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-lg bg-gray-950 py-1 pl-1 pr-2.5 transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-md bg-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="text-xs font-medium text-white">{{ t('home.dashboard') }}</span>
            <svg
              class="h-3 w-3 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25"
              />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-lg bg-gray-950 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-4 py-12 sm:px-6 lg:py-14">
      <div class="mx-auto max-w-6xl">
        <!-- Hero Section - Left/Right Layout -->
        <div class="mb-10 flex flex-col items-center justify-between gap-10 lg:flex-row lg:gap-14">
          <!-- Left: Text Content -->
          <div class="flex-1 text-center lg:text-left">
            <div
              class="mb-5 inline-flex items-center gap-2 rounded-md border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-primary-700 shadow-sm dark:border-dark-800 dark:bg-dark-900 dark:text-primary-300"
            >
              <Icon name="sparkles" size="sm" />
              {{ t('home.heroSubtitle') }}
            </div>
            <h1
              class="mb-4 text-4xl font-semibold tracking-tight text-gray-950 dark:text-white md:text-5xl"
            >
              {{ siteName }}
            </h1>
            <p v-if="siteSubtitle" class="mb-3 text-base font-medium text-gray-600 dark:text-dark-300">
              {{ siteSubtitle }}
            </p>
            <p class="mx-auto mb-7 max-w-2xl text-base leading-7 text-gray-600 dark:text-dark-300 lg:mx-0">
              {{ t('home.heroDescription') }}
            </p>

            <!-- CTA Button -->
            <div class="flex flex-col items-center justify-center gap-3 sm:flex-row lg:justify-start">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/register'"
                class="btn btn-primary px-6 py-2.5 text-sm shadow-sm"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="md" class="ml-2" :stroke-width="2" />
              </router-link>
              <a
                v-if="docsLink"
                :href="docsLink.href"
                :target="docsLink.external ? '_blank' : undefined"
                :rel="docsLink.external ? 'noopener noreferrer' : undefined"
                class="inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-gray-700 shadow-sm transition hover:border-primary-200 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200 dark:hover:border-primary-800 dark:hover:text-primary-300"
                @click="handleDocsLinkClick"
              >
                <Icon name="book" size="sm" />
                {{ t('home.docs') }}
                <span class="rounded bg-primary-50 px-2 py-0.5 text-[10px] text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
                  {{ t('home.docsBadge') }}
                </span>
              </a>
            </div>
          </div>

          <!-- Right: Terminal Animation -->
          <div class="flex flex-1 justify-center lg:justify-end">
            <div class="terminal-container">
              <div class="terminal-window">
                <!-- Window header -->
                <div class="terminal-header">
                  <div class="terminal-buttons">
                    <span class="btn-close"></span>
                    <span class="btn-minimize"></span>
                    <span class="btn-maximize"></span>
                  </div>
                  <span class="terminal-title">terminal</span>
                </div>
                <!-- Terminal content -->
                <div class="terminal-body">
                  <div class="code-line line-1">
                    <span class="code-prompt">$</span>
                    <span class="code-cmd">curl</span>
                    <span class="code-flag">-X POST</span>
                    <span class="code-url">/v1/messages</span>
                  </div>
                  <div class="code-line line-2">
                    <span class="code-comment"># {{ t('home.terminal.scheduling') }}</span>
                  </div>
                  <div class="code-line line-3">
                    <span class="code-success">200 OK</span>
                    <span class="code-response">{ "content": "Hello!" }</span>
                  </div>
                  <div class="code-line line-4">
                    <span class="code-prompt">$</span>
                    <span class="cursor"></span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Value Rail -->
        <div class="mb-12 grid gap-3 md:grid-cols-4">
          <div
            v-for="item in valueRail"
            :key="item.value"
            class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <p class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">
              {{ item.value }}
            </p>
            <p class="mt-1 text-sm font-bold text-gray-800 dark:text-dark-100">
              {{ item.label }}
            </p>
            <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
              {{ item.description }}
            </p>
          </div>
        </div>

        <!-- Feature Tags - Centered -->
        <div class="mb-12 flex flex-wrap items-center justify-center gap-4 md:gap-6">
          <div
            class="inline-flex items-center gap-2.5 rounded-full border border-gray-200/50 bg-white/80 px-5 py-2.5 shadow-sm backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/80"
          >
            <Icon name="swap" size="sm" class="text-primary-500" />
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{
              t('home.tags.subscriptionToApi')
            }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2.5 rounded-full border border-gray-200/50 bg-white/80 px-5 py-2.5 shadow-sm backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/80"
          >
            <Icon name="shield" size="sm" class="text-primary-500" />
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{
              t('home.tags.stickySession')
            }}</span>
          </div>
          <div
            class="inline-flex items-center gap-2.5 rounded-full border border-gray-200/50 bg-white/80 px-5 py-2.5 shadow-sm backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/80"
          >
            <Icon name="chart" size="sm" class="text-primary-500" />
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{
              t('home.tags.realtimeBilling')
            }}</span>
          </div>
        </div>

        <!-- Cost & Quick Start -->
        <div class="mb-12 grid gap-6 lg:grid-cols-[1.05fr_0.95fr]">
          <section
            class="relative overflow-hidden rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <div class="relative">
              <div class="mb-4 inline-flex items-center gap-2 rounded-md bg-emerald-50 px-3 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200">
                <Icon name="calculator" size="sm" />
                {{ t('home.cost.kicker') }}
              </div>
              <h2 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white md:text-3xl">
                {{ t('home.cost.title') }}
              </h2>
              <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-dark-300">
                {{ t('home.cost.description') }}
              </p>
              <div class="mt-5 grid gap-3 sm:grid-cols-3">
                <div
                  v-for="item in costFacts"
                  :key="item.label"
                  class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-800 dark:bg-dark-950/60"
                >
                  <p class="text-lg font-semibold text-emerald-700 dark:text-emerald-300">
                    {{ item.value }}
                  </p>
                  <p class="mt-1 text-xs font-bold text-gray-700 dark:text-dark-200">
                    {{ item.label }}
                  </p>
                </div>
              </div>
              <div class="mt-5 rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 text-sm leading-7 text-gray-700 dark:border-dark-700 dark:bg-dark-950/60 dark:text-dark-200">
                <p class="font-bold text-gray-950 dark:text-white">{{ t('home.cost.formulaTitle') }}</p>
                <p class="mt-1">{{ t('home.cost.formula') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('home.cost.note') }}</p>
              </div>
            </div>
          </section>

          <section
            class="rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <div class="mb-4 inline-flex items-center gap-2 rounded-md bg-gray-950 px-3 py-1 text-xs font-medium text-white dark:bg-white dark:text-gray-950">
              <Icon name="bolt" size="sm" />
              {{ t('home.quickStart.kicker') }}
            </div>
            <h2 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white md:text-3xl">
              {{ t('home.quickStart.title') }}
            </h2>
            <div class="mt-5 space-y-3">
              <div
                v-for="(step, index) in quickStartSteps"
                :key="step.title"
                class="flex gap-4 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-800 dark:bg-dark-950/60"
              >
                <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary-600 text-sm font-semibold text-white">
                  {{ index + 1 }}
                </div>
                <div>
                  <p class="font-bold text-gray-950 dark:text-white">{{ step.title }}</p>
                  <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">
                    {{ step.description }}
                  </p>
                </div>
              </div>
            </div>
            <div class="mt-5 flex flex-col gap-3 sm:flex-row">
              <router-link
                :to="isAuthenticated ? '/keys' : '/register'"
                class="btn btn-primary justify-center"
              >
                {{ isAuthenticated ? t('home.quickStart.createKey') : t('home.quickStart.register') }}
              </router-link>
              <a
                v-if="docsLink"
                :href="docsLink.href"
                :target="docsLink.external ? '_blank' : undefined"
                :rel="docsLink.external ? 'noopener noreferrer' : undefined"
                class="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200 dark:hover:border-primary-700 dark:hover:text-primary-300"
                @click="handleDocsLinkClick"
              >
                <Icon name="book" size="sm" />
                {{ t('home.quickStart.readGuide') }}
              </a>
            </div>
          </section>
        </div>

        <!-- Concrete Integration Path -->
        <div class="mb-12">
          <div class="mb-6 text-center">
            <div class="mb-3 inline-flex items-center gap-2 rounded-md bg-gray-950 px-3 py-1 text-xs font-medium text-white dark:bg-white dark:text-gray-950">
              <Icon name="terminal" size="sm" />
              {{ t('home.integration.kicker') }}
            </div>
            <h2 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white md:text-3xl">
              {{ t('home.integration.title') }}
            </h2>
            <p class="mx-auto mt-3 max-w-3xl text-sm leading-7 text-gray-600 dark:text-dark-300">
              {{ t('home.integration.description') }}
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <component
              :is="item.external ? 'a' : 'router-link'"
              v-for="item in integrationCards"
              :key="item.title"
              :to="item.external ? undefined : item.href"
              :href="item.external ? item.href : undefined"
              :target="item.external ? '_blank' : undefined"
              :rel="item.external ? 'noopener noreferrer' : undefined"
              class="group flex min-h-full flex-col rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition-colors hover:border-primary-200 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-primary-800"
            >
              <div class="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-primary-600 text-white">
                <Icon :name="item.icon" size="md" />
              </div>
              <h3 class="text-base font-semibold text-gray-950 dark:text-white">
                {{ item.title }}
              </h3>
              <p class="mt-2 flex-1 text-sm leading-6 text-gray-600 dark:text-dark-300">
                {{ item.description }}
              </p>
              <span class="mt-4 inline-flex items-center gap-1 text-sm font-bold text-primary-700 dark:text-primary-300">
                {{ item.action }}
                <Icon name="arrowRight" size="sm" />
              </span>
            </component>
          </div>
        </div>

        <!-- Features Grid -->
        <div class="mb-12 grid gap-6 md:grid-cols-3">
          <!-- Feature 1: Unified Gateway -->
          <div
            class="group rounded-lg border border-gray-200 bg-white p-6 shadow-sm transition-colors hover:border-primary-200 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-primary-800"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-blue-600"
            >
              <Icon name="server" size="lg" class="text-white" />
            </div>
            <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.unifiedGateway') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-400">
              {{ t('home.features.unifiedGatewayDesc') }}
            </p>
          </div>

          <!-- Feature 2: Account Pool -->
          <div
            class="group rounded-lg border border-gray-200 bg-white p-6 shadow-sm transition-colors hover:border-primary-200 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-primary-800"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary-600"
            >
              <svg
                class="h-6 w-6 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.multiAccount') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-400">
              {{ t('home.features.multiAccountDesc') }}
            </p>
          </div>

          <!-- Feature 3: Billing & Quota -->
          <div
            class="group rounded-lg border border-gray-200 bg-white p-6 shadow-sm transition-colors hover:border-primary-200 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-primary-800"
          >
            <div
              class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-violet-600"
            >
              <svg
                class="h-6 w-6 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
                />
              </svg>
            </div>
            <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.balanceQuota') }}
            </h3>
            <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-400">
              {{ t('home.features.balanceQuotaDesc') }}
            </p>
          </div>
        </div>

        <!-- Detailed Use Cases -->
        <div class="mb-12">
          <div class="mb-6 text-center">
            <h2 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white md:text-3xl">
              {{ t('home.useCases.title') }}
            </h2>
            <p class="mx-auto mt-3 max-w-2xl text-sm leading-7 text-gray-600 dark:text-dark-300">
              {{ t('home.useCases.description') }}
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="item in useCases"
              :key="item.title"
              class="group rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition-colors hover:border-primary-200 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-primary-800"
            >
              <div
                class="mb-4 flex h-11 w-11 items-center justify-center rounded-lg"
                :class="item.iconClass"
              >
                <Icon :name="item.icon" size="lg" class="text-white" />
              </div>
              <h3 class="text-base font-semibold text-gray-950 dark:text-white">
                {{ item.title }}
              </h3>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
                {{ item.description }}
              </p>
            </div>
          </div>
        </div>

        <!-- Trust Notes -->
        <div class="mb-12 grid gap-6 lg:grid-cols-[0.85fr_1.15fr]">
          <section class="rounded-lg border border-gray-200 bg-gray-950 p-6 text-white shadow-sm dark:border-dark-800 dark:bg-black">
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-white/10">
              <Icon name="shield" size="lg" />
            </div>
            <h2 class="text-2xl font-semibold tracking-tight">{{ t('home.trust.title') }}</h2>
            <p class="mt-3 text-sm leading-7 text-gray-300">
              {{ t('home.trust.description') }}
            </p>
            <div class="mt-5 grid gap-3">
              <div
                v-for="item in trustPoints"
                :key="item"
                class="flex items-start gap-3 rounded-lg border border-white/10 bg-white/5 p-3 text-sm leading-6 text-gray-200"
              >
                <Icon name="checkCircle" size="sm" class="mt-0.5 shrink-0 text-emerald-300" />
                <span>{{ item }}</span>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-800 dark:bg-dark-900">
            <div class="mb-4 inline-flex items-center gap-2 rounded-md bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/40 dark:text-primary-200">
              <Icon name="lightbulb" size="sm" />
              {{ t('home.firstRun.kicker') }}
            </div>
            <h2 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white md:text-3xl">
              {{ t('home.firstRun.title') }}
            </h2>
            <p class="mt-3 text-sm leading-7 text-gray-600 dark:text-dark-300">
              {{ t('home.firstRun.description') }}
            </p>
            <div class="mt-5 grid gap-3 sm:grid-cols-2">
              <div
                v-for="item in firstRunTips"
                :key="item.title"
                class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-800 dark:bg-dark-950/60"
              >
                <p class="font-bold text-gray-950 dark:text-white">{{ item.title }}</p>
                <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ item.description }}</p>
              </div>
            </div>
          </section>
        </div>

        <!-- Supported Providers -->
        <div class="mb-8 text-center">
          <h2 class="mb-3 text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('home.providers.title') }}
          </h2>
          <p class="text-sm text-gray-600 dark:text-dark-400">
            {{ t('home.providers.description') }}
          </p>
        </div>

        <div class="mb-16 flex flex-wrap items-center justify-center gap-4">
          <!-- Claude - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-5 py-3 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-md bg-orange-500"
            >
              <span class="text-xs font-bold text-white">C</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.claude') }}</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- GPT - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-5 py-3 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-md bg-green-600"
            >
              <span class="text-xs font-bold text-white">G</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">GPT</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Gemini - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-5 py-3 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-md bg-blue-600"
            >
              <span class="text-xs font-bold text-white">G</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.gemini') }}</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- Antigravity - Supported -->
          <div
            class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-5 py-3 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-md bg-rose-600"
            >
              <span class="text-xs font-bold text-white">A</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.antigravity') }}</span>
            <span
              class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >{{ t('home.providers.supported') }}</span
            >
          </div>
          <!-- More - Coming Soon -->
          <div
            class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-5 py-3 opacity-70 shadow-sm dark:border-dark-800 dark:bg-dark-900"
          >
            <div
              class="flex h-8 w-8 items-center justify-center rounded-md bg-gray-600"
            >
              <span class="text-xs font-bold text-white">+</span>
            </div>
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('home.providers.more') }}</span>
            <span
              class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700 dark:text-dark-400"
              >{{ t('home.providers.soon') }}</span
            >
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/50 px-6 py-8 dark:border-dark-800/50">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docsLink"
            :href="docsLink.href"
            :target="docsLink.external ? '_blank' : undefined"
            :rel="docsLink.external ? 'noopener noreferrer' : undefined"
            class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-white"
            @click="handleDocsLinkClick"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-white"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { resolveDocsLink, shouldUseClientDocsNavigation } from '@/utils/docsLink'

const { t } = useI18n()
const router = useRouter()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const rawDocUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const docsLink = computed(() => resolveDocsLink(rawDocUrl.value, appStore.cachedPublicSettings?.custom_menu_items ?? []))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

type HomeIconName = InstanceType<typeof Icon>['$props']['name']

const valueRail = computed(() => [
  {
    value: t('home.valueRail.credit.value'),
    label: t('home.valueRail.credit.label'),
    description: t('home.valueRail.credit.description')
  },
  {
    value: t('home.valueRail.models.value'),
    label: t('home.valueRail.models.label'),
    description: t('home.valueRail.models.description')
  },
  {
    value: t('home.valueRail.compatible.value'),
    label: t('home.valueRail.compatible.label'),
    description: t('home.valueRail.compatible.description')
  },
  {
    value: t('home.valueRail.billing.value'),
    label: t('home.valueRail.billing.label'),
    description: t('home.valueRail.billing.description')
  }
])

const costFacts = computed(() => [
  { value: t('home.cost.facts.creditValue'), label: t('home.cost.facts.creditLabel') },
  { value: t('home.cost.facts.multiplierValue'), label: t('home.cost.facts.multiplierLabel') },
  { value: t('home.cost.facts.recordsValue'), label: t('home.cost.facts.recordsLabel') }
])

const quickStartSteps = computed(() => [
  {
    title: t('home.quickStart.steps.register.title'),
    description: t('home.quickStart.steps.register.description')
  },
  {
    title: t('home.quickStart.steps.group.title'),
    description: t('home.quickStart.steps.group.description')
  },
  {
    title: t('home.quickStart.steps.connect.title'),
    description: t('home.quickStart.steps.connect.description')
  }
])

const integrationCards = computed<Array<{
  title: string
  description: string
  action: string
  href: string
  icon: HomeIconName
  external?: boolean
}>>(() => [
  {
    title: t('home.integration.cards.key.title'),
    description: t('home.integration.cards.key.description'),
    action: t('home.integration.cards.key.action'),
    href: isAuthenticated.value ? '/keys' : '/register',
    icon: 'key'
  },
  {
    title: t('home.integration.cards.status.title'),
    description: t('home.integration.cards.status.description'),
    action: t('home.integration.cards.status.action'),
    href: isAuthenticated.value ? '/monitor' : '/register',
    icon: 'shield'
  },
  {
    title: t('home.integration.cards.models.title'),
    description: t('home.integration.cards.models.description'),
    action: t('home.integration.cards.models.action'),
    href: isAuthenticated.value ? '/available-channels' : '/register',
    icon: 'server'
  },
  {
    title: t('home.integration.cards.billing.title'),
    description: t('home.integration.cards.billing.description'),
    action: t('home.integration.cards.billing.action'),
    href: isAuthenticated.value ? '/usage' : '/key-usage',
    icon: 'chart'
  }
])

const useCases = computed<Array<{
  title: string
  description: string
  icon: HomeIconName
  iconClass: string
}>>(() => [
  {
    title: t('home.useCases.coding.title'),
    description: t('home.useCases.coding.description'),
    icon: 'terminal',
    iconClass: 'bg-gray-950'
  },
  {
    title: t('home.useCases.writing.title'),
    description: t('home.useCases.writing.description'),
    icon: 'document',
    iconClass: 'bg-orange-600'
  },
  {
    title: t('home.useCases.automation.title'),
    description: t('home.useCases.automation.description'),
    icon: 'cpu',
    iconClass: 'bg-sky-600'
  },
  {
    title: t('home.useCases.image.title'),
    description: t('home.useCases.image.description'),
    icon: 'sparkles',
    iconClass: 'bg-rose-600'
  }
])

const trustPoints = computed(() => [
  t('home.trust.points.modelTruth'),
  t('home.trust.points.billing'),
  t('home.trust.points.status'),
  t('home.trust.points.privacy')
])

const firstRunTips = computed(() => [
  {
    title: t('home.firstRun.tips.trySmall.title'),
    description: t('home.firstRun.tips.trySmall.description')
  },
  {
    title: t('home.firstRun.tips.serviceStatus.title'),
    description: t('home.firstRun.tips.serviceStatus.description')
  },
  {
    title: t('home.firstRun.tips.groupChoice.title'),
    description: t('home.firstRun.tips.groupChoice.description')
  },
  {
    title: t('home.firstRun.tips.records.title'),
    description: t('home.firstRun.tips.records.description')
  }
])

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function handleDocsLinkClick(event: MouseEvent) {
  const link = docsLink.value
  if (!shouldUseClientDocsNavigation(event, link)) return
  event.preventDefault()
  router.push(link?.route || link?.href || '/')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
/* Terminal Container */
.terminal-container {
  position: relative;
  display: inline-block;
  max-width: 100%;
}

/* Terminal Window */
.terminal-window {
  width: min(420px, calc(100vw - 3rem));
  background: linear-gradient(145deg, #1e293b 0%, #0f172a 100%);
  border-radius: 8px;
  box-shadow:
    0 16px 36px -24px rgba(15, 23, 42, 0.45),
    0 0 0 1px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  overflow: hidden;
  transform: none;
  transition: transform 0.3s ease;
}

.terminal-window:hover {
  transform: translateY(-2px);
}

/* Terminal Header */
.terminal-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: rgba(30, 41, 59, 0.8);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.terminal-buttons {
  display: flex;
  gap: 8px;
}

.terminal-buttons span {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.btn-close {
  background: #ef4444;
}
.btn-minimize {
  background: #eab308;
}
.btn-maximize {
  background: #22c55e;
}

.terminal-title {
  flex: 1;
  text-align: center;
  font-size: 12px;
  font-family: ui-monospace, monospace;
  color: #64748b;
  margin-right: 52px;
}

/* Terminal Body */
.terminal-body {
  padding: 20px 24px;
  font-family: ui-monospace, 'Fira Code', monospace;
  font-size: 14px;
  line-height: 2;
}

.code-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  opacity: 0;
  animation: line-appear 0.5s ease forwards;
}

.line-1 {
  animation-delay: 0.3s;
}
.line-2 {
  animation-delay: 1s;
}
.line-3 {
  animation-delay: 1.8s;
}
.line-4 {
  animation-delay: 2.5s;
}

@keyframes line-appear {
  from {
    opacity: 0;
    transform: translateY(5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.code-prompt {
  color: #22c55e;
  font-weight: bold;
}
.code-cmd {
  color: #38bdf8;
}
.code-flag {
  color: #a78bfa;
}
.code-url {
  color: #60a5fa;
}
.code-comment {
  color: #64748b;
  font-style: italic;
}
.code-success {
  color: #22c55e;
  background: rgba(34, 197, 94, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
}
.code-response {
  color: #fbbf24;
}

/* Blinking Cursor */
.cursor {
  display: inline-block;
  width: 8px;
  height: 16px;
  background: #22c55e;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

/* Dark mode adjustments */
:deep(.dark) .terminal-window {
  box-shadow:
    0 16px 36px -24px rgba(0, 0, 0, 0.7),
    0 0 0 1px rgba(37, 99, 235, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
}

@media (max-width: 640px) {
  .terminal-body {
    padding: 16px;
    font-size: 12px;
  }

  .terminal-window {
    transform: none;
  }

  .terminal-window:hover {
    transform: translateY(-2px);
  }
}
</style>
