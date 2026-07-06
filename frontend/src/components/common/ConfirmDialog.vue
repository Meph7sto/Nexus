<template>
 <BaseDialog :show="show" :title="title" width="narrow" @close="handleCancel">
 <div class="space-y-4">
 <p class="text-sm text-[var(--nx-muted)]">{{ message }}</p>
 <slot></slot>
 </div>

 <template #footer>
 <div class="flex justify-end space-x-3">
 <button
 @click="handleCancel"
 type="button"
 class="rounded border border-[var(--nx-border)] bg-[var(--nx-surface)] px-4 py-2 text-sm font-medium text-[var(--nx-text)] transition-colors hover:bg-[var(--nx-bg)] focus:outline-none focus:ring-2 focus:ring-[rgba(17,17,17,0.08)] focus:ring-offset-2"
 >
 {{ cancelText }}
 </button>
 <button
 @click="handleConfirm"
 type="button"
 :class="[
 'rounded px-4 py-2 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2',
 danger
 ? 'bg-red-600 hover:bg-red-700 focus:ring-red-500'
 : 'bg-[var(--nx-accent)] hover:bg-[var(--nx-accent-hover)] focus:ring-[rgba(255,86,0,0.24)]'
 ]"
 >
 {{ confirmText }}
 </button>
 </div>
 </template>
 </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from './BaseDialog.vue'

const { t } = useI18n()

interface Props {
 show: boolean
 title: string
 message: string
 confirmText?: string
 cancelText?: string
 danger?: boolean
}

interface Emits {
 (e: 'confirm'): void
 (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
 danger: false
})

const confirmText = computed(() => props.confirmText || t('common.confirm'))
const cancelText = computed(() => props.cancelText || t('common.cancel'))

const emit = defineEmits<Emits>()

const handleConfirm = () => {
 emit('confirm')
}

const handleCancel = () => {
 emit('cancel')
}
</script>
