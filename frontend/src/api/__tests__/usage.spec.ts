import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
 get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
 apiClient: {
 get,
 },
}))

import { usageAPI } from '@/api/usage'

describe('usage api', () => {
 beforeEach(() => {
 get.mockReset()
 get.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, pages: 1 } })
 })

 it('requests the global leaderboard with the selected metric and pagination', async () => {
 await usageAPI.getRanking({
 rank_by: 'cost',
 start_date: '2026-03-01',
 end_date: '2026-03-02',
 page: 9,
 page_size: 20,
 })

 expect(get).toHaveBeenCalledWith('/usage/ranking', {
 params: {
 rank_by: 'cost',
 start_date: '2026-03-01',
 end_date: '2026-03-02',
 page: 9,
 page_size: 20,
 },
 })
 })
})
