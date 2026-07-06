# Admin Permissions Design

## Context

Nexus currently has two user roles: `user` and `admin`. The backend treats every `admin` user as fully privileged for `/api/v1/admin/*`, and the frontend uses the same role check to show the admin router and sidebar. This feature splits the current administrator into two levels:

- `super_admin`: full administrator, equivalent to today's `admin`.
- `admin`: limited administrator whose visible pages and allowed actions are explicitly configured by a super administrator.

The first version uses direct per-user permissions, not reusable role templates.

## Goals

- Preserve current administrator access during upgrade.
- Allow super administrators to promote a user to limited administrator.
- Allow super administrators to configure which admin pages a limited administrator can view.
- Allow super administrators to configure common actions on those pages: `view`, `create`, `update`, `delete`, `export`, and `execute`.
- Enforce permissions on the backend. Frontend hiding is an experience improvement, not the security boundary.
- Put permission editing inside the existing admin user edit experience.

## Non-Goals

- Reusable permission templates or named roles.
- Per-button custom permission keys beyond the common actions.
- Fine-grained custom-page permissions.
- Allowing limited administrators to manage administrator permissions.

## Data Model

Extend `users.role` to support:

- `user`
- `admin`
- `super_admin`

Add persistent administrator permission storage. The preferred implementation is a separate table, `admin_permissions`, with one row per user and resource:

- `id`
- `user_id`
- `resource`
- `actions`
- timestamps

`actions` stores the enabled common actions for that resource. It can be represented as JSON, an array column where supported, or a normalized child table if the existing migration style favors that. The service layer should expose it as structured data:

```json
{
  "resource": "users",
  "actions": ["view", "create", "update", "delete"]
}
```

`super_admin` users do not need permission rows and always have all permissions. `admin` users without permission rows have no admin page access by default.

## Migration

The database migration must preserve current access:

```sql
UPDATE users SET role = 'super_admin' WHERE role = 'admin';
```

After migration, newly created or edited users with role `admin` are limited administrators. They require explicit permission rows before they can use admin pages.

## Permission Registry

Create a central backend permission registry that maps admin resources to labels and supported actions. Initial resources should match the current admin navigation and route groups:

- `dashboard`
- `ops`
- `users`
- `groups`
- `channels`
- `channel_monitor`
- `subscriptions`
- `accounts`
- `announcements`
- `proxies`
- `risk_control`
- `redeem_codes`
- `promo_codes`
- `affiliates`
- `orders`
- `usage`
- `settings`
- `system`
- `data_management`
- `backups`
- `user_attributes`
- `error_passthrough_rules`
- `tls_fingerprint_profiles`
- `scheduled_tests`
- `admin_permissions`

High-risk resources are super-admin-only in the first version:

- `settings`
- `system`
- `admin_permissions`

They should not be assignable to limited administrators.

## Backend Authorization

Update admin authentication so both `super_admin` and `admin` can authenticate for admin routes:

- `super_admin`: passes admin authentication and all permission checks.
- `admin`: passes admin authentication, then requires explicit route-level permission.
- `user`: rejected from admin routes.

Add middleware similar to:

```go
RequireAdminPermission(resource, action)
```

Admin routes should declare permissions when registered:

- `GET /admin/users` -> `users:view`
- `POST /admin/users` -> `users:create`
- `PUT /admin/users/:id` -> `users:update`
- `DELETE /admin/users/:id` -> `users:delete`
- `GET /admin/usage` -> `usage:view`
- export endpoints -> `*:export`
- operational actions such as test, sync, refresh, reset, recover, batch processing -> `*:execute`

Authorization rules:

- Missing authentication returns `401`.
- Non-admin roles return `403`.
- Limited administrators without the required permission return `403` with code `ADMIN_PERMISSION_DENIED`.
- Limited administrators cannot create or edit `super_admin` users.
- Limited administrators cannot edit administrator permissions.
- Limited administrators cannot elevate themselves or other users.
- Super administrators can edit roles and permissions.

Admin API Key behavior should continue to bind to the first active super administrator. The migration should guarantee at least one super administrator when an admin existed before upgrade.

## Permission APIs

Current-user responses should include the user's role and admin permission summary:

- Login response
- Current user response
- Token refresh path if it returns user data

Admin user list and detail responses should include role. User detail should include permissions when the viewer is a super administrator and the target user is an administrator.

Saving permissions should be implemented through the existing admin user edit/update flow, because the UI belongs inside `admin/users`. The service boundary must apply role and permission changes in one transaction so a user is not left in a partially elevated state.

Recommended API shape for updates:

```json
{
  "role": "admin",
  "admin_permissions": [
    { "resource": "users", "actions": ["view", "update"] },
    { "resource": "usage", "actions": ["view", "export"] }
  ]
}
```

When role changes to `user`, delete that user's admin permission rows. When role changes to `super_admin`, permission rows may be deleted or ignored; deleting keeps the state clearer.

## Frontend Authorization

Update frontend types and auth store:

- `role: 'user' | 'admin' | 'super_admin'`
- `isSuperAdmin`
- `isAdminLike`
- `adminPermissions`
- `canAdmin(resource, action)`

Router metadata should declare admin resources for admin pages. Entering an admin page requires `view` for limited administrators. Super administrators can enter all admin pages.

Routing behavior:

- `super_admin` goes to `/admin/dashboard` after login.
- `admin` goes to the first admin page where they have `view`; if none exists, show an admin no-permission state or route them to the normal user dashboard.
- `user` remains blocked from admin pages.

Sidebar behavior:

- `super_admin` sees all admin menu items.
- `admin` sees only admin menu items where `canAdmin(resource, 'view')` is true.
- Existing feature flags such as payment, ops, risk control, and channel monitor still apply.

Page button behavior:

- Add buttons require `create`.
- Edit and save controls require `update`.
- Delete controls require `delete`.
- Export controls require `export`.
- Test, sync, refresh, reset, recover, and batch operational controls require `execute`.

The first version hides disallowed controls. Backend checks still reject direct API calls.

## Permission Editing UI

The permission editor belongs inside the existing admin users edit experience.

Only `super_admin` users can see or edit the administrator permission section.

The user edit form should support three roles:

- Normal user
- Administrator
- Super administrator

When role is `admin`, show a permission matrix grouped by admin page/module:

```text
Module          View  Create  Update  Delete  Export  Execute
Users           yes   yes     yes     yes     no      no
Accounts        yes   yes     yes     yes     yes     yes
Settings        super administrator only
System          super administrator only
```

UI rules:

- Checking any action automatically checks `view`.
- Unchecking `view` clears every action for that resource.
- Super-admin-only resources are visible as unavailable or omitted from assignable rows.
- When role is `super_admin`, show explanatory text that the user has all permissions.
- When role is `user`, hide the permission matrix.
- The user list can display role tags: normal user, administrator, super administrator.

## Error Handling

Use existing error response patterns where possible.

Suggested errors:

- `403 ADMIN_PERMISSION_DENIED`: limited administrator lacks the required permission.
- `403 SUPER_ADMIN_REQUIRED`: action is reserved for super administrators.
- `400 INVALID_ADMIN_PERMISSION`: unknown resource or unsupported action.
- `400 INVALID_ROLE_TRANSITION`: invalid or forbidden role change.

## Testing

Backend tests:

- Migration upgrades existing `admin` users to `super_admin`.
- `super_admin` can access representative routes from every admin module.
- `admin` with `users:view` can list users.
- `admin` without `users:update` cannot update users.
- `admin` cannot create or edit `super_admin` users.
- `admin` cannot edit permissions.
- Missing permissions return `403 ADMIN_PERMISSION_DENIED`.
- Current-user responses include permissions for limited administrators.
- Updating a user's role and permissions writes the expected permission rows.

Frontend tests:

- `super_admin` sees all admin navigation allowed by feature flags.
- `admin` sees only resources with `view`.
- Route guard blocks admin pages without `view`.
- The user edit permission matrix is visible only to super administrators.
- A view-only administrator cannot see create, edit, delete, export, or execute controls.
- Checking an action auto-checks `view`; unchecking `view` clears the row.

## Rollout Notes

This feature touches authentication, route registration, user management, migrations, and frontend navigation. Implement it in vertical slices:

1. Add roles, permission model, migration, and service methods.
2. Add backend permission middleware and protect representative route groups.
3. Return permission summaries in auth/current-user responses.
4. Update frontend auth, router, and sidebar behavior.
5. Add the permission editor to admin user editing.
6. Expand route permission coverage and button-level checks across admin pages.
