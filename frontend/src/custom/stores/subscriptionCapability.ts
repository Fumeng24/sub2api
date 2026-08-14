import { defineStore } from 'pinia'
import { ref } from 'vue'

import userGroupsAPI from '@/custom/api/groups'

const POLL_INTERVAL_MS = 5 * 60 * 1000

export const useSubscriptionCapabilityStore = defineStore('subscriptionCapability', () => {
  const hasSubscriptionGroups = ref(false)
  const capabilityLoaded = ref(false)

  let requestGeneration = 0
  let activePromise: Promise<boolean> | null = null
  let pollerInterval: ReturnType<typeof setInterval> | null = null

  async function fetchSubscriptionCapability(force = false): Promise<boolean> {
    if (!force && capabilityLoaded.value) {
      return hasSubscriptionGroups.value
    }

    if (activePromise && !force) {
      return activePromise
    }

    const currentGeneration = ++requestGeneration
    const requestPromise = userGroupsAPI
      .getSubscriptionCapability()
      .then((capability) => {
        const available = Boolean(capability?.has_subscription_groups)
        if (currentGeneration === requestGeneration) {
          hasSubscriptionGroups.value = available
          capabilityLoaded.value = true
        }
        return available
      })
      .catch((error) => {
        if (currentGeneration === requestGeneration) {
          hasSubscriptionGroups.value = false
          capabilityLoaded.value = true
        }
        console.error('Failed to fetch subscription capability:', error)
        throw error
      })
      .finally(() => {
        if (activePromise === requestPromise) {
          activePromise = null
        }
      })

    activePromise = requestPromise
    return requestPromise
  }

  function startPolling() {
    if (pollerInterval) return

    pollerInterval = setInterval(() => {
      fetchSubscriptionCapability(true).catch((error) => {
        console.error('Subscription capability polling failed:', error)
      })
    }, POLL_INTERVAL_MS)
  }

  function stopPolling() {
    if (!pollerInterval) return

    clearInterval(pollerInterval)
    pollerInterval = null
  }

  function clear() {
    requestGeneration++
    activePromise = null
    hasSubscriptionGroups.value = false
    capabilityLoaded.value = false
    stopPolling()
  }

  return {
    hasSubscriptionGroups,
    capabilityLoaded,
    fetchSubscriptionCapability,
    startPolling,
    stopPolling,
    clear,
  }
})
