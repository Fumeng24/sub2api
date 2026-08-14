export type ThemeName = 'dark'

export const THEME_CHANGED_EVENT = 'sub2api:theme-changed'

export function isDarkTheme() {
  return true
}

export function applyTheme(_theme?: ThemeName) {
  document.documentElement.classList.add('dark')
  document.documentElement.dataset.theme = 'dark'
  document.documentElement.style.colorScheme = 'dark'
  try {
    localStorage.setItem('theme', 'dark')
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
  window.dispatchEvent(new CustomEvent(THEME_CHANGED_EVENT, { detail: { theme: 'dark' satisfies ThemeName } }))
}

export function initTheme() {
  applyTheme('dark')
}
