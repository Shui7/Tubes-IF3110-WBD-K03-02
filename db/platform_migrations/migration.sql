CREATE TABLE IF NOT EXISTS db_platform_migrations (
    version BIGINT PRIMARY KEY,
    sql_text TEXT NOT NULL,
    executed_by TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('success', 'failed')),
    error_message TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS db_platform_migration_drafts (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    sql_text TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
