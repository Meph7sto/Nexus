import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import OpenAIQuotaSummaryView from '../OpenAIQuotaSummaryView.vue'
import { accountsAPI } from '@/api/admin/accounts'
import { groupsAPI } from '@/api/admin/groups'

vi.mock('vue-i18n', () => ({
 useI18n: () => ({
  t: (key: string, params?: Record<string, unknown>) => params ? `${key} ${JSON.stringify(params)}` : key
 })
}))

vi.mock('@/api/admin/accounts', () => ({
 accountsAPI: {
  getOpenAIQuotaSummary: vi.fn()
 }
}))

vi.mock('@/api/admin/groups', () => ({
 groupsAPI: {
  getAll: vi.fn()
 }
}))

vi.mock('@/stores/app', () => ({
 useAppStore: () => ({
  showError: vi.fn()
 })
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
 default: { template: '<div><slot /></div>' }
}))

const response = {
 projection_at: '2026-07-06T15:00:00Z',
 generated_at: '2026-07-06T14:00:00Z',
 groups: [
  {
   group_id: 12,
   group_name: 'OpenAI Main',
   ungrouped: false,
   rows: [
    {
     account_type: 'oauth',
     included_count: 10,
     error_count: 1,
     inactive_count: 2,
     other_excluded_count: 0,
     missing_5h_snapshot_count: 3,
     missing_7d_snapshot_count: 4,
     avg_5h_remaining_percent: 90,
     avg_7d_remaining_percent: 84.5,
     earliest_5h_recovery: {
      account_id: 42,
      account_name: 'openai-01',
      account_type: 'oauth',
      reset_at: '2026-07-06T16:30:00Z',
      remaining_before_percent: 0,
      remaining_after_percent: 100
     },
     earliest_7d_recovery: null
    }
   ]
  }
 ]
}

const groupsResponse = [
 {
  id: 12,
  name: 'OpenAI Main',
  description: null,
  platform: 'openai',
  rate_multiplier: 1,
  is_exclusive: false,
  status: 'active',
  subscription_type: 'standard',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  peak_rate_enabled: false,
  peak_start: '00:00',
  peak_end: '00:00',
  peak_rate_multiplier: 1,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '2026-07-06T00:00:00Z',
  updated_at: '2026-07-06T00:00:00Z',
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: false
 }
]

describe('OpenAIQuotaSummaryView', () => {
 beforeEach(() => {
  vi.mocked(accountsAPI.getOpenAIQuotaSummary).mockReset()
  vi.mocked(accountsAPI.getOpenAIQuotaSummary).mockResolvedValue(response)
  vi.mocked(groupsAPI.getAll).mockReset()
  vi.mocked(groupsAPI.getAll).mockResolvedValue(groupsResponse)
 })

 it('loads and renders grouped summary rows', async () => {
  const wrapper = mount(OpenAIQuotaSummaryView)
  await flushPromises()

  expect(accountsAPI.getOpenAIQuotaSummary).toHaveBeenCalledWith({})
  expect(wrapper.text()).toContain('OpenAI Main')
  expect(wrapper.text()).toContain('oauth')
  expect(wrapper.text()).toContain('90.0%')
  expect(wrapper.text()).toContain('84.5%')
  expect(wrapper.text()).toContain('openai-01')
  expect(wrapper.text()).toContain('#42')
 })

 it('sends a future projection when hours mode is applied', async () => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-07-06T14:00:00Z'))
  try {
   const wrapper = mount(OpenAIQuotaSummaryView)
   await flushPromises()

   await wrapper.get('[data-test="projection-mode-hours"]').trigger('click')
   await wrapper.get('[data-test="projection-amount"]').setValue('2')
   await wrapper.get('[data-test="refresh"]').trigger('click')
   await flushPromises()

   expect(accountsAPI.getOpenAIQuotaSummary).toHaveBeenLastCalledWith({
    projection_at: '2026-07-06T16:00:00.000Z'
   })
  } finally {
   vi.useRealTimers()
  }
 })

 it('sends selected group and account type filters when refreshed', async () => {
  const wrapper = mount(OpenAIQuotaSummaryView)
  await flushPromises()

  expect(groupsAPI.getAll).toHaveBeenCalledWith('openai')

  await wrapper.get('[data-test="group-filter"]').setValue('12')
  await wrapper.get('[data-test="type-filter"]').setValue('oauth')
  await wrapper.get('[data-test="refresh"]').trigger('click')
  await flushPromises()

  expect(accountsAPI.getOpenAIQuotaSummary).toHaveBeenLastCalledWith({
   group: '12',
   type: 'oauth'
  })
 })
})
