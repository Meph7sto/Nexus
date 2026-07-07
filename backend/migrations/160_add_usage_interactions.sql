CREATE TABLE IF NOT EXISTS usage_interactions (
    id BIGSERIAL PRIMARY KEY,
    usage_log_id BIGINT NOT NULL UNIQUE REFERENCES usage_logs(id) ON DELETE CASCADE,
    request_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    group_id BIGINT,
    capture_status VARCHAR(20) NOT NULL DEFAULT 'complete',
    capture_error TEXT,
    request_content JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_content JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    routing_context JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_request_json JSONB,
    raw_response_json JSONB,
    redaction_applied BOOLEAN NOT NULL DEFAULT FALSE,
    redaction_keys JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_interactions_request_id ON usage_interactions(request_id);
CREATE INDEX IF NOT EXISTS idx_usage_interactions_created_at ON usage_interactions(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_interactions_user_created_at ON usage_interactions(user_id, created_at DESC);

INSERT INTO settings (key, value, created_at, updated_at)
VALUES
    ('usage_interaction_recording_enabled', 'false', NOW(), NOW()),
    ('usage_interaction_store_raw_enabled', 'false', NOW(), NOW()),
    ('usage_interaction_retention_days', '7', NOW(), NOW())
ON CONFLICT (key) DO NOTHING;
