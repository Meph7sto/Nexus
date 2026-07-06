<template>
 <div class="relative flex min-h-screen items-center justify-center overflow-hidden bg-[var(--nx-bg)] p-4 text-[var(--nx-text)]">
 <!-- Content Container -->
 <div class="relative z-10 w-full max-w-md">
 <!-- Logo/Brand -->
 <div class="mb-8 text-center">
 <!-- Custom Logo or Default Logo -->
 <template v-if="settingsLoaded">
 <div
 class="mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-lg border border-[var(--nx-border)] bg-[var(--nx-surface)]"
 >
 <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
 </div>
 <h1 class="mb-2 text-3xl font-semibold text-[var(--nx-text)]">
 {{ siteName }}
 </h1>
 <p class="text-sm text-[var(--nx-muted)]">
 {{ siteSubtitle }}
 </p>
 </template>
 </div>

 <!-- Card Container -->
 <div class="rounded-lg border border-[var(--nx-border)] bg-[var(--nx-surface)] p-8">
 <slot />
 </div>

 <!-- Footer Links -->
 <div class="mt-6 text-center text-sm">
 <slot name="footer" />
 </div>

 <!-- Copyright -->
 <div class="mt-8 text-center text-xs text-[var(--nx-muted)]">
 &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
 </div>
 </div>
 </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Nexus')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
 appStore.fetchPublicSettings()
})
</script>
