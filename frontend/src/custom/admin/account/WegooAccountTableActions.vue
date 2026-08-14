<template>
  <div class="admin-toolbar-actions">
    <slot name="before"></slot>
    <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary px-2 md:px-3">
      <Icon name="refresh" size="md" :class="[loading ? 'animate-spin' : '']" />
    </button>
    <slot name="after"></slot>
    <slot name="beforeCreate"></slot>
    <button @click="$emit('create')" class="btn btn-primary admin-toolbar-primary">{{ t('admin.accounts.createAccount') }}</button>
    <slot name="afterCreate"></slot>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps(['loading'])
defineEmits(['refresh', 'create'])

const { t } = useI18n()
</script>
