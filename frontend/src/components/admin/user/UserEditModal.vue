<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.editUser')"
    width="normal"
    @close="$emit('close')"
  >
    <form v-if="user" id="edit-user-form" @submit.prevent="handleUpdateUser" class="space-y-5">
      <div>
        <label class="input-label">{{ t('admin.users.email') }}</label>
        <input v-model="form.email" type="email" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.password') }}</label>
        <div class="flex gap-2">
          <div class="relative flex-1">
            <input v-model="form.password" type="text" class="input pr-10" :placeholder="t('admin.users.enterNewPassword')" />
            <button v-if="form.password" type="button" @click="copyPassword" class="absolute right-2 top-1/2 -translate-y-1/2 rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700" :class="passwordCopied ? 'text-green-500' : 'text-gray-400'">
              <svg v-if="passwordCopied" class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
              <svg v-else class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" /></svg>
            </button>
          </div>
          <button type="button" @click="generatePassword" class="btn btn-secondary px-3">
            <Icon name="refresh" size="md" />
          </button>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.username') }}</label>
        <input v-model="form.username" type="text" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.notes') }}</label>
        <textarea v-model="form.notes" rows="3" class="input"></textarea>
      </div>
      <div v-if="canEditAdminPermissions">
        <label class="input-label">{{ t('admin.users.columns.role') }}</label>
        <select v-model="form.role" data-test="role-select" class="input">
          <option value="user">{{ t('admin.users.roles.user') }}</option>
          <option value="admin">{{ t('admin.users.roles.admin') }}</option>
          <option value="super_admin">{{ t('admin.users.roles.superAdmin') }}</option>
        </select>
      </div>
      <section v-if="canEditAdminPermissions && form.role === 'admin'" class="space-y-3">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">Administrator Permissions</h3>
        <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
          <table class="min-w-full text-sm">
            <thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-3 py-2 text-left">Module</th>
                <th v-for="action in ADMIN_PERMISSION_ACTIONS" :key="action" class="px-3 py-2 text-center">
                  {{ action }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="row in assignablePermissionRows" :key="row.resource">
                <td class="px-3 py-2 font-medium text-gray-700 dark:text-dark-200">{{ row.label }}</td>
                <td v-for="action in ADMIN_PERMISSION_ACTIONS" :key="action" class="px-3 py-2 text-center">
                  <input
                    v-if="row.actions.includes(action)"
                    type="checkbox"
                    class="h-4 w-4 rounded border-gray-300"
                    :data-test="`perm-${row.resource}-${action}`"
                    :checked="hasPermission(row.resource, action)"
                    @change="setPermission(row.resource, action, ($event.target as HTMLInputElement).checked)"
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
      <p v-if="canEditAdminPermissions && form.role === 'super_admin'" class="input-hint">
        Super administrators have all permissions.
      </p>
      <div>
        <label class="input-label">{{ t('admin.users.columns.concurrency') }}</label>
        <input v-model.number="form.concurrency" type="number" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.form.rpmLimit') }}</label>
        <input
          v-model.number="form.rpm_limit"
          type="number"
          min="0"
          step="1"
          class="input"
          :placeholder="t('admin.users.form.rpmLimitPlaceholder')"
        />
        <p class="input-hint">{{ t('admin.users.form.rpmLimitHint') }}</p>
      </div>
      <UserAttributeForm v-model="form.customAttributes" :user-id="user?.id" />
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" type="button" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="edit-user-form" :disabled="submitting" class="btn btn-primary">
          {{ submitting ? t('admin.users.updating') : t('common.update') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import { ADMIN_PERMISSION_ACTIONS } from '@/utils/adminPermissions'
import type { AdminPermission, AdminPermissionAction, AdminPermissionResource, AdminUser, UserAttributeValuesMap, UserRole } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import UserAttributeForm from '@/components/user/UserAttributeForm.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ show: boolean, user: AdminUser | null }>()
const emit = defineEmits(['close', 'success'])
const { t } = useI18n(); const appStore = useAppStore(); const authStore = useAuthStore(); const { copyToClipboard } = useClipboard()

const submitting = ref(false); const passwordCopied = ref(false)
const canEditAdminPermissions = computed(() => authStore.isSuperAdmin)
const fullActions: AdminPermissionAction[] = [...ADMIN_PERMISSION_ACTIONS]
const assignablePermissionRows: Array<{ resource: AdminPermissionResource; label: string; actions: AdminPermissionAction[] }> = [
  { resource: 'dashboard', label: 'Dashboard', actions: ['view', 'execute'] },
  { resource: 'ops', label: 'Ops', actions: fullActions },
  { resource: 'users', label: 'Users', actions: fullActions },
  { resource: 'groups', label: 'Groups', actions: fullActions },
  { resource: 'channels', label: 'Channels', actions: fullActions },
  { resource: 'channel_monitor', label: 'Channel Monitor', actions: fullActions },
  { resource: 'subscriptions', label: 'Subscriptions', actions: fullActions },
  { resource: 'accounts', label: 'Accounts', actions: fullActions },
  { resource: 'announcements', label: 'Announcements', actions: fullActions },
  { resource: 'proxies', label: 'Proxies', actions: fullActions },
  { resource: 'risk_control', label: 'Risk Control', actions: fullActions },
  { resource: 'redeem_codes', label: 'Redeem Codes', actions: fullActions },
  { resource: 'promo_codes', label: 'Promo Codes', actions: fullActions },
  { resource: 'affiliates', label: 'Affiliates', actions: fullActions },
  { resource: 'orders', label: 'Orders', actions: fullActions },
  { resource: 'usage', label: 'Usage', actions: ['view', 'export', 'delete', 'execute'] },
  { resource: 'data_management', label: 'Data Management', actions: fullActions },
  { resource: 'backups', label: 'Backups', actions: fullActions },
  { resource: 'user_attributes', label: 'User Attributes', actions: fullActions },
  { resource: 'error_passthrough_rules', label: 'Error Passthrough Rules', actions: fullActions },
  { resource: 'tls_fingerprint_profiles', label: 'TLS Fingerprint Profiles', actions: fullActions },
  { resource: 'scheduled_tests', label: 'Scheduled Tests', actions: fullActions },
]
const form = reactive({
  email: '',
  password: '',
  username: '',
  notes: '',
  role: 'user' as UserRole,
  concurrency: 1,
  rpm_limit: 0,
  customAttributes: {} as UserAttributeValuesMap,
  admin_permissions: [] as AdminPermission[],
})

watch(() => props.user, (u) => {
  if (u) {
    Object.assign(form, {
      email: u.email,
      password: '',
      username: u.username || '',
      notes: u.notes || '',
      role: u.role,
      concurrency: u.concurrency,
      rpm_limit: u.rpm_limit ?? 0,
      customAttributes: {},
      admin_permissions: (u.admin_permissions ?? []).map((perm) => ({
        resource: perm.resource,
        actions: [...perm.actions],
      })),
    })
    passwordCopied.value = false
  }
}, { immediate: true })

const generatePassword = () => {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  let p = ''; for (let i = 0; i < 16; i++) p += chars.charAt(Math.floor(Math.random() * chars.length))
  form.password = p
}
const copyPassword = async () => {
  if (form.password && await copyToClipboard(form.password, t('admin.users.passwordCopied'))) {
    passwordCopied.value = true; setTimeout(() => passwordCopied.value = false, 2000)
  }
}

function hasPermission(resource: AdminPermissionResource, action: AdminPermissionAction): boolean {
  return form.admin_permissions.some((perm) => perm.resource === resource && perm.actions.includes(action))
}

function setPermission(resource: AdminPermissionResource, action: AdminPermissionAction, checked: boolean): void {
  let perm = form.admin_permissions.find((item) => item.resource === resource)
  if (!perm) {
    perm = { resource, actions: [] }
    form.admin_permissions.push(perm)
  }
  const actions = new Set(perm.actions)
  if (checked) {
    actions.add(action)
    if (action !== 'view') actions.add('view')
  } else {
    actions.delete(action)
    if (action === 'view') actions.clear()
  }
  perm.actions = Array.from(actions)
  const rowIndex = form.admin_permissions.findIndex((item) => item.resource === resource)
  if (perm.actions.length === 0 && rowIndex >= 0) {
    form.admin_permissions.splice(rowIndex, 1)
  }
}

function buildAdminPermissionsPayload(): AdminPermission[] {
  if (form.role !== 'admin') return []
  return form.admin_permissions.map((perm) => ({ resource: perm.resource, actions: [...perm.actions] }))
}

const handleUpdateUser = async () => {
  if (!props.user) return
  if (!form.email.trim()) {
    appStore.showError(t('admin.users.emailRequired'))
    return
  }
  if (form.concurrency < 1) {
    appStore.showError(t('admin.users.concurrencyMin'))
    return
  }
  submitting.value = true
  try {
    const data: any = { email: form.email, username: form.username, notes: form.notes, concurrency: form.concurrency, rpm_limit: form.rpm_limit }
    if (form.password.trim()) data.password = form.password.trim()
    if (canEditAdminPermissions.value) {
      data.role = form.role
      data.admin_permissions = buildAdminPermissionsPayload()
    }
    await adminAPI.users.update(props.user.id, data)
    if (Object.keys(form.customAttributes).length > 0) await adminAPI.userAttributes.updateUserAttributeValues(props.user.id, form.customAttributes)
    appStore.showSuccess(t('admin.users.userUpdated'))
    emit('success'); emit('close')
  } catch (e: any) {
    appStore.showError(e.response?.data?.detail || t('admin.users.failedToUpdate'))
  } finally { submitting.value = false }
}
</script>
