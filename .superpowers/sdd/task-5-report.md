# Task 5 Report: OpenAI Fast Force Priority Setting

## Status

DONE_WITH_CONCERNS

## Files Changed

- `backend/internal/service/openai_fast_policy_test.go`
- `backend/internal/service/openai_fast_policy_ws_test.go`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/setting_service.go`
- `backend/internal/service/settings_view.go`
- `frontend/src/api/admin/settings.ts`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/views/admin/__tests__/SettingsView.spec.ts`
- `frontend/src/i18n/locales/en.ts`
- `frontend/src/i18n/locales/zh.ts`
- `.superpowers/sdd/task-5-report.md`

## Implementation Summary

- Added backend coverage for `force_priority` rewriting known OpenAI `service_tier` values (`flex`, `auto`, `default`, `scale`, `fast`, `priority`) to `priority` for both HTTP body policy and WS `response.create` policy.
- Added validation coverage proving `SetOpenAIFastPolicySettings` accepts and persists `force_priority`.
- Added `OpenAIFastPolicyActionForcePriority = "force_priority"` and allowed it in OpenAI fast policy validation.
- Implemented `force_priority` behavior in:
  - request-view patch path in `Forward`
  - `applyOpenAIFastPolicyToBody`
  - `applyOpenAIFastPolicyToWSResponseCreate`
- Updated frontend OpenAI fast policy API types, Settings UI select casts/options, and English/Chinese locale labels.
- Added a SettingsView test asserting the force priority option is rendered when OpenAI fast policy settings are loaded.

## TDD Evidence

Red tests run before implementation:

```powershell
cd W:\projects\sub2api\Nexus\backend
go test ./internal/service -run "FastPolicy|FastForce|OpenAIFast"
```

Result: failed as expected at compile time because `OpenAIFastPolicyActionForcePriority` was undefined in the new backend tests.

```powershell
cd W:\projects\sub2api\Nexus
corepack pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts
```

Result: failed as expected. `renders OpenAI fast policy force priority action option` could not find the new option label.

## Verification

Required verification commands:

```powershell
cd W:\projects\sub2api\Nexus\backend
go test ./internal/service -run "FastPolicy|FastForce|OpenAIFast|SettingService"
```

Result: pass. Output summary: `ok github.com/Wei-Shaw/nexus/internal/service`.

Additional uncached backend confirmation:

```powershell
cd W:\projects\sub2api\Nexus\backend
go test ./internal/service -run "FastPolicy|FastForce|OpenAIFast|SettingService" -count=1
```

Result: pass. Output summary: `ok github.com/Wei-Shaw/nexus/internal/service 0.570s`.

```powershell
cd W:\projects\sub2api\Nexus
corepack pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts
```

Result: pass. Output summary: `1 passed`, `21 tests passed`.

```powershell
cd W:\projects\sub2api\Nexus
corepack pnpm --dir frontend run typecheck
```

Result: pass. Output summary: `vue-tsc --noEmit` exited 0.

Other checks:

```powershell
cd W:\projects\sub2api\Nexus
gofmt -w backend/internal/service/openai_fast_policy_test.go backend/internal/service/openai_fast_policy_ws_test.go backend/internal/service/openai_gateway_service.go backend/internal/service/setting_service.go backend/internal/service/settings_view.go
git diff --check
```

Result: pass. `git diff --check` exited 0.

## Concerns

- Existing frontend test output still emits repeated Vue warnings for unresolved `router-link`.
- Existing frontend tooling still warns that Browserslist/caniuse-lite data is 7 months old.
- These warnings pre-existed this task's scope and did not fail the required commands.

## Self-Review Notes

- Kept changes scoped to Task 5 and did not start the Task 6 service split.
- Preserved Nexus-specific Settings payload behavior, including existing usage interaction settings and bulk `openai_fast_policy_settings` save/load semantics.
- Corrected an initial local patch mistake where `force_priority` was briefly added to the Beta policy action whitelist; final diff only permits it in OpenAI fast policy validation.
- Did not commit untracked task brief files.
