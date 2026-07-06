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
            Projection
          </div>
          <div class="mt-1 text-gray-900 dark:text-gray-100">
            {{ formatDateTime(summary?.projection_at) }}
          </div>
        </div>
        <div class="rounded-md border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            Generated
          </div>
          <div class="mt-1 text-gray-900 dark:text-gray-100">
            {{ formatDateTime(summary?.generated_at) }}
          </div>
        </div>
      </div>

      <div v-if="loading && !summary" class="card p-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div v-else-if="!summary?.groups.length" class="card p-6 text-sm text-gray-500 dark:text-gray-400">
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
              {{ group.group_name }}
            </h2>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ group.ungrouped ? 'Ungrouped' : `#${group.group_id}` }}
            </p>
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ group.rows.length }} rows
          </span>
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900/60">
              <tr>
                <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">Type</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Included</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Errors</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Inactive</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Other</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Missing 5h</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Missing 7d</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Avg 5h</th>
                <th class="px-4 py-2 text-right font-medium text-gray-600 dark:text-gray-300">Avg 7d</th>
                <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">Next 5h Recovery</th>
                <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">Next 7d Recovery</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr
                v-for="row in group.rows"
                :key="`${group.group_id ?? 'ungrouped'}-${row.account_type}`"
                class="hover:bg-gray-50 dark:hover:bg-dark-700/40"
              >
                <td class="whitespace-nowrap px-4 py-3 font-medium text-gray-900 dark:text-gray-100">{{ row.account_type }}</td>
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
import { defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { accountsAPI } from '@/api/admin/accounts'
import type { OpenAIQuotaRecovery, OpenAIQuotaSummaryParams, OpenAIQuotaSummaryResponse } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'

type ProjectionMode = 'current' | 'hours' | 'days'

const { t } = useI18n()
const appStore = useAppStore()

const projectionMode = ref<ProjectionMode>('current')
const projectionAmount = ref(1)
const summary = ref<OpenAIQuotaSummaryResponse | null>(null)
const loading = ref(false)

const activeModeClass = 'bg-primary-600 text-white'
const inactiveModeClass = 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700'

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

async function loadSummary(): Promise<void> {
 loading.value = true
 try {
  summary.value = await accountsAPI.getOpenAIQuotaSummary(projectionParams())
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
    h('div', { class: 'font-medium text-gray-900 dark:text-gray-100' }, [
     props.recovery.account_name,
     h('span', { class: 'ml-1 text-xs font-normal text-gray-500 dark:text-gray-400' }, `#${props.recovery.account_id}`),
    ]),
    h('div', { class: 'text-xs text-gray-500 dark:text-gray-400' }, formatDateTime(props.recovery.reset_at)),
    h('div', { class: 'text-xs text-gray-500 dark:text-gray-400' }, `${formatPercent(props.recovery.remaining_before_percent)} -> ${formatPercent(props.recovery.remaining_after_percent)}`),
   ])
  }
 },
})

onMounted(loadSummary)
</script>
