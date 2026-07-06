import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UserDashboardCharts from '../UserDashboardCharts.vue'

const messages: Record<string, string> = {
  'dashboard.timeRange': 'Time Range',
  'dashboard.granularity': 'Granularity',
  'dashboard.day': 'Day',
  'dashboard.hour': 'Hour',
  'dashboard.modelDistribution': 'Model Distribution',
  'dashboard.model': 'Model',
  'dashboard.requests': 'Requests',
  'dashboard.tokens': 'Tokens',
  'dashboard.actual': 'Actual',
  'dashboard.standard': 'Standard',
  'dashboard.noDataAvailable': 'No data available',
  'common.refresh': 'Refresh',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('vue-chartjs', () => ({
  Doughnut: {
    props: ['data'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
}))

describe('UserDashboardCharts', () => {
  const baseProps = {
    startDate: '2026-07-05',
    endDate: '2026-07-06',
    granularity: 'day',
    trend: [],
  }

  it('does not show the model table shell while initial model data is loading', () => {
    const wrapper = mount(UserDashboardCharts, {
      props: {
        ...baseProps,
        loading: true,
        models: [],
      },
      global: {
        stubs: {
          DateRangePicker: true,
          Select: true,
          LoadingSpinner: true,
          TokenUsageTrend: true,
        },
      },
    })

    expect(wrapper.find('loading-spinner-stub').exists()).toBe(true)
    expect(wrapper.find('thead').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('No data available')
  })

  it('shows a single empty state instead of table headers when there are no models', () => {
    const wrapper = mount(UserDashboardCharts, {
      props: {
        ...baseProps,
        loading: false,
        models: [],
      },
      global: {
        stubs: {
          DateRangePicker: true,
          Select: true,
          LoadingSpinner: true,
          TokenUsageTrend: true,
        },
      },
    })

    expect(wrapper.text()).toContain('No data available')
    expect(wrapper.find('thead').exists()).toBe(false)
  })
})
