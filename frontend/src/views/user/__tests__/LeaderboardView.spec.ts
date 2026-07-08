import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import LeaderboardView from '../LeaderboardView.vue'

const { getRanking, showError } = vi.hoisted(() => ({
 getRanking: vi.fn(),
 showError: vi.fn(),
}))

vi.mock('@/api', () => ({
 usageAPI: {
 getRanking,
 },
}))

vi.mock('@/stores/app', () => ({
 useAppStore: () => ({ showError }),
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
 getPersistedPageSize: () => 20,
}))

vi.mock('vue-i18n', async () => {
 const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
 return {
 ...actual,
 useI18n: () => ({
 t: (key: string) => key,
 locale: { value: 'en' },
 }),
 }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<span />' }
const DateRangePickerStub = {
 template: '<button data-testid="date-picker" @click="$emit(\'change\', { startDate: \'2026-03-01\', endDate: \'2026-03-02\' })">date</button>',
}
const PaginationStub = {
 props: ['page', 'total', 'pageSize', 'showJump'],
 template: `
 <div data-testid="pagination" :data-show-jump="showJump">
 <button data-testid="jump-page" @click="$emit('update:page', 9)">jump</button>
 <button data-testid="page-size" @click="$emit('update:pageSize', 50)">size</button>
 </div>
 `,
}

describe('LeaderboardView', () => {
 beforeEach(() => {
 getRanking.mockReset()
 showError.mockReset()
 getRanking.mockResolvedValue({
 items: [
 {
 rank: 1,
 nickname: 'a***e',
 email: 'a****@example.com',
 requests: 3,
 total_tokens: 1200,
 total_actual_cost: 0.42,
 },
 ],
 total: 100,
 page: 1,
 page_size: 20,
 pages: 5,
 })
 })

 const mountView = () =>
 mount(LeaderboardView, {
 global: {
 stubs: {
 AppLayout: AppLayoutStub,
 DateRangePicker: DateRangePickerStub,
 Pagination: PaginationStub,
 Icon: IconStub,
 },
 },
 })

 it('loads the cost leaderboard by default', async () => {
 mountView()
 await flushPromises()

 expect(getRanking).toHaveBeenCalledWith(expect.objectContaining({
 rank_by: 'cost',
 page: 1,
 page_size: 20,
 }))
 })

 it('switches to token ranking and resets to the first page', async () => {
 const wrapper = mountView()
 await flushPromises()

 await wrapper.findAll('button').find((button) => button.text().includes('leaderboard.tokens'))!.trigger('click')
 await flushPromises()

 expect(getRanking).toHaveBeenLastCalledWith(expect.objectContaining({
 rank_by: 'tokens',
 page: 1,
 }))
 })

 it('reloads page one when the date range changes', async () => {
 const wrapper = mountView()
 await flushPromises()

 await wrapper.find('[data-testid="date-picker"]').trigger('click')
 await flushPromises()

 expect(getRanking).toHaveBeenLastCalledWith(expect.objectContaining({
 start_date: '2026-03-01',
 end_date: '2026-03-02',
 page: 1,
 }))
 })

 it('enables pagination jump and requests the jumped page', async () => {
 const wrapper = mountView()
 await flushPromises()

 expect(wrapper.find('[data-testid="pagination"]').attributes('data-show-jump')).toBe('true')
 await wrapper.find('[data-testid="jump-page"]').trigger('click')
 await flushPromises()

 expect(getRanking).toHaveBeenLastCalledWith(expect.objectContaining({
 page: 9,
 page_size: 20,
 }))
 })
})
