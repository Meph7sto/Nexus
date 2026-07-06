import { describe, expect, it } from 'vitest'
import { adminResourceForPath, applyAdminRoutePermissionMeta } from '@/utils/adminPermissions'
import type { RouteRecordRaw } from 'vue-router'

describe('admin permission routing helpers', () => {
 it('treats the OpenAI quota summary page as an accounts resource', () => {
  expect(adminResourceForPath('/admin/openai-quota-summary')).toBe('accounts')
 })

 it('applies accounts view permission meta to the OpenAI quota summary route', () => {
  const routes: RouteRecordRaw[] = [
   {
    path: '/admin/openai-quota-summary',
    component: {},
    meta: {
     requiresAuth: true,
     requiresAdmin: true
    }
   }
  ]

  applyAdminRoutePermissionMeta(routes)

  expect(routes[0].meta?.adminResource).toBe('accounts')
  expect(routes[0].meta?.adminAction).toBe('view')
 })
})
