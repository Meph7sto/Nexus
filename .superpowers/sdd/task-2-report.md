# Task 2 Report: Interaction Service, Admin API, Cleanup, And Wiring

Status: DONE

Commit:
- `cf3875b7 feat: expose admin usage interaction details`

Implemented:
- Added `UsageInteractionService` with recording settings, `RecordComplete`, `RecordFailed`, `GetByUsageLogID`, and `CleanupExpired`.
- Recording remains default-off through settings; raw JSON is stripped unless raw storage is enabled; payload maps are redacted before persistence.
- Added admin endpoint `GET /api/v1/admin/usage/:id/interaction`.
- Missing interaction rows return `{ "exists": false, "reason": "not_recorded" }`.
- Successful interaction detail responses return `{ "exists": true, "interaction": ... }`; raw payload inclusion follows `include_raw`.
- Registered the admin usage interaction route after `/search-api-keys` and before cleanup task routes.
- Wired `NewUsageInteractionRepository` and `NewUsageInteractionService`, and updated generated server wiring to pass `usageInteractionService` into `admin.NewUsageHandler`.

TDD Evidence:
- RED service run:
  - `go test ./internal/service -run TestUsageInteractionService -count=1`
  - Failed as expected with `undefined: NewUsageInteractionService` and `undefined: ErrUsageInteractionNotFound`.
- GREEN service run:
  - `go test ./internal/service -run TestUsageInteractionService -count=1`
  - Passed.
- RED handler run:
  - `go test ./internal/handler/admin -run GetInteraction -count=1`
  - Failed as expected because `NewUsageHandler` did not accept the interaction service and `GetInteraction` did not exist.
- GREEN handler run:
  - `go test ./internal/handler/admin -run GetInteraction -count=1`
  - Passed.

Verification:
- `go test ./internal/service ./internal/handler/admin -run 'UsageInteraction|GetInteraction' -count=1`
  - Passed.
- `go test ./cmd/server -run '^$' -count=1`
  - Passed; compile-only check for generated wiring.
- `git diff --check`
  - Passed with no whitespace issues.

Self-Review:
- Confirmed regular user usage handlers were not given interaction detail APIs.
- Confirmed admin route ordering matches the brief.
- Confirmed repository and service providers are registered and generated wiring constructs the admin usage handler with `usageInteractionService`.
- Confirmed not-found repository results are normalized to `service.ErrUsageInteractionNotFound`.
- Confirmed raw payload fields are cleared when `usage_interaction_store_raw_enabled` is false.
- Confirmed retention `0` keeps indefinitely in `CleanupExpired`.

Concerns:
- None.

---

## Task 2 Review Fix Report

Status: DONE

Implemented:
- Redacted `raw_request_json` and `raw_response_json` through `RedactUsageInteractionPayload` when raw storage is enabled, and included raw credential keys in the persisted redaction metadata.
- Propagated usage interaction settings read errors from `RecordComplete`, `RecordFailed`, and `CleanupExpired` instead of silently falling back to defaults.
- Wired usage interaction retention cleanup into the existing usage cleanup worker via `UsageCleanupService`.
- Added a nil-service guard for admin `GetInteraction`, returning service unavailable instead of panicking.

TDD Evidence:
- RED service run:
  - `go test ./internal/service -run 'TestUsageInteractionService_(StoreRawEnabledRedactsRawCredentialFields|SettingsReadErrorsPropagate)|TestUsageCleanupServiceRunOnceCleansExpiredUsageInteractionsWhenNoTask' -count=1`
  - Failed as expected because raw JSON was not redacted and the cleanup service had no interaction cleanup seam.
- RED handler run:
  - `go test ./internal/handler/admin -run TestUsageHandlerGetInteractionReturnsUnavailableWhenServiceMissing -count=1`
  - Failed as expected with status 200 from the missing-service path.
- GREEN focused runs:
  - `go test ./internal/service -run 'TestUsageInteractionService_(StoreRawEnabledRedactsRawCredentialFields|SettingsReadErrorsPropagate)|TestUsageCleanupServiceRunOnceCleansExpiredUsageInteractionsWhenNoTask' -count=1`
  - `go test ./internal/handler/admin -run TestUsageHandlerGetInteractionReturnsUnavailableWhenServiceMissing -count=1`
  - Both passed.

Verification:
- `go test ./internal/service ./internal/handler/admin -run 'UsageInteraction|GetInteraction' -count=1`
  - Passed.
- `go test ./internal/service -run 'UsageCleanupServiceRunOnceCleansExpiredUsageInteractionsWhenNoTask|UsageCleanupServiceRunOnce' -count=1`
  - Passed.
- `go test ./internal/service -run UsageCleanupService -count=1`
  - Passed.
- `go test ./cmd/server -run '^$' -count=1`
  - Passed.
- `git diff --check`
  - Passed with no whitespace issues.

Concerns:
- None.
