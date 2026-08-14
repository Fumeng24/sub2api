import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import ticketsAPI from '@/custom/api/tickets'
import adminTicketsAPI from '@/custom/api/admin/tickets'
import type { TicketUnreadSummary } from '@/types'

const emptySummary = (): TicketUnreadSummary => ({
  total: 0,
  open: 0,
  pending: 0,
  resolved: 0,
  closed: 0
})

const THROTTLE_MS = 60 * 1000

export const useTicketStore = defineStore('tickets', () => {
  const userSummary = ref<TicketUnreadSummary>(emptySummary())
  const adminSummary = ref<TicketUnreadSummary>(emptySummary())
  const loading = ref(false)
  const lastFetchTime = ref<Record<'admin' | 'user', number>>({ admin: 0, user: 0 })

  const userUnreadCount = computed(() => userSummary.value.total || 0)
  const adminUnreadCount = computed(() => adminSummary.value.total || 0)

  async function fetchUnreadSummary(role: 'admin' | 'support' | 'user' = 'user', force = false) {
    const storeRole = role === 'support' ? 'admin' : role
    const now = Date.now()
    if (!force && lastFetchTime.value[storeRole] > 0 && now - lastFetchTime.value[storeRole] < THROTTLE_MS) {
      return
    }
    lastFetchTime.value[storeRole] = now
    loading.value = true
    try {
      if (storeRole === 'admin') {
        adminSummary.value = await adminTicketsAPI.getUnreadSummary()
      } else {
        userSummary.value = await ticketsAPI.getUnreadSummary()
      }
    } catch (err) {
      lastFetchTime.value[storeRole] = 0
      console.error('Failed to fetch ticket unread summary:', err)
    } finally {
      loading.value = false
    }
  }

  function reset() {
    userSummary.value = emptySummary()
    adminSummary.value = emptySummary()
    lastFetchTime.value = { admin: 0, user: 0 }
    loading.value = false
  }

  return {
    userSummary,
    adminSummary,
    loading,
    userUnreadCount,
    adminUnreadCount,
    fetchUnreadSummary,
    reset
  }
})
