import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { AdminPermissionResource } from '@/types'

export function useAdminPermissionGate(resource: AdminPermissionResource) {
  let authStore: ReturnType<typeof useAuthStore> | null = null
  try {
    authStore = useAuthStore()
  } catch {
    authStore = null
  }
  const can = (action: 'create' | 'update' | 'delete' | 'export' | 'execute') => (
    authStore ? authStore.canAdmin(resource, action) : true
  )
  const canCreate = computed(() => can('create'))
  const canUpdate = computed(() => can('update'))
  const canDelete = computed(() => can('delete'))
  const canExport = computed(() => can('export'))
  const canExecute = computed(() => can('execute'))

  return {
    canCreate,
    canUpdate,
    canDelete,
    canExport,
    canExecute,
  }
}
