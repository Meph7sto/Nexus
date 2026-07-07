<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-end md:justify-between">
        <div class="min-w-0">
          <h1 class="truncate text-xl font-semibold text-gray-900 dark:text-gray-100">
            {{ t('admin.openAIQuotaSummary.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.openAIQuotaSummary.description') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <select
            v-model="selectedGroup"
            data-test="group-filter"
            class="input w-44"
          >
            <option value="">{{ t('admin.openAIQuotaSummary.allGroups') }}</option>
            <option value="ungrouped">{{ t('admin.openAIQuotaSummary.ungrouped') }}</option>
            <option v-for="group in groupOptions" :key="group.id" :value="String(group.id)">
              {{ group.name }}
            </option>
          </select>
          <select
            v-model="selectedType"
            data-test="type-filter"
            class="input w-40"
          >
            <option value="">{{ t('admin.openAIQuotaSummary.allTypes') }}</option>
            <option v-for="type in typeOptions" :key="type" :value="type">
              {{ planTypeLabel(type) }}
            </option>
          </select>
          <div class="inline-flex overflow-hidden rounded-md border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
            <button
              type="button"
              class="px-3 py-2 text-sm font-medium transition"
              :class="projectionMode === 'current' ? activeModeClass : inactiveModeClass"
              @click="projectionMode = 'current'"
            >
              {{ t('admin.openAIQuotaSummary.current') }}
            </button>
            <button
              type="button"
              data-test="projection-mode-hours"
              class="border-l border-gray-200 px-3 py-2 text-sm font-medium transition dark:border-dark-600"
              :class="projectionMode === 'hours' ? activeModeClass : inactiveModeClass"
              @click="projectionMode = 'hours'"
            >
              {{ t('admin.openAIQuotaSummary.hoursLater') }}
            </button>
            <button
              type="button"
              data-test="projection-mode-days"
              class="border-l border-gray-200 px-3 py-2 text-sm font-medium transition dark:border-dark-600"
              :class="projectionMode === 'days' ? activeModeClass : inactiveModeClass"
              @click="projectionMode = 'days'"
            >
              {{ t('admin.openAIQuotaSummary.daysLater') }}
            </button>
          </div>
          <input
            v-model.number="projectionAmount"
            data-test="projection-amount"
            type="number"
            min="1"
            class="input w-24"
            :disabled="projectionMode === 'current'"
          >
          <button
            type="button"
            data-test="refresh"
            class="btn btn-primary"
            :disabled="loading"
            @click="loadSummary"
          >
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>

      <div class="grid gap-3 text-sm sm:grid-cols-2">
        <div class="rounded-md border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.openAIQuotaSummary.projection') }}
          </div>
          <div class="mt-1 text-gray-900 dark:text-gray-100">
            {{ formatDateTime(summary?.projection_at) }}
          </div>
        </div>
        <div class="rounded-md border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.openAIQuotaSummary.generated') }}
          </div>
          <div class="mt-1 text-gray-900 dark:text-gray-100">
            {{ formatDateTime(summary?.generated_at) }}
          </div>
        </div>
      </div>

      <div v-if="loading && !summary" class="card p-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div v-else-if="!loading && !summary?.groups.length" class="card p-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.noData') }}
      </div>

      <section
        v-for="group in summary?.groups"
        v-else
        :key="group.group_id ?? 'ungrouped'"
        class="rounded-md border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <div class="min-w-0">
            <h2 class="truncate text-base font-semibold text-gray-900 dark:text-gray-100">
              {{ group.ungrouped ? t('admin.openAIQuotaSummary.ungrouped') : group.group_name }}
            </h2>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ group.ungrouped ? t('admin.openAIQuotaSummary.ungrouped') : `#${group.group_id}` }}
            </p>
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.openAIQuotaSummary.rows', { count: group.rows.length }) }}
          </span>
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900/60">
              <tr>
                <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.type') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.included') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.errors') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.inactive') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.other') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.missing5h') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.missing7d') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.avg5h') }}</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.avg7d') }}</th>
                <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.next5hRecovery') }}</th>
                <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.openAIQuotaSummary.table.next7dRecovery') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr
                v-for="row in group.rows"
                :key="`${group.group_id ?? 'ungrouped'}-${row.account_type}`"
                class="hover:bg-gray-50 dark:hover:bg-dark-700/40"
              >
                <td class="whitespace-nowrap px-4 py-3 font-medium text-gray-900 dark:text-gray-100">{{ planTypeLabel(row.account_type) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ row.included_count }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ row.error_count }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ row.inactive_count }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ row.other_excluded_count }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ row.missing_5h_snapshot_count }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ row.missing_7d_snapshot_count }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ formatPercent(row.avg_5h_remaining_percent) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-300">{{ formatPercent(row.avg_7d_remaining_percent) }}</td>
                <td class="min-w-48 px-4 py-3 text-gray-700 dark:text-gray-300">
                  <RecoveryCell :recovery="row.earliest_5h_recovery" />
                </td>
                <td class="min-w-48 px-4 py-3 text-gray-700 dark:text-gray-300">
                  <RecoveryCell :recovery="row.earliest_7d_recovery" />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { accountsAPI } from '@/api/admin/accounts'
import { groupsAPI } from '@/api/admin/groups'
import type { OpenAIQuotaRecovery, OpenAIQuotaSummaryParams, OpenAIQuotaSummaryResponse } from '@/api/admin/accounts'
import type { AdminGroup } from '@/types'
import { useAppStore } from '@/stores/app'

type ProjectionMode = 'current' | 'hours' | 'days'
type OpenAIQuotaSummaryTypeFilter = string
interface GroupFilterOption {
 id: number
 name: string
}

const { t } = useI18n()
const appStore = useAppStore()

const projectionMode = ref<ProjectionMode>('current')
const projectionAmount = ref(1)
const selectedGroup = ref('')
const selectedType = ref<OpenAIQuotaSummaryTypeFilter>('')
const groups = ref<AdminGroup[]>([])
const summary = ref<OpenAIQuotaSummaryResponse | null>(null)
const loading = ref(true)

const activeModeClass = 'bg-primary-600 text-white'
const inactiveModeClass = 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700'
const commonPlanTypes = ['free', 'plus', 'pro', 'team', 'enterprise', 'unknown']

const groupOptions = computed<GroupFilterOption[]>(() => {
 const options: GroupFilterOption[] = []
 const seen = new Set<number>()

 for (const group of groups.value) {
  if (group.platform !== 'openai') continue
  options.push({ id: group.id, name: group.name })
  seen.add(group.id)
 }

 for (const group of summary.value?.groups ?? []) {
  if (group.ungrouped || group.group_id == null || seen.has(group.group_id)) continue
  options.push({ id: group.group_id, name: group.group_name })
  seen.add(group.group_id)
 }

 return options
})

const typeOptions = computed<string[]>(() => {
 const options = new Set<string>()
 for (const type of commonPlanTypes) {
  options.add(type)
 }
 if (selectedType.value) {
  options.add(selectedType.value)
 }
 for (const group of summary.value?.groups ?? []) {
  for (const row of group.rows) {
   const type = row.account_type.trim()
   if (type) options.add(type)
  }
 }
 return Array.from(options).sort((a, b) => planTypeLabel(a).localeCompare(planTypeLabel(b)))
})

function formatPercent(value: number | null | undefined): string {
 if (value == null || Number.isNaN(value)) return '-'
 return `${value.toFixed(1)}%`
}

function formatDateTime(value: string | null | undefined): string {
 if (!value) return '-'
 const date = new Date(value)
 if (Number.isNaN(date.getTime())) return value
 return date.toLocaleString()
}

function planTypeLabel(type: string): string {
 const normalized = type.trim()
 if (!normalized) return '-'
 if (normalized.toLowerCase() === 'unknown') return 'Unknown'
 return normalized
  .split(/[-_\s]+/)
  .filter(Boolean)
  .map(part => part.charAt(0).toUpperCase() + part.slice(1))
  .join(' ')
}

function projectionParams(): OpenAIQuotaSummaryParams {
 if (projectionMode.value === 'current') return {}

 const date = new Date()
 const amount = Math.max(1, Number(projectionAmount.value) || 1)
 if (projectionMode.value === 'hours') {
  date.setHours(date.getHours() + amount)
 } else {
  date.setDate(date.getDate() + amount)
 }

 return { projection_at: date.toISOString() }
}

function summaryParams(): OpenAIQuotaSummaryParams {
 const params = projectionParams()
 if (selectedGroup.value) params.group = selectedGroup.value
 if (selectedType.value) params.type = selectedType.value
 return params
}

async function loadGroups(): Promise<void> {
 try {
  groups.value = await groupsAPI.getAllIncludingInactive()
 } catch (error) {
  appStore.showError(error instanceof Error ? error.message : String(error))
 }
}

async function loadSummary(): Promise<void> {
 loading.value = true
 try {
  summary.value = await accountsAPI.getOpenAIQuotaSummary(summaryParams())
 } catch (error) {
  appStore.showError(error instanceof Error ? error.message : String(error))
 } finally {
  loading.value = false
 }
}

const RecoveryCell = defineComponent({
 name: 'RecoveryCell',
 props: {
  recovery: {
   type: Object as () => OpenAIQuotaRecovery | null,
   default: null,
  },
 },
 setup(props) {
  return () => {
   if (!props.recovery) {
    return h('span', { class: 'text-gray-400 dark:text-gray-500' }, '-')
   }

   return h('div', { class: 'space-y-0.5' }, [
    h('div', { class: 'font-medium text-gray-900 dark:text-gray-100' }, formatDateTime(props.recovery.reset_at)),
    h('div', { class: 'text-xs text-gray-500 dark:text-gray-400' }, `${formatPercent(props.recovery.remaining_before_percent)} -> ${formatPercent(props.recovery.remaining_after_percent)}`),
   ])
  }
 },
})

onMounted(() => {
 void loadGroups()
 void loadSummary()
})
</script>
