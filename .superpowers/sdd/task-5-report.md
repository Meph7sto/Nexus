# Task 5 Report: Frontend Page, Route, Navigation, and Locales

## Status

Complete.

Commit: `f5eccadd feat: add OpenAI quota summary page`

## TDD Evidence

RED:

- Added `frontend/src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts` before production code.
- `pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts` could not start because `pnpm` was not on PATH in this shell.
- Re-ran via Corepack:
  `corepack pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
- Result: failed as expected because `../OpenAIQuotaSummaryView.vue` did not exist.

GREEN:

- Implemented `OpenAIQuotaSummaryView.vue`, route, sidebar entry, and locale keys.
- Re-ran:
  `corepack pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
- Result: 1 test file passed, 2 tests passed.

Final verification:

- `corepack pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
  - Result: 1 test file passed, 2 tests passed.
- `corepack pnpm typecheck`
  - Result: passed.
- `corepack pnpm exec eslint src/views/admin/OpenAIQuotaSummaryView.vue src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts src/router/index.ts src/components/layout/AppSidebar.vue src/i18n/locales/en.ts src/i18n/locales/zh.ts`
  - Result: passed.

## Files Changed

- Created `frontend/src/views/admin/OpenAIQuotaSummaryView.vue`
- Created `frontend/src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
- Modified `frontend/src/router/index.ts`
- Modified `frontend/src/components/layout/AppSidebar.vue`
- Modified `frontend/src/i18n/locales/en.ts`
- Modified `frontend/src/i18n/locales/zh.ts`

## Implementation Notes

- Added `/admin/openai-quota-summary` as an admin route with title and description locale keys.
- Added an admin sidebar item near account management, reusing the existing chart icon.
- Built a quiet operational page with a compact projection toolbar, generated/projection timestamps, grouped tables, and recovery cells.
- Implemented projection modes for current, hours later, and days later. Current sends `{}`; future modes send `projection_at` as an ISO timestamp.
- Used the provided `formatPercent` and `formatDateTime` helper behavior.

## Self-Review

- Confirmed the test exercises real rendered data and the projection request payload.
- Confirmed required `data-test` hooks exist.
- Confirmed route, navigation, and requested locale keys are present in both English and Chinese.
- Confirmed the page avoids hero/marketing layout and keeps controls/table content compact and scan-friendly.
- Residual risk: route permission granularity follows the task's requested write scope and route snippet; no change was made to the shared admin permission path map.

## Review Fix Report

Status: Complete.

Changes:

- Added OpenAI group and account-type filters to `OpenAIQuotaSummaryView.vue`; refresh now sends `group` and `type` alongside any `projection_at`.
- Loaded OpenAI groups through the existing admin groups API pattern.
- Mapped `/admin/openai-quota-summary` to the `accounts` admin resource in route meta and `adminResourceForPath`, which also covers sidebar permission filtering.
- Moved the page's visible labels and table headings into `admin.openAIQuotaSummary` locale keys for English and Chinese.

TDD / verification evidence:

- RED: `corepack pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
  - Result: failed because `groupsAPI.getAll` was not called and filter controls did not exist.
- RED: `corepack pnpm test:run src/utils/__tests__/adminPermissions.spec.ts`
  - Result: failed because `/admin/openai-quota-summary` mapped to `null`.
- GREEN: `corepack pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts src/utils/__tests__/adminPermissions.spec.ts`
  - Result: 2 test files passed, 5 tests passed.
- `corepack pnpm typecheck`
  - Result: passed.
- `corepack pnpm exec eslint src/views/admin/OpenAIQuotaSummaryView.vue src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts src/router/index.ts src/utils/adminPermissions.ts src/utils/__tests__/adminPermissions.spec.ts src/i18n/locales/en.ts src/i18n/locales/zh.ts`
  - Result: passed.

## Re-Review Fix Report

Status: Complete.

Changes:

- Localized ungrouped section headings instead of rendering the raw API `group_name`.
- Switched the group filter to `groupsAPI.getAllIncludingInactive()` and merged in non-ungrouped summary groups missing from the groups API response while preserving `group=<id>` and `group=ungrouped` query values.
- Rendered OpenAI quota row account types through existing `admin.accounts.*` account type labels.
- Initialized the summary area in a loading state so the empty state is not visible before the first summary request settles.

TDD / verification evidence:

- RED: `corepack pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
  - Result: failed as expected. Failures covered raw `oauth` display, missing `getAllIncludingInactive()` call, raw ungrouped heading, missing summary-derived group option, and initial empty-state flash.
- GREEN: `corepack pnpm test:run src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
  - Result: 1 test file passed, 6 tests passed.
- `corepack pnpm typecheck`
  - Result: passed.
- `corepack pnpm exec eslint src/views/admin/OpenAIQuotaSummaryView.vue src/views/admin/__tests__/OpenAIQuotaSummaryView.spec.ts`
  - Result: passed.
