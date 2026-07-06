-- Add limited administrator permissions and preserve existing admin access.

CREATE TABLE IF NOT EXISTS admin_permissions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource VARCHAR(64) NOT NULL,
    actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_permissions_user_resource_unique UNIQUE (user_id, resource)
);

CREATE INDEX IF NOT EXISTS idx_admin_permissions_user_id ON admin_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_admin_permissions_resource ON admin_permissions(resource);

UPDATE users SET role = 'super_admin' WHERE role = 'admin';
