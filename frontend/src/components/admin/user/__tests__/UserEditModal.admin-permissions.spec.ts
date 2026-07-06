import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UserEditModal from '../UserEditModal.vue'
import { adminAPI } from '@/api/admin'
import type { AdminUser } from '@/types'

const authStoreMock = vi.hoisted(() => ({
  isSuperAdmin: false,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStoreMock,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn().mockResolvedValue(true) }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: { update: vi.fn() },
    userAttributes: { updateUserAttributeValues: vi.fn() },
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const user: AdminUser = {
  id: 7,
  email: 'target@example.com',
  username: 'target',
  role: 'admin',
  balance: 0,
  concurrency: 1,
  rpm_limit: 0,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  notes: '',
  admin_permissions: [{ resource: 'users', actions: ['view'] }],
}

function mountModal() {
  return mount(UserEditModal, {
    props: { show: true, user },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        UserAttributeForm: true,
        Icon: true,
      },
    },
  })
}

describe('UserEditModal administrator permissions', () => {
  beforeEach(() => {
    authStoreMock.isSuperAdmin = false
    vi.mocked(adminAPI.users.update).mockReset()
    vi.mocked(adminAPI.users.update).mockResolvedValue(user)
  })

  it('shows role and permission controls only to super administrators', () => {
    const limited = mountModal()
    expect(limited.find('[data-test="role-select"]').exists()).toBe(false)
    expect(limited.text()).not.toContain('Administrator Permissions')

    authStoreMock.isSuperAdmin = true
    const superAdmin = mountModal()
    expect(superAdmin.find('[data-test="role-select"]').exists()).toBe(true)
    expect(superAdmin.text()).toContain('Administrator Permissions')
  })

  it('auto-checks view when update is checked and submits permissions', async () => {
    authStoreMock.isSuperAdmin = true
    const wrapper = mountModal()

    await wrapper.find('[data-test="role-select"]').setValue('admin')
    await wrapper.find('[data-test="perm-users-update"]').setValue(true)
    await wrapper.find('form').trigger('submit.prevent')

    expect(adminAPI.users.update).toHaveBeenCalledWith(7, expect.objectContaining({
      role: 'admin',
      admin_permissions: expect.arrayContaining([
        expect.objectContaining({ resource: 'users', actions: expect.arrayContaining(['view', 'update']) }),
      ]),
    }))
  })

  it('selects every assignable permission when select all is checked', async () => {
    authStoreMock.isSuperAdmin = true
    const wrapper = mountModal()

    await wrapper.find('[data-test="perm-select-all"]').setValue(true)
    await wrapper.find('form').trigger('submit.prevent')

    expect(adminAPI.users.update).toHaveBeenCalledWith(7, expect.objectContaining({
      role: 'admin',
      admin_permissions: expect.arrayContaining([
        expect.objectContaining({
          resource: 'dashboard',
          actions: expect.arrayContaining(['view', 'execute']),
        }),
        expect.objectContaining({
          resource: 'users',
          actions: expect.arrayContaining(['view', 'create', 'update', 'delete', 'export', 'execute']),
        }),
        expect.objectContaining({
          resource: 'usage',
          actions: expect.arrayContaining(['view', 'delete', 'export', 'execute']),
        }),
      ]),
    }))
  })
})
