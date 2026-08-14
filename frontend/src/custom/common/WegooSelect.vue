<template>
  <div class="relative" ref="containerRef">
    <button
      ref="triggerRef"
      type="button"
      @click="toggle"
      :disabled="disabled"
      :aria-expanded="isOpen"
      :aria-haspopup="true"
      aria-label="Select option"
      :class="[
        'select-trigger',
        isOpen && 'select-trigger-open',
        error && 'select-trigger-error',
        disabled && 'select-trigger-disabled'
      ]"
      @keydown.down.prevent="onTriggerKeyDown"
      @keydown.up.prevent="onTriggerKeyDown"
    >
      <span class="select-value">
        <slot name="selected" :option="selectedOption">
          {{ selectedLabel }}
        </slot>
      </span>
      <span
        v-if="clearable && hasValue && !disabled"
        class="select-clear"
        role="button"
        tabindex="-1"
        aria-label="Clear selection"
        @click.stop="clearSelection"
        @mousedown.stop
        @keydown.enter.stop.prevent="clearSelection"
      >
        <Icon name="x" size="sm" />
      </span>
      <span class="select-icon">
        <Icon
          name="chevronDown"
          size="md"
          :class="['transition-transform duration-200', isOpen && 'rotate-180']"
        />
      </span>
    </button>

    <!-- Teleport dropdown to body to escape stacking context -->
    <Teleport to="body">
      <Transition name="select-dropdown">
        <div
          v-if="isOpen"
          ref="dropdownRef"
          class="select-dropdown-portal"
          :class="[instanceId]"
          :style="dropdownStyle"
          role="listbox"
          @click.stop
          @mousedown.stop
          @keydown="onDropdownKeyDown"
        >
          <!-- Search input -->
          <div v-if="isSearchable" class="select-search">
            <Icon name="search" size="sm" class="text-[color:var(--apple-muted-2)]" />
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              :placeholder="searchPlaceholderText"
              class="select-search-input"
              @click.stop
            />
          </div>

          <!-- Horizontal group tabs -->
          <div v-if="hasGroupTabs" class="select-group-tabs">
            <button
              v-for="group in groupTabOptions"
              :key="String(getGroupKey(group))"
              type="button"
              class="select-group-tab"
              :class="[
                `select-group-tab-${String(getGroupKey(group))}`,
                activeGroupKey === getGroupKey(group) && 'select-group-tab-active'
              ]"
              @click.stop="activeGroupKey = getGroupKey(group)"
            >
              {{ getOptionLabel(group) }}
            </button>
          </div>

          <!-- Options list -->
          <div class="select-options" ref="optionsListRef">
            <div
              v-for="(option, index) in filteredOptions"
              :key="`${typeof getOptionValue(option)}:${String(getOptionValue(option) ?? '')}`"
              role="option"
              :aria-selected="isSelected(option)"
              :aria-disabled="isOptionDisabled(option)"
              @click.stop="!isOptionDisabled(option) && selectOption(option)"
              @mouseenter="handleOptionMouseEnter(option, index)"
              :class="[
                'select-option',
                isGroupHeaderOption(option) && 'select-option-group',
                isSelected(option) && 'select-option-selected',
                isOptionDisabled(option) && !isGroupHeaderOption(option) && 'select-option-disabled',
                focusedIndex === index && !isGroupHeaderOption(option) && 'select-option-focused'
              ]"
            >
              <slot name="option" :option="option" :selected="isSelected(option)">
                <Icon
                  v-if="option._creatable"
                  name="search"
                  size="sm"
                  class="flex-shrink-0 text-[color:var(--apple-muted-2)]"
                />
                <span class="select-option-label" :class="option._creatable && 'italic text-[color:var(--apple-muted)]'">{{ getOptionLabel(option) }}</span>
                <Icon
                  v-if="isSelected(option)"
                  name="check"
                  size="sm"
                  class="text-[color:var(--apple-blue)]"
                  :stroke-width="2"
                />
              </slot>
            </div>

            <!-- Empty state -->
            <div v-if="filteredOptions.length === 0" class="select-empty">
              {{ emptyTextDisplay }}
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

// Instance ID for unique click-outside detection
const instanceId = `select-${Math.random().toString(36).substring(2, 9)}`

export interface SelectOption {
  value: string | number | boolean | null
  label: string
  disabled?: boolean
  [key: string]: unknown
}

interface Props {
  modelValue: string | number | boolean | null | undefined
  options: SelectOption[] | Array<Record<string, unknown>>
  placeholder?: string
  disabled?: boolean
  error?: boolean
  searchable?: boolean | 'auto'
  searchPlaceholder?: string
  emptyText?: string
  valueKey?: string
  labelKey?: string
  creatable?: boolean
  creatablePrefix?: string
  clearable?: boolean
  groupTabs?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: string | number | boolean | null): void
  (e: 'change', value: string | number | boolean | null, option: SelectOption | null): void
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  error: false,
  searchable: 'auto',
  creatable: false,
  creatablePrefix: '',
  clearable: false,
  groupTabs: false,
  valueKey: 'value',
  labelKey: 'label'
})

const emit = defineEmits<Emits>()

const isOpen = ref(false)
const searchQuery = ref('')
const focusedIndex = ref(-1)
const containerRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const optionsListRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<'bottom' | 'top'>('bottom')
const triggerRect = ref<DOMRect | null>(null)
const activeGroupKey = ref<string | number | boolean | null>(null)
const viewportPadding = 8

// i18n placeholders
const placeholderText = computed(() => props.placeholder ?? t('common.selectOption'))
const searchPlaceholderText = computed(() => props.searchPlaceholder ?? t('common.searchPlaceholder'))
const emptyTextDisplay = computed(() => props.emptyText ?? t('common.noOptionsFound'))

const isSearchable = computed(() => {
  if (props.searchable === 'auto') return props.options.length > 5
  return props.searchable
})

// Computed style for teleported dropdown
const dropdownStyle = computed(() => {
  if (!triggerRect.value) return {}

  const rect = triggerRect.value
  const viewportWidth = window.innerWidth
  const maxDropdownWidth = Math.max(0, viewportWidth - viewportPadding * 2)
  const desiredWidth = Math.min(Math.max(rect.width, 200), maxDropdownWidth)
  const maxLeft = Math.max(viewportPadding, viewportWidth - desiredWidth - viewportPadding)
  const clampedLeft = Math.min(Math.max(rect.left, viewportPadding), maxLeft)
  const availableHeight = dropdownPosition.value === 'top'
    ? rect.top - viewportPadding - 4
    : window.innerHeight - rect.bottom - viewportPadding - 4
  const style: Record<string, string> = {
    position: 'fixed',
    left: `${clampedLeft}px`,
    minWidth: `${desiredWidth}px`,
    maxWidth: `${maxDropdownWidth}px`,
    width: `${desiredWidth}px`,
    zIndex: '100000020',
    maxHeight: `${Math.max(72, Math.min(320, availableHeight))}px`
  }

  if (dropdownPosition.value === 'top') {
    style.bottom = `${window.innerHeight - rect.top + 4}px`
  } else {
    style.top = `${rect.bottom + 4}px`
  }

  return style
})

const getOptionValue = (option: any): any => {
  if (typeof option === 'object' && option !== null) {
    return option[props.valueKey]
  }
  return option
}

const getOptionLabel = (option: any): string => {
  if (typeof option === 'object' && option !== null) {
    return String(option[props.labelKey] ?? '')
  }
  return String(option ?? '')
}

const isGroupHeaderOption = (option: any): boolean => {
  if (typeof option === 'object' && option !== null) {
    return option.kind === 'group'
  }
  return false
}

const getGroupKey = (option: any): string | number | boolean | null => {
  if (typeof option === 'object' && option !== null) {
    return (option.groupKey ?? option[props.valueKey] ?? null) as string | number | boolean | null
  }
  return null
}

const isOptionDisabled = (option: any): boolean => {
  if (typeof option === 'object' && option !== null) {
    return !!option.disabled || isGroupHeaderOption(option)
  }
  return false
}

const selectedOption = computed(() => {
  return props.options.find((opt) => getOptionValue(opt) === props.modelValue) || null
})

const groupTabOptions = computed(() => (props.options as any[]).filter(isGroupHeaderOption))
const hasGroupTabs = computed(() => props.groupTabs && groupTabOptions.value.length > 0)

const selectedGroupKey = computed(() => {
  if (!selectedOption.value) return null
  return getGroupKey(selectedOption.value)
})

const ensureActiveGroupKey = () => {
  if (!hasGroupTabs.value) {
    activeGroupKey.value = null
    return
  }
  const available = groupTabOptions.value.map(getGroupKey)
  const selectedKey = selectedGroupKey.value
  if (selectedKey !== null && available.includes(selectedKey)) {
    activeGroupKey.value = selectedKey
    return
  }
  if (!available.includes(activeGroupKey.value)) {
    activeGroupKey.value = available[0] ?? null
  }
}

const selectedLabel = computed(() => {
  if (selectedOption.value) {
    return getOptionLabel(selectedOption.value)
  }
  // In creatable mode, show the raw value if no matching option
  if (props.creatable && props.modelValue) {
    return String(props.modelValue)
  }
  return placeholderText.value
})

const hasValue = computed(
  () => props.modelValue !== null && props.modelValue !== undefined && props.modelValue !== ''
)

const filteredOptions = computed(() => {
  let opts = props.options as any[]
  if (hasGroupTabs.value) {
    opts = opts.filter((opt) => {
      if (isGroupHeaderOption(opt)) return false
      return activeGroupKey.value === null || getGroupKey(opt) === activeGroupKey.value
    })
  }
  if (isSearchable.value && searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    const optionMatches = (opt: any) => {
      // Match label
      if (getOptionLabel(opt).toLowerCase().includes(query)) return true
      // Also match description if present
      if (opt.description && String(opt.description).toLowerCase().includes(query)) return true
      return false
    }

    if (!hasGroupTabs.value && opts.some(isGroupHeaderOption)) {
      const grouped: any[] = []
      let currentGroup: any | null = null
      let currentItems: any[] = []
      let currentGroupMatches = false

      const flushGroup = () => {
        if (currentItems.length > 0) {
          if (currentGroup) grouped.push(currentGroup)
          grouped.push(...currentItems)
        }
      }

      for (const opt of opts) {
        if (isGroupHeaderOption(opt)) {
          flushGroup()
          currentGroup = opt
          currentItems = []
          currentGroupMatches = optionMatches(opt)
          continue
        }
        if (currentGroupMatches || optionMatches(opt)) {
          currentItems.push(opt)
        }
      }
      flushGroup()
      opts = grouped
    } else {
      opts = opts.filter(optionMatches)
    }
    // In creatable mode, always prepend a fuzzy search option
    if (props.creatable && searchQuery.value.trim()) {
      const trimmed = searchQuery.value.trim()
      const prefix = props.creatablePrefix || t('common.search')
      opts = [{ [props.valueKey]: trimmed, [props.labelKey]: `${prefix} "${trimmed}"`, _creatable: true }, ...opts]
    }
  }
  return opts
})

const isSelected = (option: any): boolean => {
  return getOptionValue(option) === props.modelValue
}

const findNextEnabledIndex = (startIndex: number): number => {
  const opts = filteredOptions.value
  if (opts.length === 0) return -1
  for (let offset = 0; offset < opts.length; offset++) {
    const idx = (startIndex + offset) % opts.length
    if (!isOptionDisabled(opts[idx])) return idx
  }
  return -1
}

const findPrevEnabledIndex = (startIndex: number): number => {
  const opts = filteredOptions.value
  if (opts.length === 0) return -1
  for (let offset = 0; offset < opts.length; offset++) {
    const idx = (startIndex - offset + opts.length) % opts.length
    if (!isOptionDisabled(opts[idx])) return idx
  }
  return -1
}

const handleOptionMouseEnter = (option: any, index: number) => {
  if (isOptionDisabled(option) || isGroupHeaderOption(option)) return
  focusedIndex.value = index
}

// Update trigger rect periodically while open to follow scroll/resize
const updateTriggerRect = () => {
  if (containerRef.value) {
    triggerRect.value = containerRef.value.getBoundingClientRect()
  }
}

const calculateDropdownPosition = () => {
  if (!containerRef.value) return
  updateTriggerRect()

  nextTick(() => {
    if (!dropdownRef.value || !triggerRect.value) return
    const dropdownHeight = Math.min(dropdownRef.value.scrollHeight || 240, 320)
    const spaceBelow = window.innerHeight - triggerRect.value.bottom
    const spaceAbove = triggerRect.value.top

    if (spaceBelow < dropdownHeight && spaceAbove > spaceBelow) {
      dropdownPosition.value = 'top'
    } else {
      dropdownPosition.value = 'bottom'
    }
  })
}

const toggle = () => {
  if (props.disabled) return
  isOpen.value = !isOpen.value
}

watch(isOpen, (open) => {
  if (open) {
    ensureActiveGroupKey()
    calculateDropdownPosition()
    // Reset focused index to current selection or first item
    if (filteredOptions.value.length === 0) {
      focusedIndex.value = -1
    } else {
      const selectedIdx = filteredOptions.value.findIndex(isSelected)
      const initialIdx = selectedIdx >= 0 ? selectedIdx : 0
      focusedIndex.value = isOptionDisabled(filteredOptions.value[initialIdx])
        ? findNextEnabledIndex(initialIdx + 1)
        : initialIdx
    }

    if (isSearchable.value && !window.matchMedia?.('(pointer: coarse)').matches) {
      nextTick(() => searchInputRef.value?.focus())
    }
    // Add scroll listener to update position
    window.addEventListener('scroll', updateTriggerRect, { capture: true, passive: true })
    window.addEventListener('resize', calculateDropdownPosition)
  } else {
    searchQuery.value = ''
    focusedIndex.value = -1
    window.removeEventListener('scroll', updateTriggerRect, { capture: true })
    window.removeEventListener('resize', calculateDropdownPosition)
  }
})

watch(
  () => [props.options, props.modelValue] as const,
  () => ensureActiveGroupKey(),
  { deep: true }
)

const selectOption = (option: any) => {
  const value = getOptionValue(option) ?? null
  emit('update:modelValue', value)
  emit('change', value, option)
  isOpen.value = false
  triggerRef.value?.focus()
}

const clearSelection = () => {
  if (props.disabled) return
  emit('update:modelValue', null)
  emit('change', null, null)
}

// Keyboards
const onTriggerKeyDown = () => {
  if (!isOpen.value) {
    isOpen.value = true
  }
}

const onDropdownKeyDown = (e: KeyboardEvent) => {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      focusedIndex.value = findNextEnabledIndex(focusedIndex.value + 1)
      if (focusedIndex.value >= 0) scrollToFocused()
      break
    case 'ArrowUp':
      e.preventDefault()
      focusedIndex.value = findPrevEnabledIndex(focusedIndex.value - 1)
      if (focusedIndex.value >= 0) scrollToFocused()
      break
    case 'Enter':
      e.preventDefault()
      if (focusedIndex.value >= 0 && focusedIndex.value < filteredOptions.value.length) {
        const opt = filteredOptions.value[focusedIndex.value]
        if (!isOptionDisabled(opt)) selectOption(opt)
      }
      break
    case 'Escape':
      e.preventDefault()
      isOpen.value = false
      triggerRef.value?.focus()
      break
    case 'Tab':
      isOpen.value = false
      break
  }
}

const scrollToFocused = () => {
  nextTick(() => {
    const list = optionsListRef.value
    if (!list) return
    const focusedEl = list.children[focusedIndex.value] as HTMLElement
    if (!focusedEl) return

    if (focusedEl.offsetTop < list.scrollTop) {
      list.scrollTop = focusedEl.offsetTop
    } else if (focusedEl.offsetTop + focusedEl.offsetHeight > list.scrollTop + list.offsetHeight) {
      list.scrollTop = focusedEl.offsetTop + focusedEl.offsetHeight - list.offsetHeight
    }
  })
}

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside THIS specific instance's dropdown or trigger
  const isInDropdown = !!target.closest(`.${instanceId}`)
  const isInTrigger = containerRef.value?.contains(target)

  if (!isInDropdown && !isInTrigger && isOpen.value) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', updateTriggerRect, { capture: true })
  window.removeEventListener('resize', calculateDropdownPosition)
})
</script>

<style scoped>
.select-trigger {
  @apply flex w-full min-w-0 items-center justify-between gap-2;
  @apply rounded-lg px-3 py-2 text-sm;
  @apply bg-white dark:bg-dark-800;
  @apply border border-gray-200 dark:border-dark-600;
  @apply text-gray-900 dark:text-gray-100;
  @apply transition-colors duration-150;
  @apply focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30;
  @apply hover:border-gray-300 dark:hover:border-dark-500;
  @apply cursor-pointer;
  background: var(--apple-surface);
  border-color: var(--apple-border);
  border-radius: var(--apple-radius);
  color: var(--apple-text);
  box-shadow: none;
}

.select-trigger-open {
  @apply ring-2;
  --tw-ring-color: var(--apple-focus-ring);
  border-color: color-mix(in srgb, var(--apple-blue) 60%, var(--apple-border));
}

.select-trigger-error {
  @apply border-red-500 focus:border-red-500 focus:ring-red-500/30;
}

.select-trigger-disabled {
  @apply cursor-not-allowed opacity-55;
}

.select-value {
  @apply min-w-0 flex-1 truncate text-left;
}

.select-icon {
  @apply flex-shrink-0 text-gray-400 dark:text-dark-400;
}

.select-clear {
  @apply flex flex-shrink-0 cursor-pointer items-center justify-center;
  @apply rounded-lg text-[color:var(--apple-muted-2)] transition-colors;
  @apply hover:bg-[color:var(--apple-hover)] hover:text-[color:var(--apple-text)];
}
</style>

<style>
.select-dropdown-portal {
  @apply min-w-[200px];
  @apply bg-white dark:bg-dark-800;
  @apply rounded-lg;
  @apply border border-gray-200 dark:border-dark-700;
  @apply shadow-sm;
  @apply overflow-hidden;
  background: var(--apple-surface);
  border-color: var(--apple-border);
  border-radius: var(--apple-radius);
  box-shadow: var(--apple-shadow-sm);
  display: flex;
  min-height: 0;
  flex-direction: column;
  overscroll-behavior: contain;
  pointer-events: auto !important;
}

.select-dropdown-portal .select-search {
  @apply flex items-center gap-2 px-3 py-2;
  @apply border-b border-gray-100 dark:border-dark-700;
  border-color: var(--apple-border-soft);
  flex-shrink: 0;
}

.select-dropdown-portal .select-search-input {
  @apply min-w-0 flex-1 bg-transparent text-sm;
  @apply text-gray-900 dark:text-gray-100;
  @apply placeholder:text-gray-400 dark:placeholder:text-dark-400;
  @apply focus:outline-none;
  color: var(--apple-text);
}

.select-dropdown-portal .select-group-tabs {
  @apply grid grid-cols-3 gap-1.5 overflow-x-auto border-b border-gray-100 bg-gray-50 p-2 dark:border-dark-700 dark:bg-dark-900;
  background: var(--apple-surface-elevated);
  border-color: var(--apple-border-soft);
  flex-shrink: 0;
}

.select-dropdown-portal .select-group-tab {
  @apply relative inline-flex min-h-9 min-w-0 items-center justify-center rounded-lg border px-2.5 py-1.5 text-center text-sm font-medium;
  @apply transition-colors duration-150;
  @apply hover:bg-white dark:hover:bg-dark-800;
  background: transparent;
  border-color: transparent;
  color: var(--apple-muted);
}

@media (max-width: 480px) {
  .select-dropdown-portal .select-group-tabs {
    @apply flex;
  }

  .select-dropdown-portal .select-group-tab {
    @apply min-w-[7rem] flex-shrink-0;
  }
}

.select-dropdown-portal .select-group-tab-active {
  background: var(--apple-surface);
  border-color: var(--apple-border);
  color: var(--apple-blue);
  box-shadow: var(--apple-shadow-sm);
}

.select-dropdown-portal .select-group-tab::before {
  content: '';
  @apply mr-1.5 inline-block h-2 w-2 rounded-full align-middle;
}

.select-dropdown-portal .select-group-tab-openai {
  @apply text-emerald-700 dark:text-emerald-300;
}

.select-dropdown-portal .select-group-tab-openai::before {
  @apply bg-emerald-500;
}

.select-dropdown-portal .select-group-tab-openai.select-group-tab-active {
  color: var(--apple-blue);
}

.select-dropdown-portal .select-group-tab-anthropic {
  @apply text-orange-700 dark:text-orange-300;
}

.select-dropdown-portal .select-group-tab-anthropic::before {
  @apply bg-orange-500;
}

.select-dropdown-portal .select-group-tab-anthropic.select-group-tab-active {
  color: var(--apple-blue);
}

.select-dropdown-portal .select-group-tab-other {
  @apply text-sky-700 dark:text-sky-300;
}

.select-dropdown-portal .select-group-tab-other::before {
  @apply bg-sky-500;
}

.select-dropdown-portal .select-group-tab-other.select-group-tab-active {
  color: var(--apple-blue);
}

.select-dropdown-portal .select-options {
  @apply overflow-y-auto py-1 outline-none;
  -webkit-overflow-scrolling: touch;
  min-height: 0;
  flex: 1 1 auto;
  overscroll-behavior: contain;
  touch-action: pan-y;
}

.select-dropdown-portal .select-option {
  @apply flex items-center justify-between gap-2;
  @apply min-w-0 px-3 py-2 text-sm;
  @apply text-gray-700 dark:text-gray-300;
  @apply cursor-pointer transition-colors duration-150;
  @apply hover:bg-gray-50 dark:hover:bg-dark-700;
  color: var(--apple-text);
  pointer-events: auto !important;
}

.select-dropdown-portal .select-option-selected {
  @apply bg-primary-50 dark:bg-primary-900/20;
  @apply text-primary-700 dark:text-primary-300;
  background: color-mix(in srgb, var(--apple-blue) 10%, var(--apple-surface));
  color: var(--apple-blue);
}

.select-dropdown-portal .select-option-focused {
  @apply bg-gray-100 dark:bg-dark-700;
  background: var(--apple-hover);
}

.select-dropdown-portal .select-option-disabled {
  @apply cursor-not-allowed opacity-40;
}

.select-dropdown-portal .select-option-group {
  @apply cursor-default select-none;
  @apply bg-gray-50 dark:bg-dark-900;
  @apply text-[11px] font-semibold uppercase;
  @apply text-gray-500 dark:text-gray-400;
  background: var(--apple-surface-elevated);
  color: var(--apple-muted);
  letter-spacing: 0;
}

.select-dropdown-portal .select-option-group:hover {
  @apply bg-gray-50 dark:bg-dark-900;
  background: var(--apple-surface-elevated);
}

.select-dropdown-portal .select-option-label {
  @apply flex-1 min-w-0 truncate text-left;
}

.select-dropdown-portal .select-empty {
  @apply px-4 py-8 text-center text-sm;
  @apply text-gray-500 dark:text-dark-400;
  color: var(--apple-muted);
}

.select-dropdown-enter-active,
.select-dropdown-leave-active {
  transition: all 0.2s ease;
}

.select-dropdown-enter-from,
.select-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
