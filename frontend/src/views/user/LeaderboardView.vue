<template>
 <AppLayout>
 <section class="leaderboard-page">
 <header class="leaderboard-header">
 <div>
 <p class="section-label">{{ t('leaderboard.label') }}</p>
 <h1>{{ t('leaderboard.title') }}</h1>
 </div>
 <DateRangePicker
 v-model:start-date="startDate"
 v-model:end-date="endDate"
 @change="onDateRangeChange"
 />
 </header>

 <div class="leaderboard-toolbar">
 <div class="metric-tabs" role="tablist" :aria-label="t('leaderboard.metric')">
 <button
 type="button"
 :class="['metric-tab', rankBy === 'tokens' && 'metric-tab-active']"
 role="tab"
 :aria-selected="rankBy === 'tokens'"
 @click="setRankBy('tokens')"
 >
 <Icon name="chartBar" size="sm" />
 {{ t('leaderboard.tokens') }}
 </button>
 <button
 type="button"
 :class="['metric-tab', rankBy === 'cost' && 'metric-tab-active']"
 role="tab"
 :aria-selected="rankBy === 'cost'"
 @click="setRankBy('cost')"
 >
 <Icon name="dollar" size="sm" />
 {{ t('leaderboard.cost') }}
 </button>
 </div>

 <button type="button" class="refresh-button" :disabled="loading" @click="loadRanking">
 <Icon name="refresh" size="sm" />
 {{ t('common.refresh') }}
 </button>
 </div>

 <div class="leaderboard-card">
 <div class="leaderboard-card-header">
 <div>
 <p class="section-label">{{ rankBy === 'tokens' ? t('leaderboard.tokens') : t('leaderboard.cost') }}</p>
 <h2>{{ t('leaderboard.tableTitle') }}</h2>
 </div>
 <div class="leaderboard-total">
 <span>{{ t('leaderboard.totalUsers') }}</span>
 <strong>{{ pagination.total }}</strong>
 </div>
 </div>

 <div class="table-shell">
 <table class="leaderboard-table">
 <thead>
 <tr>
 <th>{{ t('leaderboard.columns.rank') }}</th>
 <th>{{ t('leaderboard.columns.user') }}</th>
 <th>{{ t('leaderboard.columns.email') }}</th>
 <th class="text-right">{{ t('leaderboard.columns.requests') }}</th>
 <th class="text-right">{{ t('leaderboard.columns.tokens') }}</th>
 <th class="text-right">{{ t('leaderboard.columns.cost') }}</th>
 </tr>
 </thead>
 <tbody v-if="loading">
 <tr v-for="index in pagination.page_size" :key="index">
 <td colspan="6">
 <div class="skeleton-row"></div>
 </td>
 </tr>
 </tbody>
 <tbody v-else-if="rows.length > 0">
 <tr v-for="row in rows" :key="`${row.rank}-${row.email}-${row.nickname}`">
 <td>
 <span :class="['rank-pill', row.rank <= 3 && 'rank-pill-top']">#{{ row.rank }}</span>
 </td>
 <td>
 <div class="user-cell">
 <span class="avatar-mark">{{ avatarText(row.nickname) }}</span>
 <span class="user-name">{{ row.nickname || t('common.unknown') }}</span>
 </div>
 </td>
 <td class="muted-cell">{{ row.email || t('common.notAvailable') }}</td>
 <td class="text-right tabular">{{ formatInteger(row.requests) }}</td>
 <td class="text-right tabular">{{ formatInteger(row.total_tokens) }}</td>
 <td class="text-right tabular">{{ formatCurrency(row.total_actual_cost) }}</td>
 </tr>
 </tbody>
 <tbody v-else>
 <tr>
 <td colspan="6">
 <div class="empty-panel">
 <Icon name="chartBar" size="lg" />
 <p>{{ t('leaderboard.empty') }}</p>
 </div>
 </td>
 </tr>
 </tbody>
 </table>
 </div>

 <Pagination
 v-if="pagination.total > 0"
 :page="pagination.page"
 :total="pagination.total"
 :page-size="pagination.page_size"
 :show-jump="true"
 @update:page="handlePageChange"
 @update:pageSize="handlePageSizeChange"
 />
 </div>
 </section>
 </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import type { UsageRankingItem, UsageRankingMetric } from '@/types'

const { t, locale } = useI18n()
const appStore = useAppStore()

const formatLocalDate = (date: Date): string =>
 `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`

const getLast24HoursRangeDates = () => {
 const end = new Date()
 const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
 return { start: formatLocalDate(start), end: formatLocalDate(end) }
}

const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)
const rankBy = ref<UsageRankingMetric>('tokens')
const rows = ref<UsageRankingItem[]>([])
const loading = ref(false)

const pagination = reactive({
 page: 1,
 page_size: getPersistedPageSize(),
 total: 0,
})

const formatInteger = (value: number) => new Intl.NumberFormat(locale.value).format(value || 0)

const formatCurrency = (value: number) =>
 new Intl.NumberFormat(locale.value, {
 style: 'currency',
 currency: 'USD',
 minimumFractionDigits: 4,
 maximumFractionDigits: 6,
 }).format(value || 0)

const avatarText = (value: string) => {
 const first = value.trim().charAt(0)
 return first ? first.toUpperCase() : '#'
}

const loadRanking = async () => {
 loading.value = true
 try {
 const response = await usageAPI.getRanking({
 rank_by: rankBy.value,
 start_date: startDate.value,
 end_date: endDate.value,
 page: pagination.page,
 page_size: pagination.page_size,
 })
 rows.value = response.items
 pagination.total = response.total
 pagination.page = response.page
 pagination.page_size = response.page_size
 } catch (error) {
 console.error('[LeaderboardView] loadRanking failed:', error)
 rows.value = []
 pagination.total = 0
 appStore.showError(t('leaderboard.failedToLoad'))
 } finally {
 loading.value = false
 }
}

const setRankBy = (metric: UsageRankingMetric) => {
 if (rankBy.value === metric) return
 rankBy.value = metric
 pagination.page = 1
 void loadRanking()
}

const onDateRangeChange = (range: { startDate: string; endDate: string }) => {
 startDate.value = range.startDate
 endDate.value = range.endDate
 pagination.page = 1
 void loadRanking()
}

const handlePageChange = (page: number) => {
 pagination.page = page
 void loadRanking()
}

const handlePageSizeChange = (pageSize: number) => {
 pagination.page_size = pageSize
 pagination.page = 1
 void loadRanking()
}

onMounted(() => {
 void loadRanking()
})
</script>

<style scoped>
.leaderboard-page {
 max-width: 1180px;
 margin: 0 auto;
}

.leaderboard-header {
 display: flex;
 align-items: flex-end;
 justify-content: space-between;
 gap: 16px;
 margin-bottom: 24px;
 padding-bottom: 20px;
 border-bottom: 1px solid var(--nx-border);
}

.section-label {
 margin-bottom: 8px;
 color: var(--nx-subtle);
 font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
 font-size: 12px;
 font-weight: 500;
 line-height: 1;
 letter-spacing: 1.2px;
 text-transform: uppercase;
}

.leaderboard-header h1 {
 color: var(--nx-text);
 font-size: 40px;
 font-weight: 400;
 line-height: 1;
 letter-spacing: -1.2px;
}

.leaderboard-toolbar {
 display: flex;
 align-items: center;
 justify-content: space-between;
 gap: 12px;
 margin-bottom: 16px;
}

.metric-tabs {
 display: inline-flex;
 gap: 4px;
 padding: 4px;
 background: var(--nx-surface-muted);
 border: 1px solid var(--nx-border);
 border-radius: 8px;
}

.metric-tab,
.refresh-button {
 display: inline-flex;
 align-items: center;
 justify-content: center;
 gap: 8px;
 min-height: 38px;
 padding: 8px 14px;
 border: 1px solid transparent;
 border-radius: 4px;
 color: var(--nx-muted);
 font-size: 14px;
 font-weight: 500;
 transition: background-color 0.18s ease, border-color 0.18s ease, color 0.18s ease, transform 0.18s ease;
}

.metric-tab:hover,
.refresh-button:hover:not(:disabled) {
 color: var(--nx-text);
 transform: scale(1.03);
}

.metric-tab-active {
 background: var(--nx-surface);
 border-color: var(--nx-border);
 color: var(--nx-text);
}

.refresh-button {
 background: var(--nx-text);
 color: var(--nx-surface);
}

.refresh-button:disabled {
 cursor: not-allowed;
 opacity: 0.5;
 transform: none;
}

.leaderboard-card {
 overflow: hidden;
 background: var(--nx-surface);
 border: 1px solid var(--nx-border);
 border-radius: 8px;
}

.leaderboard-card-header {
 display: flex;
 align-items: center;
 justify-content: space-between;
 gap: 16px;
 padding: 18px 20px;
 border-bottom: 1px solid var(--nx-border);
}

.leaderboard-card-header h2 {
 color: var(--nx-text);
 font-size: 24px;
 font-weight: 400;
 line-height: 1;
 letter-spacing: -0.48px;
}

.leaderboard-total {
 display: flex;
 align-items: baseline;
 gap: 8px;
 color: var(--nx-muted);
 font-size: 13px;
}

.leaderboard-total strong {
 color: var(--nx-text);
 font-size: 24px;
 font-weight: 400;
 line-height: 1;
}

.table-shell {
 overflow-x: auto;
}

.leaderboard-table {
 width: 100%;
 min-width: 760px;
 border-collapse: collapse;
 font-size: 14px;
}

.leaderboard-table th {
 padding: 12px 16px;
 background: var(--nx-bg);
 border-bottom: 1px solid var(--nx-border);
 color: var(--nx-muted);
 font-size: 12px;
 font-weight: 500;
 letter-spacing: 0.6px;
 text-align: left;
 text-transform: uppercase;
}

.leaderboard-table th.text-right {
 text-align: right;
}

.leaderboard-table td {
 padding: 14px 16px;
 border-bottom: 1px solid var(--nx-border);
 color: var(--nx-text);
 vertical-align: middle;
}

.leaderboard-table tr:last-child td {
 border-bottom: 0;
}

.leaderboard-table tbody tr:hover {
 background: var(--nx-bg);
}

.rank-pill {
 display: inline-flex;
 align-items: center;
 justify-content: center;
 min-width: 48px;
 padding: 4px 8px;
 background: var(--nx-surface-muted);
 border: 1px solid var(--nx-border);
 border-radius: 4px;
 color: var(--nx-muted);
 font-variant-numeric: tabular-nums;
}

.rank-pill-top {
 background: rgba(255, 86, 0, 0.1);
 border-color: rgba(255, 86, 0, 0.26);
 color: var(--nx-accent);
}

.user-cell {
 display: flex;
 align-items: center;
 gap: 10px;
 min-width: 180px;
}

.avatar-mark {
 display: inline-flex;
 align-items: center;
 justify-content: center;
 width: 30px;
 height: 30px;
 flex: 0 0 auto;
 background: var(--nx-text);
 border-radius: 4px;
 color: var(--nx-surface);
 font-size: 13px;
 font-weight: 600;
}

.user-name {
 min-width: 0;
 overflow: hidden;
 text-overflow: ellipsis;
 white-space: nowrap;
}

.muted-cell {
 color: var(--nx-muted);
}

.tabular {
 font-variant-numeric: tabular-nums;
}

.skeleton-row {
 height: 20px;
 border-radius: 4px;
 background: var(--nx-surface-muted);
 animation: leaderboard-skeleton 1.1s ease-in-out infinite;
}

.empty-panel {
 display: flex;
 min-height: 180px;
 flex-direction: column;
 align-items: center;
 justify-content: center;
 gap: 10px;
 color: var(--nx-subtle);
}

@keyframes leaderboard-skeleton {
 0%, 100% { opacity: 0.55; }
 50% { opacity: 1; }
}

@media (max-width: 768px) {
 .leaderboard-header,
 .leaderboard-toolbar,
 .leaderboard-card-header {
 flex-direction: column;
 align-items: stretch;
 }

 .leaderboard-header h1 {
 font-size: 32px;
 letter-spacing: -0.96px;
 }

 .metric-tabs,
 .refresh-button {
 width: 100%;
 }

 .metric-tab {
 flex: 1;
 }
}
</style>
