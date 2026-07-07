<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <button
            type="button"
            data-test="usage-interaction-back"
            class="btn btn-secondary btn-sm mb-3 inline-flex items-center gap-1.5"
            @click="goBack"
          >
            <Icon name="arrowLeft" size="sm" />
            {{ t('common.back') }}
          </button>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
            {{ t('admin.usageInteraction.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ interaction?.request_id || `#${usageLogId}` }}
          </p>
        </div>
        <div v-if="interaction" class="flex flex-wrap items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
          <span class="rounded border border-gray-200 px-2 py-1 dark:border-dark-600">
            {{ interaction.capture_status }}
          </span>
          <span v-if="interaction.redaction_applied" class="rounded border border-amber-200 bg-amber-50 px-2 py-1 text-amber-700 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-300">
            {{ t('admin.usageInteraction.redacted') }}
          </span>
        </div>
      </div>

      <div v-if="loading && !interaction" class="card p-4 text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div v-else-if="errorMessage" class="card border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300">
        {{ errorMessage }}
      </div>

      <div v-else-if="!interaction" class="card p-4 text-sm text-gray-600 dark:text-gray-300">
        {{ notFoundMessage }}
      </div>

      <template v-else>
        <div class="mb-4 flex gap-2 border-b border-gray-200 dark:border-dark-700">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            :data-test="`usage-interaction-tab-${tab.key}`"
            class="tab"
            :class="{ 'tab-active': activeTab === tab.key }"
            @click="selectTab(tab.key)"
          >
            {{ tab.label }}
          </button>
        </div>

        <div class="card overflow-hidden">
          <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ activeTabLabel }}</h2>
          </div>
          <div v-if="activeTab === 'raw' && rawLoading" class="p-4 text-sm text-gray-500 dark:text-gray-400">
            {{ t('common.loading') }}
          </div>
          <div v-else-if="activeTab === 'raw' && rawError" class="p-4 text-sm text-red-600 dark:text-red-300">
            {{ rawError }}
          </div>
          <div v-else class="divide-y divide-gray-200 dark:divide-dark-700">
            <JsonBlock
              v-for="section in activeSections"
              :key="section.title"
              :title="section.title"
              :value="section.value"
            />
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminUsageAPI } from '@/api/admin/usage'
import type { UsageInteractionDetail } from '@/api/admin/usage'

type TabKey = 'input' | 'output' | 'parameters' | 'routing' | 'raw'

const JsonBlock = defineComponent({
 name: 'JsonBlock',
 props: {
 title: { type: String, required: true },
 value: { type: null, required: true },
 },
 setup(props) {
  const formatted = computed(() => JSON.stringify(props.value, null, 2))
  return () => h('section', { class: 'p-4' }, [
   h('h3', { class: 'mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400' }, props.title),
   h('pre', { class: 'whitespace-pre-wrap break-words rounded bg-gray-50 p-3 text-xs leading-5 text-gray-900 dark:bg-dark-900 dark:text-gray-100' }, formatted.value),
  ])
 },
})

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const activeTab = ref<TabKey>('input')
const interaction = ref<UsageInteractionDetail | null>(null)
const loading = ref(false)
const rawLoading = ref(false)
const rawLoaded = ref(false)
const errorMessage = ref('')
const rawError = ref('')

const usageLogId = computed(() => Number(route.params.id))

const tabs = computed(() => [
 { key: 'input' as const, label: t('admin.usageInteraction.tabs.input') },
 { key: 'output' as const, label: t('admin.usageInteraction.tabs.output') },
 { key: 'parameters' as const, label: t('admin.usageInteraction.tabs.parameters') },
 { key: 'routing' as const, label: t('admin.usageInteraction.tabs.routing') },
 { key: 'raw' as const, label: t('admin.usageInteraction.tabs.raw') },
])

const activeTabLabel = computed(() => tabs.value.find((tab) => tab.key === activeTab.value)?.label ?? '')

const notFoundMessage = computed(() => t('admin.usageInteraction.notFound'))

const activeSections = computed(() => {
 if (!interaction.value) return []
 if (activeTab.value === 'input') {
  return [{ title: t('admin.usageInteraction.sections.input'), value: interaction.value.request_content }]
 }
 if (activeTab.value === 'output') {
  return [{ title: t('admin.usageInteraction.sections.output'), value: interaction.value.response_content }]
 }
 if (activeTab.value === 'parameters') {
  return [{ title: t('admin.usageInteraction.sections.parameters'), value: interaction.value.request_parameters }]
 }
 if (activeTab.value === 'routing') {
  return [{ title: t('admin.usageInteraction.sections.routing'), value: interaction.value.routing_context }]
 }
 return [
  { title: t('admin.usageInteraction.sections.rawRequest'), value: interaction.value.raw_request_json ?? null },
  { title: t('admin.usageInteraction.sections.rawResponse'), value: interaction.value.raw_response_json ?? null },
 ]
})

const loadInteraction = async () => {
 loading.value = true
 errorMessage.value = ''
 try {
  const response = await adminUsageAPI.getInteraction(usageLogId.value)
  interaction.value = response.exists ? response.interaction ?? null : null
 } catch (error) {
  console.error('Failed to load usage interaction:', error)
  errorMessage.value = t('admin.usageInteraction.failedToLoad')
 } finally {
  loading.value = false
 }
}

const loadRawInteraction = async () => {
 if (rawLoaded.value || rawLoading.value) return
 rawLoading.value = true
 rawError.value = ''
 try {
  const response = await adminUsageAPI.getInteraction(usageLogId.value, { includeRaw: true })
  if (response.exists && response.interaction) {
   interaction.value = response.interaction
   rawLoaded.value = true
  }
 } catch (error) {
  console.error('Failed to load raw usage interaction:', error)
  rawError.value = t('admin.usageInteraction.failedToLoadRaw')
 } finally {
  rawLoading.value = false
 }
}

const selectTab = (tab: TabKey) => {
 activeTab.value = tab
 if (tab === 'raw') {
  void loadRawInteraction()
 }
}

const goBack = () => {
 const returnTo = Array.isArray(route.query.return) ? route.query.return[0] : route.query.return
 router.push(returnTo || '/admin/usage')
}

onMounted(() => {
 void loadInteraction()
})
</script>
