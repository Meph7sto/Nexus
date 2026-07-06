## Final Review Fix - Non-Finite OpenAI Quota Snapshots

- Fixed `openAIQuotaSummaryRemaining` to parse used-percent snapshots through a single `(float64, bool)` helper and reject `NaN`/`Inf` from floats, strings, and `json.Number`.
- Added regression coverage for `"NaN"`, `"Infinity"`, `math.NaN()`, `math.Inf(1)`, and matching `json.Number` values. Rejected snapshots count as missing and contribute full quota for the affected windows.
- RED: `go test ./internal/service -run OpenAIQuotaSummary -count=1` failed at `TestBuildOpenAIQuotaSummary_NonFiniteSnapshotsCountAsMissingAndFullQuota`, expected missing count `4`, actual `0`.
- GREEN: `go test ./internal/service -run OpenAIQuotaSummary -count=1` passed: `ok github.com/Wei-Shaw/nexus/internal/service 0.333s`.
