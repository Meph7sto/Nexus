<template>
 <div>
 <button
 @click="openModal"
 class="btn btn-ghost btn-icon relative"
 :class="{ 'bg-[rgba(255,86,0,0.1)] text-[var(--nx-accent)]': unreadCount > 0 }"
 :aria-label="t('announcements.title')"
 >
 <Icon name="bell" size="md" />
 <span v-if="unreadCount > 0" class="absolute right-1 top-1 flex h-2 w-2">
 <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-500 opacity-75"></span>
 <span class="relative inline-flex h-2 w-2 rounded-full bg-red-500"></span>
 </span>
 </button>

 <!-- 公告列表 Modal -->
 <Teleport to="body">
 <Transition name="modal-fade">
 <div
 v-if="isModalOpen"
 class="modal-overlay announcement-overlay z-[100]"
 @click="closeModal"
 >
 <div
 class="modal-content w-full max-w-[620px] overflow-hidden"
 @click.stop
 >
 <div class="modal-header items-start">
 <div class="flex w-full items-start justify-between gap-4">
 <div>
 <div class="flex items-center gap-2">
 <div class="flex h-8 w-8 items-center justify-center rounded bg-[var(--nx-accent)] text-white">
 <Icon name="bell" size="sm" />
 </div>
 <h2 class="text-lg font-semibold text-[var(--nx-text)]">
 {{ t('announcements.title') }}
 </h2>
 </div>
 <p v-if="unreadCount > 0" class="mt-2 text-sm text-[var(--nx-muted)]">
 <span class="font-medium text-[var(--nx-accent)]">{{ unreadCount }}</span>
 {{ t('announcements.unread') }}
 </p>
 </div>
 <div class="flex items-center gap-2">
 <button
 v-if="unreadCount > 0"
 @click="markAllAsRead"
 :disabled="loading"
 class="btn btn-primary btn-sm"
 >
 {{ t('announcements.markAllRead') }}
 </button>
 <button
 @click="closeModal"
 class="btn btn-ghost btn-icon"
 :aria-label="t('common.close')"
 >
 <Icon name="x" size="sm" />
 </button>
 </div>
 </div>
 </div>

 <div class="max-h-[65vh] overflow-y-auto">
 <div v-if="loading" class="flex items-center justify-center py-16">
 <div class="spinner text-[var(--nx-accent)]"></div>
 </div>

 <div v-else-if="announcements.length > 0">
 <div
 v-for="item in announcements"
 :key="item.id"
 class="group relative flex min-h-[72px] cursor-pointer items-center gap-4 border-b border-[var(--nx-border)] px-6 py-4 transition-colors hover:bg-[var(--nx-bg)]"
 :class="{ 'bg-[rgba(255,86,0,0.06)]': !item.read_at }"
 style="min-height: 72px"
 @click="openDetail(item)"
 >
 <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center">
 <div
 v-if="!item.read_at"
 class="relative flex h-10 w-10 items-center justify-center rounded bg-[var(--nx-accent)] text-white"
 >
 <Icon name="infoCircle" size="md" :stroke-width="2" />
 </div>
 <div
 v-else
 class="flex h-10 w-10 items-center justify-center rounded bg-[var(--nx-bg)] text-[var(--nx-subtle)]"
 >
 <Icon name="checkCircle" size="md" :stroke-width="2" />
 </div>
 </div>

 <div class="flex min-w-0 flex-1 items-center justify-between gap-4">
 <div class="min-w-0 flex-1">
 <h3 class="truncate text-sm font-medium text-[var(--nx-text)]">
 {{ item.title }}
 </h3>
 <div class="mt-1 flex items-center gap-2">
 <time class="text-xs text-[var(--nx-subtle)]">
 {{ formatRelativeTime(item.created_at) }}
 </time>
 <span
 v-if="!item.read_at"
 class="badge badge-primary"
 >
 <span class="relative flex h-1.5 w-1.5">
 <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-[var(--nx-accent)] opacity-75"></span>
 <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-[var(--nx-accent)]"></span>
 </span>
 {{ t('announcements.unread') }}
 </span>
 </div>
 </div>

 <div class="flex-shrink-0">
 <Icon name="chevronRight" size="md" class="text-[var(--nx-subtle)] transition-transform group-hover:translate-x-1" />
 </div>
 </div>

 <div
 v-if="!item.read_at"
 class="absolute left-0 top-0 h-full w-1 bg-[var(--nx-accent)]"
 ></div>
 </div>
 </div>

 <div v-else class="empty-state py-16">
 <div class="relative mb-4">
 <div class="flex h-20 w-20 items-center justify-center rounded bg-[var(--nx-bg)]">
 <Icon name="inbox" size="xl" class="text-[var(--nx-subtle)]" />
 </div>
 <div class="absolute -right-1 -top-1 flex h-6 w-6 items-center justify-center rounded-full bg-green-500 text-white">
 <Icon name="check" size="xs" :stroke-width="2.5" />
 </div>
 </div>
 <p class="empty-state-title text-sm">{{ t('announcements.empty') }}</p>
 <p class="empty-state-description mt-1 text-xs">{{ t('announcements.emptyDescription') }}</p>
 </div>
 </div>
 </div>
 </div>
 </Transition>
 </Teleport>

 <!-- 公告详情 Modal -->
 <Teleport to="body">
 <Transition name="modal-fade">
 <div
 v-if="detailModalOpen && selectedAnnouncement"
 class="modal-overlay announcement-overlay announcement-overlay-detail z-[110]"
 @click="closeDetail"
 >
 <div
 class="modal-content w-full max-w-[780px] overflow-hidden"
 @click.stop
 >
 <div class="modal-header px-8 py-6">
 <div class="flex w-full items-start justify-between gap-4">
 <div class="flex-1 min-w-0">
 <div class="mb-3 flex items-center gap-2">
 <div class="flex h-10 w-10 items-center justify-center rounded bg-[var(--nx-accent)] text-white">
 <Icon name="infoCircle" size="md" :stroke-width="2" />
 </div>
 <div class="flex items-center gap-2">
 <span class="badge badge-gray">
 {{ t('announcements.title') }}
 </span>
 <span
 v-if="!selectedAnnouncement.read_at"
 class="badge badge-primary"
 >
 <span class="relative flex h-2 w-2">
 <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-[var(--nx-accent)] opacity-75"></span>
 <span class="relative inline-flex h-2 w-2 rounded-full bg-[var(--nx-accent)]"></span>
 </span>
 {{ t('announcements.unread') }}
 </span>
 </div>
 </div>

 <h2 class="mb-3 text-2xl font-semibold leading-tight text-[var(--nx-text)]">
 {{ selectedAnnouncement.title }}
 </h2>

 <div class="flex flex-wrap items-center gap-4 text-sm text-[var(--nx-muted)]">
 <div class="flex items-center gap-1.5">
 <Icon name="clock" size="sm" />
 <time>{{ formatRelativeWithDateTime(selectedAnnouncement.created_at) }}</time>
 </div>
 <div class="flex items-center gap-1.5">
 <Icon name="eye" size="sm" />
 <span>{{ selectedAnnouncement.read_at ? t('announcements.read') : t('announcements.unread') }}</span>
 </div>
 </div>
 </div>

 <button
 @click="closeDetail"
 class="btn btn-ghost btn-icon flex-shrink-0"
 :aria-label="t('common.close')"
 >
 <Icon name="x" size="md" />
 </button>
 </div>
 </div>

 <div class="max-h-[60vh] overflow-y-auto bg-[var(--nx-surface)] px-8 py-8">
 <div class="relative">
 <div class="absolute bottom-0 left-0 top-0 w-1 rounded-full bg-[var(--nx-accent)]"></div>

 <div class="pl-6">
 <div
 class="markdown-body max-w-none"
 v-html="renderMarkdown(selectedAnnouncement.content)"
 ></div>
 </div>
 </div>
 </div>

 <div class="modal-footer px-8 py-5">
 <div class="flex w-full flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
 <div class="flex items-center gap-2 text-xs text-[var(--nx-subtle)]">
 <Icon name="infoCircle" size="sm" />
 <span>{{ selectedAnnouncement.read_at ? t('announcements.readStatus') : t('announcements.markReadHint') }}</span>
 </div>
 <div class="flex items-center gap-3">
 <button
 @click="closeDetail"
 class="btn btn-secondary"
 >
 {{ t('common.close') }}
 </button>
 <button
 v-if="!selectedAnnouncement.read_at"
 @click="markAsReadAndClose(selectedAnnouncement.id)"
 class="btn btn-primary"
 >
 <span class="flex items-center gap-2">
 <Icon name="check" size="sm" :stroke-width="2" />
 {{ t('announcements.markRead') }}
 </span>
 </button>
 </div>
 </div>
 </div>
 </div>
 </div>
 </Transition>
 </Teleport>
 </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import type { UserAnnouncement } from '@/types'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()

// Configure marked
marked.setOptions({
 breaks: true,
 gfm: true,
})

// Use store state (storeToRefs for reactivity)
const { announcements, loading } = storeToRefs(announcementStore)
const unreadCount = computed(() => announcementStore.unreadCount)

// Local modal state
const isModalOpen = ref(false)
const detailModalOpen = ref(false)
const selectedAnnouncement = ref<UserAnnouncement | null>(null)

// Methods
function renderMarkdown(content: string): string {
 if (!content) return ''
 const html = marked.parse(content) as string
 return DOMPurify.sanitize(html)
}

function openModal() {
 isModalOpen.value = true
}

function closeModal() {
 isModalOpen.value = false
}

function openDetail(announcement: UserAnnouncement) {
 selectedAnnouncement.value = announcement
 detailModalOpen.value = true
 if (!announcement.read_at) {
 markAsRead(announcement.id)
 }
}

function closeDetail() {
 detailModalOpen.value = false
 selectedAnnouncement.value = null
}

async function markAsRead(id: number) {
 try {
 await announcementStore.markAsRead(id)
 } catch (err: any) {
 appStore.showError(err?.message || t('common.unknownError'))
 }
}

async function markAsReadAndClose(id: number) {
 await markAsRead(id)
 appStore.showSuccess(t('announcements.markedAsRead'))
 closeDetail()
}

async function markAllAsRead() {
 try {
 await announcementStore.markAllAsRead()
 appStore.showSuccess(t('announcements.allMarkedAsRead'))
 } catch (err: any) {
 appStore.showError(err?.message || t('common.unknownError'))
 }
}

function handleEscape(e: KeyboardEvent) {
 if (e.key === 'Escape') {
 if (detailModalOpen.value) {
 closeDetail()
 } else if (isModalOpen.value) {
 closeModal()
 }
 }
}

onMounted(() => {
 document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
 document.removeEventListener('keydown', handleEscape)
 document.body.style.overflow = ''
})

watch(
 [isModalOpen, detailModalOpen, () => announcementStore.currentPopup],
 ([modal, detail, popup]) => {
 document.body.style.overflow = (modal || detail || popup) ? 'hidden' : ''
 }
)
</script>

<style scoped>
.modal-fade-enter-active {
 transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-fade-leave-active {
 transition: all 0.2s cubic-bezier(0.4, 0, 1, 1);
}

.modal-fade-enter-from,
.modal-fade-leave-to {
 opacity: 0;
}

.modal-fade-enter-from > div {
 transform: scale(0.94) translateY(-12px);
 opacity: 0;
}

.modal-fade-leave-to > div {
 transform: scale(0.96) translateY(-8px);
 opacity: 0;
}

.announcement-overlay {
 align-items: flex-start;
 overflow-y: auto;
 padding-top: 8vh;
}

.announcement-overlay-detail {
 padding-top: 6vh;
}
</style>

<style>
.markdown-body {
 @apply text-[15px] leading-[1.75];
 color: var(--nx-muted);
}

.markdown-body h1 {
 @apply mb-6 mt-8 border-b pb-3 text-3xl font-semibold;
 border-color: var(--nx-border);
 color: var(--nx-text);
}

.markdown-body h2 {
 @apply mb-4 mt-7 border-b pb-2 text-2xl font-semibold;
 border-color: var(--nx-border);
 color: var(--nx-text);
}

.markdown-body h3 {
 @apply mb-3 mt-6 text-xl font-semibold;
 color: var(--nx-text);
}

.markdown-body h4 {
 @apply mb-2 mt-5 text-lg font-semibold;
 color: var(--nx-text);
}

.markdown-body p {
 @apply mb-4 leading-relaxed;
}

.markdown-body a {
 @apply font-medium underline decoration-2 underline-offset-2 transition-all;
 color: var(--nx-accent);
 text-decoration-color: rgba(255, 86, 0, 0.3);
}

.markdown-body ul,
.markdown-body ol {
 @apply mb-4 ml-6 space-y-2;
}

.markdown-body ul {
 @apply list-disc;
}

.markdown-body ol {
 @apply list-decimal;
}

.markdown-body li {
 @apply leading-relaxed;
 @apply pl-2;
}

.markdown-body li::marker {
 color: var(--nx-accent);
}

.markdown-body blockquote {
 @apply relative my-5 border-l-4 py-3 pl-5 pr-4 italic;
 background: var(--nx-bg);
 border-left-color: var(--nx-accent);
 color: var(--nx-muted);
}

.markdown-body blockquote::before {
 content: '"';
 @apply absolute -left-1 top-0 text-5xl font-serif;
 color: rgba(255, 86, 0, 0.2);
}

.markdown-body code {
 @apply rounded px-2 py-1 text-[13px] font-mono;
 background: var(--nx-bg);
 color: var(--nx-danger);
}

.markdown-body pre {
 @apply my-5 overflow-x-auto rounded border p-5;
 background: var(--nx-bg);
 border-color: var(--nx-border);
}

.markdown-body pre code {
 @apply bg-transparent p-0 text-[13px];
 color: var(--nx-text);
}

.markdown-body hr {
 @apply my-8 border-0 border-t-2;
 border-color: var(--nx-border);
}

.markdown-body table {
 @apply mb-5 w-full overflow-hidden rounded border;
 border-color: var(--nx-border);
}

.markdown-body th,
.markdown-body td {
 @apply border-r border-b px-4 py-3 text-left;
 border-color: var(--nx-border);
}

.markdown-body th:last-child,
.markdown-body td:last-child {
 @apply border-r-0;
}

.markdown-body tr:last-child td {
 @apply border-b-0;
}

.markdown-body th {
 @apply font-semibold;
 background: var(--nx-bg);
 color: var(--nx-text);
}

.markdown-body tbody tr {
 transition: background-color 0.15s ease;
}

.markdown-body tbody tr:hover {
 background: var(--nx-bg);
}

.markdown-body img {
 @apply my-5 max-w-full rounded border;
 border-color: var(--nx-border);
}

.markdown-body strong {
 @apply font-semibold;
 color: var(--nx-text);
}

.markdown-body em {
 @apply italic;
 color: var(--nx-muted);
}
</style>
