# Admin Usage Interaction Details Design

## Purpose

Add an admin-only interaction detail view to `admin/usage` so administrators can inspect the concrete content behind a usage row: the original input or prompt, the model output, request parameters, routing and billing context, and optional raw request/response JSON.

The feature is intended for operational audit and support. It must be explicit, controlled, and off by default because successful request content can contain sensitive user data.

## Scope

In scope:

- A new persistent store for successful request interaction details.
- A default-off setting that administrators must enable before new interactions are recorded.
- Full input/prompt and full output preservation without summarization or truncation.
- Optional raw request/response JSON storage and viewing.
- Credential redaction for secrets such as authorization headers, cookies, API keys, and upstream tokens.
- A usage row detail action and modal in `admin/usage`.
- Retention configuration with default `7` days and arbitrary integer-day values.
- Automatic cleanup of stored interaction details according to retention.

Out of scope:

- Backfilling historical interaction details for rows created before the feature is enabled.
- Reconstructing missing details from token counts or summaries.
- Showing raw JSON by default in the usage list or first modal view.
- Storing unredacted credentials.
- Changing billing, token counting, or existing usage aggregation semantics.

## Data Model

Create a new `usage_interactions` table instead of adding large JSON/text fields to `usage_logs`.

Core fields:

- `id`
- `usage_log_id`, unique, referencing `usage_logs.id`
- `request_id`
- `user_id`
- `api_key_id`
- `account_id`
- `group_id`, nullable
- `created_at`

Interaction content fields:

- `request_content`: structured JSON containing the full user-visible input or prompt content.
- `response_content`: structured JSON containing the full model output content.
- `request_parameters`: structured JSON for model parameters such as model, endpoint, stream flag, temperature, top_p, max_tokens, reasoning effort, tools, modalities, and related metadata when present.
- `routing_context`: structured JSON for non-secret route context such as inbound endpoint, upstream endpoint, requested model, upstream model, model mapping chain, channel ID, billing tier, billing mode, service tier, request type, and duration fields.

Raw fields:

- `raw_request_json`, nullable.
- `raw_response_json`, nullable.
- `raw_available`, boolean or derived from non-null raw fields.

Safety fields:

- `redaction_applied`, boolean.
- `redaction_keys`, JSON array or text array of redacted field names.

No content-size truncation is applied. The stored prompt, input, output, and raw JSON should preserve the original text after credential redaction.

## Settings

Add settings under the existing admin settings infrastructure:

- `usage_interaction_recording_enabled`: boolean, default `false`.
- `usage_interaction_store_raw_enabled`: boolean, default `false`.
- `usage_interaction_retention_days`: integer, default `7`.

Retention behavior:

- Any positive value means delete `usage_interactions` older than that many days.
- `0` means keep indefinitely.
- Negative values are invalid.

Raw behavior:

- When recording is enabled and raw storage is disabled, store structured input/output/parameters only.
- When both recording and raw storage are enabled, also store redacted raw request/response JSON.
- The UI still hides raw JSON behind an explicit tab or action.

## Capture Flow

Capture should happen only after a successful request produces a usage log. The interaction row must be linked to the persisted `usage_logs.id`.

For synchronous responses:

- Capture the parsed inbound request content and parameters before forwarding.
- Capture the response body after the upstream response is available.
- Record the interaction after the usage log is created.

For streaming responses:

- Capture the parsed inbound request content and parameters before forwarding.
- Aggregate the emitted output into a full response content representation while streaming.
- Record after the stream finishes and the usage log is created.
- If the stream fails before a successful usage log, do not create a `usage_interactions` row for the success path; existing ops error logging remains responsible for error details.

For WebSocket or realtime style requests:

- Capture the initial session/request payload and aggregate assistant output events when a usage log is generated.
- Store event-derived input/output in a structured form that the UI can render as turns or sections.

The write path should be best effort. Failure to store interaction details must not fail the user request, double-charge, or block usage log creation. It should log an operational warning with request ID and usage log ID.

## Redaction

Redaction applies before persistence.

Always redact:

- `Authorization`
- `Proxy-Authorization`
- `Cookie`
- `Set-Cookie`
- API key fields such as `api_key`, `apiKey`, `key`, `token`, `access_token`, `refresh_token`, `id_token`, `session_token`, `secret`, `client_secret`, and provider credential fields.
- Upstream account tokens or OAuth credentials.

Do not redact normal user prompt text or model output merely because it may be sensitive. The feature is explicitly for full interaction audit after the administrator enables it.

Redacted values should be replaced with a stable marker such as `[REDACTED]`, and `redaction_applied` should be true when any field is changed.

## API Design

Add an admin-only endpoint:

`GET /api/v1/admin/usage/:id/interaction`

Response shape without raw:

```json
{
  "exists": true,
  "usage_log_id": 123,
  "request_id": "req_abc",
  "created_at": "2026-07-07T12:00:00Z",
  "request_content": {},
  "response_content": {},
  "request_parameters": {},
  "routing_context": {},
  "raw_available": true,
  "redaction_applied": true,
  "redaction_keys": ["Authorization", "api_key"]
}
```

When `include_raw=true`, the same response also includes `raw_request_json` and `raw_response_json` when those fields were stored.

For missing details:

```json
{
  "exists": false,
  "reason": "not_recorded"
}
```

Possible missing reasons:

- `not_recorded`: feature was disabled, row is historical, or capture was unavailable.
- `cleaned_up`: detail existed but retention cleanup removed it.

Raw fields may be omitted unless the caller asks for raw explicitly with `include_raw=true`. The UI should first load the readable details, then request raw only when the administrator opens the raw tab. This keeps accidental exposure and payload size lower.

## Permissions

The endpoint requires admin authentication and usage-view permission equivalent to the current `admin/usage` page.

If the existing permission model can support a finer permission without broad churn, add a raw-specific permission such as `usage_interaction:view_raw`. If that would delay the feature, ship MVP with the same usage admin permission and keep raw hidden behind explicit UI action.

Regular user usage APIs must never include interaction details or raw JSON.

## Frontend Design

Update `admin/usage` with a compact details action per row.

The interaction modal uses tabs:

- Input
- Output
- Parameters
- Routing and Billing
- Raw

Default tab is Input. Raw is not loaded or displayed until selected. The raw tab should clearly indicate that it contains redacted raw request/response JSON. It should provide separate request and response panels with copy buttons.

Display rules:

- If no detail exists, show a concise empty state explaining that details were not recorded or have been cleaned up.
- Preserve whitespace for text content.
- Render message arrays as readable role/content blocks.
- Render structured tool calls, image metadata, or multimodal parts as expandable JSON sections.
- Keep the existing usage table dense; do not add large content columns.

## Cleanup

Extend the existing cleanup/timing infrastructure with a `usage_interactions` cleanup task.

Rules:

- If recording is disabled, cleanup still runs according to retention so old data can age out.
- If retention days is `0`, skip deletion.
- Deleting usage logs through existing usage cleanup should also delete related interactions, preferably via foreign key cascade or explicit repository cleanup.

## Testing

Backend tests:

- Default settings do not create interaction rows.
- Enabled recording creates a row linked to the correct usage log.
- Full input and output are preserved without summarization or truncation.
- Raw fields are stored only when raw storage is enabled.
- Credential fields are redacted before persistence.
- Missing interaction endpoint returns `exists: false`.
- `include_raw=false` or absent does not return raw fields.
- `include_raw=true` returns raw fields for authorized admin callers.
- Retention cleanup deletes rows older than the configured arbitrary day count and skips when set to `0`.

Frontend tests:

- Usage table exposes a details action.
- Detail modal loads and renders input, output, parameters, and routing/billing tabs.
- Raw tab loads raw JSON only after being selected.
- Missing-detail state is shown for `exists: false`.
- Raw copy actions use the existing clipboard/toast pattern.

## Risks And Mitigations

Database growth:

- Mitigated by default-off recording and default `7` day retention.
- Operators can set retention to any non-negative integer based on their storage policy.

Sensitive data exposure:

- Mitigated by default-off recording, admin-only access, raw hidden by default, and credential redaction.
- Full user content remains visible by design after the operator enables the feature.

Hot-path performance:

- Mitigated by best-effort asynchronous or post-response persistence where existing gateway flow allows it.
- Usage list queries remain lightweight because large content lives in a separate table and is fetched on demand.

Streaming completeness:

- Mitigated by aggregating output events in the same flow that computes usage. If a stream fails and no successful usage log exists, ops error logging remains the source of failure details.
