import type { RouteRecordRaw } from 'vue-router'
import type { AdminPermissionAction, AdminPermissionResource } from '@/types'

export const ADMIN_PERMISSION_ACTIONS: AdminPermissionAction[] = [
  'view',
  'create',
  'update',
  'delete',
  'export',
  'execute',
]

export function adminResourceForPath(path: string): AdminPermissionResource | null {
  if (path.startsWith('/admin/ops')) return 'ops'
  if (path.startsWith('/admin/users')) return 'users'
  if (path.startsWith('/admin/groups')) return 'groups'
  if (path.startsWith('/admin/channels/monitor')) return 'channel_monitor'
  if (path.startsWith('/admin/channels')) return 'channels'
  if (path.startsWith('/admin/subscriptions')) return 'subscriptions'
  if (path.startsWith('/admin/accounts')) return 'accounts'
  if (path.startsWith('/admin/announcements')) return 'announcements'
  if (path.startsWith('/admin/proxies')) return 'proxies'
  if (path.startsWith('/admin/risk-control')) return 'risk_control'
  if (path.startsWith('/admin/redeem')) return 'redeem_codes'
  if (path.startsWith('/admin/promo-codes')) return 'promo_codes'
  if (path.startsWith('/admin/affiliates')) return 'affiliates'
  if (path.startsWith('/admin/orders')) return 'orders'
  if (path.startsWith('/admin/usage')) return 'usage'
  if (path.startsWith('/admin/settings')) return 'settings'
  if (path.startsWith('/admin/dashboard')) return 'dashboard'
  return null
}

export function applyAdminRoutePermissionMeta(routes: RouteRecordRaw[]): void {
  for (const route of routes) {
    if (route.meta?.requiresAdmin) {
      const resource = adminResourceForPath(route.path)
      if (resource && !route.meta.adminResource) {
        route.meta.adminResource = resource
      }
      if (route.meta.adminResource && !route.meta.adminAction) {
        route.meta.adminAction = 'view'
      }
    }
    if (route.children?.length) {
      applyAdminRoutePermissionMeta(route.children)
    }
  }
}

export function firstAllowedAdminPath(
  routes: RouteRecordRaw[],
  canAdmin: (resource: AdminPermissionResource, action: AdminPermissionAction) => boolean
): string | null {
  for (const route of routes) {
    if (route.meta?.requiresAdmin && route.meta.adminResource) {
      if (canAdmin(route.meta.adminResource, route.meta.adminAction ?? 'view')) {
        return route.path
      }
    }
    if (route.children?.length) {
      const childPath = firstAllowedAdminPath(route.children, canAdmin)
      if (childPath) return childPath
    }
  }
  return null
}
