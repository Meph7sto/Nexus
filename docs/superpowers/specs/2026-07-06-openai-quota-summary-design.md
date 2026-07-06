# OpenAI Quota Summary Design

## Purpose

Add an admin-only OpenAI quota summary page that aggregates all account groups and shows theoretical current or future Codex quota headroom by account type.

The page is for capacity planning: an administrator should quickly see which OpenAI partitions have enough usable 5-hour and weekly quota, how many accounts are excluded because they are unhealthy, and when the earliest quota recovery will happen.

## Scope

In scope:

- A new admin page for OpenAI/GPT/Codex account quota summary.
- A backend summary API that aggregates OpenAI accounts by group and account type.
- Current and future theoretical quota projection.
- Counts for excluded `error` and `inactive` accounts.
- Counts for active accounts without usage snapshots, treated as theoretical full quota.
- Earliest recovery information for 5-hour and weekly windows.

Out of scope:

- Mutating account quota, clearing rate limits, or calling upstream OpenAI quota reset APIs.
- Replacing the account management list.
- Historical usage charts.
- Live upstream probing for accounts without snapshots.

## Data Sources

The feature uses existing account data:

- `accounts.platform == "openai"`
- `accounts.type` for account type grouping.
- account group bindings for partition grouping.
- `accounts.status` for inclusion or exclusion.
- `accounts.extra.codex_5h_used_percent`
- `accounts.extra.codex_5h_reset_at`
- `accounts.extra.codex_7d_used_percent`
- `accounts.extra.codex_7d_reset_at`
- optional legacy fields may remain ignored unless the backend already normalizes them into the canonical fields above.

Accounts that belong to multiple groups are counted in each group. Accounts without a group are counted in an "Ungrouped" bucket.

## Inclusion Rules

Only OpenAI accounts are considered.

For averages:

- `active` accounts are included.
- `error` accounts are excluded and counted.
- `inactive` accounts are excluded and counted.
- Other non-active statuses are excluded and counted as "other excluded" if they exist.

Snapshot handling:

- Active accounts without a Codex usage snapshot are included as theoretical full quota.
- Missing 5-hour snapshot means 5-hour remaining is `100`.
- Missing weekly snapshot means weekly remaining is `100`.
- The response exposes missing snapshot counts so the UI makes the assumption visible.

## Projection Rules

The query accepts a projection timestamp, defaulting to server `now`.

For each active account and each window:

- If there is no snapshot, remaining quota is `100`.
- If there is a snapshot and no reset time, remaining quota is `100 - used_percent`, clamped to `[0, 100]`.
- If there is a snapshot and the projection timestamp is before the reset time, remaining quota is `100 - used_percent`, clamped to `[0, 100]`.
- If there is a snapshot and the projection timestamp is at or after the reset time, remaining quota is `100`.

The displayed average is the arithmetic mean of projected remaining percentages over included active accounts in the bucket.

## Earliest Recovery

For each group/type bucket, calculate earliest recovery separately for:

- 5-hour window.
- Weekly window.

Candidate accounts:

- Active accounts only.
- Must have a valid reset time for that window.
- The reset time must be after the projection timestamp.
- The window should currently have less than `100` remaining quota.

The earliest recovery record includes:

- Account ID.
- Account name.
- Account type.
- Reset time.
- Remaining quota before recovery.
- Remaining quota after recovery, always `100` for that window.

If no candidate exists, the field is `null`.

## Backend API

Add a read-only admin endpoint:

`GET /api/v1/admin/openai/quota-summary`

Query parameters:

- `projection_at`: optional RFC3339 timestamp. Defaults to server `now`.
- `group`: optional group ID, or `ungrouped`, for narrowing the response.
- `type`: optional account type filter.

Response shape:

```json
{
  "projection_at": "2026-07-06T15:00:00Z",
  "generated_at": "2026-07-06T14:00:00Z",
  "groups": [
    {
      "group_id": 12,
      "group_name": "OpenAI East",
      "rows": [
        {
          "account_type": "oauth",
          "included_count": 10,
          "error_count": 1,
          "inactive_count": 2,
          "other_excluded_count": 0,
          "missing_5h_snapshot_count": 3,
          "missing_7d_snapshot_count": 3,
          "avg_5h_remaining_percent": 90,
          "avg_7d_remaining_percent": 84.5,
          "earliest_5h_recovery": {
            "account_id": 42,
            "account_name": "openai-01",
            "reset_at": "2026-07-06T16:30:00Z",
            "remaining_before_percent": 0,
            "remaining_after_percent": 100
          },
          "earliest_7d_recovery": null
        }
      ]
    }
  ]
}
```

The endpoint uses the existing admin account permission resource because it only reads account capacity data.

## Frontend UI

Add a new admin route:

`/admin/openai-quota-summary`

Add a sidebar item near account management:

- Chinese: `OpenAI 额度汇总`
- English: `OpenAI Quota Summary`

The page layout:

- Header with title and refresh button.
- Projection controls:
  - segmented options: current, hours later, days later.
  - numeric input for hours or days, editable by the admin.
  - display the exact projected timestamp.
- Optional filters for group and account type.
- Summary table grouped by account group.

Each table row shows:

- Account type.
- Included active accounts.
- Error accounts.
- Inactive accounts.
- Missing 5h snapshot count.
- Missing weekly snapshot count.
- Average 5h remaining.
- Average weekly remaining.
- Earliest 5h recovery.
- Earliest weekly recovery.

Empty states:

- No OpenAI accounts.
- No rows after filters.
- Group exists but has no included active accounts.

Loading and error states follow existing admin page conventions.

## Components

Backend:

- Add quota summary domain/service helpers near account service code.
- Add handler method under admin account or OpenAI admin handler.
- Add route under admin OpenAI/account routes.
- Add focused unit tests for the projection math and aggregation.

Frontend:

- Add API client types and function in `frontend/src/api/admin/accounts.ts` or a small OpenAI quota API module.
- Add `OpenAIQuotaSummaryView.vue`.
- Add route and nav labels.
- Add focused tests for projection controls and rendering aggregation rows.

## Testing

Backend tests:

- Active accounts are included in averages.
- `error` and `inactive` accounts are excluded but counted.
- Missing snapshots count as 100 remaining.
- Future projection resets accounts to 100 when `projection_at >= reset_at`.
- Earliest recovery ignores excluded accounts and missing reset times.
- Multi-group accounts appear in each group.
- Ungrouped accounts appear in the ungrouped bucket.

Frontend tests:

- Default page requests current projection.
- Hours and days controls send the expected `projection_at`.
- Rows display included, error, inactive, missing snapshot counts, averages, and earliest recovery.
- Empty and loading states render correctly.

## Open Questions Resolved

- Missing snapshots are treated as theoretical full quota, not unknown.
- `error` accounts are excluded from averages and counted.
- `inactive` accounts are excluded from averages and counted.
- Account groups are the "partitions" used for aggregation.
