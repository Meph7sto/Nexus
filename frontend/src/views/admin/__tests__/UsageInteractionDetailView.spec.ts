import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UsageInteractionDetailView from '../UsageInteractionDetailView.vue'

const { getInteraction, push, route } = vi.hoisted(() => ({
 getInteraction: vi.fn(),
 push: vi.fn(),
 route: {
 params: { id: '42' },
 query: { return: '/admin/usage?page=2' },
 },
}))

vi.mock('@/api/admin/usage', () => ({
 adminUsageAPI: {
 getInteraction,
 },
 default: {
 getInteraction,
 },
}))

vi.mock('vue-router', () => ({
 useRoute: () => route,
 useRouter: () => ({ push }),
}))

vi.mock('vue-i18n', async () => {
 const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
 return {
 ...actual,
 useI18n: () => ({
 t: (key: string) => key,
 }),
 }
})

const interactionWithoutRaw = {
 exists: true,
 interaction: {
 id: 7,
 usage_log_id: 42,
 request_id: 'req-42',
 created_at: '2026-07-07T00:00:00Z',
 capture_status: 'complete',
 capture_error: null,
 request_content: {
 messages: [
 { role: 'user', content: 'full input prompt with no truncation' },
 ],
 },
 response_content: {
 choices: [
 { message: { content: 'full output text with no summarization' } },
 ],
 },
 request_parameters: { temperature: 0.2, max_tokens: 4096 },
 routing_context: { upstream_model: 'gpt-5.3-codex', channel_id: 3 },
 raw_available: true,
 raw_request_json: null,
 raw_response_json: null,
 redaction_applied: true,
 redaction_keys: ['authorization'],
 },
}

const interactionWithRaw = {
 exists: true,
 interaction: {
 ...interactionWithoutRaw.interaction,
 raw_request_json: {
 messages: [
 { role: 'user', content: 'raw request body preserves every character' },
 ],
 },
 raw_response_json: {
 choices: [
 { message: { content: 'raw response body preserves every character' } },
 ],
 },
 },
}

const mountView = () => mount(UsageInteractionDetailView, {
 global: {
 stubs: {
 AppLayout: { template: '<div><slot /></div>' },
 Icon: true,
 },
 },
})

describe('UsageInteractionDetailView', () => {
 beforeEach(() => {
 getInteraction.mockReset()
 push.mockReset()
 route.params.id = '42'
 route.query.return = '/admin/usage?page=2'
 getInteraction.mockResolvedValue(interactionWithoutRaw)
 })

 it('loads the interaction without raw JSON by default and renders preserved input', async () => {
 const wrapper = mountView()
 await flushPromises()

 expect(getInteraction).toHaveBeenCalledTimes(1)
 expect(getInteraction).toHaveBeenCalledWith(42)
 expect(wrapper.text()).toContain('req-42')
 expect(wrapper.text()).toContain('full input prompt with no truncation')
 expect(wrapper.text()).not.toContain('raw request body preserves every character')
 })

 it('loads and renders raw JSON only after the raw tab is selected', async () => {
 getInteraction
 .mockResolvedValueOnce(interactionWithoutRaw)
 .mockResolvedValueOnce(interactionWithRaw)

 const wrapper = mountView()
 await flushPromises()

 expect(getInteraction).toHaveBeenCalledWith(42)
 expect(wrapper.text()).not.toContain('raw response body preserves every character')

 await wrapper.get('[data-test="usage-interaction-tab-raw"]').trigger('click')
 await flushPromises()

 expect(getInteraction).toHaveBeenLastCalledWith(42, { includeRaw: true })
 expect(wrapper.text()).toContain(JSON.stringify(interactionWithRaw.interaction.raw_request_json, null, 2))
 expect(wrapper.text()).toContain(JSON.stringify(interactionWithRaw.interaction.raw_response_json, null, 2))
 })

 it('returns to the route query return path from the back button', async () => {
 const wrapper = mountView()
 await flushPromises()

 await wrapper.get('[data-test="usage-interaction-back"]').trigger('click')

 expect(push).toHaveBeenCalledWith('/admin/usage?page=2')
 })
})
