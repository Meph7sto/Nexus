import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import UsageTable from '@/components/admin/usage/UsageTable.vue'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'
import UsageView from '../UsageView.vue'

const { listUsage, getStats, getSnapshotV2, getModelStats, listErrorLogs, push, route } = vi.hoisted(() => {
 vi.stubGlobal('localStorage', {
 getItem: vi.fn(() => null),
 setItem: vi.fn(),
 removeItem: vi.fn(),
 })

 return {
 listUsage: vi.fn(),
 getStats: vi.fn(),
 getSnapshotV2: vi.fn(),
 getModelStats: vi.fn(),
 listErrorLogs: vi.fn(),
 push: vi.fn(),
 route: {
 query: {},
 fullPath: '/admin/usage?page=2',
 },
 }
})

vi.mock('@/api/admin', () => ({
 adminAPI: {
 usage: {
 list: listUsage,
 getStats,
 },
 dashboard: {
 getSnapshotV2,
 getModelStats,
 },
 users: {
 getById: vi.fn(),
 },
 },
}))

vi.mock('@/api/admin/usage', () => ({
 adminUsageAPI: {
 list: vi.fn(),
 },
}))

vi.mock('@/api/admin/ops', () => ({
 listErrorLogs,
}))

vi.mock('@/stores/app', () => ({
 useAppStore: () => ({
 showError: vi.fn(),
 showWarning: vi.fn(),
 showSuccess: vi.fn(),
 showInfo: vi.fn(),
 }),
}))

vi.mock('@/utils/format', () => ({
 formatReasoningEffort: (value: string | null | undefined) => value ?? '-',
}))

vi.mock('vue-router', () => ({
 useRoute: () => route,
 useRouter: () => ({ push }),
}))

const localeValue = (messages: Record<string, any>, key: string) =>
 key.split('.').reduce((value, segment) => value?.[segment], messages)

vi.mock('vue-i18n', async () => {
 const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
 return {
 ...actual,
 useI18n: () => ({
 t: (key: string) => key,
 }),
 }
})

describe('UsageTable interaction details action', () => {
 it('emits openInteraction with the row id', async () => {
 const wrapper = mount(UsageTable, {
 props: {
 data: [{ id: 42, user_id: 1, api_key_id: 1, account_id: 1, request_id: 'req', model: 'm', stream: false, created_at: '2026-07-07T00:00:00Z' }],
 columns: [{ key: 'actions', label: 'Actions' }],
 loading: false,
 },
 global: {
 stubs: {
 DataTable: { template: '<div><slot name="cell-actions" :row="$props.data[0]" /></div>', props: ['data'] },
 EmptyState: true,
 IpGeoCell: true,
 Icon: true,
 },
 mocks: { $t: (key: string) => key },
 },
 })
 await wrapper.get('[data-test="usage-interaction-link"]').trigger('click')
 expect(wrapper.emitted('openInteraction')?.[0]).toEqual([42])
 })
})

describe('UsageView interaction details navigation', () => {
 beforeEach(() => {
 vi.useFakeTimers()
 listUsage.mockReset()
 getStats.mockReset()
 getSnapshotV2.mockReset()
 getModelStats.mockReset()
 listErrorLogs.mockReset()
 push.mockReset()
 route.fullPath = '/admin/usage?page=2'

 listUsage.mockResolvedValue({ items: [], total: 0, pages: 0 })
 getStats.mockResolvedValue({
 total_requests: 0,
 total_input_tokens: 0,
 total_output_tokens: 0,
 total_cache_tokens: 0,
 total_tokens: 0,
 total_cost: 0,
 total_actual_cost: 0,
 average_duration_ms: 0,
 })
 getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
 getModelStats.mockResolvedValue({ models: [] })
 listErrorLogs.mockResolvedValue({ items: [], total: 0, pages: 0 })
 })

 afterEach(() => {
 vi.useRealTimers()
 })

 it('pushes the interaction route with the current page as return query', async () => {
 const UsageTableStub = {
 emits: ['openInteraction'],
 template: '<button data-test="open-interaction" @click="$emit(\'openInteraction\', 42)">open</button>',
 }

 const wrapper = mount(UsageView, {
 global: {
 stubs: {
 AppLayout: { template: '<div><slot /></div>' },
 UsageStatsCards: true,
 UsageFilters: { template: '<div><slot name="after-reset" /></div>' },
 UsageTable: UsageTableStub,
 UsageExportProgress: true,
 UsageCleanupDialog: true,
 UserBalanceHistoryModal: true,
 Pagination: true,
 Select: true,
 DateRangePicker: true,
 Icon: true,
 TokenUsageTrend: true,
 ModelDistributionChart: true,
 GroupDistributionChart: true,
 EndpointDistributionChart: true,
 OpsErrorLogTable: true,
 OpsErrorDetailModal: true,
 },
 },
 })

 await wrapper.get('[data-test="open-interaction"]').trigger('click')

 expect(push).toHaveBeenCalledWith({
 name: 'AdminUsageInteraction',
 params: { id: '42' },
 query: { return: '/admin/usage?page=2' },
 })
 })
})

describe('Usage interaction locale coverage', () => {
 it.each([
 'usage.interactionDetails',
 'admin.usageInteraction.title',
 'admin.usageInteraction.description',
 'admin.usageInteraction.redacted',
 'admin.usageInteraction.notFound',
 'admin.usageInteraction.failedToLoad',
 'admin.usageInteraction.failedToLoadRaw',
 'admin.usageInteraction.tabs.input',
 'admin.usageInteraction.tabs.output',
 'admin.usageInteraction.tabs.parameters',
 'admin.usageInteraction.tabs.routing',
 'admin.usageInteraction.tabs.raw',
 'admin.usageInteraction.sections.input',
 'admin.usageInteraction.sections.output',
 'admin.usageInteraction.sections.parameters',
 'admin.usageInteraction.sections.routing',
 'admin.usageInteraction.sections.rawRequest',
 'admin.usageInteraction.sections.rawResponse',
 ])('defines %s in English and Chinese locales', (key) => {
 expect(localeValue(en, key)).toEqual(expect.any(String))
 expect(localeValue(zh, key)).toEqual(expect.any(String))
 })
})
