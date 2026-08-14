import { onScopeDispose, ref } from 'vue'

const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | null = null
let consumers = 0

function startClock() {
  if (timer) return
  timer = setInterval(() => {
    now.value = Date.now()
  }, 30_000)
}

function stopClock() {
  if (!timer || consumers > 0) return
  clearInterval(timer)
  timer = null
}

export function useMinuteNow() {
  consumers += 1
  now.value = Date.now()
  startClock()

  onScopeDispose(() => {
    consumers = Math.max(0, consumers - 1)
    stopClock()
  })

  return now
}
