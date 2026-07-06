import { describe, expect, it, vi } from 'vitest'
import { getOpenAIQuotaSummary } from '@/api/admin/accounts'
import { apiClient } from '@/api/client'

vi.mock('@/api/client', () => ({
 apiClient: {
 get: vi.fn()
 }
}))

describe('admin accounts OpenAI quota summary API', () => {
 it('passes projection, group, and type query params', async () => {
 vi.mocked(apiClient.get).mockResolvedValueOnce({
 data: { projection_at: '2026-07-06T15:00:00Z', generated_at: '2026-07-06T14:00:00Z', groups: [] }
 })

 await getOpenAIQuotaSummary({
 projection_at: '2026-07-06T15:00:00Z',
 group: 'ungrouped',
 type: 'oauth'
 })

 expect(apiClient.get).toHaveBeenCalledWith('/admin/openai/quota-summary', {
 params: {
 projection_at: '2026-07-06T15:00:00Z',
 group: 'ungrouped',
 type: 'oauth'
 }
 })
 })
})
