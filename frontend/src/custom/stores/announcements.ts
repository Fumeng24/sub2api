import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { announcementsAPI } from '@/api'
import type { UserAnnouncement } from '@/types'

const THROTTLE_MS = 60 * 1000

export const useAnnouncementStore = defineStore('announcements', () => {
  const announcements = ref<UserAnnouncement[]>([])
  const loading = ref(false)
  const lastFetchTime = ref(0)
  const popupQueue = ref<UserAnnouncement[]>([])
  const currentPopup = ref<UserAnnouncement | null>(null)

  let shownPopupIds = new Set<number>()

  const unreadCount = computed(() =>
    announcements.value.filter((announcement) => !announcement.read_at).length,
  )

  async function fetchAnnouncements(force = false): Promise<boolean> {
    if (loading.value) return true

    const now = Date.now()
    if (!force && lastFetchTime.value > 0 && now - lastFetchTime.value < THROTTLE_MS) {
      return true
    }

    lastFetchTime.value = now
    try {
      loading.value = true
      announcements.value = await announcementsAPI.list(false)
      enqueueNewPopups()
      return true
    } catch (error) {
      lastFetchTime.value = 0
      console.error('Failed to fetch announcements:', error)
      return false
    } finally {
      loading.value = false
    }
  }

  function enqueueNewPopups() {
    const newPopups = announcements.value.filter(
      (announcement) =>
        announcement.notify_mode === 'popup' &&
        !announcement.read_at &&
        !shownPopupIds.has(announcement.id),
    )

    for (const popup of newPopups) {
      if (!popupQueue.value.some((queued) => queued.id === popup.id)) {
        popupQueue.value.push(popup)
      }
    }

    if (!currentPopup.value) {
      showNextPopup()
    }
  }

  function showNextPopup() {
    const next = popupQueue.value.shift() ?? null
    currentPopup.value = next
    if (next) {
      shownPopupIds.add(next.id)
    }
  }

  async function dismissPopup() {
    if (!currentPopup.value) return

    const id = currentPopup.value.id
    currentPopup.value = null
    await markAsRead(id)

    if (popupQueue.value.length > 0) {
      window.setTimeout(showNextPopup, 300)
    }
  }

  async function markAsRead(id: number): Promise<boolean> {
    try {
      await announcementsAPI.markRead(id)
      const announcement = announcements.value.find((item) => item.id === id)
      if (announcement && !announcement.read_at) {
        announcement.read_at = new Date().toISOString()
      }
      return true
    } catch (error) {
      console.error('Failed to mark announcement as read:', error)
      return false
    }
  }

  async function markAllAsRead() {
    const unread = announcements.value.filter((announcement) => !announcement.read_at)
    if (unread.length === 0) return

    try {
      loading.value = true
      for (let offset = 0; offset < unread.length; offset += 8) {
        const batch = unread.slice(offset, offset + 8)
        await Promise.all(batch.map((announcement) => announcementsAPI.markRead(announcement.id)))
      }
      const readAt = new Date().toISOString()
      for (const announcement of announcements.value) {
        if (!announcement.read_at) {
          announcement.read_at = readAt
        }
      }
    } finally {
      loading.value = false
    }
  }

  function reset() {
    announcements.value = []
    loading.value = false
    lastFetchTime.value = 0
    popupQueue.value = []
    currentPopup.value = null
    shownPopupIds = new Set<number>()
  }

  return {
    announcements,
    loading,
    currentPopup,
    unreadCount,
    fetchAnnouncements,
    dismissPopup,
    markAsRead,
    markAllAsRead,
    reset,
  }
})
