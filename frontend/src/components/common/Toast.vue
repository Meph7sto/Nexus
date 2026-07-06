<template>
 <Teleport to="body">
 <div
 class="pointer-events-none fixed right-4 top-4 z-[9999] space-y-3"
 aria-live="polite"
 aria-atomic="true"
 >
 <TransitionGroup
 enter-active-class="transition ease-out duration-300"
 enter-from-class="opacity-0 translate-x-full"
 enter-to-class="opacity-100 translate-x-0"
 leave-active-class="transition ease-in duration-200"
 leave-from-class="opacity-100 translate-x-0"
 leave-to-class="opacity-0 translate-x-full"
 >
 <div
 v-for="toast in toasts"
 :key="toast.id"
 :class="[
 'pointer-events-auto min-w-[320px] max-w-md overflow-hidden rounded-lg',
 'border border-l-4 border-[var(--nx-border)] bg-[var(--nx-surface)] shadow-[0_8px_20px_rgba(17,17,17,0.08)]',
 getBorderColor(toast.type)
 ]"
 >
 <div class="p-4">
 <div class="flex items-start gap-3">
 <!-- Icon -->
 <div class="mt-0.5 flex-shrink-0">
 <Icon
 :name="getToastIconName(toast.type)"
 size="md"
 :class="getIconColor(toast.type)"
 aria-hidden="true"
 />
 </div>

 <!-- Content -->
 <div class="min-w-0 flex-1">
 <p v-if="toast.title" class="text-sm font-semibold text-[var(--nx-text)]">
 {{ toast.title }}
 </p>
 <p
 :class="[
 'text-sm leading-relaxed',
 toast.title
 ? 'mt-1 text-[var(--nx-muted)]'
 : 'text-[var(--nx-text)]'
 ]"
 >
 {{ toast.message }}
 </p>
 </div>

 <!-- Close button -->
 <button
 @click="removeToast(toast.id)"
 class="-m-1 flex-shrink-0 rounded p-1 text-[var(--nx-subtle)] transition-colors hover:bg-[var(--nx-bg)] hover:text-[var(--nx-text)]"
 aria-label="Close notification"
 >
 <Icon name="x" size="sm" />
 </button>
 </div>
 </div>

 <!-- Progress bar -->
 <div v-if="toast.duration" class="h-1 bg-[var(--nx-bg)]">
 <div
 :class="['h-full toast-progress', getProgressBarColor(toast.type)]"
 :style="{ animationDuration: `${toast.duration}ms` }"
 ></div>
 </div>
 </div>
 </TransitionGroup>
 </div>
 </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const toasts = computed(() => appStore.toasts)

const getToastIconName = (type: string): 'checkCircle' | 'xCircle' | 'exclamationTriangle' | 'infoCircle' => {
 switch (type) {
 case 'success':
 return 'checkCircle'
 case 'error':
 return 'xCircle'
 case 'warning':
 return 'exclamationTriangle'
 case 'info':
 default:
 return 'infoCircle'
 }
}

const getIconColor = (type: string): string => {
 const colors: Record<string, string> = {
 success: 'text-[var(--nx-success)]',
 error: 'text-[var(--nx-danger)]',
 warning: 'text-[var(--nx-warning)]',
 info: 'text-[var(--nx-accent)]'
 }
 return colors[type] || colors.info
}

const getBorderColor = (type: string): string => {
 const colors: Record<string, string> = {
 success: 'border-l-[var(--nx-success)]',
 error: 'border-l-[var(--nx-danger)]',
 warning: 'border-l-[var(--nx-warning)]',
 info: 'border-l-[var(--nx-accent)]'
 }
 return colors[type] || colors.info
}

const getProgressBarColor = (type: string): string => {
 const colors: Record<string, string> = {
 success: 'bg-[var(--nx-success)]',
 error: 'bg-[var(--nx-danger)]',
 warning: 'bg-[var(--nx-warning)]',
 info: 'bg-[var(--nx-accent)]'
 }
 return colors[type] || colors.info
}

const removeToast = (id: string) => {
 appStore.hideToast(id)
}
</script>

<style scoped>
.toast-progress {
 width: 100%;
 animation-name: toast-progress-shrink;
 animation-timing-function: linear;
 animation-fill-mode: forwards;
}

@keyframes toast-progress-shrink {
 from {
 width: 100%;
 }
 to {
 width: 0%;
 }
}
</style>
