import type { OpenAIMessagesDispatchModelConfig } from '@/types'
import {
  createDefaultMessagesDispatchFormState as createUpstreamDefaultState,
  messagesDispatchConfigToFormState as upstreamConfigToFormState,
  resetMessagesDispatchFormState as resetUpstreamFormState,
  type MessagesDispatchFormState,
} from '@/views/admin/groupsMessagesDispatch'

export type {
  MessagesDispatchFormState,
  MessagesDispatchMappingRow,
} from '@/views/admin/groupsMessagesDispatch'
export { messagesDispatchFormStateToConfig } from '@/views/admin/groupsMessagesDispatch'

const siteDefaults = {
  opus_mapped_model: 'gpt-5.5',
  sonnet_mapped_model: 'gpt-5.5',
} as const

export function createDefaultMessagesDispatchFormState(): MessagesDispatchFormState {
  return {
    ...createUpstreamDefaultState(),
    ...siteDefaults,
  }
}

export function messagesDispatchConfigToFormState(
  config?: OpenAIMessagesDispatchModelConfig | null,
): MessagesDispatchFormState {
  const state = upstreamConfigToFormState(config)
  if (!config?.opus_mapped_model?.trim()) {
    state.opus_mapped_model = siteDefaults.opus_mapped_model
  }
  if (!config?.sonnet_mapped_model?.trim()) {
    state.sonnet_mapped_model = siteDefaults.sonnet_mapped_model
  }
  return state
}

export function resetMessagesDispatchFormState(target: MessagesDispatchFormState): void {
  resetUpstreamFormState(target)
  target.opus_mapped_model = siteDefaults.opus_mapped_model
  target.sonnet_mapped_model = siteDefaults.sonnet_mapped_model
}
