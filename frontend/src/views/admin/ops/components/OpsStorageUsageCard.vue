<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import { opsAPI, type OpsStorageUsageItem, type OpsStorageUsageResponse } from '@/api/admin/ops'
import { formatBytes } from '@/utils/format'

const props = defineProps<{
 refreshKey?: number | string | null
 fullscreen?: boolean
}>()

const { t } = useI18n()

const loading = ref(false)
const data = ref<OpsStorageUsageResponse | null>(null)
const errorMessage = ref('')
let controller: AbortController | null = null

const totalLabel = computed(() => formatStorageBytes(data.value?.total_used_bytes ?? null))

const visibleItems = computed(() => {
 const items = data.value?.items ?? []
 return [...items].sort((a, b) => itemRank(a) - itemRank(b))
})

function itemRank(item: OpsStorageUsageItem): number {
 if (item.key === 'postgres_data') return 0
 if (item.key === 'docker') return 1
 if (item.key === 'postgres_db') return 2
 if (item.key === 'app_data') return 3
 return 10
}

function formatStorageBytes(value: number | null | undefined): string {
 if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
 return formatBytes(value, value >= 1024 * 1024 * 1024 ? 1 : 0)
}

function statusLabel(item: OpsStorageUsageItem): string {
 if (item.status === 'ok') return formatStorageBytes(item.used_bytes)
 if (item.status === 'unconfigured') return t('admin.ops.storage.status.unconfigured')
 return t('admin.ops.storage.status.unavailable')
}

function statusClass(item: OpsStorageUsageItem): string {
 if (item.status === 'ok') return 'text-gray-900'
 if (item.status === 'unconfigured') return 'text-gray-400'
 return 'text-yellow-600'
}

async function loadStorageUsage() {
 if (loading.value) return
 controller?.abort()
 controller = new AbortController()
 loading.value = true
 errorMessage.value = ''
 try {
 data.value = await opsAPI.getStorageUsage({ signal: controller.signal })
 } catch (err: any) {
 if (err?.code === 'ERR_CANCELED') return
 errorMessage.value = err?.message || t('admin.ops.storage.loadFailed')
 } finally {
 loading.value = false
 }
}

watch(
 () => props.refreshKey,
 () => {
 loadStorageUsage()
 }
)

onMounted(() => {
 loadStorageUsage()
})

onUnmounted(() => {
 controller?.abort()
 controller = null
})
</script>

<template>
 <div class="rounded-lg bg-gray-50 p-3">
 <div class="flex items-center justify-between gap-2">
 <div class="flex items-center gap-1">
 <div class="text-[10px] font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.storage.title') }}</div>
 <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.storage.tooltip')" />
 </div>
 <span v-if="loading" class="h-2 w-2 animate-pulse rounded-full bg-blue-400"></span>
 </div>

 <div class="mt-1 text-lg font-black text-gray-900">
 {{ loading && !data ? '-' : totalLabel }}
 </div>

 <div v-if="errorMessage && !data" class="mt-1 truncate text-[10px] text-yellow-600" :title="errorMessage">
 {{ t('admin.ops.storage.loadFailed') }}
 </div>
 <div v-else-if="!visibleItems.length" class="mt-1 text-[10px] text-gray-500">
 {{ t('admin.ops.noData') }}
 </div>
 <div v-else class="mt-1 space-y-0.5 text-[10px] text-gray-500">
 <div
 v-for="item in visibleItems"
 :key="item.key"
 class="flex items-center justify-between gap-2"
 :title="item.error || item.path || item.label"
 >
 <span class="min-w-0 truncate">{{ item.label }}</span>
 <span class="shrink-0 font-mono font-semibold" :class="statusClass(item)">
 {{ statusLabel(item) }}
 </span>
 </div>
 </div>
 </div>
</template>
