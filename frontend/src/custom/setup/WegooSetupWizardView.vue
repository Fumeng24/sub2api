<template>
  <div class="setup-gateway-shell dark min-h-screen px-4 py-8 text-white sm:px-6">
    <div class="mx-auto w-full max-w-5xl">
      <!-- Logo & Title -->
      <header class="setup-gateway-hero mb-6">
        <div class="min-w-0">
          <p class="setup-gateway-kicker">
            AI Gateway · First Run
          </p>
          <div class="mt-4 flex items-start gap-3">
            <div
              class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/[0.06] text-[#4dd4e6]"
            >
              <Icon name="cog" size="lg" />
            </div>
            <div class="min-w-0">
              <h1 class="text-2xl font-semibold tracking-normal text-white sm:text-3xl">
                {{ t('setup.title') }}
              </h1>
              <p class="mt-2 max-w-2xl text-sm leading-6 text-white/58">
                {{ t('setup.description') }}
              </p>
            </div>
          </div>
        </div>
        <aside class="setup-gateway-status">
          <p class="text-xs font-medium uppercase text-white/42">
            {{ t('common.status') }}
          </p>
          <p class="mt-2 text-lg font-semibold text-white">
            {{ steps[currentStep]?.title }}
          </p>
          <p class="mt-1 text-xs leading-5 text-white/48">
            {{ currentStep + 1 }} / {{ steps.length }}
          </p>
        </aside>
      </header>

      <!-- Progress Steps -->
      <div class="mb-6 rounded-lg border border-white/10 bg-white/[0.035] p-3">
        <div class="flex flex-wrap items-center justify-center gap-2 sm:flex-nowrap sm:gap-0">
          <template v-for="(step, index) in steps" :key="step.id">
            <div class="flex items-center">
              <div
                :class="[
                  'flex h-10 w-10 items-center justify-center rounded-lg text-sm font-semibold transition-all',
                  currentStep > index
                    ? 'bg-[#4dd4e6] text-[#071014]'
                    : currentStep === index
                      ? 'bg-[#4dd4e6] text-[#071014] ring-4 ring-[#4dd4e6]/15'
                      : 'bg-white/[0.06] text-white/42 ring-1 ring-white/10'
                ]"
              >
                <Icon
                  v-if="currentStep > index"
                  name="check"
                  size="md"
                  :stroke-width="2"
                />
                <span v-else>{{ index + 1 }}</span>
              </div>
              <span
                class="ml-2 hidden text-sm font-medium sm:inline"
                :class="
                  currentStep >= index
                    ? 'text-white'
                    : 'text-white/38'
                "
              >
                {{ step.title }}
              </span>
            </div>
            <div
              v-if="index < steps.length - 1"
              class="mx-3 hidden h-0.5 w-12 sm:block"
              :class="currentStep > index ? 'bg-[#4dd4e6]' : 'bg-white/10'"
            ></div>
          </template>
        </div>
      </div>

      <!-- Step Content -->
      <div class="setup-gateway-panel rounded-lg border border-white/10 bg-[#0d1219]/82 p-5 shadow-[0_24px_70px_rgba(0,0,0,0.30)] sm:p-8">
        <!-- Step 1: Database -->
        <div v-if="currentStep === 0" data-test="setup-step-database" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.database.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.database.description') }}
            </p>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('setup.database.host') }}</label>
              <input
                v-model="formData.database.host"
                type="text"
                class="input"
                placeholder="localhost"
              />
            </div>
            <div>
              <label class="input-label">{{ t('setup.database.port') }}</label>
              <input
                v-model.number="formData.database.port"
                type="number"
                class="input"
                placeholder="5432"
              />
            </div>
          </div>

          <div class="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.035] p-3">
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t("setup.redis.enableTls") }}
              </p>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                {{ t("setup.redis.enableTlsHint") }}
              </p>
            </div>
            <Toggle v-model="formData.redis.enable_tls" />
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('setup.database.username') }}</label>
              <input
                v-model="formData.database.user"
                type="text"
                class="input"
                placeholder="postgres"
              />
            </div>
            <div>
              <label class="input-label">{{ t('setup.database.password') }}</label>
              <input
                v-model="formData.database.password"
                type="password"
                class="input"
                :placeholder="t('setup.database.passwordPlaceholder')"
              />
            </div>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('setup.database.databaseName') }}</label>
              <input
                v-model="formData.database.dbname"
                type="text"
                class="input"
                placeholder="sub2api"
              />
            </div>
            <div>
              <label class="input-label">{{ t('setup.database.sslMode') }}</label>
              <Select
                v-model="formData.database.sslmode"
                :options="[
                  { value: 'disable', label: t('setup.database.ssl.disable') },
                  { value: 'require', label: t('setup.database.ssl.require') },
                  { value: 'verify-ca', label: t('setup.database.ssl.verifyCa') },
                  { value: 'verify-full', label: t('setup.database.ssl.verifyFull') }
                ]"
              />
            </div>
          </div>

          <button
            data-test="setup-test-db"
            @click="testDatabaseConnection"
            :disabled="testingDb"
            class="btn btn-secondary w-full"
          >
            <svg
              v-if="testingDb"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <Icon v-else-if="dbConnected" name="check" size="md" class="mr-2 text-green-500" :stroke-width="2" />
            {{
              testingDb
                ? t('setup.status.testing')
                : dbConnected
                  ? t('setup.status.success')
                  : t('setup.status.testConnection')
            }}
          </button>
        </div>

        <!-- Step 2: Redis -->
        <div v-if="currentStep === 1" data-test="setup-step-redis" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.redis.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.redis.description') }}
            </p>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('setup.redis.host') }}</label>
              <input
                v-model="formData.redis.host"
                type="text"
                class="input"
                placeholder="localhost"
              />
            </div>
            <div>
              <label class="input-label">{{ t('setup.redis.port') }}</label>
              <input
                v-model.number="formData.redis.port"
                type="number"
                class="input"
                placeholder="6379"
              />
            </div>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('setup.redis.password') }}</label>
              <input
                v-model="formData.redis.password"
                type="password"
                class="input"
                :placeholder="t('setup.redis.passwordPlaceholder')"
              />
            </div>
            <div>
              <label class="input-label">{{ t('setup.redis.database') }}</label>
              <input
                v-model.number="formData.redis.db"
                type="number"
                class="input"
                placeholder="0"
              />
            </div>
          </div>

          <div class="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.035] p-3">
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t("setup.redis.enableTls") }}
              </p>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                {{ t("setup.redis.enableTlsHint") }}
              </p>
            </div>
            <Toggle v-model="formData.redis.enable_tls" />
          </div>

          <button
            data-test="setup-test-redis"
            @click="testRedisConnection"
            :disabled="testingRedis"
            class="btn btn-secondary w-full"
          >
            <svg
              v-if="testingRedis"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <Icon
              v-else-if="redisConnected"
              name="check"
              size="md"
              class="mr-2 text-green-500"
              :stroke-width="2"
            />
            {{
              testingRedis
                ? t('setup.status.testing')
                : redisConnected
                  ? t('setup.status.success')
                  : t('setup.status.testConnection')
            }}
          </button>
        </div>

        <!-- Step 3: Admin -->
        <div v-if="currentStep === 2" data-test="setup-step-admin" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.admin.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.admin.description') }}
            </p>
          </div>

          <div>
            <label class="input-label">{{ t('setup.admin.email') }}</label>
            <input
              data-test="setup-admin-email"
              v-model="formData.admin.email"
              type="email"
              class="input"
              placeholder="admin@example.com"
            />
          </div>

          <div>
            <label class="input-label">{{ t('setup.admin.password') }}</label>
            <input
              data-test="setup-admin-password"
              v-model="formData.admin.password"
              type="password"
              class="input"
              :placeholder="t('setup.admin.passwordPlaceholder')"
            />
          </div>

          <div>
            <label class="input-label">{{ t('setup.admin.confirmPassword') }}</label>
            <input
              data-test="setup-admin-confirm-password"
              v-model="confirmPassword"
              type="password"
              class="input"
              :placeholder="t('setup.admin.confirmPasswordPlaceholder')"
            />
            <p
              v-if="confirmPassword && formData.admin.password !== confirmPassword"
              class="input-error-text"
            >
              {{ t('setup.admin.passwordMismatch') }}
            </p>
          </div>
        </div>

        <!-- Step 4: Complete -->
        <div v-if="currentStep === 3" data-test="setup-step-complete" class="space-y-6">
          <div class="mb-6 text-center">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('setup.ready.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('setup.ready.description') }}
            </p>
          </div>

          <div class="space-y-4">
            <div class="rounded-lg border border-white/10 bg-white/[0.035] p-4">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('setup.ready.database') }}
              </h3>
              <p class="break-words text-gray-900 dark:text-white">
                {{ formData.database.user }}@{{ formData.database.host }}:{{
                  formData.database.port
                }}/{{ formData.database.dbname }}
              </p>
            </div>

            <div class="rounded-lg border border-white/10 bg-white/[0.035] p-4">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('setup.ready.redis') }}
              </h3>
              <p class="break-words text-gray-900 dark:text-white">
                {{ formData.redis.host }}:{{ formData.redis.port }}
              </p>
            </div>

            <div class="rounded-lg border border-white/10 bg-white/[0.035] p-4">
              <h3 class="mb-2 text-sm font-medium text-gray-500 dark:text-dark-400">
                {{ t('setup.ready.adminEmail') }}
              </h3>
              <p class="break-words text-gray-900 dark:text-white">{{ formData.admin.email }}</p>
            </div>
          </div>
        </div>

        <!-- Error Message -->
        <div
          v-if="errorMessage"
          class="mt-6 rounded-md border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationCircle" size="md" class="flex-shrink-0 text-red-500" />
            <p class="text-sm text-red-700 dark:text-red-400">{{ errorMessage }}</p>
          </div>
        </div>

        <!-- Success Message -->
        <div
          v-if="installSuccess"
          class="mt-6 rounded-md border border-green-200 bg-green-50 p-4 dark:border-green-800/50 dark:bg-green-900/20"
        >
          <div class="flex items-start gap-3">
            <svg
              v-if="!serviceReady"
              class="h-5 w-5 flex-shrink-0 animate-spin text-green-500"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <Icon v-else name="checkCircle" size="md" class="flex-shrink-0 text-green-500" />
            <div>
              <p class="text-sm font-medium text-green-700 dark:text-green-400">
                {{ t('setup.status.completed') }}
              </p>
              <p class="mt-1 text-sm text-green-600 dark:text-green-500">
                {{
                  serviceReady
                    ? t('setup.status.redirecting')
                    : t('setup.status.restarting')
                }}
              </p>
            </div>
          </div>
        </div>

        <!-- Navigation Buttons -->
        <div class="mt-8 flex flex-col-reverse gap-3 sm:flex-row sm:justify-between">
          <button
            v-if="currentStep > 0 && !installSuccess"
            @click="currentStep--"
            class="btn btn-secondary w-full sm:w-auto"
          >
            <Icon name="chevronLeft" size="sm" class="mr-2" :stroke-width="2" />
            {{ t('common.back') }}
          </button>
          <div v-else></div>

          <button
            v-if="currentStep < 3"
            data-test="setup-next"
            @click="nextStep"
            :disabled="!canProceed"
            class="btn btn-primary w-full sm:w-auto"
          >
            {{ t('common.next') }}
            <Icon name="chevronRight" size="sm" class="ml-2" :stroke-width="2" />
          </button>

          <button
            v-else-if="!installSuccess"
            data-test="setup-install"
            @click="performInstall"
            :disabled="installing"
            class="btn btn-primary w-full sm:w-auto"
          >
            <svg
              v-if="installing"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ installing ? t('setup.status.installing') : t('setup.status.completeInstallation') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { testDatabase, testRedis, install, type InstallRequest } from '@/api/setup'
import { buildGatewayUrl } from '@/api/client'
import Select from '@/custom/common/WegooSelect.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const steps = computed(() => [
  { id: 'database', title: t('setup.database.title') },
  { id: 'redis', title: t('setup.redis.title') },
  { id: 'admin', title: t('setup.admin.title') },
  { id: 'complete', title: t('setup.ready.title') }
])

const currentStep = ref(0)
const errorMessage = ref('')
const installSuccess = ref(false)

// Connection test states
const testingDb = ref(false)
const testingRedis = ref(false)
const dbConnected = ref(false)
const redisConnected = ref(false)
const installing = ref(false)
const confirmPassword = ref('')
const serviceReady = ref(false)

// Default server port
const getCurrentPort = (): number => {
  const port = window.location.port
  if (port) {
    return parseInt(port, 10)
  }

  return window.location.protocol === 'https:' ? 443 : 80
}

const formData = reactive<InstallRequest>({
  database: {
    host: 'localhost',
    port: 5432,
    user: 'postgres',
    password: '',
    dbname: 'sub2api',
    sslmode: 'disable'
  },
  redis: {
    host: 'localhost',
    port: 6379,
    username: '',
    password: '',
    db: 0,
    enable_tls: false
  },
  admin: {
    email: '',
    password: ''
  },
  server: {
    host: '0.0.0.0',
    port: getCurrentPort(), // Use current port from browser
    mode: 'release'
  }
})

const canProceed = computed(() => {
  switch (currentStep.value) {
    case 0:
      return dbConnected.value
    case 1:
      return redisConnected.value
    case 2:
      return (
        formData.admin.email &&
        formData.admin.password.length >= 8 &&
        formData.admin.password === confirmPassword.value
      )
    default:
      return true
  }
})

async function testDatabaseConnection() {
  testingDb.value = true
  errorMessage.value = ''
  dbConnected.value = false

  try {
    await testDatabase(formData.database)
    dbConnected.value = true
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string; message?: string } }; message?: string }
    errorMessage.value =
      err.response?.data?.detail || err.response?.data?.message || err.message || 'Connection failed'
  } finally {
    testingDb.value = false
  }
}

async function testRedisConnection() {
  testingRedis.value = true
  errorMessage.value = ''
  redisConnected.value = false

  try {
    await testRedis(formData.redis)
    redisConnected.value = true
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string; message?: string } }; message?: string }
    errorMessage.value =
      err.response?.data?.detail || err.response?.data?.message || err.message || 'Connection failed'
  } finally {
    testingRedis.value = false
  }
}

function nextStep() {
  if (canProceed.value) {
    errorMessage.value = ''
    currentStep.value++
  }
}

async function performInstall() {
  installing.value = true
  errorMessage.value = ''

  try {
    await install(formData)
    installSuccess.value = true
    // Start polling for service restart
    waitForServiceRestart()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { detail?: string; message?: string } }; message?: string }
    errorMessage.value =
      err.response?.data?.detail || err.response?.data?.message || err.message || 'Installation failed'
  } finally {
    installing.value = false
  }
}

// Wait for service to restart and become available
async function waitForServiceRestart() {
  const maxAttempts = 60 // Increase to 60 attempts, ~60 seconds max
  const interval = 1000 // 1 second between attempts

  // Wait a moment for the service to start restarting
  await new Promise((resolve) => setTimeout(resolve, 3000))

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      // Use setup status endpoint as it tells us the real mode
      // Service might return 404 or connection refused while restarting
      const response = await fetch(buildGatewayUrl('/setup/status'), {
        method: 'GET',
        cache: 'no-store'
      })

      if (response.ok) {
        const data = await response.json()
        // If needs_setup is false, service has restarted in normal mode
        if (data.data && !data.data.needs_setup) {
          serviceReady.value = true
          // Redirect to login page after a short delay
          setTimeout(() => {
            window.location.href = '/login'
          }, 1500)
          return
        }
      }
    } catch {
      // Service not ready or network error during restart, continue polling
    }

    await new Promise((resolve) => setTimeout(resolve, interval))
  }

  // If we reach here, service didn't restart in time
  // Show a message to refresh manually
  errorMessage.value = t('setup.status.timeout')
}
</script>

<style scoped>
.setup-gateway-shell {
  background:
    radial-gradient(circle at 14% 0%, rgba(77, 212, 230, 0.13), transparent 32rem),
    radial-gradient(circle at 88% 6%, rgba(212, 168, 92, 0.10), transparent 28rem),
    linear-gradient(180deg, #070a10 0%, #090d13 52%, #070a10 100%);
}

.setup-gateway-shell::before {
  pointer-events: none;
  position: fixed;
  inset: 0;
  content: '';
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.024) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.024) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.72), transparent 72%);
}

.setup-gateway-hero {
  position: relative;
  display: grid;
  gap: 1rem;
  border: 1px solid rgba(255, 255, 255, 0.10);
  border-radius: 0.5rem;
  background: rgba(255, 255, 255, 0.04);
  padding: 1.25rem;
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.24);
}

.setup-gateway-kicker {
  display: inline-flex;
  border: 1px solid rgba(77, 212, 230, 0.26);
  border-radius: 999px;
  background: rgba(77, 212, 230, 0.09);
  padding: 0.35rem 0.75rem;
  color: rgba(77, 212, 230, 0.95);
  font-size: 0.75rem;
  font-weight: 700;
}

.setup-gateway-status {
  border: 1px solid rgba(255, 255, 255, 0.10);
  border-radius: 0.5rem;
  background: rgba(255, 255, 255, 0.04);
  padding: 1rem;
}

.setup-gateway-panel {
  position: relative;
}

.setup-gateway-shell :deep(.input),
.setup-gateway-shell :deep(.select-trigger) {
  min-height: 2.75rem;
  border-color: rgba(255, 255, 255, 0.10);
  background: rgba(255, 255, 255, 0.045);
  color: rgba(255, 255, 255, 0.94);
}

.setup-gateway-shell :deep(.input::placeholder) {
  color: rgba(255, 255, 255, 0.34);
}

.setup-gateway-shell :deep(.input:focus) {
  border-color: rgba(77, 212, 230, 0.72);
  box-shadow: 0 0 0 3px rgba(77, 212, 230, 0.13);
}

.setup-gateway-shell :deep(.input-label) {
  color: rgba(255, 255, 255, 0.70);
}

.setup-gateway-shell :deep(.input-error-text) {
  color: #e8a87c;
}

.setup-gateway-shell :deep(.btn) {
  border-radius: 0.5rem;
}

.setup-gateway-shell :deep(.btn-primary) {
  border-color: rgba(77, 212, 230, 0.34);
  background: linear-gradient(135deg, #4dd4e6, #78e6f4);
  color: #071014;
  box-shadow: 0 14px 34px rgba(77, 212, 230, 0.18);
}

.setup-gateway-shell :deep(.btn-secondary) {
  border-color: rgba(255, 255, 255, 0.10);
  background: rgba(255, 255, 255, 0.045);
  color: rgba(255, 255, 255, 0.88);
}

@media (min-width: 768px) {
  .setup-gateway-hero {
    grid-template-columns: minmax(0, 1fr) 14rem;
    align-items: start;
  }
}
</style>
